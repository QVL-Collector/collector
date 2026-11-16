-- +goose Up
ALTER DEFAULT PRIVILEGES IN SCHEMA public GRANT SELECT, INSERT ON TABLES TO transactions_execution;

CREATE TABLE commande (
    id                       uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    reservation_id           uuid NOT NULL CONSTRAINT commande_une_par_reservation UNIQUE,
    article_id               uuid NOT NULL,
    acheteur_id              uuid NOT NULL,
    statut                   text NOT NULL CONSTRAINT commande_statut_connu CHECK (statut IN ('en_attente_paiement')),
    devise                   text NOT NULL CONSTRAINT commande_devise_connue CHECK (devise = 'EUR'),
    prix_centimes            bigint NOT NULL CONSTRAINT commande_prix_positif CHECK (prix_centimes >= 0),
    frais_port_centimes      bigint NOT NULL CONSTRAINT commande_frais_port_positifs CHECK (frais_port_centimes >= 0),
    total_paye_centimes      bigint NOT NULL,
    commission_centimes      bigint NOT NULL CONSTRAINT commande_commission_positive CHECK (commission_centimes >= 0),
    montant_vendeur_centimes bigint NOT NULL CONSTRAINT commande_montant_vendeur_positif CHECK (montant_vendeur_centimes >= 0),
    echeance                 timestamptz NOT NULL,
    cree_le                  timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT commande_total_paye CHECK (total_paye_centimes = prix_centimes + frais_port_centimes),
    CONSTRAINT commande_repartition CHECK (commission_centimes + montant_vendeur_centimes = total_paye_centimes)
);
