package openai

import (
	"fmt"
	"github.com/gigurra/ai/domain"
	"github.com/gigurra/ai/util"
	"github.com/samber/lo"
	"sort"
)

type OpenAIConfig struct {
	APIKey       string  `yaml:"api_key"`
	Organization string  `yaml:"organization"`
	Project      string  `yaml:"project"`
	Temperature  float64 `yaml:"temperature"`
}

const baseUrl = "https://api.openai.com/v1/"

type OpenAIModelListing struct {
	Object string        `json:"object"` // will be set to "list"
	Data   []OpenAIModel `json:"data"`
}

type OpenAIModel struct {
	ID        string `json:"id"`
	Object    string `json:"object"` // will be set to "model"
	CreatedAt int    `json:"created"`
	OwnedBy   string `json:"owned_by"`
}

type OpenAIProvider struct {
	cfg OpenAIConfig
}

type OpenAIMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type OpenAIBasicAskRequest struct {
	Model       string          `json:"model"`
	Messages    []OpenAIMessage `json:"messages"`
	Temperature float64         `json:"temperature"`
}

func (o OpenAIProvider) BasicAsk(question domain.Question) (string, error) {
	//TODO implement me

	panic("implement me")
}

// prove that OpenAIProvider implements the Provider interface
var _ domain.Provider = &OpenAIProvider{}

func NewOpenAIProvider(cfg OpenAIConfig) *OpenAIProvider {
	return &OpenAIProvider{cfg: cfg}
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

func (o OpenAIProvider) ListModels() ([]string, error) {

	url := baseUrl + "models"

	listing, err := util.HttpGetRecvJson[OpenAIModelListing](url, util.GetParams{
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

	return lo.Map(listing.Data, func(item OpenAIModel, index int) string {
		return item.ID
	}), nil
}
