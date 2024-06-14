package google_cloud_provider

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"github.com/GiGurra/cmder"
	"github.com/gigurra/ai/common"
	"github.com/gigurra/ai/domain"
	"github.com/samber/lo"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"
)

type Config struct {
	ProjectID  string `yaml:"project_id"`
	LocationID string `yaml:"location_id"`
	ModelId    string `yaml:"model_id"`
}

type Provider struct {
	cfg         Config
	accessToken string
}

type GenerationConfig struct {
	MaxOutputTokens int     `json:"maxOutputTokens"`
	Temperature     float64 `json:"temperature"`
	TopP            float64 `json:"topP"`
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

/*
{
    "contents": [
        {
            "role": "user",
            "parts": [
                {
                    "text": "Please say banana"
                },
            ]
        }
    ],
    "generationConfig": {
        "maxOutputTokens": 8192,
        "temperature": 1,
        "topP": 0.95,
    },
    "safetySettings": [
        {
            "category": "HARM_CATEGORY_HATE_SPEECH",
            "threshold": "BLOCK_MEDIUM_AND_ABOVE"
        },
        {
            "category": "HARM_CATEGORY_DANGEROUS_CONTENT",
            "threshold": "BLOCK_MEDIUM_AND_ABOVE"
        },
        {
            "category": "HARM_CATEGORY_SEXUALLY_EXPLICIT",
            "threshold": "BLOCK_MEDIUM_AND_ABOVE"
        },
        {
            "category": "HARM_CATEGORY_HARASSMENT",
            "threshold": "BLOCK_MEDIUM_AND_ABOVE"
        }
    ],
}

export API_ENDPOINT="europe-west4-aiplatform.googleapis.com"
                                                    export PROJECT_ID="XYZ"
                                                    export LOCATION_ID="europe-west4"
                                                    export MODEL_ID="gemini-1.5-flash-001"

curl -X POST \
	-H "Authorization: Bearer $(gcloud auth print-access-token)" \
	-H "Content-Type: application/json" \
	"https://${API_ENDPOINT}/v1/projects/${PROJECT_ID}/locations/${LOCATION_ID}/publishers/google/models/${MODEL_ID}:streamGenerateContent" \
	-d '@request.json'

*/

type RequestData struct {
	Content          []Content         `json:"content"`
	GenerationConfig *GenerationConfig `json:"generationConfig,omitempty"`
	SafetySettings   []SafetySetting   `json:"safetySettings,omitempty"`
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

type Candidate struct {
	Content Content `json:"content"`
}

func (o Provider) BasicAsk(question domain.Question) (domain.Response, error) {
	_ = RequestData{
		Content: lo.Map(question.Messages, func(m domain.Message, _ int) Content {
			return Content{
				Role: domainRoleToGoogleRole(m.SourceType),
				Parts: []Part{{
					Text: m.Content,
				}},
			}
		}),
		GenerationConfig: &GenerationConfig{
			MaxOutputTokens: 8192,
			Temperature:     1,
			TopP:            0.95,
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

	panic("Not Implemented")
}

func (o Provider) BasicAskStream(question domain.Question) <-chan domain.RespChunk {
	bodyT := RequestData{
		Content: lo.Map(question.Messages, func(m domain.Message, _ int) Content {
			return Content{
				Role: domainRoleToGoogleRole(m.SourceType),
				Parts: []Part{{
					Text: m.Content,
				}},
			}
		}),
		GenerationConfig: &GenerationConfig{
			MaxOutputTokens: 8192,
			Temperature:     1,
			TopP:            0.95,
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

	bodyBytes, err := json.Marshal(bodyT)
	if err != nil {
		panic(fmt.Sprintf("Failed to marshal body: %v", err))
	}

	bodyReadCloser := io.NopCloser(bytes.NewReader(bodyBytes))

	host := fmt.Sprintf("%s-aiplatform.googleapis.com", o.cfg.LocationID)
	apiEndpoint := fmt.Sprintf("https://%s", host)
	u, err := url.Parse(fmt.Sprintf("https://%s/v1/projects/%s/locations/%s/publishers/google/models/%s:streamGenerateContent", apiEndpoint, o.cfg.ProjectID, o.cfg.LocationID, o.cfg.ModelId))
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
		panic(fmt.Sprintf("Failed to do request: %v", err))
	}
	defer func() {
		err := res.Body.Close()
		if err != nil {
			slog.Error(fmt.Sprintf("Failed to close body: %v", err))
		}
	}()

	if res.StatusCode != 200 {
		panic(fmt.Sprintf("Failed to do request, unexpected status code: %v", res.StatusCode))
	}

	// print response body to stdout
	_, err = io.Copy(os.Stdout, res.Body)

	time.Sleep(1 * time.Second)

	panic("TODO: Implement request return - Not Implemented")
}

// prove that OpenAIProvider implements the Provider interface
var _ domain.Provider = &Provider{}

func NewGoogleCloudProvider(cfg Config, _ bool) *Provider {
	// get access token. TODO: Use a library instead to lower the dependency on gcloud and latencies
	res := cmder.New("gcloud", "auth", "print-access-token").Run(context.Background())
	if res.Err != nil {
		common.FailAndExit(1, "Failed to get access token with gcloud. Check if you are logged in.")
	}
	return &Provider{
		cfg:         cfg,
		accessToken: strings.TrimSpace(res.StdOut),
	}
}

func (o Provider) ListModels() ([]string, error) {
	return []string{
		"gemini-1.5-flash-001",
		"gemini-1.5-pro-001",
		"gemini-experimental",
	}, nil
}
