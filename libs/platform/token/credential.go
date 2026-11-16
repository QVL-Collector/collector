package token

import "log/slog"

const hiddenValue = "[masqué]"

type Credential string

func (c Credential) Reveal() string {
	return string(c)
}

func (c Credential) String() string {
	return hiddenValue
}

func (c Credential) GoString() string {
	return hiddenValue
}

func (c Credential) LogValue() slog.Value {
	return slog.StringValue(hiddenValue)
}
