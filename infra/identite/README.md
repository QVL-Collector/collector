# Identité : realm Keycloak `collector`

Realm décrit en code ([D06](../../docs/decisions/D06-authentification-autorisation.md), option A). Image épinglée : `quay.io/keycloak/keycloak:26.8.1` (empreinte dans [identite.go](identite.go)).

| Élément | Valeur |
|---|---|
| Realm | `collector` |
| Rôles de realm | `acheteur` (rôle par défaut : `default-roles-collector`, composite d'`acheteur`, déclaré aussi dans `roles.realm`, sans quoi l'import ne l'applique pas, constat XP03), `vendeur`, `admin` (composite de `moderation` et `administration`, ES-20) |
| Client du site | `collector-web` : public, code d'autorisation seul, PKCE `S256` obligatoire, implicite et « mot de passe » désactivés, `Full scope allowed` désactivé, URI de redirection exactes (`/rappel`) ; portées `basic`, `roles`, `profile`, `web-origins` |
| Audiences du jeton d'accès | `place-de-marche`, `transactions`, `interactions` (`OIDC_AUDIENCE` des services) |
| Client de service | `transactions` (US-030) : confidentiel (`client-secret`), comptes de service seuls (aucun flux standard, implicite, « mot de passe »), `Full scope allowed` désactivé, portées `basic` et `reserver` ; jeton de 5 min sans jeton de rafraîchissement |
| Portée `reserver` | Attribuée au seul client `transactions` (MA3) ; ajoute l'audience `place-de-marche` et figure dans `scope` |
| Compte de service | `service-account-transactions` déclaré dans le realm avec `realmRoles: []` : aucun rôle de realm, pas même le rôle par défaut `acheteur` (piège de D06, MA5) ; son jeton ne porte ni `realm_access` ni l'audience `transactions` |
| Jeton d'accès | 5 min ; rafraîchissement à usage unique (rotation) |
| Second facteur | OTP TOTP (6 chiffres, 30 s) imposé au seul rôle `admin` (ES-10, US-022 AC6) |

Les portées sont déclarées dans le realm (`clientScopes`) : un realm qui en déclare une ne reçoit plus les portées intégrées de Keycloak à l'import, d'où `basic` (`sub`, `auth_time`), `roles` (`realm_access.roles`), `profile` (nom et `preferred_username` dans le jeton d'identité, lus par le front) et `web-origins` écrites ici, sur le modèle validé par XP03. `email` et `acr` ne sont plus émises : aucun service ni le front ne les lit.

Flux du navigateur `navigateur collector`, lié comme `browserFlow` du realm : copie du flux standard (cookie, redirection vers un fournisseur d'identité, formulaires) dont le sous-flux conditionnel `navigateur collector OTP admin` enchaîne « Condition - user role » (`condUserRole` = `admin`) et « OTP Form » (REQUIRED). Un admin sans OTP doit le configurer (`CONFIGURE_TOTP`) avant d'obtenir un code ; le vendeur et l'acheteur n'en voient jamais.

Aucun utilisateur ni secret dans le realm versionné (le compte de service n'a pas d'identifiant de connexion) : les comptes de démonstration sont créés à chaque session avec des mots de passe générés (US-008 AC3). Le secret du client `transactions` est l'emplacement `${SECRET_CLIENT_TRANSACTIONS}`, rempli à l'import par la variable d'environnement de même nom, tirée au hasard à chaque session et remise à Transactions dans `OIDC_CLIENT_SECRET` (D06 avis sécurité § 5). **Variable obligatoire** : absente, Keycloak garderait le texte de l'emplacement comme secret, connu de tous. La commande d'import ci-dessous s'arrête si elle est vide (`${SECRET_CLIENT_TRANSACTIONS:?}`), le test échoue si le secret importé n'est pas la valeur tirée, et le script d'allumage devra appliquer la même règle.

## Importer le realm en local

Depuis la racine du dépôt (Git Bash) :

```bash
MSYS_NO_PATHCONV=1 docker run --rm -p 8080:8080 \
  -e KC_BOOTSTRAP_ADMIN_USERNAME=amorce \
  -e KC_BOOTSTRAP_ADMIN_PASSWORD="$(openssl rand -hex 16)" \
  -e SECRET_CLIENT_TRANSACTIONS="${SECRET_CLIENT_TRANSACTIONS:?à tirer pour la session}" \
  -v "$(pwd -W)/infra/identite/realm-collector.json:/opt/keycloak/data/import/realm-collector.json:ro" \
  quay.io/keycloak/keycloak:26.8.1 start-dev --import-realm
```

Sous Linux ou macOS, remplacer `$(pwd -W)` par `$(pwd)`. Le realm est prêt quand `http://localhost:8080/realms/collector/.well-known/openid-configuration` répond 200 (environ une minute). Le mot de passe d'amorce n'est affiché nulle part : pour administrer, le fixer soi-même dans une variable de session. Hors `start-dev`, la même commande d'import vaut pour `kc.sh start --import-realm` ou `kc.sh import --file`.

## Ce que prouve le test

`demarrerKeycloak` (`keycloak_test.go`) tire `SECRET_CLIENT_TRANSACTIONS` au hasard pour chaque conteneur et le garde dans `keycloakDeTest.secretTransactions`, pour obtenir un jeton de service par `client_credentials`. Après l'import, il relit le secret du client `transactions` par l'API d'administration et échoue s'il diffère de la valeur tirée (emplacement `${...}` non substitué). `creerUtilisateur(t, nom, roles...)` attribue toujours `acheteur` explicitement, plus les rôles demandés : un compte cumule acheteur et vendeur (EF-07).

`TestSecondFacteurDuRealmCollector` démarre l'image épinglée avec ce realm importé (testcontainers-go), crée par l'API d'administration un vendeur et un admin aux mots de passe aléatoires, puis joue le flux du navigateur en HTTP (code d'autorisation + PKCE `S256`) :

- vendeur : redirection directe vers `/rappel` avec un `code` ; le jeton obtenu porte `vendeur`, pas `admin`, `azp` = `collector-web`, l'audience `place-de-marche` et dure 300 s ;
- admin sans OTP : page de configuration `CONFIGURE_TOTP`, aucun code ; le code n'est délivré qu'après saisie d'un TOTP valide ;
- admin avec OTP : formulaire OTP après le mot de passe, OTP faux refusé sans code, code délivré après un OTP valide ;
- demande sans PKCE ou en PKCE `plain` refusée (`invalid_request`) ; URI de redirection hors liste refusée (400).

```bash
cd infra/identite && GOWORK=off go test -count=1 -v ./...
```

Docker requis ; le test est ignoré si aucun fournisseur de conteneurs n'est joignable. En CI, le module est un module Go comme les services (`tools/ci/changements.sh` liste `infra/*/go.mod`) : job « Go (infra/identite) », sans image.
