# Schéma 01 — Vue logique du POC : services, canal, contrats

Document parent : [architecture-technique.md](../architecture-technique.md) § 3. Décisions : [D02](../../decisions/D02-decoupage-applicatif.md), [D04](../../decisions/D04-stack.md), [D05](../../decisions/D05-communication-interservice.md), [D06](../../decisions/D06-authentification-autorisation.md), [D07](../../decisions/D07-donnees.md).
Statut : D05 et D06 sont **Candidate**, sous réserve du bac à sable (XP02, XP03 de [D14](../../decisions/D14-experimentations.md)).

```mermaid
flowchart LR
  classDef construit fill:#d8f0d8,stroke:#2e7d32,color:#000
  classDef canal fill:#e3ecfa,stroke:#1f4e9c,color:#000
  classDef fictif fill:#fff3c4,stroke:#a67c00,color:#000
  classDef documente fill:#f2f2f2,stroke:#9e9e9e,color:#555,stroke-dasharray: 5 5

  WEB["Front web<br/>React, TypeScript, MUI, statique<br/>OIDC + PKCE, jetons en mémoire"]
  EP["Point d'entrée HTTPS<br/>‹domaine› et auth.‹domaine›"]
  KC["Keycloak, realm collector<br/>client collector-web : PKCE<br/>client transactions : client credentials"]

  subgraph SVC["Services Go, un par domaine, net/http sans framework"]
    PDM["Place de marché<br/>Annonces et catalogue, Comptes minimal<br/>port public /v1 · port interne « réserver »<br/>outbox + relais"]
    TRX["Transactions<br/>réduit à la réservation"]
    CTL["Contrôle de publication<br/>sans base, règles v1 puis v2"]
    ITR["Interactions minimal<br/>notification au vendeur"]
  end

  subgraph RMQ["RabbitMQ, vhost collector, topologie posée par le chart"]
    XD["controle.demandes<br/>échange direct, file du même nom"]
    XR["controle.resultats<br/>échange direct, file pdm.resultats-controle"]
    XE["pdm.evenements<br/>échange topic, clé publication.decidee.v1"]
    QI["file interactions.publication-decidee"]
    QT["file test.publication-decidee<br/>posée puis retirée par l'exploitation"]
  end

  subgraph RDS["RDS PostgreSQL, une base et deux rôles par service"]
    BM[("marche<br/>articles, révisions, outbox")]
    BT[("transactions<br/>commandes")]
    BI[("interactions<br/>notifications, unicité par décision")]
    BK[("keycloak")]
  end
  S3[("S3 privé : photos ré-encodées<br/>lues par URL présignée")]

  ADP["Adaptateur de fraude, V1"]
  TIERS["Paiement, emails, outil de fraude, V1"]

  WEB -->|HTTPS| EP
  EP -->|/v1 de chaque service| PDM
  EP --> TRX
  EP --> ITR
  EP -->|pages de connexion| KC
  WEB -->|image par URL présignée| S3
  TRX -->|jeton de service| KC
  TRX ==>|"réserver v1 : HTTP/JSON interne, OpenAPI<br/>jeton de service + jeton de l'acheteur"| PDM
  PDM -.->|demande de contrôle v1| XD
  XD -.-> CTL
  CTL -.->|résultat de contrôle v1| XR
  XR -.-> PDM
  PDM -.->|décision de publication v1| XE
  XE -.-> QI
  QI -.-> ITR
  XE -.-> QT
  PDM --> BM
  TRX --> BT
  ITR --> BI
  KC --> BK
  PDM -->|photos ré-encodées| S3
  CTL -.-|V1| ADP
  ADP -.-|V1| TIERS
  TRX -.-|V1| TIERS

  class WEB,EP,KC,PDM,TRX,CTL,ITR,BM,BT,BI,BK,S3 construit
  class XD,XR,XE,QI canal
  class QT fictif
  class ADP,TIERS documente
```

Légende :
- Vert : construit et déployé dans le POC. Bleu : objets du canal, déclarés dans le fichier de définitions de RabbitMQ (aucun service ne les crée). Jaune : abonnement de test. Gris en pointillé : V1, documenté.
- Trait plein : HTTP synchrone. Trait épais : seul appel synchrone entre services, authentifié. Pointillé fléché : message par le canal.
- Contrats : `contracts/messages/` (JSON Schema, un par message et par version, `additionalProperties: false`) et `contracts/api/` (OpenAPI par service, dont l'OpenAPI interne de « réserver »). Un test de contrat par paire producteur-consommateur ([D13](../../decisions/D13-strategie-tests-charge.md)).
- Enveloppe commune des messages : identifiant unique, type, version, date, identifiant de corrélation ; contexte de trace W3C dans les en-têtes AMQP.
- Chaque service se connecte au canal avec sa propre identité RabbitMQ : écriture sur le seul échange qu'il produit, lecture sur ses seules files, aucun droit `configure` (MC1, MC2 de D05).
