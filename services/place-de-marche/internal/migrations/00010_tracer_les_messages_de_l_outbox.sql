-- +goose Up
ALTER TABLE outbox
    ADD COLUMN traceparent text
        CONSTRAINT outbox_traceparent_w3c CHECK (traceparent ~ '^00-[0-9a-f]{32}-[0-9a-f]{16}-[0-9a-f]{2}$');
