package broker

import (
	"errors"

	amqp "github.com/rabbitmq/amqp091-go"
)

var (
	ErrConnectionLost = errors.New("connexion au broker perdue")
	ErrNotConnected   = errors.New("broker non connecté")
	ErrRejected       = errors.New("publication refusée par le broker")
	ErrUnroutable     = errors.New("publication sans file de destination")
)

func CloseReason(reason *amqp.Error) error {
	if reason == nil {
		return ErrConnectionLost
	}
	return reason
}

func IsFault(err error) bool {
	var amqpErr *amqp.Error
	if !errors.As(err, &amqpErr) {
		return false
	}
	switch amqpErr.Code {
	case amqp.AccessRefused, amqp.NotFound, amqp.PreconditionFailed, amqp.NotAllowed:
		return true
	}
	return false
}
