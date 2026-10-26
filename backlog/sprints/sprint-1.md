# Sprint 1 — Walking skeleton WS-2 sur AWS

Processus et cérémonies : [developpement.md](../../docs/processus/developpement.md). Clôture : [PILOTAGE.md](../../docs/PILOTAGE.md) § 3.4. Backlog : [BACKLOG.md](../BACKLOG.md).

| Champ | Valeur |
|---|---|
| Jalon | J4 |
| Dates | 2026-11-03 → 2026-11-09 (proposées, à confirmer par Martin) |
| Statut | Planifié |

## Objectif

À la fin du sprint, un vendeur connecté soumet un article sur `https://collector.qvl-project.com`, le Contrôle de publication applique la règle de complétude par le canal, et l'article publié apparaît au catalogue public ; l'environnement est allumé et éteint depuis zéro, livré par une CI à portes bloquantes, sans secret ni exposition superflue (C2, C4, C5, C8, avec C3 et C7).

**Conditions d'entrée** ([PILOTAGE.md](../../docs/PILOTAGE.md) § 3.4, [D14](../../docs/decisions/D14-experimentations.md)) :

- verdicts de XP01, XP02 et XP03 rendus au 2026-11-02 ; D03, D05, D06 et D08 passées de `Candidate` à `Décidée` ;
- XP02 concluant sur l'environnement managé, sinon variante WS-1 (US-011 à US-013 au sprint 2) ;
- adresse e-mail publique des commits choisie par Martin (US-001).

## US engagées

Ordre d'engagement = rang du backlog.

| US | Titre | Fonction(s) | Statut | Dépend de |
|---|---|---|---|---|
| [US-001](../US-001-publication-depot-github.md) | Publier le dépôt sur GitHub sans fuite | DevOps | À faire | — |
| [US-002](../US-002-pipeline-pr-ci-ok.md) | Pipeline de PR et vérification requise `ci-ok` | DevOps | En cours | US-001 |
| [US-004](../US-004-environnement-aws-https.md) | Allumer, exposer en HTTPS et éteindre l'environnement AWS (X17, X20 bloquants) | DevOps | À faire | US-001, XP01, XP03 |
| [US-003](../US-003-portes-securite-ci.md) | Portes de sécurité bloquantes dans la CI | DevOps | À faire | US-002 |
| [US-005](../US-005-garde-fous-cout.md) | Garde-fous de coût | DevOps | À faire | US-004 |
| [US-008](../US-008-connexion-keycloak.md) | Se connecter par le serveur d'autorisation | DevOps / Front | À faire | US-002, US-004 |
| [US-006](../US-006-cloisonnement-cluster.md) | Cloisonnement du cluster | DevOps | À faire | US-004 |
| [US-011](../US-011-canal-rabbitmq-identites.md) | Canal asynchrone avec une identité par service | DevOps | À faire | US-004, US-006, XP02 |
| [US-012](../US-012-contrat-controle-v1.md) | Contrat v1 du contrôle de publication | Back | À faire | US-002 |
| [US-009](../US-009-brouillon-article-vendeur.md) | Créer et modifier un brouillon d'article | Back / Front / DevOps | À faire | US-008 |
| [US-010](../US-010-catalogue-public.md) | Consulter le catalogue public sans compte | Back / Front / DevOps | À faire | US-009 |
| [US-013](../US-013-soumission-controle-completude.md) | Soumettre un article au contrôle de complétude | Back / Front | À faire | US-009 à US-012 |
| [US-007](../US-007-livraison-images-signees-flux.md) | Livraison continue : images signées et Flux | DevOps | À faire | US-002, US-004 |
| [US-014](../US-014-journaux-centralises.md) | Journaux centralisés et corrélés | DevOps / Back | À faire | US-004, US-013 |

Parallélisation possible : US-001 → US-002 → US-003 côté chaîne de livraison, pendant US-004 côté AWS ; US-012 dès US-002 ; développement local sur docker compose de US-009, US-010 et US-013 pendant que US-006 et US-011 se terminent sur AWS.

## Capacité

| Élément | Valeur |
|---|---|
| Sessions de travail disponibles | 7 jours calendaires ; hypothèse de 10 points par jour, soit 70 points |
| Engagé | 54 points (14 US fermes) |
| Part réservée à la dette | 20 % (14 points) ; aucune dette héritée : la réserve absorbe la dette née dans le sprint |
| Session AWS prévue (allumage / extinction) | 4 sessions de 2 h 30 à 4 h : (1) deux allumages et extinctions chronométrés (X4, X5) et X15 ; (2) X17, X20, Keycloak, cloisonnement ; (3) canal et parcours WS-2 ; (4) démonstration de fin de sprint, ZAP sur l'URL publique. Coût estimé ≈ 5,4 $ |

## Risques

| Risque | Probabilité | Impact | Parade |
|---|---|---|---|
| Bascule d'un verdict XP (EKS vers ECS Fargate, RabbitMQ vers NATS, Keycloak vers Cognito) | Moyenne | Fort : US-004, US-006, US-008, US-011 à réécrire | Replis nommés dans [architecture-technique.md](../../docs/architecture/architecture-technique.md) § 9.1 ; si la bascule déborde sur 2026-11-03, démarrage en WS-1 |
| Allumage au-delà de 45 min ou première installation en échec (ordre Secrets, amorçage, crochet de migration) | Moyenne | Fort : sessions perdues, budget | Ordre fixé dans [D10](../../docs/decisions/D10-forge-ci-cd.md) ; filet `up.sh --sans-flux` ; journal horodaté |
| X17 ou X20 rouge | Moyenne | Fort : ALB non ouvert à Internet | ALB restreint à l'adresse de Martin en attendant ; correction de l'IaC avant la session suivante |
| `ci-ok` bloqué par un job ignoré ; base Trivy ou GHCR limitée en débit | Moyenne | Moyen : PR impossibles à fusionner | Agrégateur en `if: always()` ; cache quotidien de la base Trivy, miroir `public.ecr.aws` |
| Paquet GHCR créé privé | Moyenne | Faible : `ImagePullBackOff` | Passage public à la première publication (AC7 d'US-007) |
| Densité sur deux t3.medium (Keycloak, RabbitMQ, Flux, Fluent Bit) | Moyenne | Moyen | Demandes de ressources relevées en XP01 ; quatre nœuds au plus |
| Dépassement de charge du sprint (54 points sur une hypothèse non mesurée) | Forte | Moyen | Ordre de coupe ci-dessous ; recalibrage de la capacité à la clôture |
| Dérive du budget AWS | Faible | Fort | US-005 tôt dans le sprint ; allumage à la demande ; seuil 70 % |

Ordre de coupe si le sprint déborde : (1) US-007 réduite au déploiement depuis le poste, signature conservée ; (2) US-014 en tête du sprint 2 ; le parcours WS-2 n'est jamais coupé. Référence : [D02](../../docs/decisions/D02-decoupage-applicatif.md) et [BACKLOG.md](../BACKLOG.md) § 5.

## Dette technique

| ID | Constat | Source (revue, audit, indicateur) | Impact | Remboursée (date, US) |
|---|---|---|---|---|
| — | Aucune dette à l'ouverture du sprint | — | — | — |

## Revue (démo)

- Parcours démontré :
- Écarts constatés par rapport à l'objectif :
- US non livrées et motif :

## Rétrospective

| À garder | À améliorer | Action pour le sprint suivant |
|---|---|---|
| | | |

## Indicateurs relevés

Relevé détaillé : [indicateurs-sprint-1.md](../../docs/mesures/indicateurs-sprint-1.md). Première mesure : les seuils de départ de [D12](../../docs/decisions/D12-indicateurs-qualite.md) sont confirmés sur ces valeurs.

| Indicateur | Valeur | Tendance | Axe d'amélioration |
|---|---|---|---|
| I1 — AC couverts par un test vert | | Référence | |
| I4 — Couverture par service | | Référence | |
| I5 — Complexité cognitive et duplication | | Référence | |
| I6 — Vulnérabilités et exceptions | | Référence | |
| I7 — Livraison (DORA réduit) | | Référence | |
| Vélocité (points livrés) | | Référence | Recalibrage de la capacité des sprints 2 et 3 |

I2 et I3 sont mesurés à partir du sprint 2 (premier tir de charge).

## Clôture

- [ ] Audits : [architecture](../../docs/audits/sprint-1-architecture.md), [sécurité](../../docs/audits/sprint-1-securite.md)
- [ ] Documentation touchée à jour (README « Démarrer / arrêter », `docs/exploitation.md`)
- [ ] US livrées mergées sur `main`, branches mergées supprimées
- [ ] Infrastructure éteinte, aucune ressource restante (`tools/status.sh --strict`)
- [ ] Capacité des sprints 2 et 3 recalibrée sur la vélocité mesurée
