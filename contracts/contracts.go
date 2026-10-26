package contracts

import "embed"

//go:embed messages/*.schema.json messages/*/*.schema.json
var Messages embed.FS
