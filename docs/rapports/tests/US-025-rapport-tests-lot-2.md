# US-025 — Rapport de tests front, lot 2

| Champ | Valeur |
|---|---|
| US | [US-025 — Utiliser Collector en anglais, avec des écrans accessibles](../../../backlog/US-025-francais-anglais-accessibilite.md) |
| Branche | `feat/US-025-i18n-a11y-lot-2` (depuis `main`) |
| Date | 2026-11-06 |
| Fonction | Test |
| Lot précédent | [Rapport du lot 1](US-025-rapport-tests.md) (écrans existants, parité, lint, contrastes du thème sous jsdom) |
| Périmètre du lot 2 | Parcours de bout en bout dans un vrai navigateur sur le build de démonstration : catalogue, filtre, fiche (US-023), connexion vendeur, création, soumission et suivi d'un brouillon (US-013), modification d'un article publié (US-021), écran admin « Articles à vérifier » (US-022 : publication, refus avec motif, conflit 409, accès réservé), pages d'erreur ; porte axe-core et parcours dans la CI |
| Outils | Playwright 1.64.1 (Chromium seul), `@axe-core/playwright` 4.14.0 (même `axe-core` 4.14.0 que le lot 1), Vitest 3.2.4, Node 24 |
| Verdict | **AC1, AC3, AC4 et AC5 couverts et conformes sur le périmètre du lot 2** ; 1 violation axe réelle trouvée et corrigée ; contrôle manuel au clavier d'AC5 à tracer par Martin en fin de sprint |

## Méthode

Scénarios dérivés des AC, étiquetés `US-025-ACn`, joués en boîte noire contre le build de production (`npm run build`, puis `vite preview` lancé par le `webServer` de Playwright).

- Mode démonstration : `/config.json` est remplacé par une configuration `catalogue.source: demo` ; les données viennent des adaptateurs de démonstration du catalogue, de l'espace vendeur et de l'écran admin (titres balisés, article qui répond 409 à la première décision).
- Connexion : le fournisseur d'identité est simulé par interception réseau (`web/e2e/support/fournisseurIdentite.ts`) : document de découverte, redirection d'autorisation vers `/rappel` avec un code, jetons non signés portant le rôle `vendeur`, `admin` ou `acheteur` (accès refusé). Le second facteur de l'administrateur (OTP) est exigé par Keycloak, hors du périmètre de ces tests. Le vrai client OIDC de l'application (PKCE, `state`, retour sur la page d'origine) est exercé ; aucun code de production ne simule la connexion.
- Langues : un projet Playwright par langue de `E2E_LANGUES` (`fr` par défaut, `fr,en` sur `main`) ; le navigateur est en `fr-FR`. Le parcours AC3 choisit sa langue par le sélecteur de l'en-tête ; les écrans AC4 et le parcours AC5 s'ouvrent avec un navigateur réglé sur la langue du projet (indispensable pour l'écran d'échec de démarrage, qui n'a pas de sélecteur). Le projet `bascule-fr-en` joue toujours le passage du français à l'anglais (AC1).
- Oracle : les textes attendus sont écrits en dur dans `web/e2e/support/textes.ts`, indépendamment des fichiers de traduction.

Tests : `web/e2e/US-025/`

| Fichier | Contenu | Tests par langue |
|---|---|---|
| `AC3-parcours-mise-en-vente.spec.ts` | Parcours complet jusqu'à « Publié » ; modification d'un article publié avec avertissement (US-021) ; administrateur : publication et refus avec motif (US-022) ; vendeur et acheteur refusés sur `/admin` | 5 |
| `AC4-axe-ecrans.spec.ts` | axe-core sur chaque écran de `e2e/support/ecrans.ts` (18 écrans) | 18 |
| `AC5-mise-en-vente-clavier.spec.ts` | Mise en vente au clavier seul, indicateur de focus contrôlé à chaque tabulation | 1 |
| `AC1-bascule-anglais.spec.ts` (projet `bascule-fr-en`) | Passage en anglais sans rechargement : espace vendeur (textes, erreur, motif) ; fiche introuvable ; articles à vérifier (motifs, dialogue de refus, conflit) | 3 (une fois) |

## Résultats d'exécution

| Commande (depuis `web/`) | Résultat |
|---|---|
| `npx playwright install chromium` | Chromium téléchargé |
| `npm run e2e` (`E2E_LANGUES` absent, donc `fr`) | **27 tests verts** (24 `fr`, 3 `bascule-fr-en`), 0 échec, 0 instable (≈ 1,2 min, build compris) |
| `E2E_LANGUES=fr,en npm run e2e` | **51 tests verts** (24 `fr`, 24 `en`, 3 `bascule-fr-en`), 0 échec, 0 instable (≈ 1,9 min) |
| `npx vitest run --pool=forks --poolOptions.forks.maxForks=2` | 30 fichiers, **347 tests verts** |
| `npm run typecheck` | 0 erreur (le projet `tsconfig.e2e.json` couvre `e2e/` et `playwright.config.ts`) |
| `npx eslint . --max-warnings 0` | 0 erreur, 0 avertissement |
| `npx prettier --check .` | Conforme |
| `npm run i18n:check` | « Traductions alignées : 226 clés, langues fr, en. » |
| `bash ../tools/ci/verifier-eslint.sh` | Règles obligatoires actives |
| `docker compose -f tools/ci/outils.yml run --rm actionlint -color` | Code 0, aucun constat |
| `zizmor --offline --persona=regular .github/workflows` (même image) | « No findings to report » (audits en ligne joués en CI avec `GH_TOKEN`) |

## Résultats par AC

### AC1 — Anglais partout, sans rechargement, conservé pendant la session

| Scénario | Attendu (AC) | Réel | Statut |
|---|---|---|---|
| Formulaire de nouvel article en erreur (prix « abc »), puis « EN » | Textes clés et message d'erreur en anglais, sans rechargement | Titre « New item », titre du document « New item · Collector », « Invalid value. Amount example: 12.50. », « The form contains 1 error: fix the highlighted field. », bouton « Create the draft », rôle « Seller », « Sign out » ; la saisie « abc » est conservée | OK |
| Navigation interne vers « My items » | Choix conservé | « Seller area », « My items » | OK |
| Article refusé dans la liste | Motif de refus en anglais | « Rejected », « Reason: Incomplete listing » ; aucune occurrence de « Motif : Annonce incomplète » | OK |
| Absence de rechargement | Aucun rechargement | Marqueur posé dans `window` avant la bascule toujours présent ; `/config.json` non rechargé | OK |
| Fiche introuvable, puis « EN » | Message d'erreur en anglais | « Item not found », « Sign in » ; retour au catalogue en anglais sans rechargement | OK |
| Langue conservée à travers la connexion (parcours AC3 en `en`) | Choix conservé pendant la session | Anglais choisi sur le catalogue, conservé après l'aller-retour chez le fournisseur d'identité (`lang="en"`, « Seller ») | OK |
| Articles à vérifier, puis « EN » | Motifs et messages d'erreur en anglais, sans rechargement | Titre « Items to review », « Review reason: Price to be confirmed » (plus aucun « Motif du contrôle : Prix à confirmer ») ; décision en conflit : « The item has changed, reload the list » ; dialogue de refus « Rejection reason », motif « Excessive price », annonce « … has been rejected. » ; marqueur sans rechargement présent | OK |

### AC3 — Parcours en français sur PR, en français et en anglais sur `main`

| Étape du parcours | `fr` | `en` | Statut |
|---|---|---|---|
| Catalogue, choix de la langue par le sélecteur | « Catalogue » | « Catalogue » | OK |
| Filtre « Montres » | `?categorie=montres`, annonce « … dans la catégorie Montres » | « … in the Montres category » | OK |
| Fiche du premier article | `h1` = titre de la carte, « Vendeur particulier » | « Private seller » | OK |
| Connexion vendeur (démonstration) | Retour sur la fiche, « Vendeur » dans l'en-tête | « Seller » | OK |
| Espace vendeur, « Nouvel article », création d'un brouillon complet | État « Brouillon » | « Draft » | OK |
| Soumission | « En contrôle » puis « Publié » (contrôle simulé) | « Under review » puis « Published » | OK |
| Liste « Mes articles » | Le nouvel article est « Publié » | « Published » | OK |
| Modification d'un article publié (US-021) | Avertissement « L'annonce sera retirée du catalogue » en `role="note"`, description accessible du bouton « Enregistrer et renvoyer au contrôle », succès « révision 3 : l'annonce repasse au contrôle », « En contrôle » puis « Publié » | Équivalents anglais | OK |
| Administrateur (US-022) : lien « Articles à vérifier » de l'en-tête, « Publier » une figurine, « Refuser » une carte | Annonce « … a été publié. », carte retirée ; confirmation sans motif refusée (« Choisissez un motif de refus. »), motif « Prix abusif », annonce « … a été refusé. », carte retirée ; carte restante « Motif du contrôle : Prix à confirmer » | « … has been published. », « Choose a rejection reason. », « Excessive price », « … has been rejected. », « Review reason: Price to be confirmed » | OK |
| Vendeur, puis acheteur, sur `/admin` | « Accès réservé », aucun lien « Articles à vérifier » dans l'en-tête | « Restricted access » | OK |
| Sélection de la langue selon l'événement | `E2E_LANGUES=fr` en PR | `fr,en` sur push vers `main` (et planification, déclenchement manuel) | OK (configuration ; première exécution à constater sur la PR) |

### AC4 — axe-core : aucune violation serious ou critical, porte bloquante

Tags `wcag2a`, `wcag2aa`, `wcag21a`, `wcag21aa` ; règle `color-contrast` active (vrai rendu Chromium) ; analyse après la fin des animations finies.

| Écran | FR | EN | Statut |
|---|---|---|---|
| Catalogue (visiteur) | 0 | 0 | OK |
| Catalogue filtré par catégorie | 0 | 0 | OK |
| Catalogue, catégorie inconnue | 0 | 0 | OK |
| Fiche article | 0 | 0 | OK |
| Fiche article introuvable | 0 | 0 | OK |
| En-tête connecté (vendeur) sur le catalogue | 0 | 0 | OK |
| Connexion échouée (`/rappel`) | 0 | 0 | OK |
| Espace vendeur, liste | 0 | 0 | OK |
| Espace vendeur, nouveau brouillon | 0 | 0 | OK |
| Espace vendeur, formulaire en erreur | 0 | 0 | OK |
| Espace vendeur, article publié (avertissement) | 0 | 0 | OK |
| Espace vendeur, article refusé (motif) | 0 | 0 | OK |
| Accès réservé (compte acheteur) | 0 | 0 | OK |
| Page introuvable | 0 | 0 | OK |
| Échec de démarrage | 0 | 0 | OK |
| Admin, articles à vérifier (titres balisés affichés en texte) | 0 | 0 | OK |
| Admin, dialogue de refus ouvert | 0 | 0 | OK |
| Admin, décision en conflit (409) | 0 | 0 | OK |

Porte : job `e2e` de `.github/workflows/ci.yml`, ajouté aux dépendances de `ci-ok` ; il s'exécute quand le filtre `changements` retient le front (`web/**`, `contracts/**`, workflows et outillage de CI, tout hors PR).

### AC5 — Mise en vente au clavier seul, focus visible

| Étape (aucun clic, touches Tab, Maj+Tab, Entrée, Espace et saisie) | Attendu | Réel | Statut |
|---|---|---|---|
| Tab jusqu'à « Se connecter », Entrée | Connexion | Session ouverte, focus sur le `h1` du catalogue | OK |
| Maj+Tab jusqu'à « Espace vendeur », Entrée | Espace vendeur | Focus sur le `h1` « Espace vendeur » | OK |
| Tab jusqu'à « Nouvel article », Entrée | Formulaire | Focus sur le `h1` « Nouvel article » | OK |
| Titre, description, catégorie (saisie « Figurines » sur la liste native), prix | Champs remplis | Valeurs saisies, catégorie `figurines` | OK |
| Tab jusqu'à « Créer le brouillon », Entrée | Brouillon créé | État « Brouillon » | OK |
| Tab jusqu'à « Soumettre au contrôle », Espace | Soumission | « Publié » | OK |
| Chaque élément atteint par tabulation | Indicateur visible | `outline` 3 px calculé (boutons, liens, cartes) ou contour de champ MUI focalisé d'au moins 2 px (champs texte et listes) ; jamais aucun | OK |

Observation (comportement voulu, non bloquant) : après une navigation, le focus est placé sur le `h1` de la page ; pour revenir à l'en-tête, l'utilisateur au clavier remonte par Maj+Tab.

## Violations trouvées et corrigées

| Règle axe (impact) | Critère WCAG | Écrans | Cause | Correction |
|---|---|---|---|---|
| `label-content-name-mismatch` (serious) | 2.5.3 Étiquette dans le nom | Tous les écrans avec en-tête | Boutons du sélecteur de langue affichant « FR » et « EN » avec un nom accessible « Français » et « English » qui ne contient pas le texte visible | Libellés `app.language.frFull` et `enFull` remplacés par « FR, Français » et « EN, English » dans `fr.ts` et `en.ts` (le nom commence par le texte visible). Un code entre parenthèses (« Français (FR) ») ne suffit pas : axe ignore les parenthèses. Requêtes par nom mises à jour dans cinq tests Vitest (US-010, US-023, US-025) |

Aucune violation sur les trois écrans admin. Aucune violation de contraste (`color-contrast`) sur les 18 écrans, en français comme en anglais : les couples de jetons vérifiés au lot 1 tiennent dans le rendu réel.

## Revue du lot 1 : mineures

| Mineure | État |
|---|---|
| 1. `no-restricted-syntax` contournable par composition | Déjà soldée sur `main` (`eslint.config.js` : sélecteurs `ConditionalExpression`, `LogicalExpression`, `BinaryExpression`) |
| 2. `useControlReasonLabel` et `controlReason` sans consommateur en production | Déjà soldée sur `main` : le hook, devenu `useArticleReasonLabel` avec US-022, sert l'espace vendeur (`useArticleLabels`) et l'écran admin (motif du contrôle, motifs de refus) ; prouvé en navigateur par AC1 (« Reason: Incomplete listing », « Review reason: Price to be confirmed », « Excessive price ») et AC4 |

## Contrôle manuel au clavier (AC5), à tracer par Martin en fin de sprint

À jouer sur un vrai navigateur (Chrome ou Firefox), build de démonstration (`npm run build`, puis `npm run preview`), souris débranchée ou inutilisée, en français puis en anglais.

| # | Étape | Attendu | FR | EN |
|---|---|---|---|---|
| 1 | Tab depuis le haut du catalogue | Le lien d'évitement apparaît, focus visible | ☐ | ☐ |
| 2 | Sélecteur de langue (Tab, Entrée) | Bascule sans rechargement, anneau visible | ☐ | ☐ |
| 3 | Filtre « Catégorie » (flèches) puis une carte (Entrée) | Fiche ouverte, focus sur le titre | ☐ | ☐ |
| 4 | « Se connecter », retour authentifié | Focus sur le titre de la page | ☐ | ☐ |
| 5 | Maj+Tab jusqu'à « Espace vendeur », Entrée | Espace vendeur, focus sur le titre | ☐ | ☐ |
| 6 | « Nouvel article » : remplir tous les champs, « Créer le brouillon » | Brouillon créé, focus et contour des champs visibles | ☐ | ☐ |
| 7 | Prix invalide puis enregistrement | Focus sur le champ en erreur, résumé annoncé | ☐ | ☐ |
| 8 | « Soumettre au contrôle » (Espace) | État annoncé, « Publié » après le contrôle | ☐ | ☐ |
| 9 | Ouvrir un article publié, modifier, « Enregistrer et renvoyer au contrôle » | Avertissement lu avec le bouton, retour au contrôle annoncé | ☐ | ☐ |
| 10 | Administrateur : « Articles à vérifier », « Refuser » puis motif au clavier (flèches dans le groupe de boutons radio), « Confirmer le refus » ; Échap ferme le dialogue et rend le focus au bouton | Focus piégé dans le dialogue puis restitué, décision annoncée | ☐ | ☐ |
| 11 | Avec lecteur d'écran (NVDA) sur les étapes 2, 6, 8 et 10 | Noms « FR, Français » / « EN, English », erreurs et changements d'état annoncés | ☐ | ☐ |

| Champ | Valeur |
|---|---|
| Date du contrôle | AAAA-MM-JJ |
| Navigateur et version | |
| Testeur | Martin |
| Résultat | ☐ conforme ☐ écarts (à décrire) |

## Anomalies

Aucune anomalie ouverte. La violation `label-content-name-mismatch` est corrigée dans ce lot.

## Points restants

- Première exécution du job `e2e` en CI à constater sur la PR (`fr`), puis sur `main` (`fr,en`).
- Contrôle manuel au clavier d'AC5 ci-dessus.
- Job 11 de `docs/processus/cicd.md` : tests d'API sur pile compose, hors US-025.

## Verdict

AC1, AC3, AC4 et AC5 sont couverts par des tests de bout en bout verts en français et en anglais ; la porte axe-core est bloquante via `ci-ok`. Écran admin (US-022) compris. Reste le contrôle manuel au clavier de fin de sprint.
