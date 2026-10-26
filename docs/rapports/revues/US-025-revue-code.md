# US-025 — Revue de code front (lot 1)

| Champ | Valeur |
|---|---|
| US | [US-025 — Utiliser Collector en anglais, avec des écrans accessibles](../../../backlog/US-025-francais-anglais-accessibilite.md) |
| Branche | `feat/US-025-francais-anglais-accessibilite` (base `origin/main`, commits `4fcc52b`, `27df156`) |
| Date | 2026-10-26 |
| Périmètre | Lot 1 (écrans existants) : `web/` (26 fichiers), `.github/workflows/ci.yml` (1 étape), realm XP03 (3 champs), [rapport de tests](../tests/US-025-rapport-tests.md). Lot 2 reporté (fiche, mise en vente, admin, Playwright) : hors revue |
| Références | [D06](../../decisions/D06-authentification-autorisation.md), `web/README.md` |
| Verdict | **Fusion possible** (0 bloquante, 0 majeure, 2 mineures, 2 suggestions) |

## Vérifications exécutées (depuis `web/`)

| Commande | Résultat |
|---|---|
| `npm run i18n:check` | `Traductions alignées : 70 clés, langues fr, en.` |
| `npm run typecheck` (`tsc -b`, inclut désormais `scripts/`) | 0 erreur |
| `npm run lint` | 0 erreur, 0 avertissement |
| `npx prettier --check .` | Conforme |
| `npm test` | 19 fichiers, 187 tests verts (dont US-008 et US-010 inchangés) |
| Recherche de commentaires dans le diff | Aucun |
| Recherche de mentions d'outils d'IA dans le diff et les messages de commit | Aucune |

## Points forts

- Transport de la langue dans le `state` OIDC sûr et minimal :
  - écriture : `toSignInState` (`web/src/auth/signInIntent.ts:18`) ne transporte que `returnPath` (déjà assaini par `toSafeReturnPath`) et `language` (type fermé `SupportedLanguage`) ; aucune autre donnée ;
  - lecture : `readReturnPath` et `readLanguage` (`signInIntent.ts:22-28`) traitent le `state` comme `unknown`, passent par `isRecord`, puis par `toSafeReturnPath` et la liste blanche `toSupportedLanguage` : un `state` forgé ou corrompu retombe sur `/` et `undefined`, sans injection possible (aucune interpolation, la langue n'alimente que `i18n.changeLanguage` et `ui_locales` depuis une valeur typée) ;
  - échec : `readFailedSignInLanguage` (`signInIntent.ts:31`) s'appuie sur `ErrorResponse.state` d'oidc-client-ts (vérifié dans `oidc-client-ts.d.ts`), avec la même validation ;
  - D06 respectée : aucun nouveau stockage ; la langue rejoint le `state` déjà porté par le `stateStore` existant, les jetons restent en `InMemoryWebStorage`.
- Changement de contrat `signIn({ returnPath, language })` / `completeSignIn(): Promise<SignInOutcome>` sans régression : tous les sites d'appel recherchés et mis à jour (`useAuthActions.ts:37`, `useAuthenticationGate.ts:45`, `useSignInCallback.ts:20`, `AuthSession.ts:32-34` pour la session indisponible) ; union discriminée `SignInOutcome` exhaustive ; mémoïsation `pendingCallback` conservée ; `.catch(() => unexpectedFailure)` garde l'écran d'échec en cas d'erreur inattendue ; dépendances d'effets complètes (`changeLanguage` stable sur `[i18n]`). Tests US-008 verts.
- `ui_locales` transmis au serveur d'autorisation (`createOidcSession.ts:49`), cohérent avec l'ajout minimal au realm (`internationalizationEnabled`, `supportedLocales`, `defaultLocale`, 3 lignes).
- `scripts/check-translations.ts` : sans dépendance (type stripping natif de Node 24, aligné sur `NODE_VERSION: "24"` de la CI), contrôle clés manquantes, clés en trop, textes vides et variables `{{…}}` ; code de sortie 1 bloquant. `tsconfig.node.json` : `allowImportingTsExtensions` (compatible `noEmit`) et `erasableSyntaxOnly` garantissent que le script reste exécutable par Node ; inclusion de `scripts/` dans `tsc -b`.
- Étape CI `npm run i18n:check` placée après ESLint et avant Vitest, dans le job `web` déjà conditionné aux changements.
- Thème : anneau de focus factorisé (`focusRing`, `createAppTheme.ts:55`) et appliqué à `.Mui-focusVisible` des `ButtonBase` (focus visible malgré l'ondulation) ; gris des `ToggleButton` remplacé par `textSecondary` ; calcul WCAG correct (`contrast.ts`, linéarisation sRGB et offset 0,05) et couples de jetons déclarés (`themeContrastPairs.ts`).
- Motifs de refus : liste fermée `controlReasons` alignée sur le contrat `controle.resultat` v1 (test croisé avec le schéma JSON), clés typées via `Translation`.
- README à jour (commande `i18n:check`, conservation de la langue, règles de lint, focus, contrastes).
- TypeScript strict : aucune assertion de type ni `any` dans le code de production ajouté.

## Issues

### Bloquantes

Aucune.

### Majeures

Aucune.

### Mineures

1. `web/eslint.config.js:29-34` — la règle `no-restricted-syntax` ne cible que le littéral enfant direct de l'attribut ou de son `JSXExpressionContainer`. Contournements triviaux non détectés : `aria-label={cond ? 'Fermer' : 'Ouvrir'}`, `title={'Fermer' + suffix}`, `label={x ?? 'Défaut'}`. Ajouter des sélecteurs `ConditionalExpression > Literal`, `LogicalExpression > Literal` et `BinaryExpression > Literal` sous `${textAttribute}` (la liste d'attributs elle-même n'est pas trop large : lint vert sans faux positif). Dev : front.
2. `web/src/i18n/useControlReasonLabel.ts:5` et `web/src/shared/controlReason.ts:1` — aucun consommateur en production dans le lot 1 (seuls les tests US-025 les utilisent). Justifié par AC1 et les écrans du lot 2 (fiche, mise en vente, admin) ; à brancher dans le lot 2, sinon ce sera du code mort. Dev : front.

### Suggestions

1. `web/scripts/check-translations.ts:54` — `compare(reference.language, expected, expected)` sert uniquement à détecter les textes vides du français (clés et variables trivialement égales). Extraire une fonction `emptyTexts(language, translations)` rendrait l'intention explicite. Dev : front.
2. `web/src/ui/theme/contrast.ts` et `web/src/ui/theme/themeContrastPairs.ts` — utilisés uniquement par `AC4-contrastes-wcag.test.ts`, donc comptés dans la couverture du code de production. Choix documenté dans le README et défendable (preuve hors navigateur à côté des jetons) ; à garder tel quel ou à déplacer près des tests si la couverture doit refléter le seul code livré. Dev : front.

## Verdict

| Sévérité | Nombre |
|---|---|
| Bloquante | 0 |
| Majeure | 0 |
| Mineure | 2 |
| Suggestion | 2 |

**Fusion possible pour le lot 1.** Les mineures ne bloquent pas ; la mineure 2 est à solder dans le lot 2.
