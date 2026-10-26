# US-017 — Rester disponible quand une instance tombe sous charge

| Champ | Valeur |
|---|---|
| Epic | E4 Exploitation, charge et disponibilité |
| Statut | À faire |
| Sprint | 2 (J5) |
| Engagement | Ferme |
| Estimation | 2 points |
| Dépend de | US-016 |
| Fonction(s) | DevOps |
| Service | Place de marché, Plateforme |
| Critères grille | C5, C6 |
| Exigences | ENF-11 |
| Branche | `feat/US-017-perte-instance-sous-charge` |

## Récit

En tant que visiteur, je veux que le catalogue continue de répondre même si un serveur tombe pendant un pic, afin de ne pas voir d'erreur au moment où je cherche un objet.

## Critères d'acceptation

### Fonctionnels

- **AC1** — Given un tir en cours sur le catalogue, When un pod de la Place de marché est supprimé de force, Then le taux d'erreurs sur la fenêtre du tir reste inférieur à 1 % et un remplaçant est prêt en 60 secondes au plus (X10).
- **AC2** — Given l'événement, When la mesure est relevée, Then l'indicateur I3 (erreurs à la perte d'un pod) est inscrit dans le relevé du sprint.
- **AC3** — Given `tools/demo.sh`, When le palier « perte d'un pod » est atteint, Then le script supprime le pod et affiche la reprise.

### Non fonctionnels

- **AC4 (Disponibilité)** — Given le Deployment de la Place de marché, When sa configuration est relue, Then il a au moins 2 réplicas répartis sur des nœuds différents et un budget d'interruption qui garde au moins un pod.

## Sécurité

Aucune exigence ES directement servie ; la disponibilité est un risque relevé en [exigences-securite.md](../docs/securite/exigences-securite.md) § 4.9.

## Tests attendus

| Type | Outil | Ce qui est prouvé | Fonction |
|---|---|---|---|
| Résilience | `kubectl delete pod` pendant le tir k6 | AC1, AC2, AC3 | DevOps |
| Configuration en code | Relecture du chart rendu | AC4 | Revue de code |

## Hors périmètre

- Arrêt d'un nœud (X11) : non éliminatoire, hors US.

## Décisions liées

- [D03 — Hébergement et orchestration](../docs/decisions/D03-hebergement-orchestration.md) (X10) · [D13](../docs/decisions/D13-strategie-tests-charge.md) · [D12](../docs/decisions/D12-indicateurs-qualite.md) (I3)

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
