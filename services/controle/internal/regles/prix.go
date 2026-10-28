package regles

type PriceRange struct {
	MinCents       int64
	MaxCents       int64
	VigilanceCents int64
}

func (p PriceRange) accepts(priceCents int64) bool {
	return priceCents >= p.MinCents && priceCents <= p.MaxCents
}

func (p PriceRange) exceedsVigilance(priceCents int64) bool {
	return priceCents > p.VigilanceCents
}
