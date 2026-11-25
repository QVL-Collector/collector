-- +goose Up
ALTER TABLE article
    ADD COLUMN demande_controle_correlation_id uuid,
    ADD COLUMN demande_controle_traceparent    text
        CONSTRAINT article_demande_controle_traceparent_w3c
            CHECK (demande_controle_traceparent ~ '^00-[0-9a-f]{32}-[0-9a-f]{16}-[0-9a-f]{2}$');
