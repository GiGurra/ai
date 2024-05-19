package domain

type Message struct {
	SourceType string // system, user or response
	Content    string // the message content
}

type Question struct {
	Messages []Message
}

type Provider interface {
	ListModels() ([]string, error)

	// BasicAsk asks a question and returns the answer. The most primitive use case.
	BasicAsk(question Question) (string, error)
}
