# US-021 — Revue de code

| Champ | Valeur |
|---|---|
| US | [US-021 — Recontrôler toute modification d'un article](../../../backlog/US-021-revalidation-apres-modification.md) |
| Branche | `feat/US-021-revalidation-apres-modification` (base `origin/main`, commit `a2669a9`) |
| Date | 2026-10-28 |
| Périmètre | `git diff origin/main...HEAD` : 20 fichiers. Back `services/place-de-marche` (domaine `article.go`, stockage `postgres.go`, faux stockage et tests), contrat `v1.openapi.yaml`, front `web/src/features/seller` (hooks, composants, client de démonstration), i18n, tests d'acceptation US-013 et US-021, [rapport de tests](../tests/US-021.md) |
| Références | [D02](../../decisions/D02-decoupage-applicatif.md) (revalidation sans seuil, disponibilité), `services/place-de-marche/README.md`, `web/README.md`, US-031 (hors périmètre : article réservé) |
| Verdict | **Fusion après correction** (0 bloquante, 1 majeure, 3 mineures, 4 suggestions) |

## Vérifications exécutées

| Commande | Résultat |
|---|---|
| `services/place-de-marche` : `gofmt -l .` | Aucun fichier |
| `services/place-de-marche` : `go vet ./...` | 0 constat |
| `services/place-de-marche` : `golangci-lint run ./...` | 0 issues |
| `services/place-de-marche` : `go test ./...` (Docker disponible) | Vert, 105 s ; `TestUS021…` (10 tests en mémoire et le bout en bout PostgreSQL + RabbitMQ) verts, `TestUS009IntegrationPostgreSQL` (dont modifications concurrentes) et `TestUS013BoutEnBout…` verts |
| `web` : `npm run i18n:check` | `Traductions alignées : 181 clés, langues fr, en.` |
| `web` : `npm run typecheck`, `npm run lint`, `npx prettier --check .` | 0 erreur, 0 avertissement, conforme |
| `web` : `npx vitest run` (3 passages) | 2 à 5 échecs par passage, **chaque fois sur des fichiers différents** (US-008 AC10/AC2, US-009 AC11, US-013 conflit-409, US-025 AC5), tous « Test timed out in 5000ms » ; les 16 fichiers concernés plus US-013 et US-021 relancés seuls : 191 tests verts. Lenteur de la machine, non imputable au diff (voir observation) |
| Recherche de commentaires dans les lignes ajoutées | Aucun |
| Recherche de mentions d'outils d'IA dans le diff et le message de commit | Aucune |

## Points forts

- Course et verrou (`services/place-de-marche/internal/annonces/stockage/postgres.go:120-146`) : `pgx.BeginFunc` + `SELECT etat, revision … FOR UPDATE` (`lockedStateQuery`, l. 89) avant toute écriture. Vérifié contre les deux courses :
  - modification contre résultat de contrôle : `ApplyReview` (l. 221) fait un `UPDATE … WHERE revision = $2 AND etat = 'en_controle'` ; si la modification passe d'abord, la réévaluation du `WHERE` après attente du verrou ne trouve plus la révision : sans effet. Si le résultat passe d'abord, le `FOR UPDATE` relit la ligne validée (`publie`, même révision) et la modification la renvoie au contrôle. Aucun entrelacement ne publie l'ancienne version ;
  - deux modifications concurrentes : la seconde attend le verrou puis lit la révision + 1 et répond 409. Le chemin est commun aux brouillons et aux articles publiés, donc exercé sur PostgreSQL par `ac2ModificationsConcurrentes` (`scenarios_test.go:173`, 8 requêtes, une gagnante).
- Enveloppement et mapping : `fmt.Errorf("modification de l'article : %w", err)` (l. 135) conserve les sentinelles ; `writeError` (`internal/annonces/api/handlers.go:255-270`) utilise `errors.Is` : 404 (`ErrArticleNotFound`, y compris article d'un autre vendeur, `ownedState` filtrant sur `vendeur_id`), 409 `revision-perimee`, 409 `article-non-modifiable`. Couvert sur PostgreSQL (`scenarios_test.go:167-168, 217, 311`) et en mémoire (`TestUS021AC1ModificationRefuseeDUnPublieSansEffet`, `TestUS021AC1PublieDUnAutreVendeurNonModifiable`).
- Outbox dans la même transaction : `reviseInTransaction` (l. 157) écrit l'article (`en_controle`, motif `NULL`, révision + 1) puis `enqueueReview` (l. 213) sur le même `tx` ; un échec d'encodage ou d'écriture annule la modification. `enqueueReview` et `scanReviewRequest` factorisent proprement ce que `Submit` faisait déjà (DRY sans changement de comportement, `TestUS013BoutEnBout…/AC5_échec_d'écriture_dans_l'outbox_annule_la_soumission` vert). La demande porte la nouvelle révision (vérifié par `exigerDemandeEnOutbox`, bout en bout).
- Résultats sur l'ancienne révision sans effet (AC2, AC5) : aucune logique nouvelle à faire confiance, la garde existante `revision = $2 AND etat = 'en_controle'` suffit puisque toute modification incrémente la révision. Prouvé pour les trois issues (`publie`, `refuse`, `a_verifier`) depuis `en_controle` et `a_verifier`, et par redélivraison réelle sur RabbitMQ (`revalidation_bout_en_bout_test.go:96-103`).
- Contrainte `article_motif_selon_etat` respectée : `motif = NULL` posé avec `etat = 'en_controle'` dans la même instruction (`reviseQuery`, l. 43-52).
- Domaine (`internal/annonces/article.go:61-94`) : `ReviewedOnEdit` et `Submittable` disjoints, `Editable = sellerHeld ∪ reviewOnEdit`, `VerifyEditable` ordonne état puis révision comme `explainRefusal` le faisait ; listes exposées par `slices.Clone`. `memberSinceColumn` et `contentAssignments` évitent de dupliquer le SQL entre `submitQuery`, `updateDraftQuery` et `reviseQuery`.
- Contrat OpenAPI mis à jour (résumé, réponse 200, description du schéma) et vérifié par `TestUS021ReponsesDeModificationConformesAuContratOpenAPI`.
- Front :
  - logique dans les hooks (`useRecontrolWarning`, `submitLabel` et `savedAnnouncement` dans `useArticleEditor`), composant de présentation pur (`RecontrolWarningNotice`) ; `mode` retiré du modèle sans consommateur restant (recherché) ;
  - accessibilité de l'avertissement : `Alert` en `role="note"` (pas d'annonce intempestive à l'ouverture, contrairement au `role="alert"` par défaut de MUI), `useId` et `aria-describedby` sur le bouton d'enregistrement : le lecteur d'écran lit l'avertissement au moment d'agir ; libellé du bouton explicite (« Enregistrer et renvoyer au contrôle ») ; axe-core sans violation sur les trois états ;
  - i18n : 6 clés ajoutées en fr et en, alignées (`i18n:check`), message de succès distinct quand l'article repasse au contrôle ;
  - `acceptsChanges` scindé en `acceptsSubmission` / `triggersRecontrol` : la soumission reste limitée à brouillon/refusé (`useSubmitForControl.ts:55`) ;
  - client de démonstration : `scheduleControl` ignore déjà une révision périmée (`demoSellerClient.ts:117-126`), donc l'ancienne planification est sans effet après une nouvelle modification.
- Retrait de `TestUS013AC1ArticleEnControleNonModifiable` justifié (règle inversée par AC2) et tracé dans le rapport de tests.

## Issues

### Bloquantes

Aucune.

### Majeures

1. `web/src/features/seller/components/ArticleWorkspace.tsx:41-45`, `web/src/features/seller/hooks/useArticleWorkspace.ts:35`, `web/src/features/seller/components/ArticleSummary.tsx`, `web/src/features/seller/hooks/useArticleSummary.ts` — **code mort introduit par le diff**. `acceptsChanges` (`sellerArticle.ts:87`) est désormais vrai pour les cinq valeurs de `ArticleState`, et `parseSellerArticle` rejette tout autre état (`sellerArticlePayload.ts:41`, `expectOneOf(…, articleStates, …)`) : `isEditable` vaut toujours `true`, la branche `ArticleSummary` est inatteignable, et aucun test ne l'exerce plus. Même cause dans `demoSellerClient.ts:175` : `expectAllowed(acceptsChanges, …)` ne peut plus lever `notModifiable`. Les clés `seller.editor.readOnly` et `seller.notProvided` (fr.ts/en.ts:57-58, 105) n'ont pas d'autre consommateur.
   Sur la justification « gardé pour US-031 » : US-031 est déclarée `Fonction(s) : Back` et D02 modélise la réservation comme une **disponibilité** (disponible, réservé, vendu), distincte de l'état de contrôle ; elle n'entrera pas dans `ArticleState`, donc ce code ne se rallumera pas tel quel. Le garder relève de YAGNI et le laisse pourrir sans test.
   Fix : supprimer `ArticleSummary.tsx`, `useArticleSummary.ts`, `isEditable` (modèle et branche, rendre `ArticleForm` directement), `acceptsChanges` et le contrôle `expectAllowed(acceptsChanges, …)` du client de démonstration, ainsi que les deux clés i18n orphelines. Si Martin veut le conserver, tracer la décision (ajout de la fonction Front et du cas « réservé » dans US-031) ; dans ce cas l'issue tombe. Dev : front.

### Mineures

1. Nommage `UpdateDraft` / `updateDraft` / `modifierBrouillon` — l'opération ne porte plus sur les seuls brouillons (publié, en contrôle, à vérifier). Sites : port `Articles.UpdateDraft` (`internal/annonces/service.go:30`), `Service.UpdateDraft` (`service.go:60`), `Store.UpdateDraft` (`stockage/postgres.go:120`), handler `updateDraft` et `draftUpdatePayload` (`api/handlers.go:106`, `api/payload.go:86`), `operationId: modifierBrouillon` et schéma `ModificationBrouillon` (`contracts/api/place-de-marche/v1.openapi.yaml:172, 363`, dont la description dit maintenant « contenu d'un article »), `SellerClient.updateDraft` (`web/src/features/seller/domain/sellerArticle.ts:60`). Renommer en `UpdateArticle` / `updateArticle` / `modifierArticle` / `ModificationArticle` (l'`operationId` et le nom de schéma ne sont pas sur le fil : aucun client cassé). `updateDraftInTransaction` et `updateDraftQuery` restent justes (branche brouillon/refusé uniquement). Dev : back (Go, contrat) et front (client).
2. `services/place-de-marche/README.md:3, 19-26` et `web/README.md:61-73` — documentation non étendue à US-021 : le README du service décrit le module comme « brouillons du vendeur » et le cycle de soumission ne mentionne pas la seconde entrée vers `en_controle` (modification d'un article `publie`, `en_controle` ou `a_verifier` : transaction avec `FOR UPDATE`, motif effacé, demande en outbox) ; le README web (table « Espace vendeur (US-009, US-013) ») ne mentionne ni l'avertissement ni le libellé « Enregistrer et renvoyer au contrôle ». Rien n'y est faux, d'où mineure ; la DoD exige « Documentation touchée à jour ». Ajouter un point « Modification après publication (US-021) » à chacun. Dev : back et front.
3. `services/place-de-marche/internal/annonces/article.go:74-76` — `Status.Editable()` est vrai pour les cinq états admis par la contrainte `article_etat_connu` : `VerifyEditable` ne renvoie plus jamais `ErrArticleNotEditable` sur le chemin PostgreSQL, et le 409 `article-non-modifiable` (`handlers.go:264`) n'est plus atteignable par le service. Contrairement au front, le maintien est défendable : US-031 AC5 (Back) exige un 409 pour la modification d'un article réservé, que cette sentinelle et ce type de problème serviront. À conserver, mais nommer la condition par ce qu'elle est (`sellerHeldStatuses ∪ reviewOnEditStatuses` = états modifiables par le vendeur) et rebrancher la garde de disponibilité dans `VerifyEditable` en US-031 ; si US-031 choisit un autre type de problème, supprimer alors `ErrArticleNotEditable` et `notEditable`. Dev : back.

### Suggestions

1. `services/place-de-marche/internal/annonces/stockage/postgres.go:243` — `explainRefusal` n'a plus qu'un appelant (`Submit`, l. 182) : les paramètres `allowed` et `notAllowed` ne servent plus la généricité. `Submit` pourrait adopter le même schéma que `UpdateDraft` (`FOR UPDATE` puis `VerifySubmittable` dans la transaction), ce qui supprimerait `explainRefusal`, la lecture hors transaction après échec et unifierait les deux chemins. Dev : back.
2. `services/place-de-marche/cmd/place-de-marche/memoire_test.go:20-23` — `etatsModifiablesSansControle` et `etatsRecontrolesALaModification` redéclarent les listes du domaine. Défendable (oracle indépendant du code testé) ; sinon utiliser `annonces.SellerHeldStatuses()` et `annonces.ReviewOnEditStatuses()`. Dev : back.
3. `services/place-de-marche/internal/annonces/stockage/postgres.go:30-52` — après `lockEditable`, les gardes `revision = $3 AND etat = ANY($4)` de `updateDraftQuery` et `reviseQuery` ne peuvent plus échouer (ligne verrouillée) ; si elles échouaient, `pgx.ErrNoRows` remonterait en 500. Les garder en défense en profondeur est acceptable ; un test de modifications concurrentes d'un article **publié** sur PostgreSQL (comptage des demandes en outbox = 1) documenterait l'invariant « une seule nouvelle demande ». Dev : back.
4. `web/src/features/seller/hooks/useArticleEditor.ts:142-155` — `savedAnnouncement` et `submitLabel` sont des fonctions locales appelées une seule fois au retour : des constantes calculées (ou deux petites fonctions pures hors du hook prenant `t`) seraient plus directes. Dev : front.

### Observations (hors issues)

- Échecs Vitest intermittents par dépassement de délai (5 s) sous charge, tournant d'un fichier à l'autre entre passages et absents en exécution ciblée ; aucun fichier touché par le diff n'échoue de façon reproductible. Le rapport de tests utilise `--pool=forks --poolOptions.forks.maxForks=2` ; à surveiller en CI, hors US-021.
- Le fichier de revue est nommé `US-021.md` à la demande ; les revues précédentes suivent `US-0XX-revue-code.md`.

## Verdict

| Sévérité | Nombre |
|---|---|
| Bloquante | 0 |
| Majeure | 1 |
| Mineure | 3 |
| Suggestion | 4 |

**Fusion après correction de la majeure 1** (ou décision tracée de Martin de conserver `ArticleSummary`). Le back est prêt : transaction, verrou, outbox et neutralisation des résultats périmés sont corrects et prouvés sur PostgreSQL et RabbitMQ réels. Les mineures 1 et 2 sont à solder de préférence dans la même branche (renommage mécanique, deux paragraphes de README).
