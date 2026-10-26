# US-025 — Rapport de tests front, lot 1

| Champ | Valeur |
|---|---|
| US | [US-025 — Utiliser Collector en anglais, avec des écrans accessibles](../../../backlog/US-025-francais-anglais-accessibilite.md) |
| Branche | `feat/US-025-francais-anglais-accessibilite` |
| Date | 2026-10-26 |
| Fonction | Test |
| Périmètre du lot 1 | Écrans existants : catalogue (tous états), connexion (en-tête, rappel réussi et échoué), espace vendeur, pages d'erreur (accès refusé, page introuvable, échec de démarrage) ; script de parité, lint anti-texte en dur, contrastes du thème, focus visible |
| Reporté au lot 2 | Fiche article (US-023), mise en vente (US-013 front), écran admin (US-022) : écrans inexistants à ce jour |
| Outils | Vitest 3.2.4, Testing Library, jsdom 26.1.0, `axe-core` 4.14.0, ESLint 9.35.0, Node 24 |
| Verdict | **Tous les AC du périmètre du lot 1 sont couverts et conformes** |

## Méthode

Scénarios Given/When/Then dérivés des AC, étiquetés `US-025-ACn`. L'application complète est démarrée en boîte noire par les harnais existants, réutilisés sans modification : `US-008/identityProviderHarness.tsx` (démarrage réel, `fetch` et fournisseur d'identité simulés, rechargement de page simulé au retour sur `/rappel`) et `US-010/catalogueHarness.tsx` (faux client de catalogue pour les états vide, erreur, hors limites, chargement).

Tests : `web/src/acceptance/US-025/`

| Fichier | Contenu | Tests |
|---|---|---|
| `AC1-langue-conservee-connexion.test.tsx` | `ui_locales` transmis ; langue conservée après connexion réussie, échouée, `state` forgé ; cas inverse navigateur EN / choix FR | 9 |
| `AC1-bascule-sans-rechargement.test.tsx` | Passage en anglais sur page introuvable, accès refusé, espace vendeur ; conservation lors d'une navigation interne | 4 |
| `AC1-motifs-de-refus-traduits.test.tsx` | Motifs du contrat `controle.resultat` v1 : liste identique au site, libellés FR et EN non vides, traduits, réactifs au changement de langue | 6 |
| `AC2-parite-et-texte-en-dur.test.ts` | Script de parité sur les vraies traductions puis sur des traductions temporaires défectueuses ; lint de composants sondes | 11 |
| `AC4-contrastes-wcag.test.ts` | Implémentation de référence du rapport WCAG écrite dans le test (vérifiée sur valeurs connues), appliquée à chaque couple de `themeContrastPairs` | 30 |
| `AC4-axe-ecrans-fr-en.test.tsx` | axe-core sur 7 écrans × FR/EN | 14 |
| `AC4-axe-etats-catalogue-fr-en.test.tsx` | axe-core sur 5 états du catalogue × FR/EN | 10 |
| `AC5-focus-visible-mouvements-reduits.test.tsx` | Anneau de focus après tabulation, avec et sans `prefers-reduced-motion: reduce` | 7 |

## Résultats d'exécution

| Commande (depuis `web/`) | Résultat |
|---|---|
| `npm test` | 19 fichiers, **187 tests verts** (97 existants inchangés + 90 nouveaux), 0 échec |
| `npm run lint` | 0 erreur, 0 avertissement |
| `npm run typecheck` | 0 erreur |
| `npm run i18n:check` | « Traductions alignées : 70 clés, langues fr, en. », code de sortie 0 |
| `prettier --check src/acceptance/US-025` | Conforme |

## Résultats par AC

### AC1 — Anglais partout, sans rechargement, conservé pendant la session

| Scénario | Attendu (AC) | Réel | Statut |
|---|---|---|---|
| Nominal : anglais choisi puis « Sign in » | Choix conservé pendant la session | `ui_locales=en` dans la requête d'autorisation | OK |
| Nominal : français | Idem | `ui_locales=fr` | OK |
| Limite : EN puis retour FR avant connexion | Dernier choix retenu | `ui_locales=fr` | OK |
| Nominal : navigateur FR, choix EN, connexion réussie | Anglais conservé | En-tête « Seller », « Sign out », bouton English pressé, `lang="en"` | OK |
| Nominal : EN choisi sur la page 2, connexion réussie | Anglais conservé, page conservée | Page 2, « Published items », « 30 published items, page 2 of 3 » | OK |
| Erreur : EN choisi, connexion refusée (`access_denied`) | Message d'erreur en anglais | « Sign-in did not complete », `lang="en"`, aucun texte français | OK |
| Erreur : `state` forgé | Pas de session, écran d'échec | Écran d'échec, aucun jeton émis | OK |
| Inverse : navigateur EN, choix FR, connexion réussie | Français conservé | « Vendeur », `lang="fr"` | OK |
| Bascule sur page introuvable | Anglais sans rechargement | Titre, texte et titre de document en anglais ; aucun nouveau chargement de `/config.json` | OK |
| Bascule sur accès refusé et espace vendeur | Messages en anglais | « Restricted access », « Seller area », « Sign out » | OK |
| Navigation interne après bascule | Choix conservé | Catalogue en anglais après « Back to the catalogue » | OK |
| Motifs de refus du contrat | Motifs en anglais | 6 motifs du contrat = 6 motifs du site ; libellés FR et EN non vides, jamais la clé brute, EN différent de FR, suivent le changement de langue | OK |
| Motifs affichés sur les écrans fiche, mise en vente, admin | — | Écrans inexistants | **Lot 2** |

Observation de test (non bloquante) : après un retour réussi sur `/rappel`, l'attribut `lang` de `<html>` est mis à jour quelques instants après le rendu des textes anglais ; l'assertion attend donc la valeur finale.

### AC2 — Parité et texte en dur bloqués en CI

| Scénario | Attendu (AC) | Réel | Statut |
|---|---|---|---|
| Traductions réelles | Script vert | Code 0 | OK |
| Traductions temporaires complètes | Script vert | Code 0 | OK |
| Clé absente en anglais | Script bloque | Code ≠ 0, clé `catalogue.title` citée | OK |
| Clé imbriquée absente en français | Script bloque | Code ≠ 0, clé `catalogue.empty.title` citée | OK |
| Section entière absente | Script bloque | Code ≠ 0 | OK |
| Texte vide (espaces) | Bloqué (README) | Code ≠ 0 | OK |
| Variable `{{…}}` différente | Bloqué (README) | Code ≠ 0 | OK |
| Composant traduit | Lint vert | Aucun message | OK |
| Texte JSX en dur, `aria-label` en dur, `alt` en dur | Lint bloque | Erreur `i18next/no-literal-string` ou `no-restricted-syntax`, sans erreur d'analyse | OK |

Le script de parité n'exporte pas de fonction : il est exécuté sur une copie placée dans un dossier temporaire avec des fichiers `fr.ts`/`en.ts` fabriqués. Les composants sondes du lint sont écrits dans `src/features/` le temps du test puis supprimés. Le branchement effectif de ces deux commandes dans le pipeline CI n'est pas vérifiable ici (périmètre DevOps).

### AC3 — Parcours joué en français sur PR, en français et en anglais sur `main`

| Scénario | Attendu (AC) | Réel | Statut |
|---|---|---|---|
| Parcours de recette FR/EN | Exécution selon la branche | Les tests du lot sont joués dans les deux langues à chaque exécution | Partiel |
| Sélection de la langue selon PR / `main` dans le pipeline | — | Configuration CI hors du front ; parcours Playwright non livré | **Lot 2** (DevOps) |

### AC4 — axe-core sans violation serious ou critical

| Écran | FR | EN | Statut |
|---|---|---|---|
| Catalogue visiteur | 0 | 0 | OK |
| Catalogue vendeur connecté | 0 | 0 | OK |
| Catalogue vide, erreur, hors limites, chargement | 0 | 0 | OK |
| Espace vendeur | 0 | 0 | OK |
| Accès refusé | 0 | 0 | OK |
| Échec de connexion | 0 | 0 | OK |
| Page introuvable | 0 | 0 | OK |
| Échec de démarrage | 0 | 0 | OK |
| Contrastes : 22 couples de `themeContrastPairs` | Texte ≥ 4,5:1, non-texte ≥ 3:1 | Tous au-dessus du seuil (référence indépendante de `contrast.ts`) ; seuils déclarés = WCAG AA | OK |
| Fiche, admin | — | Écrans inexistants | **Lot 2** |

La règle axe `color-contrast` est désactivée sous jsdom (pas de rendu) ; elle est compensée par le test des couples. La porte bloquante en CI via `@axe-core/playwright` (outil prévu par l'US) reste à brancher.

### AC5 — Parcours de mise en vente au clavier, focus visible

| Scénario | Attendu (AC) | Réel | Statut |
|---|---|---|---|
| « Se connecter » au clavier, mouvements réduits et normaux | Focus visible | Absent avant focus ; `outline` solide de 0,1875 rem (3 px) après tabulation | OK |
| Bouton « English » au clavier, deux préférences | Focus visible | Anneau présent | OK |
| Lien d'évitement, pagination, « Se déconnecter » (mouvements réduits) | Focus visible | Anneau présent | OK |
| Parcours de mise en vente de bout en bout au clavier | — | Écran inexistant | **Lot 2** |
| Vérification manuelle tracée en fin de sprint | — | À réaliser sur navigateur réel | **Lot 2** |

Limite jsdom : la feuille de style Emotion est bien appliquée et `getComputedStyle` expose le raccourci `outline`, mais pas le rendu réel ; la visibilité perçue reste à confirmer lors du contrôle manuel.

## Anomalies

Aucune anomalie constatée sur le périmètre du lot 1.

## Points reportés au lot 2

- Écrans fiche (US-023), mise en vente (US-013 front), admin (US-022) : AC1 (traduction, motifs de refus affichés), AC4 (axe FR/EN), AC5 (parcours clavier de bout en bout).
- AC3 et porte AC4 : parcours Playwright `fr`/`en` et `@axe-core/playwright` dans la CI, sélection de langue selon PR / `main`.
- AC5 : contrôle manuel au clavier tracé en fin de sprint.

## Verdict

Tous les AC sont couverts et conformes sur le périmètre du lot 1 ; AC3 et les écrans non encore livrés restent à tester au lot 2.
