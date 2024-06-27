package anthropic_provider

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/GiGurra/sse-parser"
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

	promptTokens := 0
	completionTokens := 0
	totalTokens := 0

	for chunk := range stream {
		if chunk.Err != nil {
			return nil, chunk.Err
		}

		promptTokens += chunk.Resp.GetUsage().PromptTokens
		completionTokens += chunk.Resp.GetUsage().CompletionTokens
		totalTokens += chunk.Resp.GetUsage().TotalTokens

		if len(chunk.Resp.GetChoices()) == 0 {
			continue // this is the final chunk
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
			PromptTokens:     promptTokens,
			CompletionTokens: completionTokens,
			TotalTokens:      totalTokens,
		},
	}, nil
}

type ResponseStreamHandler struct {
	resChan                  chan domain.RespChunk
	parser                   sse_parser.Parser
	isInsideTextContentBlock bool
	accumInputTokens         int
	accumOutputTokens        int
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

type Usage struct {
	InputTokens  int `json:"input_tokens"`
	OutputTokens int `json:"output_tokens"`
}

type MessageStartMessage struct {
	ID           string `json:"id"`
	Type         string `json:"type"`
	Role         string `json:"role"`
	Model        string `json:"model"`
	Content      any    `json:"content"`
	StopReason   any    `json:"stop_reason"`
	StopSequence any    `json:"stop_sequence"`
	Usage        Usage  `json:"usage"`
}

type MessageStart struct {
	Type    string              `json:"type"`
	Message MessageStartMessage `json:"message"`
}

type MessageDeltaDelta struct {
	StopReason   any `json:"stop_reason"`
	StopSequence any `json:"stop_sequence"`
}

type MessageDelta struct {
	Type  string            `json:"type"`
	Delta MessageDeltaDelta `json:"delta"`
	Usage Usage             `json:"usage"`
}

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

		defer close(resChan)
		defer closeBody()

		accumInputTokens := 0
		accumOutputTokens := 0
		isInsideTextContentBlock := false

		parser := sse_parser.NewParser(isValidJsonObject)
		for msg := range parser.Stream(res.Body, 100) {

			eventType := strings.TrimSpace(msg.Event)
			dataStr := msg.Data

			switch eventType {
			case "message_start":
				var messageStart MessageStart
				err := json.Unmarshal([]byte(dataStr), &messageStart)
				if err != nil {
					resChan <- domain.RespChunk{
						Err: fmt.Errorf("failed to unmarshal message start: %v", err),
					}
					return
				}
				accumInputTokens += messageStart.Message.Usage.InputTokens
				// apparently we shouldn't count these output tokens, to stay consistent with
				// anthropic's own token counting (see https://console.anthropic.com/settings/logs)
				//accumOutputTokens += messageStart.Message.Usage.OutputTokens
			case "content_block_start":
				var contentBlockStart ContentBlockStart
				err := json.Unmarshal([]byte(dataStr), &contentBlockStart)
				if err != nil {
					resChan <- domain.RespChunk{
						Err: fmt.Errorf("failed to unmarshal content block start: %v", err),
					}
					return
				}
				if contentBlockStart.ContentBlock.Type == "text" {
					isInsideTextContentBlock = true
				}
			case "content_block_delta":
				if isInsideTextContentBlock {
					var contentBlockDelta ContentBlockDelta
					err := json.Unmarshal([]byte(dataStr), &contentBlockDelta)
					if err != nil {
						resChan <- domain.RespChunk{
							Err: fmt.Errorf("failed to unmarshal content block delta: %v", err),
						}
						return
					}
					resChan <- domain.RespChunk{
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
				isInsideTextContentBlock = false
			case "message_stop":
				// we're done!
				resChan <- domain.RespChunk{
					Resp: &BasicAskResponse{
						Choices: []domain.Choice{},
						Usage: domain.Usage{
							PromptTokens:     accumInputTokens,
							CompletionTokens: accumOutputTokens,
							TotalTokens:      accumInputTokens + accumOutputTokens,
						},
					},
				}
			case "message_delta":
				var messageDelta MessageDelta
				err := json.Unmarshal([]byte(dataStr), &messageDelta)
				if err != nil {
					resChan <- domain.RespChunk{
						Err: fmt.Errorf("failed to unmarshal message delta: %v", err),
					}
					return
				}
				accumInputTokens += messageDelta.Usage.InputTokens
				accumOutputTokens += messageDelta.Usage.OutputTokens
			default:
				// do nothing, unsupported (by our ai) events
			}
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

func isValidJsonObject(str string) bool {
	var jsObj map[string]any
	err := json.Unmarshal([]byte(str), &jsObj)
	return err == nil
}
