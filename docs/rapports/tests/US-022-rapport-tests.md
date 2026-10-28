# Rapport de tests — US-022 Trancher les articles « à vérifier »

| Champ | Valeur |
|---|---|
| US | [US-022](../../../backlog/US-022-ecran-admin-a-verifier.md) |
| Branche | `feat/US-022-ecran-admin-a-verifier` (empilée sur US-021) |
| Date | 2026-10-28 |
| Fonction | Test |
| Périmètre | Back : AC1, AC2, AC3, AC4, AC5 (Place de marché, en mémoire et sur PostgreSQL réel). Front : AC1, AC2, AC3, AC5, AC7, AC8 (écran `/admin`), axe-core. AC6 hors campagne (voir § 5) |
| Référentiel | AC de l'US, contrat `contracts/api/place-de-marche/v1.openapi.yaml` (tag Administration) : `GET /marche/v1/admin/articles` (rôle `admin`, seuls les `a_verifier`, du plus ancien au plus récent, contenu, révision, motif du contrôle, `vendeurId`) ; `POST /marche/v1/admin/articles/{id}/decision` `{revision, decision, motif?}` → 200 `publie`/`refuse`, 409 `revision-perimee` / `article-non-a-verifier`, 404, 400, 422 ; ligne `decision_admin` et log `event=annonces.decision_admin` |
| Verdict | **Conforme** : tous les AC couverts ici sont conformes ; aucune anomalie. Playwright E2E, CSP en navigateur réel et AC6 non couverts par cette campagne (voir § 5) |

## 1. Livrables de test

| Fichier | Contenu |
|---|---|
| `services/place-de-marche/cmd/place-de-marche/administration_test.go` | Tests d'acceptation en mémoire `TestUS022…` (21 tests, 76 avec les sous-tests) et aides partagées (jeton admin, décision, liste, lecture du journal JSON) |
| `services/place-de-marche/cmd/place-de-marche/administration_integration_test.go` | `TestUS022IntegrationPostgreSQL` : PostgreSQL 18.3 par testcontainers-go, rôle d'exécution réel, lecture de `decision_admin` par le rôle maître (7 scénarios) |
| `services/place-de-marche/cmd/place-de-marche/environnement_test.go` | `demarrerJournalise` : même serveur que `demarrer`, avec un journal fourni |
| `services/place-de-marche/cmd/place-de-marche/canal_fictif_test.go` | Le banc capture aussi le journal HTTP (`journalHTTP`, JSON) pour vérifier l'événement d'audit |
| `web/src/acceptance/US-022/adminReviewHarness.ts` | API d'administration simulée (liste triée, décision avec 409/422), libellés attendus, ouverture de l'écran en admin |
| `web/src/acceptance/US-022/ecran-admin-a-verifier.test.tsx` | AC1, AC2, AC3, AC5, AC7, AC8 sur l'application complète (`startApplication`, connexion OIDC simulée, `adminClient` en source `api`) : 23 tests, 33 cas |

## 2. Exécutions

| Commande | Résultat |
|---|---|
| `services/place-de-marche` : `go test ./...` (Docker, sans `-race`) | Vert, 104 s (tous les tests du service, dont US-022 en mémoire et sur PostgreSQL) |
| `services/place-de-marche` : `go vet`, `gofmt -l`, `golangci-lint run ./...` | 0 constat |
| `web` : `npx vitest run --pool=forks --poolOptions.forks.maxForks=2` | Vert : 30 fichiers, 347 tests (dont les 33 cas US-022) ; aucun dépassement de délai US-008 sur ce passage |
| `web` : `npm run -s typecheck` | 0 erreur |
| `web` : `npx eslint src` | 0 constat |
| `web` : `npx prettier --check src` | Conforme |

Le message jsdom « HTMLCanvasElement.prototype.getContext » affiché pendant les passages axe-core est un bruit d'environnement, sans effet sur le résultat.

## 3. Résultats par AC

### AC1 — Liste des articles à vérifier avec le contenu de la révision examinée et le motif du contrôle

| Scénario | Test | Résultat |
|---|---|---|
| Given 3 articles `a_verifier` (2 vendeurs) parmi des brouillon, en contrôle, refusé, publié When l'admin liste Then seuls les 3, du plus ancien au plus récent, total 3 | `TestUS022AC1ListeLesSeulsArticlesAVerifierDuPlusAncienAuPlusRecent` | Conforme |
| Chaque élément = id, révision, titre, description, catégorie, prix, frais de port, motif du contrôle (`vendeur_a_verifier`), `vendeurId` = sujet du vendeur, `modifieLe` | `TestUS022AC1ChaqueArticleAvecContenuRevisionMotifEtVendeur` | Conforme |
| Aucun article à vérifier → `elements: []`, page 1, taille 20, total 0 | `TestUS022AC1AucunArticleAVerifierListeVide` | Conforme |
| Pagination : page 2 taille 1 → le 2e ; page au-delà → vide, total 3 ; `taille=0`, `taille=51`, `page=0`, `page=abc` → 400 | `TestUS022AC1PaginationDeLaListe` | Conforme |
| Article à vérifier modifié par le vendeur → sort de la liste ; revenu à vérifier → listé en révision N+1 avec le nouveau titre | `TestUS022AC1ListeMontreLaRevisionExamineeApresNouveauControle` | Conforme |
| PostgreSQL : seuls les `a_verifier`, ordre, `vendeurId` et motif ; chaque article listé relu `a_verifier` par son vendeur | intégration « AC1 » | Conforme |
| Écran : cartes dans l'ordre reçu, description, catégorie, prix, frais, « Motif du contrôle : … », « Révision examinée : N », identifiant du vendeur ; annonce « 2 articles à vérifier » | front « AC1 … contenu de la révision examinée » | Conforme |
| Écran : les 6 motifs du contrôle affichés avec leur libellé ; article sans catégorie ni prix toujours affiché et actionnable ; état vide ; appel de liste porteur du jeton | front « AC1 » (9 cas) | Conforme |

### AC2 — Publier → au catalogue ; refuser avec un motif de la liste fermée → le vendeur voit « refusé » et le motif

| Scénario | Test | Résultat |
|---|---|---|
| Publier → 200 `publie`, motif nul ; présent dans `GET /marche/v1/articles`, fiche publique au bon titre ; vendeur lit `publie` ; sorti de la liste admin | `TestUS022AC2PublierFaitApparaitreAuCatalogue` | Conforme |
| Refuser avec chacun des 5 motifs (`annonce_incomplete`, `coordonnees_detectees`, `prix_abusif`, `article_interdit`, `vendeur_non_fiable`) → 200 `refuse` + motif ; `GET` du vendeur et « Mes articles » : `refuse` + motif ; hors catalogue, fiche 404 ; sorti de la liste | `TestUS022AC2RefuserAvecChaqueMotifDeLaListeFermee` (5 cas) | Conforme |
| PostgreSQL : publier → catalogue ; refuser `prix_abusif` → vendeur lit `refuse`/`prix_abusif` (lecture et liste), hors catalogue | intégration « AC2 AC4 publier » et « AC2 AC4 refuser » | Conforme |
| Écran : « Publier « titre » » → `POST …/{id}/decision` `{revision: 5, decision: "publier"}` (révision affichée, sans motif), annonce « … a été publié. », carte retirée | front « AC2 … le publie » | Conforme |
| Écran : 21 articles à vérifier (taille de page 20), l'admin va en page 2 et publie le seul article de cette page → annonce « … a été publié. », l'écran revient en page 1 avec les 20 restants dans l'ordre, « Aucun article à vérifier » n'est pas affiché, plus de bouton « Aller à la page 2 » ni de page 2 courante (correction de la majeure 1 de la revue) | front « AC2 … tranche le seul article de la page 2 » | Conforme |
| Écran : le dialogue « Refuser l’annonce » propose exactement les 5 motifs de la liste fermée, aucun présélectionné | front « AC2 … exactement les cinq motifs » | Conforme |
| Écran : refus avec chacun des 5 motifs → `{revision, decision: "refuser", motif}`, annonce « … a été refusé. », liste vide ensuite | front « AC2 … refuse avec le motif » (5 cas) | Conforme |
| Écran : confirmer sans motif → « Choisissez un motif de refus. », aucun appel ; annuler → aucun appel, article toujours listé | front « AC2 » (2 cas) | Conforme |

### AC3 — Décision sur la révision N alors que l'article est en N+1 : 409, et l'écran invite à recharger

| Scénario | Test | Résultat |
|---|---|---|
| Vendeur modifie (N+1 en contrôle), puis publier ou refuser sur N → 409 `revision-perimee` ; article identique, hors catalogue ; aucun événement d'audit | `TestUS022AC3DecisionSurRevisionPerimeeRefuseeSansEffet` (2 cas) | Conforme |
| Vendeur modifie, nouvelle révision revenue `a_verifier`, décision sur N (publier ou refuser) → 409 `revision-perimee`, rien d'appliqué, toujours listé | idem (2 cas) | Conforme |
| Révision future (N+1 inexistante) → 409 `revision-perimee`, sans effet | `TestUS022AC3RevisionFutureRefusee` | Conforme |
| Après rechargement, décision sur N+1 → 200 `publie` avec le titre modifié, au catalogue | `TestUS022AC3NouvelleRevisionTranchableApresRechargement` | Conforme |
| Article brouillon, en contrôle, publié, refusé → 409 `article-non-a-verifier`, sans effet | `TestUS022AC3ArticleQuiNEstPlusAVerifierRefuse` (4 cas) | Conforme |
| Seconde décision (autre admin) sur la même révision déjà refusée → 409 `article-non-a-verifier`, refus conservé | `TestUS022AC3SecondeDecisionSurLaMemeRevisionRefusee` | Conforme |
| PostgreSQL : article passé en N+1 et revenu `a_verifier`, décisions publier et refuser sur N → 409 `revision-perimee`, article identique, **aucune ligne** `decision_admin` ; décision sur N+1 → appliquée et enregistrée | intégration « AC3 révision N » | Conforme |
| PostgreSQL : 4 décisions concurrentes (publier/refuser) sur la même révision → exactement un 200 et trois 409, une seule ligne `decision_admin`, état final cohérent avec cette ligne | intégration « AC3 décisions concurrentes » | Conforme |
| Écran : révision changée depuis l'affichage, publier ou refuser → « L’article a changé, rechargez la liste » et bouton « Recharger la liste » ; rien d'appliqué | front « AC3 » (2 cas) | Conforme |
| Écran : 409 `article-non-a-verifier` → même invitation à recharger | front « AC3 … déjà tranché ailleurs » | Conforme |
| Écran : « Recharger la liste » → nouvelle lecture, carte en révision N+1, la décision suivante envoie N+1 | front « AC3 … recharge » | Conforme |

### AC4 — Événement d'audit : admin, article, révision, décision, motif

| Scénario | Test | Résultat |
|---|---|---|
| Publier et refuser (`coordonnees_detectees`) → exactement un événement `event=annonces.decision_admin` de niveau INFO pour l'article, portant le sujet de l'admin, l'id de l'article, la révision, la décision et le motif | `TestUS022AC4DecisionAppliqueeJournalisee` (2 cas) | Conforme |
| PostgreSQL, publier : une ligne `decision_admin` exacte (`admin_id` = sujet du jeton, révision, `publier`, motif NULL) et l'événement journalisé | intégration « AC2 AC4 publier » | Conforme |
| PostgreSQL, refuser : une ligne exacte (`refuser`, `prix_abusif`) et l'événement journalisé | intégration « AC2 AC4 refuser » | Conforme |
| PostgreSQL : décisions refusées (422 sans motif, 422 motif hors liste, 400 révision 0, 403 vendeur, 401 sans jeton) → aucune ligne, article inchangé | intégration « décisions invalides ou interdites » | Conforme |
| Rôle d'exécution : `DELETE` et `UPDATE` sur `decision_admin` refusés (42501) | `TestUS009IntegrationPostgreSQL` / droits minimaux (déjà étendu sur la branche) | Conforme |

### AC5 (Autorisation) — Vendeur ou acheteur : 403 sur la liste et la décision

| Scénario | Test | Résultat |
|---|---|---|
| Jetons vendeur, vendeur propriétaire de l'article, acheteur, vendeur+acheteur, sans rôle, rôles voisins (`admins`, `Admin`, `administrateur`), `admin` en rôle client hors realm → 403 Problem Details sur les deux routes ; article inchangé, aucun événement d'audit | `TestUS022AC5VendeurEtAcheteurInterditsSurLesDeuxRoutes` (14 cas) | Conforme |
| Sans jeton, jeton admin expiré, jeton admin d'un autre émetteur → 401 avec `WWW-Authenticate` sur les deux routes, sans effet | `TestUS022AC5SansJetonOuJetonInvalideNonAuthentifie` | Conforme |
| Jeton admin sur la route vendeur → 403 (pas d'élévation croisée) | `TestUS022AC5AdminNAccedePasAuxRoutesDuVendeur` | Conforme |
| Écran : vendeur et acheteur connectés ouvrent `/admin` → titre « Accès réservé », aucune action, **aucun appel** à `/marche/v1/admin/…` | front « AC5 » (2 cas) | Conforme |

### Saisies invalides et conformité au contrat

| Scénario | Test | Résultat |
|---|---|---|
| 422 `saisie-invalide`, erreur `motif` : refuser sans motif (`obligatoire`), publier avec motif, motif du contrôle `categorie_inconnue`, motif inconnu, motif vide (`invalide`) ; sans effet | `TestUS022MotifInvalideRefuseEn422SansEffet` (5 cas) | Conforme |
| 400 : révision absente, 0, négative, en texte ; décision absente ou inconnue ; champ inconnu ; clé dupliquée ; JSON mal formé ; corps vide ; tableau ; sans effet | `TestUS022SaisieInvalideRefuseeEn400SansEffet` (11 cas) | Conforme |
| 415 (`text/plain`), 413 (corps > 32 Kio) ; sans effet | `TestUS022CorpsHorsFormatRefuseSansEffet` | Conforme |
| Article inexistant → 404 | `TestUS022DecisionSurArticleInexistantIntrouvable` | Conforme |
| Toutes les réponses jouées (200 liste et décision, 400, 401, 403, 404, 409, 415, 422) validées contre le contrat OpenAPI, en mémoire et sur PostgreSQL | `TestUS022ReponsesAdministrationConformesAuContratOpenAPI`, intégration « contrat » | Conforme |

### AC7 (Sécurité, sorties) — Titre avec balisage affiché comme du texte

| Scénario | Test | Résultat |
|---|---|---|
| Titre `<script>…</script><b>Gras</b> & <img src=x onerror=…>` et description `<i>…</i><script>…</script>` → titre de carte au texte exact, sans enfant ; aucun `script`, `b`, `i`, `img[src=x]` créé dans l'application ; description affichée littéralement ; aucun script exécuté | front « AC7 … comme du texte » | Conforme |
| Même titre dans le dialogue de refus, les libellés accessibles des boutons et l'annonce « … a été publié. » → texte, aucun élément créé, aucun script exécuté | front « AC7 … dialogue, libellés et annonce » | Conforme |

Le volet « sans violation de la politique de sécurité du contenu » ne se prouve pas en jsdom (pas d'application de CSP) : voir § 5.

### AC8 (Accessibilité) — axe-core sans violation serious ou critical

| Scénario | Test | Résultat |
|---|---|---|
| Liste de 2 articles ; dialogue de refus avec l'erreur de motif ; invitation à recharger après 409 ; état vide ; « Accès réservé » pour un vendeur | front « AC8 » (5 cas) | Conforme : aucune violation serious ou critical |

## 4. Anomalies

Aucune anomalie : aucun comportement observé ne contredit un AC d'US-022 ni le contrat Administration.

Observation sans écart : à la fermeture du dialogue de refus, l'animation MUI laisse brièvement le reste de la page hors de l'arbre d'accessibilité ; les tests attendent la fermeture avant d'agir. Comportement standard, aucun AC concerné.

## 5. Limites et AC non couverts ici

- **AC6 (second facteur admin, pas pour le vendeur)** : hors campagne à la demande de l'orchestrateur, couvert par le travail en cours dans `infra/identite/` (non touché).
- **Playwright** (acceptation API AC2/AC3/AC5, E2E AC1/AC6/AC7/AC8) prévu par l'US : non couvert (pas de navigateur réel ni d'environnement déployé). Les preuves reposent sur l'API réelle en processus (en mémoire et PostgreSQL) côté back et sur une API d'administration simulée côté front (jsdom). Le parcours bout en bout navigateur → API → base reste à jouer.
- **CSP en navigateur réel (AC7)** : jsdom n'applique pas la politique de sécurité du contenu ; seule la neutralisation du rendu (aucun élément créé, aucun script exécuté) est prouvée. L'absence de violation CSP (rapport `securitypolicyviolation`) reste à vérifier en Playwright avec les en-têtes servis.
- Contraste non mesurable en jsdom (`color-contrast` désactivé, comme pour US-013, US-021 et US-023).
- Front testé en français seulement.
- L'intégration PostgreSQL d'US-022 n'implique pas RabbitMQ : les articles sont amenés à `a_verifier` par le gestionnaire de résultats réel appelé en processus.

## 6. Conclusion

Tous les AC couverts ici (AC1, AC2, AC3, AC4, AC5, AC7, AC8) sont conformes. AC6, le volet CSP navigateur d'AC7 et les parcours Playwright restent à exécuter hors de cette campagne.
