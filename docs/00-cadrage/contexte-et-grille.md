# Contexte et grille d'évaluation

Synthèse de travail des documents de la consigne. Les sources restent hors du dépôt, ne sont jamais modifiées et font foi en cas d'écart.
Exigences reformulées : [exigences.md](exigences.md). Règles du projet : [PILOTAGE.md](../PILOTAGE.md).

## 1. Sources

| Document | Version | Usage |
|---|---|---|
| `contexte_Collector.pdf` | V1 – 04/03/2025 | Cas d'entreprise et exigences métier |
| `Consignes eval bloc Superviser et Assurer Développment.pdf` | — | Travail demandé, livrables, soutenance |
| `Grille_eval_bloc_INFMAALSIA2V_Superviser_dev_apps_logicielles.xlsx` | — | Grille individuelle : **seule grille applicable** |
| `grille dévaluation Projet collaboratif superviser dev apps logicielles.xlsx` | — | Grille collective : non applicable (projet réalisé seul) |

## 2. Contexte de l'entreprise

### 2.1 Collector

- Start-up française de 5 ans, historiquement orientée **événementiel** : salons d'objets de collection « du quotidien » dans plusieurs villes de France (baskets en édition limitée, posters dédicacés, figurines originales, cassettes vidéo…).
- Partenariats existants : institutions (mairies), acteurs locaux (magasins spécialisés, friperies), particuliers collectionneurs.
- Levée de fonds par **financement participatif** pour lancer **Collector.shop**, application web de vente d'objets de collection **entre particuliers**.
- Segment : objets « vintage » rares ou uniques qui ravivent souvenirs et émotions. **Hors segment** : luxe et brocante classique.
- Modèle économique : **commission de 5 %** sur chaque transaction.

### 2.2 Organisation et équipe

| Fonction | Effectif | Remarque |
|---|---|---|
| Dirigeants | 2 | Spécialistes de l'événementiel, anciens chefs de projet IT en ESN |
| Administratif et RH | 1 | — |
| Communication et marketing | 1 | Spécialité marketing digital |
| Lead Developer | 1 | Rôle tenu par Martin (note 2 du contexte : « Vous pour la validation de bloc individuelle ») |
| Développeurs confirmés | 2 | 5 ans d'expérience, arrivés avec la levée de fonds |
| Recrutements | — | « De nouvelles embauches sont envisagées » |

L'équipe IT est nouvellement constituée : elle sert de base à la cartographie des compétences (C9).

### 2.3 Système d'information existant

- Locaux en **incubateur** ; accès internet par le **wifi partagé** de l'incubateur.
- Bureautique **Office 365 Business Standard**, messagerie **Exchange**, **Power BI**, licence **Adobe Creative Cloud**.
- Postes : ordinateurs portables **Windows 11** pour tous, un **Mac** pour la conception graphique.
- Web : un **site vitrine WordPress** hébergé chez un fournisseur français.
- Aucune application métier ni infrastructure d'hébergement applicatif : Collector.shop part de zéro.

### 2.4 Enjeux exprimés par la direction

- **Sécurité de premier plan** : l'application porte des transactions financières.
- **Évolutivité** : ajouter rapidement des fonctionnalités (enchères, ventes en streaming lors d'événements, bot avant-vente, analyse des ventes) pour « réagir à l'évolution du marché ».
- **Internationalisation** et **accessibilité**.
- **Intégration d'un outil de détection des fraudes**, développé en interne ou acheté (non décidé).
- **Automatisation** : contrôle des annonces et publicités ciblées sur des sites partenaires.
- Le sujet est fictif : des exigences peuvent être ajoutées pour répondre aux consignes ou les enrichir, en restant cohérent avec le contexte.

### 2.5 Cadre réel de l'exercice

- Martin réalise seul le projet ; **seule la grille individuelle compte** (10 critères, objectif 5/5 sur chacun).
- Hébergement : **AWS Academy Learner Lab**, crédits inférieurs à 100 $. L'environnement est allumé pour un usage précis puis éteint. Contraintes probables à vérifier : voir [PILOTAGE.md](../PILOTAGE.md) § 1.
- Soutenance individuelle : **20 minutes de présentation + 15 minutes d'échange**, dans plus de 6 semaines à la date du 2026-10-09 (date exacte à reporter ici).

## 3. Travail demandé

### 3.1 Compétences évaluées

| Bloc de compétences | Compétences |
|---|---|
| Élaborer le processus d'assurance qualité logicielle | Évaluer la qualité du développement · Définir la politique de tests · Définir la politique de sécurité |
| Piloter le développement et le déploiement d'applications | Superviser la mise en place d'une chaîne de livraison continue · Faciliter la montée en compétences des développeurs · Orchestrer la mise en production pour garantir la disponibilité et la montée en charge |
| Maintenir et développer son expertise en développement d'applications | Expérimenter des technologies innovantes en bac à sable · Démontrer la faisabilité par la réalisation d'un POC · Développer des applications complexes |

### 3.2 Mission

En tant que Lead Developer :

- définir un processus de développement respectant les exigences de qualité logicielle (appui possible sur ISO 25010) ;
- développer la **V1** avec un pipeline CI/CD et tests automatisés, un déploiement via un orchestrateur ou un fournisseur cloud, une sécurité minimale (HTTPS, authentification/autorisation, scan de vulnérabilités) — toutes les fonctionnalités ne sont pas attendues, **au minimum une fonctionnalité métier** ;
- analyser la sécurité de la V1 et proposer un plan de remédiation.

| Phase | Contenu |
|---|---|
| 1. Structuration du processus | Indicateurs qualité, cycle de vie DevSecOps, CI/CD, tests, compétences et formation |
| 2. Développement et déploiement | Exigences, architecture, expérimentations en bac à sable, POC |
| 3. Plan de remédiation | Analyse sécurité de la V1, préconisations justifiées |

### 3.3 Exigences minimales de la consigne

| # | Exigence minimale | Source (consigne) | Critères |
|---|---|---|---|
| M1 | Processus de développement conforme aux exigences de qualité (performance, fiabilité, disponibilité, flexibilité, capacité fonctionnelle…) | § 1 et 1.1 | C6, C7 |
| M2 | **4 indicateurs** de qualité, avec la justification de leur rôle contre l'accumulation de dette technique | § 1.1.1 | C6 |
| M3 | Cycle de vie formalisé en **DevSecOps**, avec les mesures de sécurité de chaque étape clé | § 1.1.2 | C4, C8 |
| M4 | **Schéma détaillé** du processus CI/CD | § 1.1.2 | C4 |
| M5 | Tests automatisés dans le pipeline : types, outils, lien avec le suivi des indicateurs | § 1.1.2 | C4, C6, C7 |
| M6 | Cartographie des compétences et **au moins une action de formation**, avec des profils réalistes | § 1.1.3 | C9 |
| M7 | Exigences fonctionnelles reformulées ; fonctionnalité du POC décrite en **backlog (US + critères d'acceptation)** | § 1.2.1 | C3 |
| M8 | **Schéma d'architecture technique** intégrant la sécurité et les solutions retenues (base de données, protocole, framework…) | § 1.2.1 | C2, C10 |
| M9 | **Expérimentation en bac à sable** des technologies phares : environnement, étapes reproductibles, difficultés, limites, résultats justifiant l'adoption ou le rejet | § 1.2.2 | C1 |
| M10 | **Au moins une fonctionnalité métier** implémentée selon le backlog, prouvée par des tests d'appels voire des tests d'acceptation | § 1.2.3 | C3, C7 |
| M11 | Composants conformes à l'architecture et aux solutions techniques retenues | § 1.2.3 | C2 |
| M12 | Sécurité minimale : **HTTPS/TLS**, **authentification et autorisation**, **détection des vulnérabilités** | § 1 et 1.2.3 | C8, C2 |
| M13 | **Au moins une composante d'observabilité** : logs centralisés, collecte de métriques ou traces distribuées | § 1.2.3 | C5, C6 |
| M14 | Pipeline CI/CD avec les principales étapes et **au moins 2 types de tests** permettant d'évaluer les métriques | § 1.2.3 | C4, C7, C6 |
| M15 | Déploiement via un **orchestrateur ou un fournisseur cloud** | § 1 | C5, C2 |
| M16 | **Test de charge en soutenance** pour présenter la montée en charge (intégration au pipeline facultative) | § 1.2.3 | C5, C10 |
| M17 | Audit sécurité de la V1 : analyse des résultats de tests et des métriques, vulnérabilités potentielles (appui possible sur les tests de charge) | § 1.3 | C8 |
| M18 | **Plan de remédiation** adapté au contexte, recommandations justifiées | § 1.3 | C8 |
| M19 | Soutenance 20 + 15 min, démonstration préparée, vocabulaire précis | Conseils | C10 |

### 3.4 Tolérances et conseils de la consigne

- Pas de pénalité si le pipeline ne peut pas mettre en œuvre l'étape de déploiement en (pré)production.
- Les tests d'acceptation peuvent compter parmi les deux types de tests minimum du pipeline.
- Compétences : pas de contrainte budgétaire ni de limite d'effectif pour l'équipe fictive, mais des profils réalistes (pas de « moutons à 5 pattes »).
- Évaluation technique : le jury est composé de spécialistes, la précision des termes est exigée (« pas de "je sais faire mais pas expliqué" »).
- Schémas lisibles et complets ; la forme du support de présentation n'est pas évaluée.
- Démonstration préparée pour limiter « l'effet démo » ; une **vidéo de secours** est autorisée.

### 3.5 Expérimentations en bac à sable

**Non valides** selon la consigne :

- le test d'un langage de programmation ;
- la communication (REST, gRPC, WebSocket, etc.) entre un composant front-end et un composant back-end ;
- l'exécution locale de tests unitaires ;
- l'implémentation d'un composant avec un framework (MVC, ORM, service REST, etc.).

**Exemples valides** cités : plateforme CI/CD intégrant qualité de code et frameworks de tests ; service mesh ; services d'infrastructure (API gateway, résilience, découverte de services) ; broker de messages déployé dans un orchestrateur avec tests de publication et consommation ; provisionnement de certificats sécurisant un point d'entrée ; clustering d'un SGBD ; observabilité ; technologie serverless ; traitement distribué sur plusieurs nœuds.

Par « technologie », la consigne entend la **plateforme support** des composants développés (middleware orienté messages, système de sécurité, SGBD, observabilité, orchestrateur, services d'un fournisseur cloud…).

## 4. Grille d'évaluation individuelle

### 4.1 Barème

- Chaque critère est noté **5, 4, 2 ou 1 point** (pas de 3 ni de 0). La note du bloc est la moyenne des 10 critères.
- Conversion : **A** si moyenne ≥ 4,6 · **B** si ≥ 3,6 et < 4,6 · **C** si ≥ 1,6 et < 3,6 · **D** si < 1,6.
- A et B valent « Acquis » (A : objectifs atteints ; B : écarts mineurs). C et D valent « Non acquis » (écarts majeurs ou critiques).
- Objectif du projet : 5/5 sur chaque critère. Pour mémoire, la mention A tolère quelques notes à 4 (exemple : six critères à 5 et quatre à 4 donnent 4,6).

### 4.2 Les 10 critères — descripteur du niveau 5

Descripteurs recopiés mot pour mot (corrections typographiques signalées en § 4.3). La ligne « Écart à éviter » résume le niveau 4 : c'est ce qui fait perdre le point.

#### C1 — Fiabilité du protocole d'expérimentation

> Le protocole d’expérimentation en bac à sable est présenté.
>
> Il comprend les technologies et plateformes testées ainsi que leurs interactions.
>
> Les difficultés rencontrées et les résultats obtenus sont présentés et permettent de valider l’adoption des technologies.

Écart à éviter : protocole imprécis, étapes ou intégrations omises, expérimentation non reproductible intégralement, résultats qui ne justifient pas totalement l'adoption.

#### C2 — Qualité technique du prototype

> L’implémentation technique du POC valide la faisabilité des points clés de l’architecture de la solution applicative à développer (découpage applicatif, communication interservice, sécurité, hébergement, orchestration, etc.), elle est documentée.

Écart à éviter : certains composants non conformes à l'architecture proposée ; documentation incomplète rendant l'exécution difficile.

#### C3 — Démonstration de fonctionnalité métier

> La fonctionnalité métier développée au niveau du POC répond aux besoins fonctionnels exprimés par la maîtrise d’ouvrage

Écart à éviter : fonctionnalité incomplète ou manquant de pertinence par rapport au domaine métier.

#### C4 — Conformité du processus CI/CD

> Le processus de livraison continue est schématisé. Il est conforme aux exigences de développement et de mise en production.
>
> Le POC intègre la plateforme de livraison continue conformément au processus décrit.

Écart à éviter : étapes ou enchaînements insuffisamment détaillés ; plateforme CI/CD du POC limitée à une seule étape.

#### C5 — Validation de l'environnement d'exécution / de déploiement en production

> Le POC s’exécute dans un environnement managé.
>
> La disponibilité et la montée en charge de l’application sont démontrées

Écart à éviter : environnement managé sans démonstration de la disponibilité ni de la montée en charge.

#### C6 — Pertinence et mesure des indicateurs de qualité logicielle

> Au moins 4 indicateurs sont définis. Ils servent les objectifs de qualité logicielle (fonctionnalité, performance, maintenabilité, fiabilité).
>
> Leur mesure met en évidence des axes d’amélioration pouvant réduire la dette technique.

Écart à éviter : 3 indicateurs seulement, ou des mesures qui ne font pas ressortir d'axes d'amélioration de la dette technique.

#### C7 — Conformité du processus de test

> Le processus de test est formalisé. Il présente les différents types de tests, les outils de tests associés (unitaires, intégration, non régression, performances, fonctionnels, conformité réglementaire, etc.) et les parties prenantes.
>
> Au moins deux types de tests sont appliqués au POC. Ils sont exécutés avec succès.

Écart à éviter : processus incomplet (types de tests, outils, sorties attendues) ; un seul type de test, ou une couverture trop limitée.

#### C8 — Validation de la remédiation des risques de sécurité

> Le plan de remédiation priorise les actions pour réduire les risques de sécurité critiques du parc applicatif du SI.
>
> Le POC intègre au moins deux bonnes pratiques pour sécuriser l’application.

Écart à éviter : justification incomplète de la priorisation ; une seule bonne pratique intégrée au POC.

#### C9 — Cohérence de la gestion des compétences de l'équipe

> Les compétences de l’équipe sont recensées.
>
> Les expertises techniques à acquérir sont identifiées.
>
> Au moins une action de formation est proposée.

Écart à éviter : action de formation réduite à un intitulé. Pour l'éviter, la détailler (objectifs, public, modalités, durée, évaluation).

#### C10 — Qualité de la présentation

> Les schémas et tableaux sont lisibles et adaptés à la présentation orale.
>
> La démonstration du POC est préparée.
>
> La prestation orale est fluide et dynamique. Le discours est structuré. Le vocabulaire est précis et adapté.

Écart à éviter : adaptation des supports à l'oral perfectible ; prestation compréhensible mais hésitante.

### 4.3 Corrections apportées à la source

Seules des fautes de frappe évidentes ont été corrigées ; le sens est inchangé.

| Emplacement | Source | Corrigé |
|---|---|---|
| Titre C1 | « d'experimentation » | « d'expérimentation » |
| Titre C8 | « 8.Validation … sécurité. » | « 8. Validation … sécurité » |
| Titre C9 | « Cohérence de le gestion » | « Cohérence de la gestion » |
| Niveau 5 de C10 | « lisibles␣␣et adaptés à la␣␣présentation » (espaces doubles) | espaces simples |
| Niveau 5 de C10 | « 'La démonstration » (apostrophe parasite) | « La démonstration » |
| Niveau 5 de C10 | « la prestation orale … dynamique.␣␣le discours » | « La prestation orale … dynamique. Le discours » |

## 5. Matrice critère → livrables → fichiers → jalon

Jalons : voir [PILOTAGE.md](../PILOTAGE.md) § 1. Décisions : voir [INDEX.md](../decisions/INDEX.md).

| Critère | Livrables attendus | Fichiers du dépôt | Décisions | Jalons |
|---|---|---|---|---|
| C1 Protocole d'expérimentation | Un protocole par technologie phare : objectif et critères de succès fixés avant, environnement et versions, technologies et interactions (schéma), étapes reproductibles, résultats, difficultés, limites, verdict d'adoption ou de rejet | `docs/experimentations/XPnn-*.md`, `sandbox/XPnn-*/`, `docs/architecture/schemas/` | D14 ; D03, D05, D06, D09, D10 en `Candidate` | J2 (périmètre), J3 (exécution) |
| C2 Qualité technique du POC | POC déployé validant découpage, communication interservice, sécurité, hébergement, orchestration ; architecture documentée ; documentation d'exécution | `services/`, `web/`, `infra/`, `docs/architecture/architecture-technique.md`, `docs/architecture/schemas/`, `docs/exploitation.md`, `README.md` | D02 à D08, D11 | J2, J4, J5 |
| C3 Fonctionnalité métier | Exigences reformulées ; fonctionnalité choisie ; backlog (US + AC) ; démonstration du parcours | `docs/00-cadrage/exigences.md`, `docs/decisions/D01-*.md`, `backlog/US-xxx-*.md`, `tests/acceptance/`, `presentation/demo-script.md` | D01 | J1, J5 |
| C4 Processus CI/CD | Schéma détaillé du processus (étapes, enchaînements, portes qualité et sécurité) ; pipeline du POC conforme au schéma | `docs/processus/cicd.md`, `docs/processus/devsecops.md`, `docs/architecture/schemas/`, définition du pipeline (emplacement selon D10) | D10 | J3 (schéma), J4 (pipeline) |
| C5 Environnement managé, disponibilité, charge | POC en environnement managé ; disponibilité démontrée ; montée en charge démontrée par test de charge en soutenance ; allumage et extinction maîtrisés | `infra/`, `tools/`, `tests/charge/`, `docs/mesures/charge/`, `docs/exploitation.md` | D03, D09, D11, D13 | J4, J5, J7 |
| C6 Indicateurs qualité | Au moins 4 indicateurs justifiés ; mesures par sprint ; axes d'amélioration de la dette | `docs/processus/qualite-indicateurs.md`, `docs/mesures/indicateurs-sprint-N.md`, `docs/mesures/analyse-dette.md` | D12 (avec D09, D10) | J3 (définition), J4–J5 (mesures), J6 (analyse) |
| C7 Processus de test | Processus formalisé : types de tests, outils, sorties attendues, parties prenantes ; au moins 2 types de tests verts sur le POC | `docs/processus/tests.md`, `tests/acceptance/`, `tests/charge/`, tests des `services/` et du `web/`, rapports de CI | D13, D10 | J3 (processus), J4–J5 (exécution) |
| C8 Remédiation sécurité | Exigences de sécurité ; au moins 2 bonnes pratiques intégrées au POC ; audit de la V1 ; plan de remédiation priorisé et justifié sur le **parc applicatif du SI** | `docs/securite/exigences-securite.md`, `docs/processus/devsecops.md`, `docs/securite/audit-v1.md`, `docs/securite/plan-remediation.md`, `docs/audits/sprint-N-securite.md` | D06, D08, D10 | J3 (exigences), J4–J5 (pratiques), J6 (audit, plan) |
| C9 Compétences | Compétences recensées ; expertises à acquérir ; action de formation détaillée | `docs/processus/competences-formation.md` | Impact de D04 | J6 |
| C10 Présentation | Plan de soutenance ; script de démonstration ; schémas lisibles ; réponses préparées ; vidéo de secours | `presentation/plan-soutenance.md`, `presentation/demo-script.md`, `presentation/faq-architecture.md`, `presentation/faq-securite.md`, `docs/architecture/schemas/` | Section « À savoir défendre » de chaque décision | J7 |

La vidéo de secours est conservée hors du dépôt : les fichiers vidéo sont exclus par le `.gitignore`.

## 6. Points d'attention relevés dans les sources

1. **C8 vise le « parc applicatif du SI »**, pas seulement le POC : le plan de remédiation doit aussi couvrir l'existant (site vitrine, messagerie et bureautique, postes, wifi partagé de l'incubateur).
2. **C2 cite la communication interservice** : le POC doit comporter au moins deux composants qui communiquent, sinon ce point clé n'est pas validé.
3. **C5 exige des démonstrations**, pas seulement un déploiement : la disponibilité (résistance à la perte d'une instance) et la montée en charge (test de charge) sont à montrer en soutenance.
4. **Tolérance sur le déploiement automatisé** (§ 3.4) : elle ne dispense pas de C5, qui exige un environnement managé, ni de C4, qui attend un processus « conforme aux exigences de développement et de mise en production ».
5. **C7 cite les « parties prenantes »** du processus de test : elles sont à nommer dans `docs/processus/tests.md`.
6. **Budget** : la consigne n'impose aucune contrainte budgétaire à l'équipe fictive (compétences, formation) ; la limite de crédits AWS s'applique, elle, au POC réel.
7. **Rôles utilisateurs** : le contexte annonce « 2 profils utilisateurs » puis décrit un rôle admin et un parcours de catalogue sans authentification. Quatre rôles sont retenus dans [exigences.md](exigences.md).
8. **Coquilles du contexte** sans effet sur le sens : « contrôle de the Collector », « Hasbrow », « prix trop élevée ».
