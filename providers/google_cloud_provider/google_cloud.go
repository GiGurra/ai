package google_cloud_provider

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"github.com/GiGurra/cmder"
	"github.com/bcicen/jstream"
	"github.com/gigurra/ai/common"
	"github.com/gigurra/ai/domain"
	"github.com/samber/lo"
	"io"
	"log/slog"
	"net/http"
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

type GenerationConfig struct {
	MaxOutputTokens int     `json:"maxOutputTokens"`
	Temperature     float64 `json:"temperature"`
	TopP            float64 `json:"topP"`
	TopK            float64 `json:"topK"`
}

type SafetySetting struct {
	Category  string `json:"category"`
	Threshold string `json:"threshold"`
}

type Content struct {
	Role  string `json:"role"`
	Parts []Part `json:"parts"`
}

type Part struct {
	Text string `json:"text"`
}

type RequestData struct {
	Contents         []Content         `json:"contents"`
	GenerationConfig *GenerationConfig `json:"generationConfig,omitempty"`
	SafetySettings   []SafetySetting   `json:"safetySettings,omitempty"`
}

type UsageMetadata struct {
	PromptTokenCount     int `json:"promptTokenCount"`
	CandidatesTokenCount int `json:"candidatesTokenCount"`
	TotalTokenCount      int `json:"totalTokenCount"`
}

type ContentResponse struct {
	Candidates    []Candidate   `json:"candidates"`
	UsageMetadata UsageMetadata `json:"usageMetadata,omitempty"`
}

func domainRoleToGoogleRole(role domain.SourceType) string {
	switch role {
	case domain.System:
		return "user"
	case domain.User:
		return "user"
	case domain.Assistant:
		return "model"
	default:
		panic("Unknown role")
	}
}

func googleToDomainRole(role string) domain.SourceType {
	switch role {
	case "user":
		return domain.User
	case "model":
		return domain.Assistant
	default:
		panic(fmt.Sprintf("Unknown google role: %s", role))
	}
}

type SafetyRating struct {
	Category         string  `json:"category"`
	Probability      string  `json:"probability"`
	ProbabilityScore float64 `json:"probabilityScore"`
	Severity         string  `json:"severity"`
	SeverityScore    float64 `json:"severityScore"`
}

type Candidate struct {
	Content       Content        `json:"content"`
	SafetyRatings []SafetyRating `json:"safetyRatings,omitempty"`
	FinishReason  string         `json:"finishReason,omitempty"`
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

	return &RespImpl{
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

func cfgOrDefaultF(cfgVal float64, defaultVal float64) float64 {
	if cfgVal > 0 {
		return cfgVal
	}
	return defaultVal
}

func cfgOrDefaultI(cfgVal int, defaultVal int) int {
	if cfgVal != 0 {
		return cfgVal
	}
	return defaultVal
}

func (o Provider) BasicAskStream(question domain.Question) <-chan domain.RespChunk {

	respChan := make(chan domain.RespChunk)

	go func() {
		defer close(respChan)
		bodyT := RequestData{
			Contents: lo.Map(question.Messages, func(m domain.Message, _ int) Content {
				return Content{
					Role: domainRoleToGoogleRole(m.SourceType),
					Parts: []Part{{
						Text: m.Content,
					}},
				}
			}),
			GenerationConfig: &GenerationConfig{
				MaxOutputTokens: cfgOrDefaultI(o.cfg.MaxOutputTokens, 8192),
				Temperature:     cfgOrDefaultF(o.cfg.Temperature, 0.1),
				TopP:            cfgOrDefaultF(o.cfg.TopP, 1.0),
				TopK:            cfgOrDefaultF(o.cfg.TopK, 40.0),
			},
			//SafetySettings: []SafetySetting{
			//	{
			//		Category:  "HARM_CATEGORY_HATE_SPEECH",
			//		Threshold: "BLOCK_MEDIUM_AND_ABOVE",
			//	},
			//	{
			//		Category:  "HARM_CATEGORY_DANGEROUS_CONTENT",
			//		Threshold: "BLOCK_MEDIUM_AND_ABOVE",
			//	},
			//	{
			//		Category:  "HARM_CATEGORY_SEXUALLY_EXPLICIT",
			//		Threshold: "BLOCK_MEDIUM_AND_ABOVE",
			//	},
			//	{
			//		Category:  "HARM_CATEGORY_HARASSMENT",
			//		Threshold: "BLOCK_MEDIUM_AND_ABOVE",
			//	},
			//},
		}

		if o.cfg.Verbose {
			slog.Info(fmt.Sprintf("GenerationConfig: %+v", bodyT.GenerationConfig))
		}

		bodyBytes, err := json.Marshal(bodyT)
		if err != nil {
			panic(fmt.Sprintf("Failed to marshal body: %v", err))
		}

		bodyReadCloser := io.NopCloser(bytes.NewReader(bodyBytes))

		host := fmt.Sprintf("%s-aiplatform.googleapis.com", o.cfg.LocationID)
		apiEndpoint := fmt.Sprintf("https://%s", host)
		u, err := url.Parse(fmt.Sprintf("%s/v1/projects/%s/locations/%s/publishers/google/models/%s:streamGenerateContent", apiEndpoint, o.cfg.ProjectID, o.cfg.LocationID, o.cfg.ModelId))
		if err != nil {
			panic(fmt.Sprintf("Failed to parse url: %v", err))
		}

		request := http.Request{
			Method: "POST",
			URL:    u,
			Header: http.Header{
				"Authorization": []string{fmt.Sprintf("Bearer %s", o.accessToken)},
				"Content-Type":  []string{"application/json"},
			},
			Body:          bodyReadCloser,
			ContentLength: int64(len(bodyBytes)),
			Host:          host,
		}

		res, err := http.DefaultClient.Do(&request)
		if err != nil {
			respBody, _ := io.ReadAll(res.Body)
			panic(fmt.Sprintf("Failed to do request: %v: %s", err, string(respBody)))
		}
		defer func() {
			err := res.Body.Close()
			if err != nil {
				slog.Error(fmt.Sprintf("Failed to close body: %v", err))
			}
		}()

		if res.StatusCode != 200 {
			respBody, _ := io.ReadAll(res.Body)
			panic(fmt.Sprintf("Failed to do request, unexpected status code: %v: %s", res.StatusCode, string(respBody)))
		}

		// print response body to stdout

		decoder := jstream.NewDecoder(res.Body, 1)
		for mv := range decoder.Stream() {
			jsonRepr, _ := json.Marshal(mv.Value)
			// read back as Content struct
			var content ContentResponse
			err := json.Unmarshal(jsonRepr, &content)
			if err != nil {
				panic(fmt.Sprintf("Failed to unmarshal response: %v", err))
			}
			if len(content.Candidates) == 0 {
				panic(fmt.Sprintf("No candidates in response"))
			}
			firstCandidate := content.Candidates[0]
			//if firstCandidate.FinishReason != "" {
			//	return
			//}
			respChan <- domain.RespChunk{
				Resp: &RespImpl{
					Choices: []domain.Choice{
						{
							Index: 0,
							Message: domain.Message{
								SourceType: googleToDomainRole(firstCandidate.Content.Role),
								Content:    firstCandidate.Content.Parts[0].Text,
							},
						},
					},
				},
			}
		}

	}()
	return respChan
}

type RespImpl struct {
	Choices []domain.Choice
	Usage   domain.Usage
}

func (r *RespImpl) GetChoices() []domain.Choice {
	return r.Choices
}

func (r *RespImpl) GetUsage() domain.Usage {
	return r.Usage
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
