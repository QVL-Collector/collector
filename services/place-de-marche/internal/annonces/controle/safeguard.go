package controle

import (
	"context"
	"log/slog"
	"time"

	"collector/services/place-de-marche/internal/annonces"
)

const (
	overdueAfter  = 2 * time.Minute
	sweepInterval = 30 * time.Second
	sweepBatch    = 50
)

type OverdueReviews interface {
	ResendOverdueReviews(ctx context.Context, overdue time.Duration, limit int) ([]annonces.Article, error)
}

type Safeguard struct {
	reviews OverdueReviews
	logger  *slog.Logger
}

func NewSafeguard(reviews OverdueReviews, logger *slog.Logger) *Safeguard {
	return &Safeguard{reviews: reviews, logger: logger}
}

func (s *Safeguard) Run(ctx context.Context) error {
	ticker := time.NewTicker(sweepInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
			s.sweep(ctx)
		}
	}
}

func (s *Safeguard) sweep(ctx context.Context) {
	for {
		resent, err := s.reviews.ResendOverdueReviews(ctx, overdueAfter, sweepBatch)
		if err != nil {
			s.logFailure(ctx, err)
			return
		}
		s.logResent(ctx, resent)
		if len(resent) < sweepBatch {
			return
		}
	}
}

func (s *Safeguard) logResent(ctx context.Context, resent []annonces.Article) {
	for _, article := range resent {
		s.logger.WarnContext(ctx, "contrôle sans résultat après le délai, demande réémise",
			slog.String("event", "controle.demande_relancee"),
			slog.String("article_id", article.ID.String()),
			slog.Int("revision", int(article.Revision)),
		)
	}
}

func (s *Safeguard) logFailure(ctx context.Context, err error) {
	if ctx.Err() != nil {
		return
	}
	s.logger.WarnContext(ctx, "relance des contrôles en retard impossible",
		slog.String("event", "controle.relance_en_echec"),
		slog.Any("erreur", err),
	)
}
