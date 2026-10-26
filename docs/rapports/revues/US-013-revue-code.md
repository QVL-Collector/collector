# Revue de code — US-013 Soumettre un article au contrôle automatique de complétude

| Champ | Valeur |
|---|---|
| US | [US-013](../../../backlog/US-013-soumission-controle-completude.md) |
| Branche | `feat/US-013-soumission-controle-completude` |
| Périmètre | `git diff origin/main...HEAD` (2 commits `cf8f8f9`, `270d51e` ; 52 fichiers, +3809 / -63) |
| Date | 2026-10-26 |
| Référentiel | AC de l'US, [D02](../../decisions/D02-decoupage-applicatif.md), [D05](../../decisions/D05-communication-interservice.md), [D07](../../decisions/D07-donnees.md), `contracts/messages/`, `contracts/api/place-de-marche/v1.openapi.yaml`, [rapport de tests](../tests/US-013-rapport-tests.md) |
| Verdict | **À corriger** : 0 bloquante, 1 majeure, 5 mineures, 2 suggestions |

## 1. Vérifications exécutées

| Commande | Résultat |
|---|---|
| `gofmt -l .` (`libs/platform`, `services/place-de-marche`, `services/controle`) | Aucun fichier |
| `go vet ./...` (trois modules) | Aucun constat |
| `golangci-lint run ./...` v2.15.0 (trois modules) | 0 constat |
| `go test -short ./...` (trois modules) | Vert (tests Docker sautés ; le bout en bout est rapporté vert par le rapport de tests) |
| Commentaires ajoutés (Go, SQL, Dockerfile) | Aucun ; seules les directives `-- +goose` |
| Mention d'outil d'IA, d'assistant, d'agent ou de flotte ; jeton JWT en clair | Aucune |

## 2. Points forts

- Transaction de soumission : `stockage/postgres.go:121-136` fait la garde (vendeur, révision, état), le passage `en_controle`, l'encodage validé contre le contrat et `outbox.Enqueue` dans une seule `pgx.BeginFunc` ; un échec d'encodage ou d'écriture annule l'état (AC5, testé par `ac5EchecDeLOutboxAnnuleLaSoumission`).
- Garde d'idempotence conforme à D05/D07 : `applyReviewQuery` (`postgres.go:795-798`) n'agit qu'en `en_controle` à la révision du résultat, sans table de messages traités ; `resultat_controle` n'est qu'un journal écrit après la garde. Doublon, révision antérieure et course entre réplicas (réévaluation du `WHERE` sous verrou de ligne) sont sans effet (AC6).
- Concurrence soumission/modification : les deux requêtes portent la révision attendue et l'incrémentent ; `en_controle` n'est ni modifiable ni soumettable. `explainRefusal` factorise proprement le 404/409 des deux chemins.
- Relais d'outbox (`libs/platform/outbox/relay.go`) : `FOR UPDATE SKIP LOCKED`, publication par lot avec confirmations différées, `mandatory` + retours, marquage dans la même transaction, pause sur `NotifyBlocked`, sortie sur fermeture du canal de publication. Toute panne produit au pire un doublon, absorbé par la garde.
- Consommateur (`libs/platform/broker/consumer.go`) : `user-id` vérifié avant tout décodage (AC7), acquittement manuel après l'effet, rejet sans remise en file pour un message non conforme, erreur transitoire en attente croissante sans `Nack` (risque n°1 de D05 traité), faute AMQP remontée.
- Briques `libs/platform/{broker,outbox}` techniques : aucune dépendance vers `services/` ni vers `contracts/`. Aucune déclaration de topologie (`Consume`/`Publish` seulement).
- Règle de complétude (`controle/internal/regles/regles.go:52-76`) : fonction pure, espaces seuls traités comme vides, catégorie inconnue en `a_verifier` jamais `publie` (AC4), `rules_version` porté par chaque résultat ; 100 % couverte.
- Arrêt propre par `errgroup.WithContext` dans les deux services ; `Supervisor.serve` rend `nil` sur annulation.
- Migration 00003 : contraintes `CHECK` qui lient motif et état, unicité `(article_id, revision)`, index partiel des lignes en attente ; droits du rôle d'exécution hérités des `DEFAULT PRIVILEGES` (sans `DELETE`), vérifiés par le test de droits minimaux.
- Dockerfile du Contrôle : images épinglées par empreinte, `distroless:nonroot`, `USER 65532`, `CGO_ENABLED=0`, `-mod=readonly`.

## 3. Issues

### Bloquantes

Aucune.

### Majeures

1. **`libs/platform/broker/consumer.go:44-53` et `:115` — la fermeture du canal de publication n'est pas surveillée par le consommateur du Contrôle.** Fait constaté : `WorkerWithPublisher` ouvre un `Publisher` mais la boucle (`loop`, `wait`) n'écoute que la fermeture du canal de consommation. Si le broker ferme le canal de publication au niveau canal (échange `controle.resultats` absent : 404 ; droit `write` refusé : 403), `amqp091-go` v1.16.0 solde les confirmations en attente à `false` (`confirms.go:199-207`) → `ErrRejected`, puis chaque `Publish` suivant renvoie `ErrClosed` (code 504). Aucune des deux n'est une faute pour `IsFault` : le handler réessaie indéfiniment (attente plafonnée à 10 s), la connexion n'est jamais rouverte, `/disponibilite` reste vert et aucune demande n'est plus traitée jusqu'au redémarrage du pod.
   Pourquoi : contredit le README du Contrôle (« un échange absent, ou un droit refusé, arrête le service en erreur ») et D05 (faute à signaler, pas à réessayer en boucle) ; le relais de la Place de marché, lui, traite le cas (`relay.go:65-66`). Blocage silencieux du walking skeleton sur une erreur de topologie ou de droits.
   Correction : faire écouter `publisher.Closed()` par la boucle du consommateur (par exemple passer à `loop`/`wait` un second canal de fermeture, ou combiner les deux) et rendre `CloseReason(reason)`, pour que le superviseur classe la faute (arrêt) ou la coupure (reconnexion). Ajouter un test de bout en bout : Contrôle démarré sans liaison d'écriture ou sans l'échange des résultats → le processus s'arrête en erreur.
   → back (correction), testeur (test de non-régression)

### Mineures

2. **`services/place-de-marche/internal/annonces/stockage/postgres.go:792`** — `member_since` est déduit de la date du premier article du vendeur, pas de l'ancienneté du compte. Sans effet en v1 (la règle ne lit pas ce champ), mais la valeur envoyée au Contrôle est sémantiquement fausse et alimentera la règle « statut du vendeur » (US-020). Correction : documenter l'approximation dans le README, ou la remplacer par la date du compte quand elle sera disponible (US-020). → back
3. **`libs/platform/broker/consumer.go:23-24,100`** — `Delivery.UserID` et `Delivery.Redelivered` sont renseignés mais lus par aucun handler (vérifié par recherche dans `services/` et `libs/`). Code mort : les retirer, ou les garder seulement si un usage est prévu dans le sprint. → back
4. **`libs/platform/outbox/relay.go:88-100` avec `publisher.go:84-86`** — un événement sans file de destination (`ErrUnroutable`) fait échouer tout le lot à chaque cycle : blocage en tête de l'outbox et un avertissement toutes les 500 ms, sans attente croissante. Cas d'une liaison absente côté IaC, donc rare ; prévoir au moins une attente croissante entre lots en échec (le `Backoff` existe déjà). → back
5. **US-013 « Mesures : MC2, MC6, MC9 »** — MC9 (rejeu de l'outbox par Job, fenêtre plafonnée) n'est pas livré et son report n'est écrit nulle part ; US-024 le reprend. Préciser dans l'US ou le README de la Place de marché que MC9 est reporté à US-024. → back
6. **`services/place-de-marche/README.md:51`** — `idle_in_transaction_session_timeout` est annoncé « à poser sur ce rôle », mais aucun script d'amorçage ne le pose dans le dépôt. À porter dans l'amorçage de la base `marche` (D07, MD1-MD2) au moment de l'IaC, et à tracer dans le suivi. → devops

### Suggestions

7. **`services/place-de-marche/cmd/place-de-marche/serve.go:71`** — la disponibilité de la Place de marché ignore le broker (choix documenté dans le README). `Supervisor.Ready` existe : le combiner à `pool.Ping` rendrait le pod non prêt pendant une coupure longue du canal, comme le Contrôle. À trancher, non bloquant. → back
8. **`services/place-de-marche/internal/annonces/controle/contract.go` et `services/controle/internal/canal/contract.go`** — les constantes de contrat (`controle.demande`, `controle.resultat`, clé `v1`, version 1) sont dupliquées dans les deux services. Les exposer depuis le module `contracts` (déjà importé par les deux) éviterait une dérive silencieuse. → back

## 4. Couverture des AC

AC1 à AC9 couverts côté back par le code et les tests (en mémoire, bout en bout RabbitMQ + PostgreSQL, contrat OpenAPI). AC10 (front, `aria-live`, axe-core) hors du diff : à livrer et tester côté front avant de clore l'US. Apparition au catalogue (AC1) : l'état `publie` est posé ; la lecture du catalogue relève de l'US qui l'expose (clarification déjà notée dans le rapport de tests).

## 5. Retours par fonction

- **Back** : issue 1 (majeure, à corriger avant merge), puis 2 à 5, suggestions 7 et 8.
- **Testeur** : test de non-régression de l'issue 1 (arrêt en erreur du Contrôle sur échange des résultats absent ou écriture refusée).
- **Devops** : issue 6.
