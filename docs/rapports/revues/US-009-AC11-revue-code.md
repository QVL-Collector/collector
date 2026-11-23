# US-009 AC11 — Revue de code (annonce des erreurs du formulaire)

| Champ | Valeur |
|---|---|
| US | [US-009 — Créer et modifier un brouillon d'article](../../../backlog/US-009-brouillon-article-vendeur.md) |
| Branche | `feat/US-009-accessibilite-formulaire` (HEAD `47ad500`, identique à `main`) ; changements non commités revus par `git diff main` et fichiers non suivis |
| Date | 2026-11-23 |
| Périmètre | AC11 seulement. Front `web/src/features/seller` : `hooks/useArticleFieldFocus.ts` (focus différé après rendu), `hooks/useArticleEditor.ts` (compteur `attempt` dans `FormFeedback` `invalid`), `components/ArticleFormFeedback.tsx` (`key` de l'alerte) ; tests ajoutés `src/acceptance/US-009/AC11-annonce-des-erreurs.test.tsx`, `e2e/US-009/AC11-formulaire-accessible.spec.ts` ([rapport de tests](../tests/US-009-AC11-rapport-tests.md)) |
| Références | AC11 : « aucune violation serious ou critical ; chaque champ a un libellé et ses erreurs sont annoncées » |
| Hors revue (connu) | AC5, AC10 (DevOps) ; dépassement de 60 s du `beforeAll` de `US-025/AC2-parite-et-texte-en-dur.test.ts` (lint typé lent, signalé par le testeur, sans lien avec AC11) |
| Verdict | **Fusion possible** (0 bloquante, 0 majeure, 0 mineure, 1 suggestion) |

## Vérifications exécutées

| Commande | Résultat |
|---|---|
| `npx tsc -b` (`web`) | 0 erreur |
| `npx eslint src/features/seller src/acceptance/US-009 e2e/US-009` | 0 problème |
| `npx prettier --check src/features/seller src/acceptance/US-009 e2e/US-009` | Conforme |
| `npx vitest run src/acceptance/US-009 src/acceptance/US-013 src/acceptance/US-021 src/features/seller` (non-régression vendeur) | 7 fichiers, 98 tests verts. Un premier passage lancé en même temps que `tsc` et ESLint a donné 4 échecs par délai dépassé (dont `US-013/conflit-409.test.tsx`) ; relancé seul, tout est vert. Même instabilité sous charge que celle décrite par le testeur, sans lien avec le diff |
| Playwright `e2e/US-009` | Non rejoué en revue ; 10/10 vert selon le rapport de tests |
| Commentaires ajoutés (`//`, `/* */`) dans le diff | Aucun |
| Appelants des symboles partagés modifiés | `useArticleFieldFocus` / `focusField` : seul appelant `useArticleEditor.ts:56` et `:77`, signature inchangée ; `FieldRefCallback` (`useArticleFieldModels.ts:4`) inchangé. `FormFeedback` (variante `invalid` élargie de `attempt`) : seul constructeur `useArticleEditor.ts:70-74`, seul lecteur `ArticleFormFeedback.tsx` (rendu par `ArticleForm.tsx:84`) ; aucun autre `kind: 'invalid'` de ce type dans `src` (celui de `articleFailure.ts:6` est un autre type). `tsc` vert |

## Points forts

- **Focus réellement après le rendu des erreurs** : `focusField` (`useArticleFieldFocus.ts:45-47`) ne focalise plus directement, il pose une demande `FocusRequest` ; l'effet (`:40-44`) appelle `focus()` après le commit. `setFieldErrors`, `setFeedback` et `setRequest` sont groupés dans le même rendu (gestionnaire d'événement côté client, `onError` de la mutation côté API, regroupement automatique de React 18) : au moment du focus, `aria-invalid` et `aria-describedby` sont déjà dans le DOM. Le nouvel objet à chaque appel rend la demande unique : un second échec sur le même champ redéclenche bien l'effet.
- **Nouvelle annonce sans double alerte** : `key={feedback.attempt}` (`ArticleFormFeedback.tsx:42`) remplace le nœud `role="alert"` au lieu de le laisser intact ; le `switch` ne rend qu'une branche, donc une seule alerte à l'écran. Compteur vérifié sur les deux chemins : erreur côté client répétée → `attempt` incrémenté (`useArticleEditor.ts:73`), nouvelle clé ; erreur renvoyée par l'API → `submit` remet `noFeedback` avant l'appel (`:116`), l'alerte est démontée puis remontée.
- **Diff minimal et ciblé** : 19 lignes ajoutées, aucune signature publique modifiée, aucun code mort laissé (tous les imports ajoutés sont utilisés, ancienne focalisation directe supprimée).
- **Tests qui prouvent le comportement, pas l'état final** : relevé de `aria-invalid` et de la description au `focusin` (un focus prématuré serait vu), comptage des mutations dans `role="alert"` (une alerte intacte donnerait 0) ; FR et EN ; cas limites utiles (catégorie seule, trois clics successifs, retour du focus sur le titre, axe-core après une seconde soumission) ; écouteurs et observateurs nettoyés par `onTestFinished`. Le Playwright confirme dans un vrai navigateur, contraste compris.

## Issues

### Bloquantes

Aucune.

### Majeures

Aucune.

### Mineures

Aucune.

### Suggestions

1. `web/src/features/seller/hooks/useArticleEditor.ts:73` — `attempt: current.kind === 'invalid' ? current.attempt + 1 : 1`. Pour une erreur renvoyée par l'API, `attempt` vaut toujours 1 : la nouvelle annonce ne tient qu'au rendu intermédiaire `noFeedback` posé par `submit` (`:116`), qui démonte l'alerte. C'est correct aujourd'hui et couvert par le test « trois erreurs affichées, Entrée dans le prix, l'API renvoie les mêmes trois erreurs », mais la garantie dépend d'une autre ligne du hook. Un compteur strictement croissant (par exemple `attempt: (current.kind === 'invalid' ? current.attempt : previousAttempt.current) + 1` via un `useRef`, ou un simple compteur de soumissions en erreur) rendrait la clé unique par construction, sans dépendre de la remise à zéro. Non bloquant. Dev : front.

## Verdict

| Sévérité | Nombre |
|---|---|
| Bloquante | 0 |
| Majeure | 0 |
| Mineure | 0 |
| Suggestion | 1 |

**Fusion possible.** Les deux défauts visés par AC11 sont corrigés de façon juste et simple : le premier champ en erreur reçoit le focus après le rendu, donc déjà `aria-invalid` et décrit par son message ; une nouvelle soumission en erreur, même avec un nombre d'erreurs identique, remplace l'alerte et sera de nouveau annoncée, avec une seule alerte à l'écran. Aucun appelant impacté, aucune régression sur les suites vendeur (US-009, US-013, US-021, `features/seller`). Format, lint et types propres, sans commentaire. Répartition : front pour la suggestion 1. Aucune correction back, testeur ou devops.
