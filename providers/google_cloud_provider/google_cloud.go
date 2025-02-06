package google_cloud_provider

import (
	"context"
	"fmt"
	"github.com/GiGurra/cmder"
	"github.com/gigurra/ai/common"
	"github.com/gigurra/ai/domain"
	"github.com/gigurra/ai/providers/google_common"
	"net/url"
	"strings"
)

type Config struct {
	ProjectID       string  `yaml:"project_id"`
	LocationID      string  `yaml:"location_id"`
	ModelId         string  `yaml:"model_id"`
	MaxOutputTokens int     `yaml:"max_output_tokens"`
	Temperature     float64 `yaml:"temperature"`
	TopP            float64 `yaml:"top_p"`
	TopK            float64 `yaml:"top_k"`
	Verbose         bool    `yaml:"verbose"`
}

func (c Config) WithVerbose(verbose bool) Config {
	c.Verbose = verbose
	return c
}

type Provider struct {
	cfg         Config
	accessToken string
}

func (o Provider) BasicAsk(question domain.Question) (domain.Response, error) {
	stream := o.BasicAskStream(question)
	acc := strings.Builder{}
	for respChunk := range stream {
		if respChunk.Err != nil {
			return nil, respChunk.Err
		}
		cs := respChunk.Resp.GetChoices()
		if len(cs) != 1 {
			return nil, fmt.Errorf("expected exactly one choice")
		}
		acc.WriteString(cs[0].Message.Content)
	}

	return &google_common.RespImpl{
		Choices: []domain.Choice{
			{
				Index: 0,
				Message: domain.Message{
					SourceType: domain.System,
					Content:    acc.String(),
				},
			},
		},
	}, nil
}

func (o Provider) BasicAskStream(question domain.Question) <-chan domain.RespChunk {

	host := fmt.Sprintf("%s-aiplatform.googleapis.com", o.cfg.LocationID)
	baseUrl := fmt.Sprintf("https://%s", host)
	endpointUrl, err := url.Parse(
		fmt.Sprintf(
			"%s/v1/projects/%s/locations/%s/publishers/google/models/%s:streamGenerateContent",
			baseUrl, o.cfg.ProjectID, o.cfg.LocationID, o.cfg.ModelId,
		),
	)
	if err != nil {
		common.FailAndExit(1, fmt.Sprintf("failed to parse URL: %v", err))
	}

	cfg := &google_common.Config{
		ModelId:         o.cfg.ModelId,
		MaxOutputTokens: o.cfg.MaxOutputTokens,
		Temperature:     o.cfg.Temperature,
		TopP:            o.cfg.TopP,
		TopK:            o.cfg.TopK,
		Verbose:         o.cfg.Verbose,
	}

	authHeader := fmt.Sprintf("Bearer %s", o.accessToken)

	return google_common.BasicAskStream(
		endpointUrl,
		authHeader,
		cfg,
		question,
	)
}

// prove that OpenAIProvider implements the Provider interface
var _ domain.Provider = &Provider{}

func NewGoogleCloudProvider(cfg Config, Verbose bool) *Provider {
	// get access token. TODO: Use a library instead to lower the dependency on gcloud and latencies
	res := cmder.New("gcloud", "auth", "print-access-token").Run(context.Background())
	if res.Err != nil {
		common.FailAndExit(1, "Failed to get access token with gcloud. Check if you are logged in.")
	}
	return &Provider{
		cfg:         cfg.WithVerbose(Verbose),
		accessToken: strings.TrimSpace(res.StdOut),
	}
}

func (o Provider) ListModels() ([]string, error) {
	return []string{
		"gemini-1.5-flash-001",
		"gemini-1.5-flash-002",
		"gemini-1.5-pro-001",
		"gemini-experimental",
		"gemini-2.0-flash-001",
	}, nil
}
