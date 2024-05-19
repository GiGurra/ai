package openai_provider

import (
	"context"
	"fmt"
	"github.com/gigurra/ai/domain"
	"github.com/samber/lo"
	"github.com/sashabaranov/go-openai"
	"sort"
)

type Config struct {
	APIKey       string  `yaml:"api_key"`
	Organization string  `yaml:"organization"`
	Project      string  `yaml:"project"`
	Temperature  float64 `yaml:"temperature"`
	Model        string  `yaml:"model"`
}

type Provider struct {
	cfg    Config
	client *openai.Client
}

type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type BasicAskRequest struct {
	Model       string    `json:"model"`
	Messages    []Message `json:"messages"`
	Temperature float64   `json:"temperature"`
}

type Choice struct {
	Index        int     `json:"index"`
	Message      Message `json:"message"`
	LogProbs     any     `json:"logprobs"`
	FinishReason string  `json:"finish_reason"`
}

type BasicAskUsage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

type BasicAskResponse struct {
	ID                string        `json:"id"`
	Object            string        `json:"object"` // will be set to "chat.completion"
	Created           int64         `json:"created"`
	Model             string        `json:"model"`
	Choices           []Choice      `json:"choices"`
	Usage             BasicAskUsage `json:"usage"`
	SystemFingerprint any           `json:"system_fingerprint"`
}

func (o BasicAskResponse) GetID() string {
	return o.ID
}

func (o BasicAskResponse) GetObjectType() string {
	return o.Object
}

func (o BasicAskResponse) GetCreated() int64 {
	return o.Created
}

func (o BasicAskResponse) GetModel() string {
	return o.Model
}

func (o BasicAskResponse) GetChoices() []domain.Choice {
	return lo.Map(o.Choices, func(item Choice, index int) domain.Choice {
		return domain.Choice{
			Index: 0,
			Message: domain.Message{
				SourceType: domain.SourceType(item.Message.Role),
				Content:    item.Message.Content,
			},
		}
	})
}

func (o BasicAskResponse) GetUsage() domain.Usage {
	return domain.Usage{
		PromptTokens:     o.Usage.PromptTokens,
		CompletionTokens: o.Usage.CompletionTokens,
		TotalTokens:      o.Usage.TotalTokens,
	}
}

func (o BasicAskResponse) GetSystemFingerprint() any {
	return o.SystemFingerprint
}

// prove OpenAIBasicAskResponse implements the Response interface
var _ domain.Response = BasicAskResponse{}

func (o Provider) authHeaders() map[string]string {
	return filterOutEmptyValues(map[string]string{
		"Authorization":       "Bearer " + o.cfg.APIKey,
		"OpenAI-Organization": o.cfg.Organization,
		"OpenAI-Project":      o.cfg.Project,
	})
}

func (o Provider) BasicAsk(question domain.Question) (domain.Response, error) {

	res, err := o.client.CreateChatCompletion(
		context.Background(),
		openai.ChatCompletionRequest{
			Model: o.cfg.Model,
			Messages: lo.Map(question.Messages, func(message domain.Message, index int) openai.ChatCompletionMessage {
				return openai.ChatCompletionMessage{
					Role:    string(message.SourceType),
					Content: message.Content,
				}
			}),
		},
	)
	if err != nil {
		var zero BasicAskResponse
		return zero, fmt.Errorf("failed to ask question: %w", err)
	}

	resp := BasicAskResponse{
		ID:      res.ID,
		Object:  res.Object,
		Created: res.Created,
		Model:   res.Model,
		Choices: lo.Map(res.Choices, func(item openai.ChatCompletionChoice, index int) Choice {
			return Choice{
				Index:        item.Index,
				Message:      Message{Role: item.Message.Role, Content: item.Message.Content},
				LogProbs:     item.LogProbs,
				FinishReason: string(item.FinishReason),
			}
		}),
		Usage: BasicAskUsage{
			PromptTokens:     res.Usage.PromptTokens,
			CompletionTokens: res.Usage.CompletionTokens,
			TotalTokens:      res.Usage.TotalTokens,
		},
		SystemFingerprint: res.SystemFingerprint,
	}

	return resp, nil
}

// prove that OpenAIProvider implements the Provider interface
var _ domain.Provider = &Provider{}

func NewOpenAIProvider(cfg Config) *Provider {

	client := openai.NewClient(cfg.APIKey)

	return &Provider{
		cfg:    cfg,
		client: client,
	}
}

func filterOutEmptyValues(mapIn map[string]string) map[string]string {
	mapOut := make(map[string]string)
	for k, v := range mapIn {
		if v != "" {
			mapOut[k] = v
		}
	}
	return mapOut
}

func (o Provider) ListModels() ([]string, error) {

	res, err := o.client.ListModels(context.Background())
	if err != nil {
		return nil, fmt.Errorf("failed to list models: %w", err)
	}

	// sort the models by id
	sort.Slice(res.Models, func(i, j int) bool {
		return res.Models[i].ID < res.Models[j].ID
	})

	return lo.Map(res.Models, func(item openai.Model, index int) string {
		return item.ID
	}), nil
}
