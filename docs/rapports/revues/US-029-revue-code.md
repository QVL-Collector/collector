# US-029 — Revue de code DevOps et tests

| Champ | Valeur |
|---|---|
| US | [US-029 — Aucune notification perdue quand Interactions est arrêté](../../../backlog/US-029-reprise-interactions.md) |
| Branche | `feat/US-029-reprise-interactions` (empilée sur `origin/feat/US-028-notification-vendeur`, HEAD `31ac1c0`) ; changements non commités revus par `git diff` et fichiers non suivis |
| Date | 2026-11-13 |
| Périmètre | `infra/canal/outils/abonnement-test.sh` (nouveau), `outils/lib.sh`, `outils/identifiants.sh`, `infra/canal/README.md` ; `services/place-de-marche/cmd/place-de-marche/reprise_interactions_test.go` (nouveau), `notification_vendeur_test.go` (banc découpé), `canal_test.go` (port de gestion exposé) ; [rapport de tests](../tests/US-029-rapport-tests.md) |
| Références | [D02](../../decisions/D02-decoupage-applicatif.md) (second abonné), [D05](../../decisions/D05-communication-interservice.md) (MC8, M-CAN-06), ES-31, ES-38 |
| Hors revue (connu) | Test « après déploiement » (mise à zéro du Deployment d'Interactions) ; mise à l'échelle (US-033) |
| Verdict | **Fusion après correction** (0 bloquante, 1 majeure, 3 mineures, 2 suggestions) |

## Vérifications exécutées

| Commande | Résultat |
|---|---|
| `gofmt -l .`, `go vet ./...` (`services/place-de-marche`) | 0 fichier, 0 remarque |
| `golangci-lint run --config ../../.golangci.yml ./...` (`services/place-de-marche`) | 0 issue |
| `go test ./cmd/place-de-marche/ -run 'TestUS029\|TestUS028' -count=1 -v` (Docker 28.5.1) | `ok` 87 s : `TestUS028…` PASS (41 s, banc découpé sans régression), `TestUS029…` PASS (46 s, 6 sous-tests) ; arrêt brutal après 1 notification sur 2 000, **0 relivrée** sur ce passage |
| `bash -n abonnement-test.sh` | Syntaxe valide |
| `KUBE_CONTEXT=arn:aws:eks:prod abonnement-test.sh poser kubernetes` ; action inconnue | Refus immédiat (garde-fou, puis usage), code 1, avant `charger_mdp`, `kubectl` ou `curl` |
| Appelants de `lancerInteractions` (supprimé), `demarrerBancNotification`, `journalInter` | Plus aucun appel à `lancerInteractions` ; `demarrerBancNotification` (`notification_vendeur_test.go:404`) recomposé sur `demarrerBancSansInteractions` ; `journalInter` toujours alimenté (`:220`, `reprise_interactions_test.go:143`) |
| `mdp_aleatoire` déplacé dans `lib.sh` | Seuls appelants : `identifiants.sh:48` (source `lib.sh` à la ligne 10) et `abonnement-test.sh:80` ; copie de `sandbox/XP02` hors périmètre |
| Exposition de `15672/tcp` dans `demarrerRabbitMQ` (`canal_test.go:155`) | Tous les bancs RabbitMQ du paquet en héritent ; attente inchangée sur `5672/tcp` ; suite US-028 verte |
| Comportement de `kill $!` sur une fonction lancée en arrière-plan (Git Bash, motif de `ouvrir_kubernetes` reproduit avec `sleep`) | Le sous-shell meurt, **le processus enfant survit** (PPID 1) ; voir majeure 1 |
| Secrets dans `abonnement-test.sh` | Mot de passe `admin-canal` et corps contenant le mot de passe de `abonne-test` passés par `curl -K -` (entrée standard, `printf` intégré) ; jamais en argument, jamais journalisés ; `.genere/` en `umask 077`, ignoré par git |
| `shellcheck` | Non disponible sur le poste de revue ; tracé sans remarque dans le README (0.11.0) |

## Points forts

- Couverture réelle des quatre AC sur le vrai système : AC1 par les vraies routes (contrôle et administrateur, publiés et refusés), catalogue et fiches vérifiés, file `interactions.publication-decidee` stable à 9 et zéro notification (`reprise_interactions_test.go:201-235`) ; AC2 exact par vendeur, par (article, révision, issue) et par motif (`:237-289`) ; AC3 avec corps identique **octet pour octet** à la ligne d'outbox, `user-id`, échange, clé, et rapprochement `message_id` côté Interactions (`:452-498`) ; AC4 au-delà de la publication : déclaration, liaison, suppression, échange par défaut, API de gestion (`:528-596`).
- Arrêt brutal en cours de consommation crédible : 2 000 décisions confirmées par le broker (`PublishWithDeferredConfirmWithContext` + `WaitContext`, `:314-350`), `Kill` dès la première notification, garde explicite que l'arrêt a bien eu lieu avant la fin (`:373-375`), retour des non acquittés attendu avant la relance (`:377-380`).
- Pas de faux vert sur les files : `exigerLaFileStable` (`:170-177`) attend la valeur puis la reconfirme 2 s plus tard ; toutes les attentes sont bornées (20 s, 90 s, 1 min, 10 s par appel HTTP).
- Découpage du banc d'US-028 propre et sans changement de scénario : `instanceInteractions` porte le cycle de vie, `arreterBrutalement` idempotent (`arretee`) et enregistré en `t.Cleanup` sur le test qui lance, `exigerEnVie` réutilisé par l'attente de disponibilité et par la boucle d'arrêt (`notification_vendeur_test.go:145-222`) ; `lancerInteractions` supprimé, pas de code mort laissé.
- Script d'exploitation sûr sur le fond : garde-fou `KUBE_CONTEXT` avant toute commande, identifiants par l'entrée standard de `curl`, chemins de retrait qui acceptent 404 (idempotence) puis **vérifient** l'absence de chaque élément y compris le mot de passe local (`abonnement-test.sh:105-118`), droits relus après la pose (`:86-87`), lecture en `ack_requeue_true` (non destructive).
- Factorisation au passage : `mdp_aleatoire` remonté dans `lib.sh` au lieu d'être recopié.
- Documentation à jour : README du canal (table des identités, usage, prérequis `jq`, trace de l'essai du 2026-11-13), cohérente avec D02 (§ second abonné) et D05 (M-CAN-06 : abonnement posé et retiré par l'exploitation).

## Issues

### Bloquantes

Aucune.

### Majeures

1. `infra/canal/outils/abonnement-test.sh:57-58` et `:36` — le port-forward n'est pas fermé. `kc -n … port-forward … &` lance la **fonction** `kc` dans un sous-shell : `$!` est le PID de ce sous-shell, pas celui de `kubectl`. `nettoyer` tue le sous-shell et `kubectl` survit, rattaché à PID 1 (reproduit sur le poste avec le même motif : enfant toujours vivant après `kill "$PF"`). Conséquences : 127.0.0.1:15673 reste ouvert vers l'API de gestion après le script ; à l'appel suivant, le nouveau port-forward échoue à se lier mais `ouvrir_kubernetes` reçoit 200 de l'**ancien**, et parle au cluster d'un `KUBE_CONTEXT` précédent : le garde-fou de contexte est contourné. Le mode `kubernetes` n'a d'ailleurs pas été joué (le README ne trace que docker compose). Correction : lancer le binaire directement, `kubectl --context "$KUBE_CONTEXT" -n "$NS_CANAL" port-forward svc/rabbitmq "$PORT_KUBE:15672" >/dev/null 2>&1 &`, pour que `$!` soit `kubectl` ; dans la boucle d'attente, exiger `kill -0 "$PF"` avant d'accepter le 200 ; refuser de démarrer si 15673 répond déjà. Même motif préexistant dans `verifier.sh:183` / `:123` : corriger dans le même commit. Jouer ensuite `poser`, `lire`, `retirer` en `kubernetes` sur k3d et vérifier qu'aucun `kubectl port-forward` ne reste, puis tracer l'essai dans le README. Dev : devops.

### Mineures

1. `infra/canal/outils/abonnement-test.sh` — `jq` n'est pas vérifié au démarrage. Sans lui, `poser` journalise des droits vides, `lire` s'arrête après avoir commencé, et surtout `retirer` (`:113`) compare une chaîne vide à `0` et annonce un faux « retrait incomplet : liaison ». Ajouter en tête, après le `case` des arguments, `command -v jq >/dev/null || echec "jq requis"` (idem `curl`). Dev : devops.
2. `services/place-de-marche/cmd/place-de-marche/reprise_interactions_test.go:500-507` — `instantDe` recopie `instant` du même paquet (`comptes_migration_test.go:35-42`, corps identique). Supprimer `instantDe` ; aux lignes 489-490, remplacer l'analyse manuelle de `n.DecideLe` par `if !instant(t, n.DecideLe).Equal(instant(t, donnees.DecidedAt))`. Dev : testeur.
3. `reprise_interactions_test.go:70` et `docs/rapports/tests/US-029-rapport-tests.md` (AC2, Observations) — le sous-test s'intitule « décisions relivrées sans perte ni doublon », mais la relivraison d'une décision déjà notifiée dépend de l'instant du `Kill` : 1 dans le passage du rapport, **0** dans le passage de revue (journal `:381` : « dont 0 déjà notifiée(s) »). Le test prouve toujours « sans perte » et le compte exact ; la déduplication sur relivraison est, elle, prouvée de façon déterministe par `services/interactions/cmd/interactions/integration_test.go:212-233` (US-028). Renommer en « décisions non acquittées rendues à la file, sans perte ni doublon », et écrire « 0 ou 1 relivraison selon l'instant de l'arrêt ; la déduplication est prouvée par le test d'intégration d'Interactions » dans le rapport. Dev : testeur.

### Suggestions

1. `abonnement-test.sh:72-73` (et `argumentsAbonnementDeTest`, `reprise_interactions_test.go:32-37`) — la file de test est en `reject-publish`. D'après D05 (§ files de travail, ligne 242), un refus nacke le producteur et laisse l'événement dans l'outbox : un abonnement de test **oublié** et plein (10 000 décisions en moins de 24 h) bloquerait la diffusion vers Interactions, l'inverse du but de l'US. Le risque est faible (durée de vie 24 h, `retirer` documenté), et la règle « jamais `drop-head` » de D05 vise les files de travail, pas une sonde sans consommateur. À trancher : passer la sonde en `drop-head` (README et constante du test alignés), ou garder `reject-publish` et l'écrire dans le README comme risque connu. Dev : devops (testeur pour la constante).
2. `reprise_interactions_test.go:437-449`, `:575-581`, `:613-639` — la lecture des liaisons de la file (`[]struct{ RoutingKey string }`) est déclarée deux fois, et le retrait parcourt trois listes avec trois structures anonymes et trois boucles identiques. Extraire `liaisonsDeLAbonnement(t) []string` et `exigerAbsentDeLaListe(t, chemin, champ, nom)`. Dev : testeur.

## Verdict

| Sévérité | Nombre |
|---|---|
| Bloquante | 0 |
| Majeure | 1 |
| Mineure | 3 |
| Suggestion | 2 |

**Fusion après correction de la majeure 1.** Les AC1 à AC4 sont réellement couverts par un test de bout en bout vert et rejoué en revue, sans régression sur US-028, avec des attentes bornées et un nettoyage des processus correct côté Go. Aucun secret en argument ni en journal dans le script, `retirer` est idempotent et vérifie ce qu'il annonce. Seule la fermeture du port-forward du mode `kubernetes` est fausse ; elle laisse un tunnel ouvert et peut contourner le garde-fou de contexte. Les mineures se règlent dans le même passage (devops : 1, testeur : 2 et 3).

## Retours traités (2026-11-13)

| Retour | Traitement |
|---|---|
| Majeure 1 — port-forward non fermé | `kubectl` lancé directement (`$!` exact), `kill -0` dans l'attente, refus si 15673 répond déjà ; même correction dans `verifier.sh`. Joué sur k3d : poser, lire, refus, retirer ×2, aucun `kubectl` restant, 15673 fermé ; `verifier.sh kubernetes` 79/79 |
| Mineure 1 — `jq` non vérifié | `curl` et `jq` exigés au démarrage |
| Mineure 2 — `instantDe` dupliqué | Supprimé, `instant` réutilisé |
| Mineure 3 — « relivrées » | Sous-test renommé, rapport : 0 ou 1 relivraison |
| Suggestion 1 — `reject-publish` | File de test en `drop-head` : un abonnement oublié ne bloque jamais l'outbox (D05) |
| Suggestion 2 — retrait répétitif | Factorisé (`clesDesLiaisonsDeLAbonnementDeTest`, `exigerAbsentDeLaListe`) |

Verdict après corrections : **fusion possible**.
