package anthropic_provider

import (
	"errors"
	"fmt"
	"github.com/gigurra/ai/domain"
	"log/slog"
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

func (o Provider) BasicAsk(question domain.Question) (domain.Response, error) {

	return nil, fmt.Errorf("not implemented")
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
