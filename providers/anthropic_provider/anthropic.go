package anthropic_provider

import (
	"errors"
	"fmt"
	"github.com/gigurra/ai/domain"
	"log/slog"
	"strings"
)

type Config struct {
	APIKey  string `yaml:"api_key"`
	Model   string `yaml:"model"`
	Version string `yaml:"version"`
}

type Provider struct {
	cfg Config
}

type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
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

func (o Provider) BasicAskStream(question domain.Question) <-chan domain.RespChunk {
	resChan := make(chan domain.RespChunk, 1024)
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
