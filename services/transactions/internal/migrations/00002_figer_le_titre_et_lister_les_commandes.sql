-- +goose Up
ALTER TABLE commande
    ADD COLUMN titre text NOT NULL DEFAULT 'Article sans titre'
        CONSTRAINT commande_titre_longueur CHECK (char_length(titre) BETWEEN 1 AND 200);

ALTER TABLE commande ALTER COLUMN titre DROP DEFAULT;

CREATE INDEX commande_par_acheteur_idx ON commande (acheteur_id, cree_le DESC, id DESC);
