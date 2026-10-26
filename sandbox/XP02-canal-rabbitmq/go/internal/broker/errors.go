package broker

import (
	"errors"

	amqp "github.com/rabbitmq/amqp091-go"
)

var (
	ErrConnectionLost = errors.New("connexion au broker perdue")
	ErrRejected       = errors.New("publication refusée par le broker")
)

func CloseReason(reason *amqp.Error) error {
	if reason == nil {
		return ErrConnectionLost
	}
	return reason
}

func Fault(err error) (code int, reason string, ok bool) {
	var amqpErr *amqp.Error
	if !errors.As(err, &amqpErr) {
		return 0, "", false
	}
	switch amqpErr.Code {
	case amqp.AccessRefused, amqp.NotFound, amqp.PreconditionFailed, amqp.NotAllowed:
		return amqpErr.Code, amqpErr.Reason, true
	}
	return 0, "", false
}

func IsFault(err error) bool {
	_, _, ok := Fault(err)
	return ok
}
