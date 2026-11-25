-- +goose Up
CREATE TABLE photo (
    id            uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    article_id    uuid NOT NULL REFERENCES article (id),
    cle_objet     text NOT NULL CONSTRAINT photo_cle_objet_unique UNIQUE
                  CONSTRAINT photo_cle_objet_longueur CHECK (char_length(cle_objet) BETWEEN 1 AND 256),
    largeur       integer NOT NULL CONSTRAINT photo_largeur_positive CHECK (largeur > 0),
    hauteur       integer NOT NULL CONSTRAINT photo_hauteur_positive CHECK (hauteur > 0),
    taille_octets integer NOT NULL CONSTRAINT photo_taille_positive CHECK (taille_octets > 0),
    cree_le       timestamptz NOT NULL DEFAULT now(),
    retiree_le    timestamptz
);

CREATE INDEX photo_article_active_idx ON photo (article_id, cree_le, id) WHERE retiree_le IS NULL;

REVOKE UPDATE ON photo FROM marche_execution;
GRANT UPDATE (retiree_le) ON photo TO marche_execution;
