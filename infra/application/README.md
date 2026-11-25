# infra/application — Chart de l'application (US-007)

Un seul chart pour les quatre services Go, le front et Keycloak ([D03](../../docs/decisions/D03-hebergement-orchestration.md) : « un chart Helm commun »), une seule `HelmRelease` et un seul retour arrière. Chaque objet porte son namespace (un par service, MS4) ; le canal reste le chart [infra/canal](../canal/README.md), seconde release dont l'application dépend. Seul le fichier de valeurs change entre k3d et EKS (X2).

| Fichier | Rôle |
|---|---|
| `values.yaml` | Valeurs communes, sans secret ni image |
| `valeurs-aws.yaml`, `valeurs-k3d.yaml` | Domaine, sources de l'ALB, destinations de la base, TLS, accès AWS de la Place de marché |
| `deploy/demo/empreintes.yaml` (branche `env/demo`) | `images.<service>` : seule source des images, par empreinte, écrite par le job de publication |

## Valeurs

| Clé | Rôle |
|---|---|
| `registreAttendu` | Préfixe imposé à toute image (`ghcr.io/qvl-collector/`) ; `up-flux.sh --images-locales` le remplace sur k3d |
| `images.<service>` | Rempli par `deploy/demo/empreintes.yaml` d'`env/demo` (avec `source`, commit de `main` des images, sans effet sur le rendu) |
| `exposition.sources` | Sources admises vers les NodePorts et mandataires de confiance de Keycloak (`KC_PROXY_TRUSTED_ADDRESSES`) ; vide : aucune entrée extérieure |
| `base.destinations` | Pairs NetworkPolicy vers PostgreSQL ; vide refusé au rendu (sortie vers toute adresse) |
| `base.autorites` | ConfigMap `ca.pem` du paquet d'autorités RDS, posé par `up-flux.sh` sur AWS |
| `placeDeMarche.accesAws` | Seul pod à joindre l'IMDS (rôle du nœud) et HTTPS sortant, pour les photos (MS3, X16) ; étiquette `collector/acces-aws`, seule exception admise par `tools/sortie-sans-imds.jq` |
| `placeDeMarche.rejeu.depuis` | Rend le Job de rejeu de l'outbox (MC9), lancé par l'exploitation |
| `migrations.imageAttente` | Conteneur d'initialisation qui attend la base : une politique réseau n'admet un pod neuf qu'après quelques secondes (constaté sur k3d), et `goose` ne réessaie pas |
| `<service>.nodePort` | Repris des sorties Terraform (`nodeport_front`, `nodeports_application`) par `up-flux.sh` sur AWS |

AWS (`valeurs-aws.yaml`, `cidr_vpc` 10.30.0.0/16) : sources = sous-réseaux publics de l'ALB 10.30.0.0/24 et 10.30.1.0/24, base = sous-réseaux privés 10.30.16.0/20 et 10.30.32.0/20, `accesAws` actif, descente HPA en 60 s (X7 bis de XP01 : 10 min 16 s avec 300 s). k3d (`valeurs-k3d.yaml`) : domaine `collector.localhost`, base locale sans TLS, une réplique par service, IMDS absente.

## Contrat avec `tools/up-flux.sh`

Avant le chart : namespaces, refus par défaut et DNS ([infra/k8s/application.yaml](../k8s/application.yaml)), bases amorcées, puis un Secret par usage, dans le seul namespace qui le monte ([secrets-autorises.txt](../k8s/secrets-autorises.txt)) :

| Namespace | Secret (clés) | Monté par |
|---|---|---|
| `place-de-marche`, `interactions`, `transactions` | `base-execution` (`BASE_DE_DONNEES_URL`) | Deployment |
| idem | `base-migration` (`BASE_DE_DONNEES_URL`) | Job de migration (crochet `pre-install`, `pre-upgrade`) et Job de rejeu |
| `place-de-marche`, `controle`, `interactions` | `canal-amqp` (`CANAL_MOT_DE_PASSE`) | Deployment |
| `transactions` | `oidc-client` (`OIDC_CLIENT_SECRET`) | Deployment |
| `identite` | `keycloak` (`KC_DB_*`, `KC_BOOTSTRAP_ADMIN_*`, `SECRET_CLIENT_TRANSACTIONS`) | Keycloak |

ConfigMap `autorites-rds` (clé `ca.pem`) sur AWS, monté en `/etc/autorites-rds` ; facultatif sur k3d.

## Sécurité des pods et réseau

- Non root, système de fichiers en lecture seule, `drop: [ALL]`, `seccomp` `RuntimeDefault`, aucun jeton de compte de service monté ; PSS `restricted` sur chaque namespace.
- Image refusée au rendu si elle n'est pas épinglée par empreinte ou hors de `registreAttendu`.
- HPA : Place de marché et front, 2 à 6 (D03) ; Contrôle mis à l'échelle par KEDA (US-026).
- Flux ouverts ([templates/politiques-reseau.yaml](templates/politiques-reseau.yaml)) : ALB vers 8080 du front, des API et de Keycloak ; 8081 (« réserver ») depuis Transactions seul, jamais exposé ; base pour les services, leurs migrations et Keycloak ; 5672 du canal pour la Place de marché, le Contrôle et Interactions ; Keycloak pour les trois API. IMDS et HTTPS sortant pour la seule Place de marché (`placeDeMarche.accesAws`, MS3, X16).

## Contrôles hors ligne

```bash
helm lint infra/application -f infra/application/valeurs-aws.yaml -f <empreintes>
helm template application infra/application -n collector -f infra/application/valeurs-k3d.yaml -f <empreintes>
```

Rejeu de l'outbox (MC9) : `--set placeDeMarche.rejeu.depuis=<RFC3339>` rend un Job `rejouer` sous le rôle de migration.
