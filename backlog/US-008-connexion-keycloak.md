# US-008 — Se connecter à Collector par le serveur d'autorisation

| Champ | Valeur |
|---|---|
| Epic | E3 Mise en vente contrôlée |
| Statut | Livrée côté front ; AC3, AC6 à AC9 (realm, infra) à prouver |
| Sprint | 1 (J4) |
| Engagement | Ferme |
| Estimation | 5 points |
| Dépend de | US-002, US-004 ; verdict de XP03 |
| Fonction(s) | DevOps (Keycloak, realm), Front (application, connexion) |
| Service | Front, serveur d'autorisation |
| Critères grille | C8, C2 |
| Exigences | EF-06, ENF-03, ENF-26, ENF-30 |
| Branche | `feat/US-008-connexion-keycloak` |

## Récit

En tant que vendeur, je veux me connecter à Collector depuis le site, afin d'accéder à mon espace vendeur sans que mon mot de passe ne transite par les services de Collector.

## Critères d'acceptation

### Fonctionnels

- **AC1** — Given un visiteur sur `collector.qvl-project.com`, When il clique « Se connecter », Then il est redirigé vers `auth.collector.qvl-project.com` (realm `collector`), s'authentifie avec un compte fictif, revient sur le site et voit son nom et son rôle.
- **AC2** — Given un utilisateur connecté, When il clique « Se déconnecter », Then sa session est fermée et le site revient à l'état visiteur.
- **AC3** — Given chaque `up`, When le realm est importé, Then les comptes fictifs vendeur A, vendeur B, acheteur et admin existent avec des mots de passe générés pour la session, jamais versionnés.
- **AC4** — Given l'application front, When elle est construite, Then c'est une application statique React, TypeScript strict et MUI servie par nginx non privilégié, qui lit sa configuration (`config.json`) à l'exécution, et tous ses textes viennent des fichiers de traduction (français par défaut).

### Non fonctionnels

- **AC5 (Sécurité)** — Given le client OIDC du front, When la connexion a lieu, Then le flux est « code d'autorisation » avec PKCE `S256` ; le jeton d'accès dure 5 minutes et reste en mémoire (ni `localStorage`, ni `sessionStorage`).
- **AC6 (Sécurité)** — Given un jeton de rafraîchissement déjà utilisé, When il est rejoué, Then il est refusé (rotation, MT2).
- **AC7 (Sécurité, X17 étendu)** — Given l'environnement allumé, When `/admin/`, `/realms/master/` et `/realms/collector/account` sont demandés depuis Internet, Then ils répondent 404 et le port de gestion 9000 est injoignable ; le document de découverte et la page de connexion répondent 200.
- **AC8 (Sécurité)** — Given le JSON du realm versionné, When la CI s'exécute, Then le contrôle du realm bloque toute URI de redirection ou origine non exacte, flux implicite ou « mot de passe » actif, `Full scope allowed` actif ou secret en clair.
- **AC9 (Sécurité)** — Given la page servie par nginx, When ses en-têtes sont relus, Then une politique de sécurité du contenu stricte n'autorise que les scripts de la plateforme, et les en-têtes de sécurité du front sont présents.
- **AC10 (Accessibilité)** — Given la page d'accueil et le bouton de connexion, When axe-core les analyse, Then aucune violation « serious » ou « critical » n'est relevée.

## Sécurité

| Exigence | Comment l'US la sert | AC |
|---|---|---|
| ES-02 | Authentification déléguée, OIDC + PKCE, rotation du rafraîchissement | AC1, AC5, AC6 |
| ES-07 | Console Keycloak non exposée | AC7 |
| ES-08 | CSP stricte du front | AC9 |
| ES-06 | Mots de passe des comptes de test générés à chaque session | AC3 |

Mesures : MA1, MA2, MA7, MA10, MA11, MA14, ME6. Frontières : navigateur, serveur d'autorisation.

## Tests attendus

| Type | Outil | Ce qui est prouvé | Fonction |
|---|---|---|---|
| Unitaire front | Vitest, Testing Library | AC2, AC4 (stockage du jeton) | Front |
| Acceptation navigateur | Playwright | AC1, AC2 | Test |
| Sécurité négative | Playwright API, script curl après déploiement | AC6, AC7 | Test, DevOps |
| Contrôle propre en CI | Script `jq` sur le realm | AC8 | DevOps |
| Accessibilité | `@axe-core/playwright` | AC10 | Test |

## Hors périmètre

- Second facteur de l'admin : US-022.
- Vérification des jetons par les services : US-009.
- Traduction anglaise : US-025.

## Décisions liées

- [D06 — Authentification et autorisation](../docs/decisions/D06-authentification-autorisation.md)
- [D04 — Stack](../docs/decisions/D04-stack.md)
- [D08 — Exposition, TLS, domaine](../docs/decisions/D08-exposition-tls-domaine.md)

## Definition of Done

Reprise de [PILOTAGE.md](../docs/PILOTAGE.md) § 5, cochée au fil du circuit.

- [ ] Tous les AC sont couverts par des tests verts en CI (`ci-ok`).
- [ ] Revue de code faite, retours traités.
- [ ] Martin a testé et prononcé « je valide la livraison de la feature ».
- [ ] Documentation touchée à jour.
- [ ] Statut, date de validation et liens des rapports renseignés ci-dessous.
- [ ] Aucune donnée sensible ni credential ajouté au dépôt.

## Suivi

| Étape | Lien / date |
|---|---|
| Rapport de test | |
| Rapport de revue de code | |
| Validation Martin | 2026-10-26 |
| Livraison (PR / merge) | https://github.com/QVL-Collector/collector/pull/15 |
