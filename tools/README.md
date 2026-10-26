# tools — allumage et extinction de l'environnement AWS

Scripts Bash ([D11](../docs/decisions/D11-iac-allumage-cout.md)), lancés sous Git Bash par `bash tools/<script>.sh`. Prérequis : AWS CLI, Terraform 1.14.5, `jq`, `curl` ; `kubectl` et Helm pour EKS. Profil `collector-lab` à jour après « Start Lab », jamais d'identifiant en argument ni dans le dépôt.

| Script | Rôle |
|---|---|
| `up.sh eks\|ecs --debut-lab HH:MM` | Contrôles (identité, région, refus sous 75 min restantes, seuil de coût G5, version EKS, couche persistante, inventaire si l'état est vide), purge des photos, `apply` horodaté, destruction programmée T-30/T-10 (G1), kubeconfig, front par Helm si `--chart`, attente du HTTPS 200 |
| `up.sh --prolonger --debut-lab HH:MM` | Réarme la destruction programmée après un nouveau « Start Lab » |
| `down.sh` | Flux puis releases Helm, vidage des photos, secrets `collector/*` supprimés sans délai, `destroy` avec reprise sur les interfaces, planifications supprimées, inventaire strict (G2), ligne de coût (G5) |
| `inventaire.sh [--strict]` | Inventaire du compte indépendant de l'état, hors couche persistante ; `--strict` échoue s'il reste une ressource |
| `lib.sh` | Fonctions communes (profil, masquage du compte, journal de session) |

**`down.sh` doit se terminer avant « End Lab »** : lab arrêté, `LabRole` perd ses droits et la destruction programmée échoue (XP00 S8b). `up.sh` affiche l'heure limite de lancement de `down.sh`.

Traces : phases horodatées dans `docs/mesures/sessions.csv` (X4, X5), coût par session dans `docs/mesures/couts.csv` (G5, sans compte, adresse ni identifiant de session). Fichiers de session et journaux d'`apply` dans `tools/.session/`, copies d'état dans `infra/.etat-sauvegardes/` : ignorés par git.
