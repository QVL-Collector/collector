# Schéma 03 — Livraison : GitHub, GHCR, Flux, EKS

Document parent : [architecture-technique.md](../architecture-technique.md) § 9.4. Décisions : [D10](../../decisions/D10-forge-ci-cd.md), [D11](../../decisions/D11-iac-allumage-cout.md), [D07](../../decisions/D07-donnees.md) (migrations).
Le détail des portes et du cycle DevSecOps vit dans `docs/processus/` (D10, « Conséquences ») ; ce schéma n'en montre que l'enchaînement.

```mermaid
flowchart LR
  classDef ci fill:#e3ecfa,stroke:#1f4e9c,color:#000
  classDef cluster fill:#d8f0d8,stroke:#2e7d32,color:#000
  classDef poste fill:#fff3c4,stroke:#a67c00,color:#000

  DEV["Branche de travail"]
  PR["Pull request vers main"]

  subgraph GHA["GitHub Actions, dépôt public"]
    PORTES["Portes bloquantes<br/>gitleaks, lint et types, contrôles propres,<br/>tests unitaires et d'intégration,<br/>govulncheck, Semgrep, Trivy fs, config, image"]
    OK{"ci-ok"}
    BUILD["Sur main : une image par service,<br/>front et Keycloak, construite une fois<br/>étiquette sha-commit, SBOM"]
    SIGN["cosign sans clé<br/>identité du workflow de main"]
    PUBLI["Job de publication<br/>écrit les empreintes sur env/demo"]
  end

  GHCR[("GHCR<br/>images et signatures")]
  ENV["Branche env/demo<br/>empreintes seules"]

  subgraph POSTE["Poste de Martin, à chaque session"]
    UP["up.sh<br/>Terraform, Secrets, cosign verify,<br/>installation de Flux"]
    P1["Filet P1<br/>helm upgrade depuis le poste"]
  end

  subgraph EKS["Cluster EKS, namespaces applicatifs"]
    FLUX["Flux<br/>source-controller, helm-controller<br/>compte de service limité"]
    MIG["Job de migration goose<br/>crochet pre-install, pre-upgrade"]
    APP["Services, front, Keycloak, RabbitMQ<br/>déployés par empreinte"]
  end

  DEV --> PR --> PORTES --> OK
  OK -->|vert, fusion| BUILD
  OK -->|rouge| PR
  BUILD --> SIGN --> GHCR
  SIGN --> PUBLI --> ENV
  UP -->|vérifie les signatures| GHCR
  UP -->|installe| FLUX
  FLUX -->|tire l'état voulu| ENV
  FLUX --> MIG
  MIG -->|succès| APP
  APP -->|images tirées par empreinte| GHCR
  P1 -.->|si Flux indisponible| APP

  class PORTES,OK,BUILD,SIGN,PUBLI ci
  class FLUX,MIG,APP cluster
  class UP,P1 poste
```

Lecture :
- **La CI ne détient aucun identifiant AWS ni du cluster** : elle construit, analyse, signe et publie des empreintes. Le cluster tire l'état voulu (modèle « pull »), ce qui rend la livraison continue compatible avec une API du cluster fermée à Internet (MS1) et sans fédération OIDC vers AWS.
- **Promotion sans reconstruction** : l'image construite sur `main` est celle qui tourne sur EKS, en local (k3d) et en acceptation.
- **Une migration en échec arrête le déploiement** (crochet Helm, D07).
- **Frontières de confiance de la chaîne (FC9)** : compte GitHub de Martin (double facteur), job de publication (seul à écrire `env/demo`), droits de Flux (namespaces applicatifs, aucune ressource de cluster). Mesures MP1 à MP10 de D10.
