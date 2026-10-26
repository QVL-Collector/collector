# Cartographie des compétences et plan de formation

| Champ | Valeur |
|---|---|
| Critère grille | C9 — Cohérence de la gestion des compétences de l'équipe |
| Version | 0.1 — première version, 26/10/2026 |
| Revue prévue | J6 (audit, remédiation, dette, compétences) : niveaux réévalués d'après le POC réalisé |
| Sources | [contexte-et-grille.md](../00-cadrage/contexte-et-grille.md) § 2.2 ; [INDEX.md](../decisions/INDEX.md) ; [D04](../decisions/D04-stack.md) |

## 1. Périmètre et hypothèses

L'équipe IT de Collector.shop est nouvellement constituée (contexte § 2.2) : un lead developer et deux développeurs confirmés (5 ans d'expérience), des embauches étant envisagées. La consigne ne nomme pas les deux développeurs confirmés. D04 indique seulement que l'expertise de l'équipe couvre Go et Rust au back, React, TypeScript et MUI au front.

Hypothèses retenues, explicites et à confirmer :

| Profil | Statut | Hypothèse |
|---|---|---|
| Lead developer (Martin) | Réel | Compétences déclarées : Java/Kotlin (Spring), TypeScript (Node), Go, Rust ; Angular, Vue ; pas React |
| Développeur confirmé back | Hypothétique | 5 ans, Go et Rust, API REST, PostgreSQL ; Docker en usage courant, pas d'exploitation Kubernetes |
| Développeur confirmé front | Hypothétique | 5 ans, React + TypeScript, MUI, tests de composants ; peu d'exposition au back et au cloud |
| Embauche envisagée « plateforme » | Hypothétique, à recruter | Profil DevOps/SRE junior à confirmé : AWS, Kubernetes, Terraform, CI/CD |

Pas de profil « mouton à 5 pattes » : chaque personne a un cœur de compétence et des zones d'apprentissage.

## 2. Échelle de niveaux

| Niveau | Libellé | Signification |
|---|---|---|
| 0 | Aucun | Jamais pratiqué |
| 1 | Notions | Connaît les concepts, ne produit pas seul |
| 2 | Pratique encadrée | Produit avec relecture et appui |
| 3 | Autonome | Produit et dépanne seul, explique ses choix |
| 4 | Référent | Forme les autres, tranche les choix, défend à l'oral |

## 3. Compétences requises par la stack

Chaque compétence découle d'une décision : elle n'est requise que parce qu'un choix l'impose.

| Domaine | Compétence | Décision source |
|---|---|---|
| Back | Go (services HTTP, tests, concurrence) | D04 |
| Back | PostgreSQL (rôles, transactions, mise à jour conditionnelle) | D07 |
| Back | Messagerie RabbitMQ (outbox, idempotence, files en échec) | D05 |
| Front | React + TypeScript strict (hooks, effets, composants contrôlés) | D04 |
| Front | MUI, i18n (react-i18next), accessibilité | D04 |
| Sécurité | OIDC + PKCE, Keycloak, audiences par service | D06 |
| Sécurité | Pratiques OWASP, analyse de vulnérabilités | D06, D10 (à venir) |
| Plateforme | Conteneurs (Docker, images minimales) | D03 |
| Plateforme | Kubernetes et Amazon EKS (déploiement, HPA, KEDA, sondes) | D03 |
| Plateforme | AWS (RDS, S3, IAM, contraintes du Learner Lab) | D03, D07 |
| Plateforme | Terraform (IaC, allumage et extinction) | D11 (à venir) |
| Plateforme | CI/CD et portes DevSecOps | D10 (à venir) |
| Qualité | Observabilité (logs, métriques, traces) | D09 (à venir) |
| Qualité | Tests (unitaires, parcours, charge) | D13 (à venir) |

## 4. Matrice des compétences

Format : niveau actuel → niveau cible. Les niveaux des profils hypothétiques sont des estimations.

| Compétence | Lead (Martin) | Confirmé back | Confirmé front | Embauche plateforme |
|---|---|---|---|---|
| Go | 3 → 4 | 3 → 4 | 0 → 1 | 1 → 2 |
| PostgreSQL | 3 → 3 | 3 → 3 | 1 → 1 | 2 → 2 |
| RabbitMQ (outbox, idempotence) | 2 → 3 | 1 → 3 | 0 → 1 | 2 → 3 |
| React + TypeScript | 1 → 3 | 0 → 1 | 4 → 4 | 0 → 0 |
| MUI, i18n, accessibilité | 1 → 2 | 0 → 0 | 3 → 4 | 0 → 0 |
| OIDC, Keycloak | 2 → 3 | 1 → 3 | 1 → 3 | 1 → 2 |
| Pratiques OWASP | 2 → 3 | 2 → 3 | 2 → 3 | 2 → 3 |
| Docker | 3 → 3 | 2 → 3 | 1 → 2 | 3 → 4 |
| Kubernetes, EKS | 1 → 3 | 1 → 2 | 0 → 1 | 2 → 4 |
| AWS (RDS, S3, IAM) | 2 → 3 | 1 → 2 | 0 → 1 | 3 → 4 |
| Terraform | 1 → 2 | 0 → 1 | 0 → 0 | 3 → 4 |
| CI/CD, DevSecOps | 2 → 3 | 2 → 2 | 2 → 2 | 3 → 4 |
| Observabilité | 2 → 3 | 1 → 2 | 0 → 1 | 2 → 3 |
| Tests de charge | 1 → 2 | 1 → 2 | 0 → 1 | 2 → 3 |

Lecture : Go et React sont couverts par au moins un niveau 3 dans l'équipe ; Kubernetes, EKS et Terraform ne le sont par personne tant que l'embauche plateforme n'est pas faite.

## 5. Écarts priorisés

| Rang | Écart | Qui | Impact sur le projet | Risque si non traité | Réponse |
|---|---|---|---|---|---|
| 1 | Kubernetes et EKS | Lead, confirmé back | Socle des 4 services (D02, D03) ; disponibilité et montée en charge à démontrer (C5) | Élevé : personne n'est autonome, dépendance à une embauche non faite, panne non diagnostiquée en démonstration | Formation F1 (§ 6) |
| 2 | React | Lead | Front du POC ; vocabulaire précis exigé à l'oral (C10) | Moyen : un référent React existe dans l'équipe | Formation F2 (§ 7) + binôme |
| 3 | Terraform | Lead, confirmé back | IaC, allumage et extinction maîtrisés du coût (D11) | Moyen : dérive de coût, environnement non reproductible | Formation courte après D11, puis embauche |
| 4 | RabbitMQ (outbox, idempotence) | Confirmé back | Contrôle de publication asynchrone (D01, D05) | Moyen : messages perdus ou traités deux fois | Revue de code et binôme avec le lead |
| 5 | OIDC, Keycloak | Toute l'équipe | Application qui porte des transactions financières (D06) | Élevé sur la sécurité, faible sur la charge | Atelier interne d'une demi-journée, doc « À savoir défendre » de D06 |
| 6 | Observabilité | Lead, confirmé back | Composante exigée (D09) | Faible à moyen | Autoformation guidée après D09 |

## 6. Action de formation détaillée — F1 « Kubernetes et Amazon EKS pour développeurs »

Choisie en priorité : c'est l'écart le plus critique (rang 1), sans référent interne, et il conditionne le déploiement des quatre services.

| Rubrique | Contenu |
|---|---|
| Public | Lead developer et développeur confirmé back (2 personnes) ; le développeur front suit le module 1 seulement |
| Prérequis | Docker (construire et lancer une image), ligne de commande, notions HTTP ; compte AWS du Learner Lab actif |
| Durée | 3 jours (21 h), dont 60 % de pratique |
| Modalités | Présentiel ou classe virtuelle avec un organisme spécialisé, sur cluster de formation ; travaux pratiques sur un service Go de Collector.shop |
| Effectif | 2 à 3 stagiaires |

**Objectifs pédagogiques** — à l'issue, le stagiaire sait :

1. Expliquer le rôle du plan de contrôle, des nœuds, des pods, des Deployments et des Services.
2. Écrire et appliquer les manifestes d'un service (Deployment, Service, ConfigMap, Secret).
3. Configurer les sondes de vivacité et de disponibilité, les requêtes et limites de ressources.
4. Mettre en place une mise à l'échelle automatique (HPA sur le CPU, KEDA sur la longueur d'une file).
5. Diagnostiquer un pod en échec (`kubectl describe`, `logs`, événements).
6. Situer EKS : node group géré, IAM, réseau VPC, exposition par équilibreur de charge.

**Programme séquencé**

| Séquence | Durée | Contenu | Travaux pratiques |
|---|---|---|---|
| J1 matin | 3,5 h | Architecture Kubernetes, objets de base, `kubectl` | Déployer un service Go sur un cluster local (kind) |
| J1 après-midi | 3,5 h | Configuration et secrets, sondes, ressources | Rendre le service tolérant au redémarrage |
| J2 matin | 3,5 h | Réseau : Services, Ingress, DNS interne | Faire communiquer deux services en HTTP interne |
| J2 après-midi | 3,5 h | Mise à l'échelle : HPA, KEDA | Déclencher une montée en charge sur une file RabbitMQ |
| J3 matin | 3,5 h | Amazon EKS : node group géré, IAM, équilibreur | Déployer le même service sur EKS dans le Learner Lab, puis l'éteindre |
| J3 après-midi | 3,5 h | Diagnostic et bonnes pratiques de sécurité (comptes de service, images non root) | Atelier panne : retrouver la cause de trois pannes injectées |

**Évaluation et livrables**

| Moment | Forme | Critère de réussite |
|---|---|---|
| Fin de formation | Atelier panne noté | 3 pannes sur 3 diagnostiquées en moins de 45 min |
| Sprint 1 (J4) | Livrable réel : manifestes du walking skeleton relus en revue de code | Déployé, sondes et ressources renseignées |
| Sprint 2 (J5) | Démonstration interne : perte d'une instance et montée en charge | Service toujours disponible, réplicas ajoutés automatiquement |
| J6 | Auto-évaluation et réévaluation de la matrice | Niveau 3 atteint pour le lead |

**Coût estimé** : environ 1 800 € HT par stagiaire en interentreprises, soit 3 600 € pour deux, plus le temps de travail (6 jours-personne). La consigne n'impose pas de contrainte budgétaire à l'équipe ; ce coût est éligible au plan de développement des compétences de l'entreprise.

**Calendrier relatif aux sprints** : avant J4 (pendant J3, avec la configuration AWS), pour que le sprint 1 serve de mise en pratique immédiate.

## 7. Action esquissée — F2 « React pour développeurs Angular et Vue »

| Rubrique | Contenu |
|---|---|
| Public | Lead developer |
| Objectif | Coder le front du POC seul et défendre à l'oral : rendu, état et props, hooks et dépendances, effets, composants contrôlés, contexte |
| Durée | 3 à 5 jours (estimation D04), réduite par la pratique de l'API de composition de Vue |
| Modalités | Autoformation sur la documentation officielle de React, puis binôme avec le développeur front confirmé (référent, niveau 4) |
| Évaluation | Premier écran du POC relu par le référent ; questions orales sur les hooks et les effets avant J7 |
| Calendrier | Avant le premier écran front, au plus tard au sprint 2 (J5) |

## 8. Dispositifs continus

| Dispositif | Fonctionnement | Écart traité |
|---|---|---|
| Revue de code croisée | Chaque merge est relu par une autre personne ; le relecteur explique un point appris | Tous, en particulier RabbitMQ et React |
| Binôme (pair programming) | Une session par semaine sur la zone d'apprentissage d'un des deux | React (lead + front), Kubernetes (lead + back) |
| Documentation « À savoir défendre » | Chaque décision comporte une section de questions et réponses précises | Vocabulaire technique, transmission aux embauches |
| Veille | 30 min par semaine, une note partagée sur Go, React, Kubernetes ou la sécurité | Maintien du niveau |
| Intégration d'une embauche | Parcours : lecture des décisions D01 à D14, puis F1 si besoin, puis binôme | Recrutement plateforme |

## 9. Suivi

La matrice du § 4 est réévaluée en J6 à partir de faits observables (livrables produits, revues, démonstrations). Les écarts encore ouverts alimentent le plan de la V1.
