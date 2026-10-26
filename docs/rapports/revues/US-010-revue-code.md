# US-010 — Revue de code front

| Champ | Valeur |
|---|---|
| US | [US-010 — Consulter le catalogue public sans compte](../../../backlog/US-010-catalogue-public.md) |
| Branche | `feat/US-010-catalogue-public` (base `main`, commits `ca0797f`, `fad5c07`) |
| Date | 2026-10-26 |
| Périmètre | `web/` (100 fichiers), `README.md` (1 ligne), [rapport de tests](../tests/US-010-rapport-tests.md) |
| Références | [D04](../../decisions/D04-stack.md), [D06](../../decisions/D06-authentification-autorisation.md), [D08](../../decisions/D08-exposition-tls-domaine.md) |
| Verdict | **Fusion possible après correction des 2 issues majeures** (0 bloquante, 2 majeures, 5 mineures, 5 suggestions) |

## Vérifications exécutées (depuis `web/`)

| Commande | Résultat |
|---|---|
| `npm run lint` | 0 erreur, 0 avertissement |
| `npm run typecheck` (`tsc -b`) | 0 erreur |
| `npm test` | 4 fichiers, 36 tests verts |
| `npm run build` | Build OK ; avertissement de taille (bloc principal 547 kB, 175 kB gzip) |
| `npm run format:check` | Échec local sur 91 fichiers, cause : fins de ligne CRLF de l'extraction Windows (voir m1) ; conforme avec `--end-of-line auto` |
| `tools/ci/verifier-eslint.sh` (simulé, `jq` absent en local) | `react/no-danger` en erreur, 31 règles `jsx-a11y` en erreur : passe |
| `npx tsc --noEmit` (commande de la CI) | **0 fichier vérifié** (voir M1) |
| Recherche de commentaires dans `web/` | Aucun commentaire dans le code, la configuration nginx ou le Dockerfile |
| Recherche de mentions d'outils d'IA dans le diff et les messages de commit | Aucune |

## Points forts

- Séparation logique et présentation tenue : composants de `features/catalogue/components/` en JSX pur ; état, requêtes et URL dans `hooks/` (`useCataloguePage.ts`, `useCataloguePageParam.ts`, `usePublishedArticles.ts`) ; vue dérivée en fonction pure (`catalogueView.ts:22`) et `switch` exhaustif (`CatalogueContent.tsx:11`, règle `switch-exhaustiveness-check`).
- TypeScript strict poussé (`noUncheckedIndexedAccess`, `verbatimModuleSyntax`, `strictTypeChecked`) ; aucune assertion de type ni `any` dans `src/` hors tests.
- Frontière d'entrée validée champ par champ (`shared/payload.ts`, `cataloguePayload.ts:41`) : un écart de contrat donne l'état d'erreur, jamais un rendu partiel.
- Sécurité :
  - aucun HTML interprété : `react/no-danger`, `no-unsanitized`, `no-eval` en erreur ; titres rendus en texte JSX (`ArticleCard.tsx:29`) ; `escapeValue: false` d'i18next légitime puisque React échappe ;
  - jetons : `localStorage`, `sessionStorage`, `document.cookie` interdits au lint (`eslint.config.js:50-60`) ; `getAccessToken` en mémoire (`auth/AuthSession.ts`) ; `credentials: 'omit'` (`httpClient.ts:45`) ;
  - `marketplaceApiUrl` limité à un chemin de même origine (`runtimeConfig.ts:31`) : le futur `Authorization: Bearer` ne peut pas partir vers une autre origine ;
  - CSP nginx conforme à D04/D08 : `script-src 'self'` sans `unsafe-inline`, `style-src 'self' 'unsafe-inline'` (écart Emotion accepté par D04 et D08 § Q5), `frame-ancestors 'none'`, `base-uri 'none'`, `object-src 'none'` ; `modulePreload.polyfill: false` et `assetsInlineLimit: 0` évitent tout script ou ressource en ligne ;
  - en-têtes `always` repris dans chaque `location` et `server_tokens off` (ME2) ; HSTS laissé à l'ALB (ME1) ;
  - image `nginx-unprivileged`, `USER 101`, port 8080, `npm ci --ignore-scripts`.
- Accessibilité : lien d'évitement, focus sur le `h1` à chaque navigation (`usePageIdentity.ts`), zone `role="status"`, `aria-busy` pendant le changement de page, squelette `aria-hidden`, libellés de pagination traduits, `lang` des options de langue, animations coupées sous `prefers-reduced-motion`.
- Nommage cohérent et sans nombre magique (`cataloguePageSize`, `demoLatencyMs`, `clientErrorStatuses`, `themeVariables`).
- Tests d'acceptation en boîte noire sur l'application complète, étiquetés par AC, avec cas limites (`?page=abc`, `?page=99`, 12 articles exacts).

## Issues

### Bloquantes

Aucune.

### Majeures

**M1 — La vérification de types de la CI ne vérifie aucun fichier** — dev front

- `web/tsconfig.json:2` : `"files": []` avec `references` (solution de projets) ; `.github/workflows/ci.yml:156` lance `npx --no-install tsc --noEmit`, qui sans `-b` ne suit pas les références.
- Preuve : `npx tsc --noEmit --listFilesOnly` liste 0 fichier, contre 100 fichiers de `src/` pour `tsconfig.app.json`.
- Impact : l'étape « Types » de la CI est toujours verte ; une erreur de type introduite plus tard ne serait pas détectée par la CI (le build Vite ne vérifie pas les types). Le garde-fou TypeScript strict n'existe qu'en local.
- Correction : remplacer l'étape par `npm run typecheck` (`tsc -b`) dans `ci.yml` (fichier de la chaîne CI, à coordonner avec DevOps), ou à défaut `npx --no-install tsc -b`.

**M2 — Code mort et configuration sans consommateur** — dev front

- `web/src/api/ApiError.ts:3-11` et `web/src/api/problemDetails.ts` (module entier) : `ApiError.kind`, `ApiError.problem` et `ProblemSummary` ne sont lus nulle part (recherche dans `src/`, tests compris) ; `readProblemSummary` lit et analyse le corps de chaque réponse en erreur (`httpClient.ts:62`) pour un résultat jamais utilisé.
- `web/src/config/runtimeConfig.ts:15-18, 28, 52-61` : `OidcConfig`, `RuntimeConfig.oidc` et `parseOidcConfig` ne sont consommés par aucun code ; seule la clé `oidc: null` de `public/config.json` et des tests les fait exister.
- `web/src/auth/AuthSession.ts:2,7` : `isAuthenticated` jamais lu.
- Impact : du code exécuté et maintenu sans effet (YAGNI) ; la forme du contrat OIDC et des Problem Details est figée avant US-008 et avant l'OpenAPI (US-009, US-013).
- Correction : supprimer `problemDetails.ts`, `ProblemSummary`, les champs `kind` et `problem` (garder `status`, seul lu par `createQueryClient.ts:11`) ; retirer `oidc` du type, du parseur, de `public/config.json`, des deux tests et du tableau de `web/README.md` ; retirer `isAuthenticated`. US-008 les réintroduira avec leur premier usage. Si l'équipe préfère garder l'extension OIDC, l'acter dans le README comme décision et la tester ; en l'état, rien ne la prouve.

### Mineures

**m1 — Fins de ligne non imposées : `format:check` échoue sur une extraction Windows** — dev front

- `.gitattributes:1` (`* text=auto`) et `core.autocrlf=true` donnent des fichiers CRLF dans l'arbre de travail ; `.prettierrc.json:6` exige `"endOfLine": "lf"`. Résultat : `npm run format:check` signale 91 fichiers en local, alors que le dépôt est en LF et que le rapport de tests indique « Conforme ».
- Correction : ajouter `web/** text eol=lf` dans `.gitattributes` (ou `* text=auto eol=lf`).

**m2 — Exports inutiles** — dev front

- `web/src/features/catalogue/hooks/usePublishedArticles.ts:5` (`publishedArticlesQueryKey`) et `web/src/features/catalogue/api/httpCatalogueClient.ts:5` (`publishedArticlesPath`) ne sont utilisés que dans leur fichier. Retirer `export`.

**m3 — Prop `hidden` trompeuse** — dev front

- `web/src/ui/components/CardGrid.tsx:6,17` : `hidden` porte la sémantique de l'attribut HTML (« masqué visuellement ») mais ne pose que `aria-hidden`. Renommer en `decorative` (ou `ariaHidden`) pour lever l'ambiguïté.

**m4 — Harnais de test : nom de hook et global non restauré** — testeur

- `web/src/acceptance/US-010/catalogueHarness.tsx:59` : `useLanguage` n'est pas un hook et porte le nom de `src/i18n/useLanguage.ts`. Renommer en `switchLanguage`.
- `catalogueHarness.tsx:48` : `globalThis.Request` est remplacé dans `beforeAll` et jamais restauré (`vi.unstubAllGlobals` ne couvre pas une affectation directe). Utiliser `vi.stubGlobal('Request', RequestWithoutJsdomSignal)` dans `beforeEach`, ou restaurer l'original dans `afterAll`.

**m5 — Rapport de tests : ligne `format:check` inexacte sous Windows** — testeur

- `docs/rapports/tests/US-010-rapport-tests.md`, tableau « Résultats d'exécution » : `npm run format:check` « Conforme » n'est vrai que sur une extraction LF. Préciser l'environnement ou relancer après m1.

### Suggestions

- **s1** — `web/src/i18n/useMoneyFormatter.ts:12` : un `Intl.NumberFormat` est créé par article à chaque rendu. Mettre les formateurs en cache par devise (une `Map` dans un `useMemo` sur `language`). Dev front.
- **s2** — Build : bloc principal de 547 kB (avertissement Vite). Séparer React, MUI et TanStack dans `build.rollupOptions.output.manualChunks`. Dev front.
- **s3** — `web/public/config.json:4` : l'image embarque `catalogue.source = "demo"` par défaut ; sans ConfigMap monté, la production afficherait le jeu fictif (titres piégés compris). Mettre `"api"` dans le fichier de l'image dès que l'API d'US-009 existe, et garder la démo pour le développement. Dev front.
- **s4** — `web/Dockerfile:1,8` : images épinglées par étiquette ; un épinglage par empreinte (`@sha256:`) renforcerait la reproductibilité et l'analyse ES-05. Dev front.
- **s5** — La CI front ne lance ni `format:check` ni `build` (`ci.yml:134-170`, antérieur à cette US). Les ajouter fermerait les écarts de m1 et de s2. À coordonner avec DevOps.

## Points contrôlés sans remarque

- `unsafe-inline` limité à `style-src`, conformément à D04 et D08 ; aucun `unsafe-inline` ni `unsafe-eval` dans `script-src`.
- `img-src 'self'` sans `blob:` ni hôte du bucket : suffisant pour US-010 (pas de photo) ; à élargir avec la première US qui affiche des photos (D08 § Q5).
- Système de fichiers en lecture seule : le README documente les `emptyDir` sur `/tmp` et `/etc/nginx/conf.d` nécessaires à `envsubst`.
- `escapeValue: false` d'i18next, `?page=` borné par expression régulière, annulation des requêtes par `AbortSignal`, nouvelle tentative limitée et désactivée sur 4xx.

## Synthèse

| Gravité | Nombre | Dev |
|---|---|---|
| Bloquante | 0 | — |
| Majeure | 2 | front (M1, M2) |
| Mineure | 5 | front (m1, m2, m3), testeur (m4, m5) |
| Suggestion | 5 | front, DevOps pour s5 |

Prêt à fusionner : **avec corrections** (M1 et M2).
