-- +goose Up
ALTER TABLE article
    ADD COLUMN motif text
        CONSTRAINT article_motif_connu CHECK (motif IN ('annonce_incomplete', 'coordonnees_detectees', 'compte_non_autorise',
                                                        'prix_a_confirmer', 'vendeur_a_verifier', 'categorie_inconnue')),
    ADD CONSTRAINT article_motif_selon_etat CHECK ((etat IN ('refuse', 'a_verifier')) = (motif IS NOT NULL));

CREATE TABLE resultat_controle (
    message_id     uuid PRIMARY KEY,
    article_id     uuid NOT NULL REFERENCES article (id),
    revision       integer NOT NULL CONSTRAINT resultat_controle_revision_positive CHECK (revision >= 1),
    issue          text NOT NULL CONSTRAINT resultat_controle_issue_connue CHECK (issue IN ('publie', 'refuse', 'a_verifier')),
    motif          text CONSTRAINT resultat_controle_motif_connu CHECK (motif IN ('annonce_incomplete', 'coordonnees_detectees',
                                                                                  'compte_non_autorise', 'prix_a_confirmer',
                                                                                  'vendeur_a_verifier', 'categorie_inconnue')),
    version_regles integer NOT NULL CONSTRAINT resultat_controle_version_positive CHECK (version_regles >= 1),
    recu_le        timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT resultat_controle_une_issue_par_revision UNIQUE (article_id, revision),
    CONSTRAINT resultat_controle_motif_selon_issue CHECK ((issue = 'publie') = (motif IS NULL))
);

CREATE TABLE outbox (
    id             uuid PRIMARY KEY,
    echange        text NOT NULL CONSTRAINT outbox_echange_present CHECK (char_length(echange) BETWEEN 1 AND 255),
    cle_routage    text NOT NULL CONSTRAINT outbox_cle_routage_bornee CHECK (char_length(cle_routage) <= 255),
    type_message   text NOT NULL CONSTRAINT outbox_type_present CHECK (char_length(type_message) BETWEEN 1 AND 64),
    correlation_id uuid NOT NULL,
    corps          bytea NOT NULL,
    cree_le        timestamptz NOT NULL DEFAULT now(),
    publie_le      timestamptz
);

CREATE INDEX outbox_en_attente_idx ON outbox (cree_le, id) WHERE publie_le IS NULL;
