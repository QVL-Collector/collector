# US-018 — Refuser les annonces qui contiennent des coordonnées personnelles

| Champ | Valeur |
|---|---|
| Epic | E3 Mise en vente contrôlée |
| Statut | À faire |
| Sprint | 2 (J5) |
| Engagement | Ferme |
| Estimation | 3 points |
| Dépend de | US-013 |
| Fonction(s) | Back |
| Service | Contrôle |
| Critères grille | C3, C8 |
| Exigences | EF-17, EF-36 (annonces) |
| Branche | `feat/US-018-filtrage-coordonnees` |

## Récit

En tant que direction de Collector, je veux qu'une annonce contenant un email, un numéro de téléphone ou un lien externe soit refusée, afin que les ventes ne contournent pas la plateforme et sa commission.

## Critères d'acceptation

### Fonctionnels

- **AC1** — Given un titre ou une description contenant un email, un numéro de téléphone français ou un lien externe en clair, When l'article est contrôlé, Then il est « refusé » avec le motif « coordonnées détectées ».
- **AC2** — Given une variante masquée (« zéro six … », « arobase », « point com », chiffres séparés par des espaces), When l'article est contrôlé, Then il est refusé avec le même motif.
- **AC3** — Given un texte sans coordonnées mais avec des dates, dimensions, références d'édition ou numéros de série, When l'article est contrôlé, Then il n'est pas refusé pour ce motif.
- **AC4** — Given un article refusé pour coordonnées, When le vendeur les retire et le soumet de nouveau, Then il est publié.
- **AC5** — Given la nouvelle règle, When le résultat est émis, Then il porte la version de règles v2 sans changer la version du contrat v1.

### Non fonctionnels

- **AC6 (Performance, sécurité)** — Given un texte de longueur maximale construit pour piéger l'analyse, When il est contrôlé, Then la détection se termine en moins de 50 ms (aucun moteur d'expression régulière à retour arrière).

## Sécurité

| Exigence | Comment l'US la sert | AC |
|---|---|---|
| ES-16 | Filtrage des coordonnées dans les annonces (BP4) | AC1 à AC4 |
| ES-08 | Analyse bornée, sans déni de service par l'entrée | AC6 |

Menaces : M-ANN-04, mécanisme de M-CHA-01.

## Tests attendus

| Type | Outil | Ce qui est prouvé | Fonction |
|---|---|---|---|
| Unitaire back | `go test`, table de cas positifs et négatifs, fuzzing | AC1, AC2, AC3, AC6 | Back |
| Contrat | Test de la paire avec règles v2 | AC5 | Back |
| Acceptation API et E2E | Playwright (démonstration « jean point dupont arobase… ») | AC1, AC4 | Test |

## Hors périmètre

- Filtrage du nom de boutique et du chat : V1.

## Décisions liées

- [D01 — Fonctionnalité du POC](../docs/decisions/D01-fonctionnalite-poc.md) · [D04 — Stack](../docs/decisions/D04-stack.md) (interdits de code)

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
