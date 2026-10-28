# Collector — front web

Application monopage statique : React 19, TypeScript strict, MUI 7, Vite, React Router, TanStack Query, react-i18next ([D04](../docs/decisions/D04-stack.md)). Servie par `nginx-unprivileged` sur la même origine que les API ([D08](../docs/decisions/D08-exposition-tls-domaine.md)).

## Commandes

Node 24 et npm 11. Dépendances en versions exactes.

| Commande                          | Effet                                                                |
| --------------------------------- | -------------------------------------------------------------------- |
| `npm ci --ignore-scripts`         | Installation verrouillée                                             |
| `npm run dev`                     | Serveur de développement sur `http://localhost:5173` (jamais exposé) |
| `npm run typecheck`               | Vérification des types                                               |
| `npm run lint`                    | ESLint, zéro avertissement toléré                                    |
| `npm run format:check`            | Prettier                                                             |
| `npm run i18n:check`              | Parité des traductions FR/EN (clés, textes vides, variables)         |
| `npm test`                        | Vitest (jsdom), rapport JUnit dans `test-results/`                   |
| `npm run test:coverage`           | Couverture v8 (`text`, `json-summary`, `lcov`)                       |
| `npm run build`                   | Types puis build de production dans `dist/`                          |
| `docker build -t collector-web .` | Image nginx non root, port 8080                                      |

## Organisation

| Dossier                                | Contenu                                                                                                       |
| -------------------------------------- | ------------------------------------------------------------------------------------------------------------- |
| `src/app/`                             | Démarrage, fournisseurs, routes, mise en page, pages techniques                                               |
| `src/features/<écran>/`                | `domain/` (types), `api/` (client typé et adaptateurs), `hooks/` (logique), `components/` (présentation pure) |
| `src/ui/`                              | Thème (`themeVariables`, seules couleurs du code), composants et hooks d'interface réutilisables, animations  |
| `src/i18n/`                            | Langues, traductions typées, formats `Intl`                                                                   |
| `src/api/`, `src/config/`, `src/auth/` | Client HTTP, configuration d'exécution, session                                                               |
| `nginx/`                               | Configuration nginx et en-têtes de sécurité                                                                   |

Un composant ne contient que du JSX ; état, effets et appels vivent dans un hook `useXxx` du même domaine.

## Configuration d'exécution

`/config.json` est lu au démarrage, jamais figé au build (ENF-29). En production, il est monté depuis un ConfigMap ; `public/config.json` sert au développement.

| Clé                      | Valeurs                                                                 | Rôle                                                                               |
| ------------------------ | ----------------------------------------------------------------------- | ---------------------------------------------------------------------------------- |
| `marketplaceApiUrl`      | chemin de même origine, ex. `/marche`                                   | Préfixe de la Place de marché ; une URL absolue est refusée                        |
| `catalogue.source`       | `api`, `demo`                                                           | Client HTTP réel ou adaptateur de démonstration (catalogue et espace vendeur)      |
| `catalogue.demoScenario` | `nominal`, `empty`, `error`                                             | États du catalogue rejouables sans back                                            |
| `oidc.authority`         | URL de l'émetteur, ex. `https://auth.<domaine>/realms/collector`        | HTTPS obligatoire (HTTP admis sur `localhost` seulement), sans requête ni fragment |
| `oidc.clientId`          | `collector-web`                                                         | Client public du realm                                                             |
| `oidc.scope`             | portées séparées par une espace, `openid` obligatoire ; défaut `openid` | `profile` requis pour afficher le nom (AC1 de US-008)                              |

Une configuration invalide affiche l'écran d'échec de démarrage. L'origine de `oidc.authority` doit être celle de `COLLECTOR_IDP_ORIGIN` (CSP).

## Catalogue public (US-010, US-023)

Contrat : `contracts/api/place-de-marche/v1.openapi.yaml`, routes publiques sans authentification, isolées dans `src/features/catalogue/api/` (`cataloguePayload.ts` valide chaque réponse champ par champ ; un écart produit l'état d'erreur, pas de nouvel essai sur 4xx).

- `/` : catalogue paginé (`GET /v1/articles?page=&taille=12`), filtre par catégorie (`GET /v1/categories`, `?categorie=` combiné à `?page=`) ; chaque carte mène à sa fiche.
- `/articles/:id` : fiche (`GET /v1/articles/{id}`) avec titre, prix, frais de port (« non renseignés » si `null`), catégorie (lien vers le catalogue filtré), description et « Vendeur particulier ».
- Filtre : liste native au clavier, libellé « Catégorie » ; un choix pousse une entrée d'historique (retour arrière vers le filtre précédent), revient à la page 1 et garde le focus sur la liste ; résultat annoncé dans la zone `role="status"`. Un identifiant hors motif du contrat est ignoré ; un 400 `categorie_inconnue` affiche « Cette catégorie n'existe pas » (US-023).
- Fiche : description en texte brut (sauts de ligne conservés, aucun HTML interprété) ; un 404 affiche « Article introuvable » (tout autre échec : état d'erreur avec « Réessayer ») en `h1`, focus sur le titre. « Retour au catalogue » restaure le filtre et la page d'origine (état de navigation, US-023).
- Démonstration : catégories tirées du jeu d'articles, 400 `categorie_inconnue` et 404 simulés ; une description contient du balisage.
- Photos : hors périmètre tant qu'US-020 n'est pas livrée.

## Espace vendeur (US-009, US-013)

Contrat : `contracts/api/place-de-marche/v1.openapi.yaml`, isolé dans `src/features/seller/api/` (`sellerArticlePayload.ts` valide chaque réponse).

| Route                       | Écran                                                                                                   |
| --------------------------- | ------------------------------------------------------------------------------------------------------- |
| `/vendeur`                  | « Mes articles » (`GET /v1/vendeur/articles`), état, motif traduit, date de modification                |
| `/vendeur/articles/nouveau` | Création d'un brouillon (`POST /v1/vendeur/articles`), catégories de `GET /v1/categories`               |
| `/vendeur/articles/:id`     | Modification (`PUT`, révision lue envoyée), « Soumettre au contrôle » (`POST …/soumission`), état suivi |

- Erreurs `application/problem+json` lues par `src/api/problemDetails.ts` : `erreurs[].champ` est rattaché au champ du formulaire (message par `code`, focus sur le premier champ en erreur, résumé en `role="alert"`).
- 409 `revision-perimee` : rien n'est écrasé, l'écran explique le conflit et propose de recharger la dernière version ; 409 `article-non-modifiable` ou `article-non-soumettable` : l'article est relu.
- Soumission bloquée tant que des modifications ne sont pas enregistrées. Pendant `en_controle`, l'article est relu toutes les 2 s ; chaque changement d'état est annoncé dans une zone `role="status"` (AC10 de US-013).
- Démonstration (`source: demo`) : faux client vendeur en mémoire, contrôle simulé en 2,5 s (incomplet → refusé, catégorie « Objets divers » → à vérifier, sinon publié), et un brouillon « Album de timbres » qui répond 409 à la première écriture pour rejouer le conflit.

### Modification après publication (US-021)

- Tous les états sont modifiables. Un article `publie`, `en_controle` ou `a_verifier` enregistré repasse en `en_controle` (motif effacé, révision + 1) ; un brouillon ou un refusé reste dans son état. La soumission reste réservée à `brouillon` et `refuse`.
- Avant d'enregistrer, un avertissement (`Alert` en `role="note"`, relié au bouton par `aria-describedby`) annonce le retrait du catalogue (publié) ou la relance du contrôle sur la nouvelle version (en contrôle, à vérifier) ; le bouton devient « Enregistrer et renvoyer au contrôle » et le succès annonce le retour au contrôle.
- Un résultat de contrôle portant sur une révision antérieure est sans effet (démonstration comprise).

## Articles à vérifier (US-022)

Contrat : `contracts/api/place-de-marche/v1.openapi.yaml`, tag Administration, isolé dans `src/features/admin/api/` (`reviewArticlePayload.ts` valide chaque réponse).

- `/admin` (rôle `admin`) : « Articles à vérifier » (`GET /v1/admin/articles`), contenu de la révision examinée, motif du contrôle traduit et révision.
- « Publier » ou « Refuser » (`POST /v1/admin/articles/{id}/decision`, révision examinée envoyée). Refuser ouvre une boîte de dialogue : motif choisi dans la liste fermée (`src/shared/articleReason.ts`), puis « Confirmer le refus ». L'article quitte la liste, la décision est annoncée en `role="status"` et le focus revient sur le titre de la liste.
- 409 (ou 404) : décision non appliquée, « L'article a changé, rechargez la liste » et bouton « Recharger la liste » ; 403 : message dédié.
- Les motifs de refus de l'admin (`prix_abusif`, `article_interdit`, `vendeur_non_fiable`) sont traduits côté vendeur avec ceux du contrôle (`useControlReasonLabel`).
- Lien « Articles à vérifier » dans l'en-tête pour le seul rôle `admin` ; un autre rôle voit « Accès réservé ».
- Démonstration : trois articles à vérifier, dont deux titres avec balisage (`<script>`, `<b>`) affichés en texte ; la « Pièce de 5 francs » répond 409 à la première décision pour rejouer le conflit.

## Sécurité

- Aucun HTML libre : `react/no-danger`, `no-unsanitized`, `no-eval` en erreur ; un titre contenant du balisage s'affiche comme du texte (US-010, AC4). Le jeu de démonstration en contient deux.
- `localStorage`, `sessionStorage` et `document.cookie` interdits au lint : les jetons restent en mémoire (D06). Seule exception : l'état transitoire de la requête d'autorisation (voir « Connexion »).
- CSP posée par nginx sur toutes les réponses, 404 comprises : `script-src 'self'`, `style-src 'self' 'unsafe-inline'` (Emotion, D04), `connect-src 'self' ${COLLECTOR_IDP_ORIGIN}`, `frame-ancestors 'none'`. Polices auto-hébergées, `assetsInlineLimit: 0`.
- HSTS est posé par l'écouteur de l'ALB, pas ici (D08, ME1).
- Image : `COLLECTOR_IDP_ORIGIN` injecté au démarrage par les modèles nginx. En système de fichiers en lecture seule, monter un `emptyDir` sur `/tmp` et sur `/etc/nginx/conf.d`.

## Connexion (US-008)

Bibliothèque : `oidc-client-ts` (maintenue, PKCE `S256` natif, stockage des utilisateurs injectable, rafraîchissement par jeton sans iframe), sans surcouche React : `src/auth/` expose une `AuthSession` lue par `useSyncExternalStore`.

| Élément                           | Comportement                                                                                                                                                                                                                           |
| --------------------------------- | -------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| Connexion                         | Bouton « Se connecter » de l'en-tête → `auth.` (code d'autorisation + PKCE `S256`, `response_mode=fragment` : le code n'atteint jamais les journaux nginx) → `/rappel` → retour à la page d'origine (chemin de même origine validé)    |
| Jetons                            | `InMemoryWebStorage` uniquement ; `Authorization: Bearer` ajouté par `createHttpClient` via `session.getAccessToken` ; jeton expiré jamais envoyé                                                                                      |
| État de la requête d'autorisation | `state` et vérificateur PKCE dans `sessionStorage` (préfixe `collector.oidc.`), seule exception au lint, limitée à `src/auth/createOidcStateStore.ts` ; aucun jeton, entrée supprimée au retour, entrées périmées purgées au démarrage |
| Renouvellement                    | Silencieux par jeton de rafraîchissement (rotation Keycloak) une minute avant l'expiration ; en cas d'échec, l'état visiteur revient                                                                                                   |
| Rechargement                      | Mémoire vide : visiteur sur les pages publiques ; une page protégée relance la connexion, sans formulaire tant que la session `auth.` vit                                                                                              |
| Déconnexion                       | Révocation des jetons, fin de session Keycloak (`end_session`), retour à `/`                                                                                                                                                           |
| En-tête                           | Nom (`name`, sinon `preferred_username`) et rôle principal (`realm_access.roles` du jeton d'accès : admin, puis vendeur, puis acheteur) ; icônes seules sous `sm`                                                                      |
| Route protégée                    | `RequireAuthentication` (`src/app/routing/`), ex. `/vendeur` avec le rôle `vendeur` ; refus affiché sans appel d'API, le contrôle réel reste côté services (US-009)                                                                    |

Le rôle affiché n'est qu'un confort d'interface : aucune décision d'autorisation n'est prise par le front.

## Internationalisation et accessibilité

- Français par défaut (langue du navigateur sinon), anglais ; changement de langue sans rechargement, `lang` de `<html>` et textes internes MUI suivis.
- Langue conservée pendant la session sans aucun stockage (D06) : elle voyage dans le `state` OIDC de la connexion et est réappliquée au retour sur `/rappel`, succès ou échec ; `ui_locales` est transmis au serveur d'autorisation. Un rechargement complet repart de la langue du navigateur (US-025).
- Motifs du contrôle (`src/shared/controlReason.ts`, liste fermée du contrat) et motifs de refus de l'admin (`src/shared/articleReason.ts`) traduits par `useControlReasonLabel` ; montants par `useMoneyFormatter` (`Intl`, `fr-FR` ou `en-GB`).
- `src/i18n/locales/en.ts` est typé sur `fr.ts` : une clé manquante ou en trop casse la compilation. `npm run i18n:check` (Node 24, sans dépendance) bloque aussi les textes vides et les variables `{{…}}` différentes d'une langue à l'autre.
- Texte en dur bloqué au lint : `i18next/no-literal-string` dans le JSX, `no-restricted-syntax` sur les attributs lus par les aides techniques (`aria-label`, `title`, `alt`, `label`…).
- Lien d'évitement, titre de page et focus sur le `h1` à chaque navigation, annonces dans une zone `role="status"`, cibles tactiles de 2,75 rem, animations coupées sous `prefers-reduced-motion`.
- Focus visible partout, ondulation coupée comprise : anneau de 0,1875 rem sur `:focus-visible` et sur `.Mui-focusVisible` des `ButtonBase`.
- Contrastes : `src/ui/theme/themeContrastPairs.ts` déclare les couples de jetons utilisés (texte 4,5:1, non-texte 3:1) et `contrast.ts` calcule le rapport WCAG ; jsdom ne calculant pas les contrastes, c'est la preuve automatisée hors navigateur. Le gris par défaut des `ToggleButton` (4,4:1) est remplacé par `textSecondary`.
