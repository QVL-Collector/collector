-- +goose Up
CREATE TABLE compte (
    id          uuid PRIMARY KEY,
    inscrit_le  timestamptz NOT NULL DEFAULT now(),
    suspendu    boolean NOT NULL DEFAULT false,
    suspendu_le timestamptz,
    CONSTRAINT compte_suspension_datee CHECK (suspendu = (suspendu_le IS NOT NULL))
);

INSERT INTO compte (id, inscrit_le)
SELECT vendeur_id, min(cree_le)
FROM article
GROUP BY vendeur_id;

REVOKE UPDATE ON compte FROM marche_execution;
