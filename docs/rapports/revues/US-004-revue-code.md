# US-004 — Revue de code

| Champ | Valeur |
|---|---|
| US | [US-004 — Allumer, exposer en HTTPS et éteindre l'environnement AWS depuis zéro](../../../backlog/US-004-environnement-aws-https.md) |
| Branche | `feat/US-004-environnement-aws-https` (HEAD `32cb326`, 9 commits), revue par `git diff origin/main...HEAD` (54 fichiers, +3392/-5) |
| Date | 2026-11-20 |
| Périmètre | `infra/persistant` (zone, CAA, certificat, budget, bucket des photos, journaux, clé KMS, imports) ; `infra/ephemere` (réseau, ALB, DNS, EKS, RDS, retrait d'ECS) ; chart `infra/front` ; `infra/k8s` ; `tools/{lib, bootstrap, up, down, status, inventaire, alb, controle-apres-deploiement}.sh` ; sonde Lambda `tools/sonde-hors-plage/sonde.py` ; `.trivyignore`, `.gitignore` ; README racine, `infra/README.md`, `tools/README.md`, D08, backlog, mesures archivées |
| Références | [D03](../../decisions/D03-hebergement-orchestration.md) (EKS confirmée), [D08](../../decisions/D08-exposition-tls-domaine.md), [D11](../../decisions/D11-iac-allumage-cout.md) (G1 à G5, MI1 à MI10), [D14](../../decisions/D14-experimentations.md) (X17, X20 bloquants) ; décisions de Martin du 2026-11-20 : exceptions Trivy AWS-0053 et AWS-0040, clé KMS pour AWS-0039, 400 de l'ALB accepté pour XE4, AC9 rejoué avec l'image de la CI (US-007) |
| Hors revue (connu) | AC9 rouge tant que l'image nginx provisoire est déployée (US-007) ; seconde mesure d'AC2 et d'AC4 ; ME4 et ME8 avant la démonstration publique ; validation Sécurité des deux exceptions Trivy (en cours) |
| Verdict | **Fusion après corrections** (0 bloquante, 4 majeures, 12 mineures, 5 suggestions) |

## Vérifications exécutées

Aucun appel à l'API AWS.

| Commande | Résultat |
|---|---|
| `shellcheck -x -P SCRIPTDIR --severity=style tools/*.sh` (image `koalaman/shellcheck:v0.11.0` épinglée dans `tools/ci/outils.yml`) | 0 erreur, 0 avertissement, 0 style ; 8 `info` : SC2329 sur des fonctions appelées indirectement (`trap nettoyer`, `chercher <source> <fonction>`) et SC2016 voulu (`port_ouvert`) : faux positifs. Le seuil de la CI (`--severity=warning`) est vert |
| `terraform fmt -check` sur les `.tf` suivis par git | Conformes. `fmt -check -recursive infra` sort en 3 sur le seul `infra/persistant/persistant.auto.tfvars` local, ignoré par git, hors diff |
| `terraform validate` (`infra/ephemere`, `infra/persistant`) | `Success! The configuration is valid.` (les deux) |
| `helm lint infra/front` et `-f valeurs-aws.yaml` | 0 échec. Sans valeurs, le gabarit refuse l'image non épinglée (`fail`), comme prévu |
| `bash tools/ci/verifier-exceptions.sh` | 2 exceptions actives, aucune invalide ni échue (effet de bord : `rapports/i6-exceptions.json` réécrit, dossier ignoré par git) |
| Fins de ligne (`git ls-files --eol tools/*.sh`, recherche de `\r`) | `eol=lf` imposé par `.gitattributes`, aucun CRLF dans les scripts, les `.tf`, le chart ni les manifestes |
| `MSYS_NO_PATHCONV=1 curl -s -o /dev/null -w '%{http_code}' https://example.com` (Git Bash) | Affiche `200` mais sort en 23 : `curl` natif n'ouvre pas `/dev/null` (voir mineure 4) |
| Filtre jq de `controler_sortie_imds` sur deux politiques permissives | Les deux sont jugées conformes (voir majeure 3) |
| Rapports archivés `docs/mesures/US-004/*.txt`, `sessions.csv`, `couts.csv` | Aucun numéro de compte, aucune adresse IP, aucune valeur de secret |
| Restes d'ECS (`ecs`, `orchestrateur`, `port_front`, `port_cible`, `local.eks`) | Seul `q_ecs` reste dans `inventaire.sh`, volontairement (orphelins, documenté dans le backlog). Aucune autre référence |
| Usages des variables et fonctions de `lib.sh` | Toutes utilisées. `contexte_k8s` est toujours vraie (mineure 5). Les exports `TF_VAR_alb_ouvert` et `TF_VAR_preuve_controle` sont toujours écrasés (mineure 6) |
| Commentaires ajoutés | En-têtes d'usage des scripts (lus par `usage()`) et en-têtes de `.example`/`secrets-autorises.txt` : conformes à l'usage de `main`. Commentaires « pourquoi » dans les `.tf`, `lib.sh` et `valeurs-aws.yaml` : mineure 11 |

## Points forts

- **AC10 tenu par trois barrières indépendantes** : `alb.sh` n'ouvre que sur un verdict vert de la session **et** de la série d'état courante (`alb.sh:321-326`) ; une précondition Terraform refuse `alb_ouvert` sans `preuve_controle == session` (`securite.tf:19-24`) ; tout `apply`/`destroy` des scripts passe `-var=alb_ouvert=false` (`lib.sh:146`), au-dessus des `*.auto.tfvars`. Une fois l'ALB ouvert, le contrôle est rejoué et l'ALB est refermé s'il est rouge (`alb.sh:332-336`).
- **Contrôle fermé par défaut** : `nv` (non vérifié) compte comme rouge au même titre que `ko` (`controle-apres-deploiement.sh:544`). Une sonde absente, Docker absent ou `gh` non authentifié ne peuvent pas produire un vert. Dans l'inventaire, une lecture impossible compte comme une ressource trouvée (`inventaire.sh:32-34`).
- **X20 sans exposition** : les valeurs restent en mémoire. La recherche passe par `grep -F -f <(…)` (substitution de processus, aucun fichier), seuls des compteurs sont écrits (`:387-418`), puis `VALEURS` et `brut` sont vidés. Le mot de passe RDS est géré par AWS (`manage_master_user_password`), donc absent de l'état, et la recherche de clés sensibles dans `tf show -json` (MI4) le prouve. Les rapports archivés sont masqués (`masquer_ip`), ce qui est vérifié sur les fichiers versionnés.
- **Couche persistante prudente** : le plan est refusé dès qu'il contient une action `delete`, donc aussi un remplacement (`bootstrap.sh:219-224`) ; `prevent_destroy` protège la zone, le certificat et l'audit. Le nom du bucket est calculé à l'identique en Terraform et en Bash (empreinte du compte, aucun numéro dans le dépôt), et la cohérence est vérifiée (`bootstrap.sh:272-273`). Le bucket refuse le trafic hors TLS et impose son propriétaire. La clé KMS a la rotation activée, et sa politique se limite à la racine du compte (administration) et à `LabRole` (chiffrement, `CreateGrant` réservé aux ressources AWS).
- **G1 conforme à MI5** : groupe de planifications dédié, purgé puis recréé ; cibles désignées par identifiant exact ; `ActionAfterCompletion=DELETE` ; armé même si l'`apply` échoue (`up.sh:110-167`, `:243-244`) ; supprimé par `down.sh`.
- **Inventaire indépendant de l'état** et fidèle à la liste de D11 G2/MI8 (`inventaire.sh:137-157`), y compris instantanés RDS, interfaces disponibles, groupes ouverts à Internet, planifications et groupes de journaux de la sonde.
- **Chart du front durci** : empreinte d'image exigée par le gabarit (`front.yaml:1-3`), `runAsNonRoot`, `seccomp` `RuntimeDefault`, `drop: [ALL]`, aucun jeton de compte de service, namespace en Pod Security `restricted` (`socle.yaml:9`).
- **Retrait d'ECS propre** : aucune variable, sortie ou branche orpheline ; ressources EKS sans `count` ; `q_ecs` conservé à dessein pour les orphelins.
- **Portabilité Git Bash traitée à la source** : `jq` enveloppé pour retirer le CR, chemins natifs pour Terraform (`pwd -W`) et pour les outils Windows (`cygpath -m`, `nul`), LF imposé par `.gitattributes`.

## Issues

### Bloquantes

Aucune.

### Majeures

1. `tools/up.sh:46-48` et `:55-56` — **`--fin-lab` sans borne haute : AC3 et G1 contournés par une saisie erronée.**
   Constat : si l'heure saisie est déjà passée (par exemple l'heure de début tapée à la place de l'heure de fin), `FIN_LAB` est reporté de 24 h et `RESTANT_MIN` atteint jusqu'à 1 440 min. Rien ne vérifie `RESTANT_MIN ≤ DUREE_LAB_MIN` (240). Conséquences : le refus sous 75 min (AC3) est franchi, et les planifications G1 sont armées pour le lendemain, quand `LabRole` n'a plus de droits (XP00 S8b). Le contrôle `t10 <= now+120` (`:124`) ne voit rien. En sens inverse, `--debut-lab` échoue au passage de minuit (`:52`, « Start Lab dans le futur ») pour un lab démarré la veille.
   Correction : après le calcul, `[ "$RESTANT_MIN" -le "$DUREE_LAB_MIN" ] || echec "fin du lab dans plus de $DUREE_LAB_MIN min : heure saisie erronée ?"`. Pour `--debut-lab`, retirer 24 h si `DEBUT_LAB > maintenant` avant d'appliquer la même borne. Dev : devops.

2. `tools/up.sh:208-216`, `tools/down.sh:141` — **un `down` en échec ne bloque pas le `up` suivant (AC4) et la reprise n'inventorie pas (D11).**
   Constat : si le `destroy` échoue, `down.sh` garde le fichier de session (`noter_cout` le réécrit, `:103`) et l'état reste non vide. Le `up` suivant passe alors par la branche « état existant », qui ne lance pas `status.sh --strict` : il rafraîchit puis réapplique. Or AC4 exige qu'un échec « bloque le `up` suivant », le message de `down.sh:141` l'annonce (« session suivante bloquée tant que l'inventaire n'est pas vide »), et D11 (§ Reprise après expiration) exige qu'« avant tout nouvel `apply`, `up.sh` relance l'inventaire ; tout orphelin se supprime ou s'importe ». Une ressource sans nom imposé (NAT, EIP, VPC) créée hors de l'état serait dupliquée sans alerte.
   Correction : en cas d'échec, `down.sh` écrit un marqueur dans `session.env` (par exemple `EXTINCTION_INCOMPLETE=1`), et `up.sh` refuse de démarrer tant qu'il est présent (« relancer down.sh »). Dans la branche de reprise légitime (après un `up` interrompu), afficher l'inventaire non strict avant l'`apply` et lister ce qui n'est pas dans `tf state list`. Aligner ensuite le message de `down.sh:141`. Dev : devops.

3. `tools/controle-apres-deploiement.sh:370-385` — **le contrôle `sortie-sans-imds` peut être vert alors que l'IMDS est joignable.**
   Constat : le filtre juge un namespace conforme dès qu'il contient **une** politique `podSelector: {}` de type `Egress` dont **tous** les `ipBlock` `0.0.0.0/0` excluent `169.254.169.254/32`. Sans aucun `ipBlock` `0.0.0.0/0`, le test se réduit à `[] | all`, qui vaut `true`. Ainsi `egress: [{}]` (tout autoriser) passe. De même, `to: [{ipBlock 0.0.0.0/0 except IMDS}, {ipBlock 169.254.0.0/16}]` passe. Les deux cas ont été rejoués avec jq 1.8.2 : `true`. En outre, les politiques réseau s'additionnent : une seconde politique du namespace qui ouvre toute la sortie annule l'exception, et elle n'est pas examinée. Aujourd'hui seul `socle.yaml` existe et le résultat est juste. Mais AC11 fait hériter ce contrôle aux US suivantes, qui ajouteront namespaces et charts, et ce contrôle conditionne l'ouverture de l'ALB.
   Correction : pour chaque namespace, exiger au moins une politique conforme **et** qu'aucune politique `Egress` du namespace n'autorise l'IMDS. Concrètement, chaque règle de sortie doit avoir un `to` non vide, et chaque `ipBlock` dont le CIDR contient `169.254.169.254` doit l'exclure. Une approximation acceptable : n'admettre que `0.0.0.0/0` avec l'exception, ou des CIDR privés (`10.0.0.0/8`…). Ajouter les deux cas ci-dessus comme cas négatifs à un test du filtre. Dev : devops (le durcissement complet reste US-006).

4. `README.md:23` et `:12` — **documentation racine fausse après cette US.**
   Constat : « `demo.sh` et `bootstrap.sh` sont à venir » alors que `tools/bootstrap.sh` est livré (AC1), et la section « Démarrer / arrêter » ne mentionne pas le `bootstrap` préalable. La ligne `:12`, « Code : aucun service, front ou infrastructure pour l'instant », devient fausse avec `infra/`.
   Correction : retirer `bootstrap.sh` de « à venir », ajouter `bash tools/bootstrap.sh   # une fois (couche persistante)` avant `up.sh`, et mettre à jour la ligne « Code ». Dev : devops.

### Mineures

1. `tools/down.sh:31-46` — **une étape préalable non essentielle peut empêcher le `destroy`.** Dans `retirer_charges_cluster`, un échec de `flux uninstall`, de `helm uninstall --wait --timeout 5m` (dans un `while` d'un tube, sous `pipefail`) ou de `kubectl get svc` arrête `down.sh` par `set -e`, avant le `destroy`, sans ligne de journal ni phase. Un second lancement retombe sur le même échec, et seul G1 reste comme filet. Correction : `|| journal "ATTENTION : release $release non retirée"` sur chaque désinstallation et sur `flux`. Garder le refus explicite pour un Service `LoadBalancer` ou un PVC restant (`:45-46`), qui est voulu. Dev : devops.

2. `tools/controle-apres-deploiement.sh:142-185` — **balayage de l'ALB ouvert par défaut si aucune adresse n'est reconnue.** Les adresses de l'ALB sont identifiées par la description `ELB_app/$NOM_ALB/`. Si le format change, `IPS_ALB` est vide : `adresses-publiques` reste `ok` (« 0 de l'ALB ») et aucun résultat `balayage-alb` ni `alb-*` hors plage n'est produit. Correction : `resultat X17 balayage-alb ko "aucune adresse de l'ALB identifiée"` si `${#IPS_ALB[@]} -lt 2` (ALB sur deux zones), et de même si `${#IPS_NAT[@]} -ne 1`. Dev : devops.

3. `tools/down.sh:108-113`, `tools/controle-apres-deploiement.sh:44-48` et `:261-264` — **suppression de la sonde dupliquée et couplée.** Le même couple `delete-function` + `delete-log-group` est écrit trois fois. Dans `down.sh`, le groupe `/aws/lambda/collector-sonde-hors-plage` n'est supprimé que si la fonction existe encore. Si le contrôle a supprimé la fonction mais pas le groupe (erreur masquée par `|| true`), `down.sh` ne le retire pas et `status.sh --strict` échoue sur `logs:groupe`. Correction : une seule fonction `supprimer_sonde_externe` dans `lib.sh`, qui supprime les deux sans condition, appelée par `nettoyer`, `nettoyer_sonde_seule` et `down.sh`. Dev : devops.

4. `tools/up.sh:273` — **`curl -o /dev/null` sous Git Bash ne fonctionne que par accident.** Avec `MSYS_NO_PATHCONV=1` (`lib.sh:8`), le `curl` natif ne peut pas ouvrir `/dev/null`. Vérifié : il affiche `200` puis sort en 23, et seuls `-w` et `|| true` sauvent la mesure d'AC2. Le contrôle a déjà été corrigé avec `$NUL` (`controle-apres-deploiement.sh:64-65`). Correction : déplacer `chemin_natif` et `NUL` dans `lib.sh` et les utiliser dans `up.sh`. Dev : devops.

5. `tools/lib.sh:148-158` — **garde morte.** `k()` appelle `contexte_k8s "$NOM"` avec la constante `NOM="collector"`, qui correspond toujours à `*collector*` : la branche `echec` est inatteignable. Tous les appels passent déjà `--context "$NOM"`. Correction : supprimer `contexte_k8s` et son appel, ou contrôler le contexte réellement utilisé si une autre intention existait. Dev : devops.

6. `tools/lib.sh:139-144` — **exports sans effet.** `TF_VAR_alb_ouvert=false` et `TF_VAR_preuve_controle=""` sont toujours écrasés par `-var` (`ALB_FERME` dans `up.sh`/`down.sh`, `-var` explicites dans `alb.sh:304`). Une `TF_VAR_` a de plus une priorité inférieure aux `*.auto.tfvars` et ne protège rien. `infra/README.md` désigne d'ailleurs bien `-var` comme garde. Correction : retirer ces deux exports. Dev : devops.

7. `tools/up.sh:236-239`, `tools/alb.sh:304`, `tools/down.sh:82`, `infra/ephemere/securite.tf:10` — **adresse du poste affichée en clair dans le terminal.** Les sorties de Terraform passent par `masquer` (compte seulement) et non par `masquer_ip`. Or la clé `for_each` des règles de l'ALB contient l'adresse (`"443-<ip>/32"`), et elle apparaît dans les messages `apply_complete` et les différences de plan. Rien n'est versionné (journal JSONL dans `tools/.session/`, ignoré), mais la sortie peut finir dans une capture vidéo de soutenance ou un journal collé dans un rapport. Correction : `masquer_ip` sur ces trois tubes. Indexer en plus les règles par port seul (`local.sources_alb` n'a qu'un élément), ce qui retire l'adresse des adresses de ressources. Dev : devops.

8. `tools/controle-apres-deploiement.sh:214-219` — **interpréteur Python non portable.** Sous Linux, seul `python3` existe souvent, ce qui donne `nv`, donc rouge en permanence. Sous Windows, l'alias du Microsoft Store satisfait `command -v python` mais échoue, et `set -e` arrête alors tout le contrôle sans verdict. Correction : choisir `python3`, sinon `python`, et protéger la construction de l'archive par `|| { resultat X17 hors-plage nv "archive de la sonde non construite"; return 0; }`. Mentionner `python3` dans `tools/README.md:3`. Dev : devops.

9. `tools/bootstrap.sh:242-246` — **comparaison des serveurs de noms dépendante de la locale.** `sort` (GNU, collation de la locale, ponctuation ignorée) est comparé au `sort` octet par octet de jq, avec un espace final ajouté pour égaliser. Des noms comme `ns-1.awsdns-…` et `ns-12…` peuvent sortir dans un ordre différent, d'où un faux « délégation non constatée » (avertissement seulement). Correction : `LC_ALL=C sort`, comme dans `controle-apres-deploiement.sh:320`, et comparer sans l'astuce de l'espace final (`tr '\n' ' '` puis `sed 's/ $//'`). Dev : devops.

10. **Duplications à factoriser dans `lib.sh`** :
    - recherche de la zone : `up.sh:74-76`, `status.sh:200-201`, `inventaire.sh:115-116` ;
    - recherche du certificat `ISSUED` : `bootstrap.sh:255-256`, `up.sh:77-78`, `status.sh:204-205` ;
    - sauvegarde d'état : `lib.sh:115-120` et `bootstrap.sh:233-238`, identiques au dossier et au préfixe près ;
    - calcul du coût : `down.sh:98` et `status.sh:189` ;
    - identifiant de session : `up.sh:211` et `down.sh:78` ;
    - `terraform -chdir="$DOSSIER_PERSISTANT"` : `bootstrap.sh:183` et `controle-apres-deploiement.sh:421` ;
    - deux fonctions homonymes `controler_persistant`, de sens différents (`up.sh:72`, qui refuse ; `status.sh:197`, qui affiche et compte).

    Correction : `zone_collector`, `certificat_emis`, `sauvegarder_etat <dossier> <prefixe> <suffixe>`, `cout_estime <minutes>`, `nouvelle_session`, `tfp` dans `lib.sh`, et renommer l'une des deux `controler_persistant` (par exemple `exiger_persistant` dans `up.sh`). Dev : devops.

11. **Commentaires de justification dans le code** : `lib.sh:46`, `infra/persistant/photos.tf:1`, `import.tf:1`, `cle-secrets-eks.tf:1-2`, `journaux.tf:1`, `infra/front/valeurs-aws.yaml:1-3`. La convention n'admet que les en-têtes d'usage et les références de mesure (précédent US-011, mineure 7). Ces raisons figurent déjà dans `infra/README.md` et `tools/README.md`, sauf celle de `journaux.tf` (« créés avant Fluent Bit, sinon il les crée sans rétention »). Correction : supprimer ces commentaires et ajouter la raison de `journaux.tf` à `infra/README.md`. Dev : devops.

12. `infra/README.md:17` — **premier lancement attendu incomplet** : la liste omet la clé KMS et son alias (`cle-secrets-eks.tf`), créés au premier `bootstrap`. Correction : ajouter « clé KMS et alias des Secrets EKS ». Dev : devops.

### Suggestions

1. **Sorties non consommées** : `infra/ephemere/sorties.tf:9` (`sous_reseaux_prives`), `:66` (`version_ami_noeuds`, alors que le contrôle relit le paramètre SSM, `controle-apres-deploiement.sh:527`), `:70` (`arn_groupe_cible_front`) ; `infra/persistant/sorties.tf:1` (`id_zone`) et `:14` (`groupes_journaux`). Aucune n'est lue par les scripts. Utiliser `version_ami_noeuds` dans le contrôle et retirer les autres, ou les assumer comme interface de lecture humaine. Dev : devops.

2. `.trivyignore:14-26` — **exceptions globales.** AWS-0053 et AWS-0040 sont ignorées pour tout le dépôt : un second ALB public ou un autre point d'accès EKS ne serait plus signalé par Trivy. Seul le contrôle d'exécution (« un seul load balancer ») compense. Envisager `.trivyignore.yaml` avec `paths` (et adapter `verifier-exceptions.sh`). Rappel : `devsecops.md` § 3 exige la validation de la fonction Sécurité **puis** de Martin, et les deux entrées portent encore « Sécurité à confirmer ». Dev : devops ; validation : Sécurité.

3. **AC6 sans garde en CI** : la CI lance Trivy `config` et `shellcheck`, mais ni `terraform fmt -check`, ni `init -backend=false && validate`, ni `helm lint`. Une régression des validations de variables (G4, AC6) ne serait vue qu'en session AWS. Ajouter ces trois commandes au job « Workflows et scripts » quand `infra/**` change, et au besoin un `terraform test` (`.tftest.hcl` avec `expect_failures` sur `noeuds_max = 5` et `version_eks = "1.34"`) qui prouve AC6 sans AWS (fournisseur simulé). Dev : testeur, avec devops pour le job.

4. `infra/front/values.yaml:5-7` et `templates/front.yaml:31`, `:58` — `service.port` sert de port du conteneur, alors que le port du Service est écrit en dur à 80. Renommer en `conteneur.port`, ou exposer les deux. Dev : devops.

5. `tools/lib.sh:44` — `masquer_ip` masque aussi `169.254.169.254`. Le rapport archivé affiche alors « politique de sortie sans `<ip>/32` » (`docs/mesures/US-004/controle-20261120T1146Z.txt`), ce qui retire sa valeur de preuve. Écrire « sans l'IMDS » dans le détail, ou exclure `169.254.0.0/16` du masquage. Dev : devops.

## Verdict

| Sévérité | Nombre |
|---|---|
| Bloquante | 0 |
| Majeure | 4 |
| Mineure | 12 |
| Suggestion | 5 |

**Fusion après corrections des quatre majeures.** L'architecture de l'US est solide :
- **Couches** : deux couches séparées, la persistante protégée contre toute destruction.
- **Garde-fous de coût** : G1, G2, G4 et G5 implémentés et tracés.
- **AC10** : ouverture de l'ALB verrouillée par verdict, état et précondition.
- **Contrôle après déploiement** : rouge par défaut, X20 mené sans jamais afficher de valeur.
- **Retrait d'ECS** : aucun résidu.

Les vérifications hors AWS sont vertes (shellcheck, `fmt`, `validate`, `helm lint`, exceptions Trivy), et les mesures archivées ne contiennent ni compte, ni adresse, ni secret.

Les majeures portent sur trois garde-fous qui peuvent être contournés et sur une documentation devenue fausse :
- la saisie de `--fin-lab` neutralise AC3 et G1 (1) ;
- un `down` en échec ne bloque pas le `up` suivant, contrairement à AC4 et D11 (2) ;
- le contrôle IMDS dont hériteront les US suivantes peut être vert à tort (3) ;
- le README racine annonce `bootstrap.sh` « à venir » (4).

Répartition : devops pour les 4 majeures, les 12 mineures et les suggestions 1, 2, 4 et 5 ; testeur pour la suggestion 3 ; validation Sécurité attendue pour les exceptions Trivy. Aucune correction back.

## Retours traités

Corrections faites dans l'arbre de travail de la branche, sans commit et sans appel à l'API AWS.

| N° | Traitement |
|---|---|
| Majeure 1 | `up.sh` : heure validée `00:00` à `23:59` ; `--debut-lab` dans le futur est reporté à la veille, ce qui couvre le passage de minuit ; `RESTANT_MIN` est borné à `DUREE_LAB_MIN` (240), sinon refus « heure saisie erronée ? ». Une saisie de l'heure de début à la place de l'heure de fin ne peut plus franchir AC3 ni armer G1 pour le lendemain |
| Majeure 2 | `down.sh` écrit `EXTINCTION_INCOMPLETE=1` dans `session.env` dès son lancement et ne retire le fichier qu'une fois le `destroy` **et** l'inventaire strict verts : un échec ou une interruption laisse la marque. `up.sh` refuse alors de démarrer (« relancer down.sh », AC4), et `status.sh` l'affiche. Dans la branche de reprise d'un `up` interrompu, `up.sh` compare l'inventaire à l'état (`signaler_hors_etat`) et signale chaque ressource hors état avant l'`apply` (D11). Messages de `down.sh` alignés |
| Majeure 3 | Filtre extrait dans `tools/sortie-sans-imds.jq` et appliqué à l'**union** des politiques du namespace : au moins une politique de sortie sur tous les pods, et chaque règle de sortie de chaque politique doit avoir un `to` non vide dont aucun `ipBlock` ne couvre 169.254.169.254 sans l'exclure (calcul d'appartenance CIDR). `policyTypes` implicite pris en compte. Test sans AWS `tools/tests/sortie-sans-imds.sh` : 7 cas positifs, 10 négatifs (dont `egress: [{}]`, `169.254.0.0/16` après l'exception, `128.0.0.0/1`, seconde politique ouverte), tous verts |
| Majeure 4 | `README.md` : ligne « Code » à jour, `bootstrap.sh` ajouté avant `up.sh`, `status.sh` et `alb.sh` cités, seul `demo.sh` reste « à venir » ; lignes `infra/` et `tools/` de la carte passées « En cours » |
| Mineure 1 | `retirer_charges_cluster` : échec de `flux uninstall`, de la liste ou d'un `helm uninstall`, ou lecture impossible des Services et volumes → ligne `ATTENTION` et poursuite vers le `destroy`. Refus explicite conservé pour un Service `LoadBalancer` ou un PVC restant |
| Mineure 2 | `ko` sur `balayage-alb` si moins de 2 adresses d'ALB identifiées, et sur `balayage-nat` si les adresses de NAT ne sont pas exactement 1 |
| Mineure 3 | `supprimer_sonde_externe` dans `lib.sh` : supprime la fonction et son groupe de journaux sans condition ; appelée par `nettoyer`, le contrôle (avant et après la sonde) et `down.sh`. `nettoyer_sonde_seule` et la copie de `down.sh` retirées |
| Mineure 4 | `chemin_natif` et `NUL` déplacés dans `lib.sh` ; `up.sh` mesure AC2 avec `curl -o "$NUL"` |
| Mineure 5 | `contexte_k8s` et son appel supprimés ; `k()` passe `--context "$NOM"` directement. `tools/README.md` mis à jour |
| Mineure 6 | Exports `TF_VAR_alb_ouvert` et `TF_VAR_preuve_controle` retirés ; la garde reste `-var` (`ALB_FERME`, `alb.sh`) |
| Mineure 7 | `masquer_ip` sur les sorties de `tf apply -refresh-only` et `tf apply -json` (`up.sh`), `tf apply` (`alb.sh`) et `tf destroy` (`down.sh`). Règles d'entrée de l'ALB indexées par port seul (`for_each = toset(["80", "443"])`, `local.source_alb`) : l'adresse du poste n'apparaît plus dans les adresses de ressources |
| Mineure 8 | Interpréteur choisi parmi `python3` puis `python`, retenu seulement s'il importe `zipfile` (écarte l'alias du Microsoft Store) ; construction de l'archive protégée (`nv`, sans arrêt du contrôle). Prérequis de `tools/README.md` à jour |
| Mineure 9 | `bootstrap.sh` : `LC_ALL=C sort` et comparaison sans espace final |
| Mineure 10 | Dans `lib.sh` : `zone_collector` (up, status, inventaire), `certificat_emis` (bootstrap, up, status), `sauvegarder_etat <dossier> <préfixe> <suffixe>` (éphémère et persistant, `sauvegarder_persistant` retirée), `cout_estime` (down, status), `nouvelle_session` (up, down), `tfp` (bootstrap, contrôle). `controler_persistant` de `up.sh` renommée `exiger_persistant` |
| Mineure 11 | Commentaires retirés de `lib.sh` (`jq`), `photos.tf`, `import.tf`, `cle-secrets-eks.tf`, `journaux.tf` et `valeurs-aws.yaml`. Raisons reportées : Fluent Bit et clé KMS persistante dans `infra/README.md`, image remplacée par celle de la CI (D10) dans `infra/README.md`, CRLF de `jq` dans `tools/README.md` ; bucket en CLI et imports y figuraient déjà |
| Mineure 12 | `infra/README.md` : « clé KMS et alias des Secrets EKS » ajoutés au premier lancement attendu |
| Suggestion 1 | Non traitée (hors demande) : sorties conservées comme interface de lecture humaine |
| Suggestion 2 | Non traitée : les deux exceptions restent « Sécurité à confirmer », validation Sécurité puis Martin attendue (`devsecops.md` § 3) |
| Suggestion 3 | Non traitée : CI Terraform et Helm hors périmètre de l'US |
| Suggestion 4 | `conteneur.port` (port de nginx) et `service.port` (80, auparavant écrit en dur) séparés dans `values.yaml` et le gabarit ; `service.nodePort` inchangé, donc `up.sh` aussi |
| Suggestion 5 | Le détail du contrôle ne cite plus l'adresse : « toute sortie exclut l'IMDS dans chaque namespace applicatif (union des politiques) ». Le rapport déjà archivé du 2026-11-20 n'est pas réécrit |

### Vérifications après corrections

| Commande | Résultat |
|---|---|
| `bash -n` sur `tools/*.sh` et `tools/tests/*.sh` | Aucune erreur |
| `shellcheck --severity=warning` (image épinglée de `tools/ci/outils.yml`, mêmes fichiers que la CI plus le test) | 0 constat. Le passage `--severity=style -x` ne donne que les `info` déjà connus (SC2329, SC2016) |
| `bash tools/tests/sortie-sans-imds.sh` | 17 cas sur 17 |
| `terraform fmt -check -recursive infra` | Conforme (le `persistant.auto.tfvars` local, ignoré par git, a été reformaté, espaces seulement) |
| `terraform init -backend=false` puis `validate` (`persistant`, `ephemere`) | `Success! The configuration is valid.` (les deux) |
| `helm lint infra/front` et `-f valeurs-aws.yaml` | 0 échec ; rendu : `containerPort: 8080`, Service `port: 80`, `nodePort: 30080` |
| `bash tools/ci/trivy.sh config infra infra` | 0 bloquant (10 LOW, 9 MEDIUM) |

À rejouer en session AWS : `up.sh` (fenêtre, reprise et marque d'extinction), `down.sh` (marque retirée sur succès), contrôle après déploiement (filtre IMDS, sonde avec `python3`).
