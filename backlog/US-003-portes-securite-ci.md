# US-003 — Portes de sécurité bloquantes dans la CI

| Champ | Valeur |
|---|---|
| Epic | E1 Chaîne de livraison |
| Statut | En cours |
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

### Réalisation et preuves restantes

Réalisé hors ligne le 2026-10-26, sur la branche de US-002, en attente de la publication du dépôt (US-001) : jobs `secrets`, `sast`, `config`, govulncheck et Trivy `image` dans le job Go, job `securite` (compteurs I6) et exécution quotidienne dans [ci.yml](../.github/workflows/ci.yml) ; scripts [secrets.sh](../tools/ci/secrets.sh), [govulncheck.sh](../tools/ci/govulncheck.sh), [semgrep.sh](../tools/ci/semgrep.sh), [trivy.sh](../tools/ci/trivy.sh), [verifier-exceptions.sh](../tools/ci/verifier-exceptions.sh) ; règles [Semgrep](../tools/ci/semgrep/regles.yml) et leurs cas d'essai ; [.trivyignore](../.trivyignore), [.semgrepignore](../.semgrepignore).

Prouvé sur le poste, sur une copie du dépôt et des cas d'essai jetables (rien de vulnérable n'est versionné) :

| AC | Preuve locale | Reste à prouver sur GitHub |
|---|---|---|
| AC1 | gitleaks : historique propre vert ; commit d'un jeton fictif rouge en mode PR (diff) ; `main` reste verte quand une autre branche contient le secret | Exécution PR et `main`, SARIF dans l'onglet sécurité |
| AC2 | Module d'essai appelant `language.ParseAcceptLanguage` de `golang.org/x/text` v0.3.7 : 1 vulnérabilité atteignable, rouge ; version corrigée : vert | Exécution sur un service réel |
| AC3 | Verrou npm d'essai (lodash 4.17.20) : 2 élevées corrigeables, rouge ; Dockerfile sans `USER` : rouge ; dépôt actuel : vert | Exécution sur `web/` et `infra/` réels ; chart rendu à vérifier avec US-007 |
| AC4 | `semgrep --test` vert sur les cas d'essai ; `import "C"` et `dangerouslySetInnerHTML` (même avec `nosemgrep`) : 2 constats, rouge | Exécution en PR |
| AC5 | Image d'essai `alpine:3.18.0` : 9 critiques ou élevées corrigeables, rouge | Exécution sur l'image d'un service |
| AC6 | Exceptions complètes à 20 jours : porte verte ; mêmes exceptions échues : contrôle rouge et Trivy rouge ; champ manquant ou échéance > 30 jours : rouge | Démonstration en PR (exception valide puis échue) |
| AC7 | Compteurs JSON produits par chaque porte ; SARIF générés | SARIF visibles dans l'onglet sécurité ; résumé et artefact `i6` du job `securite` |
| AC8 | Composants prouvés un par un ci-dessus | Branche BP3 (dépendance vulnérable et secret fictif) : PR rouge et non fusionnable, `main` verte |

