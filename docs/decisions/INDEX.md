# Index des décisions

Statuts et circuit de décision : voir [PILOTAGE.md](../PILOTAGE.md). Gabarit : [_GABARIT.md](_GABARIT.md).
Le lien est renseigné dès que le fichier de la décision existe.

Sessions : S1 = D01 · S2 = D02 · S3 = D03 + D04 · S4 = D05 à D11 · S5 = D12 à D14.

| ID | Titre | Session | Statut | Critères grille | Dépend de | Fichier |
|---|---|---|---|---|---|---|
| D01 | Exigences reformulées et fonctionnalité métier du POC | S1 | Décidée | C3, C2, C7, C5 | — | [D01-fonctionnalite-poc.md](D01-fonctionnalite-poc.md) |
| D02 | Architecture cible V1, découpage applicatif, périmètre du POC | S2 | À instruire | C2, C10 | D01 | |
| D03 | Hébergement et orchestration AWS | S3 | À instruire | C5, C2 | D02 | |
| D04 | Stack back et front | S3 | À instruire | C2, C9 | D02 | |
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
