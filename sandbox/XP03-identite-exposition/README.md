# XP03 — Identité et exposition : bac à sable local

Partie locale du protocole [XP03](../../docs/experimentations/XP03-identite-exposition.md) (§ B) : docker compose, puis k3d. La session EKS reprend le realm, l'image optimisée, le service de test et les scripts de vérification (`PLATEFORME=eks`) ; les règles d'ALB et le DNS relèvent de l'IaC de `sandbox/`.

## Contenu

| Chemin | Rôle |
|---|---|
| `versions.env` | Images épinglées (version et empreinte) |
| `keycloak/Dockerfile` | Image Keycloak optimisée (`kc.sh build` : PostgreSQL, santé, fonctions inutiles désactivées) + liste de mots de passe interdits |
| `realm/collector-realm.json` | Realm `collector` versionné, valeurs sensibles en emplacements `${...}` seulement (MA1) |
| `caddy/Caddyfile` | Proxy local `tls internal` : liste blanche de MA7, 404 ailleurs, refus des traversées et encodages (ME8), 80 → 301 |
| `verificateur/envoy.yaml` | Service de test vérifiant le jeton (Envoy, configuration seule) : signature, `iss` public exact, `aud`, `exp` (30 s de tolérance), `typ`, `azp`, portée `reserver` ; route d'acheteur `/achat` (MA5) ; refus en 401 |
| `docker-compose.yml` | PostgreSQL 18.3 (base neuve à chaque lancement), Keycloak, vérificateur (port interne 8081, jamais publié), Caddy (127.0.0.1:80/443) |
| `k8s/` | Même pile sur k3d : namespaces, identité, vérificateur et appelant, politiques réseau (G7-4, MA8) |
| `scripts/` | `identifiants.sh`, `compose.sh`, `verifier.sh`, `k3d.sh`, bibliothèques `lib.sh` et `oidc.sh` |
| `.env`, `.genere/` | Générés à chaque session, ignorés par git |

## Realm de test

- Clients : `collector-web` (public, code d'autorisation + PKCE S256 seul, URI exactes `${URL_FRONT}/rappel`), `transactions` (confidentiel, comptes de service seuls, portée par défaut `reserver` → `aud` `place-de-marche`), `xp03-autre` (confidentiel, portées optionnelles `aud-place-de-marche`, `aud-interactions`). `Full scope allowed` désactivé partout ; flux implicite et « mot de passe » désactivés.
- Rôles : `acheteur` (par défaut), `vendeur`, `moderation`, `administration`, `admin` (composite). Comptes `acheteur-test`, `vendeur-test`, `admin-test` (OTP préenregistré depuis `GRAINE_OTP`).
- Jeton d'accès 5 min, session inactive 30 min, session maximale 8 h, rotation du jeton de rafraîchissement (réutilisation 0), force brute sans verrouillage permanent, politique `length(12)`, `notUsername`, `notEmail`, liste interdite.
- Flux navigateur `navigateur-collector` : cookie, sinon formulaire, puis OTP conditionnel au rôle `admin` (MA12).
- Compte d'administration initial `admin-temporaire` (realm `master`), mot de passe de session ; API d'administration jamais routée par le proxy, utilisée par l'adresse interne seulement.

## Prérequis

Docker 28.5.1, k3d 5.9.0, kubectl, `jq`, `openssl`, `xxd`, curl du poste (Git Bash). Ports 80 et 443 libres sur `127.0.0.1` (sinon `PORT_HTTP`, `PORT_HTTPS`). `*.localhost` est résolu en boucle locale par curl et les navigateurs. Les scripts utilisent l'autorité locale de Caddy (`.genere/caddy-racine.crt`, jamais `-k`) ; `--ssl-no-revoke` est requis par le curl Schannel de Windows.

## Déroulé local (protocole § B)

```bash
scripts/compose.sh up                       # B1 : secrets, image optimisée, pile prête
scripts/verifier.sh k2                      # B1 : K2 indicatif, 2 mesures sur base neuve (pile recréée)
scripts/verifier.sh xe4                     # B2 : liste blanche (XE4 + ME8), port 9000, 80 -> 301
scripts/verifier.sh k3                      # B3 : 5 x 401, 1 x 200 (+ MA5, MA3) ; ~5,5 min (jeton expiré)
scripts/verifier.sh ma12                    # B4 : OTP demandé à l'admin, pas au vendeur
scripts/verifier.sh ma                      # MA1, MA2, MA3, MA5, MA9, MA11, MA13 (paramètres et refus de flux)
scripts/verifier.sh trivy                   # MA14 : Trivy sur l'image optimisée
scripts/compose.sh down                     # supprime conteneurs, réseaux, volume Caddy, .env, .genere/
```

`scripts/verifier.sh tout` enchaîne `xe4`, `ma12`, `ma`, `k3`. Résultats : `docs/mesures/XP03-identite-exposition.csv` et `docs/mesures/XP03-identite-exposition/` (seule la charge utile décodée `aud`, `azp`, `scope`, `typ`, `exp`, rôles est consignée, MX5) ; `MESURES_DIR` les redirige pour un essai à blanc.

## k3d (protocole § B5)

Contexte kubectl courant sur le cluster k3d de XP01 (ou le cluster jetable de XP02 : `../XP02-canal-rabbitmq/scripts/k3d.sh cluster-essai up`).

```bash
scripts/k3d.sh up                           # image importée, Secrets, realm en ConfigMap, pile prête
scripts/k3d.sh k2                           # K2 : création du pod -> Ready, 2 mesures, base neuve
PLATEFORME=k3d scripts/verifier.sh k3       # K3 depuis l'appelant de transactions-test
PLATEFORME=k3d scripts/verifier.sh ma12
scripts/k3d.sh reseau                       # G7-4 : ns-intrus refusé ; témoins depuis transactions-test
scripts/k3d.sh down
```

Sans proxy sur k3d, le poste joint Keycloak par `port-forward` (`localhost:18080`, ouvert par `verifier.sh`) : l'émetteur de débogage y est `http://localhost:18080/realms/collector`.

## Constats de préparation (2026-10-26, à blanc)

- Compose : pile prête en 65 s (construction de l'image comprise au premier lancement) ; K2 43 s puis 31 s ; XE4 + ME8 32/32 (traversées, encodages, casse, barre oblique inverse : 404) ; K3 6/6 ; MA12 5/5 ; MA 18/18.
- k3d : K2 156 s puis 91 s (sous le seuil de 3 min, mais la première mesure s'en approche : sur EKS, image tirée du registre en plus) ; K3 9/9 ; MA12 5/5 ; G7-4 9/9.
- Le rôle par défaut `acheteur` n'est appliqué que si `default-roles-collector` est déclaré dans `roles.realm` (la clé `defaultRole` seule ne suffit pas à l'import).
- Le compte de service de `transactions` n'a aucun rôle de realm (piège de l'avis sécurité de D06 écarté) et son jeton n'en porte aucun.
- MA14 (Trivy) non validé : téléchargement de la base Java de Trivy en erreur d'entrée-sortie, puis arrêt de Docker Desktop (disque C: plein à 98 %). Le script signale désormais une analyse non aboutie comme échec.
- L'OTP doit être saisi dans sa période : un essai est rejoué à la période suivante en cas de décalage d'horloge entre le poste et Docker.
