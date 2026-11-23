# US-009 AC11 — Rapport de tests front (annonce des erreurs du formulaire)

| Champ     | Valeur                                                                                                                                                                                                                                                                                                                                                                                                                                                |
| --------- | ----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| US        | [US-009 — Créer et modifier un brouillon d'article](../../../backlog/US-009-brouillon-article-vendeur.md)                                                                                                                                                                                                                                                                                                                                             |
| Branche   | `feat/US-009-accessibilite-formulaire`                                                                                                                                                                                                                                                                                                                                                                                                                |
| Date      | 2026-11-23                                                                                                                                                                                                                                                                                                                                                                                                                                            |
| Fonction  | Test                                                                                                                                                                                                                                                                                                                                                                                                                                                  |
| Périmètre | AC11 : « Given le formulaire, When axe-core l'analyse, Then aucune violation serious ou critical ; chaque champ a un libellé et ses erreurs sont annoncées. » Deux comportements corrigés côté front sont vérifiés : (1) focus sur le premier champ en erreur, déjà décrit par son erreur au moment où il le reçoit ; (2) une nouvelle soumission en erreur avec le même nombre d'erreurs est de nouveau annoncée, et une seule alerte reste affichée |
| Outils    | Vitest 3.2.4 + Testing Library + user-event 14.6.1 + axe-core (jsdom) ; Playwright 1.64.1 + @axe-core/playwright (Chromium, WCAG 2.1 A et AA), projets `fr` et `en`                                                                                                                                                                                                                                                                                   |
| Verdict   | **Tous les AC du périmètre sont couverts et conformes**                                                                                                                                                                                                                                                                                                                                                                                               |

## Méthode

Scénarios Given/When/Then dérivés de l'AC11 et des deux comportements attendus fournis par l'orchestrateur, écrits sans lire le code de production. On n'observe que le DOM accessible :

- **Description au moment du focus** : un écouteur `focusin` posé avant la soumission relève, au moment exact où chaque élément reçoit le focus, son `aria-invalid` et le texte des éléments pointés par `aria-describedby`. Si le focus arrivait avant le rendu des erreurs, le relevé montrerait `aria-invalid` différent de `true` ou une description sans le message.
- **Nouvelle annonce** : un `MutationObserver` compte les ajouts de nœuds et les changements de texte dans `role="alert"` après une soumission en erreur déjà annoncée. Une alerte laissée intacte (même texte, même nœud) donne 0 rendu, donc aucune nouvelle annonce pour un lecteur d'écran.

Fichiers ajoutés (tests uniquement) :

| Fichier                                                       | Contenu                                                   |
| ------------------------------------------------------------- | --------------------------------------------------------- |
| `web/src/acceptance/US-009/AC11-annonce-des-erreurs.test.tsx` | 13 cas Vitest (FR et EN)                                  |
| `web/e2e/US-009/AC11-formulaire-accessible.spec.ts`           | 3 cas Playwright, exécutés en `fr` et `en` (6 exécutions) |

Le cas axe « formulaire en erreur » était déjà couvert par l'écran « espace vendeur, formulaire en erreur » de `e2e/support/ecrans.ts` (US-025-AC4), après une seule soumission. Le nouveau cas Playwright ajoute l'analyse axe après une **seconde** soumission en erreur, en FR et en EN. `e2e/support/ecrans.ts` n'a pas été modifié.

## Résultats d'exécution

| Commande                                                                                                                                  | Résultat                                                                                                       |
| ----------------------------------------------------------------------------------------------------------------------------------------- | -------------------------------------------------------------------------------------------------------------- |
| `npx vitest run src/acceptance/US-009`                                                                                                    | Vert : 2 fichiers, 25 tests (12 existants + 13 nouveaux)                                                       |
| `npx vitest run` (suite complète)                                                                                                         | 549 tests réussis, 4 sautés, 0 test en échec. 1 fichier en échec, sur un hook hors périmètre (voir ci-dessous) |
| `E2E_LANGUES=fr,en npx playwright test e2e/US-009 e2e/US-025/AC4-axe-ecrans.spec.ts -g "US-009\|formulaire en erreur\|nouveau brouillon"` | Vert : 10/10 (6 US-009 + écrans axe « nouveau brouillon » et « formulaire en erreur », FR et EN)               |
| `npx tsc -b`                                                                                                                              | 0 erreur                                                                                                       |
| `npx eslint src e2e`                                                                                                                      | 0 problème                                                                                                     |
| `npx prettier --check .`                                                                                                                  | Conforme                                                                                                       |

Stabilité de la suite complète (sans lien avec AC11) :

- Premier passage : 17 échecs, tous par délai dépassé (`Test timed out in 5000ms` ou `findByRole` introuvable après 1 s), sur des fichiers de plusieurs US. Ce passage a duré pendant une mise en veille du poste (démarré le 20/11, terminé le 23/11). Un deuxième passage interrompu de la même façon a donné 7 échecs du même type.
- Passage sur poste stable : tous les tests verts. Seul le `beforeAll` de `src/acceptance/US-025/AC2-parite-et-texte-en-dur.test.ts` (« lint contre le texte en dur ») dépasse sa limite de 60 s, et il échoue aussi lancé seul. Sur ce poste, `npx eslint src` prend 73 s, ce qui explique le dépassement. Ce test lance ESLint sur des sondes temporaires et ne dépend pas d'AC11 ni des fichiers ajoutés. C'est un point d'environnement ou de performance du lint typé à surveiller en CI, pas une anomalie AC11.

## AC11 — Le premier champ en erreur reçoit le focus déjà décrit par son erreur

| Scénario                                                                                    | Attendu                                                                                                                                                                                                            | Réel                                                                                       | Statut   |
| ------------------------------------------------------------------------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------ | ------------------------------------------------------------------------------------------ | -------- |
| Vitest, FR et EN : 400 de l'API sur titre, catégorie et prix, enregistrement d'un brouillon | Focus sur le titre ; au `focusin`, `aria-invalid="true"` et description contenant « Texte trop long. » / « Text too long. »                                                                                        | Focus sur le titre ; au `focusin`, `aria-invalid="true"`, description contenant le message | Conforme |
| Vitest, FR et EN : prix illisible `abc` à la création                                       | Focus sur le prix ; au `focusin`, `aria-invalid="true"` et description contenant « Valeur invalide. Exemple de montant : 12,50. » / « Invalid value. Amount example: 12.50. »                                      | Conforme à l'attendu                                                                       | Conforme |
| Vitest, limite : erreur sur la seule catégorie (liste déroulante)                           | Focus sur la catégorie, déjà `aria-invalid="true"` et décrite par « Catégorie inconnue. »                                                                                                                          | Conforme à l'attendu                                                                       | Conforme |
| Playwright Chromium, FR et EN : prix `abc`, clic sur « Créer le brouillon »                 | Alerte « Le formulaire contient 1 erreur… » ; focus sur le prix ; au `focusin`, `aria-invalid="true"` et description contenant le message ; description accessible calculée par le navigateur contenant le message | Conforme à l'attendu                                                                       | Conforme |

## AC11 — Une nouvelle soumission en erreur est de nouveau annoncée

| Scénario                                                                                       | Attendu                                                                                                  | Réel                                             | Statut   |
| ---------------------------------------------------------------------------------------------- | -------------------------------------------------------------------------------------------------------- | ------------------------------------------------ | -------- |
| Vitest, FR et EN : 3 erreurs affichées, Entrée dans le prix, l'API renvoie les mêmes 3 erreurs | Au moins un nouveau rendu dans l'alerte ; une seule `role="alert"` ; texte « …3 erreurs… »               | Nouveau rendu observé ; 1 alerte ; texte attendu | Conforme |
| Vitest, FR et EN : prix illisible déjà signalé, Entrée dans le prix                            | Nouveau rendu ; une seule alerte ; « …1 erreur… »                                                        | Conforme à l'attendu                             | Conforme |
| Vitest, limite : trois clics successifs sur « Créer le brouillon » avec la même erreur         | Un rendu de l'alerte par soumission (≥ 1, ≥ 2, ≥ 3) ; une seule alerte à la fin                          | Conforme à l'attendu                             | Conforme |
| Vitest : nouvelle soumission par Entrée en erreur                                              | Focus revenu au premier champ en erreur (titre), toujours `aria-invalid="true"` et décrit par son erreur | Conforme à l'attendu                             | Conforme |
| Playwright Chromium, FR et EN : prix `abc`, clic puis Entrée dans le prix                      | Nouveau rendu dans l'alerte ; 1 alerte ; texte du résumé ; focus sur le prix                             | Conforme à l'attendu                             | Conforme |

## AC11 — axe-core

| Scénario                                                                                                                                          | Attendu                              | Réel | Statut   |
| ------------------------------------------------------------------------------------------------------------------------------------------------- | ------------------------------------ | ---- | -------- |
| Vitest (jsdom, sans contraste), FR et EN : après une seconde soumission en erreur                                                                 | Aucune violation serious ou critical | `[]` | Conforme |
| Playwright Chromium (WCAG 2.1 A et AA, contraste compris), FR et EN : après une seconde soumission en erreur, `lang` du document égal à la langue | Aucune violation serious ou critical | `[]` | Conforme |
| Playwright, non-régression : écrans « espace vendeur, nouveau brouillon » et « formulaire en erreur » (US-025-AC4), FR et EN                      | Aucune violation serious ou critical | `[]` | Conforme |

Avec axe-core en jsdom, la console affiche `HTMLCanvasElement.prototype.getContext` non implémenté (règle `label-content-name-mismatch`). C'est du bruit connu de jsdom, sans effet sur le résultat.

## Anomalies

Aucune anomalie AC11 constatée.

## Limites

- Les deux corrections n'ont pas été contre-éprouvées sur la version antérieure du front, car le testeur ne modifie pas le code de production. Ce qui permet aux tests de détecter une régression vient de la méthode : relevé au `focusin` et comptage des mutations de l'alerte. Ils ne reposent pas sur l'état final du DOM.
- L'annonce effective par un lecteur d'écran (NVDA, VoiceOver) n'est pas automatisable ici. On prouve son prérequis technique : une région `role="alert"` re-rendue à chaque soumission en erreur.
- En Playwright, la démo ne permet de déclencher que l'erreur de saisie côté client (prix illisible). Le cas à plusieurs erreurs renvoyées par l'API est couvert en Vitest.

## Conclusion

Tous les AC du périmètre (AC11) sont couverts et conformes.
