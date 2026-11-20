# Indicateurs du sprint 2

Run de référence : non disponible (relevé non généré) · Tir : non mesuré (aucune session AWS du POC, US-016 et US-017 reportées) · Date : 2026-11-20 (relevé a posteriori)

Définitions, seuils et règle de publication : [qualite-indicateurs.md](../processus/qualite-indicateurs.md), [D12](../decisions/D12-indicateurs-qualite.md). Sprint : [sprint-2.md](../../backlog/sprints/sprint-2.md). Sprint précédent : [indicateurs-sprint-1.md](indicateurs-sprint-1.md).

**Limite du relevé** : mêmes limites qu'au sprint 1 (DT-01) : relevé rédigé à la main, sans artefact de CI ni API GitHub, sans JSON. Le sprint 2 devait être le premier relevé complet (I1 à I7) : I2 et I3 manquent faute de tir, ce que [qualite-indicateurs.md](../processus/qualite-indicateurs.md) § 1 prévoit (session AWS manquée).

| # | Indicateur | Valeur | Cible | Sprint N-1 | Tendance | État |
|---|---|---|---|---|---|---|
| I1 | AC couverts (dont sécurité) | Non mesuré : JUnit de `main` non croisé avec les AC | 100 % | Non mesuré | Non calculable | Non mesuré |
| I2 | p95 catalogue ; délai de mise à l'échelle | Non mesuré : aucun tir, US-015 et US-016 reportées au sprint 3 | < 500 ms ; ≤ 90 s | — | — | Non mesuré |
| I3 | 5xx sous perte de pod ; perdus ; écartés | Non mesuré : aucun tir, US-017 reportée au sprint 3 | < 1 % ; 0 ; 0 | — | — | Non mesuré |
| I4 | Couverture par service (min.) | Pas de relevé sur `main` ; porte « pas de baisse » non active (DT-02) ; valeurs ponctuelles ci-dessous | Pas de baisse > 0,5 pt | Non mesuré | Non calculable | Non mesuré (valeurs indicatives) |
| I5 | Fonctions > 15 ; duplication | Non mesuré : porte `gocognit` seule (DT-03) | 0 nouvelle ; < 3 % | Non mesuré | Non calculable | Non mesuré |
| I6 | Ouvertes par sévérité ; exceptions ; hors délai | Exceptions : 0 (`.trivyignore` vide au 2026-11-06) ; vulnérabilités ouvertes : non relevées | 0 ; 0 échue ; 0 | 0 exception | Stable (exceptions) | Partiel |
| I7 | Réussite ; délai de livraison ; durée ; relances | Non mesuré : API GitHub non relevée. Constat : 10 PR d'US mergées (US-025 lot 1 dès le 2026-10-26, puis du 2026-10-28 au 2026-11-06) | ≥ 90 % ; < 1 j ; < 15 min | Non mesuré | Non calculable | Non mesuré |

Vélocité (hors D12) : 24 points livrés (22 fermes, 2 coupables) sur 37 fermes et 5 coupables ; 18 points reportés faute d'AWS. Non comparable au sprint 1 : les deux sprints se chevauchent et sont bornés par le blocage AWS, pas par la capacité.

Preuves de fiabilité hors I3 (tests fonctionnels, pas sous charge) : contrôle arrêté puis relancé sans annonce publiée ni perdue ([US-024](../rapports/tests/US-024-rapport-tests.md)). Elles ne remplacent pas la mesure I3 du tir.

### Valeurs indicatives de couverture (I4)

Mesures du poste de test, relevées dans les rapports, pas sur `main`.

| Module ou paquet | Couverture | Source |
|---|---|---|
| `services/place-de-marche` (total, `-coverpkg`) | 83,9 % | [US-023](../rapports/tests/US-023-rapport-tests.md) |
| `services/controle` `regles` · `coordonnees` · `canal` | 100 % · 98,5 % · 61,4 % | [US-019](../rapports/tests/US-019-rapport-tests.md) |

## Trois pires éléments par indicateur

| # | Élément 1 | Élément 2 | Élément 3 |
|---|---|---|---|
| I1 | US-022 : AC6 (second facteur admin) et parcours Playwright non couverts par la campagne | US-027 : AC4 et AC5 (script curl contre l'ALB) non joués | US-023 : AC2 livré sans photos (US-020 non livrée) ; US-025 : AC5, contrôle manuel au clavier non tracé |
| I2 | Non mesuré | | |
| I3 | Non mesuré | | |
| I4 | `services/controle/internal/canal` : 61,4 % | Porte « pas de baisse » absente : aucune baisse ne peut être détectée | |
| I5 | Non mesuré | | |
| I6 | Non mesuré (compteurs `i6` non lus) | | |
| I7 | Non mesuré | | |

## Changements de seuil ou de périmètre

Aucun changement de seuil. Écart de périmètre : la porte I4 « pas de baisse », bloquante dès le sprint 2 selon D12, n'est pas en place (DT-02).

## Analyse

- Aucun des sept indicateurs n'est mesuré selon sa formule ; I6 n'est connu que pour les exceptions (0, stable).
- I2 et I3, qui portent C5, dépendent tous deux de la session AWS 3 du sprint 3 (US-015, US-016, US-017) : sans elle, la soutenance n'a aucune mesure de charge du POC (seules existent celles du service de test de XP01).
- Les AC sans preuve se concentrent sur les vérifications après déploiement (ALB, realm, Playwright sur environnement), toutes dépendantes d'AWS.
- Action prioritaire pour C6 : écrire `tools/indicateurs` (DT-01) et activer la porte de couverture (DT-02) avant le relevé du sprint 3, pour disposer d'au moins une série mesurée.

## Dettes ouvertes (`DT-nn`)

| ID | Constat | Inscrite dans |
|---|---|---|
| DT-01 | `tools/indicateurs` absent : relevés non générés | [sprint-1.md](../../backlog/sprints/sprint-1.md) |
| DT-02 | Porte de couverture « pas de baisse » non active, pas de `couverture-reference.json` | [sprint-2.md](../../backlog/sprints/sprint-2.md) |
| DT-03 | Duplication (jscpd) et complexité front (`sonarjs`) non outillées | [sprint-2.md](../../backlog/sprints/sprint-2.md) |
