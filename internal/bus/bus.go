package bus

type MessageHandler func(data []byte) error

type Subscriber interface {
	Start(subject string, handler MessageHandler) error
	Close()
}
