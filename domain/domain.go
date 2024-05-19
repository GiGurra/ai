package domain

type SourceType string

// Using openai naming here, we can change it later
const (
	System    SourceType = "system"
	User      SourceType = "user"
	Assistant SourceType = "assistant"
)

type Message struct {
	SourceType SourceType // system, user or assistant
	Content    string     // the message content
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
	GetID() string
	GetObjectType() string
	GetCreated() int64
	GetModel() string
	GetChoices() []Choice
	GetUsage() Usage
	GetSystemFingerprint() any
}

type Provider interface {
	ListModels() ([]string, error)

	// BasicAsk asks a question and returns the answer. The most primitive use case.
	BasicAsk(question Question) (Response, error)
	BasicAskStream(question Question) <-chan RespChunk
}
