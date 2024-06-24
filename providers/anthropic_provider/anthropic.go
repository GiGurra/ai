package anthropic_provider

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/gigurra/ai/common"
	"github.com/gigurra/ai/domain"
	"github.com/samber/lo"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
)

type Config struct {
	APIKey          string `yaml:"api_key"`
	Model           string `yaml:"model_id"`
	Version         string `yaml:"version"`
	MaxOutputTokens int    `yaml:"max_output_tokens"`
}

type Provider struct {
	cfg Config
}

type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type RequestBody struct {
	Model     string    `json:"model"`
	Messages  []Message `json:"messages"`
	MaxTokens *int      `json:"max_tokens,omitempty"`
	Stream    bool      `json:"stream"`
}

// see https://docs.anthropic.com/en/api/messages-streaming#basic-streaming-request

type BasicAskResponse struct {
	Choices []domain.Choice
	Usage   domain.Usage
}

var _ domain.Response = &BasicAskResponse{}

func (r *BasicAskResponse) GetChoices() []domain.Choice {
	return r.Choices
}

func (r *BasicAskResponse) GetUsage() domain.Usage {
	return r.Usage
}

func (o Provider) BasicAsk(question domain.Question) (domain.Response, error) {

	stream := o.BasicAskStream(question)

	accum := strings.Builder{}

	for chunk := range stream {
		if chunk.Err != nil {
			return nil, chunk.Err
		}
		if len(chunk.Resp.GetChoices()) == 0 {
			return nil, fmt.Errorf("expected at least one choice")
		}
		accum.WriteString(chunk.Resp.GetChoices()[0].Message.Content)
	}

	return &BasicAskResponse{
		Choices: []domain.Choice{
			{
				Index: 0,
				Message: domain.Message{
					SourceType: domain.Assistant,
					Content:    accum.String(),
				},
			},
		},
		Usage: domain.Usage{
			PromptTokens:     0,
			CompletionTokens: 0,
			TotalTokens:      0,
		},
	}, nil
}

type ResponseStreamHandler struct {
	resChan                  chan domain.RespChunk
	buffer                   strings.Builder
	isInsideTextContentBlock bool
}

type ContentBlock struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

type ContentBlockStart struct {
	Type         string       `json:"type"`
	Index        int          `json:"index"`
	ContentBlock ContentBlock `json:"content_block"`
}

type ContentBlockDelta struct {
	Type  string       `json:"type"`
	Index int          `json:"index"`
	Delta ContentBlock `json:"delta"`
}

func (p2 *ResponseStreamHandler) ProcessBuffer(forceFlush bool) error {
retry:

	//slog.Info(fmt.Sprintf("Processing buffer: %s", strings.TrimSpace(p2.buffer.String())))

	if p2.buffer.Len() == 0 {
		return nil
	}

	// sanitize/remove empty lines
	origLinesInBuffer := strings.Split(strings.TrimSpace(p2.buffer.String()), "\n")
	linesInBuffer := lo.Map(origLinesInBuffer, func(item string, index int) string {
		return strings.TrimSpace(item)
	})
	linesInBuffer = lo.Filter(linesInBuffer, func(item string, index int) bool {
		return item != ""
	})

	if len(linesInBuffer) != len(origLinesInBuffer) {
		p2.buffer.Reset()
		for i := 0; i < len(linesInBuffer); i++ {
			p2.buffer.WriteString(linesInBuffer[i])
			p2.buffer.WriteString("\n")
		}
		goto retry
	}

	if len(linesInBuffer) >= 2 {
		eventLine := linesInBuffer[0]
		dataLine := linesInBuffer[1]

		//slog.Info(fmt.Sprintf("Event line: %s", eventLine))
		if !strings.HasPrefix(eventLine, "event: ") || !strings.HasPrefix(dataLine, "data: ") {
			return fmt.Errorf(fmt.Sprintf("Invalid SSE event contents: %s", p2.buffer.String()))
		}

		eventType := strings.TrimSpace(strings.TrimPrefix(eventLine, "event: "))
		dataStr := strings.TrimSpace(strings.TrimPrefix(dataLine, "data: "))

		// check if dataStr is a valid json object, which indicates we have received all the data
		var jsObj map[string]any
		err := json.Unmarshal([]byte(dataStr), &jsObj)
		if err != nil {
			// not enough data received yet, just return and wait for next chunk
			if !forceFlush {
				//slog.Info(fmt.Sprintf("Incomplete data in buffer, waiting for next chunk: %s", p2.buffer.String()))
				return nil
			} else {
				return fmt.Errorf("incomplete data in buffer when final flush/Checkbuffer was called: %s", p2.buffer.String())
			}
		}

		//slog.Info(fmt.Sprintf("Received event: %s, data: %s", eventType, dataStr))

		switch eventType {
		case "message_start":
			// ignore, TODO: Read input token count here
		case "content_block_start":
			var contentBlockStart ContentBlockStart
			err := json.Unmarshal([]byte(dataStr), &contentBlockStart)
			if err != nil {
				return fmt.Errorf("failed to unmarshal content block start: %v", err)
			}
			if contentBlockStart.ContentBlock.Type == "text" {
				p2.isInsideTextContentBlock = true
			}
		case "content_block_delta":
			if p2.isInsideTextContentBlock {
				var contentBlockDelta ContentBlockDelta
				err := json.Unmarshal([]byte(dataStr), &contentBlockDelta)
				if err != nil {
					return fmt.Errorf("failed to unmarshal content block delta: %v", err)
				}
				p2.resChan <- domain.RespChunk{
					Resp: &BasicAskResponse{
						Choices: []domain.Choice{
							{
								Index: 0,
								Message: domain.Message{
									SourceType: domain.Assistant,
									Content:    contentBlockDelta.Delta.Text,
								},
							},
						},
					},
				}
			} else {
				//slog.Info(fmt.Sprintf("Ignoring content block delta: %s", dataStr))
				// ignore, we're not inside a content block
			}
		case "content_block_stop":
			p2.isInsideTextContentBlock = false
		case "message_stop":
			// we're done!
		default:
			// do nothing, unsupported (by our ai) events
		}

		// now we know the entire event has been received
		p2.buffer.Reset()
		for i := 2; i < len(linesInBuffer); i++ {
			p2.buffer.WriteString(linesInBuffer[i])
			p2.buffer.WriteString("\n")
		}
		//time.Sleep(1 * time.Second)
		goto retry // we might have already received another event
	}

	// not enough data yet to process
	return nil
}

func (p2 *ResponseStreamHandler) Write(p []byte) (int, error) {
	p2.buffer.WriteString(string(p))
	err := p2.ProcessBuffer(false)
	if err != nil {
		return 0, fmt.Errorf("failed to check/process receive buffer: %v", err)
	}
	return len(p), nil
}

func (p2 *ResponseStreamHandler) finish() error {
	err := p2.ProcessBuffer(true)
	if err != nil {
		return fmt.Errorf("failed to check/process receive buffer: %v", err)
	}
	return nil
}

var _ io.Writer = &ResponseStreamHandler{}

func (o Provider) BasicAskStream(question domain.Question) <-chan domain.RespChunk {
	resChan := make(chan domain.RespChunk, 1024)

	host := "api.anthropic.com"
	u, err := url.Parse("https://" + host + "/v1/messages")
	if err != nil {
		common.FailAndExit(1, fmt.Sprintf("failed to parse url: %v", err))
	}

	if o.cfg.APIKey == "" {
		common.FailAndExit(1, "Anthropic API key is required")
	}

	if o.cfg.Model == "" {
		common.FailAndExit(1, "Anthropic model is required")
	}

	if o.cfg.Version == "" {
		common.FailAndExit(1, "Anthropic api version configuration parameter is required")
	}

	if o.cfg.MaxOutputTokens == 0 {
		common.FailAndExit(1, "Anthropic max_output_tokens configuration parameter is required")
	}

	body := RequestBody{
		Model: o.cfg.Model,
		Messages: lo.Map(question.Messages, func(message domain.Message, index int) Message {
			return Message{
				Role:    string(message.SourceType),
				Content: message.Content,
			}
		}),
		MaxTokens: &o.cfg.MaxOutputTokens,
		Stream:    true,
	}

	bodyBytes, err := json.Marshal(body)
	if err != nil {
		common.FailAndExit(1, fmt.Sprintf("failed to marshal request body: %v", err))
	}

	bodyReadCloser := io.NopCloser(bytes.NewReader(bodyBytes))

	request := http.Request{
		Method: "POST",
		URL:    u,
		Header: http.Header{
			"anthropic-version": []string{o.cfg.Version},
			"Content-Type":      []string{"application/json"},
			"x-api-key":         []string{o.cfg.APIKey},
		},
		Body:          bodyReadCloser,
		ContentLength: int64(len(bodyBytes)),
		Host:          host,
	}

	res, err := http.DefaultClient.Do(&request)
	if err != nil {
		respBody, _ := io.ReadAll(res.Body)
		common.FailAndExit(1, fmt.Sprintf("Failed to do request: %v: %s", err, string(respBody)))
	}

	closeBody := func() {
		err := res.Body.Close()
		if err != nil {
			slog.Error(fmt.Sprintf("Failed to close body: %v", err))
		}
	}

	if res.StatusCode != 200 {
		defer closeBody()
		respBody, _ := io.ReadAll(res.Body)
		common.FailAndExit(1, fmt.Sprintf("Failed to do request, unexpected status code: %v: %s", res.StatusCode, string(respBody)))
	}

	go func() {
		defer closeBody()
		defer close(resChan)
		forwarder := ResponseStreamHandler{resChan: resChan}
		_, err = io.Copy(&forwarder, res.Body)
		if err != nil {
			common.FailAndExit(1, fmt.Sprintf("failed to receive response body: %v", err))
		}
		err = forwarder.finish()
		if err != nil {
			common.FailAndExit(1, fmt.Sprintf("failed to finish response body: %v", err))
		}
	}()

	return resChan
}

// prove that OpenAIProvider implements the Provider interface
var _ domain.Provider = &Provider{}

func NewAnthropicProvider(cfg Config, verbose bool) *Provider {

	provider := &Provider{
		cfg: cfg,
	}

	return provider
}

func (o Provider) ListModels() ([]string, error) {
	slog.Warn("ListModels not implemented. Returning hardcoded model list with only claude-3-5-sonnet-20240620")
	return []string{"claude-3-5-sonnet-20240620"}, nil
}
