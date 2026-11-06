package controle

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"collector/services/place-de-marche/internal/annonces"
)

const sweepBatch = 50

var errInvalidSafeguardTiming = errors.New("rythme du filet invalide : délai initial, plafond et période positifs, plafond au moins égal au délai initial")

type SafeguardTiming struct {
	FirstDelay time.Duration
	MaxDelay   time.Duration
	SweepEvery time.Duration
}

func (t SafeguardTiming) valid() bool {
	return t.FirstDelay > 0 && t.SweepEvery > 0 && t.MaxDelay >= t.FirstDelay
}

func (t SafeguardTiming) backoff() annonces.ResendBackoff {
	return annonces.ResendBackoff{First: t.FirstDelay, Max: t.MaxDelay}
}

type OverdueReviews interface {
	ResendOverdueReviews(ctx context.Context, backoff annonces.ResendBackoff, limit int) ([]annonces.ResentReview, error)
}

type Safeguard struct {
	reviews OverdueReviews
	timing  SafeguardTiming
	logger  *slog.Logger
}

func NewSafeguard(reviews OverdueReviews, timing SafeguardTiming, logger *slog.Logger) (*Safeguard, error) {
	if !timing.valid() {
		return nil, errInvalidSafeguardTiming
	}
	return &Safeguard{reviews: reviews, timing: timing, logger: logger}, nil
}

func (s *Safeguard) Run(ctx context.Context) error {
	ticker := time.NewTicker(s.timing.SweepEvery)
	defer ticker.Stop()
	for {
		s.sweep(ctx)
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
		}
	}
}

func (s *Safeguard) sweep(ctx context.Context) {
	for ctx.Err() == nil {
		resent, err := s.reviews.ResendOverdueReviews(ctx, s.timing.backoff(), sweepBatch)
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

func (s *Safeguard) logResent(ctx context.Context, resent []annonces.ResentReview) {
	for _, review := range resent {
		s.logger.WarnContext(ctx, "contrôle sans résultat après le délai, demande réémise",
			slog.String("event", "controle.demande_relancee"),
			slog.String("article_id", review.Article.ID.String()),
			slog.Int("revision", int(review.Article.Revision)),
			slog.Int("relances", review.Resends),
			slog.Duration("prochaine_dans", review.NextIn),
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
