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

| Clé                      | Valeurs                                 | Rôle                                                                |
| ------------------------ | --------------------------------------- | ------------------------------------------------------------------- |
| `marketplaceApiUrl`      | chemin de même origine, ex. `/marche`   | Préfixe de la Place de marché ; une URL absolue est refusée         |
| `catalogue.source`       | `api`, `demo`                           | Client HTTP réel ou adaptateur de démonstration chargé à la demande |
| `catalogue.demoScenario` | `nominal`, `empty`, `error`             | États du catalogue rejouables sans back                             |

La configuration OIDC (émetteur, client) sera ajoutée avec la connexion (US-008).

## Contrat supposé du catalogue public

Le contrat OpenAPI n'existe pas encore (US-009, US-013). Hypothèse isolée dans `src/features/catalogue/api/` :

- `GET {marketplaceApiUrl}/v1/articles?page=1&taille=12`, sans authentification ;
- réponse `{ "elements": [{ "id", "titre", "prix": { "montantCentimes", "devise" }, "categorie": { "id", "libelle" } }], "page", "taille", "total" }` ;
- erreurs en `application/problem+json` (RFC 9457) ; le front n'en lit que le statut HTTP (pas de nouvel essai sur 4xx).

La réponse est validée champ par champ avant affichage ; un écart de contrat produit l'état d'erreur. Seul `cataloguePayload.ts` change quand l'OpenAPI sera publiée.

## Sécurité

- Aucun HTML libre : `react/no-danger`, `no-unsanitized`, `no-eval` en erreur ; un titre contenant du balisage s'affiche comme du texte (US-010, AC4). Le jeu de démonstration en contient deux.
- `localStorage`, `sessionStorage` et `document.cookie` interdits au lint : les jetons resteront en mémoire (D06).
- CSP posée par nginx sur toutes les réponses, 404 comprises : `script-src 'self'`, `style-src 'self' 'unsafe-inline'` (Emotion, D04), `connect-src 'self' ${COLLECTOR_AUTH_ORIGIN}`, `frame-ancestors 'none'`. Polices auto-hébergées, `assetsInlineLimit: 0`.
- HSTS est posé par l'écouteur de l'ALB, pas ici (D08, ME1).
- Image : `COLLECTOR_AUTH_ORIGIN` injecté au démarrage par les modèles nginx. En système de fichiers en lecture seule, monter un `emptyDir` sur `/tmp` et sur `/etc/nginx/conf.d`.

## Point d'extension de la connexion (US-008)

`src/auth/AuthSession.ts` définit la session ; `anonymousSession` est passée par défaut à `createAppServices`. US-008 fournira une session OIDC (PKCE `S256`, `InMemoryWebStorage`) : le client HTTP ajoute alors `Authorization: Bearer` sans autre changement.

## Internationalisation et accessibilité

- Français par défaut, anglais ; changement de langue sans rechargement, `lang` de `<html>` et textes internes MUI suivis.
- `src/i18n/locales/en.ts` est typé sur `fr.ts` : une clé manquante ou en trop casse la compilation. `i18next/no-literal-string` bloque le texte en dur dans le JSX.
- Lien d'évitement, titre de page et focus sur le `h1` à chaque navigation, annonces dans une zone `role="status"`, cibles tactiles de 2,75 rem, focus visible, animations coupées sous `prefers-reduced-motion`.
