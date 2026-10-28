# US-023 — Rapport de tests

| Champ | Valeur |
|---|---|
| US | [US-023 — Filtrer le catalogue par catégorie et consulter la fiche d'un article](../../../backlog/US-023-catalogue-filtre-fiche.md) |
| Branche | `feat/US-023-catalogue-filtre-fiche` |
| Date | 2026-10-28 |

## Back — API publique du catalogue

| Champ | Valeur |
|---|---|
| Fonction | Test |
| Périmètre | `services/place-de-marche` : US-023 AC1 (filtre), AC3 (404), partie API d'AC2 (contenu de la fiche) ; US-010 AC2 (seuls les publiés, sans compte, aucune donnée du vendeur) |
| Hors périmètre | Photos et mention « vendeur particulier » d'AC2 (affichage front, E2E), AC4 (charge k6), AC5 (axe-core), front `web/` |
| Outils | `go test`, `httptest`, kin-openapi 0.152.0 (validation des réponses contre `contracts/api/place-de-marche/v1.openapi.yaml`), testcontainers-go 0.44.0 avec PostgreSQL 18.3 |
| Verdict | **Tous les AC du périmètre sont couverts et conformes** |

### Méthode

Scénarios dérivés des AC, étiquetés `US-023-ACn` ou `US-010-AC2`, joués sans jeton sur `GET /marche/v1/articles` et `GET /marche/v1/articles/{id}`. Les articles sont amenés dans chaque état par le parcours réel : création et soumission par l'API du vendeur, puis issue de contrôle (`publie`, `refuse`, `a_verifier`) livrée au consommateur de résultats sous forme de message `controle.resultat` v1 signé et validé. Les catégories utilisées sont lues dans `GET /categories` ; les assertions sont exprimées en écart par rapport à l'état initial, ce qui permet de jouer le même jeu sur une base partagée.

| Fichier | Contenu |
|---|---|
| `cmd/place-de-marche/catalogue_test.go` | 14 scénarios ; banc en mémoire (un banc neuf par scénario) et validation de toutes les réponses du catalogue contre le contrat OpenAPI |
| `cmd/place-de-marche/catalogue_integration_test.go` | Mêmes scénarios sur PostgreSQL réel, rôle `marche_execution`, contrat, index partiels, rejeu des migrations, droits du rôle |
| `cmd/place-de-marche/memoire_test.go` | `ListPublished` et `FindPublished` ajoutés au stockage en mémoire (les tests ne compilaient plus) |
| `cmd/place-de-marche/canal_fictif_test.go`, `integration_test.go` | Extraction de `monterBanc` et `servicePostgreSQL`, sans changement des assertions existantes |

### Résultats d'exécution (depuis `services/place-de-marche`)

| Commande | Résultat |
|---|---|
| `go test ./... -count=1 -cover` | `ok` (≈ 72 s), tous les tests verts, dont US-009, US-013 et l'intégration RabbitMQ existants |
| `go test ./cmd/... -count=1 -coverpkg=./...` | Couverture du service 83,9 % ; `internal/annonces/stockage/catalogue.go` : `ListPublished` 81,2 %, `FindPublished` 87,5 % ; `api/catalogue.go` 85,7 à 100 % |
| `golangci-lint run --config ../../.golangci.yml ./...` (v2.15.0) | 0 issue |
| Conteneurs | Arrêtés et supprimés par testcontainers en fin de test ; aucun conteneur PostgreSQL ou RabbitMQ résiduel |

### US-023 AC1 — Filtre par catégorie

| Scénario | Attendu (AC) | Réel (mémoire et PostgreSQL) | Statut |
|---|---|---|---|
| Nominal : 2 publiés en catégorie A, 1 en B, un article de A dans chaque état non publié | Seuls les publiés de A listés | Filtre A : +2 exactement, tous en catégorie A, ni l'article de B ni les non publiés ; filtre B : +1, sans les articles de A | Conforme |
| Catégorie existante sans article publié (seulement des non publiés) | Liste vide | 200, `total` 0, `elements` vide, page 1 | Conforme |
| Tri : deux articles publiés dans l'ordre inverse de leur création, puis un troisième | Du plus récemment publié au plus ancien | Le dernier publié en tête ; l'ordre suit la publication et non la création, avec et sans filtre | Conforme |
| Pagination : taille 1 sur tout le filtre | Pages cohérentes avec le total | Chaque page : 1 élément, `page` et `taille` renvoyés, `total` constant ; concaténation identique à la liste complète, sans doublon | Conforme |
| Page `total + 1` et page 10000 | Page vide | 200, `elements` vide, `total` inchangé | Conforme |
| Sans paramètre | Page 1, taille 20 | `page` 1, `taille` 20, au plus 20 éléments | Conforme |
| Bornes : `page=1&taille=1`, `page=10000`, `taille=50`, filtre + `taille=50` | Acceptées | 200 | Conforme |
| `categorie=inconnue` | 400 `categorie_inconnue` | 400, `erreurs[{champ: categorie, code: categorie_inconnue}]` | Conforme |
| `categorie=` vide, `categorie` répétée (deux valeurs ou la même deux fois) | 400 `invalide` | 400 `invalide` | Conforme |
| `page=0`, `-1`, `10001`, `taille=0`, `51` | 400 `hors_limites` | 400 `hors_limites` | Conforme |
| `page=`, `taille=`, `page=abc`, `taille=1.5`, `page` ou `taille` répétés | 400 `invalide` | 400 `invalide` | Conforme |
| `categorie=FIGURINES` (hors format) | 400 | 400 | Conforme |

### US-023 AC2 — Fiche (partie API)

| Scénario | Attendu (AC) | Réel | Statut |
|---|---|---|---|
| Article publié avec frais de port | Titre, description, prix, frais de port, catégorie | Fiche identique au contenu saisi (id, titre, description multiligne, prix, frais, catégorie id et libellé) | Conforme |
| Article publié sans frais de port | Fiche lisible | `fraisDePort: null` | Conforme |

Photos et mention « vendeur particulier » : aucune donnée correspondante dans le contrat de l'API ; leur affichage relève du front et de l'E2E.

### US-023 AC3 — Article non publié introuvable

| Scénario | Attendu (AC) | Réel | Statut |
|---|---|---|---|
| Fiche d'un article brouillon, en contrôle, refusé, à vérifier | 404 | 404 `application/problem+json`, `type`, `title`, `detail` identiques à un article inexistant | Conforme |
| Identifiant inexistant, mal formé, presque UUID | 404 identique | 404 identique | Conforme |
| Témoin : article publié | 200 | 200 | Conforme |

### US-010 AC2 — Seuls les articles publiés, sans compte, aucune donnée du vendeur

| Scénario | Attendu (AC) | Réel | Statut |
|---|---|---|---|
| Un article dans chaque état non publié | Aucun n'apparaît | Absents de la liste complète et du filtre ; `total` inchangé | Conforme |
| Lecture directe de chacun | 404 | 404 | Conforme |
| Article refusé puis publié | Absent tant que refusé, visible une fois publié | 404 et absent, puis 200 et présent | Conforme |
| Sans compte | Lecture sans jeton | Toutes les requêtes du catalogue sans en-tête `Authorization` : 200 | Conforme |
| Données du vendeur | Aucune | Ni l'identifiant du vendeur, ni `vendeur`, `seller`, `etat`, `revision`, `motif` dans la liste ou la fiche ; clés limitées à `id, titre, prix, categorie` (entrée) et `+ description, fraisDePort` (fiche) | Conforme |

### Contrat, base de données et migrations

| Scénario | Attendu | Réel | Statut |
|---|---|---|---|
| Toutes les réponses du catalogue (200, 400, 404), mémoire et PostgreSQL | Conformes au contrat OpenAPI v1 | Aucune non-conformité relevée par kin-openapi | Conforme |
| Lectures avec le rôle `marche_execution` | Suffisant pour le catalogue | Tous les scénarios verts sur une connexion en `marche_execution` | Conforme |
| Droits du rôle d'exécution | DML limité à ses tables | DDL, `DELETE`, `TRUNCATE` refusés (42501) ; aucun objet possédé | Conforme |
| Migrations appliquées deux fois au démarrage, puis rejouées sur une base peuplée | Sans erreur ni effet | Aucune erreur ; catalogue identique avant et après le rejeu | Conforme |
| Migration 00004 | Index partiels sur les publiés | `article_catalogue_idx (modifie_le DESC, id DESC)` et `article_catalogue_categorie_idx (categorie_id, modifie_le DESC, id DESC)`, tous deux `WHERE etat = 'publie'` | Conforme |

### Anomalies

Aucune anomalie : aucun comportement observé ne contredit un AC du périmètre Back.

### Observations (sans anomalie)

- Les catégories amorcées en base (migration 00002 : `bandes-dessinees`, `cartes-a-collectionner`, `figurines`…) diffèrent du jeu fictif du banc en mémoire (`cartes-postales`, `figurines`, `timbres`) ; les scénarios lisent donc les catégories par `GET /categories`.
- Le tri « du plus récemment publié » s'appuie sur la date de dernière modification (`modifie_le`) ; c'est équivalent tant qu'un article publié n'est plus modifiable, ce qu'impose US-013.

## Front — catalogue filtré, fiche et page introuvable

| Champ | Valeur |
|---|---|
| Fonction | Test |
| Périmètre | `web/` : US-023 AC1 (filtre), AC2 (fiche), AC3 (page « Article introuvable »), AC5 (axe-core, clavier) |
| Hors périmètre | AC4 (charge k6) ; photos d'AC2 (aucun champ photo dans le contrat v1, dépend d'US-020) |
| Outils | Vitest 4 (jsdom), Testing Library, user-event, axe-core ; API simulée au niveau HTTP (`fetch`) selon `contracts/api/place-de-marche/v1.openapi.yaml`, client réel (`source: api`) |
| Verdict | **Tous les AC du périmètre Front sont couverts et conformes** |

### Méthode

Scénarios dérivés des AC, étiquetés `US-023-ACn`, joués sur l'application complète. Le faux serveur (`publicMarketplaceHarness.tsx`) répond à `GET /marche/v1/categories`, `GET /marche/v1/articles` (filtre `categorie`, pagination, 400 `categorie_inconnue`) et `GET /marche/v1/articles/{id}` (404 pour tout article non publié), en `application/problem+json` pour les erreurs.

| Fichier (`web/src/acceptance/`) | Contenu |
|---|---|
| `US-023/publicMarketplaceHarness.tsx` | Faux serveur HTTP de la Place de marché, jeu d'articles, analyse axe-core |
| `US-023/AC1-filtre-par-categorie.test.tsx` | 12 scénarios |
| `US-023/AC2-fiche-article.test.tsx` | 8 scénarios |
| `US-023/AC3-article-introuvable.test.tsx` | 9 scénarios |
| `US-023/AC5-accessibilite.test.tsx` | 13 scénarios |
| `US-010/catalogueHarness.tsx` | `listCategories` (liste vide) et `getPublishedArticle` (rejet, hors périmètre) ajoutés au faux client : le typecheck échouait ; assertions US-010 inchangées |

### Résultats d'exécution (depuis `web/`)

| Commande | Résultat |
|---|---|
| `npx vitest run --pool=forks --poolOptions.forks.maxForks=2` | 28 fichiers, 303 tests verts (261 existants + 42 US-023) |
| `npm run typecheck` | OK |
| `npm run lint` | 0 erreur, 0 avertissement |
| `npm run i18n:check` | 175 clés alignées FR/EN |
| `npm run build` | OK |
| `npm run format` | Appliqué |

### US-023 AC1 — Filtre par catégorie

| Scénario | Attendu (AC) | Réel | Statut |
|---|---|---|---|
| Nominal : 14 figurines et 3 timbres publiés, 4 timbres non publiés ; choix « Timbres » | Seuls les timbres publiés listés | 3 cartes, toutes « Timbres », aucun non publié ni figurine | Conforme |
| Contenu du filtre | « Toutes les catégories » puis catégories de l'API | Options dans l'ordre de l'API, valeur initiale vide | Conforme |
| Requête émise | `categorie` et page 1 | `GET /marche/v1/articles?categorie=timbres&page=1…` | Conforme |
| Choix depuis la page 2 | `?categorie=` dans l'URL, retour page 1 | `categorie=figurines`, page 1, annonce « 14 articles publiés dans la catégorie Figurines, page 1 sur 2 » | Conforme |
| Pagination sous filtre | Filtre conservé | Page 2 : `categorie` conservé, 2 cartes | Conforme |
| Retour arrière après deux choix | Filtre précédent restauré | URL, valeur du filtre et liste reviennent à « Figurines » | Conforme |
| Adresse directe `?categorie=timbres` | Filtre présélectionné | Valeur « timbres », 3 cartes | Conforme |
| Choix « Toutes les catégories » | Paramètre retiré, tous les publiés | `categorie` absent, « 17 articles publiés, page 1 sur 2 » | Conforme |
| Catégorie existante sans publié | État « catégorie vide » + action | Titre « Aucun article publié dans cette catégorie », pas de liste ; « Voir toutes les catégories » rétablit 12 cartes | Conforme |
| Catégorie inconnue (400 `categorie_inconnue`) | État dédié, pas l'état d'erreur | « Cette catégorie n'existe pas », action de retour fonctionnelle | Conforme |
| Identifiant hors motif (`<script>`) | Ignoré | Jamais transmis à l'API, catalogue affiché | Conforme |
| Cartes filtrées | Mènent à la fiche | `href="/articles/{id}"` | Conforme |

### US-023 AC2 — Fiche

| Scénario | Attendu (AC) | Réel | Statut |
|---|---|---|---|
| Article publié avec frais | Titre, description, prix, frais, catégorie, « vendeur particulier » | Tous présents (150,00 €, 6,90 €, Figurines, « Vendeur particulier ») | Conforme |
| Appel API | `GET /v1/articles/{id}` public | Chemin `/marche/v1/articles/{id}` appelé | Conforme |
| Description multiligne | Texte lisible | Sauts de ligne conservés (`white-space: pre…`) | Conforme |
| `fraisDePort: null` | Fiche lisible | « Frais de port … Non renseignés par le vendeur » | Conforme |
| Titre et description piégés (`<b>`, `<script>`, `<img onerror>`) | Affichés comme texte (ES-08) | Texte littéral, aucun `script`, `img[onerror]` ni `b` créé, aucun code exécuté | Conforme |
| Lien de catégorie | Catalogue filtré | `/?categorie=figurines`, 1 carte | Conforme |
| Retour au catalogue depuis une fiche ouverte en page 2 filtrée | Filtre et page restaurés | `categorie=figurines&page=2` | Conforme |
| Ouverture depuis une carte | Titre de page, focus `h1` | « Figurine Astérix 1985 · Collector », focus sur l'unique `h1` | Conforme |
| Photos | Visibles | Non testable : absent du contrat v1 (US-020) | Clarification |

### US-023 AC3 — Article introuvable

| Scénario | Attendu (AC) | Réel | Statut |
|---|---|---|---|
| Article brouillon, en contrôle, refusé, à vérifier (404 API) | Page « article introuvable » | `h1` unique « Article introuvable », titre « Article introuvable · Collector », « Retour au catalogue », aucune donnée de l'article | Conforme |
| Identifiant inexistant, identifiant mal formé | Idem | Idem | Conforme |
| Article retiré entre la liste et la fiche | Idem, focus `h1` | Page introuvable, focus sur le `h1` | Conforme |
| « Retour au catalogue » | Catalogue | `/`, `h1` « Catalogue » | Conforme |
| Anglais | Page traduite | « Item not found » | Conforme |

### US-023 AC5 — Accessibilité

| Scénario | Attendu (AC) | Réel | Statut |
|---|---|---|---|
| axe-core : catalogue filtré FR/EN, catégorie vide FR, catégorie inconnue EN, fiche FR/EN (frais non renseignés, description piégée), article introuvable FR/EN | Aucune violation serious ou critical | Aucune ; `lang` suivi | Conforme |
| Filtre au clavier | Atteignable par Tab, utilisable | Atteint par Tab, choix appliqué, focus conservé sur le filtre (pas sur le `h1`) | Conforme |
| Annonce du résultat | Zone `role="status"` | Une seule zone, « 14 articles publiés dans la catégorie Figurines, page 1 sur 2 » | Conforme |
| Nature du filtre | Liste native libellée | `select` « Catégorie » (« Category » en anglais), focalisable | Conforme |
| Fiche au clavier | Liens atteignables | Lien de catégorie puis « Retour au catalogue » atteints par Tab, Entrée ramène au catalogue | Conforme |

### Anomalies

Aucune anomalie : aucun comportement observé ne contredit un AC du périmètre Front.

### Observations (sans anomalie)

- Photos d'AC2 : ni champ dans le contrat v1 ni affichage ; à couvrir à la livraison d'US-020 (clarification à trancher au niveau de l'US).
- Sur une ouverture directe de l'adresse (sans navigation interne), le focus ne passe pas sur le `h1` ; les AC ne l'exigent pas, le focus est bien déplacé lors d'une navigation dans l'application.
- Contraste non mesurable en jsdom (`color-contrast` désactivé, comme pour US-010 et US-025) : couvert par `AC4-contrastes-wcag.test.ts` d'US-025.
