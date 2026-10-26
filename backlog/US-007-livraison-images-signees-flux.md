# US-007 — Livraison continue : images signées et déploiement par Flux

| Champ | Valeur |
|---|---|
| Epic | E1 Chaîne de livraison |
| Statut | À faire |
| Sprint | 1 (J4) |
| Engagement | Ferme (ordre de coupe S1 n° 1 : repli sur le déploiement depuis le poste, signature conservée) |
| Estimation | 5 points |
| Dépend de | US-002, US-004 |
| Fonction(s) | DevOps |
| Service | Plateforme |
| Critères grille | C4, C8 |
| Exigences | ENF-22, ENF-29 |
| Branche | `feat/US-007-livraison-images-signees-flux` |

## Récit

En tant que Lead Developer, je veux qu'une fusion sur `main` produise une image unique, signée, publiée et déployée par empreinte sans que la CI ne touche AWS, afin de démontrer une livraison continue conforme au schéma CI/CD et réversible.

## Critères d'acceptation

### Fonctionnels

- **AC1** — Given une fusion sur `main`, When la CI de `main` s'exécute, Then chaque image modifiée est construite une seule fois, étiquetée `sha-<commit>`, analysée par Trivy, accompagnée d'un SBOM CycloneDX, poussée sur GHCR et signée par cosign sans clé ; `cosign verify` réussit dans le même job.
- **AC2** — Given les images publiées, When le job de publication se termine, Then leurs empreintes `sha256` sont écrites sur la branche `env/demo`, seul contenu de cette branche, par ce seul job.
- **AC3** — Given un `up`, When les Secrets sont créés, Then `up.sh` installe Flux (source-controller et helm-controller seulement, limités au namespace de l'application), qui applique la `HelmRelease` depuis `env/demo` ; `up.sh` attend la réconciliation puis lance les tests de fumée.
- **AC4** — Given une empreinte d'image non signée dans `env/demo`, When `up.sh` vérifie les signatures, Then le déploiement est refusé.
- **AC5** — Given une nouvelle version en échec à la réconciliation, When Flux applique la `HelmRelease`, Then la version précédente est restaurée automatiquement (retour arrière).
- **AC6** — Given Flux indisponible, When `tools/up.sh --sans-flux` s'exécute, Then le même chart est installé par `helm upgrade --install` avec les empreintes lues dans le même fichier de `env/demo` ; ce chemin est joué au moins une fois par sprint sur k3d.
- **AC7** — Given les paquets GHCR, When ils sont tirés depuis la NAT sans jeton, Then le tirage réussit (paquets publics).

### Non fonctionnels

- **AC8 (Sécurité)** — Given la CI, When ses secrets et ses journaux sont relus, Then elle ne contient aucun identifiant AWS ; seul le job de publication sur `main` a les droits d'écriture sur le registre et sur `env/demo`.
- **AC9 (Sécurité)** — Given le compte de service de Flux, When `kubectl auth can-i --list` est exécuté, Then ses droits se limitent aux namespaces applicatifs.

## Sécurité

| Exigence | Comment l'US la sert | AC |
|---|---|---|
| ES-32 | Image construite une fois, signée, vérifiée, promue par empreinte ; droits de publication restreints | AC1, AC2, AC4, AC8, AC9 |
| ES-05 | Analyse d'image et SBOM avant publication | AC1 |
| ES-06 | Aucun identifiant AWS dans la CI | AC8 |

Mesures : MS1, MP5, MP6, MP10, MI5.

## Tests attendus

| Type | Outil | Ce qui est prouvé | Fonction |
|---|---|---|---|
| Contrôle de pipeline | Fusion de démonstration, lecture de GHCR et de `env/demo` | AC1, AC2, AC7 | DevOps |
| Après déploiement | `flux get`, tests de fumée, retour arrière provoqué | AC3, AC5 | DevOps, Test |
| Sécurité négative | Empreinte non signée, `kubectl auth can-i` | AC4, AC9 | Test |
| Filet | `up.sh --sans-flux` sur k3d | AC6 | DevOps |

## Hors périmètre

- Revérification continue des signatures dans le cluster : V1 (écart documenté, architecture § 9.8).
- Suspension de Flux pendant la soutenance : procédure de `presentation/demo-script.md`.

## Décisions liées

- [D10 — Forge git, CI/CD](../docs/decisions/D10-forge-ci-cd.md) (F1, R1, P2 avec P1 en filet)
- [D11 — IaC, allumage et extinction](../docs/decisions/D11-iac-allumage-cout.md)

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
