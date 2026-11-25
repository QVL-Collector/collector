# US-007 — Revue de code

| Champ | Valeur |
|---|---|
| US | [US-007 — Livraison continue : images signées et déploiement par Flux](../../../backlog/US-007-livraison-images-signees-flux.md) ; politiques réseau jugées sur [US-006](../../../backlog/US-006-cloisonnement-cluster.md) |
| Branche | `feat/US-007-livraison-images-signees-flux` (HEAD `c92ce3c`, 1 commit), revue par `git diff origin/main...HEAD` (35 fichiers, +2533/-5) |
| Date | 2026-11-25 |
| Périmètre | chart `infra/application` ; `infra/flux` ; `infra/k8s/{application,base-locale}.yaml`, `secrets-autorises.txt` ; `infra/identite/Dockerfile` ; `infra/ephemere/{application,sorties}.tf` ; `tools/ci/{publier.sh,outils.yml}` ; job `publication` de `.github/workflows/ci.yml` ; `tools/up-flux.sh` ; `tools/github/configurer-depot.sh` ; retouches `infra/canal` ; `.trivyignore` (KSV-0056) ; `infra/README.md` |
| Références | [D03](../../decisions/D03-hebergement-orchestration.md) (MS3 à MS5, X16), [D06](../../decisions/D06-authentification-autorisation.md), [D07](../../decisions/D07-donnees.md) (MD2), [D08](../../decisions/D08-exposition-tls-domaine.md) (liste blanche de l'ALB), [D09](../../decisions/D09-observabilite.md), [D10](../../decisions/D10-forge-ci-cd.md) (R1, P2, filet P1, MP4 à MP6), [D11](../../decisions/D11-iac-allumage-cout.md) ; [cicd.md](../../processus/cicd.md) ; précédent [US-004](US-004-revue-code.md) |
| Hors revue (connu) | Revérification continue des signatures dans le cluster (V1, hors périmètre de l'US) ; passage des paquets GHCR en public (manuel, D10) ; validation Sécurité de KSV-0056 (en cours) |
| Verdict | **Non fusionnable en l'état** (3 bloquantes, 6 majeures, 8 mineures, 6 suggestions) |

## Vérifications exécutées

Aucun appel à l'API AWS, aucune poussée, aucune écriture Git.

| Commande | Résultat |
|---|---|
| `helm lint infra/application -f valeurs-aws.yaml -f <empreintes>` et idem `valeurs-k3d.yaml` (fichier d'empreintes fictif de six images `ghcr.io/qvl-collector/<nom>@sha256:…`) | 0 échec (les deux) |
| `helm template application infra/application -n collector` (les deux fichiers de valeurs) | Rendu OK : 37 objets sur AWS, 36 sur k3d (`sortie-aws` absente) |
| `helm template` sans empreintes, puis avec un registre étranger | Refus attendu : « images.transactions est obligatoire et épinglée par empreinte », puis « hors du registre attendu » |
| `helm lint infra/canal` | 0 échec |
| `terraform fmt -check` (`application.tf`, `sorties.tf`) | Conformes |
| `terraform -chdir=infra/ephemere validate` | `Success! The configuration is valid.` |
| `shellcheck -x -P SCRIPTDIR --severity=style` (image `koalaman/shellcheck:v0.11.0` de `tools/ci/outils.yml`) sur `up-flux.sh`, `publier.sh`, `configurer-depot.sh`, `canal/outils/kubernetes.sh` | 0 constat. Shellcheck ne signale pas les fonctions non définies (voir bloquante 1) |
| `actionlint` (image épinglée) | 0 constat |
| `zizmor --offline .github/workflows/ci.yml` (image épinglée) | « No findings » |
| `bash tools/ci/trivy.sh config . revue-us007` | 0 bloquant (19 LOW, 13 MEDIUM) |
| `bash tools/ci/verifier-exceptions.sh` | 3 exceptions actives (dont KSV-0056), aucune invalide ni échue |
| `flux install --components=source-controller,helm-controller --export` (image `flux-cli:v2.10.1` épinglée de `up-flux.sh`) | RBAC et politiques réseau réellement installés relus (bloquante 3, majeure 3) |
| Filtre `tools/sortie-sans-imds.jq` appliqué au rendu AWS + `infra/k8s/*.yaml` + manifeste Flux, namespace par namespace | `place-de-marche` → `false`, `flux-system` → `false` ; `front`, `interactions`, `identite`, `collector` → `true` (majeure 2) |
| `. tools/lib.sh; type contexte_k8s` (et `git show origin/main:tools/lib.sh`) | `contexte_k8s: not found` : fonction retirée par les retours d'US-004, absente de `main` (bloquante 1) |
| Recherche des usages : `application_active`, `nodeports_application`, `up-flux`, `chemin_natif`, `AMQP_URL` | Voir bloquante 2, mineures 1 et suggestion 2 |
| Variables d'environnement du chart contre `config.go`/`libs/platform` (`CANAL_*`, `OIDC_*`, `BASE_DE_DONNEES_*`, `PLACE_DE_MARCHE_URL_INTERNE`, `RESERVATION_*`) | Noms et clés concordants ; `/interne` servi seulement sur le port 8081 (`serve.go:79`), jamais en NodePort |

## Points forts

- **Job de publication bien confiné** (`ci.yml:430-461`) : seul job en écriture (`contents`, `packages`, `id-token`), `permissions: {}` au niveau du workflow, `persist-credentials: false`, actions épinglées par empreinte, aucun secret AWS, exécuté seulement sur `push` vers `main` après `ci-ok`, concurrence `publication` sans annulation. Construction des images Go retirée du job `go` sur `push` (`ci.yml:255`) : une seule construction par image sur `main` (AC1).
- **Chaîne de signature cohérente** (`publier.sh:73-97`) : Trivy bloquant avant toute poussée, SBOM CycloneDX, poussée, empreinte relue et validée par expression, `cosign sign` puis `attest`, puis `verify` **et** `verify-attestation` avec l'identité exacte du workflow de `main` et l'émetteur GitHub. La même identité est exigée à l'allumage (`up-flux.sh:42-43`, `:85-109`), refus si une seule image échoue (AC4).
- **`env/demo` sans poussée forcée** (`publier.sh:99-118`) : branche reconstruite à partir de son dernier commit, seul fichier d'empreintes conservé (AC2), `git push` simple. Un échec de lecture retombe sur une branche orpheline, que le serveur refuse alors en non-fast-forward : pas d'écrasement possible par le script.
- **Chart durci et fermé par défaut** : empreinte et registre exigés au rendu (`_aides.tpl:8-17`), non root, `readOnlyRootFilesystem`, `drop: [ALL]`, `seccomp`, aucun jeton monté, PSS `restricted` sur les sept namespaces (`k8s/application.yaml`). `base.destinations` vide refusé au rendu (`politiques-reseau.yaml:5-7`). Port 8081 ouvert depuis Transactions seul (`:34-54`, `:121-141`), IMDS réservé au pod étiqueté `collector/acces-aws` (`:142-168`).
- **RBAC du déployeur** (`flux/deployeur.yaml`) : aucun objet de cluster, aucune `Role`/`RoleBinding` créable, Secrets lisibles dans `collector` seul, liaisons par namespace. L'usurpation est limitée au seul compte `deployeur` (`:159-178`), `--no-cross-namespace-refs` et `--default-service-account` sont imposés (`up-flux.sh:311-313`).
- **Retour arrière Flux** conforme à AC5 : `upgrade.remediation.strategy: rollback`, `remediateLastFailure: true`, `cleanupOnFail` (`releases.yaml:23-29`, `:56-64`).
- **Secrets un par usage** (MD2) : rôles `migration`/`execution` par base, `CONNECTION LIMIT`, `statement_timeout`, mots de passe passés par `\getenv` (jamais dans le SQL ni dans le ConfigMap), Secret maître supprimé après l'amorçage (`up-flux.sh:142-262`). `secrets-autorises.txt` mis à jour en conséquence (X20).
- **ALB** : 404 explicite sur `/interne` et sur toute traversée encodée du domaine `auth.` (`application.tf:98-154`), validation Terraform qui interdit de router un chemin `interne`, règles ajoutées à la liste blanche XE4 (`sorties.tf:41-49`).
- **Images Flux et outils épinglés par empreinte** (`up-flux.sh:47-52`, `outils.yml:49-50`), avec un refus si le contrôleur Helm n'est pas épinglé (`up-flux.sh:307`).

## Issues

### Bloquantes

1. `tools/up-flux.sh:36` — **le script s'arrête dès sa première instruction utile.**
   Constat : `contexte_k8s "$CONTEXTE"` appelle une fonction qui n'existe plus. Elle a été supprimée de `tools/lib.sh` par la mineure 5 de la revue d'US-004 et elle est absente de `origin/main`. Sous `set -e`, bash sort en 127 (« command not found ») avant de lire les empreintes. AC3, AC4 et AC6 ne peuvent donc pas être joués, ni sur k3d, ni sur AWS. Shellcheck ne détecte pas ce cas.
   Correction : retirer l'appel. La garde utile est déjà en place : `kc()` et `hm()` passent `--context "$CONTEXTE"`, et `canal/outils/lib.sh` (`exiger_contexte`) filtre `k3d-*` et `*collector*`. Si une liste blanche est voulue ici, réutiliser la même (motifs `k3d-*`, `*collector*`) dans une fonction de `tools/lib.sh`. Rejouer ensuite `up-flux.sh --valeurs k3d` de bout en bout. Dev : devops.

2. `tools/up.sh:287-289`, `infra/ephemere/application.tf:1-5`, `sorties.tf:51` — **sur AWS, l'application ne peut pas être installée ni exposée.**
   Constat : trois défauts se cumulent.
   - **`up.sh` n'appelle pas `up-flux.sh`.** Il installe toujours la release `front` d'US-004 (`infra/front`, Deployment, Service et HPA `front` dans le namespace `front`, NodePort 30080). Le chart `infra/application` rend des objets de même nom dans le même namespace (`front.yaml`). Lancé après `up.sh`, `up-flux.sh` voit donc Helm refuser l'adoption de ces objets (`meta.helm.sh/release-name` vaut `front`, pas `application`) : la `HelmRelease` ne devient jamais `Ready`.
   - **`application_active` n'est jamais passé à `true`.** Aucun script ni `tfvars` ne le fait, donc les règles de l'ALB vers les API et `auth.` restent du code mort. La sortie `nodeports_application` n'est lue nulle part.
   - **Le libellé des AC n'est pas tenu.** AC3, AC6, [D10](../../decisions/D10-forge-ci-cd.md) § Déploiement (« Ordre dans `up.sh` ») et [cicd.md](../../processus/cicd.md) parlent de `up.sh` et de `up.sh --sans-flux`, qui n'existent pas. `tools/README.md` ne mentionne pas `up-flux.sh`.

   Correction :
   - faire appeler `up-flux.sh --contexte "$NOM" --valeurs aws [--sans-flux]` par `up.sh` à la place de `helm upgrade --install front`, avec `--sans-flux` relayé ;
   - retirer la release `front` existante, ou supprimer `infra/front` si le front de l'application la remplace ;
   - passer `-var=application_active=true` dans les `apply` de `up.sh`, `alb.sh` et `down.sh` (à côté de `ALB_FERME`) ;
   - consommer `nodeports_application`, par exemple en `--set` des NodePorts, pour une seule source des ports, ou supprimer la sortie ;
   - documenter le script dans `tools/README.md`.

   Si le découpage en deux scripts est voulu, le tracer dans D10 et dans l'US. Dev : devops.

3. `tools/up-flux.sh:291-308` — **les contrôleurs Flux joignent l'IMDS et obtiennent le rôle du nœud (US-006 AC5, MS3, X16).**
   Constat : `flux install --export` installe dans `flux-system` la politique `allow-egress` (`podSelector: {}`, `egress: [{}]`), que `manifestes_flux` ne filtre pas. Or `imds_limite_sauts` vaut 2 par défaut (`variables.tf:146-150`), parce que l'IMDS doit rester joignable des pods et être filtré par politique réseau. Résultat : source-controller et helm-controller peuvent demander un jeton IMDSv2 puis les identifiants de `LabRole`. source-controller tire du contenu d'Internet, c'est donc la surface la plus exposée. AC5 d'US-006 réserve l'IMDS à la Place de marché, à Fluent Bit et au Cluster Autoscaler. Le contrôle `sortie-sans-imds` le signale : `flux-system` → `false` (filtre rejoué sur le manifeste réel).
   Correction : dans `manifestes_flux`, remplacer la règle de sortie d'`allow-egress` par `- to: [{ipBlock: {cidr: 0.0.0.0/0, except: [169.254.169.254/32]}}]`. Le faire dans le manifeste, et non par un correctif appliqué ensuite, sinon l'`apply --server-side --force-conflicts` suivant le rétablirait. Ajouter un cas négatif au test du filtre (`tools/tests/sortie-sans-imds.sh`) avec la politique Flux d'origine. Dev : devops.

### Majeures

1. `tools/ci/publier.sh:63-72`, `:125-133`, `.github/workflows/ci.yml:17-19` — **une image modifiée peut ne jamais être publiée.**
   Constat : « modifiée » est calculé sur `github.event.before..github.sha`, et l'empreinte précédente est conservée sinon. Deux cas laissent `env/demo` sur une image périmée, sans aucune alerte :
   - **Fusions rapprochées** : la concurrence du workflow sur `main` (`ci-refs/heads/main`) annule l'exécution en attente quand une troisième arrive (comportement de GitHub). Le commit annulé n'est dans aucune plage publiée.
   - **Publication en échec** : si un Trivy bloquant ou une erreur cosign arrête la publication, le push suivant qui ne touche pas ce service garde l'ancienne empreinte.

   [cicd.md](../../processus/cicd.md) l.118 exige d'ailleurs que `env/demo` porte « la référence du commit source », ce que le fichier ne fait pas.
   Correction : écrire `source: <commit>` dans `deploy/demo/empreintes.yaml` (clé hors de `images`, donc sans effet sur le chart) et calculer `modifiee` sur `source..COMMIT`. Reconstruire tout si la clé est absente ou si le commit est inconnu. Dev : devops.

2. `infra/application/templates/politiques-reseau.yaml:142-168` et `tools/sortie-sans-imds.jq` — **le contrôle après déploiement devient rouge dès que l'application est installée sur AWS.**
   Constat : `sortie-aws` ouvre légitimement `169.254.169.254/32` au pod de la Place de marché (US-006 AC5). Mais le filtre hérité d'US-004 refuse tout `ipBlock` couvrant l'IMDS dans le namespace : `place-de-marche` → `false`. Le contrôle rejoué (`alb.sh ouvrir`, AC10 d'US-004) ne peut donc plus être vert avec l'application déployée, et l'ALB ne s'ouvre plus.
   Correction : modéliser l'exception dans le filtre. Une règle peut viser l'IMDS seulement si sa politique sélectionne `collector/acces-aws: "true"` (`matchLabels`), dans le namespace `place-de-marche`. Garder le refus partout ailleurs. Ajouter le cas positif (rendu AWS du chart) et un cas négatif (même règle sans le sélecteur) au test du filtre. Dev : devops.

3. `tools/up-flux.sh:309` et manifeste Flux (`crd-controller-flux-system`) — **AC9 et AC3 non tenus : Flux garde des droits sur tout le cluster.**
   Constat : supprimer `cluster-reconciler-flux-system` retire `cluster-admin`. Mais la ClusterRoleBinding `crd-controller-flux-system`, installée par `flux install`, reste en place : helm-controller et source-controller gardent `get/list/watch` sur **tous les Secrets du cluster**, ainsi que `create/update/patch/delete` sur **tous les ConfigMaps**, dont `kube-system/coredns`. Les deux contrôleurs tournent aussi en `--watch-all-namespaces=true`. `kubectl auth can-i --list --as=system:serviceaccount:flux-system:helm-controller` (AC9) montrera donc ces droits hors des namespaces applicatifs. AC3 demande des contrôleurs « limités au namespace de l'application ». Le commentaire de `deployeur.yaml:3` (« Secrets lisibles dans le seul namespace collector ») est vrai pour le déployeur, pas pour Flux.
   Correction, la plus simple : `flux install --namespace=collector --watch-all-namespaces=false`. Dans le manifeste exporté, remplacer `crd-controller-flux-system` par une `Role` dans `collector` : groupes `*.toolkit.fluxcd.io`, `secrets`, `configmaps`, `events`, `leases`. Ne garder au niveau du cluster que la lecture des `namespaces` si elle est nécessaire. Rejouer AC9 et archiver la sortie. Dev : devops.

4. `infra/flux/sources.yaml:5-16`, `tools/up-flux.sh:71-109` — **la révision vérifiée n'est pas celle que Flux déploie (AC4).**
   Constat : `up-flux.sh` vérifie les signatures des empreintes lues à l'instant T. Flux relit ensuite `env/demo` toutes les minutes par sa propre source et applique toute révision ultérieure sans vérification. C'est vrai dès la première réconciliation si un commit arrive entre-temps, et pendant toute la session ensuite. De plus, `--empreintes FICHIER` sans `--sans-flux` vérifie un fichier local que Flux n'utilise pas. Enfin, le ruleset d'`env/demo` contourne toutes ses règles pour GitHub Actions (majeure 5) : la garantie d'AC4 repose donc sur la seule vérification de `up-flux.sh`. La revérification continue est hors périmètre (V1), mais déployer **la** révision vérifiée ne l'est pas.
   Correction :
   - épingler `env-demo` sur le commit vérifié (`ref.commit: <sha>` injecté par `up-flux.sh`, comme les valeurs) ;
   - refuser `--empreintes` sans `--sans-flux` ;
   - tracer dans l'US que toute nouvelle publication en session passe par une relance de la vérification (`up-flux.sh`).

   Cela rejoint la consigne de D10 (`flux suspend` pendant la démonstration). Dev : devops.

5. `tools/github/configurer-depot.sh:112-134` — **la protection d'`env/demo` contre la poussée forcée et la suppression ne s'applique pas au seul écrivain.**
   Constat : `bypass_mode: always` pour l'intégration GitHub Actions s'applique à **toutes** les règles de l'ensemble, `non_fast_forward` et `deletion` comprises. Toute exécution d'Actions munie de `contents: write` peut forcer ou supprimer `env/demo`. C'est le cas d'un workflow modifié sur une branche du dépôt : en `pull_request` interne, le jeton accepte les permissions déclarées. AC8 (« seul le job de publication sur `main` ») ne repose donc que sur le contenu de `ci.yml`.
   Correction :
   - scinder la protection en deux ensembles : `deletion` + `non_fast_forward` **sans** contournement, et `creation` + `update` avec contournement d'Actions ;
   - pour réserver réellement l'écriture au job de `main`, préférer une clé de déploiement (ou une App) rangée dans un environnement `publication` limité à `main`, utilisée par le seul job `publication`, plutôt que le contournement d'Actions.

   Dev : devops.

6. `tools/up-flux.sh:165-175`, `:266-289` — **relancer `up-flux.sh` sur un cluster existant casse les connexions et l'OIDC de Transactions.**
   Constat : chaque exécution régénère tous les mots de passe des rôles (`ALTER ROLE … PASSWORD`), le secret du client `transactions` et les Secrets. Les pods déjà démarrés gardent les anciennes valeurs, et la `HelmRelease` inchangée ne relance rien : toute nouvelle connexion à la base échoue. Le cas est probable, puisque le script s'arrête en `echec` sur un délai d'attente de `HelmRelease` et invite à corriger puis relancer. Keycloak démarre en `--import-realm`, qui ignore un realm existant : son secret de client reste l'ancien, alors que `oidc-client` porte le nouveau, et Transactions échoue à l'échange client.
   Correction : réutiliser les valeurs des Secrets existants s'ils sont présents (lecture par `kc get secret`). Ne régénérer qu'à la première installation, ou sur demande explicite (`--rotation`), suivie de `kubectl rollout restart` des Deployments concernés. Dev : devops.

### Mineures

1. `tools/up-flux.sh:338` — **`chemin_natif` redéfinie à l'identique** de `tools/lib.sh:68`, déjà sourcée. Correction : supprimer la redéfinition. Dev : devops.

2. `tools/up-flux.sh:354` — **`curl -o /dev/null` sous Git Bash** : même défaut que la mineure 4 d'US-004 (sortie 23, sauvée par `|| true`), alors que `$NUL` existe dans `lib.sh:70`. Correction : `-o "$NUL"`. Remplacer aussi `sleep 3` par une attente bornée de l'ouverture du port. Dev : devops.

3. `tools/up-flux.sh:254-259` — **Secret maître laissé dans `collector` si l'amorçage échoue.** `echec` sort avant la suppression de `secret/amorcage-base`, qui contient le mot de passe maître de RDS. Il reste alors lisible par le compte `deployeur` (`versions-helm`) et par Flux (majeure 3), contrairement à MD2. Correction : `trap` local qui supprime le Secret, le ConfigMap et la politique `amorcage-base` sur toute sortie. Dev : devops.

4. `tools/down.sh:34` contre `tools/up-flux.sh:293-296` — **Flux non retiré si la CLI `flux` est absente.** `up-flux.sh` sait installer Flux par l'image `flux-cli` ; `down.sh` ne le retire qu'avec la CLI locale. Flux peut alors recréer les releases pendant `down` : c'est le risque que le commentaire de `down.sh:35` veut éviter. Correction : même repli Docker dans `down.sh`, ou `kubectl delete -f` du manifeste de session `flux-install.yaml`. Dev : devops.

5. `tools/up-flux.sh:308-313` — **fenêtre `cluster-admin` à chaque relance.** `apply --server-side --force-conflicts` rétablit la ClusterRoleBinding `cluster-admin` et réécrit `args` (liste atomique). helm-controller redémarre alors sans `--default-service-account`, avec `cluster-admin`, alors que les `HelmRelease` existent déjà, jusqu'au `delete` et au `patch` suivants. Correction : retirer `cluster-reconciler-flux-system` du manifeste et y injecter les deux arguments par `sed`, comme les images, avant l'unique `apply`. Dev : devops.

6. `infra/application/templates/front.yaml:40` — **l'annotation de redémarrage ne couvre pas toute la configuration.** `collector/empreinte-config` ne hache que l'émetteur, alors que `config.json` est monté en `subPath`, donc jamais rafraîchi. Correction : hacher le `dict` complet, par exemple `include` d'un gabarit `front.config` puis `sha256sum`. Dev : devops.

7. **Nombre d'images codé en dur à deux endroits** : `up-flux.sh:81` (`-eq 6`) et la liste `IMAGES` de `publier.sh:31-38`. Ajouter une image oblige à modifier les deux. Correction : comparer au nombre de clés `images` attendues par le chart, ou partager la liste, par exemple un fichier `tools/ci/images.txt` lu par les deux scripts. Dev : devops.

8. **Commentaires de justification** (convention zéro commentaire, précédent US-004 mineure 11) : `infra/application/values.yaml:1-2, 8, 13, 20, 22, 54, 56, 99`, `valeurs-aws.yaml:1-2`, `valeurs-k3d.yaml:1-2`, `templates/politiques-reseau.yaml:1-4`, `flux/deployeur.yaml:1-3, 159`, `flux/releases.yaml:1-3`, `flux/sources.yaml:1-2`, `k8s/application.yaml:1-4, 204-205`, `k8s/base-locale.yaml:1-2`. Les raisons ont leur place dans `infra/application/README.md` et `infra/README.md`, qui en portent déjà l'essentiel. Restent admis : l'en-tête d'usage de `publier.sh` et `up-flux.sh`, le format de `.trivyignore`, les commentaires de permissions de `ci.yml` (style existant). Dev : devops.

### Suggestions

1. **Images du front et de Keycloak jamais construites en PR.** Seules les images Go le sont (job `go`). Un `Dockerfile` cassé ou une vulnérabilité élevée de l'image Keycloak, nouvelle dans cette US, n'apparaît qu'après fusion. Elle arrête alors **toute** la publication, puisque `publier.sh` sort à la première erreur. Ajouter une construction sans poussée, avec Trivy, quand `web/` ou `infra/identite/` change (MA14 de D10). Dev : devops.

2. `infra/canal/outils/kubernetes.sh:50` — la clé `AMQP_URL`, qui contient le mot de passe, n'est plus lue par aucun service (seul `sandbox/XP02` l'utilise, avec ses propres scripts). La retirer réduit la copie du secret, et le README du canal s'en trouve simplifié. Dev : devops.

3. **Chart lu sur `main`, images lues sur `env/demo`.** Une fusion qui change à la fois le chart et une image déploie le nouveau chart en une minute avec les anciennes images, jusqu'à la fin de la publication (20 min visées). La majeure 1 permet d'épingler aussi le chart : `ref.commit` du `GitRepository` `collector` sur le commit `source` d'`env/demo`. Dev : devops.

4. `tools/up-flux.sh:94-107` — vérifier aussi que chaque clé pointe vers son propre dépôt (`images.front` → `…/front@sha256:`). Sans ce contrôle, une signature valide d'une autre image du même workflow passe. Dev : devops.

5. `infra/application/templates/identite.yaml:32` — `KC_PROXY_HEADERS=xforwarded` sans `KC_PROXY_TRUSTED_ADDRESSES`. Le risque est faible, l'entrée étant limitée aux sous-réseaux de l'ALB et aux trois API, mais ajouter les CIDR de `exposition.sources` ferme le point. Dev : devops.

6. **À vérifier en test (hypothèse, non constatée)** : la `HelmRelease` se réconcilie-t-elle quand **seul** `env/demo` change ? La révision du `GitRepository` `collector` reste celle de `main`, et seul le contenu inclus change. Le test d'AC3 doit inclure une publication sans commit sur `infra/application`, suivie de `flux get helmreleases`. Fonction : Test.

## Verdict

| Sévérité | Nombre |
|---|---|
| Bloquante | 3 |
| Majeure | 6 |
| Mineure | 8 |
| Suggestion | 6 |

**Non fusionnable en l'état.** La partie CI est solide. Le job de publication est confiné et épinglé, sans AWS. La signature et la vérification se font par identité de workflow. `env/demo` est écrite sans forçage. Côté chart, PSS `restricted`, empreintes exigées au rendu, refus par défaut, 8081 privé et IMDS réservé par étiquette sont en place. Les outils hors ligne sont tous verts (`helm lint`/`template`, `fmt`, `validate`, shellcheck, actionlint, zizmor, Trivy `config`, exceptions).

En revanche, la chaîne de déploiement ne fonctionne pas :
- `up-flux.sh` s'arrête à sa première ligne (1) ;
- sur AWS, rien ne relie `up.sh`, la release `front` existante et les règles de l'ALB à l'application (2) ;
- l'installation de Flux ouvre l'IMDS à ses contrôleurs (3).

Les majeures portent sur :
- la fidélité de la publication (1) ;
- le contrôle IMDS hérité d'US-004 (2) ;
- les droits réels de Flux face à AC9 (3) ;
- l'écart entre la révision vérifiée et la révision déployée (4) ;
- la protection d'`env/demo` (5) ;
- la relance de `up-flux.sh` (6).

Répartition : devops pour les 3 bloquantes, les 6 majeures, les 8 mineures et les suggestions 1 à 5 ; testeur pour la suggestion 6, et pour rejouer AC3 à AC6 et AC9 sur k3d une fois les bloquantes levées. Aucune correction back ni front.

## Retours traités

2026-11-25, branche posée sur US-005, sans appel AWS, sans poussée ni commit.

| Point | Traitement |
|---|---|
| B1 | `contexte_autorise` ajoutée à `tools/lib.sh` (motifs `k3d-*`, `*collector*`), appelée par `up-flux.sh` ; contexte hors liste refusé avant tout appel (constaté avec `docker-desktop`) |
| B2 | `up.sh` appelle `up-flux.sh --contexte collector --valeurs aws`, et relaie `--sans-flux` ; `--chart` et `--valeurs` retirés. `infra/front` supprimé : le chart applicatif reprend image par empreinte, NodePort 30080, HPA 2 à 6 et descente en 60 s. `application_active` vaut `true` par défaut. `nodeports_application` et `nodeport_front` sont lus par `up-flux.sh` (`--set` en filet P1, `spec.values` de la `HelmRelease` sous Flux) et par le contrôle (NodePorts admis, ports balayés). `down.sh` retire Flux même sans CLI (`HelmRelease` désinstallées, puis manifeste de session ou regénéré par l'image `flux-cli`) et efface `infra/canal/.genere/`. Documentation : `tools/README.md`, `infra/README.md`, D10 (« Mise en œuvre »), `cicd.md`, US-007 |
| B3 | `manifestes_flux` (`tools/lib.sh`) écarte toutes les `NetworkPolicy` de `flux install`, dont `allow-egress`, avant l'unique `apply` ; `infra/flux/controleurs.yaml` ouvre DNS, 443 et 6443 sans IMDS, plus 9090 entre helm-controller et source-controller. Cas « politique allow-egress de flux install » ajouté au test du filtre (refusé) |
| M1 | `env/demo` porte `source: <commit>` ; `publier.sh` compare depuis ce commit (reconstruction complète s'il est absent, inconnu ou non ancêtre) |
| M2 | Filtre : une règle vers `169.254.169.254/32` seul n'est admise que dans une politique du namespace `place-de-marche` qui sélectionne `collector/acces-aws: "true"`. Cinq cas ajoutés : un positif (chart AWS), quatre négatifs (sans étiquette, autre namespace, exception qui ouvre tout, exception vers 169.254.0.0/16) |
| M3 | Flux installé dans `collector`, `--watch-all-namespaces=false`. `cluster-reconciler` et `crd-controller` (liaisons et rôle) retirés du manifeste ; `Role` `flux-controleurs` limitée à `collector`, ressources nommées une à une (KSV-0112) |
| M4 | `GitRepository` `env-demo` et `collector` épinglées par `ref.commit` sur le commit vérifié d'`env/demo` et sur son commit `source` (suggestion 3 incluse) ; `--empreintes` refusé sans `--sans-flux` ; règle de session tracée dans l'US, D10 et `cicd.md` |
| M5 | Deux ensembles de règles : `protection-env-demo` (suppression, poussée forcée) sans aucun contournement ; `ecriture-env-demo` (création, mise à jour) contourné par les seules clés de déploiement. Écriture par une clé rangée dans l'environnement `publication`, limité à `main` ; le job passe à `contents: read`. Mode `configurer-depot.sh --cle-env-demo`. Limite documentée dans D10 : GitHub ne sait pas nommer une clé précise ; mesure compensatoire : une seule clé en écriture, et `cosign verify` sur l'identité du workflow de `main` avant tout déploiement |
| M6 | Mots de passe et secret du client `transactions` repris des Secrets existants ; `--rotation` régénère les rôles de base puis redémarre les services |
| m1 à m8 | `chemin_natif` local retiré ; `-o "$NUL"` et attente bornée du port ; nettoyage de l'amorçage sur toute sortie (`trap EXIT`) ; Flux retiré sans CLI ; plus de fenêtre `cluster-admin` (arguments injectés et liaisons retirées avant l'`apply`) ; annotation du front sur toute la configuration ; liste partagée `tools/ci/images.txt` ; commentaires de justification retirés, raisons reportées dans `infra/application/README.md` et `infra/README.md` |
| Suggestions | 1 : image du front construite et passée à Trivy en PR (job « Front ») ; Keycloak l'est par le job `go` d'`infra/identite` (Dockerfile au contexte racine). 2 : clé `AMQP_URL` retirée du Secret `canal-amqp`. 3 : voir M4. 4 : chaque clé doit pointer vers `…/<nom>@sha256:`. 5 : `KC_PROXY_TRUSTED_ADDRESSES` = `exposition.sources`. 6 : sans objet, le chart et les images étant épinglés ensemble (M4) |
| Constats repris | `sortie-sans-imds` du namespace `front` retirée de `socle.yaml` (namespace `front` déplacé dans `application.yaml`) ; `CHEMINS_INTERDITS` complétés (`/interne/…`, traversées vers `/interne`, `/sante`, `/disponibilite`), rejoués aussi sur `auth.` |

### Preuves rejouées

| Commande | Résultat |
|---|---|
| `helm lint` et `helm template` (`valeurs-aws.yaml`, `valeurs-k3d.yaml`) ; `helm lint infra/canal` | 0 échec |
| kubeconform `-strict` (rendus, `infra/k8s`, `infra/flux`, manifeste Flux filtré) | 134 valides, 0 invalide ; 7 CRD Flux sans schéma dans le catalogue |
| `terraform fmt -check -recursive infra`, `validate` (éphémère) | Conformes, valide |
| shellcheck (CI, `--severity=warning`), actionlint, zizmor `--offline` | 0 constat |
| `bash tools/tests/*.sh` | Tous les cas passent (dont les 6 nouveaux) |
| `trivy.sh config infra infra` et `config . config` ; `verifier-exceptions.sh` | 0 bloquant ; 3 exceptions valides |
| Filtre IMDS hors ligne : rendu AWS + `application.yaml` + `socle.yaml` + manifeste Flux + `controleurs.yaml` | `true` dans les 8 namespaces, `place-de-marche` (exception) et `collector` (Flux) compris |
| k3d neuf : `up-flux.sh --sans-flux --images-locales` | Amorçage, migrations, 6 Deployments, catégories 200, en 4 min ; Secret d'amorçage absent ensuite ; filtre IMDS `true` sur les 10 namespaces du cluster ; front vers Internet refusé |
| Relance identique (M6) | Empreinte des Secrets inchangée, aucun redémarrage ; jeton `client_credentials` de Transactions obtenu avec `oidc-client` (200) |
| Flux restreint (manifeste filtré, `controleurs.yaml`, `deployeur.yaml` ; chart en OCI local et politique d'essai vers le registre local, hors dépôt) | `canal` et `application` `Ready`, sans aucune `ClusterRoleBinding` Flux |
| AC9 (`kubectl auth can-i`) | helm-controller et source-controller : aucun Secret ni ConfigMap hors `collector`, aucun Deployment nulle part ; usurpation de `deployeur` seule. `deployeur` : Secrets de `collector` seuls, aucun objet de cluster ni `RoleBinding` |
| AC5 | Empreinte invalide du front : mise à jour en échec, `RollbackSucceeded`, v3 déployée, image d'origine restaurée |
| AC4 | Empreintes GHCR non signées : les six refusées, allumage arrêté ; empreinte d'un autre dépôt et `--empreintes` sans `--sans-flux` refusés avant tout appel au cluster |
| `down.sh` (fonction `retirer_flux`, sans CLI `flux`) | CRD, contrôleurs et releases retirés |

Non joué hors AWS : `publier.sh` sur `main` (GHCR, clé de déploiement, `env/demo`), règles et environnement GitHub, chemin AWS d'`up-flux.sh` (RDS, TLS, Secrets Manager, NodePorts Terraform), `up.sh` et `down.sh` de bout en bout.
