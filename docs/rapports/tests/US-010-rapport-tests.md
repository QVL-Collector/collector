# US-010 — Rapport de tests front

| Champ | Valeur |
|---|---|
| US | [US-010 — Consulter le catalogue public sans compte](../../../backlog/US-010-catalogue-public.md) |
| Branche | `feat/US-010-catalogue-public` |
| Date | 2026-10-26 |
| Fonction | Test |
| Périmètre de ce lot | Front (`web/`) : AC1, AC2 (partie front), AC4 (partie front), AC6 |
| Hors périmètre de ce lot | AC3 (amorçage idempotent, Back/DevOps), AC5 (performance de l'API, Back), filtrage serveur et 404 d'AC2 (Back), absence de violation CSP d'AC4 en navigateur réel (E2E Playwright) |
| Outils | Vitest 3.2.4, Testing Library, jsdom 26.1.0, `axe-core` 4.14.0 (ajouté en version exacte) |
| Verdict | **Tous les AC du périmètre front sont couverts et conformes** |

## Méthode

Scénarios Given/When/Then dérivés des AC de l'US, étiquetés `US-010-ACn`, écrits sans lecture du code des composants. L'application complète (`App`, routeur, fournisseurs, i18n) est rendue en boîte noire avec, selon le cas :

- un faux client de catalogue injecté par `AppServices` (états nominal, vide, chargement, erreur, hors limites, titres piégés) ;
- le vrai client HTTP (`catalogue.source = api`) avec `fetch` simulé, pour observer la requête émise ;
- l'adaptateur de démonstration (`catalogue.source = demo`, scénario `nominal`).

Tests : `web/src/acceptance/US-010/`

| Fichier | Contenu |
|---|---|
| `catalogueHarness.tsx` | Rendu de l'application, faux clients, données fictives |
| `AC1-catalogue-pagine.test.tsx` | Liste, pagination, `?page=`, états |
| `AC2-api-publique.test.tsx` | Requête publique sans authentification, affichage strict de la réponse |
| `AC4-titres-neutralises.test.tsx` | Titres `<script>` et `<img onerror>` |
| `AC6-accessibilite.test.tsx` | axe-core, structure, clavier, focus, FR/EN |

## Résultats d'exécution

| Commande (depuis `web/`) | Résultat |
|---|---|
| `npm test` | 4 fichiers, **36 tests verts**, 0 échec (≈ 11 s) ; JUnit dans `test-results/junit.xml` |
| `npm run test:coverage` | Lignes 86,92 %, branches 86,66 %, fonctions 89,06 % ; 100 % des composants et hooks de `features/catalogue` |
| `npm run lint` | 0 erreur, 0 avertissement |
| `npm run typecheck` | 0 erreur |
| `npm run format:check` | Conforme sur une extraction en fins de ligne LF (imposées par `.gitattributes`) ; échec relevé en revue sur une extraction CRLF sous Windows (m5) |

## AC1 — Liste paginée des articles publiés

| Scénario | Attendu (AC) | Réel | Statut |
|---|---|---|---|
| Nominal : 30 articles, ouverture du catalogue | Liste avec titre, prix, catégorie, paginée | Liste « Articles publiés » de 12 éléments ; titre, prix `11,25 €`, catégorie présents ; page 1 demandée | Conforme |
| Choix de la page 2 | Page 2 affichée | Articles 13 à 24 affichés, page 2 demandée, `?page=2` dans l'URL | Conforme |
| Ouverture directe de `?page=3` | Dernière page | 6 articles (25 à 30), bouton « Page 3, page actuelle » avec `aria-current="page"` | Conforme |
| Retour arrière de l'historique | Page précédente réaffichée | Page 1 réaffichée | Conforme |
| Limite : exactement 12 articles | Une seule page | 12 éléments, pas de page 2 proposée | Conforme |
| Limite : `?page=abc`, `0`, `-2`, `1.5` | Pas de page invalide demandée | Page 1 affichée, aucune autre page demandée | Conforme |
| Limite : `?page=99` | État hors limites | « Cette page du catalogue est vide », « Revenir à la première page » ramène à la page 1 | Conforme |
| Chargement | État de chargement | Zone `role="status"` : « Chargement du catalogue en cours » | Conforme |
| Vide | État vide | « Aucun article publié pour le moment », ni liste ni pagination | Conforme |
| Erreur | État d'erreur | « Le catalogue est momentanément indisponible » et bouton « Réessayer » actif | Conforme |
| Erreur puis reprise | Liste après « Réessayer » | Nouvel appel émis, liste affichée | Conforme |

## AC2 — Aucun article non publié (partie front)

| Scénario | Attendu (AC) | Réel | Statut |
|---|---|---|---|
| Visiteur non connecté | Lecture par l'API publique | `GET /marche/v1/articles?page=1…`, sans en-tête `Authorization` | Conforme |
| Réponse de deux articles publiés | Seuls ceux-ci apparaissent | Exactement 2 éléments listés, ceux de la réponse | Conforme |
| Réponse 404 `application/problem+json` | Aucun article affiché | État d'erreur, aucune liste | Conforme |
| Réponse hors contrat | Aucun article affiché | État d'erreur, aucune liste | Conforme |

Le filtrage des statuts brouillon, contrôle, refusé, à vérifier et la réponse 404 de la lecture directe relèvent de l'API (tests d'intégration Back et d'acceptation API) : non couverts par ce lot.

## AC4 — Titre balisé affiché comme du texte (partie front)

| Scénario | Attendu (AC) | Réel | Statut |
|---|---|---|---|
| Titre `<script>window.pwned = true</script>Vase Gallé` | Affiché comme du texte | Texte brut visible, aucun `<script>` dans la liste, `window.pwned` indéfini | Conforme |
| Titre `<img src="x" onerror="window.pwned = true">Timbre rare` | Affiché comme du texte | Texte brut visible, aucun `<img>` ni attribut `onerror`, `window.pwned` indéfini | Conforme |
| Titre piégé parmi des titres ordinaires | Texte | Titre piégé exposé comme intitulé `h2` textuel, 3 éléments | Conforme |
| Jeu de démonstration, toutes les pages | Texte | Au moins 2 titres balisés rencontrés, aucun `script`, `img`, `iframe` ni `onerror` injecté | Conforme |

Non couvert ici : « aucune violation de la politique de sécurité du contenu ». jsdom n'applique pas de CSP ; la preuve relève de l'E2E Playwright sur l'image nginx (US, « Tests attendus »).

## AC6 — Accessibilité

| Scénario | Attendu (AC) | Réel | Statut |
|---|---|---|---|
| axe-core, liste paginée | 0 violation serious ou critical | 0 | Conforme |
| axe-core, état vide | Idem | 0 | Conforme |
| axe-core, état d'erreur | Idem | 0 | Conforme |
| axe-core, état hors limites | Idem | 0 | Conforme |
| axe-core, état de chargement | Idem | 0 | Conforme |
| axe-core, page en anglais | Idem | 0 | Conforme |
| Structure | Titre de document, un seul `h1`, `lang`, zone principale | « Catalogue · Collector », un `h1` « Catalogue », `lang="fr"`, `main` présent | Conforme |
| Premier arrêt Tab | Lien d'évitement | « Aller au contenu principal » focalisé, cible l'`id` du `main` | Conforme |
| Pagination au clavier (Tab puis Entrée sur page 2) | Navigation clavier, focus sur `h1` | Page 2 affichée, focus sur le `h1` | Conforme |
| Retour à la page 1 | Page courante signalée et annoncée | `aria-current="page"`, statut « 30 articles publiés, page 1 sur 3 » | Conforme |
| « Réessayer » au clavier | Focalisable | Atteint par Tab | Conforme |

Limite de l'outil : la règle `color-contrast` est désactivée sous jsdom (pas de rendu ni de canvas) ; le contraste est à vérifier par `@axe-core/playwright` en navigateur réel.

## Internationalisation FR/EN

| Scénario | Attendu | Réel | Statut |
|---|---|---|---|
| Langue anglaise au chargement | Interface en anglais | « Published items », « Catalogue pagination », prix `€11.25`, « 30 published items, page 1 of 3 », `lang="en"` | Conforme |
| Bascule FR vers English | Sans rechargement | Libellés en anglais, bouton « English » `aria-pressed="true"`, `lang="en"`, articles conservés | Conforme |
| Erreur en anglais | Bouton traduit | « Try again » actif | Conforme |

## Anomalies

Aucune anomalie : aucun comportement observé ne contredit un AC du périmètre front.

## Observations (sans anomalie)

- Environnement de test : sous jsdom, l'`AbortSignal` de jsdom est refusé par le `Request` natif de Node lors des navigations du routeur ; le harnais neutralise ce signal pour les seuls tests. Artefact d'environnement, pas un défaut du produit.
- axe-core écrit sur `stderr` « HTMLCanvasElement.prototype.getContext not implemented » : bruit de jsdom, sans effet sur les résultats.
- Les cas `?page=` invalides ne sont pas exigés explicitement par l'AC1 ; ils sont testés comme cas limites et sont conformes.
- `axe-core` 4.14.0 ajouté en `devDependencies` (version exacte) dans `web/package.json` et `web/package-lock.json`.

## Reste à couvrir hors de ce lot

| AC | Preuve attendue | Fonction |
|---|---|---|
| AC2 | Filtrage serveur et 404 en lecture directe (testcontainers-go, Playwright `request`) | Back, Test |
| AC3 | Double exécution de l'amorçage sans doublon | Back, DevOps |
| AC4 | Aucune violation CSP en navigateur réel derrière nginx (Playwright) | Test |
| AC5 | Liste publique < 500 ms à vide (Playwright `request`) | Back, Test |
| AC6 | axe-core en navigateur réel, contraste compris (`@axe-core/playwright`) | Test |
