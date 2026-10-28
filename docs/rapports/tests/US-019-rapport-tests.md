# US-019 — Rapport de tests

| Champ | Valeur |
|---|---|
| US | [US-019 — Mettre à vérifier les prix incohérents et les vendeurs à risque](../../../backlog/US-019-regles-prix-statut-vendeur.md) |
| Branche | `feat/US-019-regles-prix-statut-vendeur` |
| Date | 2026-10-28 |
| Fonction | Test |
| Périmètre | Service Contrôle (règles v3, paire de contrats), contrat `controle.demande` / `controle.resultat` v1, bout en bout Place de marché ↔ Contrôle (RabbitMQ et PostgreSQL réels sous Docker) |
| Hors périmètre | Acceptation API Playwright `request` (AC1, AC2, AC5), à faire avec la fonction Test |
| Outils | Go 1.26, `go test`, golangci-lint 2.15.0 |
| Verdict | **Tous les AC sont couverts et conformes au niveau du Contrôle** ; AC2 et la partie « module Comptes » d'AC4 restent à clarifier de bout en bout (voir § Points à clarifier) |

## Méthode

Scénarios Given/When/Then dérivés des AC, étiquetés `US019ACn`. Les seuils attendus sont ceux documentés dans `services/controle/README.md` (§ Seuils), recopiés dans les tests et non lus depuis le référentiel : un seuil modifié sans mise à jour de la documentation fait échouer la campagne. Aucun commentaire dans le code de test.

| Fichier | Contenu | Tests |
|---|---|---|
| `services/controle/internal/regles/prix_vendeur_test.go` | Bornes de prix par catégorie, seuil de vigilance, ancienneté autour de 30 jours, suspension, ordre des règles, non-divulgation, pureté | 14 tests, 125 cas |
| `services/controle/internal/canal/paire_test.go` | Paire demande → résultat à travers le contrat v1 (exemples réels, prix hors plage, ancienneté lue dans la demande, données du vendeur) | 5 tests |
| `services/controle/internal/regles/regles_test.go` | Adapté à `New(Catalogue, SeniorityPolicy)` et `Evaluate(Submission)` (vendeur inscrit en 2025, non suspendu) ; version exigée passée à 3 ; `TestUS013AC1TexteMinimalSuffit` à 45 € au lieu d'1 centime pour continuer à prouver la complétude ; référentiel fourni = catalogue d'une seule catégorie | 8 tests adaptés |
| `services/controle/internal/regles/coordonnees_test.go` | Même évaluation par une soumission d'un vendeur en règle | 7 tests adaptés |
| `contracts/messages/controle.resultat/exemples/v1/valides/` | `refuse-compte-non-autorise.json`, `a-verifier-prix-a-confirmer.json`, `a-verifier-vendeur-a-verifier.json` (règles v3) | 3 exemples |
| `contracts/messages/controle.demande/exemples/v1/invalides/schema_non_respecte/nom-vendeur.json` | Demande portant un nom de vendeur, rejetée par le contrat | 1 exemple |
| `services/place-de-marche/cmd/place-de-marche/bout_en_bout_test.go` | Version des règles v3 ; prix du scénario « chaque catégorie » passé de 15 € à 25 € (15 € est hors de la plage `montres`) ; 2 scénarios US-019 ajoutés | 2 sous-tests ajoutés, 2 adaptés |

`vendeur-suspendu.json` reste un exemple valide du contrat (suite `libs/platform/message` verte) ; le Contrôle le refuse `compte_non_autorise`, ce que prouve `TestUS019AC2ExempleVendeurSuspenduConformeEtRefuse`.

## Résultats par AC

| AC | Scénario | Attendu (AC) | Réel | Statut |
|---|---|---|---|---|
| AC1 | 7 catégories : minimum − 1 centime, maximum + 1 centime, prix nul, 1 000 000 € | `a_verifier` / `prix_a_confirmer` | Conforme | ✅ |
| AC1 | 7 catégories : prix exactement au minimum et au maximum (vendeur ancien) | Publié (bornes dans la plage) | Conforme | ✅ |
| AC1 | Vendeur récent, prix sous le minimum | `prix_a_confirmer` | Conforme | ✅ |
| AC1 | Via le contrat : demande à 3 000,01 € en `figurines` | Résultat conforme v1, `prix_a_confirmer`, sans montant ni « plage », « seuil », « min », « max » | Conforme | ✅ |
| AC1 | Bout en bout : figurine à 0,99 € | À vérifier, `prix_a_confirmer` visible du vendeur, résultat sans plage | Conforme | ✅ |
| AC2 | Vendeur suspendu avec annonce irréprochable, vide, incomplète, avec coordonnées, catégorie inconnue, prix hors plage, au-dessus de la vigilance ; vendeur récent et suspendu | `refuse` / `compte_non_autorise` | Conforme dans les 8 cas | ✅ |
| AC2 | Exemple de contrat `vendeur-suspendu.json` traité par le Contrôle | `refuse` / `compte_non_autorise`, résultat conforme v1 | Conforme | ✅ |
| AC3 | 7 catégories, vendeur de 10 jours, vigilance + 1 centime | `a_verifier` / `vendeur_a_verifier` | Conforme | ✅ |
| AC3 | 7 catégories, vendeur de 10 jours, prix au seuil et un centime dessous | Publié (« supérieur au seuil ») | Conforme | ✅ |
| AC3 | 7 catégories, vendeur de 30 jours, vigilance + 1 centime | Publié | Conforme | ✅ |
| AC3 | Inscrit le 3 octobre : soumis le 1er novembre 23:59:59 UTC (29 j), le 2 novembre 00:00 UTC (30 j), le 3 novembre (31 j), le 2 novembre 01:30 +02:00 (29 j en UTC), le jour même, avant l'inscription | 29 j et moins : à vérifier ; 30 j et plus : publié ; jours entiers UTC | Conforme dans les 6 cas | ✅ |
| AC3 | Seuil configuré à 7 jours : 6 j → à vérifier, 7 j → publié | Le seuil injecté fait foi | Conforme | ✅ |
| AC3 | Via le contrat : `member_since` 2026-10-04 / 2026-10-03, `submitted_at` 2026-11-02T00:00Z | `vendeur_a_verifier` / publié, sans seuil dans le résultat | Conforme | ✅ |
| AC3 | Bout en bout : vendeur créé pendant le test, figurine à 300,01 € | À vérifier, `vendeur_a_verifier` | Conforme | ✅ |
| AC4 | Évaluations datées de 2020 et de 2040 | Issue fixée par `submitted_at`, pas par l'horloge | Conforme | ✅ |
| AC4 | Même soumission évaluée 1 000 fois ; catalogue modifié après `New` | Même verdict | Conforme | ✅ |
| AC4 | Demande portant `email`, `display_name` ou `name` du vendeur | Rejetée par le contrat | Rejetée | ✅ |
| AC4 | Bout en bout : demande écrite dans l'outbox | Vendeur limité à `account_id`, `member_since`, `suspended` | Conforme | ✅ |
| AC5 | 7 catégories, prix au milieu de la plage, vendeur ancien, sans coordonnées | Publié | Conforme | ✅ |
| AC5 | Bout en bout : scénarios US-013 de publication (dont chaque catégorie à 25 €) | Publié | Conforme | ✅ |
| Ordre | Incomplète + prix hors plage ; coordonnées + prix hors plage ; coordonnées + vendeur récent au-dessus de la vigilance ; catégorie inconnue + prix extrême ; prix hors plage + vendeur récent au-dessus de la vigilance | Refus avant « à vérifier », puis catégorie → prix → vendeur | `annonce_incomplete`, `coordonnees_detectees`, `coordonnees_detectees`, `categorie_inconnue`, `prix_a_confirmer` | ✅ |
| Non-divulgation | Champs du verdict ; motifs US-019 dans la liste fermée du contrat | Issue et motif seulement | Conforme | ✅ |

## Non-régression

Toutes les suites US-013 et US-018 du Contrôle et le bout en bout Place de marché ↔ Contrôle (18 sous-tests) passent en règles v3.

## Exécution

| Commande | Résultat |
|---|---|
| `go test ./... -count=1 -cover` dans `services/controle` | ✅ `regles` 100,0 %, `coordonnees` 98,5 %, `canal` 45,5 % |
| `go test ./... -count=1 -cover` dans `services/place-de-marche` (Docker) | ✅ `cmd/place-de-marche` 50,0 %, bout en bout 18/18 |
| `go test ./message -count=1` dans `libs/platform` | ✅ exemples valides acceptés, invalides rejetés |
| `golangci-lint run --config ../../.golangci.yml ./...` (2.15.0) dans les deux services | ✅ 0 issue |
| `go test -race` | Non exécuté localement (cgo indisponible) ; les tests ajoutés ne sont pas chronométrés |

Conteneurs de test supprimés en fin de campagne.

## Points à clarifier

1. **AC2 de bout en bout** : aucun moyen exposé par la Place de marché (API ou jeu amorcé) ne permet de suspendre un vendeur, le scénario n'a donc pas pu être joué de bout en bout. Le refus `compte_non_autorise` est prouvé au niveau du Contrôle et du contrat, pas depuis le « module Comptes minimal » cité par l'AC. À trancher : la suspension côté Place de marché fait-elle partie de cette US ?
2. **AC4 « interface interne du module Comptes »** : non vérifiable en boîte noire. Observé : le vendeur créé pendant le test est traité comme récent, l'ancienneté transmise est donc bien calculée côté Place de marché.

Aucune anomalie : aucun comportement observé ne contredit un AC.
