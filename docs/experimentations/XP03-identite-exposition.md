# XP03 — Identité et exposition : Keycloak, ALB, ACM, Route 53

| Champ | Valeur |
|---|---|
| Décisions testées | [D06](../decisions/D06-authentification-autorisation.md) et [D08](../decisions/D08-exposition-tls-domaine.md) (statut `Candidate (XP03)`) |
| Dates prévues | XE1, XE2 pendant XP00 (2026-10-27) ; local 2026-10-29 ; EKS session 2 2026-10-31 |
| Date d'exécution | 2026-10-28, locale : docker compose de 09 h 07 à 09 h 22 UTC, k3d (cluster jetable `xp02-essai`) de 09 h 44 à 10 h 04 UTC ; XE1, XE2 et EKS non exécutés |
| Exécutée par | Martin |
| Scripts | `sandbox/XP03-identite-exposition/` (realm de test, règles d'ALB, service de test vérifiant l'audience, proxy local) |
| Dépend de | [XP00](XP00-sonde-compte.md) (S6, S7), [XP01](XP01-orchestrateur.md) (cluster, ALB, RDS) |

## Objectif et critères de succès

Objectif : prouver un point d'entrée HTTPS à certificat managé qui n'expose que la liste blanche, et un Keycloak qui tient dans le cluster et protège « réserver » par l'audience du jeton.

Hypothèse : la zone Route 53 `collector.qvl-project.com`, déléguée depuis Cloudflare, et le certificat ACM générique servent l'ALB ; Keycloak (≈ 1 à 1,25 Gio) tient sur deux t3.medium avec RabbitMQ et la charge de test, et démarre sur base neuve en moins de 3 min.

### Noms

| Nom | Rôle | Enregistrement |
|---|---|---|
| `collector.qvl-project.com` | Front et API (service de test) | Alias `A` vers l'ALB, couche éphémère, TTL 60 s |
| `auth.collector.qvl-project.com` | Keycloak, pages de connexion seulement | Alias `A` vers l'ALB, couche éphémère, TTL 60 s |
| Certificat ACM | `collector.qvl-project.com` et `*.collector.qvl-project.com` | Couche persistante, validation DNS dans la zone Route 53 |

Repli D2 (Route 53 refusé en XP00) : les deux noms en `CNAME` vers le nom DNS de l'ALB, posés dans Cloudflare en « DNS only », repointés à chaque session.

### Critères

| # | Mesure | Seuil | Bloquant | Si échec | Mesure obtenue (local, 2026-10-28) | Atteint |
|---|---|---|---|---|---|---|
| XE1 | Délégation `NS` de `collector.qvl-project.com` (XP00 S6), puis résolution des deux alias depuis 1.1.1.1 et 8.8.8.8 | Nom DNS de l'ALB résolu | **Oui** | Repli D2 (Cloudflare) | Non exécuté (DNS public, EKS) | Reporté à la session EKS |
| XE2 | Certificat ACM `ISSUED` (XP00 S7), attaché à l'écouteur 443, politique `ELBSecurityPolicy-TLS13-1-2-Res-2021-06` | `openssl s_client` montre le certificat ACM pour les deux noms | **Oui** | Chaîne 2 de D08 | Non exécuté (ACM, ALB) | Reporté à la session EKS |
| XE4 + ME8 | Liste blanche vue de l'adresse de Martin, avec `curl --path-as-is` : « réserver », `/admin/`, `/realms/master/`, `/realms/collector/account`, port 9000, broker ; traversées et encodages (`..`, `%2e%2e`, `%2f`, `%2F`, casse `/ADMIN/`, `/Realms/Master/`) | 404 (ou délai expiré pour les ports) sur 100 % des chemins interdits ; 200 sur le document de découverte et la page de connexion | **Oui** | Règles d'ALB corrigées et rejouées avant verdict ; non corrigeable : chaîne revue (D08) | compose (Caddy) : découverte, `certs`, page de connexion 200 ; 26/26 chemins interdits 404 ; `/reserver` 404 ; 9000 sans réponse | Oui en local ; ALB reporté à la session EKS |
| K1 | Mémoire de Keycloak au repos (15 min) et au pic, avec RabbitMQ et la charge de test, sur deux t3.medium | Aucun `OOMKilled`, aucun pod `Pending` | **Oui** | Troisième nœud au repos (D06, D09) ; à défaut repli Cognito si S13 le permet | Indicatif : Keycloak 651 Mio (compose) et 671 Mio (k3d) après les tests, aucun `OOMKilled` | Reporté à la session EKS |
| K2 | Démarrage sur base neuve avec import du realm : de la création du pod à `/health/ready` 200 (port 9000), 2 mesures | ≤ 3 min | **Oui** | Réglage JVM ; sinon repli de D06 | compose 29 s et 30 s ; k3d 110 s (tirage de PostgreSQL compris) et 41 s | Oui en local ; EKS reporté à la session EKS |
| K3 | « Réserver » (port interne du service de test, depuis le namespace `transactions-test`) | Refusé (401) sans jeton, avec un jeton d'utilisateur, avec le jeton d'un autre client, avec un jeton expiré, avec un jeton d'une autre audience ; accepté (200) avec le jeton de `transactions` (portée `reserver`, `aud` `place-de-marche`, `azp` `transactions`) | **Oui** | Configuration du realm ou de la vérification corrigée ; défaut structurel : repli de D06 | compose et k3d : 5 × 401, 1 × 200 (9/9 avec MA3 et MA5) | Oui en local ; EKS reporté à la session EKS |
| G7-4 (X19) | Pod quelconque vers la console Keycloak (8080 `/admin/`, 9000) ; flux témoin : ALB → page de connexion | Refus ; témoin 200 | **Oui** | Voir G7 de XP01 | k3d : `ns-intrus` → 8080, 9000, 8081, 5432 fermés ; témoins depuis `transactions-test` ouverts, page de connexion 200 | Oui (débogage) ; EKS reporté à la session EKS |
| XE3 | Analyse TLS de `collector.qvl-project.com` et `auth.collector.qvl-project.com` | TLS 1.2 minimum ; 80 redirigé en 301 ; HSTS présent ; 404 sur hôte inconnu | Non | IaC corrigée (recette du sprint 1) | Indicatif compose : 80 → 301 ; pas de HSTS en local | Reporté à la session EKS |
| MA12 | MFA : OTP demandé à l'admin de test, pas au vendeur de test | Constat | Non | Realm corrigé | compose et k3d : OTP demandé à l'admin, pas au vendeur (5/5) | Oui en local |
| Densité D09 | Mémoire de la pile d'observabilité allégée ajoutée au relevé de K1, si le temps le permet | Relevé | Non | Estimation de D09 conservée | Non exécuté | Reporté à la session EKS |
| MX2 | Depuis CloudShell (autre adresse) : `curl https://collector.qvl-project.com/` | Délai expiré | Non (contrôle) | Groupe de sécurité de l'ALB corrigé | Non exécuté (CloudShell) | Reporté à la session EKS |

XE5 (limitation de débit) est reportée en AC du sprint 1 (option B de D14), bloquante avant toute ouverture de l'ALB à Internet (ME4).

## Environnement

- **Local** : docker compose (Keycloak, PostgreSQL, service de test vérifiant le jeton, Caddy en `tls internal`) sur `collector.localhost` et `auth.collector.localhost`, puis le chart sur k3d. Même realm, même test de liste blanche qu'en cloud. Pas de HSTS en local.
- **AWS** : cluster EKS de XP01 (session 2), us-east-1 ; ALB de XP01 avec deux règles par hôte, liste blanche de Keycloak (MA7) en deux ou trois règles, action par défaut 404, écouteur 80 en 301 ; base Keycloak dans RDS (schéma et compte dédiés) ; groupe de sécurité de l'ALB au /32 de Martin (MX2).
- **Keycloak** : demande 768 Mio, limite 1,25 Gio, `-XX:MaxRAMPercentage=70`, demande CPU 500m ; `start --optimized --import-realm` ; `KC_HOSTNAME=https://auth.collector.qvl-project.com`.

### Versions à épingler

| Élément | Version | État |
|---|---|---|
| Keycloak (image officielle, construite en mode optimisé) | 26.8.1 : `quay.io/keycloak/keycloak:26.8.1@sha256:a78d6b79572168cb25886b80a8b5b8d4ed4390743e3eecb810538683e209dd6b`, image optimisée construite par `sandbox/XP03-identite-exposition/keycloak/Dockerfile` | Fixée |
| Registre de l'image optimisée | GHCR privé (D10) ou dépôt ECR de bac à sable supprimé en fin de session | À fixer |
| PostgreSQL local / RDS | 18.3 (XP00 S10) ; image locale `postgres:18.3@sha256:7e32e9833a6fb1c92c32552794cb6ed569d51b445a54907d35fc112ef39684db` | Fixée |
| Caddy (proxy local) | 2.11.7 : `caddy:2.11.7-alpine@sha256:eb4d35b7452d1ab6171fbdbe8d9275b2301dad6dad92115d91e19fb9b24d510f` | Fixée |
| Provider Terraform AWS | celui de XP01 ; vérifier qu'il expose les attributs d'en-têtes de réponse de l'écouteur (HSTS) | À vérifier |
| Outil d'analyse TLS | `drwetter/testssl.sh:3.2@sha256:e855bd0bd9808f2d3f809a190bbedda3422329bdfc9f0c117822bf5f0a2b3474` | Fixée |
| OpenSSL, curl | curl 8.17.0 (Schannel) ; OpenSSL 3.5.4 (Git Bash) | Relevée |
| EKS, kubectl, Helm, k3d | voir [XP01](XP01-orchestrateur.md) | Communes |

## Technos et interactions

Navigateur ou `curl` (adresse de Martin) → Route 53 (`collector.qvl-project.com`, délégué depuis Cloudflare) → ALB (certificat ACM, règles par hôte et par chemin) → NodePorts → service de test et Keycloak. Keycloak → RDS. `transactions-test` → Keycloak (client credentials, adresse interne) → « réserver » du service de test (port interne, jamais routé). Le service de test lit le jeu de clés par l'adresse interne de Keycloak et vérifie `iss` public (ME6).

## Sécurité du bac à sable

| Mesure | Application dans XP03 |
|---|---|
| MX1 | Realm de test : comptes acheteur, vendeur, admin fictifs ; mots de passe, secret de `transactions`, graine OTP et compte d'administration initial générés à chaque session ; realm avec emplacements `${...}` seulement |
| MX2 | ALB au /32 de Martin pendant toute l'XP ; contrôle depuis CloudShell |
| MX4 | Aucun secret dans `sandbox/XP03-identite-exposition/` ; flux « mot de passe » et implicite désactivés dans le realm testé (MA1) |
| MX5 | Aucun jeton complet consigné : seuls la charge utile décodée (`aud`, `azp`, `exp`) et le code HTTP ; valeur piégée cherchée dans les journaux en fin de session |
| MX7 | Alias `collector` et `auth.collector` supprimés par le `destroy` et vérifiés à l'inventaire ; secret de session supprimé sans délai ; dépôt ECR de bac à sable supprimé s'il est utilisé |

## Étapes reproductibles

### A. Pendant XP00 (2026-10-27)

XE1 (S6) et XE2 (S7) : zone `collector.qvl-project.com`, quatre `NS` posés dans Cloudflare par Martin, vérification `nslookup`, certificat `PENDING_VALIDATION` puis `ISSUED`. Résultats repris ici, non rejoués.

### B. Local (2026-10-29)

1. docker compose : Keycloak, PostgreSQL, realm importé. Attendu : `/health/ready` 200 ; durée K2 relevée (indicative).
2. Liste blanche via Caddy : script XE4 + ME8 contre `https://auth.collector.localhost`. Attendu : même tableau de codes qu'en cloud.
3. K3 : six cas contre « réserver ». Attendu : 5 × 401, 1 × 200.
4. MA12 : connexion admin (OTP demandé) et vendeur (aucun OTP).
5. Chart sur k3d : rejouer 1 et 3 ; G7-4 en débogage.

### C. EKS, session 2 de XP01 (2026-10-31)

1. Après G2 n° 2 de XP01 : vérifier XE1 et XE2.

```bash
nslookup collector.qvl-project.com 1.1.1.1
nslookup auth.collector.qvl-project.com 8.8.8.8
echo | openssl s_client -connect collector.qvl-project.com:443 -servername collector.qvl-project.com 2>/dev/null | openssl x509 -noout -subject -ext subjectAltName
echo | openssl s_client -connect auth.collector.qvl-project.com:443 -servername auth.collector.qvl-project.com 2>/dev/null | openssl x509 -noout -subject -ext subjectAltName
```

   Attendu : adresses de l'ALB ; sujet et SAN couvrant les deux noms.
2. **K2** : base Keycloak neuve (schéma vide), pod créé, horodatage jusqu'à `/health/ready` 200 ; deuxième mesure après suppression du schéma et du pod.
3. **XE4 + ME8** depuis le poste :

```bash
for p in /realms/collector/.well-known/openid-configuration /admin/ /admin/master/console/ /realms/master/ \
  /realms/collector/account /realms/collector/../master/ /realms/collector/%2e%2e/master/ /%2e%2e/admin/ \
  /realms%2fmaster/ /realms%2Fmaster/ /ADMIN/ /Realms/Master/ /resources/../admin/; do
  echo "$p $(curl -s -o /dev/null -w '%{http_code}' --path-as-is "https://auth.collector.qvl-project.com$p")"
done
curl -s -o /dev/null -w '%{http_code}\n' --path-as-is https://collector.qvl-project.com/reserver
curl -s --max-time 5 -o /dev/null -w '%{http_code}\n' https://auth.collector.qvl-project.com:9000/health
```

   Attendu : 200 sur le document de découverte (et la page de connexion), 404 partout ailleurs, délai expiré sur 9000. Liste exacte des chemins figée dans le script avant la session.
4. **G7-4** : depuis `ns-intrus`, `curl --max-time 5` vers le Service Keycloak (8080 `/admin/`, 9000). Attendu : délai expiré ; témoin via l'ALB 200.
5. **K3** : depuis un pod de `transactions-test`, six appels à « réserver ». Jetons :
   - utilisateur : flux code d'autorisation avec PKCE pour l'acheteur de test (méthode du script, fixée avant l'exécution ; jamais le flux « mot de passe ») ;
   - autre client : client confidentiel de test `xp03-autre`, sans portée `reserver` ;
   - expiré : jeton de `transactions` obtenu plus de 5 min avant ;
   - autre audience : jeton de `xp03-autre` d'audience `interactions` ;
   - accepté : jeton de `transactions` en client credentials.
   Attendu : 5 × 401, 1 × 200.
6. **K1** : `kubectl top pod` et état des pods toutes les 30 s pendant 15 min au repos, puis au pic (tir k6 de G5 de XP01 sur le service de test, et 20 utilisateurs virtuels sur le point de jeton de Keycloak pendant 5 min, broker en vague). Attendu : aucun `OOMKilled`, aucun `Pending`.
7. **MA12**, **XE3** (`testssl.sh`, `curl -sI http://collector.qvl-project.com/` → 301, en-tête `Strict-Transport-Security`, hôte inconnu → 404), **MX2** depuis CloudShell, puis densité D09 si le temps le permet.
8. Extinction : `destroy` de XP01 (G3), inventaire incluant l'absence des alias `collector` et `auth.collector` dans la zone ; seuls les `NS`, `SOA` et le `CNAME` de validation restent.

## Durée et coût

- **Préparation** : 1 j (realm de test, règles d'ALB, service de test vérifiant l'audience, proxy local).
- **Exécution** : 0,5 j en local ; ≈ 2 h dans la session EKS 2.
- **Coût** : compris dans XP01 ; zone DNS 0,50 $ par mois (XP00).

## Données collectées

- **CSV** : `docs/mesures/XP03-identite-exposition.csv`, colonnes `horodatage_utc;xp;session;plateforme;critere;etape;valeur;unite;seuil;atteint;commentaire` (code HTTP par chemin et par cas de jeton, durées K2, mémoire K1).
- **Sorties** : rapport `testssl.sh`, résultat de `openssl s_client` (sujet et SAN), dans `docs/mesures/XP03-identite-exposition/`.
- **Captures** : écran de connexion sur `auth.collector.qvl-project.com` (cadenas, certificat ACM), demande d'OTP de l'admin, `kubectl top` au pic.

## Résultats

| # | Local | EKS | Atteint |
|---|---|---|---|
| XE1 | — | Reporté à la session EKS | Reporté |
| XE2 | — | Reporté à la session EKS | Reporté |
| XE4 + ME8 | Caddy : 200 sur découverte, `certs` et page de connexion ; 26/26 chemins interdits en 404 (traversées, encodages, casse, barre oblique inverse) ; `/reserver` 404 ; 9000 sans réponse | Reporté à la session EKS | Oui en local |
| K1 | — (indicatif : Keycloak 651 Mio compose, 671 Mio k3d, 20m CPU au repos) | Reporté à la session EKS | Reporté |
| K2 | compose 29 s, 30 s ; k3d 110 s (tirage de l'image PostgreSQL sur un nouvel agent compris), 41 s | Reporté à la session EKS | Oui en local |
| K3 | compose et k3d : 5 × 401, 1 × 200 ; MA3 et MA5 conformes (9/9 chaque fois) | Reporté à la session EKS | Oui en local |
| G7-4 | k3d : `ns-intrus` → 8080, 9000, 8081, 5432 fermés ; témoins `transactions-test` ouverts, page de connexion 200 (9/9) | Reporté à la session EKS | Oui (débogage) |
| XE3 | — (indicatif : 80 → 301 par Caddy) | Reporté à la session EKS | Reporté |
| MA12 | compose et k3d : OTP demandé à l'admin, pas au vendeur (5/5 chaque fois) | Reporté à la session EKS | Oui en local |
| MX2 | — | Reporté à la session EKS | Reporté |

Contrôles complémentaires du realm (`verifier.sh ma`, compose) : 18/18 (MA1, MA2, MA3, MA5, MA9, MA11, MA13). MA14 (Trivy) non rejoué. Relevés : [`docs/mesures/XP03-identite-exposition.csv`](../mesures/XP03-identite-exposition.csv) (format du CSV d'XP00, et non les colonnes annoncées dans « Données collectées » ; les scripts écrivent encore l'ancien format) et sorties dans [`docs/mesures/XP03-identite-exposition/`](../mesures/XP03-identite-exposition/) (seules les charges utiles décodées des jetons y figurent).

## Difficultés

À compléter par Martin pendant l'exécution, sans réécriture.

- **Import du realm refusé au premier lancement compose** : Keycloak s'est arrêté sur `invalidPasswordBlacklistedMessage` ; un des mots de passe aléatoires des comptes de test a été refusé par `passwordBlacklist` (filtre de Bloom, probabilité de faux positif 1e-4 par mot de passe). Relancé avec des secrets régénérés : pile prête en 31 s. Rare, mais une session EKS peut tomber dessus : `identifiants.sh` pourrait vérifier les mots de passe contre la liste, ou l'import être rejoué automatiquement avec de nouveaux secrets.
- **`scripts/compose.sh down` sans `.env` chargé** : l'interpolation de `docker-compose.yml` échoue (`MDP_POSTGRES` requis), l'arrêt ne se fait pas et `.env` n'est pas supprimé ; contourné en chargeant `.env` avant `down`. À corriger dans le script (charger `.env` dans `down`).
- **K2 k3d** : la première mesure (110 s) inclut le tirage de `postgres:18.3` (162 Mo, 3 min 20 s sur le premier agent) quand le pod change de nœud ; l'image n'est pas importée dans le cluster par `k3d.sh up`, ce qui a aussi fait expirer `up` (attente PostgreSQL de 180 s). Sur EKS, prévoir le tirage de l'image Keycloak optimisée dans la mesure.
- **G7-4 local** : le témoin « ALB → page de connexion » n'existe pas en local ; remplacé par `transactions-test` → page de connexion (200).

## Limites

- Liste blanche vérifiée depuis l'adresse de Martin seulement : l'ALB restant fermé au reste d'Internet (MX2), l'exposition publique complète (X17) est un AC bloquant du sprint 1.
- Service de test vérifiant le jeton, pas `libs/platform/` : la vérification `jwx` du POC est reprise en AC.
- XE5 et les en-têtes du front (ME2) relèvent du sprint 1.

## Conclusion

- **Relecture architecture** :
- **Relecture sécurité** (IAM, TLS, realm) :
- **Verdict de Martin** :
- **Mise à jour de D06 et D08** (dont décision du troisième nœud au repos) :
