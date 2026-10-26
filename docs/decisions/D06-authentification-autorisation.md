# D06 — Authentification et autorisation

| Champ | Valeur |
|---|---|
| Statut | Candidate (XP en D14) |
| Date | 2026-10-26 |
| Décideur | Martin |
| Dépend de | [D03](D03-hebergement-orchestration.md), [D04](D04-stack.md) |
| Impacte | D05 (authentification du canal), D07 (espace du serveur d'autorisation dans RDS), D08 (domaine, routes de l'ALB), D10 (secrets des clients), D11 (allumage), D13 (tests négatifs), D14 (expérimentation), `libs/platform/`, [exigences-securite.md](../securite/exigences-securite.md) |
| Critères grille | C8, C2, C9 |

## Contexte et besoin

D06 choisit le serveur d'autorisation, le mode de conservation des jetons dans le front et les règles d'autorisation appliquées par les services. Elle ne choisit ni le domaine (D08) ni le canal (D05).

### Exigences

Source : [exigences-securite.md](../securite/exigences-securite.md) § 5.5 (entrée « D06 ») et § 6.

| Exigence | Ce qu'elle impose à D06 | Dans le POC |
|---|---|---|
| ES-02 | Serveur d'autorisation dédié, protocole standard (OpenID Connect) ; aucun service ne stocke de mot de passe ; jetons signés, courts, vérifiés par chaque service (signature, expiration, émetteur, audience) ; session close à la déconnexion ; jamais de jeton dans une URL | Démontrée (BP2) |
| ES-03 | Rôles visiteur, acheteur, vendeur, admin ; refus par défaut ; le rôle vient du seul serveur d'autorisation | Démontrée (BP2, BP5) |
| ES-04 | Contrôle par propriété côté serveur, y compris derrière un appel entre services | Démontrée (BP5) |
| ES-38 | Jeton de service pour « réserver » : émis par le serveur d'autorisation, court, audience « Place de marché », droit limité à « réserver » ; jeton de l'acheteur revérifié par la Place de marché | Intégrée (§ 6.2) |
| ES-09 | Limitation des tentatives sans verrouillage définitif, politique de mot de passe, message identique que le compte existe ou non | Partielle (paramétrage) |
| ES-10 | Second facteur obligatoire pour l'admin | Recommandée « si simple paramétrage » (§ 6.3) |
| ES-20 | Permissions de modération et d'administration distinctes au sein du rôle admin | Documentée |

Rôles : [exigences.md](../00-cadrage/exigences.md) § Rôles. Un compte unique cumule acheteur et vendeur (EF-07) ; l'admin assure aussi la modération.

### Conditions déjà posées par D03 et D04

- **D03, conséquences** : serveur d'autorisation en conteneur dans le cluster, son espace dans RDS ; ALB pour ses seules pages de connexion, console d'administration non routée (MS2, X17, X19) ; URL d'émetteur stable malgré la recréation de l'ALB à chaque session (D08).
- **D04, avis sécurité** :
  - vérification par `jwx` : algorithmes asymétriques imposés, jeu de clés en cache, émetteur, audience, expiration, type de jeton et portée « réserver » contrôlés ; aucune option de contournement (`Skip*`) ;
  - front statique React : client public, Authorization Code avec PKCE `S256`, jetons en mémoire (`InMemoryWebStorage`), jamais en `localStorage` ;
  - CSP stricte sur les scripts (`connect-src` limité aux API et à l'émetteur).
- **D04, faisabilité back** : `x/oauth2` réécrit l'en-tête `Authorization` ; le jeton de l'acheteur passe dans un second en-tête, convention à fixer ici.

### Contraintes

- **Learner Lab** ([environnement-learner-lab.md](../00-cadrage/environnement-learner-lab.md) § 3) :
  - Cognito est absent du Readme 2025, mais User Pool, client et domaine ont été créés dans plusieurs dépôts (tfstate 2026-04) : disponibilité **[?]** jusqu'à la sonde du compte ;
  - Identity Pools inutilisables faute de rôles (non requis ici) ;
  - IAM figé : aucun rôle à créer, ce qui pèse sur les déclencheurs Lambda de Cognito [?].
- **Nœuds** : deux t3.medium de 4 Gio au repos, densité déjà tendue (D03, faisabilité : environ 25 pods pour 34 places).
- **Allumage** : 35 à 45 min par session (D03) ; base recréée à chaque session (D03, avis sécurité).
- **Local d'abord** : le même chart de minikube à EKS (D03, CR6 ; ENF-29).

## Options

### Option A — Keycloak en conteneur dans EKS

- **Description** :
  - Keycloak (image officielle, construite en mode optimisé), 1 pod, base dans RDS PostgreSQL (schéma et compte dédiés, D07).
  - Un realm `collector` décrit en fichier JSON versionné, importé au démarrage : clients, rôles, portées, politiques. Aucun réglage à la main.
  - **Front** : client public `collector-web`, Authorization Code + PKCE `S256`, URI de redirection exactes.
  - **Utilisateurs** : rôles de realm `acheteur` (par défaut à l'inscription), `vendeur`, `admin` ; `admin` est un rôle composite de `moderation` et `administration` (ES-20).
  - **Audience** : un « audience mapper » ajoute `place-de-marche`, `transactions`, `interactions` au jeton de l'utilisateur ; chaque service vérifie la sienne.
  - **Service à service** : client confidentiel `transactions` en « client credentials », seul titulaire de la portée `reserver`, qui ajoute l'audience `place-de-marche` au jeton de service.
  - **MFA admin** : sous-flux conditionnel « Condition - user role » + OTP, imposé au seul rôle `admin` (ES-10).
  - **ES-09** : détection de force brute (attente croissante, verrouillage permanent désactivé), politique de mot de passe, liste de mots de passe interdits.
- **Avantages** :
  - Couvre ES-38 à la lettre : `aud` réel dans le jeton d'accès, sans contrôle de remplacement.
  - ES-10 par paramétrage : passe de « Recommandée » à « Démontrée ».
  - Identique en local et sur EKS (même image, même realm importé) : tient CR6 et ENF-29.
  - Indépendant du Learner Lab : aucun service AWS de plus à sonder.
  - Standard du marché, transférable (C9) ; schéma de soutenance lisible (C2).
- **Inconvénients** :
  - Composant à état de plus à exploiter : mises à jour de sécurité, 1 instance (point unique de défaillance assumé par D03, échec fermé).
  - **Mémoire** : environ 1 à 1,25 Gio (JVM) [?], soit le quart d'un t3.medium ; aggrave la densité de D03.
  - **Démarrage** : de l'ordre de 20 à 40 s en mode optimisé, plus la création du schéma et l'import du realm sur une base neuve, environ 1 à 2 min [?]. Hors du chemin critique : le cluster domine l'allumage.
  - Clés de signature régénérées à chaque session (base recréée) : sans effet, les services lisent le jeu de clés publié.
  - Console d'administration à cloisonner (voir « Points transverses »).
- **Coût AWS et effort** :
  - Aucun coût AWS direct ; place sur les nœuds (voir ci-dessus), espace RDS négligeable.
  - Effort : 2,5 à 3,5 j (chart, realm en code, règles de l'ALB, MFA conditionnel, intégration front, tests négatifs).
- **Complexité ajoutée** : un conteneur et un fichier de realm. Elle est attendue : D02 et D03 prévoient déjà un serveur d'autorisation en conteneur.
- **Couverture grille** : C8 (BP2, BP5, ES-38, ES-10), C2 (frontière d'authentification visible), C9 (compétence IAM standard).

### Option B — Amazon Cognito (si la sonde le confirme)

- **Description** :
  - Un User Pool dans la couche persistante de l'IaC (D11), groupes `acheteur`, `vendeur`, `admin` (revendication `cognito:groups`).
  - Client public avec PKCE pour le front ; « resource server » `place-de-marche` et portée personnalisée `place-de-marche/reserver` ; client confidentiel `transactions` en « client credentials ».
- **Avantages** :
  - Aucun pod : libère environ 1 Gio par nœud ; rien à mettre à jour.
  - **Émetteur stable par construction** (`https://cognito-idp.<région>.amazonaws.com/<pool>`), indépendant de l'ALB et de D08.
  - Hors de l'allumage s'il est dans la couche persistante.
- **Inconvénients** :
  - **ES-38** : le jeton d'accès ne porte pas `aud`, mais `client_id` et `scope`. L'audience se vérifie par la portée (`place-de-marche/reserver`) et `client_id` = `transactions` : contrôle équivalent admis par D04, à condition de l'écrire explicitement. Pour le jeton de l'utilisateur, aucune audience par service : tout jeton du pool vaut pour les trois API [?].
  - **ES-10** : MFA réglé par pool et par utilisateur, pas par groupe ; l'imposer au seul admin demande un réglage par utilisateur ou un déclencheur Lambda [?].
  - **Local** : pas d'émulateur officiel ; il faudrait un autre serveur en local, ce qui casse la parité de CR6 et ENF-29.
  - **Learner Lab** : absent du Readme 2025 [?] ; perte du pool à toute réinitialisation du compte.
  - « Client credentials » facturé par jeton et par client depuis 2024 [?] : faible, mais nouvel élément au garde-fou coût.
- **Coût AWS et effort** :
  - Utilisateurs actifs dans la gratuité ; jetons de service quelques centimes sur le projet [?].
  - Effort : 1,5 à 2 j sur AWS, plus 1 à 1,5 j pour un substitut local et l'écart de comportement entre les deux.
- **Complexité ajoutée** : faible côté exploitation, mais deux serveurs d'autorisation différents (local et cloud).
- **Couverture grille** : C8 (BP2, BP5 ; ES-38 par contrôle équivalent ; ES-10 incertaine), C2.

### Option C — Authentification maison dans un service Go

- **Description** : un service Go qui stocke les mots de passe, émet et signe les jetons, publie son jeu de clés.
- **Avantages** : léger (quelques dizaines de Mo), aucune dépendance.
- **Inconvénients** :
  - Contraire à l'esprit d'ES-02 : le serveur « dédié, standard » serait une implémentation maison d'un protocole de sécurité.
  - Tout est à écrire et à prouver : hachage des mots de passe, rotation des clés, PKCE, révocation, force brute (ES-09), MFA (ES-10), réinitialisation (M-ACC-04). Chaque oubli est une faille sur l'actif le plus critique (A04, M-ACC-01, M-BO-01).
  - Aucun gain sur un besoin présent ; à défendre devant le jury comme une prise de risque.
- **Coût AWS et effort** : 8 à 15 j [estimation], sans la preuve de conformité.
- **Complexité ajoutée** : la plus forte, en code de sécurité.
- **Couverture grille** : C8 dégradé ; contre-exemple de bonne pratique.

### Options écartées d'emblée

| Option | Motif |
|---|---|
| Dex | Fédération d'identités ; pas d'inscription ni de gestion des comptes (EF-06) |
| Ory Kratos + Hydra | Deux composants à assembler pour ce que Keycloak fait seul |
| Zitadel, Authentik | Crédibles et plus légers en mémoire [?], mais moins documentés pour l'audience par client et le MFA par rôle ; à reconsidérer seulement si la mémoire de Keycloak bloque (voir « Points soumis aux avis ») |

### Sous-question — Jetons dans le navigateur ou intermédiaire serveur (BFF)

| Critère | V1 — SPA publique, jetons en mémoire | V2 — Intermédiaire serveur (BFF) |
|---|---|---|
| Principe | `oidc-client-ts` avec `InMemoryWebStorage`, jeton dans l'en-tête `Authorization` | Composant Go qui détient les jetons ; le navigateur n'a qu'un cookie `HttpOnly`, `Secure`, `SameSite=Strict` |
| Vol du jeton par un script injecté | Possible pendant la session | Impossible ; l'usage pendant la session reste possible |
| CSP | `style-src 'unsafe-inline'` (D04) | Nonce par réponse possible, `'unsafe-inline'` retiré |
| Requête intersite forgée | Sans objet (en-tête) | À traiter (`SameSite`, jeton anti-CSRF) |
| Composants | Aucun de plus ; front statique de D04 | Un service de plus, sessions à stocker, routage de toutes les API par lui |
| Effort | Inclus dans A | + 2 à 3 j |

Recommandation : **V1 dans le POC**. Le risque résiduel (vol du jeton par injection) est borné par la CSP stricte sur les scripts, la neutralisation des sorties (ES-07, ES-08) et des jetons d'accès de 5 min. Le BFF est documenté comme cible V1 de production, avec sa justification.

### Points transverses (valables pour A)

- **Émetteur stable** : l'émetteur est `https://auth.<domaine>/realms/collector`, fixé par l'option `hostname` de Keycloak, jamais par le nom DNS de l'ALB. À chaque `up`, l'IaC repointe l'enregistrement `auth.<domaine>` sur le nouvel ALB (zone dans la couche persistante, D08). Si D08 ne dispose pas d'un domaine stable (repli de type sslip.io), l'émetteur change à chaque session : il est alors injecté au déploiement dans le realm, le `config.json` du front et la configuration des services. Fonctionnel, mais fragile ; D08 doit le trancher.
- **Sous-domaine plutôt que chemin** : `auth.<domaine>` isole les cookies de session de Keycloak de l'origine du front ; `connect-src` ajoute cette origine (D04).
- **Console non exposée** :
  - l'ALB ne route que `/realms/collector/` et `/resources/` ; `/admin/` et `/realms/master/` répondent 404 (MS2, X17) ;
  - port de gestion (santé, métriques) non routé ;
  - administration par `kubectl port-forward` depuis le poste, politique réseau qui ne laisse joindre Keycloak que par l'ALB et les services (X19) ;
  - compte d'administration initial créé depuis Secrets Manager (MS6).
- **Allumage** : les services ne dépendent pas de Keycloak pour démarrer. `jwx` charge le jeu de clés à la première requête et réessaie ; les sondes de disponibilité des services n'appellent pas Keycloak, sinon une indisponibilité se propagerait.
- **Autorisation fine, en deux niveaux** :
  1. **Rôle**, lu dans le jeton (`realm_access.roles`), vérifié par chaque fonction, refus par défaut (ES-03).
  2. **Propriété**, vérifiée par le service dans ses propres données : `sub` du jeton comparé au propriétaire de l'article, de la notification, de la réservation (ES-04). Jamais d'identifiant d'utilisateur lu dans le corps ou un en-tête libre.
  Ces vérifications vivent dans `libs/platform/` (D04) : un vérificateur de jeton, un contrôle de rôle ; la propriété reste dans chaque service.
- **Appel « réserver »** :
  - `Authorization: Bearer <jeton de service>` : audience `place-de-marche`, portée `reserver`, `azp` = `transactions` ;
  - `X-Collector-User-Token: <jeton de l'acheteur>` : revérifié complètement par la Place de marché, qui en tire l'acheteur (`sub`).
  - L'échange de jetons (RFC 8693) ferait plus propre mais ajoute un flux à configurer et à tester : écarté pour le POC.
- **Durées** : jeton d'accès 5 min ; jeton de rafraîchissement à rotation, usage unique ; session inactive close à 30 min ; jeton de service 5 min, mis en cache par `x/oauth2/clientcredentials`.
- **Rôle vendeur en V1** : dans le POC, les comptes de test sont créés avec leurs rôles par l'import du realm. En V1, devenir vendeur exige d'attribuer le rôle : soit par une action dans Keycloak (parcours dédié), soit par un compte de service de la Place de marché limité à cette attribution. Point soumis aux avis.
- **Authentification du canal (lien D05, non tranché)** : si le broker retenu en D05 accepte les jetons OAuth 2.0, Keycloak peut émettre un jeton de service par service, audience « canal » ; sinon le broker garde ses propres identités par service, et D06 n'intervient pas. D05 tranche.

### Synthèse

| Critère | A — Keycloak | B — Cognito | C — Maison |
|---|---|---|---|
| ES-02, ES-03 | Oui | Oui | À écrire et prouver |
| ES-38 (audience) | `aud` natif | Contrôle équivalent par `scope` et `client_id` ; pas d'audience par service pour l'utilisateur [?] | À écrire |
| ES-10 (MFA admin) | Paramétrage par rôle | Par utilisateur ou Lambda [?] | À écrire |
| ES-09 | Paramétrage | Paramétrage partiel [?] | À écrire |
| Local identique au cloud (CR6, ENF-29) | Oui | Non | Oui |
| Disponibilité Learner Lab | Sans dépendance | [?], sonde requise | Sans dépendance |
| Émetteur stable | Dépend du domaine (D08) | Natif | Dépend du domaine (D08) |
| Mémoire sur les nœuds | ≈ 1 à 1,25 Gio [?] | 0 | ≈ 50 Mo |
| Allumage | + 1 à 2 min, en parallèle | 0 (couche persistante) | Négligeable |
| Effort | 2,5 à 3,5 j | 2,5 à 3,5 j (avec le substitut local) | 8 à 15 j |
| Risque sécurité | Faible ; console à cloisonner | Faible | Élevé |

## Recommandation

**Option A — Keycloak en conteneur dans EKS, base dans RDS, realm décrit en code ; SPA publique avec jetons en mémoire (V1 de la sous-question).**

1. **Elle couvre ES-38 sans contrôle de remplacement** : audience réelle dans le jeton, portée `reserver` réservée au client `transactions`. C'est l'exigence la plus pointue de D06 et un test négatif démontré.
2. **ES-10 passe en « Démontrée » par paramétrage** : MFA imposé au seul admin, réponse directe à M-BO-01, risque critique.
3. **Elle tient le « local d'abord »** retenu par D03 : même image, même realm de minikube à EKS. Cognito obligerait à deux serveurs différents.
4. **Elle ne dépend d'aucun service AWS incertain** dans le Learner Lab.
5. **Son prix est connu** : environ 1 Gio de mémoire et 1 à 2 min de démarrage, hors du chemin critique de l'allumage.

**Pourquoi pas B, plus léger.** B retire un pod et rend l'émetteur stable sans D08. C'est le repli si la mémoire de Keycloak force un troisième nœud au repos et que la sonde confirme Cognito. Il perd sur trois besoins présents : audience par service, MFA par rôle, parité locale.

**Pourquoi pas C.** Écrire soi-même le composant le plus sensible, sans gain sur aucun besoin : écartée.

**Volontairement exclu**, au nom de la simplicité : BFF (documenté pour la V1), échange de jetons RFC 8693, authentification des clients par clé privée (`private_key_jwt`), fédération avec des fournisseurs externes, haute disponibilité de Keycloak, thème de connexion personnalisé au-delà du logo et des deux langues.

### Points soumis aux avis

- **Sécurité** :
  - SPA avec jetons en mémoire plutôt que BFF : risque résiduel accepté pour le POC ?
  - Convention du second en-tête pour le jeton de l'acheteur, et contrôles de la Place de marché sur « réserver » (`aud`, `scope`, `azp`, puis jeton de l'acheteur) ;
  - cloisonnement de la console : routes de l'ALB en liste blanche, `port-forward`, compte initial ;
  - secret du client `transactions` (Secrets Manager puis Secret Kubernetes, MS6) plutôt que `private_key_jwt` ;
  - durées des jetons et de la session ; révocation à la déconnexion ;
  - attribution du rôle vendeur en V1 ;
  - sous-domaine `auth.` plutôt qu'un chemin sur l'origine du front.
- **Faisabilité** :
  - mémoire réelle de Keycloak sur t3.medium et effet sur la densité (troisième nœud au repos ?) ;
  - démarrage sur base neuve avec import du realm, à chaque session ;
  - repli si D08 n'a pas de domaine stable ;
  - tests Playwright avec jetons en mémoire (connexion rejouée par test, D04) ;
  - effort de 2,5 à 3,5 j et place dans le planning (avant le sprint 3, D04).

## Avis sécurité

Fonction : Sécurité — 2026-10-26. Périmètre : options A, B, C, sous-question des jetons, points soumis aux avis. Référence : [exigences-securite.md](../securite/exigences-securite.md), avis sécurité de D03 et D04.

**Synthèse.** Avis favorable à l'option A (Keycloak) avec V1 (SPA, jetons en mémoire), sous les mesures MA1 à MA14 ci-dessous. Le risque principal ne vient pas du protocole mais du **paramétrage** : un realm mal réglé ou une console joignable donne la main sur toutes les identités (A04, M-BO-01). Le realm en code est donc à traiter comme du code de sécurité, relu et contrôlé en CI.

**1. Options**
- **A, favorable** : seule option qui tient ES-38 (audience réelle), ES-10 (MFA par rôle) et ES-09 par paramétrage, avec un comportement identique en local et sur EKS. Les tests négatifs de D13 portent ainsi sur le serveur réellement déployé.
- **B, repli acceptable, sous conditions** : contrôle de l'audience par `client_id` et `scope` écrit dans `libs/platform/` et testé (D04) ; MFA forcé par utilisateur sur chaque compte admin, vérifié par un test ; substitut local documenté comme écart. Sans ces trois conditions, B dégrade ES-38 et ES-10.
- **C, défavorable** : réécrire hachage, rotation des clés, PKCE, force brute et MFA sur l'actif le plus critique (A04), sans aucun gain. Contraire à ES-02.

**2. Jetons en mémoire (V1) ou BFF (V2)**
- **V1 accepté pour le POC.** Le BFF retire le vol du jeton, pas son usage : un script injecté agit au nom de la victime dans les deux cas. L'écart réel est la fenêtre d'exploitation hors session, bornée par un jeton d'accès de 5 min et un jeton de rafraîchissement à rotation.
- Conditions : CSP stricte sur les scripts (D04) ; aucun jeton dans une URL, un journal ou une trace ; `Referrer-Policy: no-referrer` ; flux implicite et « mot de passe » désactivés sur `collector-web`.
- Rafraîchissement : **rotation avec détection de réutilisation** (`Revoke Refresh Token` actif, réutilisation 0). Un rechargement de page relance une redirection silencieuse vers `auth.` : jamais de repli vers `sessionStorage` pour « garder la session ».
- Risque résiduel à inscrire au registre ; BFF documenté comme cible de production, comme proposé.

**3. « Réserver » : convention des deux en-têtes**
- Convention acceptée ; RFC 8693 n'apporte pas de gain décisif au POC. Ordre de contrôle dans la Place de marché, chaque échec renvoyant 401 ou 403 sans détail :
  1. `Authorization` : signature, `iss` exact, `exp`, `aud` contient `place-de-marche`, `scope` contient `reserver`, `azp` = `transactions`, `typ` = `Bearer`.
  2. `X-Collector-User-Token` : vérification complète et indépendante (signature, `iss`, `exp`, `aud`, `typ`), `azp` = `collector-web`, rôle `acheteur`. L'acheteur est le `sub` de ce jeton ; tout identifiant d'acheteur présent dans le corps est ignoré ou refusé (M-MOD-02).
  3. Les deux en-têtes sont obligatoires ; un jeton d'utilisateur dans `Authorization` est refusé (pas de `reserver`), un jeton de service dans le second en-tête aussi (`azp`).
- **Piège Keycloak** : le compte de service d'un client reçoit les rôles par défaut du realm. Si `acheteur` y figure, le jeton de `transactions` porte `acheteur` et passe les routes d'utilisateur. Imposer `azp` = `collector-web` sur toutes les routes d'utilisateur et retirer tout rôle de realm au compte de service.
- `Full scope allowed` désactivé sur les deux clients : chaque jeton ne porte que les rôles et portées attendus.
- Le second en-tête est masqué dans les journaux et les traces ; la route « réserver » reste absente de l'ALB (ES-38, M-SYN-01).
- Limite assumée : le jeton d'utilisateur porte trois audiences ; un service compromis peut le rejouer vers les deux autres pendant 5 min (menace M-IDP-03).

**4. Cloisonnement de la console**
- Liste blanche de l'ALB acceptée : uniquement `/realms/collector/protocol/openid-connect/`, `/realms/collector/login-actions/`, `/resources/` et le document de découverte, plutôt que tout `/realms/collector/`. La console de compte (`/realms/collector/account`) n'est routée que si un besoin est écrit ; le profil utilisateur déclaratif ne rend modifiable aucun attribut servant à une décision d'autorisation (M-BO-02).
- `/admin/`, `/realms/master/` et le port de gestion 9000 : non routés, test de non-joignabilité depuis Internet à chaque déploiement (X17).
- La politique réseau (X19) n'a d'effet que si l'application des politiques réseau est activée sur le CNI d'EKS [?] : à prouver par un test de refus, sinon elle est décorative.
- Compte d'administration initial : créé depuis Secrets Manager, temporaire, remplacé à la première session par un compte nominatif avec OTP, puis supprimé. Le rôle applicatif `admin` du realm `collector` n'a aucun rôle de `realm-management`.

**5. Secret du client `transactions`**
- `client_secret` accepté pour le POC : face à un pod compromis, une clé privée de `private_key_jwt` est aussi exposée qu'un secret ; le gain est faible.
- Conditions : secret aléatoire régénéré à chaque session, absent du realm JSON (substitution par variable d'environnement à l'import), monté dans le seul pod Transactions, jamais journalisé. L'identité d'exécution partagée du Learner Lab le rend lisible par tous les services : écart connu (ES-38).

**6. Durées**
- Jeton d'accès 5 min, jeton de service 5 min, session inactive 30 min : acceptés. Ajouter une durée maximale de session (8 h) et une tolérance d'horloge de 30 s au plus dans `jwx`.
- Déconnexion : fin de session chez Keycloak, révocation du jeton de rafraîchissement, mémoire vidée. Un jeton d'accès reste valable jusqu'à son expiration (5 min) : résiduel accepté et écrit.
- Changement de mot de passe : fermeture des autres sessions imposée (ES-02).
- Admin, ES-10 : les actions à fort impact exigent une authentification récente (`auth_time` de moins de 5 min, sinon nouvelle authentification) ; en V1 au plus tard.

**7. Rôle vendeur en V1**
- Écarter un compte de service doté de `manage-users` : il gère tous les comptes, leurs identifiants et leurs sessions, ce qui en ferait une cible équivalente à l'admin.
- Préférer : la Place de marché vérifie les préconditions (email vérifié, conditions acceptées, coordonnées de reversement), puis un compte de service limité par permissions fines à l'attribution du seul rôle `vendeur` ; action tracée (ES-23). À défaut de permissions fines démontrées, attribution par l'admin. Jamais par un attribut choisi par l'utilisateur.

**8. Sous-domaine ou chemin**
- **Sous-domaine `auth.` retenu** : origine distincte, donc les pages et cookies de Keycloak sont hors de portée d'un script injecté dans le front, et chaque origine garde sa propre CSP.
- Condition : aucun cookie posé avec `Domain=<domaine>` par l'un ou l'autre ; les deux origines étant du même site, `SameSite` ne les sépare pas.

**9. Mesures minimales si A et V1 sont retenues**
- MA1. Realm JSON relu à chaque modification et contrôlé en CI : URI de redirection et origines web exactes (aucun joker), flux implicite et « mot de passe » désactivés, `Full scope allowed` désactivé, aucun secret.
- MA2. `collector-web` : client public, PKCE `S256` obligatoire, flux standard seul. `transactions` : comptes de service seuls, sans flux standard.
- MA3. Portée `reserver` attribuée au seul client `transactions` ; test négatif : `collector-web` ne l'obtient pas.
- MA4. Vérification `jwx` dans `libs/platform/` : algorithme asymétrique imposé, `iss` exact, `aud`, `exp`, `typ` ; `azp` = `collector-web` sur les routes d'utilisateur, `azp` = `transactions` sur « réserver ».
- MA5. Compte de service de `transactions` sans aucun rôle de realm ; test : son jeton est refusé sur une route d'acheteur.
- MA6. « Réserver » : les deux en-têtes obligatoires, vérifiés dans l'ordre du point 3 ; acheteur lu dans le seul `sub` du second jeton ; en-tête masqué dans journaux et traces.
- MA7. ALB en liste blanche (point 4) ; test de non-joignabilité de `/admin/`, `/realms/master/` et du port 9000 à chaque déploiement.
- MA8. Politique réseau effective sur EKS, prouvée par un test de refus.
- MA9. Compte d'administration initial temporaire, remplacé par un compte nominatif avec OTP ; rôle `admin` sans droit sur `realm-management`.
- MA10. Comptes de test : mots de passe aléatoires régénérés à chaque session depuis Secrets Manager ; compte admin de test soumis à l'OTP. Aucun mot de passe connu sur un environnement public.
- MA11. Rotation du jeton de rafraîchissement avec détection de réutilisation ; session maximale 8 h ; tolérance d'horloge 30 s au plus ; sessions fermées au changement de mot de passe.
- MA12. MFA admin par flux conditionnel ; test : OTP demandé à l'admin, pas au vendeur.
- MA13. Force brute active sans verrouillage permanent, politique de mot de passe et liste de mots de passe interdits (ES-09) ; message identique que le compte existe ou non.
- MA14. Image Keycloak figée par empreinte, analysée par Trivy (bloquant en critique et élevée corrigeables), mise à jour à chaque version corrective de sécurité.

**10. Menaces à ajouter au registre (§ 4)**
- M-IDP-01 (S, E ; FC3) : console d'administration, realm `master` ou compte d'administration initial joignables ou laissés actifs : prise de contrôle de toutes les identités. Proposée 2 × 4, Élevé.
- M-IDP-02 (E ; FC2, FC11) : confusion de jetons : jeton de service accepté sur une route d'utilisateur (rôles par défaut du compte de service), ou jeton d'utilisateur accepté sur « réserver ». Proposée 2 × 3, Modéré.
- M-IDP-03 (S ; FC11) : rejeu latéral du jeton d'utilisateur à trois audiences par un service compromis. Proposée 2 × 3, Modéré.
- M-IDP-04 (S ; FC1, FC3) : comptes de test à mot de passe connu, dont un admin, sur l'environnement public. Proposée 2 × 4, Élevé.
- M-IDP-05 (E ; FC2) : erreur de paramétrage du realm versionné (URI de redirection à joker, flux « mot de passe », portée trop large) introduite sans revue. Proposée 2 × 3, Modéré.
- Risque résiduel V1 à inscrire : usage ou vol du jeton d'accès par un script injecté pendant la session, borné par MA11 et la CSP.

**Verdict** : GO pour l'option A avec V1, sous réserve de MA1 à MA14 ; repli B acceptable aux conditions du point 1 ; C écartée.

## Avis faisabilité

Fonction : Faisabilité — 2026-10-26. Périmètre : option A et V1, mesures MA1 à MA14 de l'avis sécurité, points soumis aux avis. Référence : D03 (densité, allumage, coûts), D04 (stack), [environnement-learner-lab.md](../00-cadrage/environnement-learner-lab.md).

**Synthèse.** Avis favorable à A avec V1. Rien ne bloque techniquement ; deux points sont à tenir : la mémoire de Keycloak sur deux t3.medium, et un émetteur paramétré dès le premier jour pour ne pas dépendre de D08. L'effort de 2,5 à 3,5 j sous-estime les mesures MA : compter **5 à 6 j**.

### Exploitation

- **Mémoire et densité** :
  - Places : Keycloak ajoute 1 pod ; 26 au repos pour 34 places (D03). Les places ne sont pas le problème.
  - Mémoire : environ 3,3 Gio allouables par t3.medium, soit 6,6 Gio pour deux nœuds. Système, KEDA, Cluster Autoscaler et Flux prennent environ 1 Gio ; les services Go quelques dizaines de Mio chacun [estimation]. Keycloak à 1 à 1,25 Gio tient sur deux nœuds, avec une marge réduite au pic.
  - Réglage : demande de 768 Mio, limite de 1,25 Gio, `-XX:MaxRAMPercentage=70`, cache local réduit (sessions peu nombreuses). Demande CPU de 500m, sans limite basse : le démarrage de la JVM est gourmand en CPU.
  - Troisième nœud au repos : **non par défaut**, seulement si la mesure (D14) montre un OOMKill ou des pods en attente au repos. Coût : 0,0416 $/h, soit 0,17 $ par session et environ 4 $ sur 25 sessions, sous la limite de 9 instances.
  - Piège de charge : le hachage des mots de passe coûte du CPU. Les tirs de D03 ne se connectent pas à chaque utilisateur virtuel ; ils réutilisent des jetons obtenus avant le tir.
- **Démarrage sur base neuve** :
  - Image optimisée construite en CI (`kc.sh build`, PostgreSQL, santé activée), version figée par empreinte (MA14). `start --optimized --import-realm` : l'import n'a lieu que si le realm est absent, ce qui est le cas à chaque session.
  - Ordre : RDS prête avant Keycloak (10 à 16 min de création, D03) ; Keycloak démarre en parallèle des services, qui n'en dépendent pas.
  - Sonde de démarrage sur `/health/ready` du port de gestion 9000, avec 3 min de tolérance ; 1 à 2 min attendues [?], à mesurer (D14).
  - Mise à jour de version : le format d'import peut changer ; tester l'import en CI à chaque montée de version.
- **Secrets régénérés à chaque `up`** (MA5, MA9, MA10) :
  - Le script d'allumage tire des valeurs aléatoires : secret de `transactions`, compte d'administration initial, mots de passe des comptes de test, graine OTP du compte admin de test.
  - Il les écrit dans un seul secret JSON de Secrets Manager (0,40 $ par secret et par mois, au prorata), puis en Secret Kubernetes. Pas d'External Secrets faute d'IRSA (Learner Lab).
  - Le realm JSON ne contient que des emplacements `${...}`, remplis à l'import par variables d'environnement (MA1).
  - Extinction : suppression sans fenêtre de récupération, sinon le nom du secret est bloqué à l'`up` suivant.
- **Sous-domaine `auth.`** :
  - Zone Route 53 et certificat ACM générique dans la couche persistante (0,50 $ par mois) ; enregistrement alias vers le nouvel ALB écrit par l'IaC à chaque `up`, TTL de 60 s.
  - Repli sans domaine stable (sslip.io) : l'émetteur change à chaque session. Il est déjà une variable unique injectée dans `KC_HOSTNAME`, les URI de redirection du realm, le `config.json` du front et la configuration des services, puisque le local l'exige aussi. Le repli ne coûte donc rien en code ; il coûte un certificat Let's Encrypt par session (limite de 5 identiques par semaine) et des URL à recopier pour la soutenance.
- **Liste blanche de l'ALB** (MA7) :
  - Une règle par hôte `auth.<domaine>`, en deux ou trois règles (5 chemins au plus par condition) ; réponse 404 par défaut. `hostname-admin` non défini.
  - Test automatisé après chaque déploiement, intégré à X17 : 200 sur le document de découverte et les pages de connexion ; 404 sur `/admin/`, `/realms/master/`, `/realms/collector/account` ; port 9000 injoignable.
- **Local** : docker compose avec la même image, un PostgreSQL, le même realm et les mêmes emplacements remplis par un `.env` non versionné ; émetteur `http://localhost:8080/realms/collector`. Le chart reste validé sur minikube (CR6). Mémoire sans contrainte sur le poste.

### Développement

- **Vérification `jwx` dans `libs/platform/`** (MA4, MA6) :
  - Jeu de clés en cache à rafraîchissement automatique, réessai si Keycloak est absent ; `jwt.Parse` avec algorithme RS256 imposé, émetteur exact, audience du service, tolérance d'horloge de 30 s.
  - Validateurs ajoutés pour les revendications propres à Keycloak : `typ` = `Bearer`, `azp` attendu par route, `scope` (chaîne séparée par des espaces) contenant `reserver`, rôles dans `realm_access.roles`.
  - Tests unitaires sans Keycloak : paire de clés générée en test, jetons forgés pour chaque refus (signature, émetteur, audience, `azp`, portée, expiration, `none`). Un test d'intégration contre le Keycloak de docker compose en CI.
- **Front React** : `oidc-client-ts` avec `react-oidc-context`, `userStore` en `InMemoryWebStorage`. Au rechargement, connexion silencieuse par redirection `prompt=none` : la session Keycloak sur `auth.` (même site) évite le formulaire. Rafraîchissement par jeton à rotation, sans iframe.
- **Playwright** : les jetons en mémoire ne se sauvegardent pas, mais le cookie de session de `auth.` oui. Un projet de préparation se connecte une fois par rôle et garde l'état ; chaque test repasse par une redirection silencieuse. Mots de passe lus dans l'environnement ; OTP de l'admin calculé à partir de la graine de session (bibliothèque TOTP). Tests de MA3, MA5 et MA12 inclus.
- **Comptes de test** : un par rôle (visiteur sans compte, acheteur, vendeur, admin), créés par l'import avec leurs rôles et des mots de passe en emplacement ; admin avec OTP préenregistré (MA10).

**Effort**

| Lot | Jours |
|---|---|
| Image, chart, sondes, docker compose | 0,75 |
| Realm JSON (clients, rôles, MFA conditionnel, force brute) et contrôle en CI (MA1) | 1 à 1,25 |
| Secrets à chaque `up`, DNS, règles de l'ALB, test X17 | 0,75 à 1 |
| `libs/platform/` : vérification et tests négatifs | 1 à 1,25 |
| Front OIDC | 0,75 |
| Playwright, comptes de test, OTP | 0,75 à 1 |
| **Total** | **5 à 6 j**, avant le sprint 3 (D04) |

**Risques principaux**
1. Mémoire et CPU de la JVM sur t3.medium : OOMKill ou démarrage lent au pic. Parade : mesure en D14, réglage ci-dessus, troisième nœud à 4 $ ; repli Zitadel ou B en dernier recours.
2. D08 sans domaine stable : émetteur changeant, URL de soutenance à reconfigurer. Parade : émetteur paramétré dès le premier jour.
3. Realm en code fragile : emplacements, format d'import, MFA conditionnel mal réglé. Parade : import et tests négatifs en CI à chaque modification.
4. Connexion silencieuse et Playwright : tests instables si la session de `auth.` expire ou si le cookie est mal partagé. Parade : préparation par rôle, durées de session de test explicites.

**Verdict** : GO pour A avec V1, effort de 5 à 6 j ; pas de troisième nœud avant mesure ; émetteur paramétré dès le sprint 1.

## Décision de Martin

> **« Keycloak »** (option A) : serveur d'autorisation en conteneur dans EKS, base dans RDS, avec les mesures de sécurité MA1 à MA14. Statut **Candidate** jusqu'à la mesure de mémoire et de démarrage en bac à sable (D14). Repli : option B (Cognito) sous les conditions de l'avis sécurité (audience contrôlée par `client_id` et `scope`, testée ; second facteur imposé à chaque admin).
>
> **Navigateur : « SPA, jetons en mémoire »** : application statique React, OIDC avec PKCE, jetons d'accès de 5 min en mémoire, rotation du jeton de rafraîchissement. L'intermédiaire serveur (BFF) est la cible V1.
>
> Option C (authentification maison) : écartée.

Date : 2026-10-26

## Conséquences

L'option A est retenue ; ses conséquences :

- **Expérimentation (D14)** : D02 prévoit l'appel synchrone authentifié dans l'expérimentation du serveur d'autorisation ou du canal. Critères candidats, à figer dans le protocole :
  - mémoire au repos et au pic de Keycloak sur EKS ; démarrage sur base neuve avec import du realm ;
  - « réserver » refusé sans jeton, avec un jeton d'utilisateur, avec le jeton d'un autre client, expiré ou d'une autre audience ; accepté avec le jeton de `transactions` ;
  - `/admin/` et `/realms/master/` injoignables depuis Internet (X17) ;
  - MFA demandé à l'admin, pas au vendeur.
- **D05** : authentification du canal par jetons Keycloak seulement si le broker la prend en charge ; sinon identités propres au broker.
- **D07** : schéma et compte dédiés à Keycloak dans RDS, recréés à chaque session.
- **D08** : domaine stable et sous-domaine `auth.` ; règles de l'ALB en liste blanche pour Keycloak ; enregistrement repointé à chaque `up` ; `connect-src` et origine de l'émetteur dans la CSP.
- **D10** : realm JSON versionné et analysé (aucun secret dedans) ; secrets des clients et du compte d'administration dans Secrets Manager.
- **D11** : Keycloak démarré en parallèle des services ; image optimisée construite en CI.
- **D13** : tests négatifs de BP2, BP5 et ES-38 ; un utilisateur de test par rôle, créé par l'import du realm.
- **Code** : `libs/platform/` porte la vérification `jwx` et le contrôle de rôle ; front configuré par `config.json` (émetteur lu à l'exécution, ENF-29).
- **Sécurité** : [exigences-securite.md](../securite/exigences-securite.md) révisé : ES-10 « Démontrée », ES-09 « Partielle », risque résiduel des jetons en mémoire, BFF cible V1.
- **Compétences (C9)** : OpenID Connect et Keycloak dans la cartographie.
- **INDEX** : ligne D06 et synthèse des décisions tranchées.

## À savoir défendre en soutenance

1. **« Pourquoi un Keycloak à gérer alors que Cognito est managé ? »**
   - Cognito retire un pod, et c'est notre repli.
   - Keycloak a été retenu pour trois besoins présents : une audience par service dans le jeton (exigence de l'appel « réserver »), le second facteur imposé au seul admin, et le même serveur en local et sur AWS.
   - Cognito n'était pas garanti dans l'environnement de formation.
2. **« Où sont vos jetons dans le navigateur, et que se passe-t-il en cas de XSS ? »**
   - En mémoire seulement, jamais dans `localStorage` ; jeton d'accès de 5 min.
   - Un script injecté pourrait utiliser ou voler le jeton pendant la session : c'est le risque résiduel, borné par la CSP sans script en ligne et la neutralisation des sorties.
   - La cible de production est un intermédiaire serveur à cookie `HttpOnly`, qui empêche le vol du jeton.
3. **« Comment un service sait-il que l'appel vient bien de Transactions, pour le bon acheteur ? »**
   - Jeton de service : signature, émetteur, audience « Place de marché », portée « réserver » que seul Transactions peut obtenir.
   - Jeton de l'acheteur transmis à part et revérifié ; l'acheteur est lu dans le jeton, jamais dans la requête. Démonstration : les quatre refus, puis l'acceptation.
4. **« Le rôle dans le jeton suffit-il ? »**
   - Non : le rôle dit « est vendeur », la propriété dit « est le vendeur de cet article ». La seconde se vérifie dans les données du service. Démonstration : vendeur A accepté, vendeur B refusé.
5. **« Comment votre émetteur reste-t-il le même si l'ALB est recréé à chaque session ? »**
   - L'émetteur est un nom de domaine fixé dans Keycloak, pas l'adresse de l'ALB ; l'IaC repointe le nom sur le nouvel ALB à chaque allumage.

## Sources

Rédaction du 2026-10-26. Les points marqués [?] sont à confirmer par la sonde du compte ou l'expérimentation.

- **Projet** : [D03](D03-hebergement-orchestration.md) (conséquences D06, MS2, MS6, X17, X19) ; [D04](D04-stack.md) (avis sécurité, faisabilité front et back) ; [environnement-learner-lab.md](../00-cadrage/environnement-learner-lab.md) § 0 et § 3 (Cognito [REX]) ; [exigences-securite.md](../securite/exigences-securite.md) § 5.2, § 5.5, § 6 ; [exigences.md](../00-cadrage/exigences.md) § Rôles.
- **Keycloak** : configuration du nom d'hôte https://www.keycloak.org/server/hostname ; interface de gestion https://www.keycloak.org/server/management-interface ; dimensionnement mémoire et CPU https://www.keycloak.org/high-availability/concepts-memory-and-cpu-sizing ; conteneurs et mode optimisé https://www.keycloak.org/server/containers ; guide d'administration (audience mapper, flux conditionnels, force brute) https://www.keycloak.org/docs/latest/server_admin/
- **Cognito** : jeton d'accès (`client_id`, pas d'`aud`) https://docs.aws.amazon.com/cognito/latest/developerguide/amazon-cognito-user-pools-using-the-access-token.html ; resource servers et portées https://docs.aws.amazon.com/cognito/latest/developerguide/cognito-user-pools-define-resource-servers.html ; tarifs https://aws.amazon.com/cognito/pricing/
- **Normes** : OAuth 2.0 Security Best Current Practice (RFC 9700) ; OAuth 2.0 for Browser-Based Applications https://datatracker.ietf.org/doc/html/draft-ietf-oauth-browser-based-apps ; JWT Profile for OAuth 2.0 Access Tokens (RFC 9068) ; Token Exchange (RFC 8693).
