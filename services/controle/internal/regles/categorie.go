package regles

import "maps"

type Catalogue map[CategoryCode]PriceRange

func (c Catalogue) priceRangeOf(code CategoryCode) (PriceRange, bool) {
	priceRange, known := c[code]
	return priceRange, known
}

func (c Catalogue) clone() Catalogue {
	return maps.Clone(c)
}
