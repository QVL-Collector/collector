# US-008 — Rapport de tests front

| Champ | Valeur |
|---|---|
| US | [US-008 — Se connecter à Collector par le serveur d'autorisation](../../../backlog/US-008-connexion-keycloak.md) |
| Branche | `feat/US-008-connexion-keycloak` |
| Date | 2026-10-26 |
| Fonction | Test |
| Périmètre de ce lot | Front (`web/`) : AC1, AC2, AC4, AC5 (partie client OIDC et stockage), AC10 |
| Hors périmètre de ce lot | AC3 (import du realm, comptes fictifs), AC5 partie serveur (durée de 5 minutes fixée par le realm), AC6 (rotation du jeton de rafraîchissement), AC7 (exposition de Keycloak), AC8 (contrôle du realm en CI), AC9 (CSP et en-têtes nginx) : realm et infrastructure, à prouver avec un Keycloak réel, `curl` et Playwright |
| Outils | Vitest 3.2.4, Testing Library, jsdom 26.1.0, `axe-core` 4.14.0 |
| Verdict | **Tous les AC du périmètre front sont couverts et conformes** (anomalie A1 corrigée et revérifiée) |

## Méthode

Scénarios Given/When/Then dérivés des AC, étiquetés `US-008-ACn`, conçus sans lecture des composants. Docker étant indisponible, Keycloak est simulé **au niveau HTTP**, le plus haut possible :

- l'application démarre par son vrai point d'entrée (`startApplication`), qui lit `/config.json` et crée la vraie session `oidc-client-ts` ;
- `fetch` est intercepté : `/config.json`, document de découverte, point de jeton (code et rafraîchissement), point de révocation, API du catalogue ;
- la navigation de premier niveau vers le serveur d'autorisation est capturée (URL d'autorisation, URL de fin de session) ;
- le retour du serveur d'autorisation est simulé par un rechargement complet sur `/rappel#code=…&state=…` : la mémoire est perdue, `sessionStorage` est conservé, comme dans un navigateur.

Comptes fictifs simulés : vendeur A (`vendeur`), acheteur (`acheteur`), admin (`admin`). Jetons JWT non signés (la vérification de signature relève des services, US-009).

Tests : `web/src/acceptance/US-008/`

| Fichier | Contenu |
|---|---|
| `identityProviderHarness.tsx` | Faux serveur d'autorisation HTTP, capture de navigation, démarrage de l'application, lecture du stockage navigateur |
| `AC1-connexion.test.tsx` | Redirection vers le realm, nom et rôle, retour à la page d'origine, échecs |
| `AC1-espace-vendeur.test.tsx` | Route `/vendeur` protégée : visiteur, vendeur, acheteur |
| `AC1-retour-arriere-et-chemin-de-retour.test.tsx` | Retour arrière depuis Keycloak (`pageshow` persisté, revue M1) ; chemins de retour hors site (revue m1) |
| `AC2-deconnexion.test.tsx` | Révocation, fin de session, retour à `/` en visiteur |
| `AC4-configuration-et-traductions.test.tsx` | `config.json` à l'exécution, configurations OIDC invalides, FR par défaut et EN |
| `AC5-pkce-jetons-en-memoire.test.tsx` | Code + PKCE `S256`, aucun jeton dans le stockage, état PKCE supprimé, jeton expiré jamais envoyé |
| `AC10-accessibilite.test.tsx` | axe-core sur 8 états, clavier, focus, titre, petit écran |

## Résultats d'exécution

| Commande (depuis `web/`) | Résultat |
|---|---|
| `npm test` | 11 fichiers, **97 tests verts** (61 US-008, 36 US-010), 0 échec, après correction de A1 |
| `npm run lint` | 0 erreur, 0 avertissement |
| `npm run typecheck` | 0 erreur |
| `npx prettier --check src/acceptance` | Conforme |
| `npm run build` | Build statique dans `dist/` (`index.html`, `assets/`, `config.json`) |

## Résultats par AC

### AC1 — Connexion, nom et rôle

| Scénario | Attendu (AC) | Réel | Statut |
|---|---|---|---|
| Nominal : clic « Se connecter » | Redirection vers `auth.collector.qvl-project.com`, realm `collector` | Navigation vers `https://auth.collector.qvl-project.com/realms/collector/protocol/openid-connect/auth`, `client_id=collector-web`, `redirect_uri=<origine>/rappel`, portée `openid` | Conforme |
| Nominal : retour vendeur A | Nom et rôle affichés | « Alice Vendeur » et « Vendeur » ; bouton « Se connecter » retiré | Conforme |
| Nominal : acheteur, admin | Nom et rôle affichés | « Acheteur », « Administrateur » | Conforme |
| Limite : connexion depuis `/?page=2` | Revient sur le site | Retour sur `/?page=2`, page 2 affichée | Conforme |
| Limite : échange du code | Un seul échange | Un seul `authorization_code` au point de jeton | Conforme |
| Erreur : `error=access_denied` au retour | Reste visiteur | « La connexion n'a pas abouti », « Se connecter » disponible | Conforme |
| Erreur : `state` forgé | Aucune session | Échec affiché, aucun jeton émis | Conforme |
| Erreur : serveur d'autorisation injoignable | Reste visiteur sur le site | Aucune navigation, reste sur `/` | Conforme |
| Espace vendeur : visiteur sur `/vendeur` | Connexion exigée | Redirection vers le serveur d'autorisation (PKCE `S256`) | Conforme |
| Espace vendeur : retour depuis `/vendeur` | Arrivée dans l'espace vendeur | `/vendeur`, titre « Espace vendeur » | Conforme |
| Espace vendeur : vendeur A connecté | Accès | « Bienvenue, Alice Vendeur… » | Conforme |
| Espace vendeur : acheteur | Refus | Page « Accès réservé », aucun appel d'API hors catalogue | Conforme |
| Limite (M1) : retour arrière depuis Keycloak (`pageshow` persisté) puis nouveau clic « Se connecter » | Nouvelle redirection | Seconde navigation vers `auth.` | Conforme |
| Limite (M1) : retour arrière depuis Keycloak sur `/vendeur` | Nouvelle redirection par la garde | Seconde navigation vers `auth.` (après correction de A1) | Conforme |
| Erreur (m1) : connexion depuis `/a/..//evil.com`, `//evil.com`, `/\evil.com` | Retour sur le site | Retour sur `/`, même origine | Conforme |

### AC2 — Déconnexion

| Scénario | Attendu (AC) | Réel | Statut |
|---|---|---|---|
| Nominal : révocation | Session fermée | Jeton de rafraîchissement envoyé au point de révocation | Conforme |
| Nominal : fin de session | Session fermée | Navigation vers `end_session` avec `id_token_hint` et `post_logout_redirect_uri=<origine>/` | Conforme |
| Nominal : retour sur `/` | État visiteur | « Se connecter » affiché, ni nom ni rôle | Conforme |
| Erreur : révocation indisponible (503) | Session fermée | Fin de session tout de même lancée | Conforme |

### AC4 — Application statique, configuration à l'exécution, traductions

| Scénario | Attendu (AC) | Réel | Statut |
|---|---|---|---|
| Nominal : démarrage | `config.json` lu à l'exécution | `GET /config.json` au démarrage | Conforme |
| Nominal : autre autorité dans `config.json` | Configuration d'exécution appliquée | Redirection vers l'autorité servie, sans reconstruction | Conforme |
| Erreur : 9 configurations OIDC invalides (section absente, HTTP hors localhost, requête, fragment, URL invalide, client absent ou vide, portée sans `openid`, valeur non objet) | Refus | Écran « L'application n'a pas pu démarrer », pas de bouton de connexion | Conforme |
| Erreur : `config.json` introuvable | Refus | Écran d'échec de démarrage | Conforme |
| Nominal : langue par défaut | Français | « Se connecter », `lang="fr"` | Conforme |
| Nominal : textes d'en-tête | Issus des traductions | Rôle et « Se déconnecter » égaux aux valeurs de `fr.ts` | Conforme |
| Nominal : passage en anglais (connecté, visiteur, accès refusé, échec de connexion) | Issus des traductions | Valeurs de `en.ts`, sans rechargement, `lang="en"` | Conforme |
| Construction (relecture de la configuration de build) | React, TypeScript strict, nginx non privilégié | `tsconfig.app.json` : `strict: true` ; `Dockerfile` : `nginxinc/nginx-unprivileged`, `USER 101` ; `npm run build` produit un site statique ; lint `i18next/no-literal-string` vert | Conforme |

L'image Docker n'a pas pu être construite (Docker indisponible) : le service par nginx est vérifié sur le `Dockerfile` seulement.

### AC5 — Code d'autorisation, PKCE S256, jeton en mémoire (partie front)

| Scénario | Attendu (AC) | Réel | Statut |
|---|---|---|---|
| Nominal : requête d'autorisation | Code + PKCE `S256` | `response_type=code`, `code_challenge_method=S256`, défi base64url de 43 caractères, `state` présent, aucun secret client | Conforme |
| Limite : deux connexions | Défi et `state` uniques | Différents à chaque connexion | Conforme |
| Nominal : échange du code | PKCE complet | `code_verifier` de 43 à 128 caractères envoyé | Conforme |
| Nominal : pendant la redirection | Pas de jeton stocké | `localStorage` vide, cookies vides, seul l'état temporaire en `sessionStorage` | Conforme |
| Nominal : connecté | Ni `localStorage` ni `sessionStorage` | Aucun jeton d'accès, de rafraîchissement ni d'identité dans le stockage ni les cookies | Conforme |
| Nominal : après le rappel | État temporaire supprimé | `sessionStorage` vide | Conforme |
| Limite : rechargement | Jeton en mémoire seulement | Retour à l'état visiteur | Conforme |
| Erreur : jeton expiré (+6 min), renouvellement refusé | Jeton expiré jamais envoyé | Avant expiration, `Authorization: Bearer` présent ; après, aucune requête ne le porte | Conforme |
| Erreur : expiration, renouvellement refusé | Fin de session côté front | Retour à l'état visiteur | Conforme |

La durée de 5 minutes du jeton relève du realm (hors lot) ; le front a été exercé avec `expires_in=300`.

### AC10 — Accessibilité

| Scénario | Attendu (AC) | Réel | Statut |
|---|---|---|---|
| axe-core : accueil visiteur avec « Se connecter » | 0 serious/critical | 0 | Conforme |
| axe-core : accueil connecté, anglais, petit écran (visiteur et connecté), accès refusé, échec de connexion, échec de démarrage | 0 serious/critical | 0 | Conforme |
| Petit écran | Bouton nommé | « Se connecter » et « Se déconnecter » gardent leur nom accessible | Conforme |
| Clavier : Tab puis Entrée sur « Se connecter » | Utilisable au clavier | Redirection vers le serveur d'autorisation | Conforme |
| Clavier : Tab puis Espace sur « Se déconnecter » | Utilisable au clavier | Fin de session lancée | Conforme |
| Focus et titre après retour sur `/vendeur` | — | Focus sur le `h1`, titre « Espace vendeur · Collector » | Conforme |

La règle `color-contrast` est désactivée sous jsdom (pas de rendu) ; le contraste reste à vérifier par `@axe-core/playwright` en navigateur.

## Anomalies

| Id | AC | Sévérité | Fichiers en cause | Statut |
|---|---|---|---|---|
| A1 | AC1 | Majeure | `web/src/app/routing/useAuthenticationGate.ts`, `web/src/auth/createOidcSession.ts` | Corrigée, revérifiée le 2026-10-26 (5/5 verts dans le fichier, 97/97 au total) |

**A1 — La garde `/vendeur` ne relançait pas la connexion après un retour arrière depuis Keycloak (corrigée).**
- Attendu (AC1) : « Given un visiteur […], When il clique « Se connecter », Then il est redirigé vers `auth.collector.qvl-project.com` » ; l'espace vendeur doit rester atteignable par la connexion (revue M1).
- Étapes : ouvrir `/vendeur` en visiteur → redirection vers `auth.` ; émettre `pageshow` avec `persisted: true` (restauration depuis le cache aller-retour).
- Réel avant correction : aucune seconde navigation vers `auth.` (1 au lieu de 2), sur deux exécutions. Test : `AC1-retour-arriere-et-chemin-de-retour.test.tsx`, scénario « redirigé depuis /vendeur puis revenu par « Précédent » ».
- Le même scénario depuis le bouton « Se connecter » est conforme : la remise à zéro de la redirection en cours fonctionne pour un clic ultérieur, mais pas pour la relance faite par la garde au même événement `pageshow`.

## Limites et suites

- Keycloak simulé : JWT non signés, pas de vraie page de connexion. Le parcours AC1/AC2 réel reste à jouer en Playwright sur compose.
- AC3, AC6, AC7, AC8, AC9 et la durée de 5 minutes d'AC5 : à prouver par DevOps et Test sur le realm et l'infrastructure.
- Le harnais lit `localStorage`, `sessionStorage` et `document.cookie` par des accesseurs dédiés pour vérifier l'absence de jetons ; la règle de lint qui les interdit reste active ailleurs.
- Correction : `createOidcSession.ts` remet à zéro la redirection en cours sur `pageshow` persisté, avant la relance de la garde. Revérification : seconde navigation vers `auth.` observée.
