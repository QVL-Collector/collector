# US-020 — Ajouter une photo à un article, en toute sécurité

| Champ | Valeur |
|---|---|
| Epic | E3 Mise en vente contrôlée |
| Statut | À faire |
| Sprint | 2 (J5) |
| Engagement | Ferme (ordre de coupe S2 n° 4 : une seule photo par article) |
| Estimation | 5 points |
| Dépend de | US-013 ; X6 validé par XP01 |
| Fonction(s) | Back, Front, DevOps (bucket, accès S3 par le nœud) |
| Service | Place de marché, Front |
| Critères grille | C3, C8 |
| Exigences | EF-04 (photos), EF-12, EF-17 (photo dans la complétude) |
| Branche | `feat/US-020-photo-securisee` |

## Récit

En tant que vendeur, je veux joindre des photos à mon article, afin que les acheteurs voient l'objet ; et en tant que plateforme, je veux qu'aucun fichier piégé ni aucune donnée de localisation ne soit publié.

## Critères d'acceptation

### Fonctionnels

- **AC1** — Given un brouillon, When le vendeur dépose une photo JPEG ou PNG sous la taille maximale, Then elle est ré-encodée, stockée dans le bucket privé sous un nom généré par le serveur et affichée dans son espace.
- **AC2** — Given un article sans photo, When il est contrôlé, Then il est « refusé » avec le motif « annonce incomplète » (critère photo ajouté à la complétude, règles v2, contrat v1 inchangé).
- **AC3** — Given une photo de test portant des coordonnées GPS fictives, When l'article est publié, Then la photo servie ne contient plus aucune métadonnée de localisation.
- **AC4** — Given un article publié, When un visiteur affiche sa fiche, Then ses photos sont servies par une URL présignée de courte durée.

### Non fonctionnels

- **AC5 (Sécurité)** — Given un fichier dont le contenu n'est pas une image mais dont l'extension est `.jpg`, ou un fichier GIF, WebP ou SVG, When il est déposé, Then il est refusé (415).
- **AC6 (Sécurité)** — Given un fichier au-delà de la taille maximale, When il est déposé, Then la réponse est 413 ; given une image aux dimensions excessives (bombe de décompression), Then elle est refusée sans saturer la mémoire du pod.
- **AC7 (Autorisation)** — Given la photo d'un article non publié du vendeur A, When le vendeur B ou un visiteur demande son URL, Then la réponse est 404.
- **AC8 (Sécurité)** — Given le bucket, When l'inventaire est fait, Then l'accès public est bloqué, le chiffrement actif, les requêtes non TLS refusées ; la politique de sécurité du contenu limite `img-src` au bucket ; aucune URL présignée n'apparaît dans les journaux.
- **AC9 (Accessibilité)** — Given le dépôt de photo, When il est utilisé au clavier, Then il est opérable et chaque photo affichée a un texte alternatif.

## Sécurité

| Exigence | Comment l'US la sert | AC |
|---|---|---|
| ES-17 | Upload sécurisé : contenu vérifié, taille, ré-encodage, géolocalisation retirée | AC1, AC3, AC5, AC6 |
| ES-04 | Photo non publiée invisible pour autrui | AC7 |
| ES-07 | Bucket privé, chiffré | AC8 |

Mesures : MD5 à MD9. Frontière : stockage des photos.

## Tests attendus

| Type | Outil | Ce qui est prouvé | Fonction |
|---|---|---|---|
| Unitaire back | `go test`, fuzzing du décodage | AC5, AC6 | Back |
| Intégration | testcontainers-go | AC1, AC3 | Back |
| Sécurité négative | Playwright API (fichiers piégés, autre vendeur) | AC5 à AC7 | Test |
| Après déploiement | Inventaire du bucket, recherche d'URL présignée dans CloudWatch | AC8 | DevOps |
| E2E et accessibilité | Playwright, axe-core | AC1, AC4, AC9 | Test |

## Hors périmètre

- Antivirus, traitement d'image au-delà du ré-encodage : hors POC.

## Décisions liées

- [D07 — Données](../docs/decisions/D07-donnees.md) (F1, conditionnée à X6) · [D04](../docs/decisions/D04-stack.md) (`CGO_ENABLED=0`)

## Definition of Done

Reprise de [PILOTAGE.md](../docs/PILOTAGE.md) § 5, cochée au fil du circuit.

- [ ] Tous les AC sont couverts par des tests verts en CI (`ci-ok`).
- [ ] Revue de code faite, retours traités.
- [ ] Martin a testé et prononcé « je valide la livraison de la feature ».
- [ ] Documentation touchée à jour.
- [ ] Statut, date de validation et liens des rapports renseignés ci-dessous.
- [ ] Aucune donnée sensible ni credential ajouté au dépôt.

## Suivi

| Étape | Lien / date |
|---|---|
| Rapport de test | |
| Rapport de revue de code | |
| Validation Martin | AAAA-MM-JJ |
| Livraison (PR / merge) | |
