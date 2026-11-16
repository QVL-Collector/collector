# Rapport de tests — US-031 Garantir qu'un objet unique n'est réservé qu'une fois

## Back

| Champ | Valeur |
|---|---|
| US | [US-031](../../../backlog/US-031-garanties-reservation.md) |
| Branche | `feat/US-031-garanties-reservation` (non commitée) |
| Date | 2026-11-16 |
| Fonction | Test |
| Périmètre | `services/place-de-marche` (« réserver », catalogue et fiche, routes du vendeur `PUT /vendeur/articles/{id}` et `POST …/{id}/soumission`, `RESERVATION_DUREE`), `services/transactions` (`POST` et `GET /commandes`, statut `expiree` dérivé à la lecture) : AC1 à AC5 ; décision de pilotage du 2026-11-16 (pas de route « retirer » : le retrait passe par la modification d'US-021, donc AC5 = modification **et** soumission d'un article réservé ou vendu refusées en 409 `urn:collector:probleme:article-reserve`, contenu, prix, frais et révision inchangés) ; contrats `transactions/v1` (`statut` `expiree`), `place-de-marche/v1` (`article-reserve`), `place-de-marche/interne-v1` ; README « Garanties de la réservation (US-031) » et « Expiration (US-031) » |
| Hors périmètre | Compensation et remboursement d'une confirmation tardive (V1, hors US). Paiement : US suivantes |
| Outils | `go test`, testcontainers-go avec PostgreSQL, RabbitMQ et **vrai Keycloak** (banc d'US-030 réutilisé, `realm-collector.json`), Place de marché (`serve`) en processus de test avec **`RESERVATION_DUREE` = `RESERVATION_DUREE_MAX` = 6 s**, Contrôle et Transactions construits et lancés en processus, relais espion, kin-openapi ; tests unitaires de Transactions avec horloge injectable (`commandes.Clock`) |
| Verdict | **Tous les AC du périmètre back sont couverts et conformes** ; aucune anomalie |

### Méthode

Scénarios dérivés des AC et de la décision de pilotage, étiquetés `US031…`. Trois niveaux :

- **Concurrence et bout en bout** (`TestUS031GarantiesDeLaReservation`, un `t.Run` par scénario sur un banc dédié) : acheteurs et vendeurs créés dans Keycloak, jetons obtenus par le vrai flux du navigateur (code d'autorisation + PKCE). Articles créés, soumis par les vraies routes du vendeur et publiés par le Contrôle. Les requêtes concurrentes sont préparées à l'avance, lancées par des goroutines retenues sur un même canal puis libérées ensemble (départ synchronisé, aucun `t.Fatal` hors de la goroutine du test). Chaque tour est vérifié dans les réponses **et** en base (`reservation`, `article`, `commande`, relues en superutilisateur). L'échéance courte (6 s) permet de franchir réellement l'échéance dans le test (AC3 limite, AC4, AC5 après l'échéance). Pour les scénarios qui exigent une réservation **encore active** (AC1, AC2, AC3, AC4 avant l'échéance, AC5 article réservé), une garde explicite (`exigerAvantLEcheance`) relève l'heure juste après chaque réponse attendue et relecture, et fait échouer le scénario avec un message clair (« échéance dépassée avant la fin de … : le scénario n'est pas probant ») si l'échéance est passée entre-temps, au lieu de laisser une assertion échouer pour une mauvaise raison (retour de revue).
- **Unitaire** (`services/transactions/internal/commandes/expiration_test.go`) : `Order.AsOf` aux bornes (une nanoseconde avant, à l'échéance exacte, une nanoseconde après, autre fuseau) ; `Service` avec une horloge réglable : lecture avant et à l'échéance, nouvel essai après l'échéance.
- **Configuration** (`internal/config/reservation_test.go`) : test de table direct sur `config.LoadServer(lookup)`, instantané ; un seul cas par borne gardé de bout en bout dans `reservation_duree_config_test.go` (`serve` lancé avec une configuration complète dont seule la durée varie).

### Livrables de test

| Fichier | Contenu |
|---|---|
| `services/place-de-marche/cmd/place-de-marche/reservation_garanties_test.go` | `TestUS031GarantiesDeLaReservation` : 13 scénarios (AC1 ×2, AC2, AC3 ×3, AC4 ×2, AC5 ×4, contrats) ; lancement parallèle synchronisé (`enParallele`, `enParalleleDecale`), relectures en base |
| `services/place-de-marche/cmd/place-de-marche/reservation_duree_config_test.go` | De bout en bout : `TestUS031DureeDeReservationHorsBornesRefuseeAuDemarrage` (`999ms` pour chacune des deux variables) et `TestUS031DureeDeReservationDUneSecondeNonRefuseeAuDemarrage` (`1s`) |
| `services/place-de-marche/internal/config/reservation_test.go` | `TestUS031DureesDeReservationAcceptees` (5 cas, dont `1s`, `1.5s`, valeurs par défaut, plafond appliqué : durée, plafond et durée effective lus) et `TestUS031DureesDeReservationRefusees` (9 cas) sur `config.LoadServer` |
| `services/place-de-marche/cmd/place-de-marche/reservation_banc_test.go` | Banc d'US-030 paramétré : `demarrerBancReservationAvec(t, duree, plafond)` ; `demarrerBancReservation` garde 45 min / 20 min pour US-030 |
| `services/transactions/internal/commandes/expiration_test.go` | 6 tests : statut selon l'échéance (6 bornes), commande expirée jamais réactivée, valeurs du contrat, lecture expirée à l'échéance, nouvel essai après l'échéance, relance avant l'échéance ; la première commande de chaque scénario est vérifiée (`err`, `Created`, statut) par `premiereCommande` (retour de revue) |

### Résultats d'exécution

| Commande | Résultat | Durée |
|---|---|---|
| `go test ./cmd/place-de-marche/ -run TestUS031GarantiesDeLaReservation -count=1 -v` (première passe) | 11 scénarios PASS, 2 FAIL : **erreur de test** (comparaison de pointeurs `*int64` sur les frais de port ; la comparaison de la ligne complète, elle, était identique). Corrigée sans toucher au code de production | 191 s |
| Même commande après correction, courses vendeur/acheteurs décalées | `ok`, **13 scénarios PASS, 0 skip** | 207 s |
| `go test ./cmd/place-de-marche/ -run TestUS031 -count=1 -v` après découpage pour `gocognit` (code final) | `ok`, 3 tests de premier niveau, 30 sous-tests PASS, 0 skip ; courses : 11/1 par Transactions, 10/2 en direct | 209 s |
| `go test ./... -count=1 -v -timeout 60m` (`services/place-de-marche`) | `ok`, **97 tests de premier niveau, 0 skip, 0 échec**, dont `TestUS031GarantiesDeLaReservation` (198 s, 13 scénarios PASS), les 2 tests de configuration et, sans régression, `TestUS030BoutEnBoutTransactionsPlaceDeMarche` (177 s) sur le banc paramétré | 929 s |
| `go test ./... -count=1 -v` (`services/transactions`) | `ok` dans 4 paquets (`commandes`, `commandes/api`, `commandes/marche`, `montants`), **32 tests de premier niveau, 0 skip, 0 échec** | 13 s |
| Après les retours de revue : `go test ./cmd/place-de-marche/ ./internal/config/ -run TestUS031 -count=1 -v` | `ok` : banc 13 scénarios PASS (201 s, aucune garde d'échéance déclenchée), 2 tests de bout en bout de la configuration, 2 tests de table `LoadServer` (14 cas, 0,00 s) ; 0 skip | 209 s |
| Après les retours de revue : `go test ./... -count=1` (`services/transactions`) | `ok` dans 4 paquets, 0 skip | 13 s |
| `gofmt -l`, `go vet ./...`, `golangci-lint run --config <racine>/.golangci.yml ./...` dans `services/place-de-marche` et `services/transactions` | 0 issue (3 `gocognit` et 1 `staticcheck` sur le premier jet du test, corrigés par découpage ; 1 `gosec` G101 sur l'URL de base du test de table, mot de passe retiré) ; repassé après les retours de revue | — |
| Conteneurs | Arrêtés et supprimés par testcontainers en fin de test | — |

### AC1 — Deux acheteurs simultanés : une seule réservation acceptée, l'autre 409

| Scénario | Attendu (AC) | Réel | Statut |
|---|---|---|---|
| Par Transactions, 6 tours sur 6 articles : 2, 8, 16, 16, 20 et 20 acheteurs différents, départ synchronisé (82 demandes, chaque tour servi en 65 à 118 ms) | Une seule réservation acceptée, les autres 409 | À chaque tour : **exactement un 201**, tous les autres **409 `article-deja-reserve`** (Problem Details) ; gagnant différent d'un tour à l'autre | Conforme |
| Même chose, en base après chaque tour | Une seule réservation | 1 ligne `reservation` pour l'article, 1 réservation active, 1 seule `commande` (celle du gagnant), `article.reservation_id` → réservation du gagnant ; fiche `reserve: true` | Conforme |
| Directement sur « réserver » (port interne, jeton de service réel), 3 tours de 16 acheteurs | Une seule réservation acceptée | 1 × 201 et 15 × 409 `article-deja-reserve` par tour ; `article.reservation_id` = `reservationId` renvoyé au gagnant ; 1 ligne et 1 réservation active | Conforme |

### AC2 — Même acheteur, double clic ou délai dépassé : même réservation, même échéance

| Scénario | Attendu (AC) | Réel | Statut |
|---|---|---|---|
| 3 tours : 12 `POST /commandes` parallèles du même acheteur sur le même article | Réservation existante renvoyée, même échéance, ni seconde réservation ni seconde commande | 12 réponses en succès : **1 × 201 et 11 × 200**, même `id` de commande et même `echeance` partout ; en base 1 commande, 1 ligne `reservation`, commande liée à la réservation de l'article | Conforme |
| Relance séquentielle 1,5 s plus tard (délai dépassé côté client) | Idem | 200, même commande, même échéance | Conforme |
| 12 appels parallèles directs sur « réserver » du même acheteur | Idem | Tous en succès, **même échéance** que la commande ; toujours 1 ligne `reservation`, 1 commande | Conforme |
| Unitaire : relance 1 s avant l'échéance | Même commande | `Created` faux, même `id`, même échéance, `en_attente_paiement` | Conforme |

### AC3 — Une réservation active par acheteur

| Scénario | Attendu (AC) | Réel | Statut |
|---|---|---|---|
| Réservation active sur A, puis B, par Transactions et en direct | Refusée | 409 `reservation-deja-active` dans les deux cas ; B `reserve: false` (fiche et catalogue), `reservation_id` NULL ; 1 seule commande pour l'acheteur, aucune sur B ; la commande sur A reste lue à l'identique | Conforme |
| Après ce refus, un autre acheteur réserve B | B toujours réservable | 201 | Conforme |
| 3 tours : 6 demandes sur A et 6 sur B, mêlées et parallèles, même acheteur | Une seule réservation | À chaque tour un seul article retenu (6 succès sur lui), **6 × 409 `reservation-deja-active`** sur l'autre ; 1 commande, 1 réservation active pour l'acheteur ; l'article perdant reste libre | Conforme |
| Limite : la réservation sur A a expiré, l'acheteur réserve B | Plus de réservation **active** : acceptée | Avant l'échéance 409 ; après, 201 `en_attente_paiement` | Conforme |

### AC4 — Échéance passée : article de nouveau disponible, commande « expirée »

| Scénario | Attendu (AC) | Réel | Statut |
|---|---|---|---|
| Avant l'échéance (6 s) | Article réservé, commande en attente | Échéance à ≤ 6 s de la création ; `GET` → `en_attente_paiement` ; `reserve: true` à la fiche et au catalogue | Conforme |
| Échéance passée (+0,7 s), visiteur | Article de nouveau disponible | `reserve: false` à la fiche et dans le catalogue | Conforme |
| Échéance passée, `GET /commandes/{id}` par l'acheteur | Commande présentée « expirée » | 200, `statut` **`expiree`**, même `id`, même échéance, mêmes montants ; un autre acheteur : 404 | Conforme |
| Nouvel essai du même acheteur sur le même article | Nouvelle réservation, nouvelle commande | 201, nouvel `id`, `en_attente_paiement`, nouvelle `reservation_id`, échéance postérieure ; article de nouveau réservé ; l'ancienne commande reste lisible `expiree` ; 2 commandes et 1 seule réservation active pour l'article | Conforme |
| Autre acheteur après l'échéance | Article disponible pour tous | Refusé (409 `article-deja-reserve`) avant l'échéance, 201 après ; la commande du premier est `expiree` ; son nouvel essai : 409 `article-deja-reserve` ; réservation active au nom du second | Conforme |
| Unitaire `Order.AsOf` : 10 min et 1 ns avant ; **à l'échéance exacte** ; 1 ns et 1 an après ; échéance exacte dans un autre fuseau | `expiree` dès `echeance <= maintenant` (contrat) | `en_attente_paiement`, `en_attente_paiement`, **`expiree`**, `expiree`, `expiree`, `expiree` ; commande d'origine, échéance et montants inchangés | Conforme |
| Unitaire : commande expirée relue à un instant antérieur | Jamais réactivée (contrat) | Reste `expiree` | Conforme |
| Unitaire `Service` (horloge réglable) : lecture 1 ns avant puis à l'échéance ; nouvel essai après l'échéance | — | `en_attente_paiement` puis `expiree` ; nouvelle commande (`Created`) sur la nouvelle réservation, ancienne lue `expiree`, 2 commandes | Conforme |

### AC5 — Article réservé : modification et retrait du vendeur refusés en 409, prix figé

| Scénario | Attendu (AC et décision) | Réel | Statut |
|---|---|---|---|
| Article publié (80 € + 5 €) réservé, le vendeur envoie : contenu complet (titre, description, prix 1 €, frais 0) ; prix seul ; frais seuls ; titre « Retirée de la vente » ; révision périmée ; soumission | 409 `article-reserve` | **6 × 409 `urn:collector:probleme:article-reserve`** (Problem Details) ; la révision périmée reçoit aussi `article-reserve` (prioritaire, comme le prévoit le contrat) ; aucun corps ne contient le sujet de l'acheteur, l'échéance ni l'identifiant de la commande | Conforme |
| Base après ces 6 refus | Contenu, prix, frais, révision inchangés | Ligne `article` complète (`to_jsonb`) **identique** octet pour octet, `modifie_le` compris | Conforme |
| Vendeur relit l'article ; fiche publique ; commande | Prix figé | Révision et état `publie` inchangés, 80 € + 5 € ; fiche toujours `reserve: true` ; commande à 80 € | Conforme |
| Après l'échéance, le vendeur modifie le prix | De nouveau acceptée | 200, révision + 1, `en_controle`, 90 € | Conforme |
| Article publié marqué `vendu` en base, les 6 mêmes demandes | 409 `article-reserve` | 6 × 409 `article-reserve`, ligne `article` identique | Conforme |
| Course : modification du vendeur contre 2 acheteurs par Transactions, 12 tours, départ du vendeur décalé de 0 à 88 ms (4 passes) | Jamais les deux succès | **Jamais 200 et 201 ensemble.** À chaque passe, 11 tours : réservation d'abord, modification 409 `article-reserve`, article intact (révision, prix, frais, état) ; 1 tour : modification d'abord (200), les deux réservations 422 `article-non-reservable`, aucune commande, aucune réservation | Conforme |
| Course : modification contre 2 appels directs à « réserver », 12 tours, décalage de 0 à 22 ms | Jamais les deux succès | Jamais les deux, sur 4 passes : 12/0, 10/2, 10/2 et 11/1 (réservation d'abord / modification d'abord) ; chaque issue cohérente comme ci-dessus | Conforme |
| Première passe (sans décalage), 10 tours par Transactions | Jamais les deux succès | 10 tours modification d'abord, réservations 422, aucune commande | Conforme |

### Configuration — `RESERVATION_DUREE`

| Scénario | Attendu (README) | Réel | Statut |
|---|---|---|---|
| `LoadServer` : `RESERVATION_DUREE` = `999ms`, `500ms`, `1ns`, `0s`, `-5s`, `cinq` | Refusée (au moins `1s`) | Erreur « RESERVATION_DUREE : durée d'au moins 1s attendue (ex. 10m, 5s) » | Conforme |
| `LoadServer` : `RESERVATION_DUREE_MAX` = `999ms`, `0s`, `trente` | Refusée | Erreur nommant `RESERVATION_DUREE_MAX` | Conforme |
| `LoadServer` : `1s`/`1s`, `1.5s`/`30m`, `6s`/`6s`, `45m`/`20m`, absentes | Acceptées, plafond appliqué | Durée effective 1 s, 1,5 s, 6 s, 20 min, 10 min (défauts `10m`/`30m`) | Conforme |
| De bout en bout : `serve` avec `RESERVATION_DUREE` ou `RESERVATION_DUREE_MAX` = `999ms` | Refus au démarrage | Arrêt en ~1 ms, message nommant la variable | Conforme |
| De bout en bout : `serve` avec `1s` / `1s` | Pas de refus | Le service tourne jusqu'à l'arrêt demandé par le test | Conforme |

### Contrats OpenAPI

| Scénario | Attendu | Réel | Statut |
|---|---|---|---|
| Échanges de Transactions du banc (201, 200, 404, 409, 422), dont **3 réponses `statut: expiree`** | Conformes à `transactions/v1` | Conformes | Conforme |
| Échanges du port interne, directs et relayés (201, 200, 409, 422) | Conformes à `place-de-marche/interne-v1` | Conformes | Conforme |
| Catalogue public et **232 échanges des routes du vendeur** (dont les 409 `article-reserve`, en parallèle compris) | Conformes à `place-de-marche/v1` | Conformes | Conforme |

### Anomalies

Aucune : aucun comportement observé ne contredit un AC.

### Observations (sans anomalie)

- **Courses vendeur/acheteur** : 5 exécutions du banc (première passe sans décalage : 10 courses ; puis après correction, dans la suite complète, sur le code découpé et après les retours de revue : 24 courses chacune), soit 106 courses, jamais deux succès. Sans décalage, la modification du vendeur part presque toujours la première (chemin plus court que Transactions → Keycloak → Place de marché). Le décalage progressif du vendeur a fait apparaître les deux ordres ; dans les deux, une seule demande l'emporte et l'article reste cohérent.
- **AC2** : sur une rafale de 12, Transactions a toujours renvoyé un seul 201 et onze 200. L'AC n'exige que la même réservation, la même échéance et une seule commande ; le nombre de 201 est relevé pour information.
- **AC5, décision de pilotage** : le refus de la révision périmée est `article-reserve` (et non `revision-perimee`), conformément au contrat qui le rend prioritaire.
- Le banc d'US-030 reste inchangé dans son comportement (durées 45 min / 20 min) : seule la signature de démarrage est paramétrée.

### Retours de la revue de code (testeur)

| Retour | Traitement |
|---|---|
| `expiration_test.go` : erreur de la première `Place` ignorée | `premiereCommande` exige `err` nul, `Created` vrai et `en_attente_paiement` avant toute autre assertion |
| Échéance de 6 s pour tout le banc, y compris les scénarios qui exigent une réservation active | Durée courte gardée (AC4, reprise après échéance) ; garde explicite `exigerAvantLEcheance` dans AC1, AC2, AC3, AC4 (avant l'échéance) et AC5 (article réservé), relevée juste après chaque réponse et relecture. Aucune garde déclenchée à la relance |
| Test « une seconde acceptée » lent (5 s par cas) | Remplacé par un test de table instantané sur `config.LoadServer` ; un seul cas `1s` gardé de bout en bout |
| Texte `seller.reserved` reformulé par le front | `reservedMessage` mis à jour (FR/EN), 22 tests Vitest US-031 verts |

## Front

| Champ | Valeur |
|---|---|
| US | [US-031](../../../backlog/US-031-garanties-reservation.md) |
| Branche | `feat/US-031-garanties-reservation` (non commitée) |
| Date | 2026-11-16 |
| Fonction | Test |
| Périmètre | `web/` : écran d'une commande au statut `expiree` (AC4), refus 409 `article-reserve` à l'enregistrement et à la soumission du vendeur (AC5) |
| Référentiel | AC de l'US ; contrats `transactions/v1` (`statut` `expiree`) et `place-de-marche/v1` (`article-reserve`) |
| Méthode | Boîte noire : application complète (Vitest/jsdom, puis build de production servi à Playwright), Place de marché et Transactions simulés. Textes attendus écrits en dur |
| Verdict | **Conforme** : tous les AC du périmètre front sont couverts et conformes ; aucune anomalie |

### 1. Livrables de test

| Fichier | Contenu |
|---|---|
| `web/src/acceptance/US-031/AC4-commande-expiree.test.tsx` | 14 tests : titre « Réservation expirée », statut « Expirée », message de disponibilité, lien vers la fiche, échéance « dépassée », montants lisibles, aucune action de paiement ou de réservation, commande en attente sans mention d'expiration, axe-core ; FR/EN |
| `web/src/acceptance/US-031/AC5-article-reserve-vendeur.test.tsx` | 8 tests : message sur 409 `article-reserve` (modification, soumission), saisie conservée, annonce aux technologies d'assistance, article présenté publié et non réécrit ; FR/EN |
| `web/src/acceptance/US-030/reservationHarness.ts` | `statut` élargi à `'en_attente_paiement' \| 'expiree'` |
| `web/e2e/US-031/AC4-commande-expiree.spec.ts` | Playwright : 3 tests par langue (écran expiré, lien → fiche réservable → nouvelle commande, axe-core) |
| `web/e2e/support/reservation.ts` | `statut` élargi à `'en_attente_paiement' \| 'expiree'` |

### 2. Exécutions (dans `web/`)

| Commande | Résultat |
|---|---|
| `npx tsc -b` | Vert |
| `npm run lint` | Vert, 0 avertissement |
| `npx prettier --check .` | Vert |
| `npx vitest run src/acceptance/US-031` | 2 fichiers, **22 tests, 22 verts, 0 ignoré** (42 s) ; relancé avec le nouveau texte `seller.reserved` : 22 sur 22 verts (45 s) |
| `npx vitest run src/acceptance/US-030 src/acceptance/US-031 src/acceptance/US-013 src/acceptance/US-021` | 13 fichiers, 146 tests, 146 verts, 0 ignoré (non-régression : réservation, éditeur du vendeur, conflits 409) |
| `npx vitest run`, machine au repos | 43 fichiers, 514 tests, 0 ignoré : 511 verts, **3 échecs hors US-031** (US-008 AC10 « Se déconnecter » au clavier et focus au retour de connexion, US-025 AC5 anneau de focus sur « Se déconnecter » : dépassements de 5 s, les tests instables déjà signalés par le rapport d'US-030). Seuls : 19 sur 19 verts. **Aucun test US-030 ni US-031 en échec** |
| `E2E_LANGUES=fr,en npx playwright test e2e/US-030 e2e/US-031` | US-030 : 16 verts ; US-031 : 6 échecs à la première passe (**erreur de test** : la page de commande, protégée, redirige d'elle-même vers la connexion ; le test cherchait un bouton « Se connecter ») |
| `E2E_LANGUES=fr,en npx playwright test e2e/US-031` après correction | **6 verts, 0 ignoré** (55 s) |

### 3. Résultats par AC

#### AC4 — Commande présentée « expirée »

| Scénario | Attendu | Réel | Statut |
|---|---|---|---|
| Commande `expiree` ouverte par l'acheteur (FR/EN, Vitest et Playwright) | Présentée « expirée » | Titre « Réservation expirée » / « Reservation expired », statut « Expirée » / « Expired » ; ni « Article réservé » ni « En attente de paiement » | Conforme |
| Même écran | Article de nouveau disponible | « La réservation a expiré, l’article est de nouveau disponible. » / EN | Conforme |
| Même écran | Échéance dépassée | « Échéance dépassée le » suivi du `<time datetime>` de l'échéance reçue ; plus de « À régler avant le » | Conforme |
| Lien vers l'article (Vitest, puis Playwright jusqu'au bout) | L'acheteur retrouve l'article disponible | Lien `/articles/{id}` ; dans le navigateur, il mène à la fiche, « Réserver » actif, un clic crée une **nouvelle** commande et ouvre sa confirmation « Article réservé » (1 seul `POST`) | Conforme |
| Commande expirée | Reste lisible | Prix et total affichés au centime | Conforme |
| Commande expirée | Aucune action trompeuse | Ni « Réserver » ni paiement sur l'écran | Conforme |
| Commande en attente (non-régression) | Pas d'expiration affichée | « En attente de paiement », aucune mention « Expirée » | Conforme |
| axe-core (jsdom FR/EN ; Playwright WCAG 2.1 A/AA FR/EN) | 0 violation sérieuse ou critique | 0 | Conforme |

#### AC5 — Refus 409 `article-reserve` côté vendeur

| Scénario | Attendu | Réel | Statut |
|---|---|---|---|
| Article publié, modification enregistrée, réponse 409 `article-reserve` (FR/EN) | Refus expliqué | « Cet article est réservé par un acheteur ou déjà vendu : il ne peut être ni modifié ni soumis pendant la réservation, ni après la vente. » / « This item is reserved by a buyer or already sold: it cannot be changed or submitted during the reservation, nor after the sale. » (texte reformulé par le front après la revue) ; 1 seul `PUT` | Conforme |
| Même refus | Aucune perte de saisie | Titre et description saisis toujours dans le formulaire | Conforme |
| Même refus | Annoncé | Message présent dans une région `status`/`alert` | Conforme |
| Même refus | Rien de réécrit | Article toujours `publie`, révision 4, titre d'origine ; pas d'état « En contrôle » affiché | Conforme |
| Soumission refusée en 409 `article-reserve` (FR/EN) | Refus expliqué | Même message | Conforme |

### 4. Anomalies

Aucune.

### 5. Remarques

- AC1 à AC3 relèvent du back (concurrence, idempotence, une réservation active) : le front n'en voit que les refus 409 déjà couverts par US-030 (`article-deja-reserve`, `reservation-deja-active`).
- Le refus de soumission `article-reserve` est simulé sur un brouillon : en réalité seul un article publié peut être réservé ou vendu. Le test prouve seulement que le message est rendu sur la route de soumission.
- La table « Tests attendus » de l'US cite Playwright `request` pour AC2 à AC5 : ces preuves sont faites par le banc Go contre les vrais services (vraie base, vrai Keycloak, échéance courte de 6 s), plus fidèle qu'un `request` Playwright sur des services simulés. Le Playwright du front prouve le parcours de l'acheteur après expiration.
