# D13 — Stratégie de test et outil de charge

| Champ | Valeur |
|---|---|
| Statut | Décidée |
| Date | 2026-10-26 |
| Décideur | Martin |
| Dépend de | [D01](D01-fonctionnalite-poc.md), [D04](D04-stack.md), [D10](D10-forge-ci-cd.md) ; reprend les points D13 de [D03](D03-hebergement-orchestration.md), [D05](D05-communication-interservice.md), [D06](D06-authentification-autorisation.md), [D07](D07-donnees.md), [D08](D08-exposition-tls-domaine.md), [D09](D09-observabilite.md) |
| Impacte | D12 (indicateurs mesurés par les tests), D14 (X7, X10 réutilisent le script de charge), `docs/processus/tests.md`, `tests/acceptance/`, `tests/charge/`, `docs/mesures/charge/`, `presentation/demo-script.md`, AC des US |
| Critères grille | C7, C5, C4, C6, C8, C3 |

## Contexte et besoin

D13 fixe **quels tests existent, avec quels outils, où ils tournent, ce qu'ils produisent et qui en répond**, puis choisit l'outil du tir de charge.
Le placement dans le pipeline est déjà tranché par D10 (unitaires et intégration sur chaque PR, acceptation sur `main` et PR ciblées, charge hors pipeline) : D13 le détaille, sans le rediscuter.

### Exigences

| Source | Exigence (extrait) | Conséquence pour D13 |
|---|---|---|
| Grille C7 | « Le processus de test est formalisé. Il présente les différents types de tests, les outils de tests associés […] et les parties prenantes. Au moins deux types de tests sont appliqués au POC. Ils sont exécutés avec succès. » Écart : « processus incomplet (types de tests, outils, sorties attendues) ». | Tableau complet type × outil × lieu × sortie × fonction ; plusieurs types verts en CI |
| M5, M14 | Tests automatisés dans le pipeline, « au moins 2 types de tests permettant d'évaluer les métriques » | Unitaires, intégration, contrat, acceptation, sécurité : cinq types en CI |
| M10 | Fonctionnalité prouvée « par des tests d'appels voire des tests d'acceptation » | Acceptation API et navigateur dérivée des AC (ENF-23) |
| M16, C5 | « Test de charge en soutenance pour présenter la montée en charge » ; « disponibilité et montée en charge démontrées » | Scénario par paliers, HPA, KEDA, perte d'un pod, vidéo de secours (D03) |
| ENF-11, ENF-14, ENF-15, ENF-16 | Perte d'une instance sous charge ; p95 < 500 ms ; < 1 % d'erreurs ; retour au minimum | Seuils pass/fail du script de charge, fixés avant le tir |
| ENF-12, ENF-17 | Aucun événement perdu ; contrat versionné de l'outil de fraude | Test de reprise ; test de contrat |
| ENF-26, ENF-30 | Accessibilité contrôlée en CI ; textes en français et en anglais | axe-core dans Playwright, parcours dans les deux langues |
| D05, D06, D07, D08 | Mesures de sécurité « prouvées par un test négatif » | Liste des tests négatifs imposés (§ 2) |

### Contraintes

- Solo : chaque outil ajouté se maintient seul. Préférer l'outillage déjà retenu (D04, D10).
- Learner Lab : charge exécutée seulement en session allumée ; budget compté ; au plus 9 instances (D03).
- Densité du cluster : environ 25 pods au repos et 31 au pic pour 34 places (D03, D09). Un générateur dans le cluster prendrait ces places.
- ALB restreint à l'adresse de Martin hors démonstration (D08) ; limite de débit par adresse (ME4), sans route de contournement.
- Jetons utilisateur de 5 min, flux « mot de passe » désactivé dans le realm (D06, MA1).

## 1. Stratégie par type de test

Pyramide classique : beaucoup de tests rapides en bas, peu de tests de bout en bout en haut. Chaque type nomme son besoin ; aucun outil nouveau hors du tir de charge.

| Type | Ce qu'il prouve | Outil | Où | Sortie attendue | Fonction (R) |
|---|---|---|---|---|---|
| Unitaire back | Règles du Contrôle (fonction pure, tables de cas), machine à états de l'article, montants au centime | `go test -race -cover` ; fuzzing natif sur le dépôt de photos et la détection des coordonnées (D04) | Poste, CI PR | Rapport JUnit, couverture publiée (non bloquante, puis « pas de baisse » au sprint 2, D10) | Back |
| Unitaire front | Composants, hooks, i18n des libellés | Vitest, Testing Library | Poste, CI PR | Rapport JUnit, couverture | Front |
| Intégration | Dépôts SQL, outbox, consommateur idempotent, vérification des jetons, concurrence de la réservation, sur **vraie base et vrai broker** | testcontainers-go (PostgreSQL, RabbitMQ) ; Keycloak de docker compose | Poste, CI PR | Rapport JUnit ; bloquant | Back |
| Contrat | Producteur et consommateur respectent `contracts/` (JSON Schema des messages, OpenAPI des API) ; clé dupliquée et message malformé refusés | `santhosh-tekuri/jsonschema`, `kin-openapi` dans `go test` ; exemples de `contracts/` rejoués des deux côtés | Poste, CI PR | Un test par paire producteur-consommateur (D05) ; bloquant | Back (producteur et consommateur), Test (exemples) |
| Acceptation API | Chaque AC Given/When/Then de l'US, résultat asynchrone du contrôle attendu dans un délai borné | Playwright (`request`), TypeScript, écrit depuis les AC, jamais depuis le code | Pile compose en CI : `main` et PR qui touchent le front ou un contrat | Rapport HTML par AC ; matrice AC → test (ENF-23) ; bloquant | Test |
| Acceptation navigateur (E2E) | Parcours soumission → contrôle → publication → consultation ; CSP active (toute violation fait échouer) | Playwright, Chromium seul | Idem | Rapport HTML, traces des échecs | Test |
| Accessibilité | Aucune violation axe « serious » ou « critical » sur chaque écran du POC | `@axe-core/playwright`, dans les deux langues ; `jsx-a11y` au lint | Idem | Liste des violations ; vérification manuelle des parcours en fin de sprint (ENF-26) | Front, Test |
| i18n | Mêmes clés en français et en anglais ; aucun texte en dur | Script de parité des fichiers de traduction ; parcours Playwright joués en `fr` et `en` | CI PR (parité), avec l'acceptation (parcours) | Clé manquante = échec | Front |
| Non-régression | Rien de ce qui marchait ne casse | Pas d'outil dédié : toute la suite ci-dessus rejouée à chaque PR et sur `main` ; chaque anomalie corrigée laisse un test qui la reproduit | CI PR, CI main | Historique des exécutions | Toutes |
| Sécurité statique | Secrets, code, dépendances, images, configuration | gitleaks, golangci-lint (gosec), Semgrep, govulncheck, Trivy (D10) | CI PR, quotidien | Rapports ; bloquant selon D10 | DevOps, Sécurité |
| Sécurité négative | Refus attendus : authentification, rôle, objet, canal, base, exposition (§ 2) | `go test` (services), Playwright API (bout en bout), script `curl` après déploiement | CI PR, CI main, AWS | Un test vert par mesure listée ; bloquant | Test, Back, Sécurité (liste) |
| DAST | En-têtes, CSP, cookies, surface exposée | ZAP baseline | Compose quotidien ; URL publique en fin de sprint (D10) | Rapport trié dans l'audit de fin de sprint ; non bloquant | Sécurité |
| Après déploiement | Pile vivante et surface conforme | Tests de fumée, X17 (D10) | AWS, fin de chaque `up` | Échec signalé par `up.sh` | DevOps |
| Performance et charge | ENF-14, ENF-15, ENF-16 ; HPA et KEDA | Outil choisi au § 3 | AWS, hors pipeline (D10) ; profil court sur le poste | Rapport du tir, CSV Prometheus, captures (D09) dans `docs/mesures/charge/<date>/` | DevOps (tir), Test (scénario), Martin (go/no-go) |
| Résilience légère | Perte d'un pod sous charge (ENF-11) ; destinataire arrêté puis relancé, aucun événement perdu (ENF-12) ; broker redémarré, articles « en contrôle » renvoyés (D05) | `kubectl delete pod` pendant le tir ; testcontainers (arrêt et relance du conteneur) | AWS ; CI PR pour ENF-12 | Taux d'erreurs sur la fenêtre ; événements comptés | DevOps, Back |

**Pas d'outil de chaos** (Chaos Mesh, Litmus) : deux pannes ciblées suffisent au besoin, une commande et un conteneur arrêté les provoquent.

### Lieux d'exécution

| Lieu | Ce qui y tourne | Déclencheur |
|---|---|---|
| Poste | Unitaires, intégration, acceptation sur compose, profil de charge court contre compose | Le développeur ou la fonction test |
| CI, PR | Lint, unitaires, intégration, contrat, parité i18n, scans ; acceptation si la PR touche le front ou un contrat | Chaque PR ; bloquant |
| CI, `main` | Toute la suite, acceptation et accessibilité comprises ; ZAP quotidien sur compose | Chaque fusion ; quotidien |
| AWS (session allumée) | Fumée et X17, tests négatifs d'exposition et de réseau, tir de charge, perte d'un pod, ZAP sur l'URL publique | `up.sh` ; manuel en fin de sprint et en soutenance |

Les tests négatifs qui dépendent de l'infrastructure réelle (politiques réseau, TLS imposé par RDS, routes de l'ALB) ne tournent que sur AWS : compose ne les reproduit pas.

### Parties prenantes

| Fonction | Responsabilité dans le test |
|---|---|
| Test | Écrit l'acceptation depuis les AC, jamais depuis le code ; tient la matrice AC → test ; rédige le scénario de charge |
| Back | Unitaires, intégration, contrat côté producteur et consommateur, tests négatifs des services |
| Front | Unitaires front, accessibilité au lint, parité i18n |
| Sécurité | Liste des tests négatifs, tri des scans et de ZAP, audit de fin de sprint |
| DevOps | Pipeline, tests après déploiement, exécution du tir sur AWS, export des mesures |
| Revue de code | Refuse une PR sans test pour un comportement nouveau ou une anomalie corrigée |
| Martin (Lead Developer) | Recette : joue le parcours, valide la livraison ; fixe les seuils du tir ; décide le recours à la vidéo |

## 2. Tests négatifs imposés par les décisions de sécurité

| Source | Test | Niveau |
|---|---|---|
| D06 BP2 | Jeton absent, expiré, signature invalide, mauvais émetteur, mauvaise audience → 401 | `go test` (paquet de vérification) + acceptation API |
| D06 BP5, ES-04 | Vendeur B sur l'article de A → 403 ou 404 ; acheteur sur une route vendeur ; non-admin sur la liste « à vérifier » | Acceptation API, un test par type de ressource |
| D06 MA3, ES-38 | `collector-web` n'obtient pas la portée `reserver` ; « réserver » refusé sans jeton de service de Transactions | Intégration (Keycloak compose) |
| D05 XC2 | Un service ne crée ni file ni liaison, ne publie pas hors de son sujet ; émetteur `user-id` falsifié refusé (MC2) | Intégration (RabbitMQ réel) |
| D05, D04 | Message à clé dupliquée, malformé ou hors schéma → mis à l'écart, jamais traité | Contrat |
| D07 MD1 | Le rôle de Transactions ne se connecte pas à la base `marche` ; `CREATE TABLE` refusé au rôle d'exécution | Intégration ; rejoué sur RDS |
| D07 | Connexion à RDS sans TLS refusée | AWS |
| D07 | Fichier non image refusé ; photo publiée sans géolocalisation ; photo non publiée inaccessible à un tiers ; décision admin périmée → 409 ; réservation concurrente : une seule gagnante | Intégration et acceptation |
| D08 ME4 | `X-Forwarded-For` forgé : 429 et adresse réelle journalisée | AWS |
| D10 X17, MC7 | `/admin/`, `/realms/master/` → 404 ; port 9000 et « réserver » injoignables depuis l'ALB | AWS, après déploiement |
| MA8 | Flux non autorisé entre deux services refusé par la politique réseau | AWS |

## 3. Sous-question : outil de charge

### Critères

| # | Critère |
|---|---|
| L1 | Scripts en texte, relus et versionnés comme du code |
| L2 | Seuils pass/fail dans le script, code de sortie non nul en cas d'échec |
| L3 | Lecture en direct pendant le tir, à côté de Grafana (HPA, KEDA, D09) |
| L4 | Modèle ouvert : débit imposé par paliers, qui ne baisse pas quand la latence monte (sinon la saturation est masquée) |
| L5 | Peu de ressources côté générateur ; exécutable depuis le poste |
| L6 | Effort et compétences : TypeScript connu (D04), un seul binaire de préférence |

### Option A — k6

- **Description** : binaire Go unique, scripts JavaScript ou TypeScript ; exécuteurs `ramping-arrival-rate` (débit par paliers) ; `thresholds` avec arrêt anticipé ; tableau de bord web intégré, exportable en HTML ; sortie Prometheus *remote write* en option ; k6-operator pour une exécution dans le cluster.
- **Avantages** : seuils natifs exprimés comme ENF-14 et ENF-15 (`p(95)<500`, `rate<0.01`) ; modèle ouvert natif ; empreinte faible ; même langage que les tests Playwright de la fonction test ; scénarios multiples dans un même script (consultation et vague de soumissions).
- **Inconvénients** : pas de vrai navigateur dans le tir de base (sans objet ici : le tir vise l'API) ; la sortie Prometheus exige d'activer la réception *remote write* sur Prometheus.
- **Coût AWS et effort** : nul côté AWS si le tir part du poste ; ≈ 1 à 1,5 j (script, paliers, export).
- **Complexité ajoutée** : un binaire, un dossier `tests/charge/`.
- **Couverture grille** : C5 (démonstration lisible), C7 (seuils = sortie attendue), C6 (p95 et erreurs comme indicateurs).

### Option B — Gatling

- **Description** : moteur JVM ; simulations en Java, Kotlin, Scala, ou JavaScript/TypeScript (SDK récent [?]) ; assertions bloquantes ; rapport HTML détaillé.
- **Avantages** : rapport HTML de grande qualité ; modèle ouvert natif (`injectOpen`) ; assertions dans la simulation.
- **Inconvénients** : JVM à installer et à dimensionner ; pas d'export Prometheus natif dans la version libre (Graphite, donc un exportateur de plus) ; suivi en direct limité à la console ; SDK JavaScript moins mûr que la version JVM.
- **Coût AWS et effort** : nul côté AWS ; ≈ 1,5 à 2 j, montée en compétence sur le DSL.
- **Complexité ajoutée** : JVM et outil de construction sur le poste.
- **Couverture grille** : C5, C7 équivalents ; rapport plus riche, direct moins lisible.

### Option C — Locust

- **Description** : scripts Python ; interface web avec courbes en direct ; mode maître et ouvriers.
- **Avantages** : interface en direct immédiate ; scénarios en Python simple.
- **Inconvénients** : modèle fermé (utilisateurs simulés), débit imposé seulement par contournement ; seuils pass/fail par extension ou code à écrire ; export Prometheus par un exportateur tiers ; un processus par cœur, à multiplier pour monter en débit ; Python hors de la stack (D04).
- **Coût AWS et effort** : nul côté AWS ; ≈ 1,5 j, plus les seuils à coder.
- **Complexité ajoutée** : un environnement Python de plus.
- **Couverture grille** : C5 correct ; C7 plus faible (seuils non natifs).

### Option D — JMeter (citée pour mémoire)

Plans de test en XML produit par une interface graphique, illisibles en revue (L1) ; JVM lourde ; export Prometheus par greffon. Référence historique, sans avantage ici.

### Où lancer le générateur

| | G1 — Poste de Martin, via l'ALB public | G2 — Pod dans le cluster | G3 — Instance EC2 dédiée dans le VPC |
|---|---|---|---|
| Réalisme | Chemin réel : DNS, TLS, ALB, limites de débit | Contourne l'ALB si l'adresse est interne ; sinon sort par la NAT (coût, adresse refusée par le groupe de sécurité) | Chemin réel |
| Fausse la mesure ? | Non | **Oui** : CPU et places prises sur les nœuds mesurés ; peut déclencher un nœud de plus à lui seul | Non |
| Réseau | Dépend du réseau du lieu (partage mobile en secours, D03) | Interne | Stable |
| Coût et effort | Nul ; l'ALB est déjà ouvert à l'adresse de Martin | k6-operator ou Job à écrire ; places à réserver | Une instance de plus (limite de 9), IaC, accès à prévoir |
| Limite de débit (ME4) | Une seule adresse : seuils de l'environnement de tir réglés en conséquence, jamais de contournement (D08) | Idem | Idem |

Le débit visé (quelques dizaines à quelques centaines de requêtes par seconde, à calibrer) tient largement sur un portable : la saturation possible est celle du réseau du lieu, pas du générateur.

### Synthèse

| Critère | A — k6 | B — Gatling | C — Locust | D — JMeter |
|---|---|---|---|---|
| L1 Scripts versionnés | JS/TS lisible | Code lisible | Python lisible | XML |
| L2 Seuils pass/fail | Natifs | Natifs (assertions) | Extension ou code | Assertions, lourdes |
| L3 Direct à côté de Grafana | Tableau de bord intégré ; Prometheus en option | Console ; Graphite | Interface web | Greffon |
| L4 Modèle ouvert | Natif | Natif | Contourné | Greffon |
| L5 Empreinte, poste | Binaire unique, faible | JVM | Python, multi-processus | JVM lourde |
| L6 Effort, stack | TypeScript connu | DSL à apprendre | Hors stack | Interface graphique |

## Recommandation

**Stratégie du § 1 et tests négatifs du § 2 ; outil de charge : k6 (option A), lancé depuis le poste via l'ALB public (G1).**

1. **k6 coche tous les critères sans ajout** : seuils natifs qui recopient ENF-14 et ENF-15, débit imposé par paliers, binaire unique, TypeScript déjà pratiqué par la fonction test.
2. **Direct au plus simple** : le tableau de bord web intégré de k6 montre le côté client (débit, p95, erreurs) ; Grafana montre le côté serveur (pods, HPA, KEDA, nœuds, déjà prévu par D09). Deux fenêtres côte à côte, aucune configuration de Prometheus. La sortie *remote write* vers Prometheus, pour un écran unique, est une option soumise ci-dessous.
3. **G1 ne fausse pas la mesure** et réutilise l'ouverture de l'ALB à l'adresse de Martin. G2 prendrait CPU et places sur les nœuds qu'il prétend mesurer ; G3 ajoute une instance sans besoin démontré. Si le réseau du lieu flanche : partage mobile, puis vidéo (D03).
4. **Un seul script pour trois usages** : calibrage en expérimentation (X7, X10 de D14), mesures de fin de sprint, soutenance.

### Scénario de soutenance

Seuils fixés et inscrits dans `tests/charge/` et dans les valeurs « démonstration » du chart **avant** le passage ; débits calibrés par X7 et X9.

| Temps | Palier | Attendu à l'écran |
|---|---|---|
| 0 | Repos | 2 pods Place de marché, 1 Contrôle ; p95 nominal |
| 0 à 2 min | Charge nominale : catalogue et fiches, visiteurs anonymes | p95 < 500 ms sur 2 pods (ENF-14) |
| 2 à 5 min | Montée par paliers jusqu'au pic (2 à 3 fois le nominal, à calibrer) | CPU au-delà de la cible du HPA ; 2 → 4 à 6 pods en moins de 90 s (X7) ; nœud ajouté si le Cluster Autoscaler est validé (X9) |
| ≈ 4 min | Vague de soumissions par des vendeurs de test (scénario parallèle) | File du Contrôle qui monte ; KEDA 1 → 2 ou 3 consommateurs, puis vidage |
| ≈ 5 min | Suppression forcée d'un pod de la Place de marché au plus fort | < 1 % d'erreurs sur la fenêtre ; remplaçant prêt en moins de 60 s (ENF-11, X10) |
| 5 à 8 min | Descente | Retour à 2 pods et 1 consommateur (stabilisation réduite à 60 s, D03) ; ENF-16 |
| Fin | Export | `make export-mesures` (D09) et rapport HTML de k6 dans `docs/mesures/charge/<date>/` |

Seuils du script : `http_req_duration{scénario:consultation} p(95)<500`, `http_req_failed rate<0.01`, contrôles de contenu à 99 %. Un 429 n'apparaît pas pendant le tir : les limites de l'environnement de tir sont réglées avec lui (D08) ; le 429 est prouvé par son propre test (ME4).
Jetons des vendeurs : obtenus dans la phase de préparation du script par le flux code + PKCE, sans flux « mot de passe » ; la vague dure moins que leur validité de 5 min, sinon renouvellement par le jeton de rafraîchissement.
Bascule vers la vidéo, décidée d'avance (D03) : test HTTPS non vert à H − 20 min, ou aucune montée 2 min après le début du tir.

### Données de test

- Exclusivement fictives : catalogue de démonstration chargé par le script idempotent (D03, D07), base recréée à chaque session, donc remise à zéro naturelle après une vague de soumissions.
- Un utilisateur de test par rôle et plusieurs vendeurs de test pour la vague, créés par l'import du realm, mots de passe régénérés à chaque session (D06, MA10), jamais dans le dépôt.
- Photos générées (images synthétiques), sans métadonnées de personnes réelles ; une photo de test avec coordonnées GPS sert au test de géolocalisation.
- Aucune donnée personnelle réelle, cohérent avec l'écart ES-26 assumé.

### Points soumis aux avis

- **Sécurité** :
  - ouverture de l'ALB à Internet pendant la soutenance seulement, le tir restant possible avec la seule adresse de Martin ;
  - seuils de limitation propres à l'environnement de tir : valeurs, et preuve qu'aucune route de contournement n'existe ;
  - si la sortie *remote write* est retenue : point d'écriture de Prometheus joignable par `port-forward` seulement ;
  - comptes vendeurs de test : création, régénération, destruction.
- **Faisabilité** :
  - effort total de D13 au-delà de ce que D04 et D10 comptent déjà (script de charge, matrice AC → test, parité i18n, tests négatifs AWS) ;
  - flux PKCE scripté dans k6 : fiabilité face aux pages de connexion de Keycloak ;
  - durée de la suite d'acceptation sur compose en CI (Keycloak 30 à 60 s, D10).
- **Martin** :
  - k6 *remote write* vers Prometheus (un écran) ou tableau de bord intégré (deux fenêtres) ;
  - profil de charge court contre compose sur `main`, non bloquant, pour qu'un script cassé ne se découvre pas le jour J : oui ou non ;
  - cible CPU du HPA (point de départ proposé : 60 %, ajusté par X7).

## Avis sécurité

**Avis : favorable au § 1, au § 2 et à k6 lancé depuis le poste (A + G1), sous réserve des mesures MT1 à MT8.** Le socle est complet côté statique (gitleaks, gosec, Semgrep, govulncheck, Trivy, contrôle du realm) et côté dynamique (ZAP baseline). Les risques de D13 sont ailleurs : des tests négatifs manquants au § 2, un relèvement des limites de débit qui affaiblirait l'environnement ouvert à Internet, et des rapports publiés dans un dépôt public.

**Complétude du § 2.** Les refus d'authentification (BP2), de rôle (ES-03) et d'objet (ES-04, IDOR) sont couverts. Il manque six cas déjà exigés ailleurs :
- **ME8** : le § 2 ne teste que `/admin/` et `/realms/master/` en clair. D08 impose aussi les traversées et encodages (`..`, `%2e%2e`, `%2f`, casse) avec `--path-as-is`. C'est le contournement de la liste blanche de l'ALB, le risque principal de D08.
- **Adjoint confus (M-MOD-02)** : le test de la ligne MA3 vérifie la portée, pas le contenu. Il faut aussi tester que « réserver » refuse un article non publié et une réservation au nom d'un autre acheteur, même quand le jeton de service est valide.
- **Rejeu** : rien ne teste le rejeu. Deux cas : un message déjà traité est redélivré sans double effet (idempotence, M-CAN-02), et un jeton de rafraîchissement déjà utilisé est refusé (rotation MA11).
- **Clés JSON dupliquées** : le test ne couvre que les messages. Il faut l'étendre aux corps d'API (même décodeur, D04), sinon deux composants peuvent lire deux valeurs différentes pour le même champ.
- **Bombe de décompression** : « fichier non image refusé » ne prouve pas les bornes. Il faut un corps au-delà de la limite (413, `MaxBytesReader`) et une image petite en octets mais énorme en pixels, refusée après `DecodeConfig` et avant décodage (D04).
- **X-Forwarded-For** : le test ME4 existe. Il doit couvrir les deux formes de falsification, un en-tête unique forgé et une liste de plusieurs valeurs.

**SAST et DAST.** Rien à ajouter au § 1. ZAP baseline reste non bloquant, ce qui est acceptable puisque les refus d'API sont prouvés par les tests négatifs, pas par ZAP.

**Flux PKCE scripté dans k6.**
- Le script lit les identifiants des vendeurs par `__ENV` ou par un fichier hors dépôt généré à chaque session (MA10). Aucun identifiant dans `tests/charge/`.
- Le flux « mot de passe » reste désactivé : le contrôle du realm (devsecops § 3) bloque déjà un raccourci qui le réactiverait pour faciliter le script.
- Le jeton de rafraîchissement ne sert que si la vague dépasse 5 min. Il reste en mémoire et n'est jamais écrit.
- `--http-debug` est interdit pendant un tir enregistré : il imprime les en-têtes `Authorization` et les cookies.
- Le callback porte `code` et `state` dans l'URL, que k6 conserve comme étiquette de métrique et donc dans le rapport HTML. Chaque requête d'authentification doit être nommée par l'étiquette `name` pour que l'URL brute n'apparaisse pas.
- Les comptes de test disparaissent avec la base au `down`. Aucun ne survit d'une session à l'autre.

**Limites de débit pendant le tir.** Relever la limite par adresse pour toutes les adresses donnerait le même plafond relevé à n'importe qui pendant la soutenance, quand l'ALB est ouvert à Internet. Deux voies acceptables, au choix de Martin :
- (a) limites relevées seulement quand le groupe de sécurité de l'ALB est restreint à l'adresse de Martin, puis remises à la valeur nominale avant l'ouverture ;
- (b) liste d'adresses en configuration (l'adresse de Martin seule), avec une limite propre au tir. L'adresse est lue par la règle ME4 (dernière valeur de `X-Forwarded-For` ajoutée par l'ALB), donc non falsifiable. Toutes les autres adresses gardent la limite nominale.

Aucune route ni aucun en-tête de contournement dans les deux cas (ME4). Le 429 est rejoué après le tir pour prouver le retour à la normale. La protection contre la force brute de Keycloak n'est jamais relevée.

**Données et rapports publics.** Le choix de données exclusivement fictives et de photos synthétiques convient. En revanche, les dossiers `docs/mesures/charge/`, les rapports ZAP et les sorties des tests négatifs sont publics. MQ5 (D12) s'applique : pas de constat ouvert détaillé dans le dépôt. Il faut aussi masquer l'adresse IP de Martin, qui apparaît dans la preuve ME4 (« adresse réelle journalisée ») et dans le paramétrage des limites.

**Mesures.**

| # | Mesure | Porte |
|---|---|---|
| MT1 | § 2 complété par ME8 (traversées, encodages, `--path-as-is`) et par l'adjoint confus (article non publié, acheteur substitué malgré un jeton de service valide) | B (après déploiement, avant ouverture de l'ALB) |
| MT2 | Tests de rejeu : message redélivré sans double effet ; jeton de rafraîchissement déjà utilisé refusé | B (intégration) |
| MT3 | Clés dupliquées testées aussi sur les corps d'API ; corps trop gros → 413 ; image à pixels excessifs refusée avant décodage | B (intégration) |
| MT4 | `X-Forwarded-For` forgé testé en valeur unique et en liste ; 429 rejoué après chaque tir | B (AWS) |
| MT5 | Identifiants k6 par `__ENV` ou fichier hors dépôt régénéré à chaque session ; `--http-debug` interdit ; étiquette `name` sur les requêtes d'authentification ; aucun jeton écrit | M (revue du script) + gitleaks |
| MT6 | Limites de tir appliquées selon la voie (a) ou (b) seulement ; aucune route ni aucun en-tête de contournement ; limites de Keycloak inchangées | M (revue des valeurs « démonstration ») |
| MT7 | Rapports publiés (k6, ZAP, négatifs) relus avant commit : pas de jeton, `code`, `state`, adresse IP de Martin ni constat ouvert détaillé (MQ5) | M |
| MT8 | Si *remote write* : point d'écriture de Prometheus joignable seulement par `port-forward`, jamais par l'ALB | B (XE4/X17) |

**Verdict** : GO sous MT1 à MT8. MT1 et MT6 sont bloquants avant toute ouverture de l'ALB à Internet.

## Avis faisabilité

**Avis : faisable, sous trois conditions.** Les jetons des vendeurs sont préparés hors du tir et non obtenus en remplissant la page de connexion depuis k6. La voie (b) est retenue pour la limite de débit. Le scénario est répété au moins deux fois sur AWS avant le jour J. Effort propre à D13, au-delà de ce que D04 et D10 comptent déjà : **≈ 8 à 10 jours**.

**Faisabilité et effort par type.**

| Type | Faisabilité | Effort propre à D13 | Point d'attention |
|---|---|---|---|
| Unitaire back, fuzzing | Immédiate | Compté dans D04 | Le fuzzing tourne sans limite de temps : `-fuzztime=30s` en CI, campagne longue sur le poste seulement |
| Intégration testcontainers-go | Bonne : Docker est présent sur les exécuteurs Linux de GitHub | Compté dans D10 ; + 0,5 j pour ENF-12 (arrêt et relance du conteneur) | Un conteneur par paquet (`TestMain`), pas par test, sinon on dépasse les 4 à 6 min. Keycloak : celui de compose, pas un conteneur par test (30 à 60 s au démarrage) |
| Contrat JSON Schema, OpenAPI | Bonne : les deux bibliothèques sont mûres | ≈ 1 j (exemples dans `contracts/`, un test par paire) | Les exemples de `contracts/` sont la seule source : les deux côtés les relisent, aucun n'en a de copie |
| Acceptation API et E2E Playwright | Bonne | ≈ 1 j de socle (pile compose, fixtures d'authentification, matrice AC → test), puis ≈ 0,25 j par US | Le résultat asynchrone du Contrôle s'attend par interrogation répétée, avec un délai maximal (`expect.poll`), jamais par une pause fixe. La matrice se génère depuis les étiquettes des tests (`@US-xx-ACn`), sinon elle n'est plus à jour au bout d'un sprint |
| Accessibilité axe | Immédiate dans Playwright | ≈ 0,25 j | Exclure les règles de contraste sur les composants tiers seulement par une exception tracée |
| i18n | Immédiate | ≈ 0,25 j (script de parité) | Jouer les parcours en `fr` et en `en` double la durée de l'acceptation : `fr` en PR, les deux langues sur `main` |
| ZAP baseline | Compté dans D10 | — | Sur compose, la cible doit être le frontal, pas chaque service |
| Négatifs § 2 et MT1 à MT4 | Bonne | ≈ 2 j : 1 j en `go test` et Playwright, 1 j pour le script `curl` AWS (ME8 avec `--path-as-is`, X-Forwarded-For en valeur unique et en liste, 429 rejoué) | MT2 (rejeu du jeton de rafraîchissement) exige la rotation active dans le realm de compose, sinon le test passe pour une mauvaise raison |

**Flux PKCE scripté dans k6.** C'est techniquement possible : GET sur l'URL d'autorisation, extraction de l'`action` du formulaire, POST des identifiants, suivi de la redirection sans la suivre jusqu'au front, échange du `code`. Mais c'est **fragile**. Le script dépend du HTML du thème Keycloak, des paramètres `session_code` et `execution` et des cookies, et une montée de version de Keycloak le casse sans prévenir. Alternatives :
- **Jetons préparés hors du tir (recommandé)** : juste avant le tir, un script Playwright passe le vrai flux code + PKCE pour chaque vendeur de test et écrit les jetons d'accès et de rafraîchissement dans un fichier hors dépôt, lu par k6 (MT5). k6 n'appelle ensuite que le point de jeton en `refresh_token`, un simple POST sans HTML. Avec la rotation MA11, chaque VU garde **son** vendeur et **son** jeton de rafraîchissement : deux VU sur le même compte s'invalident mutuellement. Il faut donc autant de vendeurs de test que de VU pour la vague (10 à 20).
- **Client de test dédié, en dev seulement** : utile sur compose pour le profil court. Il est à exclure sur AWS : il diverge du chemin réel et le contrôle du realm (MA1) doit refuser sa présence dans le realm « démonstration ».

**Scénario de soutenance et calibrage.**
- **HPA** : avec une cible CPU de 60 %, une collecte de metrics-server toutes les 15 s, une boucle HPA toutes les 15 s et un démarrage de pod de 10 à 20 s (image en cache), le passage de 2 à 4 pods tient en 60 à 90 s. **X7 doit le mesurer** : sans mesure, la fenêtre de 3 min (de 2 à 5 min) est trop juste pour garantir la montée à l'écran. Il faut fixer des `requests` CPU réalistes, sinon le pourcentage n'a pas de sens.
- **Cluster Autoscaler** : l'ajout d'un nœud prend 2 à 4 min, donc il sort de la fenêtre de 8 min. Il faut le montrer seulement si X9 le mesure sous 2 min, sinon le mentionner sans l'attendre.
- **KEDA** : intervalle d'interrogation de 5 s et `cooldownPeriod` de 60 s, à régler dans les valeurs « démonstration ». Prévoir aussi une vague assez longue (≥ 60 s) pour que la file monte réellement.
- **Perte d'un pod** : atteindre < 1 % d'erreurs exige un `preStop` (pause de 5 à 10 s), un délai de désenregistrement de l'ALB réduit (≈ 15 s), une readiness probe juste et au moins 3 pods au moment de la suppression. Sans ces réglages, les requêtes en cours échouent et ENF-11 se joue au hasard.
- **Répétitions** : 2 sessions AWS (X7 + X10, puis répétition complète chronométrée) ≈ 1 j, plus le tournage de la vidéo en J7.

**Limite de débit pendant le tir : recommandation (b).** La voie (a) est **incompatible avec le scénario** : le tir a lieu pendant la soutenance, donc pendant que l'ALB est ouvert à Internet. Il faudrait alors refermer l'ALB pour tirer, ou tirer avec la limite nominale. Elle ajoute aussi deux manipulations manuelles le jour J, avec un risque d'oubli. La voie (b) reste stable pendant toute la session et se teste (une autre adresse reçoit 429, MT4). Point d'attention : l'adresse de Martin change selon le lieu (partage mobile en secours). Elle est donc injectée au `up` ou par une commande dédiée, jamais versionnée (MT7), et elle se vérifie à H − 20 min avec le test HTTPS.

**Remote write ou deux fenêtres : deux fenêtres.** Le *remote write* demande la réception activée sur Prometheus, un `port-forward` qui doit tenir 8 min sous charge (une coupure fait perdre la courbe en pleine démonstration) et un test de plus (MT8). Le gain est cosmétique. Le tableau de bord de k6 exporté en HTML suffit comme preuve.

**Profil court sur `main` : oui, non bloquant.** Il se branche dans le job d'acceptation, où la pile compose est déjà montée : 30 à 60 s à faible débit, seuils lâches, jetons obtenus par le client de test de compose. Coût ≈ 1 à 2 min de CI et 0,25 j. Il détecte un script cassé (API renommée, contrat changé) bien avant le jour J.

**Effort total propre à D13** : script k6 et préparation des jetons 1,5 à 2 j ; contrat 1 j ; socle d'acceptation et matrice 1 j ; axe et parité 0,5 j ; négatifs et MT1 à MT4 2 j ; ENF-12 0,5 j ; profil court 0,25 j ; calibrage et répétitions 1 à 1,5 j ; réglages de résilience (`preStop`, ALB, PDB) 0,5 j. Soit **≈ 8 à 10 j**, plus ≈ 0,25 j par US pour l'acceptation.

**Risques.**
- La durée de l'acceptation en CI (8 à 12 min) plus le démarrage de Keycloak rend la cible de 15 min par PR difficile à tenir. Parades : acceptation filtrée par chemins, `fr` seul en PR, cache des images.
- La montée HPA n'est pas garantie dans la fenêtre sans X7 : c'est le plus gros risque de C5. La règle de bascule vers la vidéo (D03) le couvre.
- Le réseau du lieu peut plafonner le débit depuis le poste. Calibrer le pic sur le partage mobile, pas sur le réseau de la maison.
- La rotation MA11 et le partage de comptes entre VU produisent des 401 parasites pendant la vague.
- Des tests d'acceptation instables, à cause de l'asynchrone, finissent ignorés. Un test instable doit donc être corrigé ou mis en quarantaine tracée, jamais relancé jusqu'à ce qu'il passe.

## Décision de Martin

> **« k6 + liste d'adresses »** : stratégie à 15 types de tests, complétée des tests négatifs MT1 à MT4 ; outil de charge k6 lancé depuis le poste à travers l'ALB, jetons préparés par Playwright juste avant le tir (fichier hors dépôt, un vendeur de test par utilisateur virtuel) ; deux fenêtres en soutenance (k6 et Grafana), sans *remote write* ; profil de charge court non bloquant sur `main`. Pendant le tir, la limite de débit n'est relevée que pour l'adresse de Martin, injectée à chaque session et jamais versionnée ; adresse et paramètres de connexion masqués dans les rapports publics (MT1 à MT8). Calibrage du HPA par X7, deux répétitions sur AWS avant la soutenance.

Date : 2026-10-26

## Conséquences

La décision de Martin est enregistrée ci-dessus ; ses conséquences :

- **Processus** : `docs/processus/tests.md` reprend les § 1 et 2 (types, outils, lieux, sorties, parties prenantes) ; `developpement.md` y renvoie déjà.
- **Code et dépôt** : `tests/acceptance/` (Playwright, API et navigateur) ; `tests/charge/` (script k6, profils « court », « démonstration ») ; exemples de messages dans `contracts/`.
- **US** : chaque US porte ses AC négatifs et un AC « Performance » quand elle touche la consultation ; la matrice AC → test sert la Definition of Done.
- **D10** : job de parité i18n en PR ; profil de charge court sur `main` si Martin le retient.
- **D12** : p95, taux d'erreurs, couverture et taux d'AC couverts deviennent des indicateurs mesurés par ces tests.
- **D14** : X7 et X10 utilisent le script k6 ; protocole du flux PKCE scripté.
- **D08, D09** : seuils de limitation de l'environnement de tir ; réception *remote write* seulement si retenue.
- **Soutenance** : `presentation/demo-script.md` reprend le scénario ; vidéo de secours tournée en J7.
- **Compétences (C9)** : k6 et Playwright entrent dans la cartographie.

## À savoir défendre en soutenance

1. **« Quels types de tests, et pourquoi ceux-là ? »**
   - Pyramide : unitaires et intégration sur vraie base et vrai broker, contrat, acceptation dérivée des AC, sécurité, charge, résilience.
   - Chaque type répond à une exigence nommée (ENF-23, ENF-17, ENF-14, ENF-11, ES-04) ; cinq types bloquent la fusion en CI (M14).
2. **« Pourquoi k6 plutôt que JMeter ou Gatling ? »**
   - Seuils natifs qui recopient les exigences, débit imposé par paliers, scripts relus comme du code, binaire léger.
   - Gatling a un meilleur rapport final mais une JVM et pas d'export Prometheus libre ; JMeter produit du XML illisible en revue.
3. **« Votre tir est-il représentatif ? »**
   - La consultation anonyme domine le trafic d'une marketplace ; le pic modélise une sortie de salon (D01).
   - Débit imposé, pas d'utilisateurs fermés : la saturation n'est pas masquée. Seuils fixés avant le tir.
4. **« Pourquoi ne pas lancer le tir depuis le cluster ? »**
   - Le générateur mangerait le CPU et les places des nœuds mesurés et fausserait le HPA. Depuis le poste, il emprunte le vrai chemin (TLS, ALB, limites).
5. **« Comment prouvez-vous la sécurité, au-delà des scanners ? »**
   - Chaque mesure d'autorisation a son test négatif (jeton, rôle, objet, canal, base, exposition), vert en CI ou après déploiement.

## Sources

Consultées le 2026-10-26.

- k6 : seuils https://grafana.com/docs/k6/latest/using-k6/thresholds/ ; exécuteur `ramping-arrival-rate` https://grafana.com/docs/k6/latest/using-k6/scenarios/executors/ramping-arrival-rate/ ; tableau de bord web https://grafana.com/docs/k6/latest/results-output/web-dashboard/ ; *remote write* Prometheus https://grafana.com/docs/k6/latest/results-output/real-time/prometheus-remote-write/ ; k6-operator https://grafana.com/docs/k6/latest/testing-guides/running-distributed-tests/
- Gatling : https://docs.gatling.io/ (SDK JavaScript : maturité à vérifier [?])
- Locust : https://docs.locust.io/
- Apache JMeter : https://jmeter.apache.org/
- Playwright, tests d'accessibilité : https://playwright.dev/docs/accessibility-testing ; axe-core : https://github.com/dequelabs/axe-core
- testcontainers-go : https://golang.testcontainers.org/
- ZAP baseline : https://www.zaproxy.org/docs/docker/baseline-scan/
- Décisions du projet : D01, D03 (X7, X9, X10, scénario et plan B), D04, D05, D06, D07, D08 (ME4), D09 (export des mesures), D10 (portes du pipeline).
