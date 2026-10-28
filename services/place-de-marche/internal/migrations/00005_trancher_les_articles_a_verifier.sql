-- +goose Up
ALTER TABLE article
    DROP CONSTRAINT article_motif_connu,
    ADD CONSTRAINT article_motif_connu CHECK (motif IN ('annonce_incomplete', 'coordonnees_detectees', 'compte_non_autorise',
                                                        'prix_a_confirmer', 'vendeur_a_verifier', 'categorie_inconnue',
                                                        'prix_abusif', 'article_interdit', 'vendeur_non_fiable'));

CREATE INDEX article_a_verifier_idx ON article (modifie_le, id) WHERE etat = 'a_verifier';

CREATE TABLE decision_admin (
    id         uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    article_id uuid NOT NULL REFERENCES article (id),
    revision   integer NOT NULL CONSTRAINT decision_admin_revision_positive CHECK (revision >= 1),
    decision   text NOT NULL CONSTRAINT decision_admin_decision_connue CHECK (decision IN ('publier', 'refuser')),
    motif      text CONSTRAINT decision_admin_motif_connu CHECK (motif IN ('annonce_incomplete', 'coordonnees_detectees',
                                                                          'prix_abusif', 'article_interdit',
                                                                          'vendeur_non_fiable')),
    admin_id   uuid NOT NULL,
    decide_le  timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT decision_admin_une_decision_par_revision UNIQUE (article_id, revision),
    CONSTRAINT decision_admin_motif_selon_decision CHECK ((decision = 'refuser') = (motif IS NOT NULL))
);

REVOKE UPDATE ON decision_admin FROM marche_execution;
