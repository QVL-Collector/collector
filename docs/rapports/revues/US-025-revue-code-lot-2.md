# US-025 — Revue de code front (lot 2)

| Champ | Valeur |
|---|---|
| US | [US-025 — Utiliser Collector en anglais, avec des écrans accessibles](../../../backlog/US-025-francais-anglais-accessibilite.md) |
| Branche | `feat/US-025-i18n-a11y-lot-2` (base `origin/main`, commits `a71131c`, `4da8670`) |
| Date | 2026-11-06 |
| Périmètre | Lot 2 : Playwright et `@axe-core/playwright` dans `web/` (`playwright.config.ts`, `tsconfig.e2e.json`, `e2e/**`), libellés du sélecteur de langue (`fr.ts`, `en.ts`), 5 tests Vitest adaptés, job `e2e` de `.github/workflows/ci.yml`, dépendances, `docs/processus/cicd.md`, `web/README.md`, [rapport de tests](../tests/US-025-rapport-tests-lot-2.md) |
| Références | [revue du lot 1](US-025-revue-code.md), [D06](../../decisions/D06-authentification-autorisation.md), `docs/processus/cicd.md` (job 11, chaîne protégée) |
| Verdict | **Fusion possible** (0 bloquante, 0 majeure, 3 mineures, 2 suggestions) |

## Vérifications exécutées (depuis `web/`)

| Commande | Résultat |
|---|---|
| `npm run typecheck` (`tsc -b`, inclut désormais `tsconfig.e2e.json`) | 0 erreur |
| `npx eslint . --max-warnings=0` | 0 erreur, 0 avertissement ; les 11 fichiers `e2e/**` et `playwright.config.ts` sont bien analysés (contrôle `-f json`) |
| `npx prettier --check .` | Conforme |
| `npm run i18n:check` | `Traductions alignées : 226 clés, langues fr, en.` |
| Vitest ciblé (`--pool=forks`, 2 forks) : US-008 AC4, US-010 AC6, US-023 AC3, dossier US-025 | 11 fichiers, 131 tests verts |
| `npm run e2e` (`E2E_LANGUES` absent, donc `fr`) | 27 tests verts en 1,1 min (projets `fr` et `bascule-fr-en`) |
| Recherche de libellés `'English'` / `'Français'` restants dans les tests | Aucun (seuls des intitulés de `it(...)` les citent) |
| Recherche de commentaires dans le diff | Aucun |
| Recherche de mentions d'outils d'IA dans le diff et les messages de commit | Aucune ; commits au format Karma |

## Points forts

- Aucun code de production ajouté pour les tests : seuls `src/i18n/locales/fr.ts:9,11` et `en.ts:11,13` changent hors `src/acceptance`. Le faux fournisseur d'identité (`e2e/support/fournisseurIdentite.ts`) et la configuration de démonstration (`e2e/support/collector.ts:12`, `demarrer`) passent exclusivement par `page.route` ; ils ne sont ni importés par `src/` ni inclus dans `dist/` (Vite part de `index.html`, l'image ne copie que `dist/`).
- Jetons non signés sans risque de fuite : autorité sur `auth.collector.e2e.localhost` (TLD réservé, jamais résolu en production), servie uniquement par interception ; `marketplaceApiUrl: '/marche'` pointe vers `vite preview` sans proxy, donc aucun appel vers une API réelle ; l'artefact d'échec ne contient que ces jetons fictifs.
- Sélecteur de langue : correction WCAG 2.5.3 minimale et juste (le nom accessible « FR, Français » commence par le texte visible « FR ») ; tous les consommateurs des libellés recherchés et mis à jour (5 tests Vitest), aucun autre site d'appel (`useLanguageSwitcher.ts:19-20`, `LanguageSwitcher.tsx:38`). README explique pourquoi un code entre parenthèses ne suffit pas.
- Robustesse : aucune attente fixe ; attentes web-first partout (`toHaveText`, `toBeVisible`, `toHaveAttribute`), `waitForURL` sur prédicat de chemin, `attendreRenduStable` (`accessibilite.ts:8`) qui attend la fin des animations finies avant axe (les animations infinies sont exclues, pas de blocage sur un indicateur de chargement). Délais étendus localisés et justifiés (`timeout: 15_000` sur le passage à « Publié » du contrôle simulé).
- `tabulerJusqua` (`accessibilite.ts:77`) vérifie un indicateur de focus réel (outline, box-shadow, ou contour MUI de 2 px) à chaque tabulation et rend un message d'échec qui retrace le parcours : diagnostic immédiat.
- Preuve « sans rechargement » solide : marqueur posé sur `window` et compteur de chargements de `/config.json` (`AC1-bascule-anglais.spec.ts:18`, `:40-41`, `:56`, `:86`).
- Oracle de textes en boîte noire (`e2e/support/textes.ts`), typé par `Textes` : une clé oubliée dans une langue casse la compilation ; duplication avec les fichiers de traduction voulue et documentée (README).
- Configuration : langues validées (`lireLangues` rejette une valeur inconnue au lieu de l'ignorer), un projet par langue, projet `bascule-fr-en` toujours joué, `forbidOnly` en CI, `reuseExistingServer` désactivé en CI, `strictPort`.
- Workflow : `pull_request` seul (pas de `pull_request_target`), `permissions: contents: read` au niveau du job sur un défaut `{}`, `persist-credentials: false`, aucun secret, actions épinglées par SHA identiques au job `web`, `npm ci --ignore-scripts`, `npx --no-install`, artefact limité à `if: failure()` et 14 jours, job ajouté à `ci-ok` (qui accepte `skipped`). `E2E_LANGUES` conforme à AC3 (`fr` en PR, `fr,en` sur `main`, planification et déclenchement manuel). Déclenchement cohérent avec `tools/ci/changements.sh` (`web/*`, `contracts/*`, outillage de CI, tout hors PR).
- Dépendances en versions exactes ; `playwright-core` 1.64.1 en dépendance directe fixe le pair de `@axe-core/playwright` sur la version de `@playwright/test` : une seule entrée `node_modules/playwright-core` dans le verrou.
- Documentation à jour : `docs/processus/cicd.md` (état du job 11), README (commandes, mode démonstration, ajout d'un écran par une entrée `{ nom, ouvrir }`).
- Mineure 2 du lot 1 soldée : les motifs de contrôle traduits sont désormais vérifiés à l'écran admin (`AC1-bascule-anglais.spec.ts:66-73`).

## Issues

### Bloquantes

Aucune.

### Majeures

Aucune.

### Mineures

1. `web/eslint.config.js:41` — `playwright-report` a été ajouté à `.prettierignore` mais pas aux `ignores` ESLint. Après un échec local avec trace (`trace: 'retain-on-failure'`), le rapport HTML embarque la visionneuse de traces (`playwright-report/trace/*.js` et `trace/assets/*.js`, copie constatée dans `node_modules/playwright/lib/runner/index.js:3840-3853`) ; `npx eslint .` analyserait alors ce JavaScript minifié. Impact local uniquement (le job `web` part d'un clone propre). Ajouter `'playwright-report'` à la liste `ignores`. Dev : front.
2. `web/e2e/support/accessibilite.ts:77-96` — `tabulerJusqua` retourne `IndicateurDeFocus[]`, valeur ignorée par ses 9 appelants (`AC5-mise-en-vente-clavier.spec.ts`). Le tableau `parcourus` reste utile pour le message d'échec, mais le retour est mort : passer le type de retour à `Promise<void>`. Dev : front.
3. Exports sans consommateur hors de leur fichier : `tagsWcag21AA` (`accessibilite.ts:4`), `attendreRenduStable` (`accessibilite.ts:8`), `indicateurDeFocus` (`accessibilite.ts:35`), `configurationDemo` (`collector.ts:12`), `languesE2e` (`langues.ts:1`). Retirer le mot-clé `export` (les types `IndicateurDeFocus`, `Sens` et `Ecran` restent exportés, ils typent des signatures publiques). Dev : front.

### Suggestions

1. `web/e2e/support/collector.ts:143` et `web/e2e/support/fournisseurIdentite.ts:104` — le compte vendeur par défaut est posé deux fois (initialiseur de `FournisseurIdentite.compte` et `collector.seConnecterEn(vendeur)` dans la fixture). Garder un seul des deux. Dans le même fichier, `type OptionsCollector` puis `export type { OptionsCollector }` (`collector.ts:134-136`) peut s'écrire `export type OptionsCollector = …`. Dev : front.
2. `web/e2e/support/collector.ts:125-127` — `boutonDeDecision` construit `new RegExp(`^${libelle}`)` sans échappement alors que `contenant` (`collector.ts:32`) échappe. Sans effet avec les libellés actuels (« Publier », « Refuser », « Publish », « Reject ») ; réutiliser l'échappement de `contenant` évite une surprise si un libellé contient un point ou une parenthèse. Dev : front.

## Verdict

| Sévérité | Nombre |
|---|---|
| Bloquante | 0 |
| Majeure | 0 |
| Mineure | 3 |
| Suggestion | 2 |

**Fusion possible pour le lot 2.** Les AC1, AC3, AC4 et AC5 (partie automatisée) sont couverts par des parcours verts ; la vérification manuelle tracée d'AC5 reste prévue en fin de sprint. Les mineures ne bloquent pas et peuvent être soldées dans un commit de nettoyage.
