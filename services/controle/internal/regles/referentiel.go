package regles

const euro int64 = 100

var referenceCatalogue = Catalogue{
	"bandes-dessinees":       {MinCents: 1 * euro, MaxCents: 5_000 * euro, VigilanceCents: 300 * euro},
	"cartes-a-collectionner": {MinCents: euro / 2, MaxCents: 10_000 * euro, VigilanceCents: 500 * euro},
	"figurines":              {MinCents: 1 * euro, MaxCents: 3_000 * euro, VigilanceCents: 300 * euro},
	"jouets-anciens":         {MinCents: 2 * euro, MaxCents: 5_000 * euro, VigilanceCents: 400 * euro},
	"monnaies":               {MinCents: 1 * euro, MaxCents: 20_000 * euro, VigilanceCents: 1_000 * euro},
	"montres":                {MinCents: 20 * euro, MaxCents: 50_000 * euro, VigilanceCents: 2_000 * euro},
	"vinyles":                {MinCents: 1 * euro, MaxCents: 2_000 * euro, VigilanceCents: 200 * euro},
}

var referenceSeniority = SeniorityPolicy{MinimumDays: 30}

func Reference() Rules {
	return New(referenceCatalogue, referenceSeniority)
}
