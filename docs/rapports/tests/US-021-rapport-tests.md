# Rapport de tests — US-021 Recontrôler toute modification d'un article

| Champ | Valeur |
|---|---|
| US | [US-021](../../../backlog/US-021-revalidation-apres-modification.md) |
| Branche | `feat/US-021-revalidation-apres-modification` |
| Date | 2026-10-28 |
| Fonction | Test |
| Périmètre | Back : AC1, AC2, AC3, AC5 (Place de marché, en mémoire et sur PostgreSQL + RabbitMQ réels). Front : AC4 et volet écran de AC1/AC2 (`/vendeur/articles/:id`), axe-core |
| Référentiel | AC de l'US, contrat `PUT /marche/v1/vendeur/articles/{id}` (publié, en contrôle, à vérifier → 200, révision + 1, `en_controle`, motif nul, demande en outbox ; brouillon et refusé inchangés) |
| Verdict | **Conforme** : tous les AC couvrables ici sont couverts et conformes ; aucune anomalie. Volet Playwright E2E non couvert (voir § 6) |

## 1. Livrables de test

| Fichier | Contenu |
|---|---|
| `services/place-de-marche/cmd/place-de-marche/revalidation_test.go` | Tests d'acceptation en mémoire `TestUS021…` (11 tests, 30 avec les sous-tests) |
| `services/place-de-marche/cmd/place-de-marche/revalidation_bout_en_bout_test.go` | `TestUS021BoutEnBoutRevalidationSurPostgreSQLEtRabbitMQ` : PostgreSQL et RabbitMQ par testcontainers-go, résultats publiés sur le vrai broker sous l'identité `controle` (5 scénarios ordonnés) |
| `services/place-de-marche/cmd/place-de-marche/memoire_test.go` | Faux stockage aligné : modification d'un `publie`, `en_controle` ou `a_verifier` → `en_controle`, révision + 1, motif effacé, demande enregistrée ; brouillon et refusé inchangés ; tout autre état non modifiable |
| `services/place-de-marche/cmd/place-de-marche/soumission_test.go` | `TestUS013AC1ArticleEnControleNonModifiable` retiré (voir § 3) |
| `web/src/acceptance/US-013/sellerMarketplaceHarness.tsx` | Place de marché simulée alignée sur le même contrat de modification |
| `web/src/acceptance/US-021/revalidation-apres-modification.test.tsx` | AC4, AC1/AC2 côté écran, axe-core (11 tests) |

## 2. Exécutions

| Commande | Résultat |
|---|---|
| `services/place-de-marche` : `go test ./...` (Docker, sans `-race`) | Vert, 95 s (US-009, US-013, US-023, US-021 dont le bout en bout US-021 en 14 s) |
| `services/place-de-marche` : `go vet ./cmd/...`, `gofmt -l` | 0 constat |
| `web` : `npx vitest run --pool=forks --poolOptions.forks.maxForks=2` | Vert : 29 fichiers, 314 tests (dont 11 US-021 et tous les US-013 avec la Place de marché simulée réalignée) |
| `web` : `npm run -s typecheck` | 0 erreur |
| `web` : `npx eslint src` | 0 constat |
| `web` : `npx prettier --check src` | Conforme |

Le message jsdom « HTMLCanvasElement.prototype.getContext » affiché pendant les passages axe-core est un bruit d'environnement (jsdom sans canvas), sans effet sur le résultat.

## 3. Réparation du test US-013

`TestUS013AC1ArticleEnControleNonModifiable` attendait 409 sur la modification d'un article en contrôle ; il recevait 200 (`en_controle`, révision 3). C'est le changement voulu par US-021 AC2 : le test est retiré et la règle est couverte par `TestUS021AC2ModificationEnControleOuAVerifierSoumetLaNouvelleRevision`. Ce que US-013 teste encore de valable reste en place et vert : soumission refusée hors brouillon/refusé (`TestUS013AC1ArticleEnControleOuPublieNonSoumettable`, `TestUS013AC4…` pour `a_verifier`).

## 4. Résultats par AC

### AC1 — Article publié modifié, même du seul titre : nouvelle révision, retiré du catalogue, en contrôle, sans seuil

| Scénario | Test | Résultat |
|---|---|---|
| Given un publié présent au catalogue When seul le titre change Then 200, `en_controle`, révision + 1, motif nul, titre modifié ; relu et « Mes articles » en contrôle ; absent de `GET /marche/v1/articles`, fiche publique 404 ; une demande pour la nouvelle révision avec le nouveau titre | `TestUS021AC1SeulTitreModifieSurUnPublieRetireDuCatalogueEtRecontrole` | Conforme |
| Sans seuil : titre, prix + 1 centime, frais de port + 1 centime, frais supprimés, catégorie, description, contenu identique → même effet | `TestUS021AC1ToutChampModifieSansSeuilRecontrole` (7 cas) | Conforme |
| Vrais services : même scénario sur PostgreSQL ; 2 lignes d'outbox pour l'article, la dernière conforme au contrat v1 en révision + 1 avec le nouveau titre, puis publiée sur le canal | bout en bout « AC1 » | Conforme |
| Erreurs : révision périmée ou future (409), prix négatif, catégorie hors jeu (400) → article inchangé, toujours au catalogue, aucune demande | `TestUS021AC1ModificationRefuseeDUnPublieSansEffet` (4 cas) | Conforme |
| Erreur : publié d'un autre vendeur → 404, toujours publié et au catalogue, aucune demande | `TestUS021AC1PublieDUnAutreVendeurNonModifiable` | Conforme |
| Non-régression : brouillon et refusé modifiés → même état, révision + 1, aucune demande | `TestUS021BrouillonEtRefuseModifiesSansControle` | Conforme |
| Réponses de modification conformes au contrat OpenAPI (200 sur publié, 409 périmée) | `TestUS021ReponsesDeModificationConformesAuContratOpenAPI` | Conforme |
| Écran : publié, titre modifié, « Enregistrer et renvoyer au contrôle » → `PUT` avec la révision lue et le titre, aucune soumission séparée, état affiché et annoncé « En contrôle », « Publié » disparu | front « US-021 AC1 AC2 … article publié » | Conforme |

### AC2 — Article en contrôle ou à vérifier modifié : nouvelle révision soumise, ancienne révision sans effet

| Scénario | Test | Résultat |
|---|---|---|
| `en_controle` et `a_verifier` modifiés → 200, `en_controle`, révision + 1, motif effacé, une demande pour la nouvelle révision, hors catalogue | `TestUS021AC2ModificationEnControleOuAVerifierSoumetLaNouvelleRevision` (2 cas) | Conforme |
| Résultat `publie`, `refuse` ou `a_verifier` sur l'ancienne révision → acquitté, article strictement identique, toujours hors catalogue | `TestUS021AC2ResultatSurLAncienneRevisionSansEffet` (6 cas) | Conforme |
| Article soumis puis modifié avant le résultat : résultat `publie` de la révision soumise sans effet ; le résultat de la nouvelle révision s'applique | `TestUS021AC2ResultatSurLaRevisionSoumiseAvantModificationSansEffet` | Conforme |
| Vrais services, `a_verifier` : modifié → nouvelle ligne d'outbox (révision + 1) publiée ; ancien résultat rejoué (même `message-id`) et résultat `publie` forgé sur l'ancienne révision → état, révision, titre et `modifieLe` inchangés, hors catalogue | bout en bout « AC2 article à vérifier » | Conforme |
| Vrais services, `en_controle` : idem avec un résultat `publie` sur la révision soumise | bout en bout « AC2 article en contrôle » | Conforme |
| Écran : `en_controle` et `a_verifier` modifiés et enregistrés → état « En contrôle », motif « Catégorie inconnue » disparu | front « US-021 AC1 AC2 … » (2 cas) | Conforme |

Pas de route de « décision » humaine dans le périmètre actuel : « décision portant sur l'ancienne révision » est couvert à travers les résultats de contrôle.

### AC3 — Nouvelle révision acceptée : retour au catalogue avec le contenu modifié

| Scénario | Test | Résultat |
|---|---|---|
| Depuis `publie`, `en_controle`, `a_verifier` : modification (titre et prix) puis résultat de la nouvelle révision → `publie` à la nouvelle révision ; entrée du catalogue et fiche publique avec le titre et le prix modifiés | `TestUS021AC3NouvelleRevisionAccepteeRevientAuCatalogueModifiee` (3 cas) | Conforme |
| Vrais services : résultat `publie` de la nouvelle révision publié sur le canal → `publie`, révision + 1, titre modifié au catalogue et dans la fiche | bout en bout « AC3 » | Conforme |

### AC4 — Écran vendeur : avertissement de retrait du catalogue

| Scénario | Test | Résultat |
|---|---|---|
| Publié, titre modifié (avant tout enregistrement) → « L’annonce sera retirée du catalogue » et « … retire l’annonce du catalogue le temps du contrôle … » affichés, aucun `PUT` parti | front « US-021 AC4 … article publié » | Conforme |
| Brouillon et refusé modifiés → ni avertissement de retrait ni de reprise du contrôle | front « US-021 AC4 … » (2 cas) | Conforme |
| `en_controle` et `a_verifier` modifiés → « Le contrôle repartira sur la nouvelle version », pas d'avertissement de retrait | front « US-021 AC4 … » (2 cas) | Conforme |
| axe-core sur l'écran de modification d'un publié, d'un en contrôle et d'un à vérifier, avertissement affiché → aucune violation serious ou critical | front « US-021 axe-core … » (3 cas) | Conforme |

### AC5 (Sécurité) — Résultat redélivré après la nouvelle révision : sans effet

| Scénario | Test | Résultat |
|---|---|---|
| Vrais services : article publié par un résultat publié sur RabbitMQ, puis modifié ; ce même résultat republié à l'identique (même `message-id`, identité `controle`) et un résultat `publie` forgé sur l'ancienne révision ; un témoin publié ensuite prouve le traitement → article toujours `en_controle`, même révision, même titre, même `modifieLe`, absent du catalogue et fiche 404 | bout en bout « AC5 » | Conforme : aucune publication de l'ancienne version |
| Handler de résultats : premier résultat redélivré (`Redelivered`) puis relivré après la nouvelle révision → acquitté, article identique, hors catalogue | `TestUS021AC5ResultatRedelivreApresNouvelleRevisionSansEffet` | Conforme |

## 5. Anomalies

Aucune anomalie : aucun comportement observé ne contredit un AC d'US-021.

## 6. Limites et AC non couverts ici

- **Playwright (acceptation API et E2E, AC1, AC3, AC4)** prévu par l'US : non couvert dans cette campagne (pas de navigateur réel ni d'environnement déployé). Les preuves ci-dessus reposent sur l'API réelle (PostgreSQL + RabbitMQ) côté back et sur une Place de marché simulée côté front (jsdom). Le parcours navigateur complet (modification d'un publié → disparition du catalogue public → retour après contrôle) reste à jouer en E2E.
- Le bout en bout US-021 n'exécute pas le binaire du Contrôle : les résultats sont publiés sur le vrai broker sous l'identité `controle`, ce qui teste le chemin de réception réel. Le traitement de la nouvelle demande par le Contrôle est déjà couvert par le bout en bout US-013.
- Contraste non mesurable en jsdom (`color-contrast` désactivé, comme pour US-013 et US-023).
- Test front en français seulement pour les libellés d'avertissement.

## 7. Conclusion

Tous les AC couvrables ici (AC1, AC2, AC3, AC4, AC5) sont couverts et conformes. Le volet Playwright E2E reste à exécuter.
