package openai

import (
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
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

func ListModels(cfg OpenAIConfig) (OpenAIModelListing, error) {
	url := baseUrl + "models"

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return OpenAIModelListing{}, fmt.Errorf(".ListModels(..): failed to create request: %w", err)
	}

	if cfg.APIKey == "" {
		return OpenAIModelListing{}, fmt.Errorf(".ListModels(..): API key is required, but none was provided")
	}
	req.Header.Set("Authorization", "Bearer "+cfg.APIKey)
	if cfg.Organization != "" {
		req.Header.Set("OpenAI-Organization", cfg.Organization)
	}
	if cfg.Project != "" {
		req.Header.Set("OpenAI-Project", cfg.Project)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return OpenAIModelListing{}, fmt.Errorf(".ListModels(..): failed to send request: %w", err)
	}

	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			slog.Error("Failed to close response body: %v", err)
		}
	}(resp.Body)

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return OpenAIModelListing{}, fmt.Errorf(".ListModels(..): failed to read response: %w", err)
	}

	listing := OpenAIModelListing{}
	err = json.Unmarshal(bodyBytes, &listing)
	if err != nil {
		return OpenAIModelListing{}, fmt.Errorf(".ListModels(..): failed to unmarshal response body: %w", err)
	}

	return listing, nil
}
