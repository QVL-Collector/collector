# Sprint 3 — Diffusion vers Interactions, réservation par Transactions, rattrapage AWS

Processus et cérémonies : [developpement.md](../../docs/processus/developpement.md). Clôture : [PILOTAGE.md](../../docs/PILOTAGE.md) § 3.4. Backlog : [BACKLOG.md](../BACKLOG.md).

| Champ | Valeur |
|---|---|
| Jalon | J5 bis |
| Dates | Prévues : 2026-11-15 → 2026-11-17. Observée : premier merge le 2026-11-13 ; date de fin à fixer par Martin (avant J6) |
| Statut | En cours |

## Objectif

À la fin du sprint, les quatre services de l'option P2 tournent sur EKS : la décision de publication est diffusée à Interactions et notifiée au vendeur sans perte, un acheteur réserve un article publié par un appel synchrone authentifié, et la charge, la perte d'instance et le troisième profil de mise à l'échelle sont démontrés sur l'environnement managé (C2, C3, C5, C8, avec C6 et C7).

Contenu de [D02](../../docs/decisions/D02-decoupage-applicatif.md), « Découpage indicatif en sprints », ligne S3 : diffusion de « décision de publication » vers Interactions minimal et notification au vendeur ; Transactions réduit à la réservation, appel synchrone authentifié, espace acheteur ; tests de contrat, de concurrence et négatifs de service à service ; Interactions arrêté puis relancé sans perte (ENF-12) ; troisième profil de mise à l'échelle ; tir sur la consultation rejoué en non-régression avec les quatre services.

S'y ajoutent les US des sprints 1 et 2 reportées faute d'AWS ([sprint-1.md](sprint-1.md), [sprint-2.md](sprint-2.md)).

## US engagées

Statuts au 2026-11-20.

### Contenu propre au sprint 3

| US | Titre | Fonction(s) | Engagement | Statut | Dépend de | PR |
|---|---|---|---|---|---|---|
| [US-030](../US-030-reserver-article.md) | Réserver un article publié | Back, Front, DevOps | Ferme | Livrée (AC7 à prouver sur AWS) | US-006, US-008, US-010 | [#45](https://github.com/QVL-Collector/collector/pull/45) |
| [US-031](../US-031-garanties-reservation.md) | Garantir qu'un objet unique n'est réservé qu'une fois | Back | Ferme | Livrée | US-030 | [#46](https://github.com/QVL-Collector/collector/pull/46) |
| [US-028](../US-028-notification-vendeur.md) | Être notifié de la décision de publication | Back, Front, DevOps | Ferme | Livrée | US-011, US-013, US-025 | [#37](https://github.com/QVL-Collector/collector/pull/37) |
| [US-029](../US-029-reprise-interactions.md) | Aucune notification perdue quand Interactions est arrêté | Back, DevOps | Ferme | Livrée | US-028 | [#38](https://github.com/QVL-Collector/collector/pull/38) |
| [US-032](../US-032-espace-acheteur.md) | Suivre ses réservations dans l'espace acheteur | Back, Front | Coupable | Livrée | US-031 | [#47](https://github.com/QVL-Collector/collector/pull/47) |
| [US-033](../US-033-non-regression-quatre-services.md) | Rejouer la charge avec les quatre services | DevOps, Test | Coupable | À faire (session AWS 3) | US-016, US-028, US-030 | — |

### Reportées des sprints 1 et 2

| US | Titre | Sprint d'origine | Statut | Session AWS | PR |
|---|---|---|---|---|---|
| [US-004](../US-004-environnement-aws-https.md) | Allumer, exposer en HTTPS et éteindre l'environnement AWS | 1 | En cours : AC1, AC3, AC5 à AC8, AC10, AC11 verts le 2026-11-20 ; restent AC2 et AC4 (seconde mesure), AC9 (image de la CI) | 1 (faite), 2 | [#50](https://github.com/QVL-Collector/collector/pull/50) (partiel) |
| [US-005](../US-005-garde-fous-cout.md) | Garde-fous de coût | 1 | À faire | 2 | — |
| [US-006](../US-006-cloisonnement-cluster.md) | Cloisonnement du cluster | 1 | À faire | 2 | — |
| [US-007](../US-007-livraison-images-signees-flux.md) | Livraison continue : images signées et Flux | 1 | À faire | 2 | — |
| [US-008](../US-008-connexion-keycloak.md) | Se connecter par le serveur d'autorisation : AC d'environnement | 1 | Livrée côté front ; AC3, AC6 à AC9 à prouver | 2 | [#15](https://github.com/QVL-Collector/collector/pull/15) |
| [US-009](../US-009-brouillon-article-vendeur.md) | Créer et modifier un brouillon d'article : AC d'environnement | 1 | Livrée côté back ; AC5, AC10, AC11 à faire | 2 (AC10) | [#14](https://github.com/QVL-Collector/collector/pull/14) |
| [US-010](../US-010-catalogue-public.md) | Consulter le catalogue public : AC d'environnement | 1 | Livrée côté front ; AC3, AC5 à prouver | 2 | [#6](https://github.com/QVL-Collector/collector/pull/6) |
| [US-014](../US-014-journaux-centralises.md) | Journaux centralisés et corrélés (partie DevOps) | 1 | À faire | 3 | — |
| [US-015](../US-015-metriques-tableaux-de-bord.md) | Métriques et tableaux de bord | 2 | À faire | 3 | — |
| [US-016](../US-016-montee-en-charge-catalogue.md) | Absorber un pic de visiteurs sur le catalogue | 2 | À faire | 3 | — |
| [US-017](../US-017-perte-instance-sous-charge.md) | Rester disponible quand une instance tombe sous charge | 2 | À faire | 3 | — |
| [US-020](../US-020-photo-securisee.md) | Ajouter une photo à un article, en toute sécurité | 2 | À faire | 3 | — |
| [US-026](../US-026-controle-echelle-file.md) | Absorber une vague de soumissions (KEDA sur le Contrôle) | 2 | À faire | 3 | — |

## Plan des sessions AWS

Session de 4 h au plus (Learner Lab), `down.sh` terminé avant « End Lab ». Journal : [sessions.csv](../../docs/mesures/sessions.csv), coûts : [couts.csv](../../docs/mesures/couts.csv).

| Session | Date | Contenu | Résultat |
|---|---|---|---|
| 1 | 2026-11-20 | US-004 : `bootstrap.sh`, premier `up` (HTTPS 200 en 16 min), contrôle après déploiement, `down` (9 min, inventaire vide) | Faite : 58 min, ≈ 0,33 $ ; AC9 rouge (image nginx provisoire) |
| 2 | À fixer | **US-004 fin** (seconde mesure d'AC2 et AC4, AC9 avec l'image de la CI) ; **US-005** ; **US-006** ; **US-007** ; **AC AWS d'US-008, US-009 et US-010** (realm importé à chaque `up`, X17 étendu à Keycloak, en-têtes servis par nginx, Job d'amorçage et rôles de la base, jeu de démonstration et temps de réponse à vide) | — |
| 3 | À fixer | **US-014** (partie DevOps : Fluent Bit, groupes de journaux) ; **US-015** ; **US-016** ; **US-017** ; **US-020** ; **US-026** ; **US-033** | — |

Ordre dans la session 2 : US-007 avant la clôture d'US-004, puisque AC9 d'US-004 se rejoue avec l'image publiée par la CI (décision du 2026-11-20).

AC après déploiement d'US déjà livrées, absents du plan ci-dessus (à placer par Martin) : US-030 AC7 (route « réserver » absente de l'ALB, politique réseau), US-027 AC4 et AC5 (limitation de débit derrière l'ALB, après chaque tir), US-022 AC6 et parcours Playwright sur environnement déployé.

## Capacité

| Élément | Valeur |
|---|---|
| Sessions de travail disponibles | Plan : 3 jours calendaires, 30 points ([BACKLOG.md](../BACKLOG.md) § 2), dépassé |
| Engagé | Contenu propre : 15 points fermes, 4 coupables. Reports : 41 points (23 du sprint 1, 18 du sprint 2) |
| Part réservée à la dette | 20 % (cible) |
| Session AWS prévue (allumage / extinction) | 3 sessions (1 faite) |
| Réalisé au 2026-11-20 | 17 points livrés (US-028 à US-032) ; restent US-033 (2 points) et les 41 points reportés, dont US-004 en partie |

## Risques

| Risque | Probabilité | Impact | Parade |
|---|---|---|---|
| Contenu de la session 2 ou 3 trop large pour 4 h de lab | Forte | Fort : US reportées encore, budget | Ordre de coupe ci-dessous ; allumage chronométré (16 min mesurées) ; découpage en sessions supplémentaires |
| AC9 d'US-004 bloqué tant qu'US-007 ne publie pas l'image du front | Moyenne | Fort : ALB non ouvrable à Internet (AC10) | US-007 en tête de la session 2 |
| Clé KMS des Secrets EKS refusée par le lab | Faible (créée le 2026-11-20) | Moyen | Repli en exception documentée dans `.trivyignore` (US-004) |
| Densité : quatre services, Keycloak, RabbitMQ, Prometheus, Grafana, Fluent Bit, Flux sur les nœuds | Moyenne | Moyen | Quatre nœuds au plus (G4) ; demande CPU du broker à 500m (XP02) |
| Tests Vitest instables (US-008 AC10, US-025 AC5, US-008 AC2) | Moyenne | Moyen : `ci-ok` rouge sans cause | DT-04, branche `test/tests-instables` |
| Exceptions Trivy AWS-0053 et AWS-0040 échues le 2026-12-20 | Certaine si non renouvelées | Moyen : porte I6 rouge | Renouvellement ou solde avant l'échéance |

Ordre de coupe si le sprint déborde ([D02](../../docs/decisions/D02-decoupage-applicatif.md), [BACKLOG.md](../BACKLOG.md) § 5) : US-032 (déjà livrée), puis US-033 ; en dernier recours, repli P1 (non nécessaire : Interactions et Transactions sont livrés en local).

## Dette technique

| ID | Constat | Source (revue, audit, indicateur) | Impact | Remboursée (date, US) |
|---|---|---|---|---|
| DT-04 | Tests Vitest instables dans la suite complète : US-008 AC10, US-025 AC5, US-008 AC2 | Rapports de test d'US-030 et d'US-031 | Fiabilité de `ci-ok` | En cours (branche `test/tests-instables`) |
| DT-05 | Stockage, configuration, migrations et binaire de Transactions sans tests propres : prouvés par le banc de la Place de marché, invisibles pour I4 | Revue d'US-030, rapport de test d'US-032 | I4 sous-estimé pour Transactions | Non |

Dettes ouvertes héritées : DT-01 ([sprint-1.md](sprint-1.md)), DT-02 et DT-03 ([sprint-2.md](sprint-2.md)).

## Décisions prises pendant le sprint

| Date | Décision | Où elle est consignée |
|---|---|---|
| 2026-11-16 | Réservation par le vendeur de son propre article : 422, réponse identique à un article non réservable ; commission de 5 % du total, demi-centime arrondi vers le haut (US-030) | [Rapport de test](../../docs/rapports/tests/US-030-rapport-tests.md) et [revue](../../docs/rapports/revues/US-030-revue-code.md) d'US-030 seulement |
| 2026-11-16 | Pas de route « retirer » : le refus de modification d'un article réservé couvre le retrait ; statut `expiree` dérivé à la lecture avec l'horloge de Transactions, écart assumé avec « seule l'horloge de la base fait foi » de D07 (US-031) | [Revue d'US-031](../../docs/rapports/revues/US-031-revue-code.md) seulement |
| 2026-11-20 | EKS confirmé (XP01) | [D03](../../docs/decisions/D03-hebergement-orchestration.md), [PR #48](https://github.com/QVL-Collector/collector/pull/48) |
| 2026-11-20 | RabbitMQ confirmé avec réserves (XP02) ; Keycloak et liste blanche de l'ALB confirmés, un 400 de l'ALB accepté comme refus (XP03) | [XP02](../../docs/experimentations/XP02-canal-rabbitmq.md), [XP03](../../docs/experimentations/XP03-identite-exposition.md), [PR #49](https://github.com/QVL-Collector/collector/pull/49) ; D05, D06, D08 |
| 2026-11-20 | Exceptions Trivy AWS-0053 (ALB public) et AWS-0040 (API EKS publique) à échéance de 30 jours ; clé KMS persistante pour les Secrets EKS ; le front déployé sera l'image de la CI (US-007) | [D08](../../docs/decisions/D08-exposition-tls-domaine.md), [US-004](../US-004-environnement-aws-https.md) |
| 2026-11-20 | Audits d'architecture et de sécurité des sprints 1 à 3 regroupés en J6 (décision de pilotage) | Ce fichier, [sprint-1.md](sprint-1.md), [sprint-2.md](sprint-2.md) |

Autre fait du sprint : chaîne Go montée en 1.26.11 contre une vulnérabilité ([PR #41](https://github.com/QVL-Collector/collector/pull/41), 2026-11-13).

## Revue (démo)

- Parcours démontré :
- Écarts constatés par rapport à l'objectif :
- US non livrées et motif :

## Rétrospective

| À garder | À améliorer | Action pour le sprint suivant |
|---|---|---|
| | | |

## Indicateurs relevés

Relevé détaillé : [indicateurs-sprint-3.md](../../docs/mesures/indicateurs-sprint-3.md) (à produire à la clôture).

| Indicateur | Valeur | Tendance | Axe d'amélioration |
|---|---|---|---|
| | | | |

## Clôture

- [ ] Audits : regroupés en J6 (décision de pilotage) ; fichiers `docs/audits/sprint-3-architecture.md` et `sprint-3-securite.md` non produits
- [ ] Documentation touchée à jour
- [ ] US livrées mergées sur `main`, branches mergées supprimées
- [ ] Infrastructure éteinte, aucune ressource restante (`tools/status.sh --strict`)
