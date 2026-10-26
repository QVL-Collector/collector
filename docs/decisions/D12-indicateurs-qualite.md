# D12 — Au moins 4 indicateurs qualité et outils de mesure

| Champ | Valeur |
|---|---|
| Statut | Décidée |
| Date | 2026-10-26 |
| Décideur | Martin |
| Dépend de | [D04](D04-stack.md), [D09](D09-observabilite.md), [D10](D10-forge-ci-cd.md) |
| Impacte | D13, D14, [developpement.md](../processus/developpement.md) § 4, `docs/processus/qualite-indicateurs.md`, `docs/mesures/indicateurs-sprint-N.md`, `docs/mesures/analyse-dette.md` |
| Critères grille | C6, C7, C4 |

## Contexte et besoin

D12 fixe les indicateurs de qualité logicielle, leur formule, leur outil de mesure, leurs seuils et leur relevé par sprint.
D12 ne choisit ni les types de tests ni l'outil de charge (D13), ni les portes de sécurité (D10, déjà décidées) : elle réutilise ce que D09 et D10 produisent déjà.

### Exigences

| Source | Exigence (extrait) | Conséquence pour D12 |
|---|---|---|
| Grille C6, niveau 5 | « Au moins 4 indicateurs sont définis. Ils servent les objectifs de qualité logicielle (fonctionnalité, performance, maintenabilité, fiabilité). Leur mesure met en évidence des axes d'amélioration pouvant réduire la dette technique. » | Au moins un indicateur par objectif ; chaque mesure doit désigner où agir |
| Grille C6, niveau 4 | Écart à éviter : « 3 indicateurs seulement, ou des mesures qui ne font pas ressortir d'axes d'amélioration de la dette technique » | Un chiffre global ne suffit pas : la mesure descend au service, au fichier ou à la route |
| Consigne M2 | « 4 indicateurs de qualité, avec la justification de leur rôle contre l'accumulation de dette technique » | Une ligne « lien avec la dette » par indicateur |
| Consigne M5, M14 | Tests automatisés « permettant d'évaluer les métriques » ; lien avec le suivi des indicateurs | Les indicateurs sortent du pipeline, pas d'un relevé manuel |
| ENF-24 | « La dette technique est mesurée par au moins 4 indicateurs relevés à chaque sprint. » Vérification : « relevé par sprint ; tendance analysée » | Un fichier par sprint et une tendance d'un sprint à l'autre |
| ENF-14, ENF-15, ENF-16 | p95 < 500 ms sous charge nominale ; moins de 1 % d'erreurs au pic ; retour au minimum | Indicateur de performance mesuré dans Prometheus (D09) |
| ENF-11, ENF-12 | Perte d'une instance sans interruption ; aucun événement perdu | Indicateur de fiabilité mesuré sous tir |
| ENF-23 | « Chaque AC est relié à un test vert en CI » | Indicateur de fonctionnalité |
| ENF-05 | Aucune vulnérabilité critique non traitée à la livraison | Indicateur de sécurité, en complément des quatre objectifs |
| D10, Conséquences | Couverture, lint, vulnérabilités ouvertes et durée du pipeline produits à chaque exécution ; « outil d'analyse multi-langage à choisir par D12 » ; couverture publiée au sprint 1, puis bloquante « pas de baisse » par service | D12 choisit l'outillage d'analyse de code et fixe la règle de couverture |

### Contraintes

- **Environnement AWS éphémère** (D11) : les indicateurs de production ne se mesurent que pendant une session allumée, lors du tir de charge. Ils sont exportés dans `docs/mesures/charge/` (D09), pas lus en continu.
- **Dépôt public sur GitHub** (D10) : les services gratuits réservés aux dépôts publics sont accessibles.
- **Équipe d'une personne** : le relevé doit être automatique ; le fichier du sprint se génère, il ne se recopie pas.
- **Deux langages** : Go (4 services) et TypeScript (front). Chaque indicateur de code se mesure dans les deux.

## Le jeu d'indicateurs proposé

Commun aux deux options d'outillage ; seule la colonne « outil » de I5 change. Six indicateurs couvrent les quatre objectifs de C6 et la sécurité ; I7 est complémentaire.

| # | Objectif (ISO 25010) | Indicateur | Formule | Source | Cible | Alerte | Porte |
|---|---|---|---|---|---|---|---|
| I1 | Fonctionnalité | Taux d'AC couverts par un test vert | AC reliés à un test d'acceptation vert ÷ AC des US terminées du sprint | CI, `main` | 100 % | < 100 % | Oui : une US sans 100 % n'est pas « terminée » (DoD) |
| I2 | Performance | Latence p95 et débit du catalogue sous charge, délai de mise à l'échelle | `histogram_quantile(0.95, …)` par route ; délai entre franchissement du seuil HPA et pod prêt | Prometheus, tir en session (à partir du sprint 2) | p95 < 500 ms ; délai ≤ 90 s (X7 de D03) | p95 > 400 ms (80 % du budget) ou hausse > 20 % d'un sprint à l'autre | Non (hors pipeline, M16) ; bloque l'US qui porte l'AC « Performance » |
| I3 | Fiabilité | Taux d'erreur sous perte d'instance ; événements perdus ; messages écartés | 5xx ÷ requêtes sur la fenêtre de tir avec suppression d'un pod ; événements émis − événements traités après reprise ; entrées dans les files `.ecartes` | Prometheus, tir en session | < 1 % ; 0 perdu ; 0 écarté hors cas de test | ≥ 0,5 % ; tout événement perdu | Non ; un événement perdu bloque la clôture du sprint |
| I4 | Maintenabilité | Couverture de tests par service | Lignes couvertes ÷ lignes exécutables, par service Go et pour le front | CI, chaque PR | ≥ 70 % sur les paquets métier (indicatif) | Baisse de plus de 1 point par rapport à la référence | Sprint 1 : publiée, non bloquante ; dès le sprint 2 : bloquante « pas de baisse » par service (D10) |
| I5 | Maintenabilité | Complexité cognitive et duplication | Nombre de fonctions de complexité cognitive > 15 ; part de lignes dupliquées | CI, chaque PR | 0 fonction > 15 dans le code nouveau ; duplication < 3 % | Toute hausse d'un sprint à l'autre | Complexité : bloquante sur le code nouveau ; duplication : publiée |
| I6 | Sécurité | Vulnérabilités ouvertes par sévérité, exceptions et leur âge | Résultats govulncheck, Trivy, Semgrep, par sévérité ; entrées de `.trivyignore` et exceptions Semgrep, avec leur date | CI, chaque PR et quotidien sur `main` | 0 critique ou élevée corrigeable ; 0 exception échue | Exception échue ; moyennes en hausse | Oui pour critique et élevée (porte D10 existante) |
| I7 | Livraison (complément) | Taux de réussite du pipeline sur `main`, délai de livraison, durée du pipeline | Exécutions vertes ÷ exécutions terminées ; médiane « ouverture de PR → image publiée sur `main` » ; médiane de la durée du pipeline | API GitHub (`gh run list`, `gh pr list`) | ≥ 90 % ; < 1 jour ; < 15 min | < 80 % ; durée > 20 min | Non |

### Lien de chaque indicateur avec la dette technique

| # | Dette qu'il fait apparaître | Axe d'amélioration qu'il désigne |
|---|---|---|
| I1 | Dette de test : comportement livré sans preuve, régression silencieuse | L'AC non couvert, nommé |
| I2 | Dette de performance : requête N+1, index manquant, sérialisation coûteuse, qui ne se voit qu'en charge | La route la plus lente, comparée au sprint précédent |
| I3 | Dette de résilience : point unique de défaillance, reprise non idempotente, message mal formé | Le service qui renvoie des erreurs à la perte d'un pod ; la file qui écarte |
| I4 | Dette de test : code non exercé, donc risqué à modifier | Le service et le paquet dont la couverture baisse |
| I5 | Dette de conception : fonctions trop complexes à relire, logique copiée en plusieurs endroits | La fonction et le bloc dupliqué, par fichier |
| I6 | Dette de sécurité : dépendance vieillissante, exception acceptée puis oubliée | La dépendance à monter, l'exception à solder |
| I7 | Dette d'outillage : pipeline lent ou instable, qui pousse à contourner les portes | L'étape la plus lente, le test instable |

### Relevé par sprint et tableau de bord

- **Dans la CI** : chaque exécution écrit un résumé (`GITHUB_STEP_SUMMARY`) avec I1, I4, I5, I6 et la durée, et dépose un artefact `qualite.json` (valeurs par service).
- **Pendant un tir** : I2 et I3 se lisent dans Grafana (tableau « mise à l'échelle » de D09) ; les valeurs sont exportées après le tir dans `docs/mesures/charge/` (D09, D13).
- **En fin de sprint** : un script de `tools/` lit le dernier `qualite.json` de `main`, l'export du tir et l'API GitHub (I7), puis génère `docs/mesures/indicateurs-sprint-N.md` : valeur, cible, valeur du sprint précédent, tendance, et les trois pires éléments par indicateur (fonctions, routes, services).
- **Référence « pas de baisse »** : la couverture de référence par service est un fichier versionné, mis à jour à la revue de sprint. La porte compare à ce fichier, sans dépendre d'un artefact externe.
- **Analyse** : `docs/mesures/analyse-dette.md` (J6) reprend les tendances et en tire les actions de remboursement, inscrites au backlog.

Les métriques de CI ne sont pas poussées dans Prometheus : le cluster n'existe que pendant les sessions, la tendance tiendrait mal. Le fichier par sprint est la vraie série temporelle.

## Options d'outillage

### Option A — SonarQube Cloud, DORA complet, SLO

- **Description** :
  - analyse SonarQube Cloud à chaque PR et sur `main`, par l'action GitHub de SonarSource, avec import des rapports de couverture Go (`coverprofile`) et Vitest (LCOV) ;
  - porte de qualité « sur le code nouveau » : couverture, duplication, notes de fiabilité, sécurité et maintenabilité ;
  - dette mesurée par la méthode SQALE : effort de remédiation en minutes et ratio de dette, note de A à E ;
  - quatre indicateurs DORA (fréquence de déploiement, délai de livraison, taux d'échec des changements, temps de rétablissement) ;
  - SLO de latence et d'erreurs dans Grafana.
- **Avantages** :
  - un chiffre de dette directement parlant en soutenance (ratio SQALE, note) et un historique par analyse, sans script ;
  - complexité cognitive, duplication et couverture dans une même interface, descendues au fichier et à la ligne ;
  - outil reconnu, transférable (C9).
- **Inconvénients** :
  - un compte et un service tiers de plus, un secret `SONAR_TOKEN` dans la CI, une action tierce à épingler (MP de D10) ;
  - recouvre en partie golangci-lint, ESLint et Semgrep : deux sources pour les mêmes défauts, à réconcilier ;
  - conditions du plan gratuit et prise en charge de Go à vérifier au jour de l'adoption [?] ;
  - le ratio SQALE repose sur des coûts de remédiation forfaitaires par règle : un chiffre à expliquer, pas une mesure directe ;
  - DORA complet mal adapté au POC : la fréquence de déploiement dépend des sessions allumées, et le temps de rétablissement n'a pas de production à mesurer. Les afficher reviendrait à présenter des chiffres sans signification.
- **Coût AWS et effort** : aucun coût AWS ; 0,5 à 1 j (projet, jeton, import des couvertures, porte), plus le résumé par sprint (0,5 j).
- **Complexité ajoutée** : une plateforme externe et un secret.
- **Couverture grille** : C6 (dette chiffrée), C4 (porte de qualité dans le pipeline), C9.

### Option B — Outils natifs de la CI, sans plateforme externe

- **Description** :
  - couverture : `go test -coverprofile` puis `go tool cover -func` par service ; Vitest avec couverture v8 ;
  - complexité cognitive : linter `gocognit` de golangci-lint (déjà en place, D10) et règle `sonarjs/cognitive-complexity` d'ESLint (déjà en place, greffon à ajouter) ;
  - duplication : jscpd, qui lit Go et TypeScript ;
  - vulnérabilités : sorties SARIF et JSON des outils de D10, déjà produites ;
  - I1 : chaque test d'acceptation porte l'identifiant de son AC (`US-xxx-ACn`) ; un script croise les AC des fichiers `backlog/US-*.md` avec le rapport JUnit ;
  - DORA réduit à I7 ; SLO de D09 pour I2 et I3 ;
  - tout est agrégé dans `qualite.json` et le résumé de CI décrits plus haut.
- **Avantages** :
  - aucun compte, aucun secret, aucun service tiers : rien de neuf à sécuriser ;
  - une seule source par défaut : le linter qui bloque est celui qui mesure ;
  - formules transparentes, faciles à expliquer au jury ;
  - fonctionne aussi en local, avant la PR.
- **Inconvénients** :
  - pas de ratio SQALE ni de note globale : la dette s'exprime en éléments à traiter (fonctions trop complexes, blocs dupliqués, AC non couverts, exceptions) plutôt qu'en minutes ;
  - le script d'agrégation et de tendance est à écrire et à maintenir ;
  - pas d'interface d'exploration : la navigation se fait par le résumé de CI et le fichier du sprint.
- **Coût AWS et effort** : aucun coût AWS ; 1 à 1,5 j (jscpd, greffon ESLint, script d'agrégation et de tendance).
- **Complexité ajoutée** : un script et un outil (jscpd).
- **Couverture grille** : C6, C4, C7 (I1 relie AC et tests).

### Options écartées d'emblée

- **SonarQube Community auto-hébergé** : un serveur et une base à faire tourner hors des sessions AWS, pour un service que l'option A offre gratuitement.
- **Métriques de CI dans Prometheus et Grafana** : le cluster est détruit à chaque session ; la tendance par sprint y serait perdue.
- **Indicateurs sans outil, relevés à la main** : contraire à M5 et intenable seul.

### Synthèse

| Critère | A — SonarQube Cloud | B — Natif CI |
|---|---|---|
| 4 objectifs de C6 couverts | Oui | Oui |
| Dette chiffrée | Ratio SQALE, note A à E | Éléments à traiter, comptés et nommés |
| Axes d'amélioration | Par fichier et ligne, dans l'interface | Trois pires éléments par indicateur, dans le fichier du sprint |
| Tendance par sprint | Historique de la plateforme, plus fichier du sprint | Fichier du sprint |
| Sécurité de la chaîne | Un secret et une action tierce de plus | Rien de nouveau |
| Doublons avec D10 | Oui (lint, Semgrep) | Non |
| Effort | 1 à 1,5 j | 1 à 1,5 j |

## Recommandation

**Le jeu I1 à I6, avec I7 en complément, mesuré par l'option B (outils natifs de la CI).**

1. **Six indicateurs couvrent C6 sans trou** : fonctionnalité (I1), performance (I2), fiabilité (I3), maintenabilité (I4, I5), plus la sécurité (I6). Le niveau 5 demande quatre indicateurs ; en viser six laisse une marge si l'un d'eux n'est pas mesurable à temps (I2 et I3 dépendent du premier tir du sprint 2).
2. **Chaque mesure désigne où agir** : fonction, route, service, AC, exception. C'est le second membre du descripteur de C6, et c'est ce qu'un ratio global ne donne pas seul.
3. **B est la forme la plus simple** : elle mesure avec les outils qui bloquent déjà (D10) et n'ajoute ni compte, ni secret, ni seconde source de vérité. L'effort est le même que A.
4. **DORA est réduit à ce qui a un sens dans le POC** (I7) : taux de réussite, délai de livraison, durée. Fréquence de déploiement et temps de rétablissement sont présentés comme indicateurs de la V1, mesurables en production.

**Pourquoi pas A.** A apporte un chiffre de dette plus vendeur (SQALE). Ce gain ne justifie pas un service tiers et un secret, alors que B fait déjà ressortir les axes d'amélioration demandés. A reste un ajout possible en une demi-journée, sans changer les définitions : seule la colonne « outil » de I4 et I5 change.

### Points soumis aux avis

- **Sécurité** :
  - option A : portée du jeton `SONAR_TOKEN`, épinglage de l'action par empreinte, analyse limitée aux PR internes ;
  - option B : jscpd et le greffon ESLint entrent dans les dépendances analysées (Trivy) ;
  - I6 : faut-il bloquer aussi sur une exception de `.trivyignore` échue ?
- **Faisabilité** :
  - seuils de départ (complexité 15, duplication 3 %, couverture 70 % indicative) à confirmer sur la première mesure du sprint 1 ;
  - balisage des tests d'acceptation par identifiant d'AC, à caler avec D13 ;
  - I2 et I3 exigent un tir par sprint dès le sprint 2, dans une session déjà chargée (allumage d'environ 40 min).

## Avis sécurité

**Avis : favorable au jeu I1 à I7 et à l'option B, sous réserve des mesures MQ1 à MQ7.** B n'ajoute ni secret ni service tiers ; le risque principal de D12 n'est pas l'outillage mais la publication des résultats de sécurité dans un dépôt public.

**I6 et cohérence avec ES-05 et [devsecops.md](../processus/devsecops.md) § 3.**
- I6 est pertinent : il mesure la dette de sécurité là où elle se cache (exception acceptée puis oubliée, dépendance vieillissante).
- Trois écarts avec § 3 sont à corriger dans la définition :
  - la cible « 0 critique ou élevée corrigeable » oublie govulncheck, qui bloque toute vulnérabilité Go **atteignable** quelle que soit sa gravité (la base Go ne publie pas de gravité) ;
  - les sources omettent npm audit, OSV-Scanner, gosec et gitleaks, pourtant bloquants en § 3 ;
  - la question « faut-il bloquer sur une exception échue ? » est déjà tranchée : § 3 dit qu'une exception échue fait de nouveau échouer le pipeline. I6 doit afficher « Porte : oui » pour les exceptions échues, pas une simple alerte.
- La colonne « Alerte » parle de « moyennes en hausse » : à lire « modérées en hausse » sinon le terme est ambigu.
- Une exception Semgrep posée en ligne (`nosemgrep`) n'a ni date ni échéance : elle est invisible pour I6 et échappe à la règle des 30 jours.

**Indicateur de sécurité supplémentaire.** Pas de huitième indicateur ; deux sous-mesures suffisent.
- **Délai de correction** : § 3 fixe 48 h (P1) et 7 jours (P2) mais rien ne le mesure. I6 compte les vulnérabilités ouvertes au-delà de leur délai (date de première détection tirée de l'analyse quotidienne de `main`).
- **Tests négatifs d'autorisation** : ils sont des AC (devsecops § 2.1) et sont donc déjà dans I1. I1 isole leur part (AC de sécurité couverts ÷ AC de sécurité) : un 100 % global ne doit pas masquer un refus d'accès non testé.

**Fuite d'information par `qualite.json`, le résumé de CI et le fichier du sprint.** Dans un dépôt public, journaux, résumés de job et artefacts sont lisibles par des tiers, et `docs/mesures/indicateurs-sprint-N.md` est versionné en clair. Publier « CVE-x atteignable dans tel service, non corrigée » revient à divulguer une faille avant son correctif. Les « trois pires éléments » de I6 sont exactement cette liste.

**Option B, chaîne d'approvisionnement.** jscpd et `eslint-plugin-sonarjs` sont des dépendances de développement : `npm audit --omit=dev` ne les voit pas. Le script d'agrégation lit des artefacts produits par du code en cours de test : il doit les traiter comme des données non fiables.

**Option A, SonarQube Cloud, si elle était retenue.** Le plan gratuit vise les projets publics : les constats de sécurité y seraient consultables par tous [?] (à vérifier au jour de l'adoption), ce qui aggrave la fuite ci-dessus. S'y ajoutent `SONAR_TOKEN` (jeton d'analyse limité au projet, jamais un jeton utilisateur), l'action SonarSource épinglée par empreinte, et l'analyse restreinte aux branches internes (`pull_request`, jamais `pull_request_target`, aucun secret exposé aux PR de forks). C'est une raison de sécurité de plus pour préférer B.

**Mesures.**

| # | Mesure | Porte |
|---|---|---|
| MQ1 | I6 reprend exactement les seuils de devsecops § 3 : critique ou élevée corrigeable, Go atteignable quelle que soit la gravité, secret détecté ; sources complètes (govulncheck, Trivy, npm audit, OSV-Scanner, gosec, Semgrep, gitleaks) | B (portes D10 existantes) |
| MQ2 | Exception échue = pipeline rouge (déjà dans § 3) ; I6 publie le nombre d'exceptions, leur âge et les renouvellements (un seul permis) | B |
| MQ3 | Aucune exception Semgrep en ligne sans entrée dans un fichier d'exceptions versionné, avec date et échéance ; contrôle par le script I6 | B |
| MQ4 | I6 compte les vulnérabilités ouvertes au-delà du délai P1 ou P2 de § 3 ; I1 publie à part la part d'AC de sécurité couverts | NB (suivi audit de fin de sprint) |
| MQ5 | Données publiques de I6 = compteurs par sévérité et par service seulement ; aucun identifiant de vulnérabilité ouverte, composant ou chemin atteignable dans le résumé, `qualite.json` ou le fichier du sprint. Le détail reste dans le registre de l'audit de sécurité et n'est publié qu'une fois la faille corrigée | M (relecture du fichier du sprint) |
| MQ6 | jscpd et `eslint-plugin-sonarjs` figés dans le lockfile, installés par `npm ci --ignore-scripts`, couverts par OSV-Scanner (qui lit les dépendances de développement) | B |
| MQ7 | Le script de `tools/` ne lit que les artefacts d'exécutions `push` sur `main` (jamais d'une PR de fork) ; `qualite.json` validé par un schéma, jamais exécuté ni interpolé dans une commande | M (revue du script) |


## Avis faisabilité

**Avis : favorable au jeu I1 à I7 et à l'option B, sous réserve des points F1 à F8.** Tout est réalisable avec GitHub Actions, Prometheus et l'API GitHub. En revanche, l'effort annoncé (1 à 1,5 j) est sous-estimé : il faut compter **3,5 à 4,5 j**, dont environ 1 j partagé avec D09 et D13.

**Indicateur par indicateur.**
- **I1** : faisable. Les noms de test portent `US-xxx-ACn`. Côté Go, `gotestsum --junitfile` produit le JUnit ; Playwright fournit son rapporteur `junit`. Le script croise ce JUnit avec les AC extraits de `backlog/US-*.md`, ce qui suppose un format d'AC stable (F1). L'acceptation ne tourne que sur les PR ciblées : I1 se lit donc sur `main`, où tout s'exécute. La part des AC de sécurité (MQ4) passe par un marqueur dans l'US.
- **I2** : la latence p95 est faisable. Les compartiments par défaut de l'exportateur OpenTelemetry n'ont pas de borne à 0,4 s : l'alerte à 400 ms serait interpolée, il faut ajouter 0,4 et 0,5 aux bornes (F2). Le délai de mise à l'échelle par la différence de séries kube-state-metrics est imprécis à ± un intervalle de relevé (15 à 30 s), face à une cible de 90 s. Mieux vaut l'horodatage `kube_pod_status_ready_time` du nouveau pod, comparé à l'instant où le nombre de réplicas désirés change.
- **I3** : le taux de 5xx est faisable. « Événements perdus » est le point dur. Si l'on soustrait deux compteurs Prometheus, les incréments du pod supprimé, non relevés avant sa mort, créent de faux écarts. La mesure doit sortir de la base (lignes d'outbox publiées comparées aux lignes traitées côté consommateur), par une requête SQL lancée après le tir (F3). Les files `.ecartes` exigent d'activer les métriques par file de RabbitMQ (point [?] de D09).
- **I4** : faisable. `go test -coverprofile` par service, puis `go tool cover -func`, avec une liste d'exclusion versionnée (code généré, `main`). Vitest utilise le rapporteur `json-summary`. La référence est un fichier `couverture-reference.json` versionné. Il faut trancher l'incohérence entre « pas de baisse » (D10) et l'alerte « baisse > 1 point » : je propose une tolérance de 0,5 point dans la porte, sinon une suppression de code bien couvert bloque à tort (F4).
- **I5** : `gocognit` (`min-complexity: 15`) est faisable dans golangci-lint, avec `new-from-merge-base` pour ne bloquer que le code nouveau. Le comptage global se fait par une seconde passe non bloquante, en sortie JSON. ESLint n'a pas de mode « code nouveau » : la porte ne vise que les fichiers modifiés de la PR. jscpd lit Go et TypeScript et produit du JSON. Il faut exclure le code généré (clients OpenAPI) et les tests, sinon la duplication dépasse 3 % dès le premier jour (F5).
- **I6** : les compteurs viennent du SARIF et du JSON déjà produits. govulncheck n'a pas de gravité : il forme une catégorie « atteignable » à part (MQ1). L'âge des exceptions exige des dates lisibles par machine : `.trivyignore.yaml` avec `expired_at`, plus le fichier d'exceptions Semgrep de MQ3. La « date de première détection » (MQ4) demande un état conservé d'un jour à l'autre. Je propose un fichier JSON dans le registre de l'audit, pas dans un artefact de 90 jours (F6).
- **I7** : faisable avec `gh run list --branch main --event push --json conclusion,createdAt,updatedAt` et `gh pr list --state merged`. Le délai de livraison s'obtient en reliant la PR au run de `main` par son commit de fusion. Le quota de l'API n'est pas un sujet.

**`qualite.json` et résumé.** Un job `qualite` (`needs:` sur tous les jobs, `if: always()`) télécharge les artefacts, valide le schéma (MQ7), écrit `GITHUB_STEP_SUMMARY` et dépose `qualite.json`. En PR, le filtre de chemins rend le fichier partiel. Le schéma doit donc accepter « non mesuré » plutôt que zéro, et seul le `qualite.json` de `main` sert au relevé. Le job reste hors de `ci-ok` : les portes vivent dans les jobs d'origine.

**Script de `tools/`.** Je recommande un seul programme Go (`tools/indicateurs`), utilisé par le job `qualite` et en fin de sprint. Cela évite un troisième langage et rend le schéma testable. Il récupère le dernier artefact de `main` (`gh run download`), lit l'export du tir et appelle l'API GitHub. Il écrit `indicateurs-sprint-N.md` **et** un `indicateurs-sprint-N.json` versionné. La tendance se calcule sur le JSON du sprint précédent, jamais en relisant du Markdown (F7). Les trois pires éléments de I6 sont filtrés selon MQ5.

**Coût en durée de pipeline.** `gocognit` s'ajoute à golangci-lint pour moins de 10 s. Le greffon sonarjs coûte 10 à 20 s, et `gotestsum` ainsi que les exports de couverture sont négligeables. jscpd tourne dans un job parallèle de 30 à 60 s. Le job `qualite` ajoute 30 à 60 s en série, mais hors du chemin critique de `ci-ok`. Bilan : moins d'une minute sur `ci-ok`, la cible de 15 min est tenue.

**Effort.**

| Lot | Jours |
|---|---|
| CI : gocognit, sonarjs, jscpd, exports de couverture, JUnit, porte de référence | 1 |
| Job `qualite`, schéma, résumé | 0,75 |
| I1 : balisage et croisement des AC (avec D13) | 0,5 |
| I2 et I3 : requêtes, bornes d'histogramme, requête SQL d'événements perdus, export (avec D09, D13) | 1 |
| Script de fin de sprint et tendance | 0,5 à 0,75 |
| I6 : dates d'exception, état de première détection, filtrage MQ5 | 0,5 à 0,75 |
| Calibrage des seuils sur la première mesure (F8) | 0,25 |

**Risques.**
- I2 et I3 dépendent d'un tir par sprint dans une session d'environ 4 h, dont environ 40 min d'allumage. Une session ratée laisse le sprint sans I2 ni I3. Les six autres indicateurs maintiennent le minimum de quatre.
- La mesure des événements perdus reste fausse tant qu'elle s'appuie sur Prometheus (F3).
- Des tests instables dégradent I7 et poussent à relancer jusqu'au vert : il faut compter les relances.
- Le bruit de duplication et de complexité dans le code généré fausse la tendance s'il n'est pas exclu dès le sprint 1.
- La dérive du format des AC dans le backlog casse I1 en silence : le script doit échouer sur une US illisible.

## Décision de Martin

> **« Outils CI, tolérance 0,5 pt »** (option B) : sept indicateurs — I1 critères d'acceptation couverts par un test vert, I2 latence p95 et délai de mise à l'échelle, I3 fiabilité (erreurs à la perte d'un pod, événements perdus comptés en base, messages écartés), I4 couverture par service, I5 complexité cognitive et duplication, I6 vulnérabilités et exceptions alignées sur `devsecops.md` § 3, I7 DORA réduit — mesurés par les outils natifs de la CI, sans plateforme tierce. Porte de couverture « pas de baisse » avec une tolérance de 0,5 point par rapport à une référence versionnée. Résumés publics limités à des compteurs (MQ1 à MQ7) ; réserves de faisabilité F1 à F8. Seuils de départ confirmés à la première mesure du sprint 1.

Date : 2026-10-26

## Conséquences

La décision de Martin est enregistrée ci-dessus ; ses conséquences :

- **`docs/processus/qualite-indicateurs.md`** : définitions, formules, seuils et outils de I1 à I7 ; lien avec la table ISO 25010 de [developpement.md](../processus/developpement.md) § 4, dont le renvoi « à venir, D12 » est levé.
- **D10, pipeline** : `gocognit` activé dans golangci-lint, greffon `sonarjs` dans ESLint, jscpd, export JSON de la couverture, résumé de CI et artefact `qualite.json` ; porte « pas de baisse » de couverture à partir du sprint 2, fichier de référence versionné.
- **D13** : tests d'acceptation balisés par identifiant d'AC (I1) ; tir de charge par sprint avec suppression d'un pod (I2, I3) ; export des métriques en fin de tir.
- **D09** : requêtes Prometheus de I2 et I3 ajoutées au tableau « mise à l'échelle ».
- **D14** : aucun critère nouveau ; les relevés de I2 et I3 réutilisent X7 et X10 de D03.
- **`tools/`** : script de génération de `docs/mesures/indicateurs-sprint-N.md`.
- **Mesures** : premier relevé au sprint 1 (I1, I4 à I7), premier relevé complet au sprint 2 ; `docs/mesures/analyse-dette.md` en J6.
- **Backlog** : chaque alerte d'un indicateur ouvre un élément de remboursement de dette, priorisé à la planification.
- **Compétences (C9)** : lecture des indicateurs de qualité et méthode SQALE dans la cartographie, même si l'outil n'est pas retenu.

## À savoir défendre en soutenance

1. **« Pourquoi ces indicateurs et pas d'autres ? »**
   - Un par objectif de qualité cité par la grille, rattaché à une exigence chiffrée (ENF-14, ENF-15, ENF-11, ENF-12, ENF-23, ENF-05).
   - Chacun descend jusqu'à un élément à corriger : c'est ce qui en fait des outils contre la dette, pas des chiffres de vitrine.
2. **« Comment mesurez-vous la dette technique sans SonarQube ? »**
   - En éléments comptés et nommés : fonctions trop complexes, blocs dupliqués, baisse de couverture, AC non couverts, exceptions de sécurité en attente.
   - Le ratio SQALE convertit les mêmes défauts en minutes par des coûts forfaitaires ; il a été comparé et écarté au nom de la simplicité (option A).
3. **« Une couverture de 70 %, c'est arbitraire. »**
   - Oui, et c'est pourquoi la porte bloque une baisse, pas un pourcentage : un seuil absolu pousse à écrire des tests sans valeur (avis faisabilité de D10).
   - La couverture est lue avec I1 : un code couvert dont les AC ne sont pas testés reste une dette.
4. **« Où sont les indicateurs DORA ? »**
   - Taux de réussite du pipeline et délai de livraison sont mesurés (I7).
   - Fréquence de déploiement et temps de rétablissement supposent une production continue ; le POC est allumé par sessions. Ils sont prévus pour la V1.
5. **« Comment voit-on la tendance ? »**
   - Un fichier par sprint, généré depuis la CI et le tir, avec la valeur du sprint précédent et les pires éléments ; l'analyse de la dette en tire des actions inscrites au backlog.

## Sources

Consultées le 2026-10-26. Conditions des offres gratuites à revérifier au jour de l'adoption.

- **Grille et consigne** : [contexte-et-grille.md](../00-cadrage/contexte-et-grille.md) § 3.3 (M2, M5, M14) et § 4.2 (C6).
- **ISO/IEC 25010:2023**, modèle de qualité des produits : https://www.iso.org/standard/78176.html
- **SonarQube Cloud** : offres https://www.sonarsource.com/plans-and-pricing/sonarcloud/ ; complexité cognitive https://www.sonarsource.com/docs/CognitiveComplexity.pdf ; dette et notes https://docs.sonarsource.com/sonarqube-cloud/digging-deeper/metric-definitions/
- **Méthode SQALE** : https://www.sqale.org/
- **golangci-lint**, linters `gocognit` et `dupl` : https://golangci-lint.run/usage/linters/
- **eslint-plugin-sonarjs** : https://github.com/SonarSource/SonarJS/tree/master/packages/jsts/src/rules
- **jscpd** : https://github.com/kucherenko/jscpd
- **Couverture** : Go https://go.dev/blog/cover ; Vitest https://vitest.dev/guide/coverage
- **DORA** : https://dora.dev/guides/dora-metrics-four-keys/
- **Résumé de job GitHub Actions** : https://docs.github.com/actions/using-workflows/workflow-commands-for-github-actions#adding-a-job-summary
