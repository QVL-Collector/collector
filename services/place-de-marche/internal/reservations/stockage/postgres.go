package stockage

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"collector/services/place-de-marche/internal/annonces"
	"collector/services/place-de-marche/internal/disponibilite"
	"collector/services/place-de-marche/internal/reservations"
)

const lockBuyerQuery = `SELECT pg_advisory_xact_lock(hashtextextended($1, 0))`

const activeElsewhereQuery = `
SELECT EXISTS (
	SELECT 1 FROM article
	WHERE reserve_par = $1 AND ` + disponibilite.ReservedNow + ` AND id <> $2
)`

const heldByBuyer = `reserve_par = $2 AND ` + disponibilite.ReservedNow

const reserveQuery = `
UPDATE article
SET reservation_id = CASE WHEN ` + heldByBuyer + ` THEN reservation_id ELSE gen_random_uuid() END,
    reserve_jusqu_a = CASE WHEN ` + heldByBuyer + ` THEN reserve_jusqu_a ELSE now() + make_interval(secs => $3) END,
    reserve_par = $2
WHERE id = $1 AND etat = 'publie' AND NOT vendu AND prix_centimes IS NOT NULL AND vendeur_id <> $2
  AND (reserve_jusqu_a IS NULL OR reserve_jusqu_a <= now() OR reserve_par = $2)
RETURNING reservation_id, titre, reserve_jusqu_a, prix_centimes, coalesce(frais_port_centimes, 0)`

const recordReservationQuery = `
INSERT INTO reservation (id, article_id, acheteur_id, echeance, prix_centimes, frais_port_centimes)
VALUES ($1, $2, $3, $4, $5, $6)
ON CONFLICT (id) DO NOTHING`

const reservableQuery = `
SELECT EXISTS (
	SELECT 1 FROM article
	WHERE id = $1 AND etat = 'publie' AND NOT vendu AND prix_centimes IS NOT NULL AND vendeur_id <> $2
)`

type Store struct {
	pool *pgxpool.Pool
}

func NewStore(pool *pgxpool.Pool) *Store {
	return &Store{pool: pool}
}

func (s *Store) Reserve(ctx context.Context, request reservations.Request) (reservations.Outcome, error) {
	var outcome reservations.Outcome
	err := pgx.BeginFunc(ctx, s.pool, func(tx pgx.Tx) error {
		var err error
		outcome, err = reserveInTransaction(ctx, tx, request)
		return err
	})
	if err != nil {
		return reservations.Outcome{}, fmt.Errorf("réservation de l'article : %w", err)
	}
	return outcome, nil
}

func reserveInTransaction(ctx context.Context, tx pgx.Tx, request reservations.Request) (reservations.Outcome, error) {
	if _, err := tx.Exec(ctx, lockBuyerQuery, request.Buyer.String()); err != nil {
		return reservations.Outcome{}, fmt.Errorf("verrou de l'acheteur : %w", err)
	}
	if err := refuseIfActiveElsewhere(ctx, tx, request); err != nil {
		return reservations.Outcome{}, err
	}
	reservation, err := holdArticle(ctx, tx, request)
	if errors.Is(err, pgx.ErrNoRows) {
		return reservations.Outcome{}, explainUnavailable(ctx, tx, request)
	}
	if err != nil {
		return reservations.Outcome{}, err
	}
	created, err := recordReservation(ctx, tx, reservation)
	if err != nil {
		return reservations.Outcome{}, err
	}
	return reservations.Outcome{Reservation: reservation, Created: created}, nil
}

func refuseIfActiveElsewhere(ctx context.Context, tx pgx.Tx, request reservations.Request) error {
	var active bool
	if err := tx.QueryRow(ctx, activeElsewhereQuery, uuid.UUID(request.Buyer), uuid.UUID(request.Article)).Scan(&active); err != nil {
		return fmt.Errorf("recherche d'une réservation active : %w", err)
	}
	if active {
		return reservations.ErrBuyerAlreadyHoldsAnother
	}
	return nil
}

func holdArticle(ctx context.Context, tx pgx.Tx, request reservations.Request) (reservations.Reservation, error) {
	var (
		id               uuid.UUID
		title            string
		expiresAt        time.Time
		priceCents       int64
		shippingFeeCents int64
	)
	err := tx.QueryRow(ctx, reserveQuery,
		uuid.UUID(request.Article), uuid.UUID(request.Buyer), request.Duration.Seconds(),
	).Scan(&id, &title, &expiresAt, &priceCents, &shippingFeeCents)
	if err != nil {
		return reservations.Reservation{}, err
	}
	return reservations.Reservation{
		ID:          reservations.ID(id),
		Article:     request.Article,
		Title:       title,
		Buyer:       request.Buyer,
		ExpiresAt:   expiresAt,
		Price:       euros(priceCents),
		ShippingFee: euros(shippingFeeCents),
	}, nil
}

func recordReservation(ctx context.Context, tx pgx.Tx, reservation reservations.Reservation) (bool, error) {
	tag, err := tx.Exec(ctx, recordReservationQuery,
		uuid.UUID(reservation.ID), uuid.UUID(reservation.Article), uuid.UUID(reservation.Buyer),
		reservation.ExpiresAt, reservation.Price.Cents, reservation.ShippingFee.Cents,
	)
	if err != nil {
		return false, fmt.Errorf("journal de la réservation : %w", err)
	}
	return tag.RowsAffected() == 1, nil
}

func explainUnavailable(ctx context.Context, tx pgx.Tx, request reservations.Request) error {
	var reservable bool
	if err := tx.QueryRow(ctx, reservableQuery, uuid.UUID(request.Article), uuid.UUID(request.Buyer)).Scan(&reservable); err != nil {
		return fmt.Errorf("lecture de l'état de l'article : %w", err)
	}
	if !reservable {
		return reservations.ErrArticleNotReservable
	}
	return reservations.ErrArticleAlreadyReserved
}

func euros(cents int64) annonces.Money {
	return annonces.Money{Cents: cents, Currency: annonces.Euro}
}
