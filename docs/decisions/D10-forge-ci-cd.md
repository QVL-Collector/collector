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

Fonction : Sécurité — 2026-10-26. Références : [exigences-securite.md](../securite/exigences-securite.md) (ES-01, ES-07, ES-09, ES-29, ES-38 ; M-ACC-01, M-DIS-02, M-LIV-03, M-LIV-04) ; MS1, MS2, MS4, MS5 et MS10 de [D03](D03-hebergement-orchestration.md), MC7 de [D05](D05-communication-interservice.md), MA4, MA7 et MA13 de [D06](D06-authentification-autorisation.md), MD8 de [D07](D07-donnees.md). Avis sur dossier ; les points [?] relèvent des sondes de D14.

**Synthèse.** Avis favorable à la chaîne 1 (D1 + C1 + E1 + F1), sous les mesures ME1 à ME10. Elle ne met aucune clé privée dans le cluster et n'ajoute aucun composant exposé. Le risque principal n'est pas le TLS : c'est le **contournement de la liste blanche de l'ALB** (chemin non normalisé vers la console de Keycloak) et la **confiance accordée à `X-Forwarded-For`**, deux points à prouver par des tests négatifs plutôt qu'à supposer.

**1. Points soumis**

- **HSTS `includeSubDomains`, sans `preload`** : favorable. `preload` engagerait tout le domaine personnel de Martin dans les navigateurs pour des mois, au-delà du projet. `includeSubDomains` n'agit que s'il est servi par l'apex `<domaine>`, ce qui est le cas (front) ; conséquence acceptée : tout autre sous-domaine du domaine devra être en HTTPS. L'écouteur de l'ALB est le bon lieu ; s'il ne couvre pas la réponse fixe 404 [?], ce n'est pas bloquant, l'apex l'ayant déjà donné. Keycloak émet par défaut le même en-tête dans ses en-têtes de realm : valeurs identiques, sans conflit ; le retirer du realm garde un seul lieu.
- **Répartition des en-têtes** : favorable (ALB : HSTS ; nginx : front ; `libs/platform` : API ; Keycloak : ses pages). Quatre compléments :
  - nginx : `add_header ... always` (sinon absents des réponses 4xx et 5xx), repris dans chaque bloc `location` qui en déclare un (nginx n'hérite plus des `add_header` du niveau supérieur dès qu'un bloc en définit) ; `server_tokens off` ;
  - CSP finale du front : `default-src 'self'; script-src 'self'; style-src 'self' 'unsafe-inline'; img-src 'self' blob: https://<bucket>.s3.<région>.amazonaws.com; connect-src 'self' https://auth.<domaine>; frame-ancestors 'none'; base-uri 'none'; form-action 'self'; object-src 'none'`. Jamais `*.amazonaws.com` (MD8) ;
  - redirection silencieuse de D06 : si elle passe par une iframe (`silent-check-sso.html`), ce seul fichier doit autoriser `frame-ancestors 'self'` et le front `frame-src https://auth.<domaine>` ; l'iframe de statut de session (`checkLoginIframe`) est à désactiver, la CSP de Keycloak interdisant son intégration depuis un autre hôte [?, à vérifier à l'implémentation] ;
  - Keycloak : `KC_HOSTNAME=https://auth.<domaine>`, `proxy-headers xforwarded`, cookies `Secure` contrôlés par ZAP.
- **Limitation par réplica et `X-Forwarded-For`** : suffisante pour le POC, ES-29 étant « Recommandée ». Limite effective multipliée par 6 au plus (MS10), écart à écrire. Lire la **dernière** valeur est juste si l'ALB est en mode `append` : le fixer explicitement (`routing.http.xff_header_processing.mode`) plutôt que dépendre du défaut. En mode instance, `RemoteAddr` est l'adresse d'un nœud : l'en-tête est la seule source, et sa confiance repose sur MS2 et MS4 (NodePorts joignables depuis l'ALB seul). Trois réserves :
  - Keycloak (Quarkus) retient vraisemblablement la **première** valeur [?] : adresse falsifiable dans ses événements (D09) ; sans effet sur MA13, dont la force brute est par compte ;
  - écart ES-09 : aucune limite **par origine** sur la connexion (force brute par compte seulement, WAF refusé [?]). Le bourrage d'identifiants sur de nombreux comptes depuis une même adresse (M-ACC-01) n'est pas freiné. Accepté pour le POC sur comptes fictifs ; règle WAF à base de débit en V1 ;
  - la table des seaux par adresse doit être bornée (taille maximale, éviction), sinon la limitation devient elle-même un déni de service mémoire (M-DIS-02).
- **ALB ouvert ou restreint** : restreindre hors démonstration. Par défaut, groupe de sécurité de l'ALB limité à l'adresse de Martin (/32, comme MS1) ; ouverture à Internet par une variable explicite, le temps de la soutenance, de ZAP ou du tir de charge (adresses de D13). Keycloak et ses comptes de test sortent ainsi des balayages automatiques entre deux démonstrations. Point d'attention : si les services récupèrent les clés de Keycloak (JWKS) par `https://auth.<domaine>`, leur trafic sort par la NAT et serait refusé ; les récupérer par l'adresse interne du Service Keycloak, en vérifiant toujours `iss` égal à l'URL publique (MA4).
- **ALB vers NodePorts en clair** : accepté pour le POC. ES-01 admet « confiné » : sous-réseaux privés, NodePorts ouverts depuis le seul groupe de l'ALB (MS2), politiques réseau (MS4), pas de `hostNetwork` (PSS « restricted », MS5) ; le VPC ne permet pas d'écouter le trafic d'une autre interface. Risque résiduel faible, à documenter ; V1 : TLS jusqu'aux cibles.
- **Politique TLS** : `ELBSecurityPolicy-TLS13-1-2-2021-06` tient ES-01 mais garde en TLS 1.2 des suites CBC sans AEAD. Préférer `ELBSecurityPolicy-TLS13-1-2-Res-2021-06` (suites GCM seules), sans perte pour un navigateur récent. Faible, recommandé. Certificat : noms explicites `<domaine>` et `auth.<domaine>` plutôt que le joker ; enregistrement CAA limité à `amazon.com`.
- **« Réserver » jamais routé** : la défense est structurelle (MC7 : serveur et port distincts, Service `ClusterIP`, NodePorts hors liste refusés par MS5) ; l'ALB n'est que la seconde ligne. Le risque réel est côté Keycloak : une règle `/realms/collector/*` laisse passer `/realms/collector/../../admin/` ou ses formes encodées (`%2e%2e`, `%2f`) si l'ALB transmet le chemin brut et que Keycloak le normalise [?]. XE4 doit l'essayer avec `curl --path-as-is`, en plus des chemins directs.

**2. Point non soumis : délégation pendante.** Si le compte du lab est effacé, la zone disparaît mais le registrar délègue toujours vers ses serveurs de noms ; une zone créée dans un autre compte sur les mêmes serveurs permettrait de reprendre le domaine, certificat compris. Procédure : retirer ou repointer la délégation chez le registrar dès la perte du compte. Les alias éphémères, eux, ne laissent aucun nom pendant : favorable.

**3. Mesures minimales, si la chaîne 1 est retenue**

| Id | Mesure | Exigences, menaces | Preuve |
|---|---|---|---|
| ME1 | HSTS `max-age=31536000; includeSubDomains`, sans `preload`, par l'écouteur ; un seul lieu par en-tête | ES-01, ES-07, M-LIV-04 | XE3 ; ZAP |
| ME2 | nginx : en-têtes `always`, repris dans chaque `location`, `server_tokens off` ; CSP finale du point 1 | ES-07, M-ANN-03, M-PAY-04, MD8 | ZAP sur une page 200 et une 404 |
| ME3 | Exception de cadre limitée à `silent-check-sso.html` si la redirection silencieuse l'exige ; `checkLoginIframe` désactivé | ES-07 | Test de connexion silencieuse ; ZAP |
| ME4 | ALB en `append` explicite ; dernière valeur de `X-Forwarded-For` ; table des seaux bornée ; limites lues dans la configuration, sans route ni en-tête de contournement pour D13 | ES-29, M-DIS-02 | XE5 avec un `X-Forwarded-For` forgé : 429 et adresse réelle journalisée |
| ME5 | Groupe de sécurité de l'ALB restreint à l'adresse de Martin hors démonstration ; ouverture par variable explicite | ES-07, M-LIV-03 | Inventaire (X17) après `up` |
| ME6 | JWKS récupérées par l'adresse interne de Keycloak, `iss` public vérifié | ES-07, MA4 | Services opérationnels avec l'ALB restreint |
| ME7 | Politique `ELBSecurityPolicy-TLS13-1-2-Res-2021-06` ; certificat à noms explicites ; CAA `amazon.com` | ES-01 | XE3 |
| ME8 | XE4 étendu : traversées et encodages (`..`, `%2e%2e`, `%2f`, casse) vers `/admin/` et `/realms/master/`, avec `--path-as-is`, à chaque déploiement | ES-07, ES-38, MA7 | XE4, bloquant |
| ME9 | Écarts écrits : limite par réplica, pas de limite par origine sur la connexion, adresse falsifiable dans les événements de Keycloak [?], flux ALB vers nœuds en clair | ES-01, ES-09, ES-29 | Registre des écarts à jour |
| ME10 | Procédure « compte perdu » : délégation retirée chez le registrar | ES-07 | Procédure rédigée (D11) |

**Menaces à ajouter au registre** (§4.10, à reporter par Martin) :

| ID | STRIDE | FC | Scénario | V | I | Risque |
|---|---|---|---|---|---|---|
| M-EXP-01 | E, I | FC1, FC3 | Chemin non normalisé ou encodé qui franchit la liste blanche de l'ALB et atteint la console ou le realm `master` de Keycloak | 2 | 4 | 8 Élevé |
| M-EXP-02 | S, R | FC1 | `X-Forwarded-For` forgé : limitation de débit contournée ou adresse fausse dans les journaux | 3 | 2 | 6 Modéré |
| M-EXP-03 | D | FC1 | Table de limitation non bornée saturée par des adresses variées | 2 | 2 | 4 Modéré |
| M-EXP-04 | S, T | FC1 | Délégation DNS pendante après effacement du compte : reprise du domaine par une zone tierce | 1 | 3 | 3 Faible |

**Verdict** : GO pour la chaîne 1, sous réserve de ME1 à ME10 ; ME4 et ME8 bloquants avant toute ouverture de l'ALB à Internet.

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
