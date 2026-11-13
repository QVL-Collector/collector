-- +goose Up
ALTER DEFAULT PRIVILEGES IN SCHEMA public GRANT SELECT, INSERT ON TABLES TO interactions_execution;

CREATE TABLE notification (
    id         uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    message_id uuid NOT NULL CONSTRAINT notification_une_par_decision UNIQUE,
    vendeur_id uuid NOT NULL,
    article_id uuid NOT NULL,
    revision   integer NOT NULL CONSTRAINT notification_revision_positive CHECK (revision >= 1),
    issue      text NOT NULL CONSTRAINT notification_issue_connue CHECK (issue IN ('publie', 'refuse', 'a_verifier')),
    motif      text CONSTRAINT notification_motif_connu CHECK (motif IN ('annonce_incomplete', 'coordonnees_detectees',
                                                                        'compte_non_autorise', 'prix_a_confirmer',
                                                                        'vendeur_a_verifier', 'categorie_inconnue',
                                                                        'prix_abusif', 'article_interdit',
                                                                        'vendeur_non_fiable')),
    decide_le  timestamptz NOT NULL,
    cree_le    timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT notification_motif_selon_issue CHECK ((issue = 'publie') = (motif IS NULL))
);

CREATE INDEX notification_vendeur_decision_idx ON notification (vendeur_id, decide_le DESC, id DESC);

REVOKE UPDATE, DELETE, TRUNCATE ON notification FROM interactions_execution;
