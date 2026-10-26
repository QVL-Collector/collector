module collector/services/controle

go 1.26.0

toolchain go1.26.10

require (
	collector/contracts v0.0.0
	collector/libs/platform v0.0.0
	golang.org/x/sync v0.24.0
)

require (
	github.com/google/uuid v1.6.0 // indirect
	github.com/rabbitmq/amqp091-go v1.16.0 // indirect
	github.com/santhosh-tekuri/jsonschema/v6 v6.0.2 // indirect
	golang.org/x/text v0.38.0 // indirect
)

replace collector/libs/platform => ../../libs/platform

replace collector/contracts => ../../contracts
