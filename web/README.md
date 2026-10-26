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
| `catalogue.source`       | `api`, `demo`                                                           | Client HTTP réel ou adaptateur de démonstration chargé à la demande                |
| `catalogue.demoScenario` | `nominal`, `empty`, `error`                                             | États du catalogue rejouables sans back                                            |
| `oidc.authority`         | URL de l'émetteur, ex. `https://auth.<domaine>/realms/collector`        | HTTPS obligatoire (HTTP admis sur `localhost` seulement), sans requête ni fragment |
| `oidc.clientId`          | `collector-web`                                                         | Client public du realm                                                             |
| `oidc.scope`             | portées séparées par une espace, `openid` obligatoire ; défaut `openid` | `profile` requis pour afficher le nom (AC1 de US-008)                              |

Une configuration invalide affiche l'écran d'échec de démarrage. L'origine de `oidc.authority` doit être celle de `COLLECTOR_IDP_ORIGIN` (CSP).

## Contrat supposé du catalogue public

Le contrat OpenAPI n'existe pas encore (US-009, US-013). Hypothèse isolée dans `src/features/catalogue/api/` :

- `GET {marketplaceApiUrl}/v1/articles?page=1&taille=12`, sans authentification ;
- réponse `{ "elements": [{ "id", "titre", "prix": { "montantCentimes", "devise" }, "categorie": { "id", "libelle" } }], "page", "taille", "total" }` ;
- erreurs en `application/problem+json` (RFC 9457) ; le front n'en lit que le statut HTTP (pas de nouvel essai sur 4xx).

La réponse est validée champ par champ avant affichage ; un écart de contrat produit l'état d'erreur. Seul `cataloguePayload.ts` change quand l'OpenAPI sera publiée.

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

- Français par défaut, anglais ; changement de langue sans rechargement, `lang` de `<html>` et textes internes MUI suivis.
- `src/i18n/locales/en.ts` est typé sur `fr.ts` : une clé manquante ou en trop casse la compilation. `i18next/no-literal-string` bloque le texte en dur dans le JSX.
- Lien d'évitement, titre de page et focus sur le `h1` à chaque navigation, annonces dans une zone `role="status"`, cibles tactiles de 2,75 rem, focus visible, animations coupées sous `prefers-reduced-motion`.
