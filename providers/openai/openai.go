package openai

import (
	"encoding/json"
	"fmt"
	"github.com/gigurra/ai/domain"
	"github.com/gigurra/ai/util"
	"github.com/samber/lo"
	"sort"
)

type OpenAIConfig struct {
	APIKey       string `yaml:"api_key"`
	Organization string `yaml:"organization"`
	Project      string `yaml:"project"`
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

func (o OpenAIProvider) BasicAsk(question string) (string, error) {
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

	res, err := util.HttpClient.R().SetHeaders(filterOutEmptyValues(map[string]string{
		"Authorization":       "Bearer " + o.cfg.APIKey,
		"OpenAI-Organization": o.cfg.Organization,
		"OpenAI-Project":      o.cfg.Project,
	})).Get(url)

	if err != nil {
		return nil, fmt.Errorf(".ListModels(..): failed to send request: %w", err)
	}

	if res.StatusCode() != 200 {
		return nil, fmt.Errorf(".ListModels(..): Unexpected status code: %d", res.StatusCode())
	}

	listing := OpenAIModelListing{}
	err = json.Unmarshal(res.Body(), &listing)
	if err != nil {
		return nil, fmt.Errorf(".ListModels(..): failed to unmarshal response body: %w", err)
	}

	// sort the models by id
	sort.Slice(listing.Data, func(i, j int) bool {
		return listing.Data[i].ID < listing.Data[j].ID
	})

	return lo.Map(listing.Data, func(item OpenAIModel, index int) string {
		return item.ID
	}), nil
}
