# Identité : realm Keycloak `collector`

Realm décrit en code ([D06](../../docs/decisions/D06-authentification-autorisation.md), option A). Image épinglée : `quay.io/keycloak/keycloak:26.8.1` (empreinte dans [identite.go](identite.go)).

| Élément | Valeur |
|---|---|
| Realm | `collector` |
| Rôles de realm | `acheteur` (rôle par défaut), `vendeur`, `admin` (composite de `moderation` et `administration`, ES-20) |
| Client du site | `collector-web` : public, code d'autorisation seul, PKCE `S256` obligatoire, implicite et « mot de passe » désactivés, `Full scope allowed` désactivé, URI de redirection exactes (`/rappel`) |
| Audiences du jeton d'accès | `place-de-marche`, `transactions`, `interactions` (`OIDC_AUDIENCE` des services) |
| Jeton d'accès | 5 min ; rafraîchissement à usage unique (rotation) |
| Second facteur | OTP TOTP (6 chiffres, 30 s) imposé au seul rôle `admin` (ES-10, US-022 AC6) |

Flux du navigateur `navigateur collector`, lié comme `browserFlow` du realm : copie du flux standard (cookie, redirection vers un fournisseur d'identité, formulaires) dont le sous-flux conditionnel `navigateur collector OTP admin` enchaîne « Condition - user role » (`condUserRole` = `admin`) et « OTP Form » (REQUIRED). Un admin sans OTP doit le configurer (`CONFIGURE_TOTP`) avant d'obtenir un code ; le vendeur et l'acheteur n'en voient jamais.

Aucun utilisateur ni secret dans le realm versionné : les comptes de démonstration sont créés à chaque session avec des mots de passe générés (US-008 AC3).

## Importer le realm en local

Depuis la racine du dépôt (Git Bash) :

```bash
MSYS_NO_PATHCONV=1 docker run --rm -p 8080:8080 \
  -e KC_BOOTSTRAP_ADMIN_USERNAME=amorce \
  -e KC_BOOTSTRAP_ADMIN_PASSWORD="$(openssl rand -hex 16)" \
  -v "$(pwd -W)/infra/identite/realm-collector.json:/opt/keycloak/data/import/realm-collector.json:ro" \
  quay.io/keycloak/keycloak:26.8.1 start-dev --import-realm
```

Sous Linux ou macOS, remplacer `$(pwd -W)` par `$(pwd)`. Le realm est prêt quand `http://localhost:8080/realms/collector/.well-known/openid-configuration` répond 200 (environ une minute). Le mot de passe d'amorce n'est affiché nulle part : pour administrer, le fixer soi-même dans une variable de session. Hors `start-dev`, la même commande d'import vaut pour `kc.sh start --import-realm` ou `kc.sh import --file`.

## Ce que prouve le test

`TestSecondFacteurDuRealmCollector` démarre l'image épinglée avec ce realm importé (testcontainers-go), crée par l'API d'administration un vendeur et un admin aux mots de passe aléatoires, puis joue le flux du navigateur en HTTP (code d'autorisation + PKCE `S256`) :

- vendeur : redirection directe vers `/rappel` avec un `code` ; le jeton obtenu porte `vendeur`, pas `admin`, `azp` = `collector-web`, l'audience `place-de-marche` et dure 300 s ;
- admin sans OTP : page de configuration `CONFIGURE_TOTP`, aucun code ; le code n'est délivré qu'après saisie d'un TOTP valide ;
- admin avec OTP : formulaire OTP après le mot de passe, OTP faux refusé sans code, code délivré après un OTP valide ;
- demande sans PKCE ou en PKCE `plain` refusée (`invalid_request`) ; URI de redirection hors liste refusée (400).

```bash
cd infra/identite && GOWORK=off go test -count=1 -v ./...
```

Docker requis ; le test est ignoré si aucun fournisseur de conteneurs n'est joignable. En CI, le module est un module Go comme les services (`tools/ci/changements.sh` liste `infra/*/go.mod`) : job « Go (infra/identite) », sans image.
