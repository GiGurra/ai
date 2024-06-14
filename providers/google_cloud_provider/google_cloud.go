package google_cloud_provider

import (
	"context"
	"github.com/GiGurra/cmder"
	"github.com/gigurra/ai/common"
	"github.com/gigurra/ai/domain"
	"strings"
)

type Config struct {
	ApiEndpoint string `yaml:"api_endpoint"`
	ProjectID   string `yaml:"project_id"`
	LocationID  string `yaml:"location_id"`
	ModelId     string `yaml:"model_id"`
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

func (o Provider) BasicAsk(question domain.Question) (domain.Response, error) {
	panic("Not Implemented")
}

func (o Provider) BasicAskStream(question domain.Question) <-chan domain.RespChunk {
	panic("Not Implemented")
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
