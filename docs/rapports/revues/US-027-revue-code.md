# US-027 — Revue de code back

| Champ | Valeur |
|---|---|
| US | [US-027 — Limiter le débit sur les routes sensibles](../../../backlog/US-027-limitation-debit.md) |
| Branche | `feat/US-027-limitation-debit` (base `origin/main`, commits `8df122b`, `74cc651`) |
| Date | 2026-11-06 |
| Périmètre | `libs/platform/ratelimit` (`bucket.go`, `middleware.go`, `policy.go`), `libs/platform/httpserver/client.go`, `libs/platform/problem/problem.go`, Place de marché (`serve.go`, `internal/annonces/api/handlers.go`, `internal/config/config.go`), contrat `v1.openapi.yaml`, README de la Place de marché, tests, [rapport de tests](../tests/US-027-rapport-tests.md) |
| Références | [D08](../../decisions/D08-exposition-tls-domaine.md) (ME4, réserves sur `X-Forwarded-For` et table bornée), [D13](../../decisions/D13-strategie-tests-charge.md) (MT4, MT6), architecture § 9.8 |
| Verdict | **Fusion possible** (0 bloquante, 0 majeure, 3 mineures, 2 suggestions) |

## Vérifications exécutées

| Commande | Résultat |
|---|---|
| `go vet ./ratelimit/ ./httpserver/` (`libs/platform`) | 0 erreur |
| `go test -count=1 ./ratelimit/ ./httpserver/ ./problem/` (`libs/platform`) | Vert |
| `gofmt -l .` (`libs/platform`) | 0 fichier |
| `go vet ./...` (`services/place-de-marche`) | 0 erreur |
| `go test -count=1 -run US027 ./cmd/...` (`services/place-de-marche`) | Vert |
| Recherche de commentaires dans le diff (`//`, `/*`) | Aucun |
| Recherche des appelants de `api.Register` et `publicHandler` | `serve.go:58`, `environnement_test.go:156` : tous mis à jour |
| Recherche d'autres lecteurs de `X-Forwarded-For` / `RemoteAddr` dans le code Go | Aucun hors `httpserver/client.go` |

`go test -race`, `golangci-lint` et les bouts en bout testcontainers sont tracés verts dans le rapport de tests ; non rejoués ici.

## Points forts

- Admission atomique et tout-ou-rien (`bucket.go:65-89`) : un seul verrou, les deux seaux (adresse, compte) sont rechargés puis débités ensemble seulement si aucun n'est vide ; une requête refusée ne consomme rien. `Retry-After` prend l'attente la plus longue des seaux refusés (`bucket.go:78`), arrondie au supérieur, bornée à `[1, 3600]` (`bucket.go:37-40`) ; jamais 0 ni négatif (les jetons ne descendent pas sous 0).
- Contournement par en-tête fermé : seule la dernière valeur de la dernière ligne est lue (`client.go:19-26`), une valeur forgée en tête ou une ligne forgée en amont est ignorée ; une dernière valeur illisible retombe sur l'adresse source au lieu de créer une clé libre. Aucune route ni en-tête de contournement : la limite de tir ne dépend que de `LIMITE_DEBIT_TIR_ADRESSE` comparée à l'adresse lue par la même règle (`middleware.go:62-67`).
- IPv6 : clé par /64 masqué (`middleware.go:95-100`), forme IPv4 mappée ramenée en IPv4 et zone retirée (`client.go:41`) : pas de rotation d'adresses dans un même /64 ni de doublon `::ffff:a.b.c.d`.
- Clé de compte issue du sujet du jeton vérifié (`middleware.go:55-57`), préfixes `adresse:` / `compte:` sans collision possible ; le limiteur est posé après authentification et rôle (`handlers.go:78-80`), donc ni 401 ni 403 ne consomme la rafale d'un vendeur légitime.
- Mémoire bornée conformément à la réserve de D08 (M-DIS-02) : purge des seaux pleins chaque minute (`bucket.go:110-121`) et plafond de 50 000 clés (`bucket.go:93-95`).
- Configuration stricte : valeurs non entières, nulles ou négatives et adresse illisible refusent le démarrage (`policy.go:79-100`) au lieu d'un repli silencieux.
- Régressions traitées : les deux appelants de `publicHandler` sont adaptés ; les bancs existants reçoivent un limiteur sans effet (`environnement_test.go:134-137`) et le bout en bout réel des limites hautes par l'environnement (`services_test.go:64-65`), sans affaiblir le code de production.
- Contrat et documentation alignés : réponse `TropDeRequetes` (429 Problem Details, `Retry-After` entier ≥ 1) sur les deux routes limitées (`v1.openapi.yaml:210-211`, `:258-259`, `:820-832`) ; README (routes, deux seaux, règle d'adresse, IPv6, purge, écart § 9.8, 5 variables avec défauts) conforme au code.

## Issues

### Bloquantes

Aucune.

### Majeures

Aucune.

### Mineures

1. `libs/platform/ratelimit/middleware.go:73` — le champ de journal `adresse_client` reçoit la clé du seau, pas l'adresse : pour un client IPv6 on journalise `2001:db8:1:2::/64`, l'adresse exacte est perdue. ME4 attend comme preuve « 429 et adresse réelle journalisée ». Journaliser l'adresse lue par `httpserver.ClientAddress` dans `adresse_client` et, si utile, la clé dans un champ distinct (`cle_limite`) ; cela demande de faire remonter l'adresse jusqu'à `reject` (`clientKeyOf` rend aujourd'hui seulement la clé). Dev : back.
2. `libs/platform/ratelimit/middleware.go:96` et `libs/platform/httpserver/client.go:41` — la normalisation `Unmap().WithZone("")` est écrite deux fois ; elle n'est nécessaire dans `addressKey` que parce que `optionalAddress` (`policy.go:91-100`) re-parse l'adresse de tir sans la normaliser. Normaliser une seule fois à la lecture (dans `optionalAddress`, ou exposer depuis `httpserver` la fonction de lecture d'adresse et la réutiliser dans `policy.go`) ; `addressKey` se réduit alors au masquage IPv6. Dev : back.
3. `libs/platform/ratelimit/bucket.go:69` — `granted` contient tous les seaux consultés, y compris ceux qui refusent ; le nom est faux sur le chemin de refus. Renommer (`consulted`, `buckets`). Dev : back.

### Suggestions

1. `libs/platform/httpserver/client.go:24` — la lecture de la dernière ligne est juste si l'ALB ajoute l'adresse du client à la dernière ligne `X-Forwarded-For` (ou fusionne les lignes). Le comportement de l'ALB face à une requête portant **deux** lignes `X-Forwarded-For` n'est pas documenté dans D08 et n'est pas vérifiable en local (hypothèse, non un défaut constaté). Ajouter ce cas au script curl après déploiement (AC4/MT4) : deux `-H 'X-Forwarded-For: …'` tournants, limite atteinte → 429 attendu. Dev : test.
2. AC3, volet `up.sh` — aucun chart ni script de déploiement de la Place de marché n'existe encore (seul `sandbox/XP01-orchestrateur/up.sh`) : l'injection de `LIMITE_DEBIT_TIR_ADRESSE` à l'exécution reste à faire et à relire (contrôle manuel tracé prévu par l'US) quand le déploiement arrivera, comme le branchement du dépôt de photo (US-020) et de la réservation (Transactions) pour AC1. À garder dans le suivi de l'US plutôt que dans ce diff. Dev : DevOps.

## Verdict

| Sévérité | Nombre |
|---|---|
| Bloquante | 0 |
| Majeure | 0 |
| Mineure | 3 |
| Suggestion | 2 |

**Fusion possible.** Aucun bug de concurrence, de contournement, de fuite mémoire ni de `Retry-After` constaté ; aucune régression (tous les sites d'appel adaptés, tests verts). AC1, AC2, AC4 et AC5 sont couverts pour les deux routes livrées ; AC3 l'est pour la configuration, le volet `up.sh` dépend du déploiement. Les mineures peuvent être soldées dans un commit de nettoyage.
