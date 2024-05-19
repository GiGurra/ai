package openai

import (
	"fmt"
	"io"
	"log/slog"
	"net/http"
)

type OpenAIConfig struct {
	APIKey       string
	Organization string
	Project      string
}

const baseUrl = "https://api.openai.com/v1/"

func ListModels(cfg OpenAIConfig) (string, error) {
	url := baseUrl + "models"

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return "", fmt.Errorf(".ListModels(..): failed to create request: %w", err)
	}

	if cfg.APIKey == "" {
		return "", fmt.Errorf(".ListModels(..): API key is required, but none was provided")
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
		return "", fmt.Errorf(".ListModels(..): failed to send request: %w", err)
	}

	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			slog.Error("Failed to close response body: %v", err)
		}
	}(resp.Body)

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf(".ListModels(..): failed to read response: %w", err)
	}

	return string(bodyBytes), nil
}
