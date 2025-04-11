package google_common

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/bcicen/jstream"
	"github.com/gigurra/ai/common"
	"github.com/gigurra/ai/domain"
	"github.com/samber/lo"
	"io"
	"log/slog"
	"net/http"
	"net/url"
)

type Config struct {
	ModelId         string  `yaml:"model_id"`
	MaxOutputTokens int     `yaml:"max_output_tokens"`
	Temperature     float64 `yaml:"temperature"`
	TopP            float64 `yaml:"top_p"`
	TopK            float64 `yaml:"top_k"`
	Verbose         bool    `yaml:"verbose"`
}

type Content struct {
	Role  string `json:"role"`
	Parts []Part `json:"parts"`
}

type Part struct {
	Text string `json:"text"`
}

type GenerationConfig struct {
	MaxOutputTokens int     `json:"maxOutputTokens"`
	Temperature     float64 `json:"temperature"`
	TopP            float64 `json:"topP"`
	TopK            float64 `json:"topK"`
}

type RequestData struct {
	Contents         []Content         `json:"contents"`
	GenerationConfig *GenerationConfig `json:"generationConfig,omitempty"`
	SafetySettings   []SafetySetting   `json:"safetySettings,omitempty"`
}

type SafetySetting struct {
	Category  string `json:"category"`
	Threshold string `json:"threshold"`
}

type ContentResponse struct {
	Candidates    []Candidate   `json:"candidates"`
	UsageMetadata UsageMetadata `json:"usageMetadata,omitempty"`
}

type UsageMetadata struct {
	PromptTokenCount     int `json:"promptTokenCount"`
	CandidatesTokenCount int `json:"candidatesTokenCount"`
	TotalTokenCount      int `json:"totalTokenCount"`
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

func BasicAskStream(
	endpointUrl *url.URL,
	authHeader string,
	cfg *Config,
	question domain.Question,
) <-chan domain.RespChunk {

	respChan := make(chan domain.RespChunk)

	go func() {
		defer close(respChan)
		bodyT := RequestData{
			Contents: lo.Map(question.Messages, func(m domain.Message, _ int) Content {
				return Content{
					Role: DomainRoleToGoogleRole(m.SourceType),
					Parts: []Part{{
						Text: m.Content,
					}},
				}
			}),
			GenerationConfig: &GenerationConfig{
				MaxOutputTokens: common.CfgOrDefaultI(cfg.MaxOutputTokens, 8192),
				Temperature:     common.CfgOrDefaultF(cfg.Temperature, 0.1),
				TopP:            common.CfgOrDefaultF(cfg.TopP, 1.0),
				TopK:            common.CfgOrDefaultF(cfg.TopK, 40.0),
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

		if cfg.Verbose {
			slog.Info(fmt.Sprintf("GenerationConfig: %+v", bodyT.GenerationConfig))
		}

		bodyBytes, err := json.Marshal(bodyT)
		if err != nil {
			panic(fmt.Sprintf("Failed to marshal body: %v", err))
		}
		bodyReadCloser := io.NopCloser(bytes.NewReader(bodyBytes))

		headers := http.Header{
			"Content-Type": []string{"application/json"},
		}
		if authHeader != "" {
			headers["Authorization"] = []string{authHeader}
		}

		request := http.Request{
			Method:        "POST",
			URL:           endpointUrl,
			Header:        headers,
			Body:          bodyReadCloser,
			ContentLength: int64(len(bodyBytes)),
			Host:          endpointUrl.Host,
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

		decoder := jstream.NewDecoder(res.Body, 1)
		for mv := range decoder.Stream() {
			jsonRepr, _ := json.Marshal(mv.Value)
			if cfg.Verbose {
				slog.Info(fmt.Sprintf("[[RESPONSE DATA CHUNK]]: %s", string(jsonRepr)))
			}
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

			Usage := domain.Usage{}
			if firstCandidate.FinishReason == "STOP" {
				Usage = domain.Usage{
					PromptTokens:     content.UsageMetadata.PromptTokenCount,
					CompletionTokens: content.UsageMetadata.CandidatesTokenCount,
					TotalTokens:      content.UsageMetadata.TotalTokenCount,
				}
			}

			text := func() string {
				if len(firstCandidate.Content.Parts) > 0 {
					return firstCandidate.Content.Parts[0].Text
				}
				return ""
			}()

			respChan <- domain.RespChunk{
				Resp: &RespImpl{
					Choices: []domain.Choice{
						{
							Index: 0,
							Message: domain.Message{
								SourceType: GoogleToDomainRole(firstCandidate.Content.Role),
								Content:    text,
							},
						},
					},
					Usage: Usage,
				},
			}
		}
	}()

	return respChan
}

func DomainRoleToGoogleRole(role domain.SourceType) string {
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

func GoogleToDomainRole(role string) domain.SourceType {
	switch role {
	case "user":
		return domain.User
	case "model":
		return domain.Assistant
	default:
		panic(fmt.Sprintf("Unknown google role: %s", role))
	}
}
