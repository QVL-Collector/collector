# US-005 — Garde-fous de coût : destruction programmée et journal des coûts

| Champ | Valeur |
|---|---|
| Epic | E2 Plateforme AWS |
| Statut | En cours |
| Sprint | 1 (J4) |
| Engagement | Ferme |
| Estimation | 2 points |
| Dépend de | US-004 |
| Fonction(s) | DevOps |
| Service | Plateforme |
| Critères grille | C5 |
| Exigences | ENF-16, ENF-28 |
| Branche | `feat/US-005-garde-fous-cout` |

## Récit

En tant que Lead Developer, je veux que l'environnement se détruise de lui-même avant la fin de la session et que chaque session laisse une trace de son coût, afin qu'un oubli ne consomme pas le budget de 50 $ et que le POC reste démontrable jusqu'à la soutenance.

## Critères d'acceptation

### Fonctionnels

- **AC1** — Given un `up` dont l'`apply` a réussi, When le script se poursuit, Then la destruction programmée est armée immédiatement, même si la suite échoue : node group, RDS, ALB et NAT 30 minutes avant la fin de session ; cluster et adresse 10 minutes avant (G1).
- **AC2** — Given une destruction programmée armée, When la fin de session approche sans `down`, Then les ressources facturables sont réellement supprimées avant l'expiration des identifiants (X15), et l'inventaire strict suivant le confirme.
- **AC3** — Given un nouveau « Start Lab » pendant une session, When `tools/up.sh --prolonger` s'exécute, Then les planifications sont repoussées.
- **AC4** — Given un `down` réussi, When il se termine, Then les planifications restantes sont supprimées.
- **AC5** — Given chaque session, When `down` se termine, Then une ligne (date, durée allumée, coût estimé) est ajoutée à `docs/mesures/couts.csv` et le cumul du projet est affiché.
- **AC6** — Given un cumul estimé supérieur ou égal à 70 % du budget réel (35 $), When `tools/up.sh` s'exécute sans l'option explicite de répétition de soutenance, Then il refuse de démarrer (G5).
- **AC7** — Given un `up` réussi, When il se termine, Then un rappel est programmé sur le poste 45 minutes avant la fin de session (G7).

### Non fonctionnels

- **AC8 (Sécurité)** — Given le journal des coûts, When il est relu avant commit, Then il ne contient ni identifiant de compte, ni adresse IP, ni identifiant de session.

## Sécurité

| Exigence | Comment l'US la sert | AC |
|---|---|---|
| ES-29 | Alerte de coût couverte par le journal et le seuil d'arrêt (G5) | AC5, AC6 |

Mesures : MS10, MI5.

## Tests attendus

| Type | Outil | Ce qui est prouvé | Fonction |
|---|---|---|---|
| Après déploiement | Inventaire strict après déclenchement de G1 sur une session courte | AC1, AC2, AC4 | DevOps |
| Unitaire (script) | Jeu de journaux fictifs pour le calcul du cumul et du seuil | AC5, AC6 | DevOps |
| Contrôle manuel tracé | Relecture du journal avant commit | AC8 | Test |

## Hors périmètre

- Alerte AWS Budgets (G6) : seulement si la sonde XP00 l'a montrée créable.

## Décisions liées

- [D11 — IaC, allumage et extinction, garde-fou coût](../docs/decisions/D11-iac-allumage-cout.md) (G1, G5, G7 ; X15)

## Definition of Done

Reprise de [PILOTAGE.md](../docs/PILOTAGE.md) § 5, cochée au fil du circuit.

- [ ] Tous les AC sont couverts par des tests verts en CI (`ci-ok`).
- [ ] Revue de code faite, retours traités.
- [ ] Martin a testé et prononcé « je valide la livraison de la feature ».
- [ ] Documentation touchée à jour.
- [ ] Statut, date de validation et liens des rapports renseignés ci-dessous.
- [ ] Aucune donnée sensible ni credential ajouté au dépôt.

## Suivi

| Étape | Lien / date |
|---|---|
| Rapport de test | |
| Rapport de revue de code | |
| Validation Martin | AAAA-MM-JJ |
| Livraison (PR / merge) | |

### Avancement au 2026-11-20 (sans action AWS)

Une grande partie venait d'US-004 (`up.sh`, `down.sh`, `lib.sh`). Écart AC par AC et compléments sur la branche :

| AC | Déjà couvert | Complété |
|---|---|---|
| AC1 | Armement juste après l'`apply`, même en échec (`set +e` autour de l'`apply`) | Piège `EXIT` (Ctrl-C, `TERM`, terminal fermé) qui arme sur ce qui a été créé ; cibles lues dans l'état (`terraform show -json`) et non plus dans les sorties, absentes après un `apply` partiel ; refus si T-30 est à moins de 2 min |
| AC2 | Six planifications (node group, RDS, ALB, NAT à T-30 ; cluster, adresse à T-10), champs en casse Scheduler (session 1 d'XP01) | Cibles par adresse Terraform exacte (MI5) ; clé KMS, bucket et reste de la couche persistante jamais ciblés (testé) ; ressources restantes non facturables listées dans `tools/README.md` |
| AC3 | `up.sh --prolonger` réarme G1 sur la nouvelle fin | Rappel G7 reprogrammé aussi |
| AC4 | `down.sh` supprimait le groupe sans vérifier | `supprimer_planifications` : suppression puis attente de la disparition du groupe (évite un groupe « DELETING » qui ferait échouer l'inventaire strict) ; rappel du poste annulé une fois l'extinction réussie |
| AC5 | Ligne `date;duree_allumee_min;cout_estime_usd` et cumul affiché par `down.sh` | Écriture factorisée (`ajouter_cout`), ajout sûr après une dernière ligne sans fin de ligne |
| AC6 | Refus à 70 % sauf `--repetition-soutenance` | Contrôle déplacé avant tout appel AWS ; décision factorisée (`autoriser_allumage_g5`) et testée (34,99 $ autorisé, 35,00 $ refusé, répétition autorisée, CRLF) |
| AC7 | Rappel seulement affiché dans le terminal | Rappel programmé 45 min avant la fin, sans privilège : tâche planifiée de l'utilisateur sous Windows (boîte de dialogue), `at` + `notify-send` ou `wall` sous Linux ; armé avec G1, donc même si la suite du `up` échoue |
| AC8 | Format sans compte, adresse ni session | Relecture automatisée du `couts.csv` du dépôt dans le test |

Vérifications hors AWS : `bash -n` ; `shellcheck --severity=warning` (image épinglée de `tools/ci/outils.yml`) vert sur `tools/*.sh` et `tools/tests/*.sh` ; `actionlint` vert ; `bash tools/tests/garde-fous-cout.sh` (86 cas après la revue de code, dont l'armement et le piège de sortie avec Scheduler et Terraform simulés) et `bash tools/tests/sortie-sans-imds.sh` verts, désormais joués en CI (job « Workflows et scripts ») ; script PowerShell du rappel analysé sans erreur par le parseur PowerShell (sans enregistrement de tâche).

Reste à prouver en session AWS (X15) :

- **AC1, AC2** : session courte avec `--fin-lab` proche (≈ 80 min), sans `down` : planifications créées sous `LabRole` (`iam:PassRole` vers Scheduler), exécution à T-30 puis T-10, puis `status.sh --strict` (ressources facturables parties ; reste VPC, groupes de sécurité, journaux) ; vérifier la casse acceptée par Scheduler (`DbInstanceIdentifier`, `elasticloadbalancingv2`) et que le cluster se supprime bien après le node group (sinon avancer T-30 de 5 min, D11). Un `down.sh` termine la session avant « End Lab ».
- **AC1 (interruption)** : Ctrl-C pendant l'`apply` : le piège arme les planifications sur les ressources déjà créées.
- **AC3** : `up.sh --prolonger --fin-lab HH:MM` après un nouveau « Start Lab » : `aws scheduler list-schedules --group-name collector` montre les nouvelles échéances.
- **AC4** : après `down.sh`, aucune planification ni groupe (`inventaire.sh --strict`) et tâche `Collector-rappel-fin-lab` absente.
- **AC7** : tâche planifiée créée sans élévation sur le poste et boîte de dialogue affichée à l'heure (`Get-ScheduledTask Collector-rappel-fin-lab`).
- **AC5, AC8** : ligne réelle ajoutée au `down.sh`, relue avant commit (contrôle manuel tracé).
