# Revue de code — US-011 Canal asynchrone avec une identité par service

| Champ | Valeur |
|---|---|
| US | [US-011](../../../backlog/US-011-canal-rabbitmq-identites.md) |
| Branche | `feat/US-011-canal-rabbitmq-identites` |
| Périmètre | `git diff origin/main...HEAD` (6 commits, 21 fichiers sous `infra/canal/`, +1255) ; `.gitignore` inchangé (`.genere/` déjà ignoré sur `main`) |
| Date | 2026-10-26 |
| Référentiel | AC de l'US, [D05](../../decisions/D05-communication-interservice.md) (MC1 à MC10), [D10](../../decisions/D10-forge-ci-cd.md), [D11](../../decisions/D11-iac-allumage-cout.md), [infra/canal/README.md](../../../infra/canal/README.md) |
| Verdict | **À corriger** : aucune issue bloquante, 2 majeures, 5 mineures, 3 suggestions |

## 1. Vérifications exécutées

| Commande | Résultat |
|---|---|
| `helm lint infra/canal` (Helm 4.1.3) | 0 échec (info : `icon` recommandé) |
| `helm template canal infra/canal -n canal` puis recherche de `password`, `MDP`, valeur de cookie | Aucun secret rendu ; seules les références au Secret `rabbitmq-interne` |
| shellcheck 0.11.0 (Docker, `-x`, sévérité par défaut) sur `outils/*.sh` | 0 constat. Avec `-o all` : uniquement des styles optionnels (SC2250, SC2310, SC2312, SC2154 sur variables sourcées) |
| Empreintes d'image (`docker buildx imagetools inspect`) | `rabbitmq:4.3.7-management@sha256:79fb281e…` et `python:3.13.17-slim@sha256:d98cc29d…` conformes au registre |
| Comportement de `set -e` dans une substitution de processus `< <(…)` (bash) | Constaté : le sous-shell s'arrête à la première commande en échec, le shell principal continue (voir majeure 1) |
| Mention d'outil d'IA, d'assistant, d'agent ou de flotte (diff et messages de commit) | Aucune |
| Fichiers de `.genere/` suivis par git | Aucun |

Image shellcheck supprimée après usage. Aucun déploiement k3d rejoué : la revue s'appuie sur le code et les constats ci-dessus.

## 2. Points forts

- `broker/definitions/10-topologie.json:5-14` : `configure` vide pour toutes les identités de service, `write` sur le seul échange du producteur, `read` sur la seule file du consommateur, permission de sujet `publication.decidee.v1` seule sur `pdm.evenements` (MC1). Vérifié : aucune identité de service n'a à la fois `read` sur un échange et `write` sur une file, donc aucune liaison possible (AC5).
- Absence de `guest` : jamais créé (définitions importées), `loopback_users.guest` en défense (`10-collector.conf:10`), le `10-defaults.conf` de l'image écarté par le montage du `conf.d`, contrôle actif dans `kubernetes.sh:75` et `local.sh:15`.
- Secrets : mots de passe générés par `openssl rand`, empreintes `rabbit_password_hashing_sha256` correctement formées (`identifiants.sh:18-26`), manifestes de Secret envoyés sur l'entrée standard de `kubectl` (`kubernetes.sh:32-57`), mot de passe de `curl` par `-K -` (`verifier.sh:114`), sonde locale qui ne reçoit que des noms de variables (`verifier.sh:52`). Rien dans `values.yaml`, la ConfigMap ni le manifeste Helm.
- Cookie Erlang monté dans le seul conteneur d'initialisation (`statefulset.yaml:32-46`), absent du conteneur du broker.
- StatefulSet durci : non root (999), `readOnlyRootFilesystem`, `capabilities: drop ALL`, `allowPrivilegeEscalation: false`, seccomp `RuntimeDefault`, jeton de compte de service non monté, requêtes et limites mémoire, trois sondes, `emptyDir` bornés ; empreinte de la ConfigMap en annotation pour redémarrer sur changement de configuration.
- Politiques réseau (`politiques-reseau.yaml`) : refus par défaut entrée et sortie, 5672 aux trois namespaces clients seulement (Transactions n'utilise pas le canal, D05 l. 29), 15672 au seul pod `keda-operator`, 15692 fermé tant que D09 n'est pas là, 4369 et 25672 fermés, sortie DNS seule. Services `ClusterIP` uniquement.
- Écarts déclarés cohérents avec D05 : `delivery-limit` 4 = 5 livraisons (prouvé par une égalité stricte dans `sonde.py:264`), déclaration passive permise, purge de sa propre file seulement (cas négatif sur la file d'un autre, `sonde.py:218`), `emptyDir` pour AC3. Mise à l'écart `at-least-once` avec `reject-publish` (seule combinaison acceptée par RabbitMQ).
- README clair : contrat pour les services (Secret `canal-amqp`, clé `AMQP_URL`, `user-id`, `basic.consume`), conforme à ce qu'utilisera US-013 (échanges `controle.demandes` et `controle.resultats`, files `controle.demandes` et `pdm.resultats-controle`).
- Chart conforme à D10 (MP4) : ni namespace, ni CRD, ni RBAC ; `.helmignore` exclut `.genere/`, `outils/`, `verification/`.

## 3. Issues

### Bloquantes

Aucune.

### Majeures

1. **`infra/canal/outils/verifier.sh:143-149`, `:163-176`, `:180-181` et `infra/canal/verification/sonde.py:100-118`** — La vérification peut conclure « conforme » alors que des cas n'ont pas été exécutés.
   Constat : le verdict final ne compare `ok` qu'à `total`, le nombre de lignes reçues. Aucun nombre attendu n'est fixé et le code de sortie de la sonde est ignoré (`|| true` l. 55 ; phase du pod jamais lue, l. 88-93). Deux chemins donnent un faux vert :
   - la sonde s'interrompt sur une exception non interceptée par `tenter`. Exemple : `basic_publish(..., mandatory=True)` sous `confirm_delivery` lève `UnroutableError` (ou `NackError`), qui n'hérite pas de `AMQPConnectionError`. Si une liaison manque (régression d'AC2), la sonde meurt avant d'écrire le cas AC2, les scénarios suivants passent quand même, et le résultat est « N / N conformes » avec un code de sortie 0 ;
   - en mode `kubernetes`, sous `set -euo pipefail` (hérité de `lib.sh:5`), toute commande en échec dans le bloc `< <(…)` arrête le sous-shell en silence. Constaté en bash : les cas restants (gestion, `ClusterIP`, les quatre contrôles AC6) sont sautés, et le script se termine en succès. Cas réaliste : un `configmap` créé dans `canal-intrus` encore en suppression lors d'une relance rapprochée (namespace effacé avec `--wait=false`, l. 177).
   Pourquoi : le script est la preuve après déploiement d'AC1, AC6 et AC7 (« Tests attendus » de l'US). Un faux vert vide la preuve de sa valeur.
   Correction : faire de chaque exécution de la sonde un cas comptabilisé (code de sortie du conteneur, phase `Succeeded` du pod). Ajouter dans `tenter` une branche finale `except Exception as e` qui renvoie `"erreur %s" % type(e).__name__`. Enfin, fixer le nombre de cas attendu par mode (62 et 75 d'après le README) et le comparer à `total` avant de conclure.
   → DevOps

2. **`infra/canal/outils/kubernetes.sh:65-74`** — `up` n'est pas idempotent : relancé pendant une session, il redémarre le broker et vide les files.
   Constat : quand le StatefulSet existe déjà (`deja=1`), `rollout restart` est exécuté sans condition, même si `.genere/` existait et que les Secrets réappliqués sont identiques. Le pod est recréé, l'`emptyDir` aussi : les messages en file sont perdus (README, § Écarts : le stockage ne survit pas à la suppression du pod). Si la ConfigMap a changé, l'annotation `collector/empreinte-config` provoque déjà un redémarrage, et `up` en ajoute un second. Le commentaire l. 71 (« Nouveaux utilisateurs ») décrit une condition que le code ne teste pas.
   Pourquoi : une relance de `up` pour appliquer un changement du chart détruit les demandes de contrôle en cours (ENF-12). Le filet de la Place de marché compense, mais la perte n'est ni voulue ni signalée.
   Correction : ne redémarrer que si le contenu de `rabbitmq-interne` a changé. Par exemple, comparer l'empreinte de `.data` (`kc get secret rabbitmq-interne -o jsonpath='{.data}' | sha256sum`) avant et après `secrets`, ou ne redémarrer que si `identifiants.sh` vient d'être exécuté. Supprimer ou corriger le commentaire.
   → DevOps

### Mineures

3. **`infra/canal/outils/verifier.sh:98-108`** — Le Secret temporaire `canal-sonde` contient tous les mots de passe de la session, dont celui d'`admin-canal`, dans le namespace `place-de-marche`. Aucun `trap` ne garantit sa suppression et il n'a pas l'étiquette `app.kubernetes.io/part-of=collector-canal`. Après une interruption (Ctrl+C, échec de `pod_sonde` sous `set -e`), il reste en place, et `kubernetes.sh down` (suppression par étiquette, l. 90) ne l'efface pas. MC4 veut l'identité d'administration hors de tout Secret de namespace de service.
   Correction : poser l'étiquette du canal sur `canal-sonde`, et ajouter un `trap` de suppression (Secret, pod, ConfigMap de la sonde, namespace intrus) dès sa création.
   → DevOps

4. **`infra/canal/outils/verifier.sh:125-130`** — `fuite` écrit tous les mots de passe et le cookie dans `.genere/motifs-fuite` avec l'umask par défaut. Sous Linux, le fichier est en 644. Il n'est pas supprimé si la commande est interrompue. Par ailleurs, `identifiants.sh:36-37` crée `.genere/` avant `umask 077`, donc le répertoire est en 755.
   Correction : `umask 077` en tête de `fuite` (ou passer les motifs par `grep -f <(…)`), et `umask 077` avant le `mkdir -p` dans `identifiants.sh`.
   → DevOps

5. **`infra/canal/README.md` § Contrat pour les services** — Le Secret `canal-supervision` (clé `host`, URL de l'API de gestion avec le mot de passe de `keda`) posé dans le namespace `controle` (`kubernetes.sh:56-57`) n'est pas documenté. US-026 (KEDA) en dépendra, et sa présence dans un namespace de service est un choix de sécurité à rendre visible (lecture possible de l'inventaire des files par le tag `monitoring`, mais 15672 fermé à ce namespace).
   Correction : ajouter une ligne au contrat (nom, clé, namespace, consommateur prévu).
   → DevOps

6. **`infra/canal/outils/verifier.sh:128-129` et `:135-136`** — Retours à la ligne littéraux à l'intérieur de chaînes entre apostrophes (`tr '⏎' ' '`, format de `printf`). Le code fonctionne mais il est fragile (conversion de fins de ligne, éditeurs). `.gitattributes` gère peut-être les fins de ligne, mais le code ne devrait pas en dépendre.
   Correction : `tr '\n' ' '` et `'…|%s\n'`.
   → DevOps

7. **Commentaires qui paraphrasent le code** (convention : aucun commentaire superflu) : `outils/lib.sh:30`, `:35`, `:41` ; `outils/identifiants.sh:19` ; `outils/kubernetes.sh:33-34`, `:71` (trompeur, voir majeure 2) ; `outils/verifier.sh:29`, `:59`, `:99`, `:123-124` ; `verification/sonde.py:1-8`, `:59-60`, `:70-71`, `:232-234` ; `verification/Dockerfile:1` ; `.helmignore:1-2` ; `compose.yml:10-11`. Les en-têtes d'usage des scripts et les références de mesure (MC3, AC7) dans les manifestes restent dans l'usage des fichiers d'infrastructure de `main` (`.github/workflows/ci.yml`).
   Correction : supprimer ces commentaires, ou les remplacer par des noms de fonction explicites ; les raisons de fond figurent déjà dans le README.
   → DevOps

### Suggestions

8. **`broker/definitions/10-topologie.json:6-7`** — Les expressions de droits portent sur des noms communs aux échanges et aux files. `controle.demandes` est à la fois un échange et une file : `marche` a donc `write` sur la file `controle.demandes`, et `controle` a `read` sur l'échange `controle.demandes`. Aucune liaison n'est possible aujourd'hui, faute du droit complémentaire. Mais une future file `controle.resultats` ou un futur échange `pdm.resultats-controle` ouvrirait une liaison sans que le diff des droits le montre. Suggestion : un cas de sonde qui tente `marche` : lier la file `controle.demandes` à un échange qu'elle peut lire, ou une règle de nommage qui distingue échanges et files.
9. **`values.yaml:5`, `verification/Dockerfile:2`** — Images tirées de Docker Hub, alors que D10 (§ registres, l. 188) préfère quay.io ou `public.ecr.aws` derrière la NAT (limite de débit sans authentification) [?]. Le miroir `public.ecr.aws/docker/library/rabbitmq` porte la même empreinte. À trancher avant le premier `up` sur AWS.
10. **`outils/local.sh:23` et `outils/kubernetes.sh:92`** — `local` et `kubernetes` partagent `.genere/` : un `local.sh down` efface les identifiants d'une session Kubernetes en cours. Le prochain `kubernetes.sh up` les régénère et redémarre le broker. Suggestion : un sous-répertoire par cible, ou un avertissement dans le README.

## 4. Conformité aux points demandés

| Point | État |
|---|---|
| Moindre privilège (configure, write, read, droits de sujet) | Conforme (suggestion 8) |
| Absence de `guest` | Conforme |
| Aucun secret versionné ni rendu (manifestes, valeurs Helm) | Conforme |
| Génération des secrets | Conforme (mineure 4 sur les droits de fichiers) |
| Politiques réseau (5672, 15672, 15692, 4369, 25672, sortie) | Conforme |
| StatefulSet durci, sondes, ressources | Conforme |
| Image épinglée par empreinte | Conforme, empreintes vérifiées |
| Scripts : `set -euo pipefail`, pas de fuite dans les journaux | Conforme |
| Scripts : idempotence, fiabilité de la vérification | Majeures 1 et 2, mineure 3 |
| README | Clair ; mineure 5 |
| Cohérence avec D05 et écarts déclarés | Conforme |
| Commentaires superflus | Mineure 7 |
| Mention d'outil d'IA, d'assistant, d'agent, de flotte | Aucune |

## 5. Corrections à adresser

DevOps, 7 corrections : majeures 1 et 2, mineures 3 à 7. Les suggestions 8 à 10 sont laissées à l'appréciation de Martin.
