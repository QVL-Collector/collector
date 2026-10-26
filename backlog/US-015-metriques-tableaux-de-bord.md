# US-015 — Métriques et tableaux de bord de la plateforme

| Champ | Valeur |
|---|---|
| Epic | E4 Exploitation, charge et disponibilité |
| Statut | À faire |
| Sprint | 2 (J5) |
| Engagement | Ferme |
| Estimation | 3 points |
| Dépend de | US-007, US-013 |
| Fonction(s) | DevOps, Back (instrumentation) |
| Service | Plateforme, Place de marché, Contrôle |
| Critères grille | C5, C6 |
| Exigences | ENF-14, ENF-15, ENF-25 |
| Branche | `feat/US-015-metriques-tableaux-de-bord` |

## Récit

En tant qu'exploitant, je veux voir en direct la latence, les erreurs, le nombre de pods et la longueur des files, afin de démontrer la montée en charge et de mesurer les indicateurs I2 et I3 à chaque sprint.

## Critères d'acceptation

### Fonctionnels

- **AC1** — Given les services instrumentés (OpenTelemetry dans `libs/platform`), When Prometheus les relève sur leur port de métriques distinct, Then le débit, la latence p95, le taux d'erreurs par route et la longueur des files sont disponibles.
- **AC2** — Given Grafana, When l'exploitant l'ouvre par `port-forward`, Then les tableaux de bord versionnés affichent latence p95, erreurs, réplicas HPA, longueur des files et nœuds.
- **AC3** — Given les règles d'alerte versionnées, When un message entre dans une file `.ecartes` ou que l'outbox n'est plus publiée, Then l'alerte est visible dans Grafana en moins de 2 minutes.
- **AC4** — Given un tir terminé, When l'export est lancé, Then les métriques de la fenêtre sont exportées en CSV dans `docs/mesures/charge/<date>/`.

### Non fonctionnels

- **AC5 (Sécurité)** — Given Grafana et Prometheus, When l'inventaire des points d'entrée est fait, Then ils sont en `ClusterIP`, absents de l'ALB ; le mot de passe Grafana est généré à chaque `up`, la connexion anonyme et l'inscription sont désactivées, et le couple `admin/prom-operator` est refusé.

## Sécurité

| Exigence | Comment l'US la sert | AC |
|---|---|---|
| ES-07 | Outils d'observabilité non exposés | AC5 |
| ES-06 | Mot de passe généré par session | AC5 |
| ES-31 | Alerte sur messages écartés et outbox | AC3 |

Mesures : MO4, MO5.

## Tests attendus

| Type | Outil | Ce qui est prouvé | Fonction |
|---|---|---|---|
| Après déploiement | Requêtes Prometheus, inventaire, connexion Grafana refusée | AC1, AC2, AC5 | DevOps, Test |
| Résilience | Message invalide injecté | AC3 | Test |
| Contrôle manuel tracé | Export CSV après tir | AC4 | DevOps |

## Hors périmètre

- Notification externe des alertes : hors POC.

## Décisions liées

- [D09 — Observabilité](../docs/decisions/D09-observabilite.md) · [D12 — Indicateurs qualité](../docs/decisions/D12-indicateurs-qualite.md)

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
