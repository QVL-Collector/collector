-- +goose Up
ALTER TABLE article
    ADD COLUMN reservation_id  uuid,
    ADD COLUMN reserve_par     uuid,
    ADD COLUMN reserve_jusqu_a timestamptz,
    ADD COLUMN vendu           boolean NOT NULL DEFAULT false,
    ADD CONSTRAINT article_reservation_complete CHECK (
        (reservation_id IS NULL) = (reserve_par IS NULL) AND (reserve_par IS NULL) = (reserve_jusqu_a IS NULL)
    );

CREATE INDEX article_reserve_par_idx ON article (reserve_par) WHERE reserve_par IS NOT NULL;

CREATE TABLE reservation (
    id                  uuid PRIMARY KEY,
    article_id          uuid NOT NULL REFERENCES article (id),
    acheteur_id         uuid NOT NULL,
    echeance            timestamptz NOT NULL,
    prix_centimes       bigint NOT NULL CONSTRAINT reservation_prix_bornes CHECK (prix_centimes BETWEEN 0 AND 100000000),
    frais_port_centimes bigint NOT NULL CONSTRAINT reservation_frais_port_bornes CHECK (frais_port_centimes BETWEEN 0 AND 100000),
    reserve_le          timestamptz NOT NULL DEFAULT now()
);

REVOKE UPDATE ON reservation FROM marche_execution;
