# tools — allumage et extinction de l'environnement AWS

Scripts Bash ([D11](../docs/decisions/D11-iac-allumage-cout.md)), lancés sous Git Bash par `bash tools/<script>.sh`. Prérequis : AWS CLI, Terraform 1.14.5, `jq`, `curl`, `openssl` ; `kubectl` et Helm ; Python (archive de la sonde Lambda), Docker (Trivy) et `gh` (journaux de la CI) pour le contrôle après déploiement. Profil `collector-lab` à jour après « Start Lab », jamais d'identifiant en argument ni dans le dépôt.

| Script | Rôle |
|---|---|
| `bootstrap.sh` | Une fois, rejouable sans effet (AC1) : bucket des photos en CLI, puis `infra/persistant` (zone, certificat, budget, configuration du bucket, groupes de journaux) ; refuse tout plan qui détruit ou remplace ; affiche la délégation à poser dans Cloudflare |
| `up.sh --fin-lab HH:MM` | Contrôles (identité, région, refus sous 75 min restantes, seuil de coût G5, version EKS, couche persistante, `status.sh --strict` si l'état est vide), purge des photos, `apply` horodaté avec l'ALB fermé, destruction programmée T-30/T-10 (G1), kubeconfig, socle `infra/k8s`, front par Helm (`infra/front` par défaut), HTTPS 200, puis contrôle après déploiement (AC11) ; code retour non nul si le contrôle est rouge. `--fin-lab` reprend l'heure du compteur du lab (XP01 X14) ; `--debut-lab HH:MM` reste accepté |
| `up.sh --prolonger --fin-lab HH:MM` | Réarme la destruction programmée après un nouveau « Start Lab » |
| `controle-apres-deploiement.sh` | AC7 à AC9 : TLS (301, TLS 1.0/1.1 refusés, 1.2 accepté, certificat, HSTS) ; X17 (balayage des adresses publiques du VPC depuis le poste ; API, base et ALB vus d'une sonde Lambda éphémère hors plage ; un seul load balancer, aucun Service `LoadBalancer`, NodePorts, règles et groupe de l'ALB, liste blanche XE4/ME8, sortie sans IMDS) ; X20 (valeurs des secrets de session cherchées dans les états, releases Helm, objets du cluster, journaux du poste et de la CI ; montages de Secrets par pod ; Trivy sur les images et l'IaC ; AMI recommandée). Verdict dans `.session/controle-<session>.json`, rapports masqués dans `docs/mesures/US-004/` |
| `alb.sh ouvrir\|fermer` | AC10 : ouverture à Internet refusée sans verdict vert pour la session et l'état courants ; contrôle rejoué une fois ouvert, fermeture immédiate s'il rougit |
| `down.sh` | Flux puis releases Helm, vidage des photos, secrets `collector/*` supprimés sans délai, sonde Lambda résiduelle supprimée, `destroy` avec reprise sur les interfaces, planifications supprimées, `status.sh --strict` (AC4), ligne de coût (G5) |
| `status.sh [--strict]` | AC5 : session, temps restant, coût estimé de la session et cumul, couche persistante, ressources, nœuds, pods et HPA, dernier verdict du contrôle, inventaire. `--strict` : inventaire strict et dérive de la couche persistante (blocage d'accès public, rétentions), code retour non nul sinon (AC4, MI8) |
| `inventaire.sh [--strict]` | Inventaire du compte indépendant de l'état, hors couche persistante ; moteur de `status.sh --strict` |
| `lib.sh` | Fonctions communes (profil, chemins Windows pour Terraform, `jq` sans retour chariot, masquage du compte et des adresses, contexte `kubectl` en liste blanche, journal de session) |

**`down.sh` doit se terminer avant « End Lab »** : lab arrêté, `LabRole` perd ses droits et la destruction programmée échoue (XP00 S8b). `up.sh` affiche l'heure limite de lancement de `down.sh`.

Traces : phases horodatées dans `docs/mesures/sessions.csv` (X4, X5), coût par session dans `docs/mesures/couts.csv` (G5, sans compte, adresse ni identifiant de session). Fichiers de session et journaux d'`apply` dans `tools/.session/`, copies d'état dans `infra/.etat-sauvegardes/` : ignorés par git.
