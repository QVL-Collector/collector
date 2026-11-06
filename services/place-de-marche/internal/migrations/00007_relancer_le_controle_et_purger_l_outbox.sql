-- +goose Up
ALTER TABLE article ADD COLUMN demande_controle_le timestamptz;

UPDATE article SET demande_controle_le = modifie_le WHERE etat = 'en_controle';

ALTER TABLE article
    ADD CONSTRAINT article_demande_controle_datee CHECK (etat <> 'en_controle' OR demande_controle_le IS NOT NULL);

CREATE INDEX article_en_controle_idx ON article (demande_controle_le, id) WHERE etat = 'en_controle';

CREATE INDEX outbox_publiee_idx ON outbox (publie_le) WHERE publie_le IS NOT NULL;

-- +goose StatementBegin
CREATE FUNCTION purger_outbox_publiee() RETURNS bigint
    LANGUAGE sql
    SECURITY DEFINER
    SET search_path = public, pg_temp
AS $$
    WITH purgee AS (
        DELETE FROM outbox
        WHERE publie_le < now() - interval '7 days'
        RETURNING 1
    )
    SELECT count(*) FROM purgee;
$$;
-- +goose StatementEnd

REVOKE ALL ON FUNCTION purger_outbox_publiee() FROM PUBLIC;
GRANT EXECUTE ON FUNCTION purger_outbox_publiee() TO marche_execution;
