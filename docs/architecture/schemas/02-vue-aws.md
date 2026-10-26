# Schéma 02 — Vue technique AWS du POC

Document parent : [architecture-technique.md](../architecture-technique.md) § 9.2. Décisions : [D03](../../decisions/D03-hebergement-orchestration.md), [D07](../../decisions/D07-donnees.md), [D08](../../decisions/D08-exposition-tls-domaine.md), [D09](../../decisions/D09-observabilite.md), [D11](../../decisions/D11-iac-allumage-cout.md).
Statut : EKS (D03) et l'exposition Route 53 / ACM / ALB (D08) sont **Candidate**, sous réserve du bac à sable (XP01, XP03 de [D14](../../decisions/D14-experimentations.md)). Repli de D03 : ECS sur Fargate.

```mermaid
flowchart TB
  classDef persistant fill:#ede3f7,stroke:#6a3d9a,color:#000
  classDef ephemere fill:#d8f0d8,stroke:#2e7d32,color:#000
  classDef externe fill:#f2f2f2,stroke:#9e9e9e,color:#333

  NAV["Navigateur"]
  POSTE["Poste de Martin<br/>Terraform, scripts up et down, kubectl"]
  GH["GitHub<br/>branche env/demo"]
  GHCR[("GHCR<br/>images signées")]

  subgraph AWS["Compte Learner Lab, us-east-1"]
    R53["Route 53<br/>zone du sous-domaine délégué"]
    ACM["ACM<br/>certificat ‹domaine›, auth.‹domaine›"]
    CP["Plan de contrôle EKS 1.35+<br/>API limitée à l'adresse du poste"]
    SM["Secrets Manager<br/>secrets de session"]
    CWL[("CloudWatch Logs<br/>applicatif 7 j, audit 30 j")]
    S3[("S3 : photos<br/>privé, SSE-S3")]
    SCH["EventBridge Scheduler<br/>destruction programmée"]

    subgraph VPC["VPC sur us-east-1a et us-east-1b"]
      subgraph PUB["Sous-réseaux publics"]
        ALB["ALB HTTPS, créé par Terraform<br/>TLS 1.2+, règles en liste blanche"]
        NAT["Passerelle NAT unique"]
      end
      subgraph PRIV["Sous-réseaux privés : node group géré t3.medium, 2 à 4 nœuds"]
        subgraph APPNS["Namespaces applicatifs, un par service"]
          FRONT["Front nginx-unprivileged"]
          PDM["Place de marché<br/>HPA 2 à 6"]
          TRX["Transactions, 1"]
          ITR["Interactions, 1"]
          CTL["Contrôle<br/>KEDA 1 à 3"]
        end
        subgraph SOCLE["Socle dans le cluster, une instance chacun"]
          KC["Keycloak"]
          RMQ["RabbitMQ"]
        end
        subgraph OPS["Exploitation dans le cluster"]
          FLUX["Flux<br/>source et helm controllers"]
          OBS["Prometheus, Grafana<br/>Fluent Bit en DaemonSet"]
          SYS["metrics-server, KEDA<br/>Cluster Autoscaler"]
        end
      end
      subgraph DATA["Sous-réseaux privés de données"]
        RDS[("RDS PostgreSQL db.t3.micro<br/>Single-AZ, chiffré, TLS imposé")]
      end
      S3EP["Point de terminaison S3<br/>de type passerelle"]
    end
  end

  NAV -->|HTTPS 443| ALB
  R53 -->|alias| ALB
  ACM -.-|certificat| ALB
  ALB -->|NodePort| FRONT
  ALB -->|NodePort| PDM
  ALB -->|NodePort| TRX
  ALB -->|NodePort| ITR
  ALB -->|auth.‹domaine›, liste blanche| KC
  TRX ==>|réserver, port interne| PDM
  PDM -.- RMQ
  CTL -.- RMQ
  ITR -.- RMQ
  PDM --> RDS
  TRX --> RDS
  ITR --> RDS
  KC --> RDS
  PDM -->|rôle du nœud| S3EP
  S3EP --> S3
  NAV -->|URL présignée| S3
  OBS -->|journaux| CWL
  FLUX -->|état voulu| NAT
  NAT --> GH
  NAT -->|images| GHCR
  POSTE -->|kubectl, adresse /32| CP
  POSTE -->|secrets de session| SM
  CP -.- PRIV
  SCH -.->|fin de session| VPC

  class R53,ACM,S3,CWL persistant
  class CP,ALB,NAT,FRONT,PDM,TRX,ITR,CTL,KC,RMQ,FLUX,OBS,SYS,RDS,S3EP,SCH,SM ephemere
  class NAV,POSTE,GH,GHCR externe
```

Légende :
- Violet : couche persistante (`infra/persistant/`), appliquée une fois : zone et certificat, bucket des photos, groupes de journaux. Vert : couche éphémère (`infra/ephemere/` puis Helm), créée par `up` et détruite par `down` à chaque session.
- Le front et les API partagent l'origine `‹domaine›` ; Keycloak a sa propre origine `auth.‹domaine›`. Les photos ne passent par aucune route de l'ALB.
- Non routés par l'ALB : « réserver », consoles et ports de gestion (Keycloak, RabbitMQ), Prometheus, Grafana, ports de métriques.
- Aucun registre AWS : les images viennent de GHCR, par la NAT. Aucun load balancer créé par Kubernetes.
- Identité d'exécution unique (`LabRole`, rôle des nœuds) : écart ES-38, borné par la réservation de l'IMDS (MS3). Voir [05-securite.md](05-securite.md).
