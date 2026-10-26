-- +goose Up
CREATE TABLE categorie (
    id      text PRIMARY KEY
            CONSTRAINT categorie_id_format CHECK (char_length(id) <= 64 AND id ~ '^[a-z0-9]+(-[a-z0-9]+)*$'),
    libelle text NOT NULL
            CONSTRAINT categorie_libelle_longueur CHECK (char_length(libelle) BETWEEN 1 AND 100)
);

CREATE TABLE article (
    id                  uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    vendeur_id          uuid NOT NULL,
    etat                text NOT NULL
                        CONSTRAINT article_etat_connu CHECK (etat IN ('brouillon', 'en_controle', 'publie', 'refuse', 'a_verifier')),
    revision            integer NOT NULL CONSTRAINT article_revision_positive CHECK (revision >= 1),
    titre               text NOT NULL CONSTRAINT article_titre_longueur CHECK (char_length(titre) <= 200),
    description         text NOT NULL CONSTRAINT article_description_longueur CHECK (char_length(description) <= 5000),
    categorie_id        text REFERENCES categorie (id),
    prix_centimes       bigint CONSTRAINT article_prix_bornes CHECK (prix_centimes BETWEEN 0 AND 100000000),
    frais_port_centimes bigint CONSTRAINT article_frais_port_bornes CHECK (frais_port_centimes BETWEEN 0 AND 100000),
    cree_le             timestamptz NOT NULL DEFAULT now(),
    modifie_le          timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX article_vendeur_modification_idx ON article (vendeur_id, modifie_le DESC, id DESC);

GRANT SELECT ON categorie TO marche_execution;
GRANT SELECT, INSERT, UPDATE ON article TO marche_execution;
