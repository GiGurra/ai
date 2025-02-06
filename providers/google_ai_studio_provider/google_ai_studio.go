package google_ai_studio_provider

import (
	"fmt"
	"github.com/gigurra/ai/common"
	"github.com/gigurra/ai/domain"
	"github.com/gigurra/ai/providers/google_common"
	"net/url"
	"strings"
)

type Config struct {
	APIKey          string  `yaml:"api_key"`
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
	cfg Config
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

	endpointUrl, err := url.Parse(fmt.Sprintf(
		"https://generativelanguage.googleapis.com/v1beta/models/%s:generateContent",
		o.cfg.ModelId,
	))
	if err != nil {
		common.FailAndExit(1, fmt.Sprintf("failed to parse URL: %v", err))
	}

	q := endpointUrl.Query()
	q.Set("alt", "sse")
	q.Set("key", o.cfg.APIKey)
	endpointUrl.RawQuery = q.Encode()

	cfg := &google_common.Config{
		ModelId:         o.cfg.ModelId,
		MaxOutputTokens: o.cfg.MaxOutputTokens,
		Temperature:     o.cfg.Temperature,
		TopP:            o.cfg.TopP,
		TopK:            o.cfg.TopK,
		Verbose:         o.cfg.Verbose,
	}

	return google_common.BasicAskStream(
		endpointUrl,
		"",
		cfg,
		question,
	)
}

// prove that OpenAIProvider implements the Provider interface
var _ domain.Provider = &Provider{}

func NewGoogleAiStudioProvider(cfg Config, verbose bool) *Provider {
	return &Provider{
		cfg: cfg.WithVerbose(verbose),
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
