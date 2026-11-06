# Rapport de tests — US-027 Limiter le débit sur les routes sensibles

| Champ | Valeur |
|---|---|
| US | [US-027](../../../backlog/US-027-limitation-debit.md) |
| Branche | `feat/US-027-limitation-debit` |
| Date | 2026-11-06 |
| Fonction | Test |
| Périmètre | Back : `libs/platform/ratelimit`, `libs/platform/httpserver.ClientAddress`, Place de marché (soumission `POST /marche/v1/vendeur/articles/{id}/soumission`, modification `PUT /marche/v1/vendeur/articles/{id}`). AC1, AC2, AC4 en local ; AC3 par le chargement de la configuration et la limite relevée pour la seule adresse de tir ; AC5 par le retour de la limite nominale après un tir |
| Référentiel | AC de l'US ; contrat OpenAPI (réponse `TropDeRequetes` : 429 Problem Details, `Retry-After` entier ≥ 1) ; README de la Place de marché § Limitation de débit (variables `LIMITE_DEBIT_*`, adresse et compte, IPv6 par /64) |
| Verdict | **Conforme** : tous les AC couvrables ici sont couverts et conformes ; aucune anomalie. Volets « après déploiement » (script curl AC4/AC5 contre l'ALB, relecture des variables de session `up.sh` pour AC3) non couverts (voir § 6) |

## 1. Livrables de test

| Fichier | Contenu |
|---|---|
| `libs/platform/ratelimit/guard_test.go` | Seau de jetons à horloge injectée : rafale puis 429, `Retry-After` arrondi au supérieur, recharge, débit soutenu, IPv6 /64, `X-Forwarded-For` forgé, adresse de tir, fin de tir, purge et capacité, concurrence (16 tests, 28 avec les sous-tests) |
| `libs/platform/ratelimit/policy_test.go` | `LoadPolicy` : défauts, adresse de tir IPv4/IPv6 injectée, valeurs invalides refusées, plus petites valeurs acceptées (4 tests, 19 avec les sous-tests) |
| `libs/platform/httpserver/client_test.go` | `ClientAddress` : dernière valeur ajoutée par l'ALB, valeur unique et liste forgées, plusieurs lignes d'en-tête, IPv6, adresse source avec port, adresse illisible (6 tests, 21 avec les sous-tests) |
| `services/place-de-marche/cmd/place-de-marche/limitation_debit_test.go` | Intégration HTTP sur la Place de marché (en mémoire, vrai vérificateur de jetons) : 429 + `Retry-After` + Problem Details sur les deux routes, par adresse et par compte, 401/403 sans débit, forge, adresse de tir, routes non limitées, conformité OpenAPI des 429 (9 tests, 23 avec les sous-tests) |
| `services/place-de-marche/cmd/place-de-marche/environnement_test.go` | Réparation : `publicHandler` prend un 4e argument (limiteur) ; les bancs existants reçoivent un limiteur aux limites très hautes (`limiteurSansEffet`), `demarrerAvecLimiteur` pour les tests US-027 |
| `services/place-de-marche/cmd/place-de-marche/services_test.go` | Réparation : le service lancé par `serve` pour les bouts en bout reçoit `LIMITE_DEBIT_PAR_MINUTE=6000000` et `LIMITE_DEBIT_RAFALE=100000` (voir § 3) |

## 2. Exécutions

| Commande | Résultat |
|---|---|
| `libs/platform` : `go test -race -count=1 ./...` (conteneur `golang:1.26`, go1.26.10 linux/amd64, dépôt monté ; pas de cgo sous Windows) | Vert : httpjson, httpserver, logging, message, ratelimit, token |
| `libs/platform` : `go test -race -count=20 ./ratelimit/ ./httpserver/` (même conteneur) | Vert, 19 s ; aucune course détectée |
| `libs/platform` : `go test -count=1 ./...` (Windows) | Vert |
| `services/place-de-marche` : `go test -count=1 ./...` (Docker, testcontainers) | Vert, 113 s : PostgreSQL 18.3 et RabbitMQ 4.3.7 démarrés, bouts en bout US-013/US-018/US-019/US-021 exécutés, non ignorés |
| `golangci-lint run --config ../../.golangci.yml ./...` (2.15.0) dans `libs/platform` et `services/place-de-marche` | 0 issues dans les deux modules |
| `gofmt -l` sur les deux modules | 0 fichier |

## 3. Réparation des tests existants

- `environnement_test.go:145` ne compilait plus (`publicHandler` à 4 arguments). Les bancs existants passent désormais un limiteur `ratelimit.NewGuard` aux limites très hautes (60 000 000/min, rafale 1 000 000) : les scénarios antérieurs ne reçoivent jamais 429.
- Premier passage complet : `TestUS013BoutEnBoutPlaceDeMarcheEtControle` échouait (7 sous-tests, 429 `Too Many Requests` au lieu de 200/202/404). Ce bout en bout lance le vrai `serve` avec les valeurs par défaut (30/min, rafale 10) et enchaîne plus de 10 soumissions et modifications depuis 127.0.0.1 : c'est la limite nominale qui s'applique, comportement voulu par AC1 et non une anomalie. Le lanceur de test pose des limites très hautes par variables d'environnement ; ce passage prouve au passage que le binaire réel limite bien avec sa configuration par défaut.

## 4. Résultats par AC

### AC1 — Même adresse ou même compte au-delà de la limite : 429 avec `Retry-After`

| Scénario | Test | Résultat |
|---|---|---|
| Given 30/min rafale 10 When 10 requêtes d'une adresse puis une 11e Then 10 servies, 11e 429, `Retry-After` entier ≥ 1, `application/problem+json`, `status` 429, `type` et `title` renseignés ; le service n'a reçu que 10 requêtes | `TestUS027AC1RafaleServiePuis429AvecRetryAfter` | Conforme |
| `Retry-After` en secondes arrondies au supérieur : 30/min → 2, 7/min (8,57 s) → 9, 1/min → 60, 6000/min (0,01 s) → 1 | `TestUS027AC1RetryAfterEnSecondesArrondiesAuSuperieur` (4 cas) | Conforme |
| 1/min, 45,5 s écoulées → `Retry-After` 15 | `TestUS027AC1RetryAfterDecroitAvecLeTemps` | Conforme |
| Après l'attente annoncée par `Retry-After` : une requête servie, la suivante de nouveau 429 | `TestUS027AC1ServieDeNouveauApresRetryAfter` | Conforme |
| Débit soutenu à 30/min pendant 3 min : 90 requêtes servies, une de plus en avance → 429 | `TestUS027AC1DebitSoutenuAuRythmeConfigure` | Conforme |
| 24 h d'inactivité : la rafale reste bornée à 10 | `TestUS027AC1RafaleBorneeApresLongueInactivite` | Conforme |
| Une adresse épuisée ne pénalise pas une autre adresse | `TestUS027AC1AdressesIndependantes` | Conforme |
| IPv6 : 10 adresses d'un même /64 épuisent la limite, 11e du même /64 → 429 ; autre /64 servi | `TestUS027AC1AdresseIPv6LimiteeParSonPrefixe64` | Conforme (README) |
| 64 goroutines × 4 requêtes simultanées d'une même adresse, horloge figée → exactement 10 servies et 246 refusées (`-race`) | `TestUS027AC1RequetesConcurrentesBorneesALaRafale` | Conforme |
| Place de marché, soumission et modification : même adresse et même compte → 3 servies puis 429 Problem Details + `Retry-After` | `TestUS027AC1MemeAdresseEtMemeCompteObtiennent429AvecRetryAfter` (2 routes) | Conforme |
| Même compte depuis 4 adresses différentes → 4e en 429 (limite par compte) | `TestUS027AC1MemeCompteDepuisDesAdressesDifferentesObtient429` (2 routes) | Conforme |
| Même adresse pour 4 comptes différents → 4e en 429 (limite par adresse) | `TestUS027AC1MemeAdressePourDesComptesDifferentsObtient429` (2 routes) | Conforme |
| Après un abus, autre adresse et autre compte → 202 | `TestUS027AC1AutreAdresseEtAutreCompteNonPenalises` | Conforme |
| Limiteur après authentification et rôle : 9 requêtes sans jeton (401) et 9 d'un acheteur (403), jamais 429 ; ensuite un vendeur de la même adresse dispose de toute la rafale | `TestUS027AC1RequetesNonAuthentifieesOuSansRoleNeConsommentPasLaLimite` | Conforme |
| Catalogue, catégories, « Mes articles », lecture d'un article, création de brouillon : 30 appels d'une même adresse, jamais 429 | `TestUS027CatalogueEtAutresRoutesNonLimites` (5 routes) | Conforme |
| Réponses 429 des deux routes conformes au contrat OpenAPI (`TropDeRequetes`) | `TestUS027Reponses429ConformesAuContratOpenAPI` | Conforme |
| Purge et capacité : clé épuisée, 2 min, 60 000 adresses distinctes (au-delà des 50 000 clés), 2 min → une nouvelle adresse et l'adresse initiale reprennent leur rafale de 10 puis 429 | `TestUS027PurgeEtCapaciteLaLimiteContinueDeSAppliquer` | Conforme |

Dépôt de photo et réservation : routes absentes de la Place de marché à ce jour (US-020 et Transactions) ; non testables ici, l'intergiciel s'y branchera de la même façon (README).

### AC2 — Adresse du client : dernière valeur de `X-Forwarded-For` ajoutée par l'ALB (`append`)

| Scénario | Test | Résultat |
|---|---|---|
| Valeur unique ; valeur forgée puis ALB ; liste forgée puis ALB ; valeur non IP puis ALB ; espaces ; sans espace ; deux et trois lignes d'en-tête ; adresse de l'ALB aussi forgée en tête → la dernière valeur de la dernière ligne | `TestUS027AC2DerniereValeurAjouteeParLALB` (10 cas) | Conforme |
| Deux lignes dont la dernière diffère de la première → dernière valeur de la dernière ligne | `TestUS027AC2DerniereLigneDifferenteDeLaPremiere` | Conforme |
| IPv6 compacte, développée, après une valeur forgée → `2001:db8::5` | `TestUS027AC2AdresseIPv6AjouteeParLALB` (3 cas) | Conforme |
| Sans `X-Forwarded-For` : adresse source sans port, IPv4 et IPv6 (`[2001:db8::7]:443`) | `TestUS027AC2SansForwardedForAdresseSourceSansPort` (2 cas) | Conforme |
| Avec ou sans forge (`198.51.100.1`, liste, `::1`, `127.0.0.1`) → même adresse, donc même clé | `TestUS027AC2MemeCleAvecOuSansForgeEnTete` | Conforme |
| Adresse source illisible, sans en-tête → aucune adresse | `TestUS027AC2AdresseSourceIllisibleSansAdresse` | Conforme |

### AC3 — Limite relevée pour la seule adresse de tir, injectée à l'exécution, aucune route de contournement

| Scénario | Test | Résultat |
|---|---|---|
| Sans variable : 30/min rafale 10, tir 6000/min rafale 200, aucune adresse de tir | `TestUS027AC3ValeursParDefautSansAdresseDeTir` | Conforme |
| `LIMITE_DEBIT_TIR_ADRESSE` IPv4 et IPv6 et les 4 limites lues depuis l'environnement | `TestUS027AC3AdresseDeTirInjecteeALExecution` (2 cas) | Conforme |
| Valeurs refusées : débit et rafale nuls, négatifs, non numériques, décimal ; adresse de tir illisible ou hors plage | `TestUS027AC3ValeursInvalidesRefusees` (13 cas) | Conforme |
| Plus petites valeurs (1) acceptées | `TestUS027AC3PlusPetitesValeursAcceptees` | Conforme |
| Adresse de tir : 200 servies puis 429 ; une autre adresse reste à 10 puis 429 | `TestUS027AC3LimiteReleveePourLaSeuleAdresseDeTir` | Conforme |
| Adresse de tir forgée en tête, en ligne séparée ou dans une liste → limite nominale (10 puis 429) | `TestUS027AC3AdresseDeTirForgeeNeReleveRien` (3 cas) | Conforme |
| Sans adresse de tir configurée, l'adresse du poste de tir est à la limite nominale | `TestUS027AC3SansAdresseDeTirAucuneLimiteRelevee` | Conforme |
| Place de marché : même vendeur depuis l'adresse de tir → 20 servies (limite de tir, adresse et compte) puis 429 | `TestUS027AC3AdresseDeTirRelevePuisLimiteSurLaPlaceDeMarche` | Conforme |
| Place de marché : adresse de tir forgée devant l'adresse réelle → 429 à la limite nominale | `TestUS027AC4ForwardedForForgeResteA429SurLaPlaceDeMarche` « adresse de tir forgée » | Conforme |

Non couvert ici : l'injection par `up.sh` et l'absence de la valeur dans le dépôt relèvent du contrôle manuel tracé prévu par l'US (relecture des variables de session).

### AC4 (Sécurité) — `X-Forwarded-For` forgé (valeur unique ou liste) : la réponse reste 429

| Scénario | Test | Résultat |
|---|---|---|
| Valeur unique tournante, liste tournante, ligne forgée séparée, valeur illisible, adresse d'un autre client forgée, chacune suivie de l'adresse ajoutée par l'ALB → 10 servies puis 429 ; l'adresse réelle sans forge est aussi en 429 ; l'autre client reste servi | `TestUS027AC4ForwardedForForgeResteLimite` (5 cas) | Conforme |
| Limite atteinte, puis nouvelles forges (`192.0.2.1`, liste, `::1`, valeur vide) → 429 | `TestUS027AC4ForgeApresEpuisementResteLimitee` | Conforme |
| Place de marché, comptes différents à chaque requête : valeur unique, liste, adresse de tir forgées → 4e requête en 429 Problem Details + `Retry-After` | `TestUS027AC4ForwardedForForgeResteA429SurLaPlaceDeMarche` (3 cas) | Conforme |

### AC5 (Sécurité) — À la fin de chaque tir, le test de limitation rejoué obtient 429

| Scénario | Test | Résultat |
|---|---|---|
| Session de tir (150 requêtes de l'adresse de tir servies, client ordinaire en 429), puis politique rechargée sans `LIMITE_DEBIT_TIR_ADRESSE` → l'ancienne adresse de tir et le client obtiennent 429 dès la 11e requête | `TestUS027AC5FinDuTirLaLimiteNominaleRedonne429` | Conforme |
| Pendant le tir, les autres adresses restent à la limite nominale | `TestUS027AC3LimiteReleveePourLaSeuleAdresseDeTir` | Conforme |

## 5. Anomalies

Aucune anomalie : aucun comportement observé ne contredit un AC d'US-027.

## 6. Limites et AC non couverts ici

- **Script curl après déploiement (AC4, AC5)** prévu par l'US : non joué (pas d'ALB ni d'environnement déployé). Les preuves reposent sur l'intergiciel et la Place de marché en HTTP réel avec `X-Forwarded-For` simulant le mode `append` de l'ALB.
- **Contrôle manuel AC3** (variables de session de `up.sh`, absence de l'adresse dans le dépôt) : hors de cette campagne.
- **Dépôt de photo et réservation** : routes non encore livrées (US-020, Transactions) ; à couvrir à leur arrivée.
- `-race` exécuté dans un conteneur Linux (`golang:1.26`) faute de compilateur C sur le poste Windows.
- Limite par réplica non partagée : écart V1 assumé par l'US, non testé.

## 7. Conclusion

Tous les AC couvrables ici (AC1, AC2, AC4, AC3 pour la configuration et l'adresse de tir, AC5 pour le retour à la limite nominale) sont couverts et conformes. Restent le script curl après déploiement et le contrôle manuel des variables de session.
