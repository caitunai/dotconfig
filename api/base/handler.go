package base

type Handler interface {
	GetHandler(action string) HandlerFunc
	Clone() Handler
}

type BasicHandler struct {
	Handlers map[string]HandlerFunc
}

func (b *BasicHandler) SetHandler(action string, handlerFunc HandlerFunc) {
	if b.Handlers == nil {
		b.Handlers = make(map[string]HandlerFunc)
	}
	b.Handlers[action] = handlerFunc
}

func (b *BasicHandler) GetHandler(action string) HandlerFunc {
	return b.Handlers[action]
}
