-- +goose Up
INSERT INTO categorie (id, libelle) VALUES
    ('bandes-dessinees', 'Bandes dessinées'),
    ('cartes-a-collectionner', 'Cartes à collectionner'),
    ('figurines', 'Figurines'),
    ('jouets-anciens', 'Jouets anciens'),
    ('monnaies', 'Monnaies'),
    ('montres', 'Montres'),
    ('vinyles', 'Vinyles')
ON CONFLICT (id) DO NOTHING;
