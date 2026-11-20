# Indicateurs du sprint 1

Run de référence : non disponible (relevé non généré) · Tir : sans objet au sprint 1 · Date : 2026-11-20 (relevé a posteriori)

Définitions, seuils et règle de publication : [qualite-indicateurs.md](../processus/qualite-indicateurs.md), [D12](../decisions/D12-indicateurs-qualite.md). Sprint : [sprint-1.md](../../backlog/sprints/sprint-1.md).

**Limite du relevé** : le programme `tools/indicateurs` prévu par D12 n'existe pas (DT-01). Ce relevé est rédigé à la main à partir du dépôt, des rapports de test et de l'historique de `main`, sans lecture des artefacts de CI (`qualite.json`, `i6`, JUnit) ni de l'API GitHub. Aucun `indicateurs-sprint-1.json` n'est produit : la tendance du sprint 2 ne peut pas être calculée. Une valeur non mesurée n'est pas zéro.

| # | Indicateur | Valeur | Cible | Sprint N-1 | Tendance | État |
|---|---|---|---|---|---|---|
| I1 | AC couverts (dont sécurité) | Non mesuré : JUnit de `main` non croisé avec les AC | 100 % | — | Référence | Non mesuré |
| I2 | p95 catalogue ; délai de mise à l'échelle | Sans objet au sprint 1 (premier tir au sprint 2) | < 500 ms ; ≤ 90 s | — | — | Non mesuré |
| I3 | 5xx sous perte de pod ; perdus ; écartés | Sans objet au sprint 1 | < 1 % ; 0 ; 0 | — | — | Non mesuré |
| I4 | Couverture par service (min.) | Pas de relevé sur `main` ; valeurs ponctuelles des rapports de test ci-dessous | Publiée sans porte au sprint 1 | — | Référence | Non mesuré (valeurs indicatives) |
| I5 | Fonctions > 15 ; duplication | Non mesuré : porte `gocognit` active sur le Go, sans comptage global ; `sonarjs` et jscpd absents de la CI | 0 nouvelle ; < 3 % | — | Référence | Non mesuré |
| I6 | Ouvertes par sévérité ; exceptions ; hors délai | Exceptions : 0 (`.trivyignore` vide au 2026-10-28) ; vulnérabilités ouvertes : non relevées (artefact `i6` non lu) | 0 ; 0 échue ; 0 | — | Référence | Partiel |
| I7 | Réussite ; délai de livraison ; durée ; relances | Non mesuré : API GitHub non relevée. Constat : 9 PR d'US mergées du 2026-10-26 au 2026-10-28 | ≥ 90 % ; < 1 j ; < 15 min | — | Référence | Non mesuré |

Vélocité (hors D12, suivie par [BACKLOG.md](../../backlog/BACKLOG.md) § 2) : 31 points mergés sur 54 engagés, dont 18 points d'US complètes et 13 points d'US livrées hors AC d'environnement ; 23 points reportés faute d'AWS.

### Valeurs indicatives de couverture (I4)

Mesures du poste de test, relevées dans les rapports, pas sur `main` : elles ne servent pas de référence « pas de baisse ».

| Module ou paquet | Couverture | Source |
|---|---|---|
| `services/place-de-marche` (total, `-coverpkg`) | 77,3 % | [US-013](../rapports/tests/US-013-rapport-tests.md) |
| `services/controle` `regles` | 100 % | [US-013](../rapports/tests/US-013-rapport-tests.md) |
| `libs/platform` `message` · `token` · `logging` · `httpjson` | 82,8 % · 85,9 % · 87,5 % · 76,7 % | [US-013](../rapports/tests/US-013-rapport-tests.md) |
| `libs/platform` `broker` · `outbox` (couverts par le bout en bout seulement) | 63,0 % · 76,1 % | [US-013](../rapports/tests/US-013-rapport-tests.md) |
| `contracts` `message` | 83,5 % | [US-012](../rapports/tests/US-012-rapport-tests.md) |
| `web` (lignes) | 86,92 % | [US-010](../rapports/tests/US-010-rapport-tests.md) |

## Trois pires éléments par indicateur

| # | Élément 1 | Élément 2 | Élément 3 |
|---|---|---|---|
| I1 | US-008 : AC3, AC6 à AC9 sans preuve (realm à chaque `up`, rotation, X17 étendu, contrôle du realm en CI, en-têtes nginx) | US-009 : AC5, AC10, AC11 à faire (OpenAPI en CI, Job d'amorçage, axe sur le formulaire) | US-010 : AC3, AC5 sans preuve (amorçage du jeu de démo, temps de réponse à vide) |
| I2 | Non mesuré | | |
| I3 | Non mesuré | | |
| I4 | `libs/platform/broker` : 63,0 %, sans tests propres | `libs/platform/outbox` : 76,1 %, sans tests propres | Contrôle : couverture non mesurée par le bout en bout (processus séparé) |
| I5 | Non mesuré | | |
| I6 | Non mesuré (compteurs `i6` non lus) | | |
| I7 | Non mesuré | | |

## Changements de seuil ou de périmètre

Aucun. Les seuils de départ de D12 devaient être confirmés sur cette première mesure ; ils ne le sont pas, faute de relevé généré.

## Analyse

- Cinq indicateurs sur sept étaient attendus au sprint 1 (I1, I4 à I7) : aucun n'est mesuré selon sa formule. Les outils de mesure sont en place dans la CI pour I4 (couverture publiée), I5 (porte `gocognit`) et I6 (job `securite`, compteurs `i6`) ; il manque le relevé (`tools/indicateurs`) et, pour I5, la duplication et la complexité du front.
- Les AC sans preuve (I1) sont tous des AC d'environnement ou de CI, bloqués par l'absence d'AWS ou non repris après la publication du dépôt.
- La couverture des briques partagées du canal (`broker`, `outbox`) repose sur le seul bout en bout.

## Dettes ouvertes (`DT-nn`)

| ID | Constat | Inscrite dans |
|---|---|---|
| DT-01 | `tools/indicateurs` absent : relevés non générés, pas de JSON, tendance non calculable | [sprint-1.md](../../backlog/sprints/sprint-1.md) |
