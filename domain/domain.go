package domain

import (
	"fmt"
	"github.com/gigurra/ai/common"
	"gopkg.in/yaml.v3"
)

type SourceType string

// Using openai naming here, we can change it later
const (
	System    SourceType = "system"
	User      SourceType = "user"
	Assistant SourceType = "assistant"
)

type Message struct {
	SourceType SourceType `yaml:"source_type"` // system, user or assistant
	Content    string     `yaml:"content"`     // the message content
}

func (m Message) ToYaml() string {
	bytes, err := yaml.Marshal(m)
	if err != nil {
		common.FailAndExit(1, fmt.Sprintf("Failed to marshal message to yaml: %v", err))
	}
	return string(bytes)
}

type Question struct {
	Messages []Message
}

type RespChunk struct {
	Resp Response
	Err  error
}

type Choice struct {
	Index   int
	Message Message
}

type Usage struct {
	PromptTokens     int
	CompletionTokens int
	TotalTokens      int
}

type Response interface {
	GetChoices() []Choice
	GetUsage() Usage
}

type Provider interface {
	ListModels() ([]string, error)

	// BasicAsk asks a question and returns the answer. The most primitive use case.
	BasicAsk(question Question) (Response, error)
	BasicAskStream(question Question) <-chan RespChunk
}
