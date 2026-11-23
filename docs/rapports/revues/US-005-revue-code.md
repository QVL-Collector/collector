# US-005 — Revue de code

| Champ | Valeur |
|---|---|
| US | [US-005 — Garde-fous de coût : destruction programmée et journal des coûts](../../../backlog/US-005-garde-fous-cout.md) |
| Branche | `feat/US-005-garde-fous-cout`, diff non commité contre `main` (`47ad500`) : 6 fichiers modifiés (+262/-77) et 1 fichier non suivi (`tools/tests/garde-fous-cout.sh`) |
| Date | 2026-11-23 |
| Périmètre | `tools/lib.sh`, `tools/up.sh`, `tools/down.sh`, `tools/tests/garde-fous-cout.sh`, `.github/workflows/ci.yml` (étape « Tests des scripts sans AWS »), `tools/README.md`, backlog (section Avancement) |
| Références | [D11](../../decisions/D11-iac-allumage-cout.md) (G1, G5, G7, MI5, X15), [revue US-004](US-004-revue-code.md) |
| Hors revue (connu) | Preuves en session AWS listées dans le backlog (« Reste à prouver ») : exécution réelle de G1 sous `LabRole`, casse acceptée par Scheduler, interruption par Ctrl-C, tâche planifiée et boîte de dialogue réelles, ligne réelle de `couts.csv` |
| Verdict | **Fusion après corrections** (0 bloquante, 1 majeure, 6 mineures, 3 suggestions) |

## Vérifications exécutées

Aucun appel à l'API AWS, aucune tâche planifiée enregistrée.

| Commande | Résultat |
|---|---|
| `bash tools/tests/garde-fous-cout.sh` (Git Bash) | 53 cas, tous verts (G1 : 20 ; G5 journal : 11 ; G5 seuil : 8 ; G7 : 14), code retour 0 |
| `bash tools/tests/sortie-sans-imds.sh` | Vert, aucune régression |
| `shellcheck --severity=warning` (image `koalaman/shellcheck:v0.11.0` épinglée dans `tools/ci/outils.yml`) sur `lib.sh`, `up.sh`, `down.sh`, `tests/*.sh` | 0 avertissement. En sévérité `style` : uniquement des `info`, SC1091 (fichier sourcé non suivi sans `-x`) et SC2329 sur `powershell.exe()` du test (`garde-fous-cout.sh:134`), appelée indirectement : faux positif |
| `actionlint` (image épinglée) | Vert |
| Script PowerShell du rappel généré par `script_rappel_windows`, puis analysé par `[Parser]::ParseFile` (sans `Register-ScheduledTask`) | 0 erreur pour le script externe et pour la commande encodée interne. Code des caractères décodés : `é` = 233, `«` = 171, donc l'encodage UTF-16LE est correct. Argument encodé : 784 caractères |
| Reproduction du piège `armer_en_sortie` (copie exacte de la fonction, `armer_garde_fous` remplacée par l'écriture d'un témoin, scratchpad) | stderr normal : armé, code 129. stderr fermé (`2>&-`) : armé. stderr en erreur d'écriture (`2>/dev/full`) : **non armé**, code 4 (voir majeure 1) |
| Adresses Terraform ciblées par `ids_garde_fou` | Toutes présentes dans `infra/ephemere` au module racine, sans `count` ni `for_each` : `aws_eks_cluster.principal`, `aws_eks_node_group.principal`, `aws_db_instance.principale`, `aws_lb.public`, `aws_nat_gateway.unique`, `aws_eip.nat`. Aucun bloc `module` |
| Protections contre la suppression | `enable_deletion_protection = false` (ALB), `deletion_protection = false` et `skip_final_snapshot = true` (RDS) : les cibles G1 peuvent aboutir |
| Fins de ligne | `eol=lf` sur les scripts modifiés, aucun CR dans le test |
| `docs/mesures/couts.csv` | En-tête et `2026-11-20;58;0.33` : ni compte, ni adresse, ni session (AC8) |
| Appelants des symboles modifiés | `controler_budget` : plus aucun appelant. `cumul_couts` (retourne désormais `0.00` au lieu de `0`) : `down.sh:108` et `status.sh:47`, tous deux numériques, compatibles. `CSV_COUTS` : `controle-apres-deploiement.sh:439` (simple `cat`), compatible. Aucune régression |
| Filtre de la CI (`tools/ci/changements.sh:80`) | `*.sh` et `tools/*` déclenchent le job « Workflows et scripts » : une modification de `tools/tests/` ou de `lib.sh` rejoue bien les tests |

## Points forts

- **Cibles G1 exactes, lues dans l'état** (`lib.sh:196-227`) : sélection par adresse Terraform et `mode == "managed"`, donc une source de données homonyme (`data.aws_kms_alias`) est ignorée. Lecture par `terraform show -json` et non plus par les sorties : un `apply` partiel arme ce qui existe (test « apply partiel : NAT et adresse seules armées »). Aucune cible persistante possible : le test prouve que clé KMS et bucket ne produisent rien, et qu'aucune cible ne touche KMS, S3, Route 53, ACM, journaux ou Secrets Manager (MI5).
- **Logique pure séparée des appels AWS** : `ids_garde_fou`, `planifications_garde_fou` et `cible_planification` sont des filtres jq testables sans AWS. `planifier` n'a plus qu'à appeler la CLI. Ce découpage remplace six blocs `if` dupliqués (`up.sh` de `main`).
- **Pas de double armement** : `GARDE_FOU_TENTE=1` est posé dès l'entrée dans `armer_garde_fou` (`up.sh:99`), et le piège ne réarme que si `APPLY_LANCE=1` et `GARDE_FOU_TENTE=0` (`up.sh:141`). Il se désarme lui-même (`trap - EXIT`) et conserve le code de sortie. Le piège n'est posé qu'au moment de l'`apply` (`up.sh:242-247`), jamais en mode `--prolonger`. Un état vide n'arme aucune planification (`up.sh:103-106`, testé).
- **G5 avant tout appel AWS** (`up.sh:175-177`), décision factorisée (`autoriser_allumage_g5`) et testée aux bornes (34,99 $ autorisé, 35,00 $ refusé, répétition autorisée, CRLF, journal absent, au-delà du budget). `--prolonger` reste hors G5, comme sur `main`.
- **Journal des coûts robuste** : en-tête recréé si le fichier est vide (`-s`), fin de ligne ajoutée si elle manque (`lib.sh:189-194`), cumul correct en CRLF. Le test relit le `couts.csv` du dépôt (AC8) : format strict et absence de compte, d'adresse IP, d'identifiant de session, d'ARN ou de clé.
- **Rappel G7 sans privilège** : aucun `-User`, `Principal` ni `RunLevel` (testé). Double encodage UTF-16LE + base64 correct, ce qui évite tout problème de guillemets et d'accents. Apostrophes doublées pour PowerShell et protégées pour `sh` (testé). `StartWhenAvailable` couvre le poste en veille. L'ancienne tâche est retirée avant la nouvelle, et une échéance déjà passée est refusée sans rien enregistrer.
- **AC4 renforcé** : `supprimer_planifications` (`lib.sh:238-247`) attend la disparition du groupe, ce qui évite qu'un groupe `DELETING` fasse échouer l'inventaire strict. Le rappel n'est annulé qu'après une extinction réussie (`down.sh:252`) : un `down` en échec garde le rappel.
- **CI** : l'étape ajoutée n'utilise aucune action tierce (checkout déjà épinglé par SHA). Sous le shell par défaut `bash -e`, le premier test rouge fait échouer l'étape.

## Issues

### Bloquantes

Aucune.

### Majeures

1. `tools/up.sh:138-146` — **le piège `EXIT` peut abandonner l'armement si l'écriture sur stderr échoue, alors que c'est justement le cas « terminal fermé » annoncé.**
   Constat : `armer_en_sortie` s'exécute sous `set -euo pipefail`, que bash conserve dans les pièges. Sa première action est `journal "G1 : sortie de up.sh avant l'armement…"` (`:142`), appelé sans `||`. `journal` est un tube `printf | sed >&2` (`lib.sh:52`) : si l'écriture sur stderr échoue, `sed` sort en erreur, `pipefail` propage l'échec et `errexit` termine le shell avant `armer_garde_fous`. La reproduction a été faite avec la fonction copiée telle quelle : avec `2>/dev/full`, aucun armement et code 4 (contre 129 attendu). Or `tools/README.md` et le backlog (AC1) annoncent un armement sur `HUP`, « terminal fermé ». Dans ce cas, le terminal a disparu, et sur un pseudo-terminal raccroché les écritures échouent (EIO). L'échec n'est pas reproduit en session réelle sous Git Bash : le scénario est à confirmer avec le test « AC1 (interruption) ». Le mécanisme, lui, est démontré. Les `journal` appelés plus loin, dans `armer_garde_fous || true`, ne sont pas concernés, car `errexit` y est neutralisé.
   Correction : dans `armer_en_sortie`, juste après `local statut=$?` et `trap - EXIT`, ajouter `set +e`. On peut aussi écrire `journal … || true` et, en mode `HUP`, rediriger stderr vers un fichier de `tools/.session/` pour garder une trace. Ajouter un cas au test (voir mineure 6) : piège déclenché avec `2>/dev/full`, armement constaté. Dev : devops.

### Mineures

1. `tools/up.sh:132-136` — **rappel G7 programmé sur une session vide.** Si l'`apply` échoue sans rien créer, `armer_garde_fou` conclut « rien à armer » (`:103-106`), mais `armer_garde_fous` programme quand même le rappel du poste. Trois heures plus tard, une boîte de dialogue demande de lancer `down.sh` alors que rien ne tourne. Ce n'est pas dangereux, mais c'est trompeur. Correction : conditionner `programmer_rappel_poste` à au moins une planification à armer (par exemple, `armer_garde_fou` renvoie un code distinct ou pose `G1_ARME=1`). Dev : devops.

2. `tools/up.sh:99` et `:112-123` — **fenêtre sans garde-fou si l'armement est interrompu.** `GARDE_FOU_TENTE=1` est posé avant `supprimer_planifications` (jusqu'à 55 s d'attente) et avant la recréation du groupe (jusqu'à 60 s). Un Ctrl-C dans cette fenêtre d'environ 2 min laisse le groupe purgé, aucune planification et aucun rappel, et le piège ne réessaie pas. Correction : l'armement est idempotent (purge puis recréation). On peut donc poser `GARDE_FOU_TENTE=1` à la fin de `armer_garde_fous`, pour que le piège rejoue un armement interrompu. Un second Ctrl-C reste possible pour sortir. Dev : devops.

3. `tools/up.sh:128` — **nombre de cibles écrit en dur** : « sur 6 cibles facturables » s'affiche aussi après un `apply` partiel (par exemple « 2 planification(s) armée(s) sur 6 »). Correction : afficher `armees` sur le nombre de lignes de `planifs` (`grep -c . <<<"$planifs"`) et signaler l'écart s'il y en a un. Dev : devops.

4. `tools/lib.sh:279-287` — **annulation trop large sous Linux** : `annuler_rappel_poste` supprime toutes les tâches `at` de la file `c` de l'utilisateur, y compris celles qui ne viennent pas de Collector. Correction : repérer la tâche par son contenu (`at -c <id> | grep -q Collector`) ou mémoriser son numéro dans `session.env`. Impact faible : poste Windows en pratique. Dev : devops.

5. `tools/down.sh:103-104` — **coût calculé deux fois** : `noter_cout` appelle `cout_estime` pour le journal, puis `ajouter_cout` recalcule le même coût via `ligne_cout`. Les deux valeurs sont identiques aujourd'hui, mais elles peuvent diverger si l'une des deux évolue. Correction : faire écrire la ligne par `ajouter_cout` et lui faire renvoyer le coût (ou passer le coût en argument), puis réutiliser cette valeur dans le journal. Dev : devops.

6. `tools/up.sh:97-146` — **la logique du piège et de l'armement n'est pas testée sans AWS.** `armer_en_sortie`, le refus à T-30 et la purge avant recréation vivent dans `up.sh`, qui ne peut pas être sourcé : les 53 cas couvrent les filtres jq mais pas la nouveauté principale d'AC1, l'armement après interruption. Le plan de test de l'US prévoit AC1 en session AWS, donc ce n'est pas bloquant, mais la majeure 1 aurait été détectée. Correction : déplacer `armer_garde_fou`, `armer_garde_fous` et `armer_en_sortie` dans `lib.sh` (ou dans un fichier sourçable), puis tester avec `tf` et `aws` simulés : pas de double armement, pas d'armement sur un état vide, armement après `exit 130`, armement avec stderr en erreur. Dev : devops, avec le testeur.

### Suggestions

1. `infra/ephemere/sorties.tf:13`, `:17` et `:78` — **sorties devenues orphelines** : `id_nat`, `id_allocation_eip` et `nom_cluster_eks` n'étaient lues que par l'ancien `armer_garde_fou`. Hors `sandbox/`, plus aucun script ne les consomme (`arn_alb`, `id_rds` et `nom_node_group` restent utilisées par `controle-apres-deploiement.sh`). Les retirer, ou les assumer comme interface de lecture humaine (même traitement que la suggestion 1 de la revue US-004). Dev : devops.

2. **Échéances écrites en dur** : `1800` et `600` (`lib.sh:215`), `30 * 60` (`lib.sh:253`, `up.sh:107`, `:169`, `:303`), `10 * 60` (`up.sh:303`). Si D11 avance T-30 de 5 min (marge évoquée dans « Reste à prouver »), il faut toucher cinq endroits. Introduire `ECHEANCE_LOURDE_MIN=30` et `ECHEANCE_FINALE_MIN=10` dans `lib.sh`, à côté de `RAPPEL_AVANT_FIN_MIN`. Dev : devops.

3. `tools/up.sh:79-82` — `planifier` lance trois `jq` pour lire `nom`, `echeance` et `cible`. Un seul appel suffit : `IFS=$'\t' read -r nom echeance cible < <(jq -r '[.nom, .echeance, .cible] | @tsv' <<<"$planif")`. Dev : devops.

## Points d'attention pour la session AWS (non comptés)

- **Ctrl-C pendant l'`apply`** : `tee` et `jq` reçoivent aussi SIGINT et s'arrêtent tout de suite, alors que Terraform poursuit son arrêt propre en écrivant dans un tube fermé. Selon la plateforme, Terraform peut être interrompu avant d'avoir écrit l'état, et le piège n'armerait alors que ce que l'état connaît. C'est une hypothèse, non vérifiée : à observer pendant le test « AC1 (interruption) ». Le tube existait déjà dans US-004.
- **Fermeture du terminal** : Terraform reçoit aussi SIGHUP. Vérifier ce qui reste dans l'état et ce qui est armé (voir majeure 1).
- **Boîte de dialogue** : l'option `ServiceNotification` du `MessageBox` WPF n'a pas été affichée pendant la revue. Le contrôle visuel est à faire avec `Get-ScheduledTask Collector-rappel-fin-lab`.

## Verdict

| Sévérité | Nombre |
|---|---|
| Bloquante | 0 |
| Majeure | 1 |
| Mineure | 6 |
| Suggestion | 3 |

**Fusion après correction de la majeure.** Les garde-fous sont bien conçus :
- **G1** : cibles exactes lues dans l'état, couche persistante jamais ciblée, pas de double armement, aucune planification sur un état vide.
- **G5** : seuil contrôlé avant tout appel AWS, testé aux bornes.
- **G7** : rappel sans privilège, encodage vérifié.
- **AC8** : journal relu automatiquement.

Les vérifications hors AWS sont vertes : 53 cas, shellcheck, actionlint, analyse PowerShell. Aucune régression n'a été trouvée sur les appelants modifiés.

La majeure porte sur la seule garantie annoncée mais non tenue : l'armement par le piège quand stderr n'est plus écrivable, cas du terminal fermé. La correction tient en une ligne (`set +e` dans `armer_en_sortie`), plus un test.

Répartition : devops pour la majeure, les 6 mineures et les 3 suggestions, avec le testeur pour la mineure 6. Aucune correction front ni back.

## Retours traités

Corrections apportées le 2026-11-23 sur la même branche, sans commit et sans appel AWS.

| N° | Traitement |
|---|---|
| Majeure 1 | `armer_en_sortie` (désormais dans `lib.sh`) passe en `set +e` juste après `trap - EXIT`, et son `journal` est suivi de `\|\| true`. Sur `HUP` (code 129), stderr est redirigé vers `tools/.session/armement-sortie.log` pour garder une trace. Cas ajoutés : stderr en erreur d'écriture (`2>/dev/full`), stderr fermé (`2>&-`) et terminal fermé (trace écrite), tous armés et code de sortie conservé. Sans le correctif, ces 4 cas échouent (code 4 ou 1, aucun armement) |
| Mineure 1 | `armer_garde_fou` pose `G1_A_ARMER`, le nombre de planifications attendues d'après l'état. Le rappel passe par `programmer_rappel_session`, qui ne programme rien si ce nombre vaut 0 (voie normale et `--prolonger`). Testé : un état sans ressource facturable ne donne ni appel Scheduler ni rappel |
| Mineure 2 | `GARDE_FOU_TENTE=1` est posé à la fin de `armer_garde_fous`. Testé : un armement interrompu (`exit 130` pendant la recréation du groupe) est rejoué par le piège, avec 6 planifications créées et le groupe recréé au second essai |
| Mineure 3 | Message « N planification(s) armée(s) sur M attendue(s) d'après l'état », avec `NB_CIBLES_G1=6` comme maximum. L'écart est signalé : cibles absentes (apply partiel) et planifications refusées (retour 1). Testé (état complet, apply partiel, refus) |
| Mineure 4 | Le script `at` commence par `: Collector-rappel-fin-lab`. `annuler_rappel_poste` ne supprime que les tâches de la file `c` dont `at -c` contient cette ligne. Testé avec `atq`, `at -c` et `atrm` simulés : seule la tâche Collector est annulée |
| Mineure 5 | `ajouter_cout` écrit la ligne et renvoie le coût écrit. `noter_cout` (`down.sh`) réutilise ce coût, le second calcul est supprimé. Testé (coût renvoyé = coût de la ligne) |
| Mineure 6 | `planifier`, `armer_garde_fou`, `armer_garde_fous`, `armer_en_sortie` et `programmer_rappel_session` sont déplacées de `up.sh` vers `lib.sh`. Nouvelle section du test, avec `aws`, `tf`, `sleep` et `programmer_rappel_poste` simulés : purge avant recréation, refus à T-30, apply partiel, pas de double armement, rien avant l'apply, armement après `exit 130`, interruption pendant l'armement, cas stderr. Le test passe de 53 à 86 cas |
| Suggestion 1 | Sorties `id_nat`, `id_allocation_eip` et `nom_cluster_eks` retirées de `infra/ephemere/sorties.tf` : aucun script ni document hors `sandbox/` ne les lit (le bac à sable a son propre `sorties.tf`). `terraform fmt -check` et `validate` sont verts |
| Suggestion 2 | Ajout dans `lib.sh` de `ECHEANCE_LOURDE_MIN=30`, `ECHEANCE_FINALE_MIN=10` et des fonctions `echeance_lourde` et `echeance_finale`, utilisées par `planifications_garde_fou`, `message_rappel`, le refus à T-30, `rappel_fin`, le message final de `up.sh` et `status.sh`. Le délai d'attente du HTTPS 200 (`30 * 60`) n'a pas de lien avec ces échéances et reste tel quel |
| Suggestion 3 | `planifier` lit `nom`, `echeance` et `cible` en un seul appel `jq` (`@tsv`). Testé : nom et échéance transmis à `create-schedule` |

Vérifications après corrections : `bash -n` sur les scripts modifiés ; `shellcheck --severity=warning` (image épinglée `koalaman/shellcheck:v0.11.0`) donne 0 avertissement (SC2034 neutralisé sur les variables lues par les fonctions de `lib.sh`) ; `bash tools/tests/garde-fous-cout.sh` passe 86 cas sur 86 ; `bash tools/tests/sortie-sans-imds.sh` est vert ; `terraform fmt -check -recursive infra` et `terraform validate` (`infra/ephemere`, `init -backend=false`) sont verts. README mis à jour (trace du `HUP`, absence de rappel sans ressource, annulation ciblée de `at`).
