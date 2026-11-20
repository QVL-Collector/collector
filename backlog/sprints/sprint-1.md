# Sprint 1 — Walking skeleton WS-2 sur AWS

Processus et cérémonies : [developpement.md](../../docs/processus/developpement.md). Clôture : [PILOTAGE.md](../../docs/PILOTAGE.md) § 3.4. Backlog : [BACKLOG.md](../BACKLOG.md).

| Champ | Valeur |
|---|---|
| Jalon | J4 |
| Dates | Prévues : 2026-11-03 → 2026-11-09. Observées (premier et dernier merge du sprint) : 2026-10-26 → 2026-10-28 |
| Statut | Clos sur le périmètre local ; US AWS reportées au sprint 3 (bilan du 2026-11-20) |

## Objectif

À la fin du sprint, un vendeur connecté soumet un article sur `https://collector.qvl-project.com`, le Contrôle de publication applique la règle de complétude par le canal, et l'article publié apparaît au catalogue public ; l'environnement est allumé et éteint depuis zéro, livré par une CI à portes bloquantes, sans secret ni exposition superflue (C2, C4, C5, C8, avec C3 et C7).

**Conditions d'entrée** ([PILOTAGE.md](../../docs/PILOTAGE.md) § 3.4, [D14](../../docs/decisions/D14-experimentations.md)) :

- verdicts de XP01, XP02 et XP03 rendus au 2026-11-02 ; D03, D05, D06 et D08 passées de `Candidate` à `Décidée` ;
- XP02 concluant sur l'environnement managé, sinon variante WS-1 (US-011 à US-013 au sprint 2) ;
- adresse e-mail publique des commits choisie par Martin (US-001).

État à l'ouverture : seule la troisième condition était remplie (D10, complément du 2026-10-26). Les verdicts XP ont été rendus le 2026-11-20 (voir « Revue »).

## US engagées

Ordre d'engagement = rang du backlog. Statuts au 2026-11-20.

| US | Titre | Fonction(s) | Statut | Dépend de | PR |
|---|---|---|---|---|---|
| [US-001](../US-001-publication-depot-github.md) | Publier le dépôt sur GitHub sans fuite | DevOps | Livrée | — | [#9](https://github.com/QVL-Collector/collector/pull/9) |
| [US-002](../US-002-pipeline-pr-ci-ok.md) | Pipeline de PR et vérification requise `ci-ok` | DevOps | Livrée | US-001 | [#1](https://github.com/QVL-Collector/collector/pull/1) |
| [US-004](../US-004-environnement-aws-https.md) | Allumer, exposer en HTTPS et éteindre l'environnement AWS (X17, X20 bloquants) | DevOps | En cours, reportée au sprint 3 | US-001, XP01, XP03 | [#50](https://github.com/QVL-Collector/collector/pull/50) (partiel, sprint 3) |
| [US-003](../US-003-portes-securite-ci.md) | Portes de sécurité bloquantes dans la CI | DevOps | Livrée | US-002 | [#2](https://github.com/QVL-Collector/collector/pull/2) |
| [US-005](../US-005-garde-fous-cout.md) | Garde-fous de coût | DevOps | À faire, reportée au sprint 3 | US-004 | — |
| [US-008](../US-008-connexion-keycloak.md) | Se connecter par le serveur d'autorisation | DevOps / Front | Livrée côté front ; AC3, AC6 à AC9 à prouver | US-002, US-004 | [#15](https://github.com/QVL-Collector/collector/pull/15) |
| [US-006](../US-006-cloisonnement-cluster.md) | Cloisonnement du cluster | DevOps | À faire, reportée au sprint 3 | US-004 | — |
| [US-011](../US-011-canal-rabbitmq-identites.md) | Canal asynchrone avec une identité par service | DevOps | Livrée (en local) | US-004, US-006, XP02 | [#19](https://github.com/QVL-Collector/collector/pull/19) |
| [US-012](../US-012-contrat-controle-v1.md) | Contrat v1 du contrôle de publication | Back | Livrée | US-002 | [#4](https://github.com/QVL-Collector/collector/pull/4) |
| [US-009](../US-009-brouillon-article-vendeur.md) | Créer et modifier un brouillon d'article | Back / Front / DevOps | Livrée côté back ; AC5, AC10, AC11 à faire | US-008 | [#14](https://github.com/QVL-Collector/collector/pull/14) |
| [US-010](../US-010-catalogue-public.md) | Consulter le catalogue public sans compte | Back / Front / DevOps | Livrée côté front ; AC3, AC5 à prouver | US-009 | [#6](https://github.com/QVL-Collector/collector/pull/6) |
| [US-013](../US-013-soumission-controle-completude.md) | Soumettre un article au contrôle de complétude | Back / Front | Livrée (en local) | US-009 à US-012 | [#21](https://github.com/QVL-Collector/collector/pull/21) |
| [US-007](../US-007-livraison-images-signees-flux.md) | Livraison continue : images signées et Flux | DevOps | À faire, reportée au sprint 3 | US-002, US-004 | — |
| [US-014](../US-014-journaux-centralises.md) | Journaux centralisés et corrélés | DevOps / Back | À faire, reportée au sprint 3 | US-004, US-013 | — |

Parallélisation possible : US-001 → US-002 → US-003 côté chaîne de livraison, pendant US-004 côté AWS ; US-012 dès US-002 ; développement local sur docker compose de US-009, US-010 et US-013 pendant que US-006 et US-011 se terminent sur AWS.

## Capacité

| Élément | Valeur |
|---|---|
| Sessions de travail disponibles | 7 jours calendaires ; hypothèse de 10 points par jour, soit 70 points |
| Engagé | 54 points (14 US fermes) |
| Part réservée à la dette | 20 % (14 points) ; aucune dette héritée : la réserve absorbe la dette née dans le sprint |
| Session AWS prévue (allumage / extinction) | 4 sessions de 2 h 30 à 4 h : (1) deux allumages et extinctions chronométrés (X4, X5) et X15 ; (2) X17, X20, Keycloak, cloisonnement ; (3) canal et parcours WS-2 ; (4) démonstration de fin de sprint, ZAP sur l'URL publique. Coût estimé ≈ 5,4 $ |
| **Réalisé** | 31 points mergés sur 54 : 18 points entièrement livrés (US-001, US-002, US-003, US-011, US-012, US-013), 13 points livrés hors AC d'environnement (US-008, US-009, US-010) ; 23 points reportés (US-004 à US-007, US-014). Aucune session AWS du POC pendant le sprint |

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

Risque réalisé, non listé : **verdicts XP en retard** (prévus le 2026-11-02, rendus le 2026-11-20). Il a bloqué toute la branche AWS du chemin critique (US-004 et ses dépendantes).

Ordre de coupe si le sprint déborde : (1) US-007 réduite au déploiement depuis le poste, signature conservée ; (2) US-014 en tête du sprint 2 ; le parcours WS-2 n'est jamais coupé. Référence : [D02](../../docs/decisions/D02-decoupage-applicatif.md) et [BACKLOG.md](../BACKLOG.md) § 5.

## Dette technique

| ID | Constat | Source (revue, audit, indicateur) | Impact | Remboursée (date, US) |
|---|---|---|---|---|
| DT-01 | Programme de relevé des indicateurs (`tools/indicateurs`, D12) absent : relevé non généré, tendance non calculable | Indicateurs (relevé du sprint 1) | C6 : relevés incomplets, sans série JSON | Non |

## Revue (démo)

Aucune revue de sprint consignée à la date de clôture ; bilan établi le 2026-11-20 à partir des US, des rapports et de l'historique de `main`.

- **Parcours démontré** : walking skeleton WS-2 complet **en local** (docker compose et k3d) : connexion par Keycloak (US-008), brouillon (US-009), soumission, contrôle de complétude par le canal RabbitMQ avec une identité par service (US-011, US-012, US-013), publication au catalogue public (US-010). Chaîne de livraison en place sur GitHub : `ci-ok` requis, portes de sécurité bloquantes (US-001 à US-003).
- **Écarts constatés par rapport à l'objectif** :
  - rien n'a tourné sur AWS : pas d'URL publique, pas d'allumage ni d'extinction depuis zéro, pas de Flux ni d'images signées, pas de journaux centralisés ;
  - WS-2 a été retenu sans le verdict de XP02 sur l'environnement managé, condition de [PILOTAGE.md](../../docs/PILOTAGE.md) § 3.4 : le canal a été validé en local (XP02 en compose et k3d, [PR #23](https://github.com/QVL-Collector/collector/pull/23)), le verdict sur EKS n'est venu que le 2026-11-20. Le repli WS-1 n'a pas été appliqué ; le choix n'est pas consigné comme décision ;
  - le travail a commencé avant les dates prévues (2026-10-26 au lieu du 2026-11-03) et a enchaîné sur le sprint 2 sans clôture formelle ; US-025 lot 1 (sprint 2) a été mergée pendant le sprint 1 ([PR #20](https://github.com/QVL-Collector/collector/pull/20)).
- **US non livrées et motif** : US-004, US-005, US-006, US-007 et US-014 (23 points). **AWS bloqué jusqu'aux verdicts XP** : US-004 dépend des verdicts de XP01 et XP03, prévus le 2026-11-02 ; la session EKS 1 de XP01 ([PR #34](https://github.com/QVL-Collector/collector/pull/34), 2026-11-06) n'a pas suffi à conclure ; la session EKS 2 a permis la **décision EKS du 2026-11-20** ([PR #48](https://github.com/QVL-Collector/collector/pull/48), D03) et les verdicts de XP02 et XP03 ([PR #49](https://github.com/QVL-Collector/collector/pull/49)). Les autres US en dépendent (US-004 → US-005, US-006, US-007, US-014). Les AC d'environnement d'US-008, US-009 et US-010 (realm importé à chaque `up`, Job d'amorçage, jeu de démonstration) attendent la même session.
- **Ordre de coupe appliqué** : coupe n° 2 (US-014 hors du sprint 1), au-delà de ce qui était prévu : US-014 n'a pas été reprise en tête du sprint 2 mais reportée au sprint 3, avec toutes les US AWS. Coupe n° 1 (US-007 réduite) sans objet tant qu'US-004 n'est pas livrée. Le parcours WS-2 n'a pas été coupé.

### Décisions prises pendant le sprint

| Date | Décision | Où elle est consignée |
|---|---|---|
| 2026-10-26 | Adresse des commits : `noreply` GitHub, historique réécrit avant la première poussée (US-001) | [D10](../../docs/decisions/D10-forge-ci-cd.md), complément |
| 2026-10-26 | Texte saisi (titre, description bornés) dans `controle.demande` v1, écart assumé à l'avis sécurité, compensé par trois règles (validation d'US-012) | [D05](../../docs/decisions/D05-communication-interservice.md), complément |
| 2026-10-26 | Accès des pods aux identifiants du nœud : « 2 sauts + filtre réseau » | [D03](../../docs/decisions/D03-hebergement-orchestration.md), complément |
| 2026-10-26 | Repli P1 si le sprint 3 ne tient pas, décidé à la revue du sprint 2 | [D02](../../docs/decisions/D02-decoupage-applicatif.md), complément |

## Rétrospective

Aucune rétrospective consignée. Constats factuels proposés pour la rétrospective, à valider par Martin :

| À garder | À améliorer | Action pour le sprint suivant |
|---|---|---|
| Développement local sur docker compose et k3d : le parcours WS-2 a avancé malgré AWS bloqué | Bornes et clôture du sprint non consignées ; statuts du fichier de sprint non tenus | À trancher par Martin |
| Circuit test puis revue sur chaque US, rapports versionnés | Statuts partiels (« livrée côté front ») hors du vocabulaire de [PILOTAGE.md](../../docs/PILOTAGE.md) § 4 | À trancher par Martin |

## Indicateurs relevés

Relevé détaillé : [indicateurs-sprint-1.md](../../docs/mesures/indicateurs-sprint-1.md). Première mesure : les seuils de départ de [D12](../../docs/decisions/D12-indicateurs-qualite.md) devaient être confirmés sur ces valeurs ; faute de relevé généré, ils ne le sont pas.

| Indicateur | Valeur | Tendance | Axe d'amélioration |
|---|---|---|---|
| I1 — AC couverts par un test vert | Non mesuré (relevé non généré) ; AC sans preuve nommés dans le relevé | Référence | Prouver les AC d'environnement d'US-008, US-009, US-010 |
| I4 — Couverture par service | Valeurs ponctuelles des rapports de test (ex. Place de marché 77,3 %, web 86,9 % en lignes) | Référence | `libs/platform/broker` (63,0 %, sans tests propres) |
| I5 — Complexité cognitive et duplication | Non mesuré (porte `gocognit` active ; comptage et duplication non outillés) | Référence | Outiller jscpd et `sonarjs` |
| I6 — Vulnérabilités et exceptions | 0 exception dans `.trivyignore` ; vulnérabilités ouvertes non relevées | Référence | Lire l'artefact `i6` de `main` |
| I7 — Livraison (DORA réduit) | Non mesuré (API GitHub non relevée) ; 9 PR d'US mergées | Référence | Écrire `tools/indicateurs` (DT-01) |
| Vélocité (points livrés) | 31 points mergés (18 complets, 13 hors AC d'environnement) | Référence | Recalibrage non fait : voir « Clôture » |

I2 et I3 sont mesurés à partir du sprint 2 (premier tir de charge).

## Clôture

- [ ] Audits : [architecture](../../docs/audits/sprint-1-architecture.md), [sécurité](../../docs/audits/sprint-1-securite.md) — non faits ; regroupés en J6 (décision de pilotage)
- [ ] Documentation touchée à jour (README « Démarrer / arrêter », `docs/exploitation.md`) — README mis à jour avec US-004 ; `docs/exploitation.md` inexistant
- [ ] US livrées mergées sur `main`, branches mergées supprimées — US livrées mergées ; branches locales `feat/US-013-soumission-controle-completude` et `docs/statuts-us-livrees` encore présentes
- [x] Infrastructure éteinte, aucune ressource restante — aucune ressource du POC allumée pendant le sprint
- [ ] Capacité des sprints 2 et 3 recalibrée sur la vélocité mesurée — non fait : la vélocité est faussée par le blocage AWS (31 points en 3 jours observés, hors US AWS)
