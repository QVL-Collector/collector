# US-003 — Portes de sécurité bloquantes dans la CI

| Champ | Valeur |
|---|---|
| Epic | E1 Chaîne de livraison |
| Statut | À faire |
| Sprint | 1 (J4) |
| Engagement | Ferme |
| Estimation | 3 points |
| Dépend de | US-002 |
| Fonction(s) | DevOps |
| Service | Plateforme |
| Critères grille | C8, C4, C6 |
| Exigences | ENF-05, ENF-06 |
| Branche | `feat/US-003-portes-securite-ci` |

## Récit

En tant que Lead Developer, je veux qu'une vulnérabilité corrigeable ou un secret détecté rende la CI rouge et bloque la fusion, afin que le POC démontre un scan bloquant (bonne pratique BP3) et que l'indicateur de vulnérabilités soit mesuré à chaque exécution.

## Critères d'acceptation

### Fonctionnels

- **AC1** — Given une PR, When la CI s'exécute, Then gitleaks analyse le diff de la PR, et sur `main` l'historique complet ; tout résultat rend `ci-ok` rouge.
- **AC2** — Given un module Go, When la CI s'exécute, Then govulncheck bloque toute vulnérabilité atteignable.
- **AC3** — Given le verrou npm, les Dockerfile, `infra/` et le chart rendu, When la CI s'exécute, Then Trivy (`fs`, `config`) bloque toute vulnérabilité ou mauvaise configuration critique ou élevée corrigeable.
- **AC4** — Given le code Go et TypeScript, When la CI s'exécute, Then Semgrep applique les règles versionnées du projet (dont `import "C"`, `regexp2`, `dangerouslySetInnerHTML`) et bloque sur sévérité « ERROR ».
- **AC5** — Given une image construite sur la PR, When Trivy `image` l'analyse, Then une vulnérabilité critique ou élevée corrigeable bloque.
- **AC6** — Given une exception justifiée, When elle est ajoutée dans `.trivyignore` avec identifiant, composant, justification, mesure compensatoire, validation et échéance d'au plus 30 jours, Then la porte passe ; une exception échue refait échouer la porte.
- **AC7** — Given chaque exécution, When elle se termine, Then les rapports SARIF sont publiés dans l'onglet sécurité et les compteurs de vulnérabilités ouvertes et d'exceptions sont produits pour l'indicateur I6.

### Non fonctionnels

- **AC8 (Sécurité, démonstration BP3)** — Given une branche de démonstration contenant une dépendance vulnérable connue et un secret fictif, When sa PR est ouverte, Then la CI est rouge et la fusion refusée, alors que `main` reste verte.

## Sécurité

| Exigence | Comment l'US la sert | AC |
|---|---|---|
| ES-05 | Analyse des dépendances, des images, du code et de la configuration en code | AC2 à AC6, AC8 |
| ES-06 | Détection de secrets sur le diff et l'historique | AC1, AC8 |

Mesures : MP3, MP7, MP9 ; compteurs publics seulement (MQ1 à MQ3, MQ5). Seuils et délais : [devsecops.md](../docs/processus/devsecops.md) § 3.

## Tests attendus

| Type | Outil | Ce qui est prouvé | Fonction |
|---|---|---|---|
| Sécurité statique | gitleaks, govulncheck, Trivy, Semgrep | AC1 à AC5, AC7 | DevOps |
| Sécurité négative | Branche de démonstration BP3 | AC8 | Test |
| Contrôle de pipeline | Exception valide puis échue | AC6 | Test |

## Hors périmètre

- Contrôle du realm Keycloak (US-008), validation OpenAPI (US-009), `helm lint` (US-007) : ajoutés avec l'élément qu'ils vérifient.
- ZAP baseline : exécuté en fin de sprint, hors porte bloquante.

## Décisions liées

- [D10 — Forge git, CI/CD](../docs/decisions/D10-forge-ci-cd.md)
- [D12 — Indicateurs qualité](../docs/decisions/D12-indicateurs-qualite.md) (I6)

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
