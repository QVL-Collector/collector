# US-005 — Garde-fous de coût : destruction programmée et journal des coûts

| Champ | Valeur |
|---|---|
| Epic | E2 Plateforme AWS |
| Statut | À faire |
| Sprint | 1 (J4) |
| Engagement | Ferme |
| Estimation | 2 points |
| Dépend de | US-004 |
| Fonction(s) | DevOps |
| Service | Plateforme |
| Critères grille | C5 |
| Exigences | ENF-16, ENF-28 |
| Branche | `feat/US-005-garde-fous-cout` |

## Récit

En tant que Lead Developer, je veux que l'environnement se détruise de lui-même avant la fin de la session et que chaque session laisse une trace de son coût, afin qu'un oubli ne consomme pas le budget de 50 $ et que le POC reste démontrable jusqu'à la soutenance.

## Critères d'acceptation

### Fonctionnels

- **AC1** — Given un `up` dont l'`apply` a réussi, When le script se poursuit, Then la destruction programmée est armée immédiatement, même si la suite échoue : node group, RDS, ALB et NAT 30 minutes avant la fin de session ; cluster et adresse 10 minutes avant (G1).
- **AC2** — Given une destruction programmée armée, When la fin de session approche sans `down`, Then les ressources facturables sont réellement supprimées avant l'expiration des identifiants (X15), et l'inventaire strict suivant le confirme.
- **AC3** — Given un nouveau « Start Lab » pendant une session, When `tools/up.sh --prolonger` s'exécute, Then les planifications sont repoussées.
- **AC4** — Given un `down` réussi, When il se termine, Then les planifications restantes sont supprimées.
- **AC5** — Given chaque session, When `down` se termine, Then une ligne (date, durée allumée, coût estimé) est ajoutée à `docs/mesures/couts.csv` et le cumul du projet est affiché.
- **AC6** — Given un cumul estimé supérieur ou égal à 70 % du budget réel (35 $), When `tools/up.sh` s'exécute sans l'option explicite de répétition de soutenance, Then il refuse de démarrer (G5).
- **AC7** — Given un `up` réussi, When il se termine, Then un rappel est programmé sur le poste 45 minutes avant la fin de session (G7).

### Non fonctionnels

- **AC8 (Sécurité)** — Given le journal des coûts, When il est relu avant commit, Then il ne contient ni identifiant de compte, ni adresse IP, ni identifiant de session.

## Sécurité

| Exigence | Comment l'US la sert | AC |
|---|---|---|
| ES-29 | Alerte de coût couverte par le journal et le seuil d'arrêt (G5) | AC5, AC6 |

Mesures : MS10, MI5.

## Tests attendus

| Type | Outil | Ce qui est prouvé | Fonction |
|---|---|---|---|
| Après déploiement | Inventaire strict après déclenchement de G1 sur une session courte | AC1, AC2, AC4 | DevOps |
| Unitaire (script) | Jeu de journaux fictifs pour le calcul du cumul et du seuil | AC5, AC6 | DevOps |
| Contrôle manuel tracé | Relecture du journal avant commit | AC8 | Test |

## Hors périmètre

- Alerte AWS Budgets (G6) : seulement si la sonde XP00 l'a montrée créable.

## Décisions liées

- [D11 — IaC, allumage et extinction, garde-fou coût](../docs/decisions/D11-iac-allumage-cout.md) (G1, G5, G7 ; X15)

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
