package bus

import (
	"log"
	"time"

	"github.com/nats-io/stan.go"
)

type StanSub struct {
	conn stan.Conn
}

func NewStan(cluster, clientID, url string) (*StanSub, error) {
	c, err := stan.Connect(cluster, clientID, stan.NatsURL(url))
	if err != nil {
		return nil, err
	}
	return &StanSub{conn: c}, nil
}

func (s *StanSub) Start(subject string, handler MessageHandler) error {
	_, err := s.conn.Subscribe(subject, func(m *stan.Msg) {
		if err := handler(m.Data); err != nil {
			log.Printf("handle error: %v", err)
			return
		}
		m.Ack()
	}, stan.SetManualAckMode(),
		stan.DurableName("orders-durable"),
		stan.AckWait(30*time.Second),
		stan.DeliverAllAvailable())
	if err == nil {
		log.Printf("stan subscribed to %s", subject)
	}
	return err
}

func (s *StanSub) Close() { s.conn.Close() }
