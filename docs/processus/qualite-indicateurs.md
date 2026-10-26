# Indicateurs de qualité logicielle

Ce document définit les sept indicateurs de qualité de Collector, leur mesure et leur relevé à chaque sprint.
Il répond au livrable M2 de la consigne ([contexte-et-grille.md](../00-cadrage/contexte-et-grille.md) § 3.3) et sert le critère C6 : au moins quatre indicateurs, au service des objectifs de qualité, dont la mesure désigne des axes de réduction de la dette technique.

Le choix des indicateurs, les options comparées et les avis sont dans [D12](../decisions/D12-indicateurs-qualite.md). Les portes du pipeline sont dans [cicd.md](cicd.md), les seuils de vulnérabilité dans [devsecops.md](devsecops.md) § 3, la gestion de la dette dans [developpement.md](developpement.md) § 6.

## 1. Principes

- **Un indicateur par objectif de C6, au minimum** : fonctionnalité (I1), performance (I2), fiabilité (I3), maintenabilité (I4, I5). I6 (sécurité) et I7 (livraison) complètent. Si I2 et I3 manquent un sprint (session AWS ratée), cinq indicateurs restent mesurés.
- **Chaque mesure désigne où agir** : un AC, une route, un service, une fonction, une exception. Un chiffre global sans élément nommé ne compte pas comme mesure.
- **Mesuré par les outils qui bloquent déjà** : aucun service tiers, aucun secret. Le linter qui bloque est celui qui mesure.
- **Relevé généré, jamais recopié** : le fichier du sprint sort d'un programme, la tendance se calcule sur le JSON du sprint précédent.
- **« Non mesuré » n'est pas zéro** : une valeur absente est affichée comme telle et ne fait pas varier la tendance.
- **Seuils de départ calibrés** sur la première mesure du sprint 1 ; tout changement de seuil est noté dans le relevé du sprint où il intervient.

## 2. Fiches des indicateurs

Lecture : B = porte bloquante (pas de fusion ou pas de clôture) ; NB = non bloquant, l'alerte ouvre une dette `DT-nn`.

### I1 — AC couverts par un test vert (aptitude fonctionnelle ; ENF-23, ENF-01)

- **Formule** : AC reliés à un test d'acceptation vert ÷ AC des US terminées du sprint. Sous-mesure publiée à part : AC de sécurité couverts ÷ AC de sécurité.
- **Outil et source** : tests nommés `US-xxx-ACn` ; JUnit de `gotestsum` (Go) et de Playwright, croisé avec les AC de `backlog/US-*.md`. Lu sur `main`, où tous les tests s'exécutent.
- **Fréquence** : chaque exécution sur `main` ; relevé en fin de sprint.
- **Cible** : 100 %. **Alerte** : < 100 %. **Porte** : B, une US sous 100 % n'est pas terminée (DoD).
- **Dette détectée** : comportement livré sans preuve, régression silencieuse, refus d'accès non testé.
- **Action** : l'AC non couvert est nommé ; l'US repasse à l'étape de développement. Une US au format d'AC illisible fait échouer le programme de relevé.

### I2 — Latence et mise à l'échelle sous charge (efficacité de performance ; ENF-14 à ENF-16)

- **Formule** : `histogram_quantile(0.95, …)` par route du catalogue ; débit ; délai entre le changement du nombre de réplicas désirés et `kube_pod_status_ready_time` du nouveau pod.
- **Outil et source** : Prometheus et Grafana (D09), pendant le tir de charge (D13) ; histogramme avec bornes 0,4 s et 0,5 s ajoutées ; export dans `docs/mesures/charge/`.
- **Fréquence** : un tir par sprint à partir du sprint 2.
- **Cible** : p95 < 500 ms ; délai ≤ 90 s. **Alerte** : p95 > 400 ms ou hausse > 20 % d'un sprint à l'autre. **Porte** : NB pour le pipeline ; bloque l'US qui porte l'AC « Performance ».
- **Dette détectée** : requête N+1, index manquant, sérialisation coûteuse, invisibles hors charge.
- **Action** : la route la plus lente, comparée au sprint précédent, fait l'objet d'une `DT-nn` avec sa requête en cause.

### I3 — Fiabilité sous perte d'instance (fiabilité ; ENF-11, ENF-12)

- **Formule** : 5xx ÷ requêtes sur la fenêtre du tir avec suppression d'un pod ; événements perdus = lignes d'outbox publiées − lignes traitées côté consommateur ; entrées dans les files `.ecartes`.
- **Outil et source** : Prometheus pour les 5xx et les files (métriques par file de RabbitMQ) ; requête SQL lancée après le tir pour les événements perdus, jamais une différence de compteurs Prometheus.
- **Fréquence** : à chaque tir, à partir du sprint 2.
- **Cible** : < 1 % ; 0 perdu ; 0 écarté hors cas de test. **Alerte** : ≥ 0,5 % ; tout événement perdu. **Porte** : NB pour le pipeline ; un événement perdu bloque la clôture du sprint.
- **Dette détectée** : point unique de défaillance, reprise non idempotente, message mal formé.
- **Action** : le service qui renvoie des erreurs à la perte d'un pod, ou la file qui écarte, est nommé ; dette à impact fiabilité, engagée en priorité.

### I4 — Couverture de tests par service (maintenabilité ; ENF-21, ENF-24)

- **Formule** : lignes couvertes ÷ lignes exécutables, par service Go et pour le front ; code généré et `main` exclus par une liste versionnée.
- **Outil et source** : `go test -coverprofile` puis `go tool cover -func` ; Vitest, couverture v8, rapporteur `json-summary`. Référence : `couverture-reference.json` versionné.
- **Fréquence** : chaque PR ; relevé en fin de sprint.
- **Cible** : ≥ 70 % sur les paquets métier (indicatif). **Alerte et porte** : sprint 1, publiée sans porte ; dès le sprint 2, B si un service baisse de plus de 0,5 point par rapport à la référence.
- **Dette détectée** : code non exercé, donc risqué à modifier.
- **Action** : le service et le paquet en baisse sont nommés ; la PR ajoute les tests ou documente la suppression de code couvert. La référence est mise à jour à la revue de sprint, par une PR.

### I5 — Complexité cognitive et duplication (maintenabilité ; ENF-21, ENF-22)

- **Formule** : nombre de fonctions de complexité cognitive > 15 ; part de lignes dupliquées. Code généré (clients OpenAPI) et tests exclus.
- **Outil et source** : `gocognit` dans golangci-lint (`new-from-merge-base` pour la porte, seconde passe JSON pour le comptage global) ; `sonarjs/cognitive-complexity` dans ESLint, sur les fichiers modifiés ; jscpd pour Go et TypeScript.
- **Fréquence** : chaque PR ; relevé en fin de sprint.
- **Cible** : 0 fonction > 15 dans le code nouveau ; duplication < 3 %. **Alerte** : toute hausse d'un sprint à l'autre. **Porte** : B pour la complexité du code nouveau ; NB pour la duplication.
- **Dette détectée** : fonctions trop complexes à relire, logique copiée en plusieurs endroits.
- **Action** : la fonction et le bloc dupliqué sont nommés par fichier ; découpage ou factorisation planifiés en `DT-nn`.

### I6 — Vulnérabilités et exceptions (sécurité ; ENF-05)

- **Formule** : vulnérabilités ouvertes par sévérité et par service, selon les seuils de [devsecops.md](devsecops.md) § 3 (critique ou élevée corrigeable ; Go atteignable, catégorie à part) ; secrets détectés ; exceptions avec leur âge et leurs renouvellements ; vulnérabilités ouvertes au-delà du délai P1 ou P2.
- **Outil et source** : SARIF et JSON de govulncheck, Trivy, npm audit, OSV-Scanner, gosec, Semgrep, gitleaks ; `.trivyignore` (échéance `exp:`, champs contrôlés par `tools/ci/verifier-exceptions.sh`) ; exceptions Semgrep en ligne désactivées (`--disable-nosem`) ; état de première détection tenu dans le registre de l'audit de sécurité.
- **Fréquence** : chaque PR et analyse quotidienne de `main`.
- **Cible** : 0 critique ou élevée corrigeable, 0 Go atteignable, 0 exception échue. **Alerte** : modérées en hausse ; délai P1 ou P2 dépassé. **Porte** : B pour les seuils de § 3 et pour toute exception échue ; un `nosemgrep` sans entrée datée fait échouer le contrôle.
- **Dette détectée** : dépendance vieillissante, exception acceptée puis oubliée.
- **Action** : montée de version ou solde de l'exception ; le détail reste au registre de l'audit (§ 4).

### I7 — Livraison (complément, DORA réduit)

- **Formule** : exécutions vertes ÷ exécutions terminées sur `main` ; médiane « ouverture de PR → image publiée sur `main` » ; médiane de la durée du pipeline ; nombre de relances.
- **Outil et source** : `gh run list --branch main --event push` et `gh pr list --state merged`, la PR étant reliée au run par son commit de fusion.
- **Fréquence** : relevé en fin de sprint.
- **Cible** : ≥ 90 % ; < 1 jour ; < 15 min. **Alerte** : < 80 % ; durée > 20 min ; relances en hausse. **Porte** : NB.
- **Dette détectée** : pipeline lent ou instable, qui pousse à contourner les portes.
- **Action** : l'étape la plus lente et le test instable sont nommés. Fréquence de déploiement et temps de rétablissement sont reportés à la V1, faute de production continue.

## 3. Relevé par sprint

| Étape | Qui | Quand | Produit |
|---|---|---|---|
| 1. Vérifier que le dernier run `push` sur `main` est terminé | DevOps | Fin de sprint, après la dernière fusion | Run de référence |
| 2. Tir de charge avec suppression d'un pod (sprint ≥ 2) | DevOps, test | Session AWS de fin de sprint | Export dans `docs/mesures/charge/`, requête SQL d'événements perdus |
| 3. Générer le relevé : `tools/indicateurs` lit le `qualite.json` du run de référence, l'export du tir et l'API GitHub | DevOps | Après le tir | `docs/mesures/indicateurs-sprint-N.md` et `indicateurs-sprint-N.json` |
| 4. Relire le fichier avant fusion (règle de publication, § 4) | Sécurité | Avant la PR du relevé | Relecture notée dans la PR |
| 5. Inscrire une `DT-nn` par alerte, avec l'élément nommé | Lead Developer | Avant la rétrospective | Section « Dette technique » de `sprint-N.md` |
| 6. Mettre à jour `couverture-reference.json` | Lead Developer | Revue de sprint | PR dédiée |

- **Tendance** : calculée sur `indicateurs-sprint-(N-1).json`, jamais en relisant du Markdown. Elle est qualifiée de favorable, stable ou défavorable selon le sens de l'indicateur ; une tendance défavorable sur deux sprints consécutifs est une alerte, même sous le seuil.
- **Sources fiables seulement** : le programme ne lit que les artefacts d'exécutions `push` sur `main`, jamais d'une PR de fork ; `qualite.json` est validé par un schéma, jamais exécuté ni interpolé dans une commande.
- **Premiers relevés** : sprint 1 pour I1, I4 à I7 ; sprint 2 pour le relevé complet.

## 4. Règle de publication

Le dépôt est public : résumés de CI, artefacts et fichiers de `docs/mesures/` sont lisibles par tous.

- I6 ne publie que des **compteurs** par sévérité et par service, le nombre et l'âge des exceptions.
- Aucun identifiant de vulnérabilité ouverte, nom de composant vulnérable ni chemin atteignable dans le résumé de CI, `qualite.json` ou le fichier du sprint.
- Le détail reste dans le registre de l'audit de sécurité ; il n'est publié qu'une fois la faille corrigée.
- Les « trois pires éléments » de I6 sont donc remplacés par les services concernés et leurs compteurs.

## 5. Des indicateurs à l'analyse de la dette (J6)

`docs/mesures/analyse-dette.md` consolide les relevés de tous les sprints :

1. **Série par indicateur** : valeurs de chaque sprint lues dans les `indicateurs-sprint-N.json`, avec la tendance d'ensemble.
2. **Axes d'amélioration** : pour chaque indicateur, les éléments revenus plusieurs sprints parmi les pires (route, service, fonction, bloc dupliqué, AC).
3. **Dette tracée** : chaque `DT-nn` née d'une alerte, son statut (remboursée avec date et US, ou restante) et la part de capacité réellement consacrée au remboursement (cible 20 %).
4. **Effet mesuré** : pour chaque dette remboursée, la valeur de l'indicateur avant et après.
5. **Actions retenues** : dette restante priorisée (sécurité et fiabilité d'abord) et inscrite au backlog ; elle alimente le plan de remédiation.

## 6. Gabarit du relevé de sprint

Généré par `tools/indicateurs` ; seules les sections « Analyse » et « Dettes ouvertes » sont rédigées à la main.

```markdown
# Indicateurs du sprint N

Run de référence : <lien du run `push` sur main> · Tir : <lien de l'export ou « non mesuré »> · Date : AAAA-MM-JJ

| # | Indicateur | Valeur | Cible | Sprint N-1 | Tendance | État |
|---|---|---|---|---|---|---|
| I1 | AC couverts (dont sécurité) | | 100 % | | | OK / alerte / non mesuré |
| I2 | p95 catalogue ; délai de mise à l'échelle | | < 500 ms ; ≤ 90 s | | | |
| I3 | 5xx sous perte de pod ; perdus ; écartés | | < 1 % ; 0 ; 0 | | | |
| I4 | Couverture par service (min.) | | pas de baisse > 0,5 pt | | | |
| I5 | Fonctions > 15 ; duplication | | 0 nouvelle ; < 3 % | | | |
| I6 | Ouvertes par sévérité ; exceptions ; hors délai | | 0 ; 0 échue ; 0 | | | |
| I7 | Réussite ; délai de livraison ; durée ; relances | | ≥ 90 % ; < 1 j ; < 15 min | | | |

## Trois pires éléments par indicateur

| # | Élément 1 | Élément 2 | Élément 3 |
|---|---|---|---|
| I1 | AC non couvert | | |
| I2 | route, p95 | | |
| I3 | service ou file | | |
| I4 | service, paquet, écart | | |
| I5 | fichier, fonction ou bloc | | |
| I6 | service, compteurs seuls | | |
| I7 | étape ou test instable | | |

## Changements de seuil ou de périmètre

## Analyse

## Dettes ouvertes (`DT-nn`)
```
