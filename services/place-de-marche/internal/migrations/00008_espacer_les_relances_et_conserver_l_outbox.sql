-- +goose Up
ALTER TABLE article
    ADD COLUMN relances integer NOT NULL DEFAULT 0 CONSTRAINT article_relances_positives CHECK (relances >= 0);

-- +goose StatementBegin
CREATE FUNCTION conservation_outbox() RETURNS interval
    LANGUAGE sql
    IMMUTABLE
AS $$
    SELECT interval '7 days';
$$;
-- +goose StatementEnd

-- +goose StatementBegin
CREATE OR REPLACE FUNCTION purger_outbox_publiee() RETURNS bigint
    LANGUAGE sql
    SECURITY DEFINER
    SET search_path = public, pg_temp
AS $$
    WITH purgee AS (
        DELETE FROM outbox
        WHERE publie_le < now() - conservation_outbox()
        RETURNING 1
    )
    SELECT count(*) FROM purgee;
$$;
-- +goose StatementEnd
