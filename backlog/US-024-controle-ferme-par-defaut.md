# US-024 — Aucune annonce publiée ni perdue quand le contrôle est indisponible

| Champ | Valeur |
|---|---|
| Epic | E4 Exploitation, charge et disponibilité |
| Statut | À faire |
| Sprint | 2 (J5) |
| Engagement | Ferme |
| Estimation | 2 points |
| Dépend de | US-013, US-015 |
| Fonction(s) | Back, DevOps |
| Service | Place de marché, Contrôle |
| Critères grille | C5, C8, C2 |
| Exigences | ENF-12 |
| Branche | `feat/US-024-controle-ferme-par-defaut` |

## Récit

En tant que vendeur, je veux que mon article soumis pendant une panne du contrôle soit traité dès son retour, sans jamais être publié sans contrôle, afin de ne perdre ni ma soumission ni la confiance des acheteurs.

## Critères d'acceptation

### Fonctionnels

- **AC1** — Given le Contrôle arrêté, When un vendeur soumet un article, Then l'article reste « en contrôle » et invisible au catalogue.
- **AC2** — Given le Contrôle relancé, When il reprend la file, Then l'article soumis pendant l'arrêt est contrôlé et son issue appliquée, sans perte ni doublon.
- **AC3** — Given un message invalide dans la file de demandes, When le Contrôle le reçoit, Then il est écarté vers la file `.ecartes` après un nombre borné d'essais, sans bloquer les messages suivants, et l'alerte d'US-015 se déclenche en moins de 2 minutes.
- **AC4** — Given la fin du test, When les comptages sont relevés, Then le nombre de demandes de l'outbox égale le nombre de résultats appliqués (indicateur I3, zéro événement perdu).

## Sécurité

| Exigence | Comment l'US la sert | AC |
|---|---|---|
| ES-31 | Mode dégradé fermé par défaut | AC1, AC2 |

Mesures : MC6, MC8, MC9. Critères du bac à sable rejoués : XC3, XC4, XC5.

## Tests attendus

| Type | Outil | Ce qui est prouvé | Fonction |
|---|---|---|---|
| Résilience | testcontainers-go (arrêt et relance du consommateur) à chaque PR | AC1, AC2, AC4 | Back |
| Intégration | Message invalide injecté | AC3 | Back |
| Après déploiement | Mise à zéro du Deployment du Contrôle, comptage SQL | AC1, AC2, AC4 | DevOps |

## Hors périmètre

- Reprise d'Interactions : US-029.

## Décisions liées

- [D05 — Communication interservice](../docs/decisions/D05-communication-interservice.md) · [D12](../docs/decisions/D12-indicateurs-qualite.md) (I3)

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
