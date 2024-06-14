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
	Contents         []Content         `json:"contents"`
	GenerationConfig *GenerationConfig `json:"generationConfig,omitempty"`
	SafetySettings   []SafetySetting   `json:"safetySettings,omitempty"`
}

/*
[{
  "candidates": [
    {
      "content": {
        "role": "model",
        "parts": [
          {
            "text": "Banana"
          }
        ]
      }
    }
  ]
}
,
{
  "candidates": [
    {
      "content": {
        "role": "model",
        "parts": [
          {
            "text": " 🍌 \n"
          }
        ]
      },
      "safetyRatings": [
        {
          "category": "HARM_CATEGORY_HATE_SPEECH",
          "probability": "NEGLIGIBLE",
          "probabilityScore": 0.06108711,
          "severity": "HARM_SEVERITY_NEGLIGIBLE",
          "severityScore": 0.12231338
        },
        {
          "category": "HARM_CATEGORY_DANGEROUS_CONTENT",
          "probability": "NEGLIGIBLE",
          "probabilityScore": 0.16158468,
          "severity": "HARM_SEVERITY_NEGLIGIBLE",
          "severityScore": 0.19498022
        },
        {
          "category": "HARM_CATEGORY_HARASSMENT",
          "probability": "NEGLIGIBLE",
          "probabilityScore": 0.110471144,
          "severity": "HARM_SEVERITY_NEGLIGIBLE",
          "severityScore": 0.09619517
        },
        {
          "category": "HARM_CATEGORY_SEXUALLY_EXPLICIT",
          "probability": "NEGLIGIBLE",
          "probabilityScore": 0.2605776,
          "severity": "HARM_SEVERITY_NEGLIGIBLE",
          "severityScore": 0.18126321
        }
      ]
    }
  ]
}
,
{
  "candidates": [
    {
      "content": {
        "role": "model",
        "parts": [
          {
            "text": ""
          }
        ]
      },
      "finishReason": "STOP"
    }
  ],
  "usageMetadata": {
    "promptTokenCount": 3,
    "candidatesTokenCount": 5,
    "totalTokenCount": 8
  }
}
]
*/

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
	_ = RequestData{
		Contents: lo.Map(question.Messages, func(m domain.Message, _ int) Content {
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
		Contents: lo.Map(question.Messages, func(m domain.Message, _ int) Content {
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
		fmt.Printf("------------NEW MESSAGE-----------\n")
		fmt.Printf("%+v\n ", content)
	}
	//_, err = io.Copy(os.Stdout, res.Body)

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
