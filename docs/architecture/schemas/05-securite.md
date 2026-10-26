# Schéma 05 — Sécurité intégrée : frontières de confiance et contrôles

Document parent : [architecture-technique.md](../architecture-technique.md) § 9.5. Frontières FC1 à FC11 : [exigences-securite.md](../../securite/exigences-securite.md) § 3 et architecture § 2.3. Mesures : MS (D03), MC (D05), MA (D06), MD (D07), ME (D08), MO (D09), MP (D10), MI (D11).
Statut : D03, D05, D06 et D08 sont **Candidate** ; les critères de sécurité bloquants du bac à sable (X16, X12/X19, XC2, XC7, XC9, XE4/ME8) conditionnent leur adoption ([D14](../../decisions/D14-experimentations.md)).

```mermaid
flowchart TB
  classDef zone fill:#fdecea,stroke:#b71c1c,color:#000
  classDef ok fill:#d8f0d8,stroke:#2e7d32,color:#000
  classDef ecart fill:#fff3c4,stroke:#a67c00,color:#000

  subgraph INTERNET["Internet, non fiable"]
    NAV["Navigateur<br/>CSP stricte, jetons en mémoire,<br/>aucune décision de sécurité"]
    ATT["Balayages, rafales"]
  end

  subgraph FC1["FC1 — Entrée publique"]
    ALB["ALB : TLS 1.2+ ACM, 80 redirigé, HSTS<br/>liste blanche par hôte et chemin, 404 par défaut<br/>ouvert à la seule adresse de Martin hors démonstration"]
  end

  subgraph FC2["FC2 — Identité"]
    KC["Keycloak auth.‹domaine›<br/>PKCE, MFA admin, force brute,<br/>/admin et realm master non routés"]
  end

  subgraph CLUSTER["Cluster : politiques réseau en refus par défaut, PSS restricted"]
    subgraph NSP["Namespaces applicatifs"]
      PDM["Place de marché<br/>port public · port interne réserver"]
      TRX["Transactions"]
      ITR["Interactions"]
      CTL["Contrôle"]
    end
    RMQ["RabbitMQ — FC11<br/>une identité par service,<br/>configure vide, user-id vérifié"]
    OPS["Prometheus, Grafana, Fluent Bit<br/>ClusterIP, port-forward seul"]
  end

  subgraph FC8["FC8 — Données"]
    RDS[("RDS : chiffré, TLS verify-full,<br/>une base, deux rôles par service")]
    S3[("S3 : privé, SSE-S3, TLS imposé,<br/>URL présignée 15 min au plus")]
    CWL[("CloudWatch Logs<br/>audit hors du cluster")]
  end

  subgraph ADMIN["Plan d'administration"]
    POSTE["Poste : API EKS en /32,<br/>secrets par l'entrée standard"]
    IMDS["IMDSv2, LabRole partagé<br/>réservé à PDM, Fluent Bit, autoscaler"]
  end

  subgraph FC9["FC9 — Livraison"]
    CI["GitHub Actions : portes bloquantes<br/>aucun identifiant AWS"]
    REG["GHCR : images signées cosign"]
  end

  NAV -->|HTTPS| ALB
  ATT -.->|429, 404| ALB
  ALB -->|NodePort depuis le groupe de l'ALB seul| PDM
  ALB --> TRX
  ALB --> ITR
  ALB -->|liste blanche| KC
  TRX ==>|"jeton de service aud place-de-marche,<br/>scope reserver, azp transactions<br/>+ jeton de l'acheteur revérifié"| PDM
  PDM -.->|AMQP confiné| RMQ
  CTL -.-> RMQ
  ITR -.-> RMQ
  PDM --> RDS
  TRX --> RDS
  ITR --> RDS
  PDM -->|rôle du nœud| S3
  NAV -->|URL présignée| S3
  OPS -->|masquage, aiguillage par métadonnées| CWL
  POSTE --> CLUSTER
  IMDS -.- PDM
  CI --> REG
  REG -->|vérifié par up.sh| CLUSTER

  class ALB,KC,RMQ,RDS,S3,CWL,CI,REG ok
  class IMDS ecart
```

Légende : vert, contrôle démontré dans le POC (sous réserve des critères du bac à sable) ; jaune, écart connu du Learner Lab, borné et documenté. Trait épais : appel synchrone authentifié ; pointillé : canal.

| Contrôle | Où | Preuve prévue |
|---|---|---|
| Authentification utilisateur et rôles | Keycloak (OIDC, PKCE), vérification `jwx` dans `libs/platform/` à chaque requête, propriété vérifiée dans chaque service | Tests négatifs MA3 à MA6, ES-03, ES-04 |
| Service à service | Jeton `client_credentials` de Transactions + `X-Collector-User-Token` ; route servie sur un port interne distinct, `ClusterIP`, absent de l'ALB | MC7, MA6, XE4, X17 |
| Canal | Identités RabbitMQ par service, droits par échange et par file, `user-id` vérifié, validation stricte avant décodage | XC2, XC4, XC9 |
| Réseau | Nœuds et base en sous-réseaux privés ; politiques réseau en refus par défaut, entrée et sortie ; NodePorts ouverts au seul groupe de l'ALB | X12, X17, X19 |
| Chiffrement en transit | TLS public jusqu'à l'ALB ; TLS vers RDS ; en clair dans le VPC (ALB → NodePort, AMQP, « réserver ») : confinement réseau, écart V1 | XE3, MD3, ME9 |
| Chiffrement au repos | RDS chiffré, bucket SSE-S3, disques des nœuds chiffrés | MI9, inventaire |
| Secrets | Générés à chaque `up`, injectés par l'entrée standard, un Secret par service, jamais dans le dépôt, les valeurs Helm ni l'état Terraform | X20, MI3, MI4 |
| Admission | PSS « restricted », Services `LoadBalancer` et NodePorts hors liste refusés, pas de jeton de compte de service monté | X18 |
| Chaîne de livraison | Portes bloquantes, images signées et vérifiées, Flux limité aux namespaces applicatifs | MP1 à MP10 |
| Audit | Journaux JSON masqués, groupe d'audit CloudWatch aiguillé par les métadonnées Kubernetes | MO2, MO3, X21 |
