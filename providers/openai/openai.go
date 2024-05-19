package openai

import (
	"fmt"
	"github.com/gigurra/ai/domain"
	"github.com/gigurra/ai/util"
	"github.com/samber/lo"
	"sort"
)

type Config struct {
	APIKey       string  `yaml:"api_key"`
	Organization string  `yaml:"organization"`
	Project      string  `yaml:"project"`
	Temperature  float64 `yaml:"temperature"`
	Model        string  `yaml:"model"`
}

const baseUrl = "https://api.openai.com/v1/"

type ModelListing struct {
	Object string  `json:"object"` // will be set to "list"
	Data   []Model `json:"data"`
}

type Model struct {
	ID        string `json:"id"`
	Object    string `json:"object"` // will be set to "model"
	CreatedAt int    `json:"created"`
	OwnedBy   string `json:"owned_by"`
}

type Provider struct {
	cfg Config
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
	Created           int           `json:"created"`
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

func (o BasicAskResponse) GetCreated() int {
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

	url := baseUrl + "chat/completions"

	requestData := BasicAskRequest{
		Model: o.cfg.Model,
		Messages: lo.Map(question.Messages, func(message domain.Message, index int) Message {
			return Message{
				Role:    string(message.SourceType),
				Content: message.Content,
			}
		}),
		Temperature: o.cfg.Temperature,
	}

	resp, err := util.HttpPostRecvJson[BasicAskResponse](url, util.PostParams{
		Headers: o.authHeaders(),
		Body:    requestData,
	})
	if err != nil {
		var zero BasicAskResponse
		return zero, fmt.Errorf("failed to ask question: %w", err)
	}

	return resp, nil
}

// prove that OpenAIProvider implements the Provider interface
var _ domain.Provider = &Provider{}

func NewOpenAIProvider(cfg Config) *Provider {
	return &Provider{cfg: cfg}
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

	url := baseUrl + "models"

	listing, err := util.HttpGetRecvJson[ModelListing](url, util.GetParams{
		Headers: filterOutEmptyValues(map[string]string{
			"Authorization":       "Bearer " + o.cfg.APIKey,
			"OpenAI-Organization": o.cfg.Organization,
			"OpenAI-Project":      o.cfg.Project,
		}),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list models: %w", err)
	}

	// sort the models by id
	sort.Slice(listing.Data, func(i, j int) bool {
		return listing.Data[i].ID < listing.Data[j].ID
	})

	return lo.Map(listing.Data, func(item Model, index int) string {
		return item.ID
	}), nil
}
