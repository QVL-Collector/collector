# US-011 — Canal asynchrone avec une identité par service

| Champ | Valeur |
|---|---|
| Epic | E2 Plateforme AWS |
| Statut | À faire |
| Sprint | 1 (J4) |
| Engagement | Ferme (condition de WS-2 : verdict de XP02 concluant ; sinon WS-1 et report au sprint 2) |
| Estimation | 3 points |
| Dépend de | US-004, US-006 ; verdict de XP02 |
| Fonction(s) | DevOps |
| Service | Plateforme (RabbitMQ) |
| Critères grille | C2, C8 |
| Exigences | ENF-12 |
| Branche | `feat/US-011-canal-rabbitmq-identites` |

## Récit

En tant que Place de marché et Contrôle de publication, nous voulons échanger par un canal durable où chacun n'écrit que sur ses propres sujets, afin qu'aucune demande ne soit perdue et qu'aucun service ne puisse se faire passer pour un autre.

## Critères d'acceptation

### Fonctionnels

- **AC1** — Given un `up`, When le chart est installé, Then RabbitMQ 4.x tourne dans le cluster et la topologie (échanges, files quorum de demandes et de résultats, files `.ecartes`) est posée par le fichier de définitions, pas par les services.
- **AC2** — Given les identités `marche` et `controle`, When chacune publie sur son sujet autorisé, Then le message est accepté et livré à la file prévue.
- **AC3** — Given un message déposé, When le broker redémarre (pod supprimé), Then le message est toujours présent dans sa file quorum, ou les articles « en contrôle » sont renvoyés par la Place de marché (écart documenté : broker sans volume persistant).

### Non fonctionnels

- **AC4 (Sécurité)** — Given l'identité `controle`, When elle publie une demande de contrôle, Then la publication est refusée ; given `marche`, When elle publie un résultat de contrôle, Then elle est refusée.
- **AC5 (Sécurité)** — Given n'importe quelle identité de service, When elle tente de déclarer une file ou un abonnement, Then l'opération est refusée.
- **AC6 (Sécurité)** — Given chaque `up`, When les identifiants du broker sont créés, Then ils sont générés pour la session, injectés par l'entrée standard dans un Secret par service, et absents du dépôt, des valeurs Helm et des journaux (X20 rejoué).
- **AC7 (Sécurité)** — Given la console de gestion du broker, When elle est demandée depuis Internet ou depuis un pod non autorisé, Then elle est injoignable ; elle n'est accessible que par `port-forward`.

## Sécurité

| Exigence | Comment l'US la sert | AC |
|---|---|---|
| ES-38 | Une identité par service, droits par sujet, aucune création de file | AC2, AC4, AC5 |
| ES-06 | Identifiants générés à chaque session | AC6 |
| ES-07 | Console non exposée | AC7 |
| ES-31 | Canal durable, pas de perte silencieuse | AC1, AC3 |

Mesures : MC1, MC3, MC4, MC5, MC8. Critères du bac à sable rejoués : XC2, XC7.

## Tests attendus

| Type | Outil | Ce qui est prouvé | Fonction |
|---|---|---|---|
| Intégration | testcontainers-go (RabbitMQ avec le même fichier de définitions) | AC2, AC4, AC5 | DevOps, Back |
| Après déploiement | Script de publication et de refus par identité, inventaire | AC1, AC6, AC7 | DevOps |
| Résilience | Suppression du pod du broker | AC3 | DevOps |

## Hors périmètre

- Identité `interactions` et diffusion de la décision de publication : US-028.
- Mise à l'échelle sur la file : US-026.

## Décisions liées

- [D05 — Communication interservice](../docs/decisions/D05-communication-interservice.md)
- [D14 — Expérimentations](../docs/decisions/D14-experimentations.md) (XP02)

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
