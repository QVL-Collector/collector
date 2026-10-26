# US-006 — Cloisonnement du cluster : politiques réseau, admission, IMDS

| Champ | Valeur |
|---|---|
| Epic | E2 Plateforme AWS |
| Statut | À faire |
| Sprint | 1 (J4) |
| Engagement | Ferme |
| Estimation | 5 points |
| Dépend de | US-004 |
| Fonction(s) | DevOps |
| Service | Plateforme |
| Critères grille | C8, C2 |
| Exigences | ENF-03 |
| Branche | `feat/US-006-cloisonnement-cluster` |

## Récit

En tant que responsable de la sécurité de la plateforme, je veux que chaque pod ne puisse joindre que ce dont il a besoin et qu'aucun manifest dangereux ne soit admis, afin qu'un service compromis ne puisse ni atteindre les autres, ni obtenir les identifiants AWS du nœud.

## Critères d'acceptation

### Fonctionnels

- **AC1** — Given chaque namespace applicatif, When le chart est installé, Then une politique réseau refuse par défaut l'entrée et la sortie, et seuls les flux listés dans le chart sont ouverts (DNS, base pour les services qui en ont une, broker, Keycloak, ALB vers les NodePorts).
- **AC2** — Given les namespaces applicatifs, When un pod privilégié, en `hostNetwork` ou avec `hostPath` est soumis, Then il est refusé (PSS « restricted », X18).
- **AC3** — Given la politique d'admission, When un Service `LoadBalancer` ou un NodePort hors liste est soumis, Then il est refusé.
- **AC4** — Given les quotas posés par namespace, When un Deployment dépasse les plafonds (HPA 6, KEDA 3), Then la création est refusée.

### Non fonctionnels

- **AC5 (Sécurité, X16)** — Given un pod non autorisé, When il demande un jeton IMDSv2 puis les identifiants du rôle du nœud, Then la demande est refusée ; une requête IMDSv1 est refusée depuis tout pod ; les seuls pods autorisés (Place de marché pour S3, Fluent Bit, Cluster Autoscaler) obtiennent l'accès.
- **AC6 (Sécurité, X12, X19)** — Given un pod quelconque, When il tente de joindre la base, le broker ou la console de Keycloak sans flux autorisé, Then la connexion est refusée ; chaque refus est accompagné du test réussi du flux autorisé correspondant.
- **AC7 (Sécurité)** — Given un pod d'un namespace applicatif, When il tente de lire les Secrets d'un autre namespace, Then l'accès est refusé.

## Sécurité

| Exigence | Comment l'US la sert | AC |
|---|---|---|
| ES-38 | Cloisonnement réseau entre services, accès AWS réservé | AC1, AC5, AC6, AC7 |
| ES-07 | Aucune exposition par Service `LoadBalancer` ou NodePort hors liste | AC3 |

Mesures : MS3, MS4, MS5, MC3, MA8. Écart V1 assumé : identité d'exécution unique `LabRole` ([architecture-technique.md](../docs/architecture/architecture-technique.md) § 9.8).

## Tests attendus

| Type | Outil | Ce qui est prouvé | Fonction |
|---|---|---|---|
| Après déploiement | Pods de test (refus et flux autorisés), `kubectl auth can-i` | AC1, AC5, AC6, AC7 | DevOps, Test |
| Sécurité négative | Manifests non conformes soumis à l'admission | AC2, AC3, AC4 | Test |
| Configuration en code | Trivy `config` sur le chart rendu | AC1 à AC4 | DevOps |

## Hors périmètre

- Droits par sujet sur le broker : US-011.
- Politique « réserver » limitée à Transactions : US-030.

## Décisions liées

- [D03 — Hébergement et orchestration](../docs/decisions/D03-hebergement-orchestration.md) (MS3 à MS5 ; X12, X16, X18, X19)
- [D14 — Expérimentations](../docs/decisions/D14-experimentations.md) (critères bloquants rejoués sur l'IaC du POC)

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
