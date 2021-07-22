package gok

const (
	defaultErrorBusChanCapacity = 100
)

type GokError struct {
	Identifier string
	Err error
}

type ErrorBus struct {
	tag string
	ch chan *GokError
}

func (eb *ErrorBus) Tag() string {
	return eb.tag
}

func (eb *ErrorBus) Add(err *GokError) {
	eb.ch <- err
}

func (eb *ErrorBus) Next() *GokError {
	return <- eb.ch
}

func NewErrorBus(tag string) *ErrorBus {
	return &ErrorBus{
		tag: tag,
		ch: make(chan *GokError, defaultErrorBusChanCapacity),
	}
}