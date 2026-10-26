# US-014 — Journaux centralisés et corrélés d'un bout à l'autre

| Champ | Valeur |
|---|---|
| Epic | E4 Exploitation, charge et disponibilité |
| Statut | À faire |
| Sprint | 1 (J4) |
| Engagement | Ferme (ordre de coupe S1 n° 2 : glisse en tête du sprint 2) |
| Estimation | 3 points |
| Dépend de | US-004, US-013 |
| Fonction(s) | DevOps (Fluent Bit, groupes de journaux), Back (`slog`, corrélation) |
| Service | Plateforme, Place de marché, Contrôle |
| Critères grille | C5, C6 |
| Exigences | ENF-25 |
| Branche | `feat/US-014-journaux-centralises` |

## Récit

En tant qu'exploitant, je veux retrouver dans CloudWatch tous les journaux d'une soumission, de la Place de marché au Contrôle, par un seul identifiant, afin de diagnostiquer un incident même après l'extinction du cluster.

## Critères d'acceptation

### Fonctionnels

- **AC1** — Given une soumission d'article, When elle est traitée, Then chaque journal JSON (`slog`) de la Place de marché et du Contrôle porte `service`, `correlation_id` et `trace_id`, l'identifiant de corrélation étant recopié dans chaque message du canal.
- **AC2** — Given Fluent Bit déployé en DaemonSet, When les services journalisent, Then les journaux arrivent dans le groupe applicatif (rétention 7 jours) et les événements `audit=true` dans le groupe d'audit (rétention 30 jours), tous deux dans la couche persistante.
- **AC3** — Given un incident provoqué (Contrôle arrêté pendant une soumission), When l'exploitant cherche l'identifiant de corrélation dans CloudWatch, Then il reconstitue le parcours et identifie la cause.
- **AC4** — Given une décision de contrôle appliquée, When elle est journalisée, Then un événement d'audit horodaté indique l'article, la révision et l'issue.

### Non fonctionnels

- **AC5 (Sécurité)** — Given un jeton, une URL présignée ou un secret dans une requête, When il est journalisé, Then il est masqué à la source puis dans Fluent Bit ; une valeur piégée est absente de CloudWatch.
- **AC6 (Sécurité)** — Given un événement d'audit émis par un service, When il est rangé, Then son origine vient des métadonnées Kubernetes, pas du contenu du message.

## Sécurité

| Exigence | Comment l'US la sert | AC |
|---|---|---|
| ES-23 | Journal d'audit hors du cluster, survivant à l'extinction | AC2, AC4, AC6 |
| ES-06 | Aucun secret en journal | AC5 |

Mesures : MS8, MO1 à MO3, MO7.

## Tests attendus

| Type | Outil | Ce qui est prouvé | Fonction |
|---|---|---|---|
| Unitaire back | `go test` (masquage, propagation) | AC1, AC5 | Back |
| Après déploiement | Requête CloudWatch Logs Insights sur un identifiant | AC2, AC3, AC4, AC6 | DevOps, Test |
| Sécurité négative | Valeur piégée recherchée dans CloudWatch | AC5 | Test |

## Hors périmètre

- Métriques, Prometheus et Grafana : US-015.

## Décisions liées

- [D09 — Observabilité](../docs/decisions/D09-observabilite.md)

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
