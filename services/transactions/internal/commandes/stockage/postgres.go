package stockage

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"collector/services/transactions/internal/commandes"
	"collector/services/transactions/internal/montants"
)

const orderColumns = `id, article_id, titre, acheteur_id, reservation_id, statut, devise, prix_centimes, frais_port_centimes,
	total_paye_centimes, commission_centimes, montant_vendeur_centimes, echeance, cree_le`

const insertOrderQuery = `
INSERT INTO commande (reservation_id, article_id, titre, acheteur_id, statut, devise, prix_centimes, frais_port_centimes,
                      total_paye_centimes, commission_centimes, montant_vendeur_centimes, echeance)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
ON CONFLICT (reservation_id) DO NOTHING
RETURNING ` + orderColumns

const findByReservationQuery = `SELECT ` + orderColumns + ` FROM commande WHERE reservation_id = $1 AND acheteur_id = $2`

const findOwnedQuery = `SELECT ` + orderColumns + ` FROM commande WHERE id = $1 AND acheteur_id = $2`

const countOwnedQuery = `SELECT count(*) FROM commande WHERE acheteur_id = $1`

const listOwnedQuery = `SELECT ` + orderColumns + ` FROM commande WHERE acheteur_id = $1
ORDER BY cree_le DESC, id DESC
LIMIT $2 OFFSET $3`

var readOnlySnapshot = pgx.TxOptions{IsoLevel: pgx.RepeatableRead, AccessMode: pgx.ReadOnly}

type Store struct {
	pool *pgxpool.Pool
}

func NewStore(pool *pgxpool.Pool) *Store {
	return &Store{pool: pool}
}

func (s *Store) Record(ctx context.Context, placement commandes.Placement) (commandes.Recorded, error) {
	created, err := s.insert(ctx, placement)
	if err == nil {
		return commandes.Recorded{Order: created, Created: true}, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return commandes.Recorded{}, fmt.Errorf("enregistrement de la commande : %w", err)
	}
	existing, err := scanOrder(s.pool.QueryRow(ctx, findByReservationQuery,
		uuid.UUID(placement.Reservation.ID), uuid.UUID(placement.Buyer),
	))
	if err != nil {
		return commandes.Recorded{}, fmt.Errorf("lecture de la commande de la réservation %s : %w", placement.Reservation.ID, err)
	}
	return commandes.Recorded{Order: existing, Created: false}, nil
}

func (s *Store) insert(ctx context.Context, placement commandes.Placement) (commandes.Order, error) {
	amounts := placement.Amounts
	return scanOrder(s.pool.QueryRow(ctx, insertOrderQuery,
		uuid.UUID(placement.Reservation.ID), uuid.UUID(placement.Reservation.Article), placement.Reservation.Title,
		uuid.UUID(placement.Buyer), string(commandes.StatusAwaitingPayment), string(amounts.TotalPaid.Currency),
		amounts.Price.Cents, amounts.ShippingFee.Cents, amounts.TotalPaid.Cents,
		amounts.Commission.Cents, amounts.SellerAmount.Cents, placement.Reservation.ExpiresAt,
	))
}

func (s *Store) FindOwned(ctx context.Context, buyer commandes.BuyerID, id commandes.ID) (commandes.Order, error) {
	order, err := scanOrder(s.pool.QueryRow(ctx, findOwnedQuery, uuid.UUID(id), uuid.UUID(buyer)))
	if errors.Is(err, pgx.ErrNoRows) {
		return commandes.Order{}, commandes.ErrOrderNotFound
	}
	if err != nil {
		return commandes.Order{}, fmt.Errorf("lecture de la commande : %w", err)
	}
	return order, nil
}

func (s *Store) ListOwned(ctx context.Context, buyer commandes.BuyerID, page commandes.PageRequest) (commandes.OrderPage, error) {
	var owned commandes.OrderPage
	err := pgx.BeginTxFunc(ctx, s.pool, readOnlySnapshot, func(tx pgx.Tx) error {
		if err := tx.QueryRow(ctx, countOwnedQuery, uuid.UUID(buyer)).Scan(&owned.Total); err != nil {
			return fmt.Errorf("décompte des commandes : %w", err)
		}
		rows, err := tx.Query(ctx, listOwnedQuery, uuid.UUID(buyer), page.Size, page.Offset())
		if err != nil {
			return fmt.Errorf("liste des commandes : %w", err)
		}
		owned.Orders, err = pgx.CollectRows(rows, func(row pgx.CollectableRow) (commandes.Order, error) {
			return scanOrder(row)
		})
		return err
	})
	if err != nil {
		return commandes.OrderPage{}, fmt.Errorf("lecture des commandes de l'acheteur : %w", err)
	}
	return owned, nil
}

type orderRow struct {
	id                uuid.UUID
	article           uuid.UUID
	title             string
	buyer             uuid.UUID
	reservation       uuid.UUID
	status            string
	currency          string
	priceCents        int64
	shippingFeeCents  int64
	totalPaidCents    int64
	commissionCents   int64
	sellerAmountCents int64
	expiresAt         time.Time
	createdAt         time.Time
}

func (r *orderRow) targets() []any {
	return []any{&r.id, &r.article, &r.title, &r.buyer, &r.reservation, &r.status, &r.currency, &r.priceCents, &r.shippingFeeCents,
		&r.totalPaidCents, &r.commissionCents, &r.sellerAmountCents, &r.expiresAt, &r.createdAt}
}

func scanOrder(row pgx.Row) (commandes.Order, error) {
	var r orderRow
	if err := row.Scan(r.targets()...); err != nil {
		return commandes.Order{}, err
	}
	return r.order(), nil
}

func (r orderRow) order() commandes.Order {
	currency := montants.Currency(r.currency)
	money := func(cents int64) montants.Money {
		return montants.Money{Cents: cents, Currency: currency}
	}
	return commandes.Order{
		ID:          commandes.ID(r.id),
		Article:     commandes.ArticleID(r.article),
		Title:       r.title,
		Buyer:       commandes.BuyerID(r.buyer),
		Reservation: commandes.ReservationID(r.reservation),
		Status:      commandes.Status(r.status),
		Amounts: montants.Breakdown{
			Price:        money(r.priceCents),
			ShippingFee:  money(r.shippingFeeCents),
			TotalPaid:    money(r.totalPaidCents),
			Commission:   money(r.commissionCents),
			SellerAmount: money(r.sellerAmountCents),
		},
		ExpiresAt: r.expiresAt,
		CreatedAt: r.createdAt,
	}
}
