package anthropic_provider

import (
	"bytes"
	"encoding/json"
	"errors"
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

type PrintWriter struct {
}

func (p2 *PrintWriter) Write(p []byte) (n int, err error) {
	fmt.Printf("%s", p)
	return len(p), nil
}

var _ io.Writer = &PrintWriter{}

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

	/**
	  curl https://api.anthropic.com/v1/messages \
	       --header "anthropic-version: 2023-06-01" \
	       --header "content-type: application/json" \
	       --header "x-api-key: $ANTHROPIC_API_KEY" \
	       --data \
	  '{
	    "model": "claude-3-5-sonnet-20240620",
	    "messages": [{"role": "user", "content": "Hello"}],
	    "max_tokens": 256,
	    "stream": true
	  }'
	*/

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
			//"Authorization": []string{fmt.Sprintf("Bearer %s", o.cfg.APIKey)},
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
		panic(fmt.Sprintf("Failed to do request: %v: %s", err, string(respBody)))
	}
	defer func() {
		err := res.Body.Close()
		if err != nil {
			slog.Error(fmt.Sprintf("Failed to close body: %v", err))
		}
	}()

	if res.StatusCode != 200 {
		respBody, _ := io.ReadAll(res.Body)
		panic(fmt.Sprintf("Failed to do request, unexpected status code: %v: %s", res.StatusCode, string(respBody)))
	}

	printWriter := PrintWriter{}
	_, err = io.Copy(&printWriter, res.Body)
	if err != nil {
		panic(fmt.Sprintf("Failed to copy response body: %v", err))
	}
	//
	//req := openai.ChatCompletionRequest{
	//	Model: o.cfg.Model,
	//	Messages: lo.Map(question.Messages, func(message domain.Message, index int) openai.ChatCompletionMessage {
	//		return openai.ChatCompletionMessage{
	//			Role:    string(message.SourceType),
	//			Content: message.Content,
	//		}
	//	}),
	//	Temperature: float32(o.cfg.Temperature),
	//	Stream:      true,
	//	StreamOptions: &openai.StreamOptions{
	//		IncludeUsage: true,
	//	},
	//}
	//remoteStream, err := o.client.CreateChatCompletionStream(
	//	context.Background(),
	//	req,
	//)
	//if err != nil {
	//	resChan <- domain.RespChunk{Err: fmt.Errorf("failed to create chat completion stream: %w", err)}
	//	close(resChan)
	//	return resChan
	//}
	//go func() {
	//
	//	defer func() {
	//		err := remoteStream.Close()
	//		if err != nil {
	//			slog.Error(fmt.Sprintf("failed to close chat completion stream: %v", err))
	//		}
	//	}()
	//	defer close(resChan)
	//
	//	for {
	//		response, err := remoteStream.Recv()
	//		if errors.Is(err, io.EOF) {
	//			return //we're done
	//		}
	//
	//		if err != nil {
	//			resChan <- domain.RespChunk{Err: fmt.Errorf("failed to receive stream response: %w", err)}
	//			return
	//		}
	//
	//		resChan <- domain.RespChunk{Resp: openAiStrResp2Resp(response)}
	//	}
	//}()

	resChan <- domain.RespChunk{Err: errors.New("not implemented")}

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
