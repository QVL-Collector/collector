-- +goose Up
CREATE INDEX article_catalogue_idx ON article (modifie_le DESC, id DESC) WHERE etat = 'publie';
CREATE INDEX article_catalogue_categorie_idx ON article (categorie_id, modifie_le DESC, id DESC) WHERE etat = 'publie';
