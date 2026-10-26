# D10 — Forge git, CI/CD, registre, déploiement, portes DevSecOps

| Champ | Valeur |
|---|---|
| Statut | Décidée |
| Date | 2026-10-26 |
| Décideur | Martin |
| Dépend de | [D03](D03-hebergement-orchestration.md), [D04](D04-stack.md) |
| Impacte | D11 (allumage, `up.sh`), D12 (mesure des indicateurs), D13 (place des tests dans le pipeline), D14 (expérimentation), [PILOTAGE.md](../PILOTAGE.md) § 6, [exigences-securite.md](../securite/exigences-securite.md) (ES-05, ES-06, ES-32), README, `.github/` ou `.gitlab-ci.yml`, `deploy/` |
| Critères grille | C4, C7, C8, C6 |

## Contexte et besoin

D10 fixe où vit le code, comment il est vérifié, où sont rangées les images et comment elles arrivent dans le cluster EKS de D03.

- **Exigences** :
  - consigne : V1 « avec un pipeline CI/CD et tests automatisés », « scan de vulnérabilités » (M12, M14) ; au moins deux types de tests dans le pipeline (M14, C7) ; schéma détaillé du processus CI/CD (M4) ; cycle DevSecOps avec les mesures de chaque étape (M3) ;
  - C4 : « Le POC intègre la plateforme de livraison continue conformément au processus décrit » ; écart à éviter : « plateforme CI/CD du POC limitée à une seule étape ». Pas de pénalité si l'étape de déploiement ne peut pas être mise en œuvre ([contexte-et-grille.md](../00-cadrage/contexte-et-grille.md) § 4) ;
  - ES-05 : dépendances, images, code, configuration décrite en code et secrets analysés ; une vulnérabilité critique ou élevée **corrigeable** bloque la fusion ; exception justifiée et datée ; rapports conservés ; analyse dynamique non intrusive en fin de sprint. BP3 : une branche de démonstration rend le pipeline rouge ;
  - ES-06 : aucun secret dans le dépôt, les images, les journaux ni les sorties du pipeline ;
  - ES-32 et ENF-29 : branche principale protégée, CI verte avant fusion, droits du pipeline limités, artefact construit une fois, identifié par son empreinte, promu tel quel ;
  - [PILOTAGE.md](../PILOTAGE.md) § 6 : monorepo, `main` protégée, une branche `feat/US-xxx-slug` par US, fusion après la phrase de validation de Martin.
- **Contrôles déjà imposés par d'autres décisions** :
  - D04 : `go vet`, golangci-lint (gosec inclus), `go test -race -cover`, govulncheck, image minimale non root analysée ; front : vérification des types, ESLint (`jsx-a11y`, `react/no-danger`), Vitest, Playwright avec axe-core, audit npm ; OpenAPI par service vérifiée en CI ;
  - D06 (MA1, MA14) : realm JSON contrôlé en CI (aucun joker, flux implicite et « mot de passe » désactivés, aucun secret) ; image Keycloak optimisée construite en CI ; import testé à chaque montée de version ;
  - D07 (MD5) : « aucun import de paquet de format hors `image/jpeg` et `image/png` », contrôlé par `go list -deps` ; Job de migration en crochet Helm `pre-install` et `pre-upgrade` ;
  - D03 (MS1, décidée) : API du cluster limitée à l'adresse du poste ; « la CI hébergée ne fait que construire et pousser » ; MS6 : secrets créés depuis Secrets Manager, jamais en valeurs Helm ; MS9 : charts et images épinglés et analysés.
- **Contraintes du Learner Lab** ([environnement-learner-lab.md](../00-cadrage/environnement-learner-lab.md)) :
  - **pas de fédération OIDC** de la forge vers AWS ; identifiants de session d'environ 4 h, qui meurent avec la session (`voc-cancel-cred`) ;
  - **CodePipeline et CodeBuild indisponibles** ;
  - ECR : `LabRole` en **lecture seule** ; seule la session de Martin peut pousser. Les nœuds tirent d'ECR par leur rôle ;
  - le cluster est recréé à chaque session (D03, D11) : tout ce qui vit dans le cluster est réinstallé à chaque `up`.

## Options

Cinq sous-questions, traitées séparément ; la synthèse les assemble.

### 1. Forge

#### Option F1 — GitHub, dépôt public, GitHub Actions et GHCR

- **Description** : dépôt public, pipeline GitHub Actions, images dans GitHub Container Registry (GHCR).
- **Avantages** :
  - minutes d'exécuteurs hébergés gratuites et sans plafond pour un dépôt public [?, à vérifier à la création] ;
  - images publiques sur GHCR : le cluster les tire **sans jeton** ;
  - outils gratuits en public : analyse CodeQL, détection de secrets à la poussée, alertes de dépendances ; SonarQube Cloud gratuit en public, utile à D12 [?] ;
  - majorité des retours d'expérience Learner Lab sur GitHub ; signature sans clé (Sigstore) native depuis Actions.
- **Inconvénients** :
  - code, workflows, realm JSON et chart visibles de tous : aucun secret n'y est autorisé de toute façon (ES-06), mais la surface de reconnaissance augmente ;
  - une contribution externe (fork) peut ouvrir une PR : elle s'exécute sans secret, à condition de proscrire `pull_request_target`.
- **Coût AWS et effort** : 0 $ ; mise en place du dépôt et des règles de protection : 0,25 j.
- **Complexité ajoutée** : faible.
- **Couverture grille** : C4, C8 (rapports SARIF visibles dans l'onglet sécurité).

#### Option F2 — GitHub, dépôt privé

- **Description** : idem F1, dépôt privé.
- **Avantages** : code non exposé ; proche d'un contexte d'entreprise.
- **Inconvénients** :
  - 2 000 minutes par mois sur l'offre gratuite [?]. Un pipeline complet des quatre services et du front consomme de l'ordre de 25 à 40 minutes facturées [estimation] : 50 à 80 exécutions par mois, juste pour un sprint actif ;
  - GHCR privé : environ 500 Mo de stockage gratuit [?]. L'image Keycloak optimisée (D06) pèse à elle seule plusieurs centaines de Mo : purge obligatoire ;
  - jeton de lecture du registre à poser en Secret dans le cluster, à chaque `up` ; CodeQL et la détection de secrets à la poussée ne sont pas gratuits en privé.
- **Coût AWS et effort** : 0 $ tant que le quota tient ; 0,5 j de plus (jetons, purge).
- **Complexité ajoutée** : moyenne.
- **Couverture grille** : identique à F1.

#### Option F3 — GitLab.com, offre gratuite

- **Description** : dépôt GitLab.com, GitLab CI, registre intégré.
- **Avantages** : CI, registre et modèles d'analyse (SAST, secrets, conteneurs) intégrés ; compétence déjà pratiquée par Martin sur d'autres projets.
- **Inconvénients** :
  - environ 400 minutes de calcul par mois sur l'offre gratuite, y compris en public [?] : insuffisant sans exécuteur auto-hébergé ;
  - le rapport de vulnérabilités consolidé et le widget de MR sont réservés à l'offre Ultimate : les analyses tournent, la restitution est plus pauvre ;
  - quota de stockage du namespace [?].
- **Coût AWS et effort** : 0 $ ; exécuteur sur le poste si le quota est dépassé (+0,5 j, et la CI dépend du poste allumé).
- **Complexité ajoutée** : moyenne.
- **Couverture grille** : C4, C8.

### 2. Registre d'images

#### Option R1 — Registre de la forge seul (GHCR)

- **Description** : la CI construit une fois, pousse vers GHCR une image étiquetée `sha-<commit>` ; le cluster la tire par son empreinte (`image@sha256:…`) via la NAT.
- **Avantages** : aucun identifiant AWS dans la CI ; la CI reste verte hors session ; un seul registre ; ES-32 et ENF-29 tenus par l'empreinte.
- **Inconvénients** : GHCR n'interdit pas la réécriture d'une étiquette : l'immuabilité vient du déploiement par empreinte, pas du registre ; trafic de tirage par la NAT.
- **Coût AWS et effort** : environ 1,5 à 3 Go tirés par session [estimation] à 0,045 $/Go, soit environ 0,1 $ par session et 2 à 3 $ sur le projet ; effort nul en plus du pipeline.
- **Complexité ajoutée** : nulle.
- **Couverture grille** : C4, ES-32.

#### Option R2 — ECR seul, poussée depuis la CI

- **Description** : identifiants de session recopiés en secrets de la forge à chaque session ; la CI pousse vers ECR.
- **Avantages** : tirage par le rôle du nœud, sans sortie Internet ; étiquettes immuables natives.
- **Inconvénients** : hors session, la poussée échoue (`voc-cancel-cred`) et la CI devient rouge pour une raison étrangère au code ; identifiants aux droits proches de la console stockés dans la forge (M-LIV-02).
- **Coût AWS et effort** : 0,10 $/Go-mois ; script de recopie des identifiants (0,25 j) et une manipulation à chaque session.
- **Complexité ajoutée** : faible en code, forte en exploitation.
- **Couverture grille** : C4, mais avec une CI dépendante de la session.

#### Option R3 — GHCR comme source, copie vers ECR par `up.sh`

- **Description** : R1, puis `up.sh` copie les empreintes à déployer vers ECR avec la session de Martin (`crane copy` ou `skopeo copy`, qui conservent l'empreinte).
- **Avantages** : CI indépendante d'AWS ; tirage par le rôle du nœud ; même artefact (même empreinte).
- **Inconvénients** : une étape et un outil de plus à chaque allumage ; deux registres à expliquer.
- **Coût AWS et effort** : quelques centimes de stockage ; 0,25 j.
- **Complexité ajoutée** : faible.
- **Couverture grille** : identique à R1.

### 3. Déploiement vers EKS

#### Option P1 — Depuis le poste : `up.sh` lance `helm upgrade`

- **Description** : la CI publie les empreintes ; `up.sh`, avec la session de Martin, crée les Secrets (MS6), puis lance `helm upgrade --install` avec les empreintes de `main`.
- **Avantages** : le plus simple ; compatible avec MS1 ; déjà prévu comme filet commun par D03.
- **Inconvénients** : le déploiement sort de la plateforme CI/CD : c'est un geste manuel, ce que C4 regarde (« plateforme limitée à une seule étape ») ; aucun alignement automatique après une fusion pendant une session.
- **Coût AWS et effort** : 0 $ ; inclus dans D11.
- **Complexité ajoutée** : nulle.
- **Couverture grille** : C4 partielle (intégration continue complète, livraison manuelle).

#### Option P2 — GitOps : Flux dans le cluster

- **Description** : `up.sh` crée les Secrets, installe Flux (`flux install`, sans `flux bootstrap`, qui demanderait un jeton d'écriture sur le dépôt), puis applique une source et une `HelmRelease`. Flux tire l'état voulu depuis la forge et l'applique ; il le réapplique à chaque fusion tant que la session dure.
  - L'état voulu de l'environnement de démonstration fixe les empreintes d'images. Deux façons de le publier sans écrire sur `main` : un fichier `deploy/demo/` sur une branche `env/demo` mise à jour par la CI après la construction sur `main` ; ou un artefact OCI (chart et valeurs) poussé sur GHCR, suivi par Flux.
- **Avantages** :
  - **aucun identifiant AWS dans la CI** et API du cluster fermée à Internet : seul modèle de livraison automatique compatible avec MS1 et l'absence d'OIDC ;
  - livraison continue réelle et visible : fusion, construction, analyse, publication, réconciliation ; argument direct pour C4 ;
  - l'état du cluster est décrit et versionné ; la dérive manuelle est corrigée ;
  - les crochets Helm (Job de migration, D07) fonctionnent avec le contrôleur Helm de Flux.
- **Inconvénients** :
  - un composant de plus, réinstallé à chaque session : 2 à 3 min d'allumage, 2 à 3 pods sur un cluster déjà dense (25 à 31 pods pour 34 places, D03) ;
  - par défaut, le contrôleur Helm de Flux a des droits d'administration du cluster : à restreindre ;
  - un mécanisme de publication des empreintes à choisir (branche ou artefact OCI).
- **Coût AWS et effort** : 0 $ ; 0,5 à 1 j (estimation de D03).
- **Complexité ajoutée** : moyenne.
- **Couverture grille** : C4 complète, C1 (Flux peut entrer dans l'expérimentation), C8 (aucun secret de déploiement hors du cluster).

#### Options écartées d'emblée

- **CI qui pousse le déploiement avec des identifiants recopiés** : contraire à MS1, décidée avec D03 (l'API du cluster devrait s'ouvrir aux plages d'adresses des exécuteurs hébergés, donc à presque tout Internet) ; CI rouge hors session.
- **Exécuteur auto-hébergé sur EC2 avec `LabInstanceProfile`** : il exécuterait le code des PR avec les droits de `LabRole` ; il s'arrête en fin de session ; aucun retour d'expérience.
- **Argo CD** : même modèle que Flux, plus lourd (interface, serveur, base Redis) ; l'interface n'apporte rien que `flux get` et les événements Kubernetes ne montrent déjà.

### 4. Portes du pipeline

Proposition commune à toutes les options de forge. « Bloquant » = le pipeline échoue et la fusion est refusée.

| Étape | Outil | Déclencheur | Règle | Exigence |
|---|---|---|---|---|
| Secrets | gitleaks (diff de la PR ; historique complet sur `main` et avant l'audit) | PR, `main` | Bloquant sur tout résultat | ES-06, ENF-06 |
| Lint et types | `go vet`, golangci-lint (gosec, exhaustivité) ; `tsc --noEmit`, ESLint (`jsx-a11y`, `react/no-danger`) | PR | Bloquant | D04, ES-08 |
| Contrôles propres | realm JSON (MA1, script `jq`) ; `go list -deps` (MD5) ; validation de chaque OpenAPI ; `helm lint` et rendu du chart | PR | Bloquant | D06, D07, D04 |
| Tests unitaires | `go test -race -cover` ; Vitest | PR | Bloquant ; couverture publiée, seuil à fixer (point soumis) | C7, C6 |
| Tests d'intégration | testcontainers-go (PostgreSQL, RabbitMQ) ; Keycloak de docker compose pour la vérification des jetons (D06) | PR | Bloquant | C7, D06 |
| Dépendances | govulncheck (Go, vulnérabilités atteignables) ; Trivy `fs` sur le verrou npm (remplace `npm audit`, voir ci-dessous) | PR, quotidien sur `main` | Bloquant sur critique ou élevée corrigeable | ES-05, ENF-05 |
| Analyse statique sécurité | Semgrep (règles Go, TypeScript, React, Dockerfile) | PR | Bloquant sur sévérité « ERROR » | ES-05, ES-08 |
| Configuration en code | Trivy `config` sur `infra/` (Terraform), le chart rendu, les Dockerfile | PR | Bloquant sur critique ou élevée | ES-05, ES-07, MS9 |
| Construction | `docker buildx`, image statique non root, sans shell ; une image par service, plus le front et Keycloak | PR (sans poussée), `main` (poussée) | Échec = bloquant | D04, ES-32 |
| Analyse d'image | Trivy `image`, avant la poussée | PR, `main` | Bloquant sur critique ou élevée corrigeable | ES-05 |
| SBOM | Trivy (CycloneDX), joint à l'image | `main` | Non bloquant, archivé | ES-05, M-LIV-01 |
| Signature | cosign sans clé (identité OIDC de la forge vers Sigstore, sans lien avec AWS) | `main` | Optionnelle (point soumis) | ES-32 |
| Acceptation | Playwright et axe-core, tests d'API, sur la pile docker compose | `main`, et PR qui touchent le front ou un contrat | Bloquant ; périmètre exact en D13 | C7, M10 |
| DAST | ZAP baseline : sur la pile compose (quotidien) ; sur l'URL publique de démonstration, en déclenchement manuel | Quotidien ; fin de sprint | Non bloquant pour la fusion ; résultats triés dans l'audit de fin de sprint | ES-05, ES-01, ES-07 |
| Après déploiement | Tests de fumée et X17 : 200 sur la découverte OIDC, 404 sur `/admin/`, `/realms/master/` ; port 9000 injoignable | Fin de chaque `up` | Échec signalé par `up.sh` | MA7, ES-07 |

**Choix d'outillage, au plus simple.**
- **Trivy couvre trois analyses** (dépendances npm, images, configuration) avec un seul fichier d'exceptions (`.trivyignore`, chaque entrée datée et justifiée, comme ES-05 l'exige) et produit le SBOM : Syft et `npm audit` deviennent redondants.
- **govulncheck est gardé** pour Go : il ne signale que les vulnérabilités atteignables, donc moins de faux positifs que Trivy sur `go.mod`.
- **gitleaks est gardé** : Trivy ne lit pas l'historique git, ES-06 l'exige.
- **Semgrep plutôt que CodeQL** : indépendant de la forge, rapide ; gosec ne couvre que Go. Sur F1, CodeQL est une alternative gratuite, plus lente.
- **Rapports** : format SARIF publié dans la forge, et artefacts d'exécution conservés 90 jours ; relevé copié dans `docs/mesures/` en fin de sprint (C6, D12).

**Protection de la chaîne (ES-32).**
- `main` protégée : PR obligatoire, pas de poussée forcée, une vérification requise unique (`ci-ok`) qui agrège les jobs.
- Monorepo : un seul workflow, des jobs par service déclenchés selon les chemins modifiés (D02). Un workflow ignoré par filtre de chemin bloquerait une vérification requise : d'où le job agrégateur.
- Droits par défaut en lecture ; écriture du registre (et jeton OIDC pour cosign) réservée au job de publication sur `main` ; actions tierces épinglées par empreinte de commit ; mises à jour des actions proposées automatiquement (Dependabot).
- Seul développeur : une approbation par un tiers est impossible ; la revue reste celle du processus ([PILOTAGE.md](../PILOTAGE.md) § 5), la règle technique est la CI verte.

### 5. Environnements et promotion

| Environnement | Rôle | Déploiement | Artefact |
|---|---|---|---|
| Poste, docker compose | Développement, tests manuels | `docker compose up` | Images construites localement |
| CI, docker compose éphémère | Intégration, acceptation, ZAP | Par le pipeline | Images de la PR, non poussées |
| Poste, k3d ou minikube | Validation du chart et de Flux, expérimentation locale (D03, C1) | Même chemin que AWS | Empreintes publiées sur GHCR |
| AWS, démonstration | Environnement managé, recréé à chaque session | Option P1 ou P2 | Mêmes empreintes |

- **Promotion** : aucune reconstruction. L'image est construite une fois sur `main`, étiquetée `sha-<commit>`, et déployée par empreinte. Promouvoir vers AWS revient à changer l'empreinte de l'état voulu (P2) ou de `up.sh` (P1).
- Pas d'environnement de préproduction séparé : il doublerait le coût de session sans besoin présent. La V1 en ajouterait un.
- Images tierces (RabbitMQ, PostgreSQL local, Keycloak de base) épinglées par empreinte, tirées de registres sans limite stricte de débit (quay.io, `public.ecr.aws`) plutôt que de Docker Hub sans authentification depuis l'adresse de la NAT [?].

### Synthèse

| Sous-question | Options | Plus simple | Plus démonstrative |
|---|---|---|---|
| Forge | F1 GitHub public, F2 GitHub privé, F3 GitLab.com | F1 | F1 |
| Registre | R1 GHCR, R2 ECR poussé par la CI, R3 GHCR copié vers ECR | R1 | R1 ou R3 |
| Déploiement | P1 poste, P2 Flux | P1 | P2 |
| Portes | Tableau ci-dessus | Sans signature ni DAST distant | Toutes, signature vérifiée |

Effort total estimé : pipeline et portes 2,5 à 3,5 j ; exceptions et réglage des seuils 0,5 à 1 j ; Flux 0,5 à 1 j ; signature vérifiée 0,25 à 0,5 j. Coût AWS : environ 0,1 $ par session (tirage des images), aucun service AWS de CI.

## Recommandation

**F1 (GitHub public) + R1 (GHCR, déploiement par empreinte) + P2 (Flux), avec P1 conservé comme filet ; portes du tableau, signature en option.**

1. **Le dépôt public supprime trois problèmes d'un coup** : quota de minutes, quota de stockage du registre, et jetons de lecture à poser dans le cluster (registre et dépôt). C'est la forme la plus simple qui tient la démonstration ; le prix est l'exposition du code, sans secret par construction (ES-06, gitleaks).
2. **GHCR seul** : la CI ne connaît pas AWS et reste verte hors session ; l'empreinte garantit le « même artefact » (ENF-29, ES-32). R3 n'apporte qu'un tirage sans NAT, pour environ 0,1 $ par session : pas un besoin présent.
3. **Flux plutôt que le poste**, au nom de la démonstration complète que Martin vise : c'est la seule livraison automatique compatible avec MS1 et l'absence d'OIDC, et elle répond à l'écart de C4. Le poste reste le filet (P1) si Flux échoue en expérimentation ou gêne la densité du cluster.
4. **Publication des empreintes par une branche `env/demo`** écrite par la CI : plus facile à montrer et à expliquer au jury qu'un artefact OCI ; `main` n'est jamais écrite par un robot.
5. **Portes au plus juste** : un outil par risque (Trivy pour trois analyses, govulncheck, gitleaks, Semgrep) ; DAST hors chemin de fusion, car l'environnement AWS n'est pas toujours allumé.
6. **Signature optionnelle** : elle n'a de valeur que vérifiée. Si Martin la retient, la vérification la plus simple est `cosign verify` dans `up.sh` avant l'application ; un contrôleur d'admission serait un composant de plus, non justifié dans le POC.

### Points soumis aux avis

- **Sécurité** :
  - dépôt public : exposition des workflows, du realm, du chart et de l'architecture ; acceptable pour un POC sur données fictives ?
  - droits de Flux : restreindre le contrôleur Helm aux namespaces applicatifs (compte de service dédié) ;
  - écriture de la branche `env/demo` par la CI : règle limitant cette écriture au seul job de publication ;
  - seuils bloquants : « critique ou élevée corrigeable » partout, ou plus strict sur les images ;
  - signature : retenue et vérifiée, ou documentée pour la V1 ;
  - ZAP sur l'URL publique : périmètre et fréquence.
- **Faisabilité** :
  - densité : Flux ajoute 2 à 3 pods sur un cluster de 34 places ; à mesurer en expérimentation ;
  - ordre d'allumage : Secrets avant la réconciliation, Job de migration avant les services ;
  - durée cible du pipeline de PR (moins de 15 min ?) et caches Go et npm ;
  - seuil de couverture : bloquant dès le sprint 1, ou fixé après une première mesure (D12) ;
  - testcontainers et docker compose sur les exécuteurs hébergés : temps de démarrage de Keycloak.
- **Martin** : forge (public ou privé), Flux ou poste, signature, place des tests d'acceptation (chaque PR ou `main`).

## Avis sécurité

Fonction : Sécurité — 2026-10-26. Références : [exigences-securite.md](../securite/exigences-securite.md) (FC9, ES-05, ES-06, ES-32 ; M-LIV-01, 02, 05) ; [devsecops.md](../processus/devsecops.md) (seuils, cosign, ZAP) ; avis sécurité de [D03](D03-hebergement-orchestration.md) (MS1, MS6, MS9). Avis sur dossier ; les points [?] sont à vérifier à la création du dépôt.

**Synthèse.** Avis favorable à F1 + R1 + P2 (P1 en filet), sous les mesures MP1 à MP10. Le modèle « pull » est le bon : la CI ne détient aucun identifiant AWS ni du cluster. Le risque se déplace vers trois points, qui deviennent les vraies frontières de confiance de FC9 : le compte GitHub de Martin, le jeton du job de publication et les droits de Flux. Qui écrit `env/demo` déploie dans le cluster, avec les droits de Flux.

**1. Points soumis**

- **Dépôt public** : acceptable pour le POC sur données fictives. Aucun contrôle de sécurité ne repose sur le secret du code, du chart ou du realm ; la topologie exposée (services, ports, politiques réseau) n'ouvre rien que ES-07 ne ferme déjà. Trois conditions :
  - ce qui ne doit jamais entrer, au-delà des secrets : état Terraform (`*.tfstate`, qui contient des mots de passe), fichiers `.env`, exports de realm avec secrets de clients, jeux de données réels ; `.gitignore` et gitleaks (historique complet) en garantie ; détection de secrets à la poussée de GitHub activée (gratuite en public) ;
  - images publiques sur GHCR : elles sont lisibles de tous ; ES-06 s'y applique (Trivy `image` détecte les secrets embarqués) ; l'image Keycloak ne contient aucun secret de client ;
  - conformité : aucune donnée personnelle dans le dépôt, fixtures fictives ; pas d'enjeu RGPD tant que cela tient. En V1, dépôt privé (déjà prévu).
- **Contributions externes** : `pull_request_target` et `workflow_run` sur artefact de PR proscrits ; approbation exigée avant toute exécution pour les contributeurs externes ; aucune valeur d'événement (`title`, `body`, `head_ref`) interpolée dans un `run:`, passage par variable d'environnement (injection de commande dans le workflow).
- **Droits de Flux** : `flux install` lie par défaut ses contrôleurs à `cluster-admin`. À restreindre, c'est la mesure principale de P2 :
  - composants limités à `source-controller` et `helm-controller` (moins de pods, utile aussi à la densité) ;
  - `HelmRelease` avec `serviceAccountName` d'un compte dédié, lié à un `Role` dans le seul namespace applicatif ; `--default-service-account` sur le contrôleur Helm, pour qu'une ressource sans compte ne retombe jamais sur `cluster-admin` ; `--no-cross-namespace-refs` ;
  - le chart ne crée aucune ressource de cluster (namespace, CRD, `ClusterRole`) ni de RBAC : `up.sh` les pose avec la session de Martin. Résiduel accepté : le compte dédié lit les Secrets du namespace (Helm y range ses versions).
- **Écriture de `env/demo` par la CI** : favorable, sous conditions. `contents: write` accordé au seul job de publication, sur `main`, après succès de toutes les portes ; règles de branche sur `env/demo` : poussée forcée et suppression interdites, mise à jour réservée à GitHub Actions [?] ; la CI n'y écrit que des empreintes, jamais de valeurs libres. Le `GITHUB_TOKEN` ne peut pas modifier `.github/workflows/`, ce qui borne une compromission du job. Alternative plus sûre, au même coût : artefact OCI signé par cosign et vérifié par Flux (`OCIRepository`, `spec.verify`), qui rend inopérant un état voulu non signé.
- **Protection de `main`** : ensemble de règles sans aucune exception de contournement, administrateur compris (par défaut, un administrateur seul passe outre) ; PR obligatoire, `ci-ok` requise, poussée forcée et suppression interdites. Piège : un job agrégateur ignoré parce qu'un job amont a échoué est compté comme réussi ; `ci-ok` tourne en `if: always()` et échoue si un résultat amont vaut `failure` ou `cancelled`. Revue : zéro approbation technique (seul développeur), revue tracée dans la PR selon PILOTAGE § 5.
- **Actions et `GITHUB_TOKEN`** : épinglage par empreinte complète (40 caractères), version en commentaire, Dependabot sur l'écosystème `github-actions`. Droits par défaut du dépôt en lecture seule ; `permissions: {}` au niveau du workflow, accordées job par job (`security-events: write` pour SARIF, `packages: write` et `id-token: write` pour la publication) ; `persist-credentials: false` sur `actions/checkout` hors job qui pousse. Filtre de chemins par un script `git diff` plutôt qu'une action tierce : `tj-actions/changed-files`, courante pour cet usage, a été compromise en mars 2025 et a vidé des secrets dans les journaux de dépôts publics. Analyse des workflows ajoutée aux portes (actionlint ou zizmor, bloquant).
- **Seuils** : « critique ou élevée corrigeable » partout, conforme à ES-05, sans durcissement sur les images : bloquer le non corrigeable bloquerait pour une raison étrangère au code. Compléments : gitleaks bloquant sur tout résultat ; govulncheck bloquant sur toute vulnérabilité atteignable avec correctif (il ne donne pas de sévérité) ; chaque entrée de `.trivyignore` porte une date d'expiration (`exp:`), relue en fin de sprint ; un critique non corrigeable est inscrit au registre avec une date de revue. Analyse quotidienne sur `main` : un nouveau résultat ouvre un constat, sans bloquer les PR sans rapport.
- **Signature** : retenue, sans contrôleur d'admission. L'empreinte garantit l'intégrité d'une image, pas son origine : une empreinte d'image étrangère écrite dans `env/demo` serait déployée. Vérification utile seulement avec l'identité attendue : `--certificate-identity` limitée au workflow de publication sur `refs/heads/main`, émetteur `https://token.actions.githubusercontent.com`. Lieu : `up.sh` (P1 et premier déploiement), ou Flux si l'artefact OCI est retenu ; sinon, documentée en V1, en assumant le résiduel.
- **ZAP** : baseline seule (passive, non intrusive), jamais le scan actif sur AWS. Sur l'URL publique : déclenchement manuel, une fois par sprint avant l'audit, sur le seul domaine de démonstration (D08), session en cours. Règles ignorées versionnées et justifiées ; un constat élevé est porté au registre et corrigé avant la fin du sprint suivant. Non bloquant pour la fusion : accepté.

**2. Points non soumis**

- **Compte GitHub de Martin** : point unique de défaillance (code, `main`, `env/demo`, donc le cluster). Double facteur obligatoire, clé d'accès ou application, jamais SMS ; jetons personnels à portée fine et datés.
- **Journaux du pipeline publics** : tout ce que la CI affiche est lisible de tous ; aucun `set -x` sur une étape manipulant un jeton, masquage des valeurs dérivées (`::add-mask::`).
- **Images tierces** : épinglage par empreinte (§ 5) et Dependabot sur Dockerfile et docker compose, sinon l'épinglage fige les vulnérabilités (M-LIV-01).

**3. Mesures minimales, si la recommandation est retenue**

| Id | Mesure | Exigences, menaces | Preuve |
|---|---|---|---|
| MP1 | Ensemble de règles sur `main` sans contournement, `ci-ok` requise en `if: always()`, poussée forcée et suppression interdites | ES-32, M-LIV-05 | Export des règles ; PR de démonstration rouge non fusionnable |
| MP2 | Droits du dépôt en lecture seule, `permissions` par job, `persist-credentials: false`, aucun `pull_request_target`, approbation des contributeurs externes | ES-32, M-LIV-09 | Relecture des workflows ; rapport actionlint ou zizmor |
| MP3 | Actions épinglées par empreinte complète, Dependabot `github-actions`, filtre de chemins sans action tierce | ES-05, M-LIV-09 | Rapport zizmor sans action non épinglée |
| MP4 | Flux réduit à deux contrôleurs, compte de service dédié limité au namespace, `--default-service-account`, aucune ressource de cluster dans le chart | ES-32, M-LIV-11 | `kubectl auth can-i --list` du compte ; `ClusterRole` créé par une `HelmRelease` refusé |
| MP5 | `env/demo` écrite par le seul job de publication sur `main`, règles de branche, contenu limité aux empreintes | ES-32, M-LIV-10 | Règles de branche exportées ; historique de `env/demo` |
| MP6 | Images signées par cosign sans clé, vérifiées avec identité et émetteur attendus avant application | ES-32, M-LIV-05, M-LIV-10 | `cosign verify` dans `up.sh` ; image non signée refusée |
| MP7 | Seuils d'ES-05 ; `.trivyignore` daté avec expiration ; govulncheck bloquant si correctif | ES-05, M-LIV-01 | BP3 ; fichier d'exceptions relu en fin de sprint |
| MP8 | Rien d'interdit dans le dépôt public (tfstate, `.env`, realm avec secrets, données réelles) ; détection à la poussée activée | ES-06, M-LIV-02, M-LIV-12 | gitleaks sur l'historique ; réglages du dépôt |
| MP9 | ZAP baseline sur l'URL publique une fois par sprint, règles ignorées versionnées | ES-05, ES-07 | Rapport ZAP archivé dans `docs/mesures/` |
| MP10 | Double facteur sur le compte GitHub, jetons personnels à portée fine | ES-32, M-LIV-13 | Capture des réglages de sécurité du compte |

**4. Menaces à ajouter au registre (§ 4, FC9)**

| Id | STRIDE | Menace | Impact | V | I | Risque |
|---|---|---|---|---|---|---|
| M-LIV-09 | T, E | Action tierce compromise ou injection dans un workflow : le jeton du job (écriture du registre ou de `env/demo`) est détourné. | Image malveillante publiée, puis déployée par Flux. | 2 | 4 | 8 Élevé |
| M-LIV-10 | T | Écriture illégitime de `env/demo` (jeton, règle de branche absente) : empreinte d'une image étrangère ou manifeste arbitraire. | Code arbitraire exécuté dans le cluster de démonstration. | 2 | 3 | 6 Modéré |
| M-LIV-11 | E | Contrôleur Flux laissé `cluster-admin` : tout manifeste réconcilié obtient les droits de tout le cluster. | Prise de contrôle du cluster, accès à tous les Secrets. | 2 | 4 | 8 Élevé |
| M-LIV-12 | I | Dépôt public : un fichier interdit (état Terraform, `.env`, données réelles) ou une sortie de pipeline divulgue un secret ou une donnée. | Secret ou donnée exposés à tous, durablement (forks, caches). | 2 | 3 | 6 Modéré |
| M-LIV-13 | S, T | Compte GitHub de Martin compromis. | Contrôle complet du code, de la CI et du déploiement. | 1 | 4 | 4 Modéré |

## Avis faisabilité

Fonction : Faisabilité — 2026-10-26. Périmètre : F1 + R1 + P2 avec P1 en filet, portes du tableau. Références : D03 (densité, allumage), avis faisabilité de D06 (Keycloak, Secrets à chaque `up`) et de D07 (ordre d'allumage, Job d'amorçage), [environnement-learner-lab.md](../00-cadrage/environnement-learner-lab.md). Avis sur dossier ; quotas GitHub non revérifiés [?].

**Synthèse.** Favorable. Rien ne bloque ; trois pièges sont à traiter dès la première itération : la vérification requise `ci-ok` face aux filtres de chemins, la visibilité des paquets GHCR, l'ordre Secrets → amorçage → HelmRelease. L'effort annoncé (3,75 à 6 j) est un peu court : compter **5,5 à 7,5 j**, migration comprise.

### Pipeline

- **Durée cible** : PR en moins de 15 min de bout en bout, 10 à 12 min visées ; `main` en moins de 20 min avec l'acceptation. Exécuteurs publics : 4 vCPU et 16 Go [?], suffisant pour une pile compose complète.
- **Parallélisation** :
  - un job par service et un pour le front, déclenchés par filtre de chemins calculé dans un premier job (et non par `paths:` du workflow, qui ignorerait tout le workflow) ;
  - `ci-ok` en `if: always()`, qui échoue si un job requis est en échec ou annulé et accepte « ignoré » : sans cela, une PR qui ne touche que `docs/` reste bloquée ;
  - `concurrency` avec annulation de l'exécution précédente sur la même PR ;
  - construction et analyse d'image par service en matrice : 2 à 4 min chacune, en parallèle.
- **Cache** :
  - cache Go (modules et compilation) et npm intégrés aux actions `setup-go` et `setup-node` ; couches Docker en `type=gha`, une portée par image ;
  - base de vulnérabilités Trivy mise en cache une fois par jour : son téléchargement depuis `ghcr.io` subit des refus pour débit [REX] ; prévoir le miroir `public.ecr.aws` ; sinon la CI rougit pour une raison étrangère au code ;
  - navigateurs Playwright : Chromium seul, mis en cache.
- **Services en CI** :
  - PostgreSQL et RabbitMQ par testcontainers-go : 5 à 15 s de démarrage chacun, sans réglage particulier ;
  - Keycloak : 30 à 60 s sur exécuteur hébergé avec import du realm [estimation]. Un seul Keycloak par job d'intégration, lancé en début de job en parallèle de la compilation, attente sur `/health/ready` (port 9000) avec 3 min de tolérance. Pour les tests de PR, l'image de base en `start-dev --import-realm` suffit ; l'image optimisée est construite et testée par son propre job (MA14), sans en dépendre ;
  - la pile compose d'acceptation est construite depuis les images du même run, transmises entre jobs par artefact ou cache, jamais poussées depuis une PR.
- **Seuil de couverture** : publié dès le sprint 1, **non bloquant** ; seuil bloquant fixé au sprint 2 sur la mesure réelle (D12), sous forme de « pas de baisse » par service plutôt que d'un pourcentage absolu, plus facile à tenir seul. Un seuil arbitraire dès le départ pousse à des tests sans valeur.
- **Acceptation** : sur `main` et sur les PR qui touchent `web/`, une OpenAPI ou le chart, par le même filtre de chemins ; 8 à 12 min. Sur `main` seule, un échec arrive après la fusion et rend `main` rouge : acceptable pour un développeur seul, à condition de corriger avant toute autre fusion. La file de fusion ne règle pas ce cas sur un compte personnel [?].

### Déploiement

- **Installation de Flux à chaque `up`** : `flux install --components=source-controller,helm-controller`, idempotent, 1 à 2 min ; il peut démarrer dès que le node group est prêt, en parallèle de la création de RDS. Pas de `flux bootstrap` : le dépôt et la branche `env/demo` sont publics, la source se lit sans jeton.
- **Densité** : 2 pods (kustomize et notification non installés), environ 150 à 300 Mio de demandes au total, déjà comptés dans le 1 Gio de l'avis faisabilité de D06. Rien ne change pour les places (26 + 2 sur 34).
- **Ordre dans `up.sh`** :
  1. Terraform, kubeconfig, `flux install` ;
  2. RDS prête, mots de passe générés et rangés dans Secrets Manager (D06, D07) ;
  3. Job d'amorçage `psql`, `kubectl wait --for=condition=complete`, puis suppression du Job et du Secret maître (D07) ;
  4. Secrets Kubernetes d'exécution et de migration, ConfigMap du paquet d'autorités RDS ;
  5. seulement alors : `GitRepository` sur `env/demo`, puis `HelmRelease` ; le crochet `pre-install` lance `goose` ;
  6. `kubectl wait helmrelease --for=condition=Ready`, puis tests de fumée et X17.
  - Une seule `HelmRelease` suffit : les services ne dépendent pas de Keycloak pour démarrer (D06). `dependsOn` ne sert que si RabbitMQ ou Keycloak sont sortis dans un chart séparé.
  - Réglages : `timeout` de 10 min (crochets et import du realm), `install.remediation.retries: 3`, `upgrade.remediation` avec retour arrière.
- **Reprise après extinction** : le cluster est détruit, il n'y a rien à reprendre ; `env/demo` est la seule source de l'état voulu, et la première réconciliation déploie les dernières empreintes de `main`. Si une session expire avant `down`, les nœuds redémarrent à la session suivante et Flux reprend seul, mais avec des Secrets d'une session morte : `up` doit tout recréer, et l'inventaire de D11 détecte ces restes.
- **Démonstration** : une fusion pendant la soutenance redéploierait en direct ; `flux suspend helmrelease` avant la démonstration, `flux resume` après.
- **Tirage depuis GHCR par la NAT** :
  - sans jeton, à condition que chaque paquet soit **public** ; un paquet créé par le premier `push` peut naître privé [?] : à vérifier à la première publication, sinon un passage manuel par paquet (5 à 7 paquets), une fois pour toutes. Symptôme : `ImagePullBackOff` sur un seul service ;
  - images Go de 10 à 30 Mo, Keycloak autour de 450 Mo : environ 1,5 à 3 Go par session, conforme à R1 ;
  - images Flux tirées de `ghcr.io/fluxcd`, versions épinglées par empreinte (MS9).
- **Filet P1** : `up.sh --sans-flux` lit les empreintes **dans le même fichier** de `env/demo`, puis `helm upgrade --install` avec le même chart et les mêmes valeurs. Une seule source pour les deux chemins ; P1 exécuté au moins une fois par sprint sur k3d, sinon il ne sera pas fiable le jour où il servira.

### Migration du dépôt local vers GitHub

État relevé en lecture seule : branche `main`, 12 commits, 37 fichiers suivis, aucun remote, `.git` de 617 Ko ; `.gitignore` couvre déjà secrets, états Terraform et identifiants ; aucune mention indésirable dans l'historique. D08 et D09 ne sont pas encore suivis.

- Avant le premier `push` : gitleaks sur tout l'historique ; décider si l'adresse électronique personnelle des commits devient publique, sinon la réécrire vers l'adresse `noreply` de GitHub (impossible proprement après publication).
- Dépôt créé vide (sans README ni licence générés, pour éviter un historique divergent), remote ajouté, `main` poussée.
- La vérification `ci-ok` ne peut être rendue obligatoire qu'après une première exécution : workflow livré par une première branche, exécuté, puis protection de `main`.
- Protection de secrets à la poussée, Dependabot et fusion automatique à activer dans les réglages.
- Processus : la fusion passe désormais par une PR GitHub. L'auteur ne peut pas être relecteur de sa propre PR : la relecture reste celle du processus (PILOTAGE § 5) et la fusion automatique se déclenche sur `ci-ok` vert.

### Effort et risques

| Poste | Jours |
|---|---|
| Workflow, jobs par service, agrégateur, caches | 1,5 à 2 |
| Portes (Trivy, govulncheck, gitleaks, Semgrep, contrôles propres), exceptions, seuils | 1,5 à 2 |
| Intégration avec Keycloak, acceptation compose | 1 à 1,5 |
| Flux, publication `env/demo`, ordre dans `up.sh`, filet P1 | 1 à 1,5 |
| Migration et réglages GitHub | 0,25 à 0,5 |
| Signature vérifiée, si retenue | 0,25 à 0,5 |

Risques, par ordre de probabilité :
1. Ordre Secrets, amorçage et crochet `pre-install` mal réglé : la première installation échoue à chaque session (déjà relevé par D07).
2. `ci-ok` bloqué par un job ignoré ou mal agrégé : PR impossibles à fusionner.
3. Base Trivy ou tirage GHCR limité en débit : CI rouge hors code ; atténué par le cache et le miroir.
4. Paquet GHCR privé : `ImagePullBackOff` au premier allumage.
5. Démarrage de Keycloak instable en CI : tests d'intégration intermittents ; tolérance de 3 min et relance unique, pas plus.
6. Quotas « gratuits et sans plafond » des dépôts publics non revérifiés [?] : à confirmer à la création.

## Décision de Martin

> **« GitHub public + Flux »** : forge GitHub en dépôt public (F1), images sur GHCR construites une fois sur `main` et déployées par empreinte (R1), signées par cosign et vérifiées au déploiement ; déploiement GitOps par Flux (P2) réduit à deux contrôleurs et limité au namespace de l'application, la CI publiant les empreintes sur `env/demo` et ne touchant jamais AWS ; déploiement depuis le poste (P1) gardé en filet. Portes bloquantes : lint, tests unitaires et d'intégration, gitleaks, govulncheck, Semgrep, Trivy, contrôles propres (realm, `go list -deps`, OpenAPI, `helm lint`) ; ZAP baseline une fois par sprint. Mesures de sécurité MP1 à MP10.
>
> Avant le premier envoi vers GitHub : gitleaks sur tout l'historique et choix de l'adresse e-mail publique des commits.

Date : 2026-10-26

> **Complément du 2026-10-26, adresse des commits : « noreply GitHub »**. Les commits publics portent l'adresse `<id>+<login>@users.noreply.github.com` du compte GitHub de Martin. L'historique local est réécrit vers cette adresse avant le premier envoi, ce qui est sans risque puisqu'aucun dépôt distant n'existe encore. Mise en œuvre : US-001.

## Conséquences

La décision de Martin est enregistrée ci-dessus ; ses conséquences :

- **PILOTAGE § 6** : forge et vérification requise `ci-ok` renseignées ; règles de protection de `main` décrites.
- **ES-05, ES-06, ES-32** : statut « Démontrée » visé ; BP3 démontrée sur une branche de démonstration (dépendance vulnérable connue et secret fictif : pipeline rouge, `main` verte).
- **D11** : `up.sh` installe Flux après les Secrets, attend la réconciliation, lance les tests de fumée ; `down.sh` inchangé. P1 reste écrit et testé.
- **D12** : couverture, résultats de lint, vulnérabilités ouvertes et durée du pipeline produits à chaque exécution ; outil d'analyse multi-langage à choisir par D12.
- **D13** : unitaires et intégration sur chaque PR ; acceptation sur `main` ; charge hors pipeline (M16 la rend facultative).
- **D14** : Flux entre dans l'expérimentation « orchestrateur » (temps d'allumage, pods consommés, comportement des crochets Helm).
- **Documentation** : schéma détaillé du processus CI/CD et du cycle DevSecOps dans `docs/processus/` (M3, M4) ; procédure d'exception aux portes ; README (badges, commandes).
- **Compétences (C9)** : GitHub Actions et GitOps entrent dans la cartographie.

## À savoir défendre en soutenance

1. **« Sans OIDC vers AWS, comment déployez-vous automatiquement ? »**
   - La CI ne touche jamais AWS : elle construit, analyse et publie des images identifiées par leur empreinte.
   - Flux, dans le cluster, tire l'état voulu et l'applique (modèle « pull »). L'API du cluster reste fermée à Internet (MS1).
   - Alternative écartée : recopier des identifiants de session de 4 h dans la forge, avec des droits proches de la console.
2. **« Qu'est-ce qui bloque une fusion ? »**
   - Secret détecté, lint, tests unitaires ou d'intégration en échec, vulnérabilité critique ou élevée corrigeable dans les dépendances, le code, la configuration ou l'image.
   - Exceptions : fichier versionné, chaque entrée justifiée et datée.
   - Démonstration : la branche BP3 rend le pipeline rouge.
3. **« Comment garantissez-vous que ce qui tourne en démonstration est ce qui a été testé ? »**
   - Construction unique sur `main`, déploiement par empreinte et non par étiquette ; aucune reconstruction entre environnements (ENF-29, ES-32).
   - SBOM joint à chaque image ; signature vérifiée si retenue.
4. **« Pourquoi un dépôt public pour une plateforme de paiement ? »**
   - Choix du POC sur données fictives : aucun secret n'est dans le dépôt, ce que gitleaks vérifie à chaque PR et sur l'historique.
   - En V1 : dépôt privé, offre payante ou exécuteurs internes, fédération OIDC vers le compte AWS de production.
5. **« Pourquoi Flux et pas Argo CD ? »**
   - Même modèle, Flux est plus léger (2 à 3 contrôleurs, sans interface ni base), ce qui compte sur un cluster de deux nœuds recréé à chaque session.

## Sources

Non revérifiées pour cette décision ; quotas et tarifs à confirmer à la création du dépôt.

- Contraintes du Learner Lab : [environnement-learner-lab.md](../00-cadrage/environnement-learner-lab.md) (§ 0, IAM, ECR, CodeBuild, modèles « pull » et identifiants recopiés).
- Facturation GitHub Actions et GitHub Packages : https://docs.github.com/en/billing/managing-billing-for-your-products/managing-billing-for-github-actions/about-billing-for-github-actions ; https://docs.github.com/en/billing/managing-billing-for-your-products/managing-billing-for-github-packages/about-billing-for-github-packages
- Minutes de calcul GitLab.com : https://docs.gitlab.com/ci/pipelines/compute_minutes/
- Flux, installation et sources OCI : https://fluxcd.io/flux/installation/ ; https://fluxcd.io/flux/components/source/ocirepositories/
- Trivy : https://trivy.dev/ ; gitleaks : https://github.com/gitleaks/gitleaks ; govulncheck : https://go.dev/doc/security/vuln/ ; Semgrep : https://semgrep.dev/docs/ ; ZAP baseline : https://www.zaproxy.org/docs/docker/baseline-scan/
- Signature sans clé : https://docs.sigstore.dev/cosign/signing/overview/
- Limites de tirage Docker Hub : https://docs.docker.com/docker-hub/usage/
