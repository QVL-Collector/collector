# Rapport de tests — US-013 Soumettre un article au contrôle automatique de complétude

| Champ | Valeur |
|---|---|
| US | [US-013](../../../backlog/US-013-soumission-controle-completude.md) |
| Branche | `feat/US-013-soumission-controle-completude` |
| Date | 2026-10-26 |
| Fonction | Test |
| Périmètre | Back : AC1 à AC9 (Place de marché, Contrôle, `libs/platform`). AC10 (accessibilité, front) hors périmètre de cette campagne |
| Référentiel | AC de l'US, READMEs de `services/place-de-marche` et `services/controle`, [contracts/README.md](../../../contracts/README.md), [D05](../../decisions/D05-communication-interservice.md) |
| Verdict | **Conforme** : tous les AC back sont couverts et conformes ; aucune anomalie. Une clarification (catalogue, AC1) |

## 1. Livrables de test

| Fichier | Contenu |
|---|---|
| `services/place-de-marche/cmd/place-de-marche/memoire_test.go` | Stockage en mémoire étendu (`Submit`, `ApplyReview`) et relevé des demandes émises |
| `services/place-de-marche/cmd/place-de-marche/environnement_test.go` | Environnement HTTP adressé par URL (réutilisé en mémoire et de bout en bout), champ `motif` lu |
| `services/place-de-marche/cmd/place-de-marche/integration_test.go` | `NewStore(pool, encodeur)` ; droits minimaux étendus à `outbox` et `resultat_controle` (aucun `DELETE`) |
| `services/place-de-marche/cmd/place-de-marche/validateur_test.go` | Validateur des contrats de messages |
| `services/place-de-marche/cmd/place-de-marche/canal_fictif_test.go` | Faux canal : demande encodée par la Place de marché, décodée contre le contrat v1, verdict de substitution, résultat livré au consommateur réel |
| `services/place-de-marche/cmd/place-de-marche/soumission_test.go` | Tests d'acceptation sans Docker `TestUS013AC…` |
| `services/place-de-marche/cmd/place-de-marche/canal_test.go` | RabbitMQ par testcontainers-go, identités générées à l'exécution, file espion des résultats, publication confirmée |
| `services/place-de-marche/cmd/place-de-marche/services_test.go` | Lancement de la Place de marché (`serve`, en processus) et du Contrôle (binaire construit par `go install`) |
| `services/place-de-marche/cmd/place-de-marche/bout_en_bout_test.go` | `TestUS013BoutEnBoutPlaceDeMarcheEtControle` (15 scénarios ordonnés) |
| `services/place-de-marche/cmd/place-de-marche/testdata/canal/` | Copie de la topologie IaC (`conf.d/10-collector.conf` sans ses commentaires, `enabled_plugins`, `definitions/10-topologie.json`) |
| `services/controle/internal/regles/regles_test.go` | Règle de complétude v1 et catégorie inconnue |
| `services/place-de-marche/go.mod` | `amqp091-go` passe en dépendance directe (`go mod tidy`) |

**Topologie du broker** : recopiée de `infra/canal/broker/` (US-011) ; elle doit rester alignée sur `infra/canal`. Le test remplace le `10-defaults.conf` de l'image par un fichier vide (montage en remplacement complet comme l'IaC) et génère `00-utilisateurs.json` à chaque exécution pour les cinq identités citées par les droits (`marche`, `controle`, `interactions`, `keda`, `admin-canal`), empreintes `rabbit_password_hashing_sha256`, mots de passe aléatoires jamais écrits dans le dépôt. Images épinglées par empreinte : `postgres:18.3@sha256:7e32…84db`, `rabbitmq:4.3.7-management@sha256:79fb…d01a`. Seul ajout hors IaC : une file espion `test.espion-resultats-controle` liée à `controle.resultats` (clé `v1`) par `admin-canal`, pour capturer les résultats du Contrôle.

## 2. Exécutions

| Module | `go test ./... -count=1 -cover` | `golangci-lint run` (v2.15.0, `.golangci.yml`) |
|---|---|---|
| `libs/platform` | Vert ; `message` 82,8 %, `token` 85,9 %, `logging` 87,5 %, `httpjson` 76,7 % (`broker`, `outbox` sans test propre, couverts par le bout en bout) | 0 constat |
| `services/place-de-marche` | Vert (US-009 + US-013, dont 2 tests Docker), 74 s | 0 constat |
| `services/controle` | Vert ; `regles` 100 % | 0 constat |

Tests US-013 : 48 tests et sous-tests en mémoire, 15 scénarios de bout en bout, 27 tests et sous-tests de la règle. Tests US-009 : tous verts après réparation.

Couverture de la Place de marché mesurée avec `-coverpkg=./...,collector/libs/platform/broker,collector/libs/platform/outbox` : **77,3 %** au total ; `annonces` 92,5 %, `stockage` 88,4 %, `annonces/api` 83,8 %, `annonces/controle` 85,0 %, `migrations` 78,6 %, `config` 70,7 %, `libs/platform/outbox` 76,1 %, `libs/platform/broker` 63,0 %. Le Contrôle tourne en processus séparé : sa couverture n'est pas mesurée par le bout en bout.

Sans Docker ou avec `-short`, les tests d'intégration sont sautés explicitement (`testing.Short`, `SkipIfProviderIsNotHealthy`). Conteneurs supprimés en fin de test (`CleanupContainer`), aucun conteneur résiduel constaté.

## 3. Réparation des tests US-009

| Cassure | Réparation | Ce qui reste prouvé |
|---|---|---|
| `stockageEnMemoire` n'implémente plus `annonces.Articles` | Ajout de `Submit` (404 si autre vendeur, 409 si non soumettable ou révision périmée, `en_controle`, révision + 1, motif effacé) et `ApplyReview` (appliqué seulement en `en_controle` à la révision du résultat) | Tous les scénarios US-009 inchangés |
| `stockage.NewStore(pool)` | `NewStore(pool, controle.NewRequestEncoder(validateur))` | Scénarios US-009 sur PostgreSQL, contrat OpenAPI |
| Droits minimaux : deux tables nouvelles | `outbox` et `resultat_controle` ajoutées à la liste des tables du service ; `DELETE` refusé sur les deux en plus | Renforcé : aucune table hors service, aucun objet possédé, aucun DDL |
| Champ `motif` du contrat OpenAPI | Lu dans `articleRecu` ; les réponses restent validées contre le contrat | Contrat OpenAPI, y compris `POST …/soumission` (202, 400, 404, 409) |

## 4. Résultats par AC

### AC1 — Brouillon complet : en contrôle, puis publié en moins de 10 s

| Scénario | Test | Résultat |
|---|---|---|
| Given un brouillon complet When soumis Then 202, `en_controle`, révision + 1, sans motif ; puis `publie` dans la fiche et « Mes articles » | `TestUS013AC1BrouillonCompletEnControlePuisPublie` | Conforme |
| Given les vrais services When soumis Then `en_controle` dans « Mes articles » puis `publie` mesuré sous 10 s (≈ 0,2 s observé) | bout en bout « AC1 … moins de 10 secondes » | Conforme |
| Given chaque catégorie proposée par la Place de marché When brouillon complet soumis Then publié | bout en bout « AC1 … chaque catégorie » | Conforme (7/7) |
| Given une demande publiée Contrôle arrêté When le Contrôle démarre Then l'article est publié | bout en bout « AC1 demande en attente » | Conforme |
| Limites : sans frais de port reste complet ; révision périmée, future, absente, nulle, en texte, dupliquée, champ inconnu → 409 / 400 sans effet ni demande | `…SansFraisDePortResteComplet`, `…SoumissionAvecRevisionPerimeeRefusee`, `…SoumissionRefuseeSansEffet` | Conforme |
| Erreurs : article `en_controle` ou `publie` re-soumis → 409 ; `en_controle` non modifiable → 409 | `…ArticleEnControleOuPublieNonSoumettable`, `…ArticleEnControleNonModifiable` | Conforme |
| Soumission sans jeton → 401, jeton acheteur → 403 ; réponses conformes au contrat OpenAPI | `TestUS013SoumissionReserveeAuVendeurAuthentifie`, `TestUS013ReponsesDeSoumissionConformesAuContratOpenAPI` | Conforme |

**Clarification C1** : « reste invisible au catalogue / apparaît au catalogue » n'est pas testable côté back, aucune route de catalogue n'existant encore. Seul l'état est vérifié (`en_controle`, puis `publie`). Le contrôle de visibilité est à reprendre avec l'US du catalogue ou en E2E.

### AC2 — Brouillon sans description : refusé, motif « annonce incomplète »

| Scénario | Test | Résultat |
|---|---|---|
| Sans description, description en espaces, sans titre, sans catégorie, sans prix, brouillon vide → `refuse` + `annonce_incomplete` dans la fiche et « Mes articles » | `TestUS013AC2BrouillonIncompletRefuseAvecMotif` (6 cas) | Conforme |
| Vrais services : sans description → `refuse` + `annonce_incomplete` dans « Mes articles » | bout en bout « AC2 » | Conforme |
| Règle du Contrôle : 8 variantes d'incomplétude → `refuse` / `annonce_incomplete` | `TestUS013AC2AnnonceIncompleteRefusee` | Conforme |

### AC3 — Article refusé corrigé et resoumis : en contrôle puis publié

| Scénario | Test | Résultat |
|---|---|---|
| Refusé → modifié (révision + 1) → resoumis : 202 `en_controle`, motif effacé → `publie` | `TestUS013AC3ArticleRefuseCorrigeEtResoumisPuisPublie` et bout en bout « AC3 » | Conforme |
| Refusé resoumis sans correction → repasse en contrôle puis de nouveau refusé | `TestUS013AC3ArticleRefuseResoumisSansCorrectionRepasseEnControle` | Conforme |

### AC4 — Catégorie inconnue du Contrôle : à vérifier, jamais publié

| Scénario | Test | Résultat |
|---|---|---|
| Vrais services : catégorie `timbres` ajoutée à la Place de marché seulement → `a_verifier` + `categorie_inconnue`, toujours ainsi 2 s plus tard | bout en bout « AC4 » | Conforme |
| Place de marché : issue `a_verifier` appliquée et visible dans « Mes articles » ; article `a_verifier` non re-soumettable par le vendeur (409) | `TestUS013AC4CategorieInconnueDuControleAVerifierJamaisPublie` | Conforme |
| Règle : codes hors référentiel (dont casse différente), référentiel vide, référentiel fourni seul connu, incomplète + inconnue → jamais `publie` | `TestUS013AC4…` (4 tests) | Conforme |

### AC5 — Outbox dans la même transaction, publication, règles v1

| Scénario | Test | Résultat |
|---|---|---|
| Given une soumission Then une ligne d'outbox `controle.demandes` / `v1` / `controle.demande`, conforme au contrat, pour l'article et sa nouvelle révision, puis `publie_le` renseigné et message dans la file `controle.demandes` | bout en bout « AC5 demande écrite … » | Conforme |
| Given l'écriture dans l'outbox en échec (déclencheur posé en test) When soumis Then erreur 5xx, article toujours `brouillon` à la même révision, aucune ligne d'outbox | bout en bout « AC5 échec d'écriture … » | Conforme : état et demande sont atomiques |
| Chaque résultat du Contrôle porte `rules_version` 1 (message capturé et ligne `resultat_controle`) ; corrélation de la demande reprise | bout en bout « AC5 AC8 … » ; `TestUS013AC5VersionDesReglesEgaleA1` | Conforme |
| La demande reprend article, révision, titre, catégorie et prix soumis | `TestUS013AC5DemandePorteLArticleSaRevisionEtSonContenu` | Conforme |

### AC6 — Résultat livré deux fois ou de révision antérieure : sans effet

| Scénario | Test | Résultat |
|---|---|---|
| Même résultat livré deux fois → acquitté, fiche identique | `TestUS013AC6ResultatLivreDeuxFoisSansEffet` | Conforme |
| Résultat d'une révision antérieure (réel puis forgé) pendant le contrôle de la révision courante → sans effet ; le résultat courant s'applique | `TestUS013AC6ResultatDUneRevisionAnterieureSansEffet` | Conforme |
| Résultat pour un article inconnu → acquitté sans effet | `TestUS013AC6ResultatPourUnArticleInconnuSansEffet` | Conforme |
| Vrais services : message du Contrôle rejoué à l'identique (même `message-id`, identité `controle`) et ancien résultat `refuse` rejoué sur un article publié → états, révisions et `modifieLe` inchangés, une seule ligne `resultat_controle`, rien d'écarté | bout en bout « AC6 » | Conforme |

### AC7 — Résultat publié par une autre identité : écarté et journalisé

| Scénario | Test | Résultat |
|---|---|---|
| `admin-canal` publie sur `controle.resultats` un résultat conforme sous son identité → mis à l'écart dans `pdm.resultats-controle.ecartes`, identifiant du message journalisé, article toujours `en_controle` | bout en bout « AC7 résultat publié par une autre identité » | Conforme |
| `admin-canal` publie avec `user-id` = `controle` → canal fermé par le broker (406), rien n'arrive | bout en bout « AC7 user-id controle usurpé » | Conforme |
| Côté Contrôle : demande publiée par `admin-canal` → mise à l'écart dans `controle.demandes.ecartes`, aucun résultat produit ; le Contrôle ne journalise pas le titre en clair | bout en bout « AC7 AC8 demandes … » et « AC1 demande en attente » | Conforme |

### AC8 — Contrat v1 respecté par la paire

| Scénario | Test | Résultat |
|---|---|---|
| Producteur Place de marché : chaque demande encodée valide le schéma `controle.demande` v1 (mémoire) et la ligne d'outbox réelle aussi | `canal_fictif_test.go` (toutes les soumissions), bout en bout « AC5 » | Conforme |
| Consommateur Place de marché : 3 exemples valides acquittés, 13 invalides rejetés et journalisés | `TestUS013AC8ConsommateurDeResultatsRespecteLeContratV1` | Conforme |
| Consommateur réel : résultat `publie` avec motif, JSON tronqué, version 2 → écartés, journalisés, sans effet | bout en bout « AC8 » ; `TestUS013AC8ResultatNonConformeEcarteSansEffet` | Conforme |
| Producteur Contrôle : chaque résultat capturé valide `controle.resultat` v1, `user-id` `controle`, clé `v1` | bout en bout « AC5 AC8 … » | Conforme |
| Consommateur Contrôle : demande `{}` publiée par `marche` → écartée, sans résultat | bout en bout « AC7 AC8 demandes … » | Conforme |

### AC9 — Article du vendeur A soumis par le vendeur B : 404

| Scénario | Test | Résultat |
|---|---|---|
| B soumet l'article de A → 404 ; article inexistant → 404 ; article de A intact, aucune demande | `TestUS013AC9SoumissionDeLArticleDUnAutreVendeurIntrouvable`, bout en bout « AC9 » | Conforme |
| Réponses indiscernables (type, titre, détail) entre article d'autrui et inexistant | `TestUS013AC9ReponseIdentiquePourArticleDAutruiEtInexistant` | Conforme |

### AC10 — Accessibilité

Hors périmètre (front).

## 5. Anomalies

Aucune.

## 6. Points d'attention (non bloquants, pas des anomalies)

- **C1** : visibilité au catalogue (AC1) non vérifiable côté back tant qu'il n'existe pas de route de catalogue.
- La copie de la topologie dans `testdata/canal/` doit rester alignée sur `infra/canal/broker/` (US-011).
- `libs/platform/broker` et `outbox` n'ont pas de tests propres dans leur module : ils sont couverts par le bout en bout de la Place de marché (63 % et 76 %).
- Le bout en bout construit le Contrôle avec `go install` (toolchain Go requise sur l'exécuteur CI, déjà présente).

## 7. Conclusion

Tous les AC back (AC1 à AC9) sont couverts et conformes, en mémoire et de bout en bout avec PostgreSQL, RabbitMQ et les deux services réels. Les tests US-009 sont réparés sans perte de ce qu'ils prouvent.
