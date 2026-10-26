# Index des décisions

Statuts et circuit de décision : voir [PILOTAGE.md](../PILOTAGE.md). Gabarit : [_GABARIT.md](_GABARIT.md).
Le lien est renseigné dès que le fichier de la décision existe.

Sessions : S1 = D01 · S2 = D02 · S3 = D03 + D04 · S4 = D05 à D11 · S5 = D12 à D14.

| ID | Titre | Session | Statut | Critères grille | Dépend de | Fichier |
|---|---|---|---|---|---|---|
| D01 | Exigences reformulées et fonctionnalité métier du POC | S1 | Décidée | C3, C2, C7, C5, C8 | — | [D01-fonctionnalite-poc.md](D01-fonctionnalite-poc.md) |
| D02 | Architecture cible V1, découpage applicatif, périmètre du POC | S2 | Décidée | C2, C10 | D01 | [D02-decoupage-applicatif.md](D02-decoupage-applicatif.md) |
| D03 | Hébergement et orchestration AWS | S3 | Candidate (XP en D14) | C5, C2 | D02 | [D03-hebergement-orchestration.md](D03-hebergement-orchestration.md) |
| D04 | Stack back et front | S3 | Décidée | C2, C9 | D02 | [D04-stack.md](D04-stack.md) |
| D05 | Communication interservice (sync/async, broker) | S4 | Candidate (XP en D14) | C2, C1 | D02, D03 | [D05-communication-interservice.md](D05-communication-interservice.md) |
| D06 | Authentification et autorisation | S4 | Candidate (XP en D14) | C8, C2 | D03, D04 | [D06-authentification-autorisation.md](D06-authentification-autorisation.md) |
| D07 | Données (SGBD, stockage des photos) | S4 | Décidée | C2 | D02, D03 | [D07-donnees.md](D07-donnees.md) |
| D08 | Exposition, TLS, domaine | S4 | Candidate (XP en D14) | C8 | D03 | [D08-exposition-tls-domaine.md](D08-exposition-tls-domaine.md) |
| D09 | Observabilité | S4 | Décidée | C5, C6 | D03 | [D09-observabilite.md](D09-observabilite.md) |
| D10 | Forge git, CI/CD, registry, déploiement, portes DevSecOps | S4 | Décidée | C4, C7, C8, C6 | D03, D04 | [D10-forge-ci-cd.md](D10-forge-ci-cd.md) |
| D11 | IaC, allumage/extinction, garde-fou coût | S4 | Décidée | C5, C10 | D03, D10 | [D11-iac-allumage-cout.md](D11-iac-allumage-cout.md) |
| D12 | Au moins 4 indicateurs qualité et outils de mesure | S5 | Décidée | C6 | D04, D09, D10 | [D12-indicateurs-qualite.md](D12-indicateurs-qualite.md) |
| D13 | Stratégie de test et outil de charge | S5 | Décidée | C7, C5 | D01, D04, D10 | [D13-strategie-tests-charge.md](D13-strategie-tests-charge.md) |
| D14 | Périmètre des expérimentations bac à sable | S5 | Décidée | C1 | D03, D05, D06, D09, D10 | [D14-experimentations.md](D14-experimentations.md) |

## Synthèse des décisions tranchées

Le choix retenu et sa raison, en bref. Le détail (options, avis, conséquences) est dans chaque fichier.

| ID | Choix retenu | Raison |
|---|---|---|
| D01 | Fonctionnalité du POC : mise en vente d'un article avec contrôle automatisé de publication (option « A enrichie »). | C'est le cœur de la valeur de Collector (la confiance) ; elle porte un échange asynchrone, un contrat versionné qui servira de branchement à l'outil de fraude, et 5 bonnes pratiques de sécurité démontrables. |
| D02 | 4 services : Place de marché, Transactions, Interactions, Contrôle de publication ; les 4 sont déployés dans le POC (option P2), premier sprint en walking skeleton WS-2. | Isoler la finance dès la V1, faire évoluer et mettre à l'échelle chaque domaine séparément ; déployer les 4 services montre l'architecture cible complète. Choix fait contre la recommandation de l'architecte (monolithe modulaire), en connaissance du surcoût. |
| D03 | Amazon EKS (node group géré, HPA et KEDA), statut Candidate ; repli ECS sur Fargate ; comparaison des deux en bac à sable. | Orchestrateur managé conforme à la consigne, mise à l'échelle par domaine démontrable ; l'expérimentation (critères X1 à X21) tranche entre EKS et Fargate sur des mesures, sous les contraintes du Learner Lab. |
| D04 | Go pour les 4 services ; React, TypeScript et MUI pour le front statique. | Rester dans le périmètre des experts de l'équipe (Go au back, React + TypeScript + MUI au front) ; Go donne en plus des binaires légers qui démarrent vite, utile à l'autoscaling. La montée en compétence React est inscrite comme compétence à acquérir (C9). |
| D05 | RabbitMQ en conteneur dans EKS (statut Candidate), NATS JetStream en repli ; outbox côté émetteur, idempotence côté consommateur, topologie posée par l'IaC ; « réserver » en HTTP interne. | Amazon MQ et MSK sont indisponibles dans le lab et les files managées n'identifient pas chaque service sous LabRole ; RabbitMQ interdit à un service de créer une file, met nativement à l'écart les messages en échec et répond à l'expérimentation citée par la consigne (broker dans un orchestrateur). |
| D06 | Keycloak en conteneur dans EKS (statut Candidate), Cognito en repli ; application React statique, OIDC + PKCE, jetons de 5 min en mémoire. | Seul candidat qui porte une vraie audience par service dans les jetons (ES-38) et impose le second facteur aux seuls admins ; identique en local et sur AWS. Le BFF est reporté en V1 : risque résiduel jugé acceptable par la sécurité. |
| D07 | Une instance RDS PostgreSQL recréée à chaque session avec un jeu de démo, une base et deux rôles par service ; réservation par mise à jour conditionnelle ; photos ré-encodées par l'API puis stockées dans un bucket S3 privé. | Coût nul entre sessions et isolation des données par service sans multiplier les instances ; atomicité de la réservation garantie par la base ; le ré-encodage neutralise les fichiers piégés sans ouvrir S3 en écriture au navigateur. |
| D08 | Domaine déjà possédé par Martin, sous-domaine délégué à Route 53, certificat ACM, ALB créé par Terraform, front nginx sur la même origine que l'API (statut Candidate). | CloudFront est indisponible dans le lab ; un nom stable garde l'émetteur Keycloak et les URL de démo identiques d'une session à l'autre, et ACM évite de gérer des certificats dans le cluster. Route 53 et ACM sont sondés dans le lab avant usage. |
| D09 | Prometheus et Grafana dans le cluster pour les métriques, CloudWatch Logs via Fluent Bit pour les journaux et l'audit. | Montrer l'autoscaling en direct pendant la démo, tout en gardant une trace d'audit hors du cluster qui survit à chaque extinction ; pile allégée pour tenir sur deux nœuds. |
| D10 | GitHub public, images sur GHCR signées par cosign, déploiement GitOps par Flux (filet : déploiement depuis le poste), portes DevSecOps bloquantes. | Sans fédération OIDC vers AWS, seul un déploiement tiré par le cluster rend la livraison continue automatique sans identifiants AWS dans la CI ; le dépôt public lève les quotas de CI et de registre. |
| D11 | Terraform épinglé, état local chiffré par l'appareil, deux couches persistante et éphémère, scripts up/down/status/demo, destruction programmée et inventaire bloquant. | Le budget du Learner Lab interdit tout fonctionnement continu : l'environnement doit se recréer en ~40 min et disparaître sans résidu ; l'état local résiste à l'expiration des identifiants et reste hors de portée de LabRole. |
| D12 | Sept indicateurs (critères d'acceptation couverts, latence p95 et mise à l'échelle, fiabilité, couverture, complexité et duplication, vulnérabilités, DORA réduit) mesurés par les outils de la CI ; couverture « pas de baisse » à 0,5 point près. | Couvrir les quatre objectifs de qualité de la grille plus la sécurité, chaque indicateur relié à une forme de dette ; outils natifs pour éviter un service tiers et un jeton sur un dépôt public. |
| D13 | Stratégie à 15 types de tests avec tests négatifs de sécurité ; k6 depuis le poste à travers l'ALB, jetons préparés juste avant le tir ; limite de débit relevée pour la seule adresse de Martin pendant le tir. | Prouver le parcours métier et la montée en charge sans fausser la mesure (générateur hors du cluster) ni affaiblir la sécurité de l'environnement de démonstration ouvert au public. |
| D14 | Sonde du compte puis trois expérimentations : EKS contre ECS Fargate, RabbitMQ dans l'orchestrateur, Keycloak + ALB + ACM + Route 53 ; critères de sécurité bloquants ; du 2026-10-27 au 2026-11-02. | Fonder l'adoption d'EKS (ou son rejet) sur des mesures comparées, conformément à C1, en tenant le calendrier et le budget du Learner Lab ; l'IaC du bac à sable est reprise par le POC. |
