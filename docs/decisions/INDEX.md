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
| D05 | Communication interservice (sync/async, broker) | S4 | À instruire | C2, C1 | D02, D03 | |
| D06 | Authentification et autorisation | S4 | À instruire | C8, C2 | D03, D04 | |
| D07 | Données (SGBD, stockage des photos) | S4 | À instruire | C2 | D02, D03 | |
| D08 | Exposition, TLS, domaine | S4 | À instruire | C8 | D03 | |
| D09 | Observabilité | S4 | À instruire | C5, C6 | D03 | |
| D10 | Forge git, CI/CD, registry, déploiement, portes DevSecOps | S4 | À instruire | C4, C7, C8, C6 | D03, D04 | |
| D11 | IaC, allumage/extinction, garde-fou coût | S4 | À instruire | C5, C10 | D03, D10 | |
| D12 | Au moins 4 indicateurs qualité et outils de mesure | S5 | À instruire | C6 | D04, D09, D10 | |
| D13 | Stratégie de test et outil de charge | S5 | À instruire | C7, C5 | D01, D04, D10 | |
| D14 | Périmètre des expérimentations bac à sable | S5 | À instruire | C1 | D03, D05, D06, D09, D10 | |

## Synthèse des décisions tranchées

Le choix retenu et sa raison, en bref. Le détail (options, avis, conséquences) est dans chaque fichier.

| ID | Choix retenu | Raison |
|---|---|---|
| D01 | Fonctionnalité du POC : mise en vente d'un article avec contrôle automatisé de publication (option « A enrichie »). | C'est le cœur de la valeur de Collector (la confiance) ; elle porte un échange asynchrone, un contrat versionné qui servira de branchement à l'outil de fraude, et 5 bonnes pratiques de sécurité démontrables. |
| D02 | 4 services : Place de marché, Transactions, Interactions, Contrôle de publication ; les 4 sont déployés dans le POC (option P2), premier sprint en walking skeleton WS-2. | Isoler la finance dès la V1, faire évoluer et mettre à l'échelle chaque domaine séparément ; déployer les 4 services montre l'architecture cible complète. Choix fait contre la recommandation de l'architecte (monolithe modulaire), en connaissance du surcoût. |
| D03 | Amazon EKS (node group géré, HPA et KEDA), statut Candidate ; repli ECS sur Fargate ; comparaison des deux en bac à sable. | Orchestrateur managé conforme à la consigne, mise à l'échelle par domaine démontrable ; l'expérimentation (critères X1 à X21) tranche entre EKS et Fargate sur des mesures, sous les contraintes du Learner Lab. |
| D04 | Go pour les 4 services ; React, TypeScript et MUI pour le front statique. | Rester dans le périmètre des experts de l'équipe (Go au back, React + TypeScript + MUI au front) ; Go donne en plus des binaires légers qui démarrent vite, utile à l'autoscaling. La montée en compétence React est inscrite comme compétence à acquérir (C9). |
