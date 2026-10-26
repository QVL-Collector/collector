# D08 — Exposition, TLS, domaine

| Champ | Valeur |
|---|---|
| Statut | Candidate (XP en D14) |
| Date | 2026-10-26 |
| Décideur | Martin |
| Dépend de | [D03](D03-hebergement-orchestration.md), [D04](D04-stack.md), [D05](D05-communication-interservice.md), [D06](D06-authentification-autorisation.md), [D07](D07-donnees.md) |
| Impacte | D06 (émetteur `auth.<domaine>`), D09 (journaux du point d'entrée), D10 (`config.json`, contrôle ZAP), D11 (couches persistante et éphémère, `up` et `down`), D13 (limites de débit pendant le tir de charge), D14 (expérimentation), [exigences-securite.md](../securite/exigences-securite.md) (ES-01, ES-07, ES-29) |
| Critères grille | C8 (principal), C5, C2 |

## Contexte et besoin

D08 fixe comment le POC est joint depuis Internet : nom de domaine, certificat, point d'entrée, hébergement du front, en-têtes de sécurité et limitation de débit. Elle ne rouvre ni l'orchestrateur (D03), ni le serveur d'autorisation (D06), ni le stockage des photos (D07).

### Exigences

| Réf. | Exigence (résumé) | Source |
|---|---|---|
| ES-01 | Tout accès externe en HTTPS : redirection depuis HTTP, TLS 1.2 minimum, certificat valide renouvelé automatiquement, HSTS. Flux internes chiffrés ou confinés | [exigences-securite.md](../securite/exigences-securite.md) |
| ES-07 | Seuls les points d'entrée HTTPS publics sont joignables ; en-têtes HSTS, CSP, `nosniff`, interdiction du cadre ; origines limitées à la plateforme ; erreurs génériques | idem |
| ES-29 | Limites par origine et par compte sur connexion, inscription, recherche, upload, soumission, réservation, messages, paiement ; tailles et délais bornés. Statut POC : recommandée | idem |
| ES-38 | « Réserver » absent du point d'entrée public | idem, D05 (MC7) |

### Ce qui est déjà acquis

- **D03** : un ALB public créé par l'IaC, groupes cibles en mode instance sur des NodePorts, règles par chemin ; ni AWS Load Balancer Controller (il exige IRSA), ni Service `LoadBalancer`, ni ingress-nginx (retiré en mars 2026) ; un contrôleur d'entrée seulement « si D08 en a besoin ». Couche persistante (ECR, zone DNS, certificat, bucket, état) et couche éphémère (VPC, NAT, EKS, RDS, ALB). Mesure MS2 : TLS 1.2 minimum, HTTP redirigé, liste blanche, 404 par défaut. Critère X17 : seuls 443 et 80 répondent.
- **D04** : front React statique ; `config.json` servi par l'environnement (ENF-29) ; CSP en en-tête HTTP, `script-src 'self'`, `style-src 'self' 'unsafe-inline'`.
- **D05** : « réserver » sur un port interne, jamais routé ; console du broker par `port-forward` ; confinement réseau sans TLS interne dans le POC.
- **D06** : émetteur `https://auth.<domaine>/realms/collector`, stable si le domaine l'est ; liste blanche des chemins de Keycloak (MA7) ; console non routée ; repli sslip.io jugé « fonctionnel mais fragile », à trancher ici.
- **D07** : photos lues par URL présignée directement depuis S3 ; `img-src` limité à l'hôte exact du bucket (MD8) ; aucune route d'ALB pour les photos.

### Contraintes du Learner Lab

Source : [environnement-learner-lab.md](../00-cadrage/environnement-learner-lab.md).

- **CloudFront indisponible** : pas de HTTPS sur un site web S3 (son point de terminaison ne sert que HTTP).
- **Route 53** : achat de domaine impossible [DOC] ; création de zones hébergées réussie dans plusieurs états Terraform, contredite par un REX : **[?]**, à la sonde.
- **ACM** : certificat demandé et validé par DNS jusqu'à `ISSUED` en REX ; import de certificat refusé dans un compte, accepté dans un autre **[?]**.
- **WAF** : `CreateWebACL` refusé en REX **[?]**.
- **IAM figé** : pas d'IRSA, donc pas de contrôleur qui crée des load balancers.
- **Recréation à chaque session** : l'ALB, son nom DNS et ses adresses changent à chaque `up`.

### Les cinq sous-questions

- **Q1** domaine ; **Q2** certificat ; **Q3** point d'entrée ; **Q4** hébergement du front ; **Q5** en-têtes, HSTS, limitation de débit, mise à jour DNS.
- Elles ne sont pas indépendantes. Un ALB n'a pas d'adresse fixe : un nom dérivé d'une adresse (sslip.io, nip.io) ou un DNS dynamique sans CNAME (DuckDNS) ne peut pas le viser durablement. Sans domaine, il faut donc un NLB à Elastic IP, un certificat hors ACM et une terminaison TLS dans le cluster. Les options se lisent donc par chaînes cohérentes (voir « Synthèse »).
- **Conséquence pour D06** : le repli « sslip.io » de D06 suppose un ALB ; il n'est viable que sur un NLB à adresse fixe.

## Options

### Q1 — Domaine

#### Option D1 — Domaine personnel acheté, zone dans Route 53

- **Description** : Martin achète un domaine chez un registrar (environ 10 $ par an selon l'extension), dont il délègue les serveurs de noms une fois pour toutes à une zone Route 53 de la couche persistante.
- **Avantages** : nom stable d'une session à l'autre et pour la soutenance ; émetteur de D06 stable par construction ; validation ACM et enregistrement alias vers l'ALB gérés par Terraform, sans identifiant hors AWS.
- **Inconvénients** : dépense personnelle hors crédits ; zone à ne jamais détruire (sa recréation change les serveurs de noms et oblige à refaire la délégation, avec un délai de propagation) ; perdue si le compte du lab est effacé (dépassement du budget).
- **Coût AWS et effort** : zone 0,50 $ par mois, environ 3 $ sur le projet ; requêtes vers un alias d'ALB gratuites ; 0,25 j.
- **Complexité ajoutée** : une zone, une délégation unique.
- **Couverture grille** : C8 (HTTPS conforme, certificat géré), C5 (ressource managée, IaC).

#### Option D2 — Domaine personnel acheté, DNS chez le registrar

- **Description** : même domaine, zone conservée chez le registrar ; l'IaC repointe un CNAME vers l'ALB par l'API du registrar.
- **Avantages** : survit à un effacement du compte ; pas de zone Route 53.
- **Inconvénients** : un jeton d'API du registrar à stocker et à utiliser à chaque `up` (secret hors AWS, ES-06) ; pas d'alias à l'apex (CNAME interdit à l'apex) ; validation ACM par CNAME posé chez le registrar ; dépend du provider Terraform du registrar.
- **Coût AWS et effort** : 0 $ AWS ; 0,5 j.
- **Complexité ajoutée** : un second fournisseur dans l'IaC, un secret de plus.
- **Couverture grille** : C8, C5 ; moins « AWS ».

#### Option D3 — Sous-domaine gratuit délégable

- **Description** : sous-domaine gratuit dont on obtient la délégation des serveurs de noms (par exemple eu.org, ou un sous-domaine délégué par l'école), puis comme D1.
- **Avantages** : gratuit ; ensuite identique à D1.
- **Inconvénients** : délai d'obtention incertain (validation manuelle, parfois plusieurs semaines) **[?]** ; dépend d'un tiers.
- **Coût AWS et effort** : comme D1, plus l'attente.
- **Complexité ajoutée** : comme D1.
- **Couverture grille** : comme D1, si obtenu à temps.

#### Option D4 — Sans domaine : sslip.io ou nip.io

- **Description** : nom dérivé d'une adresse IP (`<ip>.sslip.io`). Exige une adresse fixe : NLB avec une Elastic IP par zone, dans la couche éphémère (nom différent à chaque session) ou persistante (nom stable, mais adresses facturées en permanence).
- **Avantages** : aucun achat ; aucune zone.
- **Inconvénients** : ACM inutilisable (pas de contrôle du DNS) ; impose Q2-C2 et Q3-E2 ; émetteur et URL de soutenance qui changent à chaque session si les adresses sont éphémères ; service tiers sans engagement ; limites de Let's Encrypt par domaine enregistré, selon l'inscription de sslip.io sur la liste des suffixes publics **[?]**.
- **Coût AWS et effort** : 2 Elastic IP éphémères, environ 0,04 $ par session ; persistantes, environ 7 $ par mois. Effort porté par Q2 et Q3.
- **Complexité ajoutée** : forte, par ricochet (NLB, contrôleur d'entrée, émission de certificat dans le cluster).
- **Couverture grille** : C8 tenu, C5 affaibli (TLS opéré dans le cluster).

### Q2 — Certificat

| Option | Description | Points forts | Points faibles | Coût |
|---|---|---|---|---|
| C1 — ACM, validation DNS | Certificat `<domaine>` et `*.<domaine>` dans la couche persistante ; enregistrement de validation laissé dans la zone | Gratuit ; renouvellement automatique tant que l'enregistrement de validation reste ; jamais réémis à chaque session ; aucune clé privée dans le cluster | Exige D1, D2 ou D3 ; disponibilité dans le lab à confirmer [?] | 0 $ |
| C2 — Let's Encrypt dans le cluster (cert-manager ou ACME de Traefik) | Certificat émis par le contrôleur d'entrée, clé dans un Secret Kubernetes | Fonctionne sans domaine (D4) ; même mécanisme en local | Clé privée dans le cluster ; certificat perdu à chaque `down`, donc réémis à chaque session : limite de 5 certificats identiques par semaine, sauf à le sauvegarder dans Secrets Manager (0,40 $ par mois, un script de plus) | 0 $ |
| C3 — Point de terminaison API Gateway | HTTPS natif sur `*.execute-api.<région>.amazonaws.com` | Aucun certificat à gérer | Impose Q3-E3 ; une seule origine par API, donc pas de sous-domaine `auth.` (contraire à D06) sans seconde API | 0 $ hors requêtes |

### Q3 — Point d'entrée

#### Option E1 — ALB créé par l'IaC, règles par hôte et par chemin (socle D03)

- **Description** : écouteur 443 avec le certificat C1 et la politique `ELBSecurityPolicy-TLS13-1-2-2021-06` ; écouteur 80 en redirection 301 ; `drop_invalid_header_fields` activé ; règles en liste blanche vers les NodePorts ; réponse 404 par défaut, y compris pour l'accès par le nom DNS de l'ALB ou un hôte inconnu.
- **Avantages** : déjà prévu par D03 et MS2 ; managé ; aucun pod de plus sur deux t3.medium déjà chargés par Keycloak ; HSTS posable par l'écouteur (modification des en-têtes de réponse, fonction de l'ALB depuis fin 2024 [DOC], disponibilité dans le lab [?]) ; inventaire simple (X17).
- **Inconvénients** : pas de limitation de débit (WAF refusé en REX [?]) ; pas de réécriture de chemin utilisée (chaque service sert son propre préfixe) ; routage du local différent (minikube sans ALB).
- **Coût AWS et effort** : déjà compté par D03 (environ 0,02 $/h plus les LCU) ; 0,5 j pour les règles.
- **Complexité ajoutée** : nulle au-delà de D03.
- **Couverture grille** : C8, C5, C2.

#### Option E2 — NLB créé par l'IaC, contrôleur d'entrée dans le cluster (Traefik)

- **Description** : NLB TCP vers Traefik en NodePort (Traefik plutôt qu'Envoy Gateway : un seul binaire, sans plan de contrôle séparé, empreinte plus faible). TLS au NLB (ACM) ou dans Traefik (C2). Routes en Gateway API, intergiciels d'en-têtes et de limitation de débit.
- **Avantages** : un seul endroit pour les en-têtes et le débit ; seul chemin compatible avec D4 ; mêmes routes en local et sur AWS (CR6).
- **Inconvénients** : un composant de plus à maintenir et à corriger (M-LIV-01) ; mémoire sur des nœuds déjà serrés (risque de troisième nœud) ; limitation de débit par instance de Traefik, comme dans les services (une limite globale demande un stockage partagé) ; adresse du client à préserver (protocole PROXY) ; TLS dans le cluster si C2.
- **Coût AWS et effort** : NLB à la place de l'ALB, coût voisin ; troisième nœud éventuel, environ 0,17 $ par session ; 2 à 3 j (chart, TLS, protocole PROXY, tests).
- **Complexité ajoutée** : moyenne à forte.
- **Couverture grille** : C8 ; C5 partiel (contrôleur auto-opéré).

#### Option E3 — API Gateway HTTP, VPC Link vers un ALB interne

- **Description** : API HTTP publique, intégration privée par VPC Link vers un ALB interne qui garde les règles de E1.
- **Avantages** : HTTPS natif sans domaine (C3) ; limitation de débit par route intégrée ; l'API peut rester dans la couche persistante (facturée à la requête).
- **Inconvénients** : un ALB reste nécessaire ; VPC Link et intégration à recréer à chaque `up` ; limitation par route, pas par client ; plafond de 30 s et de 10 Mo par requête ; une origine par API, donc deux API pour garder `auth.` ; schéma plus long à défendre.
- **Coût AWS et effort** : environ 1 $ par million de requêtes, soit de l'ordre de 1 $ par tir de charge ; 1,5 à 2 j.
- **Complexité ajoutée** : une couche de plus devant l'ALB.
- **Couverture grille** : C8, C5.

Exclus par D03, non rediscutés : AWS Load Balancer Controller, Service `LoadBalancer`, ingress-nginx. Exclu ici : ALB devant Traefik (deux couches pour un même rôle).

### Q4 — Hébergement du front statique

| Option | Description | Points forts | Points faibles |
|---|---|---|---|
| F1 — Conteneur `nginx-unprivileged` dans EKS | Image officielle non root, système de fichiers en lecture seule, port 8080 ; repli SPA vers `index.html` ; `config.json` monté depuis un ConfigMap, `Cache-Control: no-store` ; `index.html` non mis en cache, fichiers à empreinte en cache long ; en-têtes de Q5 | Aucun code ; configuration de quelques lignes ; même image en local et sur AWS (ENF-29) | Une image tierce de plus à analyser (ES-05) |
| F2 — Petit serveur Go (`embed`) | Binaire statique sans shell, comme les services (D04) | Même chaîne et même socle que les services, image minimale | Code à écrire et à tester (repli SPA, cache, en-têtes), environ 0,5 j |
| F3 — Site web S3 | Bucket en mode site web | Pas de pod | HTTP seulement sans CloudFront : contraire à ES-01. **Écartée** |

### Q5 — En-têtes, HSTS, limitation de débit, DNS

Mesures communes à toutes les options, avec le lieu proposé pour chacune.

- **Origines et routage** (E1) :

| Hôte | Chemin | Cible |
|---|---|---|
| `<domaine>` | préfixe de la Place de marché (port public) | Place de marché |
| `<domaine>` | préfixe d'Interactions | Interactions |
| `<domaine>` | préfixe de Transactions | Transactions |
| `<domaine>` | tout le reste | Front |
| `auth.<domaine>` | liste blanche de D06 (MA7) | Keycloak |
| tout autre hôte | — | 404 |

  - Préfixes exacts fixés dans les contrats OpenAPI (`contracts/api/`). Front et API sur la même origine : `connect-src 'self' https://auth.<domaine>`, aucune règle CORS dans les services ; seule l'origine du front est déclarée dans les « Web Origins » du client `collector-web` de Keycloak.
  - Jamais routés : « réserver », console et port de gestion de Keycloak, broker, photos (D07).
- **En-têtes** : un seul lieu par en-tête, pour éviter les doublons contradictoires.
  - HSTS `max-age=31536000; includeSubDomains`, sans `preload` (domaine personnel réutilisable) : par l'écouteur de l'ALB, ce qui couvre aussi les réponses 404 ; à défaut [?], par chaque origine.
  - Front (nginx) : CSP de D04 avec `img-src 'self' blob: https://<bucket>.s3.<région>.amazonaws.com` (hôte du bucket stable, couche persistante), `X-Content-Type-Options: nosniff`, `Referrer-Policy: no-referrer`, `X-Frame-Options: DENY` en complément de `frame-ancestors 'none'`, `Permissions-Policy` restrictive.
  - API (`libs/platform`) : `Content-Security-Policy: default-src 'none'; frame-ancestors 'none'`, `nosniff`, `Cache-Control: no-store`, erreurs en Problem Details sans trace.
  - Keycloak : en-têtes de sécurité du realm, conservés ; `proxy-headers xforwarded` derrière l'ALB.
- **Limitation de débit (ES-29)** :
  - connexion : détection de force brute du realm Keycloak (ES-09), déjà prévue par D06 ;
  - API : intergiciel de `libs/platform` à seau de jetons (`golang.org/x/time/rate`), par adresse et par compte, sur les routes sensibles (upload, soumission, réservation, messages, recherche) ; réponse 429 ;
  - adresse du client lue dans la **dernière** valeur de `X-Forwarded-For`, celle ajoutée par l'ALB, jamais la première (falsifiable) ;
  - limite par réplica : la limite effective est multipliée par le nombre de réplicas (6 au plus, MS10). Écart accepté pour le POC, limite partagée en V1 ;
  - tailles de requête et délais déjà bornés dans les serveurs Go (D04).
- **DNS à chaque `up`** :
  - zone et certificat dans la couche persistante ; enregistrements alias `<domaine>` et `auth.<domaine>` dans la couche éphémère, créés avec l'ALB et détruits avec lui : aucun nom ne pointe vers un ALB disparu ;
  - SOA de la zone : durée de cache négatif ramenée de 86 400 s (valeur par défaut de Route 53) à 60 s, sinon un nom interrogé entre deux sessions reste « inexistant » jusqu'à un jour chez les résolveurs ;
  - test HTTPS 200 sur les deux noms à la fin de `up` (X4).

### Synthèse

| Critère | Chaîne 1 — D1 + C1 + E1 + F1 | Chaîne 2 — D4 + C2 + E2 + F1 | Chaîne 3 — C3 + E3 + F1 |
|---|---|---|---|
| Nom stable (émetteur, soutenance) | Oui | Non, sauf adresses persistantes facturées | Oui si l'API est persistante, mais un hôte par API |
| Certificat | ACM, managé, jamais réémis | Let's Encrypt, réémis à chaque session | Natif |
| Sous-domaine `auth.` (D06) | Oui | Oui | Seconde API |
| Limitation de débit | Keycloak + `libs/platform` | Traefik (par instance) | Par route, pas par client |
| Composants ajoutés à D03 | Zone, certificat, image nginx | NLB, Elastic IP, Traefik, émission ACME | API, VPC Link, ALB interne |
| Coût par session de 4 h | ≈ 0 $ en plus de D03 | ≈ 0,04 à 0,2 $ | ≈ 0 $ au repos, ≈ 1 $ par tir |
| Coût fixe | 0,50 $/mois + domaine ≈ 10 $/an hors crédits | 0 $ (ou ≈ 7 $/mois pour un nom stable) | 0 $ |
| Effort | 1,5 à 2 j | 3 à 4 j | 2,5 à 3 j |
| Risque Learner Lab | Route 53 et ACM [?] | Faible côté AWS ; tiers sslip.io | VPC Link à recréer |

## Recommandation

**Chaîne 1 : domaine personnel acheté avec zone Route 53 (D1), certificat ACM validé par DNS (C1), ALB créé par l'IaC (E1), front servi par `nginx-unprivileged` (F1), mesures de Q5.**

1. **C'est le socle de D03, sans rien ajouter au cluster.** Aucun pod de plus sur des nœuds déjà occupés par Keycloak ; aucun TLS ni clé privée à opérer dans le cluster.
2. **Elle tient l'émetteur stable de D06** sans injection par session, et donne à la soutenance des URL fixes.
3. **Un contrôleur d'entrée n'apporte pas ce qui manque.** Sa limitation de débit est, elle aussi, par instance ; les en-têtes se posent aussi bien à l'origine. Le besoin nommé par D03 (« si D08 en a besoin ») n'est donc pas constitué.
4. **Son coût réel est hors AWS** : environ 10 $ par an pour le domaine, payés par Martin. Les crédits ne supportent que la zone (environ 3 $ sur le projet).

**Repli, si la sonde refuse Route 53** : D2 (même domaine, DNS chez le registrar), le reste inchangé. **Si ACM est refusé** : chaîne 2, qui est aussi la seule voie sans domaine. La chaîne 3 n'est pas recommandée : elle ajoute une couche sans retirer l'ALB et casse le sous-domaine `auth.`.

### Points soumis aux avis

- **Sécurité** :
  - HSTS par l'écouteur de l'ALB, avec `includeSubDomains` et sans `preload` ;
  - répartition des en-têtes (ALB, nginx, `libs/platform`, Keycloak) et CSP finale avec l'hôte exact du bucket ;
  - suffisance, pour le POC, d'une limitation par réplica dans les services et de la détection de force brute de Keycloak au regard d'ES-29 et d'ES-09 ; lecture de l'adresse du client dans `X-Forwarded-For` ;
  - groupe de sécurité de l'ALB ouvert à Internet à chaque session, ou restreint aux adresses de Martin hors démonstration ;
  - trafic de l'ALB vers les NodePorts en clair dans le VPC (confinement d'ES-01, cohérent avec D05) ;
  - enregistrements DNS éphémères (pas de nom pendant vers un ALB disparu).
- **Faisabilité** :
  - sonde du compte : zone Route 53, certificat ACM `ISSUED`, attributs d'en-têtes de l'écouteur dans le provider Terraform et dans le lab ;
  - SOA à 60 s de cache négatif ;
  - F1 (`nginx-unprivileged`) ou F2 (serveur Go) ;
  - limites de débit compatibles avec le tir de charge de D13, lancé depuis peu d'adresses, sans route de contournement ;
  - effort de 1,5 à 2 j.
- **Martin** : achat d'un domaine personnel (et lequel), ou domaine déjà possédé.

## Avis sécurité

Fonction : Sécurité — 2026-10-26. Références : [exigences-securite.md](../securite/exigences-securite.md) (ES-01, ES-07, ES-09, ES-29, ES-38 ; M-ACC-01, M-DIS-02, M-LIV-03, M-LIV-04) ; MS1, MS2, MS4, MS5 et MS10 de [D03](D03-hebergement-orchestration.md), MC7 de [D05](D05-communication-interservice.md), MA4, MA7 et MA13 de [D06](D06-authentification-autorisation.md), MD8 de [D07](D07-donnees.md). Avis sur dossier ; les points [?] relèvent des sondes de D14.

**Synthèse.** Avis favorable à la chaîne 1 (D1 + C1 + E1 + F1), sous les mesures ME1 à ME10. Elle ne met aucune clé privée dans le cluster et n'ajoute aucun composant exposé. Le risque principal n'est pas le TLS : c'est le **contournement de la liste blanche de l'ALB** (chemin non normalisé vers la console de Keycloak) et la **confiance accordée à `X-Forwarded-For`**, deux points à prouver par des tests négatifs plutôt qu'à supposer.

**1. Points soumis**

- **HSTS `includeSubDomains`, sans `preload`** : favorable. `preload` engagerait tout le domaine personnel de Martin dans les navigateurs pour des mois, au-delà du projet. `includeSubDomains` n'agit que s'il est servi par l'apex `<domaine>`, ce qui est le cas (front) ; conséquence acceptée : tout autre sous-domaine du domaine devra être en HTTPS. L'écouteur de l'ALB est le bon lieu ; s'il ne couvre pas la réponse fixe 404 [?], ce n'est pas bloquant, l'apex l'ayant déjà donné. Keycloak émet par défaut le même en-tête dans ses en-têtes de realm : valeurs identiques, sans conflit ; le retirer du realm garde un seul lieu.
- **Répartition des en-têtes** : favorable (ALB : HSTS ; nginx : front ; `libs/platform` : API ; Keycloak : ses pages). Quatre compléments :
  - nginx : `add_header ... always` (sinon absents des réponses 4xx et 5xx), repris dans chaque bloc `location` qui en déclare un (nginx n'hérite plus des `add_header` du niveau supérieur dès qu'un bloc en définit) ; `server_tokens off` ;
  - CSP finale du front : `default-src 'self'; script-src 'self'; style-src 'self' 'unsafe-inline'; img-src 'self' blob: https://<bucket>.s3.<région>.amazonaws.com; connect-src 'self' https://auth.<domaine>; frame-ancestors 'none'; base-uri 'none'; form-action 'self'; object-src 'none'`. Jamais `*.amazonaws.com` (MD8) ;
  - redirection silencieuse de D06 : si elle passe par une iframe (`silent-check-sso.html`), ce seul fichier doit autoriser `frame-ancestors 'self'` et le front `frame-src https://auth.<domaine>` ; l'iframe de statut de session (`checkLoginIframe`) est à désactiver, la CSP de Keycloak interdisant son intégration depuis un autre hôte [?, à vérifier à l'implémentation] ;
  - Keycloak : `KC_HOSTNAME=https://auth.<domaine>`, `proxy-headers xforwarded`, cookies `Secure` contrôlés par ZAP.
- **Limitation par réplica et `X-Forwarded-For`** : suffisante pour le POC, ES-29 étant « Recommandée ». Limite effective multipliée par 6 au plus (MS10), écart à écrire. Lire la **dernière** valeur est juste si l'ALB est en mode `append` : le fixer explicitement (`routing.http.xff_header_processing.mode`) plutôt que dépendre du défaut. En mode instance, `RemoteAddr` est l'adresse d'un nœud : l'en-tête est la seule source, et sa confiance repose sur MS2 et MS4 (NodePorts joignables depuis l'ALB seul). Trois réserves :
  - Keycloak (Quarkus) retient vraisemblablement la **première** valeur [?] : adresse falsifiable dans ses événements (D09) ; sans effet sur MA13, dont la force brute est par compte ;
  - écart ES-09 : aucune limite **par origine** sur la connexion (force brute par compte seulement, WAF refusé [?]). Le bourrage d'identifiants sur de nombreux comptes depuis une même adresse (M-ACC-01) n'est pas freiné. Accepté pour le POC sur comptes fictifs ; règle WAF à base de débit en V1 ;
  - la table des seaux par adresse doit être bornée (taille maximale, éviction), sinon la limitation devient elle-même un déni de service mémoire (M-DIS-02).
- **ALB ouvert ou restreint** : restreindre hors démonstration. Par défaut, groupe de sécurité de l'ALB limité à l'adresse de Martin (/32, comme MS1) ; ouverture à Internet par une variable explicite, le temps de la soutenance, de ZAP ou du tir de charge (adresses de D13). Keycloak et ses comptes de test sortent ainsi des balayages automatiques entre deux démonstrations. Point d'attention : si les services récupèrent les clés de Keycloak (JWKS) par `https://auth.<domaine>`, leur trafic sort par la NAT et serait refusé ; les récupérer par l'adresse interne du Service Keycloak, en vérifiant toujours `iss` égal à l'URL publique (MA4).
- **ALB vers NodePorts en clair** : accepté pour le POC. ES-01 admet « confiné » : sous-réseaux privés, NodePorts ouverts depuis le seul groupe de l'ALB (MS2), politiques réseau (MS4), pas de `hostNetwork` (PSS « restricted », MS5) ; le VPC ne permet pas d'écouter le trafic d'une autre interface. Risque résiduel faible, à documenter ; V1 : TLS jusqu'aux cibles.
- **Politique TLS** : `ELBSecurityPolicy-TLS13-1-2-2021-06` tient ES-01 mais garde en TLS 1.2 des suites CBC sans AEAD. Préférer `ELBSecurityPolicy-TLS13-1-2-Res-2021-06` (suites GCM seules), sans perte pour un navigateur récent. Faible, recommandé. Certificat : noms explicites `<domaine>` et `auth.<domaine>` plutôt que le joker ; enregistrement CAA limité à `amazon.com`.
- **« Réserver » jamais routé** : la défense est structurelle (MC7 : serveur et port distincts, Service `ClusterIP`, NodePorts hors liste refusés par MS5) ; l'ALB n'est que la seconde ligne. Le risque réel est côté Keycloak : une règle `/realms/collector/*` laisse passer `/realms/collector/../../admin/` ou ses formes encodées (`%2e%2e`, `%2f`) si l'ALB transmet le chemin brut et que Keycloak le normalise [?]. XE4 doit l'essayer avec `curl --path-as-is`, en plus des chemins directs.

**2. Point non soumis : délégation pendante.** Si le compte du lab est effacé, la zone disparaît mais le registrar délègue toujours vers ses serveurs de noms ; une zone créée dans un autre compte sur les mêmes serveurs permettrait de reprendre le domaine, certificat compris. Procédure : retirer ou repointer la délégation chez le registrar dès la perte du compte. Les alias éphémères, eux, ne laissent aucun nom pendant : favorable.

**3. Mesures minimales, si la chaîne 1 est retenue**

| Id | Mesure | Exigences, menaces | Preuve |
|---|---|---|---|
| ME1 | HSTS `max-age=31536000; includeSubDomains`, sans `preload`, par l'écouteur ; un seul lieu par en-tête | ES-01, ES-07, M-LIV-04 | XE3 ; ZAP |
| ME2 | nginx : en-têtes `always`, repris dans chaque `location`, `server_tokens off` ; CSP finale du point 1 | ES-07, M-ANN-03, M-PAY-04, MD8 | ZAP sur une page 200 et une 404 |
| ME3 | Exception de cadre limitée à `silent-check-sso.html` si la redirection silencieuse l'exige ; `checkLoginIframe` désactivé | ES-07 | Test de connexion silencieuse ; ZAP |
| ME4 | ALB en `append` explicite ; dernière valeur de `X-Forwarded-For` ; table des seaux bornée ; limites lues dans la configuration, sans route ni en-tête de contournement pour D13 | ES-29, M-DIS-02 | XE5 avec un `X-Forwarded-For` forgé : 429 et adresse réelle journalisée |
| ME5 | Groupe de sécurité de l'ALB restreint à l'adresse de Martin hors démonstration ; ouverture par variable explicite | ES-07, M-LIV-03 | Inventaire (X17) après `up` |
| ME6 | JWKS récupérées par l'adresse interne de Keycloak, `iss` public vérifié | ES-07, MA4 | Services opérationnels avec l'ALB restreint |
| ME7 | Politique `ELBSecurityPolicy-TLS13-1-2-Res-2021-06` ; certificat à noms explicites ; CAA `amazon.com` | ES-01 | XE3 |
| ME8 | XE4 étendu : traversées et encodages (`..`, `%2e%2e`, `%2f`, casse) vers `/admin/` et `/realms/master/`, avec `--path-as-is`, à chaque déploiement | ES-07, ES-38, MA7 | XE4, bloquant |
| ME9 | Écarts écrits : limite par réplica, pas de limite par origine sur la connexion, adresse falsifiable dans les événements de Keycloak [?], flux ALB vers nœuds en clair | ES-01, ES-09, ES-29 | Registre des écarts à jour |
| ME10 | Procédure « compte perdu » : délégation retirée chez le registrar | ES-07 | Procédure rédigée (D11) |

**Menaces à ajouter au registre** (§4.10, à reporter par Martin) :

| ID | STRIDE | FC | Scénario | V | I | Risque |
|---|---|---|---|---|---|---|
| M-EXP-01 | E, I | FC1, FC3 | Chemin non normalisé ou encodé qui franchit la liste blanche de l'ALB et atteint la console ou le realm `master` de Keycloak | 2 | 4 | 8 Élevé |
| M-EXP-02 | S, R | FC1 | `X-Forwarded-For` forgé : limitation de débit contournée ou adresse fausse dans les journaux | 3 | 2 | 6 Modéré |
| M-EXP-03 | D | FC1 | Table de limitation non bornée saturée par des adresses variées | 2 | 2 | 4 Modéré |
| M-EXP-04 | S, T | FC1 | Délégation DNS pendante après effacement du compte : reprise du domaine par une zone tierce | 1 | 3 | 3 Faible |

**Verdict** : GO pour la chaîne 1, sous réserve de ME1 à ME10 ; ME4 et ME8 bloquants avant toute ouverture de l'ALB à Internet.

## Avis faisabilité

**Verdict : chaîne 1 faisable**, sous réserve de la sonde du premier jour. L'effort de 1,5 à 2 j est un peu optimiste : compter **2,5 à 3 j** (détail ci-dessous).

### Exploitation

- **Sonde Route 53 et ACM [?], avant tout achat de domaine** :
  - étape 1, sans domaine : `create-hosted-zone` sur un nom quelconque, puis `request-certificate` ; un certificat `PENDING_VALIDATION` suffit à prouver que l'API est autorisée. Supprimer les deux ensuite ;
  - étape 2, après achat et délégation : XE1 (`dig NS` depuis un résolveur public), puis XE2 (`ISSUED` en quelques minutes) ;
  - vérifier aussi que la version du provider Terraform AWS épinglée expose les attributs de modification des en-têtes de réponse de l'écouteur (HSTS) [?], et que le lab les accepte. À défaut, HSTS par chaque origine, comme prévu.
- **Couche persistante** : zone, certificat (apex et `*.<domaine>` dans un seul certificat), enregistrements de validation et SOA, dans un état Terraform séparé, avec `prevent_destroy`. La couche éphémère ne lit que ses sorties (identifiant de zone, ARN du certificat). Le SOA et les NS existent déjà dans la zone : les gérer avec `allow_overwrite`.
- **Recréation à chaque `up`** : ALB, écouteurs, règles, groupes cibles et alias `A` (`<domaine>`, `auth.<domaine>`) dans la couche éphémère. L'alias prend effet dès sa création chez Route 53 ; seul le cache des résolveurs compte. Le test HTTPS 200 de fin de `up` (X4) doit réessayer pendant 1 à 2 min avant de conclure à l'échec.
- **Cache négatif SOA** : faisable en une ressource. Correction de chiffre : la durée effective du cache négatif est le minimum entre le TTL de l'enregistrement SOA (900 s par défaut) et son champ « minimum » (86 400 s), soit **15 min** et non un jour. La recommandation tient : ramener les deux valeurs à 60 s.
- **Règles d'écoute en Terraform vers les NodePorts** :
  - NodePorts fixés dans les valeurs du chart (plage 30000-32767), une variable partagée entre chart et Terraform ;
  - groupes cibles en mode instance, contrôle de santé sur le chemin de disponibilité de chaque service, par son NodePort ;
  - enregistrement des nœuds par `aws_autoscaling_attachment` sur l'ASG du node group managé : les nœuds remplacés sont enregistrés seuls, sans contrôleur. `TargetGroupBinding` est une ressource du AWS Load Balancer Controller : **impossible ici**, et inutile avec cet attachement ;
  - limite de l'ALB : 5 valeurs de condition par règle. La liste blanche de Keycloak (MA7) demandera plusieurs règles, avec des priorités à figer ;
  - action par défaut : réponse fixe 404 ; écouteur 80 en redirection 301.
- **Front, F1 ou F2** : avis pour **F1** (0,25 j). Point d'attention : avec un système de fichiers en lecture seule, `nginx-unprivileged` a besoin d'un `emptyDir` monté sur `/tmp` (fichier pid et tampons). F2 n'apporte qu'une image de moins à analyser, pour 0,5 j de code et de tests en plus.
- **Local** : l'ALB ne se reproduit pas. Équivalent proposé : un proxy inverse dans docker compose (ou sur k3d) avec une autorité locale (`tls internal` de Caddy, ou mkcert), sur `collector.localhost` et `auth.collector.localhost`, qui résolvent vers 127.0.0.1. Pas de HSTS en local. Risque : deux descriptions du routage (règles de l'ALB, configuration du proxy) qui divergent. Parade : un même test de liste blanche (XE4) joué en local et sur AWS. Effort 0,25 à 0,5 j.

### Développement

- **En-têtes dans `libs/platform`** : un intergiciel de quelques lignes, testé par un test de table sur la réponse ; 0,25 j.
- **Limitation de débit** :
  - une table de `rate.Limiter` par clé (adresse, compte), avec purge des entrées inactives pour borner la mémoire ; réponse 429 avec `Retry-After` ; 0,5 à 0,75 j avec les tests ;
  - seuils en variables d'environnement du chart, pour les régler avec le tir de charge de D13 sans recompiler. Un tir lancé depuis peu d'adresses déclenchera la limite par adresse : prévoir des comptes de test multiples et des seuils propres à l'environnement de tir, jamais un en-tête ou une route de contournement.
- **`X-Forwarded-For`** : dernière valeur, ajoutée par l'ALB. L'adresse source vue par le pod est celle d'un nœud (trafic NodePort traduit), donc inexploitable. Les appels internes et le local sans proxy n'ont pas cet en-tête : repli sur l'adresse source. Côté Keycloak, restreindre les adresses de proxy de confiance au bloc du VPC [?].

### Effort et risques

| Lot | Effort |
|---|---|
| Sonde, couche persistante, SOA | 0,5 j |
| ALB, règles, groupes cibles, attachement ASG, alias | 0,75 j |
| Front F1 et proxy local | 0,5 j |
| En-têtes et limitation de débit | 0,75 à 1 j |
| **Total** | **2,5 à 3 j** |

- **Effacement du compte** : la zone recréée change de serveurs de noms ; la re-délégation chez le registrar peut prendre plusieurs heures. Garder D2 prêt (compte du registrar actif) comme repli de soutenance.
- **Expiration de session pendant `apply`** : ALB et alias se créent en quelques minutes, risque faible si l'ordre de D03 est tenu.
- **Repli sslip.io de D06 : caduc tel qu'écrit.** Il supposait un ALB, qui n'a pas d'adresse fixe. Il ne survit que dans la chaîne 2 (NLB à Elastic IP). Le premier repli devient D2, qui garde l'émetteur stable ; D06 est à mettre à jour en conséquence.

## Décision de Martin

> **« J'ai déjà un domaine »** : chaîne 1 avec le domaine que Martin possède déjà. Un sous-domaine dédié au projet est délégué à une zone Route 53 ; repli D2 (DNS chez le registrar) si la sonde refuse Route 53. Certificat ACM générique et zone dans la couche persistante, alias vers l'ALB dans la couche éphémère, ALB créé par Terraform sans contrôleur d'entrée, front nginx-unprivileged (F1), front et API sur la même origine. Mesures de sécurité ME1 à ME10, dont ME4 et ME8 avant toute ouverture de l'ALB à Internet ; ALB restreint à l'adresse de Martin hors démonstration. Statut **Candidate** jusqu'à la sonde Route 53 / ACM et aux critères XE1 à XE5 (D14).
>
> **Domaine** (précision de Martin, 2026-10-26) : `qvl-project.com`, enregistré et géré chez Cloudflare. Le projet utilise le sous-domaine dédié `collector.qvl-project.com`, pour rester séparé des projets personnels de Martin : délégation à une zone Route 53 par des enregistrements NS posés dans Cloudflare (DNS seul, sans proxy). Noms : `collector.qvl-project.com` (front et API), `auth.collector.qvl-project.com` (Keycloak) ; certificat ACM pour `collector.qvl-project.com` et `*.collector.qvl-project.com`, validé dans la zone Route 53. Repli D2 : enregistrements posés directement dans Cloudflare.
>
> Conséquence immédiate : le repli « sans domaine stable » (sslip.io) de D06 est caduc avec un ALB ; il ne survit qu'en chaîne 2, si ACM est refusé.

Date : 2026-10-26

## Conséquences

La décision de Martin est enregistrée ci-dessus ; ses conséquences :

- **D06** : `KC_HOSTNAME` à `https://auth.<domaine>` dès le sprint 1 ; « Web Origins » limité à `https://<domaine>` ; le repli sslip.io de D06 est remplacé par le repli de D08 (chaîne 2, sur NLB).
- **D07** : `img-src` sur l'hôte exact du bucket, de style hébergé virtuel (MD8).
- **D09** : journaux d'accès de l'ALB vers S3 (bucket persistant) à décider par D09.
- **D10** : image `nginx-unprivileged` analysée comme les autres ; `config.json` produit au déploiement ; ZAP passif sur `https://<domaine>` et `https://auth.<domaine>` (en-têtes, CSP, cookies).
- **D11** : zone, certificat et enregistrement de validation dans la couche persistante, jamais détruits ; alias dans la couche éphémère ; `down` vérifie qu'aucun alias ne reste ; procédure documentée de re-délégation si le compte est effacé.
- **D13** : limites de débit et tir de charge réglés ensemble ; un 429 attendu est un résultat de test, pas un échec.
- **D14** : critères d'expérimentation à ajouter :

| ID | Mesure | Critère de réussite | Bascule si échec |
|---|---|---|---|
| XE1 | Zone Route 53 créée, délégation vérifiée (`dig NS`) | Résolution depuis un résolveur public | D2 |
| XE2 | Certificat ACM `<domaine>` et `*.<domaine>` | `ISSUED` et attaché à l'écouteur | Chaîne 2 |
| XE3 | Analyse TLS du point d'entrée | TLS 1.2 minimum, 80 redirigé, HSTS présent, 404 sur hôte inconnu | IaC corrigée |
| XE4 | Liste blanche vue d'Internet | « Réserver », `/admin/`, `/realms/master/`, broker : injoignables (avec X17) | Bloquant |
| XE5 | Rafale sur une route sensible | 429 au-delà du seuil ; adresse du client correcte dans les journaux | Intergiciel corrigé |

- **Code** : intergiciels d'en-têtes et de débit dans `libs/platform/` ; préfixe de route par service ; configuration nginx versionnée avec le front.
- **Coût** : environ 10 $ par an hors crédits ; 0,50 $ par mois sur les crédits ; rien par session au-delà de D03.

## À savoir défendre en soutenance

1. **« Pourquoi un domaine payant, alors que des solutions gratuites existent ? »**
   - Sans domaine, aucun certificat ACM ; un nom tiré d'une adresse exige une adresse fixe, donc un NLB, un contrôleur d'entrée et des certificats réémis à chaque session.
   - Environ 10 $ par an retirent ces trois composants et donnent un émetteur OIDC stable.
2. **« Pourquoi pas de contrôleur d'entrée ? »**
   - L'ALB, créé par l'IaC, fait le routage, le TLS et la redirection. Les en-têtes sont posés à l'origine, le HSTS par l'écouteur.
   - La limitation de débit d'un contrôleur serait elle aussi par instance : elle n'apporte rien de plus que l'intergiciel des services pour le POC.
3. **« Comment le front est-il servi en HTTPS sans CloudFront ? »**
   - Par un conteneur nginx non root derrière l'ALB ; un site web S3 ne sert que du HTTP.
4. **« Comment êtes-vous sûrs que "réserver" et la console ne sont pas joignables ? »**
   - Règles de l'ALB en liste blanche, 404 par défaut ; « réserver » sur un serveur et un port distincts (D05) ; vérifié à chaque déploiement vu d'Internet (X17, XE4).
5. **« Le nom survit-il à la recréation de l'ALB ? »**
   - La zone et le certificat sont persistants ; seul l'alias est recréé à chaque allumage, avec un cache négatif court pour que le nom réponde aussitôt.

## Sources

Consultées le 2026-10-26.

- Contraintes du lab : [environnement-learner-lab.md](../00-cadrage/environnement-learner-lab.md) (CloudFront, Route 53, ACM, WAF, API Gateway, voies HTTPS).
- ALB : modification des en-têtes de réponse https://docs.aws.amazon.com/elasticloadbalancing/latest/application/header-modification.html ; politiques TLS https://docs.aws.amazon.com/elasticloadbalancing/latest/application/describe-ssl-policies.html ; `X-Forwarded-For` https://docs.aws.amazon.com/elasticloadbalancing/latest/application/x-forwarded-headers.html
- ACM : validation DNS et renouvellement géré https://docs.aws.amazon.com/acm/latest/userguide/dns-validation.html
- Route 53 : enregistrements SOA et cache négatif https://docs.aws.amazon.com/Route53/latest/DeveloperGuide/SOA-NSrecords.html ; alias https://docs.aws.amazon.com/Route53/latest/DeveloperGuide/resource-record-sets-choosing-alias-non-alias.html ; tarifs https://aws.amazon.com/route53/pricing/
- API Gateway : VPC Link des API HTTP https://docs.aws.amazon.com/apigateway/latest/developerguide/http-api-vpc-links.html ; quotas https://docs.aws.amazon.com/apigateway/latest/developerguide/http-api-quotas.html
- Let's Encrypt : limites https://letsencrypt.org/docs/rate-limits/ ; liste des suffixes publics https://publicsuffix.org/
- sslip.io : https://sslip.io/
- Traefik : Gateway API et intergiciels https://doc.traefik.io/traefik/
- nginx non root : https://hub.docker.com/r/nginxinc/nginx-unprivileged
- Limitation de débit Go : https://pkg.go.dev/golang.org/x/time/rate
- Keycloak derrière un proxy : https://www.keycloak.org/server/reverseproxy
