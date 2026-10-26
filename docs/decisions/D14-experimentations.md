# D14 — Périmètre des expérimentations bac à sable

| Champ | Valeur |
|---|---|
| Statut | Décidée |
| Date | 2026-10-26 |
| Décideur | Martin |
| Dépend de | [D03](D03-hebergement-orchestration.md), [D05](D05-communication-interservice.md), [D06](D06-authentification-autorisation.md), [D07](D07-donnees.md), [D08](D08-exposition-tls-domaine.md), [D09](D09-observabilite.md), [D10](D10-forge-ci-cd.md), [D11](D11-iac-allumage-cout.md) |
| Impacte | Statuts de D03, D05, D06, D08 ; `docs/experimentations/XP00` à `XP03` ; `sandbox/` ; US d'infrastructure du sprint 1 ; D13 (outil de tir) ; jalon J3 de [PILOTAGE.md](../PILOTAGE.md) |
| Critères grille | C1 (principal), C5, C8, C2 |

## Contexte et besoin

D14 fixe **quelles** expérimentations sont menées avant J4, dans quel ordre et à quel coût.
Elle ne rouvre aucune décision : D03, D05, D06 et D08 restent `Candidate` avec leurs critères de bascule ; D14 les regroupe et les ordonne.
Le protocole détaillé de chaque expérimentation est rédigé ensuite par DevOps dans `docs/experimentations/XPnn-slug.md` ([PILOTAGE.md](../PILOTAGE.md) § 3.2).

### Exigences

| Source | Exigence (extrait) | Conséquence pour D14 |
|---|---|---|
| Grille, C1 niveau 5 | Protocole présenté ; technologies, plateformes et interactions ; difficultés et résultats « permettent de valider l'adoption » | Chaque XP : étapes reproductibles, versions exactes, critères fixés avant l'exécution, verdict argumenté |
| Grille, C1 « écart à éviter » | Étapes ou intégrations omises, non reproductible, résultats qui ne justifient pas totalement l'adoption | Scripts complets dans `sandbox/XPnn-slug/`, extinction et inventaire inclus |
| Consigne § 3.5 | Exemples valides : broker dans un orchestrateur avec publication et consommation, certificats sécurisant un point d'entrée, observabilité, orchestrateur, services d'un fournisseur cloud | XP01, XP02 et XP03 visent trois exemples cités |
| Consigne § 3.5 | Non valides : test d'un langage, communication front-back, tests unitaires locaux, composant avec un framework | Aucune XP ne teste Go, React ni un appel REST front-back |
| D03, décision de Martin | « Oui, comparer les deux » : EKS et ECS Fargate sur les mêmes critères mesurables | XP01 est comparative |
| D05 § Conséquences | Expérimentation du canal mutualisée avec l'orchestrateur, avant J4 (condition de WS-2) | XP02 partage le cluster de test de XP01 |
| D11 § Conséquences | Sonde du premier jour : `manage_master_user_password`, budget AWS, `iam:PassRole` vers Scheduler | Sonde XP00 |

### Contraintes

- **Calendrier** : aujourd'hui 2026-10-26 ; soutenance au plus tôt vers le 2026-11-20. J3 ne dispose que d'**une semaine** (2026-10-27 au 2026-11-02) ; J4 démarre le 2026-11-03.
- **Learner Lab** ([environnement-learner-lab.md](../00-cadrage/environnement-learner-lab.md)) : sessions de 4 h, `LabRole` seul, budget de 50 ou 100 $ affiché avec 8 à 12 h de retard, 9 instances et 32 vCPU au plus, `servicequotas:GetServiceQuota` refusé.
- **Bac à sable jetable** : charge de test = un service HTTP de test et un producteur/consommateur de test, pas le code du POC. Le POC ne dépend jamais de `sandbox/`.
- **IaC du POC pas encore écrite** (D11 : 5 à 6 j). Les XP ne peuvent pas l'attendre : elles utilisent une IaC minimale propre à `sandbox/`, réduite au réseau, au cluster et à l'ALB.

### Inventaire des critères déjà posés

| Décision | Critères | Nature |
|---|---|---|
| D03 | X1 à X21 ; bascule vers ECS si X1, X4, X7, X10 échouent, X6 sans repli, X16, X12/X19 | Choix de l'orchestrateur et mesures de sécurité MS1 à MS10 |
| D05 | XC1 à XC9 ; repli NATS JetStream | Choix du broker et droits du canal |
| D06 | Mémoire au repos et au pic, démarrage avec import du realm, refus de « réserver » selon le jeton, `/admin/` injoignable, MFA admin | Confirmation de Keycloak |
| D07 | X6 (S3 par le rôle du nœud, présignature) | Conditionne F1 |
| D08 | Sonde Route 53 et ACM ; XE1 à XE5 | Chaîne d'exposition |
| D09 | Fluent Bit vers CloudWatch par le rôle du nœud, mémoire de la pile, délégation de préfixes, alerte `.ecartes`, valeur piégée absente des journaux | Configuration de la pile retenue (D09 est `Décidée`) |
| D10 | Flux : temps d'allumage, pods consommés, crochets Helm | Configuration (D10 est `Décidée`) |
| D11 | X4, X5, X13, X14, X15 ; inventaire | Garde-fous d'exploitation |

**Constat** : environ 45 critères, mais deux natures très différentes.
- **Critères de choix** : leur échec fait basculer une décision `Candidate` (X1, X4, X6, X7, X10, X12, X16, X19, XC1, XC2, XC3, XC5, XC9, mémoire Keycloak, XE1, XE2, XE4).
- **Critères de recette** : ils vérifient la configuration du POC (X17, X18, X20, X21, XE3, XE5, critères D09 et D10). Leur échec se corrige dans l'IaC ou le chart ; il ne change aucun choix de technologie.

## Sonde du compte (XP00) — premier jour

Session de configuration AWS avec Martin ([PILOTAGE.md](../PILOTAGE.md) § 3.2), avant toute XP cloud.
Un script en lecture puis en création minimale (`sandbox/XP00-sonde/`), une heure, moins de 0,20 $ plus la zone DNS (0,50 $ par mois).
Résultats consignés dans `docs/experimentations/XP00-sonde-compte.md` ; les faits durables sont reportés une seule fois dans [environnement-learner-lab.md](../00-cadrage/environnement-learner-lab.md).

| # | Point sondé | Méthode | Si refus |
|---|---|---|---|
| S1 | Budget réel (50 ou 100 $), consommé à date, région | Lecture dans l'interface du lab | Avec 50 $ : sessions en allumage à la demande (D11) et option C de D14 à reconsidérer |
| S2 | Droits de `LabRole` : identité, rôles `LabEks*`, `iam:GetRole` sur `voclabs` | `sts get-caller-identity`, `iam list-roles` | Repli `LabRole` pour le cluster (D03) |
| S3 | EKS : versions proposées et leur support | `eks describe-cluster-versions` | 1.35 ou plus absente : coût du support étendu à intégrer, X13 revu |
| S4 | ECS et Fargate : création d'un cluster et d'une tâche de 0,25 vCPU | CLI, puis suppression | Comparatif XP01 réduit à EKS ; rejet de Fargate documentaire seulement |
| S5 | Quotas : vCPU Fargate, instances | `GetServiceQuota` (refus attendu) ; à défaut, plafond de 4 vCPU Fargate fixé dans le protocole | Plafond prudent maintenu |
| S6 | Route 53 : zone hébergée du sous-domaine, délégation chez le registrar | Création de la zone (couche persistante) | Repli D2 de D08 (DNS chez le registrar) |
| S7 | ACM : certificat `<domaine>` et `*.<domaine>` validé par DNS | `RequestCertificate`, attente `ISSUED` | Chaîne 2 de D08 |
| S8 | EventBridge Scheduler sous `LabRole` (`iam:PassRole`), comportement hors session (X15) | Planification ponctuelle à +10 min supprimant une ressource gratuite (groupe de sécurité vide), puis « End Lab » | G1 de D11 réduit à `down.sh` et au rappel sur le poste |
| S9 | AWS Budgets : alerte à 50 % et 80 % | `budgets create-budget` | G6 abandonné (D11) |
| S10 | RDS : `manage_master_user_password`, chiffrement, `rds.force_ssl` | Lecture des droits ; création différée à XP01 | `password_wo` (D11) |
| S11 | S3 sous la SCP : bucket, Block Public Access, politique refusant le non-TLS | CLI | Écart documenté (D07) |
| S12 | CloudWatch Logs : groupe avec rétention, écriture, `StartQuery` | CLI | Écart de D09 à arbitrer avant le sprint 1 |
| S13 | Cognito (repli de D06) | Liste des User Pools | Repli de D06 marqué indisponible |

Ordre : S1, S2, S6 en premier (la propagation DNS prend du temps), puis le reste pendant l'attente de S7.
S6 et S7 constituent aussi l'étape 1 de XP03 (XE1, XE2) : rien n'est fait deux fois.

## Regroupement proposé

Trois expérimentations, un seul cluster de test EKS pour XP01 à XP03, un cluster ECS pour la seule comparaison.
Chacune : étape locale sur k3d (k3s applique les politiques réseau), puis étape AWS ; extinction et inventaire (X5) à la fin de chaque session cloud.

### XP01 — Orchestrateur : EKS comparé à ECS Fargate (D03)

- **Objectif** : confirmer ou rejeter EKS sur ses critères de bascule, et produire une comparaison mesurée avec ECS Fargate.
- **Règle de verdict**, inchangée : la règle de bascule de D03. La comparaison ne la remplace pas ; elle argumente le rejet de l'option non retenue.
- **Grille commune**, mêmes seuils, même charge de test, même tir :

| # | Mesure | EKS | ECS Fargate | Seuil | Nature |
|---|---|---|---|---|---|
| G1 (X1) | Disponibilité dans le compte | Cluster et node group | Cluster et service | Créés | Bloquant |
| G2 (X4) | Allumage depuis zéro jusqu'au 200 HTTPS par l'ALB | IaC de `sandbox/` | IaC de `sandbox/` | ≤ 45 min, deux fois (EKS) | Bloquant (EKS) ; mesuré une fois (ECS) |
| G3 (X5) | Extinction puis inventaire | idem | idem | ≤ 20 min, aucune ressource facturable restante | Bloquant |
| G4 (X6) | S3 (écriture, lecture, URL présignée) sans identifiant statique | Rôle du nœud par IMDS | Rôle de tâche `LabRole` | Réussi | Bloquant |
| G5 (X7) | Délai seuil franchi → capacité prête ; p95 et erreurs | HPA | Target tracking | ≤ 90 s ; p95 < 500 ms ; < 1 % d'erreurs | Bloquant |
| G6 (X10) | Perte d'une instance sous tir | Pod supprimé | Tâche arrêtée | < 1 % d'erreurs ; remplaçant ≤ 60 s | Bloquant |
| G7 (X12, X19) | Cloisonnement : flux interdit refusé, flux autorisé fonctionnel | Politiques réseau | Groupes de sécurité par service | Refus constaté | Bloquant |
| G8 (X16) | Identifiants AWS obtenus par un composant non autorisé | IMDS depuis un pod non autorisé | Tâche sans rôle | Refus à 100 % | Bloquant (motif sécurité) |
| G9 (X13) | Coût horaire mesuré | Heures × prix | idem | ≤ 0,40 $/h | Non bloquant |
| G10 | Effort et complexité | Fichiers et lignes d'IaC et de manifests, composants à exploiter | idem | Relevé | Non bloquant |

- **Propres à EKS, non bloquants** : X2 et X3 (local), densité des pods et délégation de préfixes (D09), X14 si la session le permet.
- **Limite assumée** : la mise à l'échelle sur la file (X8, XC6) n'est mesurée que sur EKS ; sur ECS elle demande un publieur de métrique (0,5 à 1 j, D03). Elle figure dans la comparaison comme coût documenté, non mesuré.
- **Durée** : préparation 2 j (DevOps : IaC de `sandbox/`, chart de test, tâches ECS) ; exécution 0,5 j local, deux sessions EKS (partagées avec XP02 et XP03) et une session ECS de 2 h.
- **Coût** : environ 2,70 $ (EKS) et 1,25 $ (ECS, déjà prévu par D11).
- **Sortie** : D03 `Décidée` (EKS adopté, Fargate rejeté sur la grille) ou bascule vers ECS (EKS rejeté, motif mesuré).

### XP02 — Broker RabbitMQ dans l'orchestrateur (D05)

- **Objectif** : prouver la publication, la consommation et les droits du canal dans le cluster ; exemple cité par la consigne.
- **Bloquants** : XC1 (point à point et diffusion), XC2 (droits par service), XC3 (abonné arrêté, aucune perte), XC5 (redémarrage, suppression du pod, rejeu de l'outbox de test), XC9 (API de gestion, `user-id` usurpé, ports Erlang).
- **Non bloquants** : XC4 (à corriger avant J4), XC6 (KEDA, reprend X8), XC7 (avec G7), XC8 (empreinte).
- **Protocole** : en local sur docker compose puis k3d, XC1 à XC5 et la partie hors réseau de XC9 ; sur EKS, la totalité, pendant la session 1.
- **Durée et coût** : préparation 0,5 j (fichier de définitions, producteur et consommateur de test) ; exécution 0,5 j local et 1 h dans la session EKS 1 ; aucun coût propre.
- **Sortie** : D05 `Décidée` (RabbitMQ adopté) ou bascule vers NATS JetStream, avec XC2 rejoué sur la liste complète de `$JS.API`. NATS n'est expérimenté que dans ce cas.

### XP03 — Identité et exposition : Keycloak, ALB, ACM, Route 53 (D06, D08)

- **Objectif** : prouver un point d'entrée HTTPS à certificat managé, qui n'expose que la liste blanche, et un Keycloak qui tient dans le cluster.
- **Bloquants** : XE1 et XE2 (joués pendant XP00) ; XE4 étendu par ME8 (traversées et encodages vers `/admin/` et `/realms/master/`) ; Keycloak sans OOMKill ni pod en attente, au repos et au pic, sur deux t3.medium avec RabbitMQ et la charge de test ; démarrage sur base neuve avec import du realm ≤ 3 min ; « réserver » refusé sans jeton, avec un jeton d'utilisateur, d'un autre client, expiré ou d'une autre audience, accepté avec le jeton de `transactions`.
- **Non bloquants** : XE3 (analyse TLS), MFA demandé à l'admin et pas au vendeur, mémoire de la pile D09 ajoutée au relevé de densité.
- **Protocole** : en local, Keycloak, realm et vérification des jetons derrière un proxy à autorité locale (D08) ; sur EKS, session 2.
- **Durée et coût** : préparation 1 j (realm de test, règles d'ALB, service de test vérifiant l'audience) ; exécution 0,5 j local et la session EKS 2 ; zone DNS 0,50 $ par mois.
- **Sortie** : D06 et D08 `Décidées`, ou repli (Cognito ; D2 ou chaîne 2 de D08) ; décision du troisième nœud au repos (D06, D09) sur mesure.

### Mutualisation

Un seul cluster de test EKS sert les trois XP pendant deux sessions : cela économise environ deux allumages (≈ 2,70 $ et 1,5 h) et teste les intégrations réelles (broker, Keycloak et services de test cloisonnés sur les mêmes nœuds), ce que C1 demande.
Le prix : un échec bloquant de XP01 invalide aussi les mesures de XP02 et XP03 sur EKS ; elles sont alors rejouées sur ECS (une session de plus).

## Calendrier J3

| Date | Travail | Sessions cloud |
|---|---|---|
| 2026-10-27 | DevOps : protocoles XP00 à XP03, critères figés, script de sonde. Martin : sonde XP00, délégation DNS, certificat | 1 h |
| 2026-10-28 | DevOps : IaC de `sandbox/`, chart de test, définitions RabbitMQ, realm. Martin : étapes locales (k3d) de XP01 à XP03 | — |
| 2026-10-29 | Session EKS 1 : G1 à G8 côté EKS (G2 première mesure), XP02 complète, densité, X5 | 4 h |
| 2026-10-30 | Session EKS 2 : G2 seconde mesure, XP03, X14 si le temps le permet, G9, X5 | 4 h |
| 2026-10-31 | Session ECS : grille G1 à G10 côté Fargate, X5 | 2 h |
| 2026-11-01 | Résultats, relectures architecture et sécurité, verdicts de Martin, mise à jour de D03, D05, D06, D08 et de l'index | — |
| 2026-11-02 | Réserve : une session de rejeu en cas de bascule | 0 à 4 h |

- Le chemin critique est la préparation de DevOps (≈ 3,5 j comprimés en 2 jours) : c'est le risque principal du calendrier.
- Une bascule (EKS ou RabbitMQ) consomme la réserve ; au-delà, le sprint 1 part en WS-1 (D02), sans attendre.
- **Budget J3** : ≈ 0,2 + 2,70 + 1,25 + 0,50 ≈ **4,65 $**, réserve de rejeu ≈ 1,35 $.

## Options

### Option A — Tout expérimenter

- **Description** : X1 à X21, XC1 à XC9, critères Keycloak, XE1 à XE5, critères D09 et D10, sur EKS, et la transposition complète de X1 à X21 sur ECS.
- **Avantages** : couverture maximale ; toutes les mesures de sécurité démontrées avant le premier code du POC.
- **Inconvénients** : 10 à 12 j et 5 à 6 sessions ; plusieurs critères (X17, X18, X20, X21, XE5, Flux, Fluent Bit) testeraient l'IaC jetable de `sandbox/` et devraient être refaits sur celle du POC.
- **Coût AWS et effort** : ≈ 8 $ ; J3 déborde d'une à deux semaines sur un calendrier de 25 jours.
- **Complexité ajoutée** : deux jeux d'IaC complets à maintenir.
- **Couverture grille** : C1 complet, mais retarde C2, C3, C5 (sprints) au point de les menacer.

### Option B — Critères de choix et comparaison EKS/Fargate (recommandée)

- **Description** : XP00, puis XP01 à XP03 réduites aux critères de choix et à la grille commune G1 à G10. Les critères de recette (X17, X18, X20, X21, XE3 en contrôle répété, XE5, critères D09 et D10, X9, X11) deviennent des AC des US d'infrastructure du sprint 1, vérifiés sur l'IaC du POC.
- **Avantages** : tient en une semaine ; chaque XP peut changer une décision, ce qui donne du poids à son verdict ; les critères de recette sont vérifiés là où ils comptent, sur l'environnement de démonstration.
- **Inconvénients** : X17 et X20, conditions de l'avis favorable de D03, sont prouvées au sprint 1 et non avant ; le Cluster Autoscaler (X9) n'est mesuré qu'au sprint 2, avec la charge.
- **Coût AWS et effort** : ≈ 4,65 $ ; ≈ 3,5 j de préparation et 3 sessions.
- **Complexité ajoutée** : une IaC de `sandbox/` minimale, jetée après J3.
- **Couverture grille** : C1 complet (trois technologies citées, une adoption et un rejet mesurés) ; C5 et C8 au sprint 1.

### Option C — Minimal

- **Description** : XP00 et XP02 seules (broker dans EKS) ; EKS, Keycloak et la chaîne d'exposition confirmés au sprint 1.
- **Avantages** : 2 j, 1 session, ≈ 2 $.
- **Inconvénients** : contredit la décision de Martin sur D03 (« comparer les deux ») ; aucun rejet mesuré, donc un C1 fragile ; un échec d'EKS ou de Keycloak découvert en plein sprint 1.
- **Coût AWS et effort** : ≈ 2 $ ; 2 j.
- **Complexité ajoutée** : nulle.
- **Couverture grille** : C1 partiel (adoption sans rejet argumenté).

## Recommandation

**Option B.** C'est la seule qui tient dans la semaine de J3 tout en respectant la décision de comparer EKS et Fargate.
Le tri entre critères de choix et critères de recette retire de J3 ce qui ne peut changer aucune décision, sans rien perdre : ces critères sont vérifiés au sprint 1 sur l'environnement réel.
La sonde du premier jour lève à elle seule cinq incertitudes (Route 53, ACM, Scheduler, budget, Cognito) pour 1 h et quelques centimes.

## Ce qui alimente C1

| Élément | Source | Attendu par le niveau 5 |
|---|---|---|
| Adoption argumentée | XP01 (orchestrateur retenu), XP02 (broker dans l'orchestrateur), XP03 (certificat sur le point d'entrée) | Résultats mesurés au regard de critères fixés avant |
| Rejet argumenté | XP01 : l'orchestrateur non retenu, sur la grille G1 à G10 | Même charge, mêmes seuils, coût et effort relevés |
| Technologies et interactions | Schéma du cluster de test : ALB, services de test, RabbitMQ, Keycloak, RDS, S3, IMDS | Chaque XP lie son schéma |
| Difficultés | Section « Difficultés » de chaque XP, tenue pendant l'exécution | Notées par Martin, sans réécriture |
| Reproductibilité | `sandbox/XPnn-slug/`, versions épinglées, extinction et inventaire | Rejouable de zéro |

- Le rejet n'est **pas joué d'avance** : si Fargate l'emporte sur les critères de bascule, c'est EKS qui est rejeté, et le dossier C1 reste complet.
- Les refus constatés par la sonde (Amazon MQ, CloudFront, Cognito éventuel) sont des faits du lab, pas des rejets expérimentaux : ils ne comptent pas pour C1.

## Avis sécurité

Fonction : Sécurité — 2026-10-26. Références : avis sécurité et critères X12, X16 à X21, MS1 à MS6 de [D03](D03-hebergement-orchestration.md) ; XC2, XC7, XC9, MC1 à MC5 de [D05](D05-communication-interservice.md) ; tests négatifs de jetons et MA7 de [D06](D06-authentification-autorisation.md) ; ME4, ME5, ME8, ME10 de [D08](D08-exposition-tls-domaine.md) ; [exigences-securite.md](../securite/exigences-securite.md) (ES-06, ES-07, ES-38). Avis sur dossier, avant toute sonde.

**Synthèse.** Avis favorable à l'option B. Le tri entre critères de choix et critères de recette est juste du point de vue de la sécurité : les trois critères qui peuvent faire rejeter EKS pour un motif de sécurité (X16, X12, X19) et ceux qui fondent le choix de RabbitMQ (XC2, XC9) restent dans le bac à sable. Deux réserves : XC7 doit y rester bloquant, et le bac à sable lui-même doit respecter un socle minimal (MX1 à MX8), car il expose un ALB, un Keycloak et le rôle `LabRole` sur un vrai compte.

**1. Report vers les AC du sprint 1**

- **X18 et X21** : report accepté sans réserve ; ils sont non éliminatoires et ne testent que la configuration. Exception : si X16 n'est tenu que par le repli de MS3 (composants système en `hostNetwork`), l'interdiction de `hostNetwork` dans les namespaces applicatifs devient une condition de ce repli ; X18 est alors joué en local dès XP01.
- **X17** : report accepté pour la version complète (balayage du VPC du POC, inventaire des Services, règles de l'ALB). Il reste une **condition de l'avis favorable de D03** : D03 peut passer `Décidée` à l'issue de XP01, mais X17 devient un AC **bloquant** de la première US d'infrastructure, avant toute ouverture de l'ALB du POC à Internet. La partie qui protège le bac à sable lui-même (MX2, MX3) n'est pas reportée.
- **X20** : report accepté pour les releases Helm, la CI et l'analyse des images du POC, avec le même statut bloquant (révocation, puis procédure corrigée). La partie « aucun secret dans l'IaC de `sandbox/`, son état et ses journaux » n'est pas reportée : elle est réduite à MX4 et MX5.
- **XE3, XE5, X9, X11, critères D09 et D10** : report accepté. ME4 (XE5) reste bloquant avant toute ouverture à Internet de l'ALB du POC, comme le prévoit D08.

**2. Critères qui restent bloquants dans le bac à sable**

- **X16 (G8)** : bloquant, joué sur le cluster EKS, jamais seulement sur k3d (pas d'IMDS en local). Il doit couvrir l'écart résiduel relevé dans D05 : depuis un pod non autorisé, tenter d'obtenir `LabRole` puis de lire, comme un nœud, les Secrets d'un autre namespace. Un succès invalide le cloisonnement des Secrets par service : bascule vers B, motif de sécurité.
- **X12 et X19 (G7)** : bloquants, sur EKS, avec l'application des politiques réseau du CNI du VPC activée et constatée ; chaque refus accompagné du flux autorisé, sinon un refus peut venir d'une panne et non d'une politique. Le passage sur k3d ne prouve rien pour EKS.
- **XC7** : à reclasser **bloquant** (D14 le place en non bloquant « avec G7 »). Le verdict sécurité de D05 en fait une condition, au même titre que XC2 et XC9 : la console de gestion et le port AMQP joignables par un pod quelconque annulent MC1 et MC4. Le test est le même que G7, sans coût supplémentaire.
- **XC2 et XC9** : bloquants, comme proposé ; XC9 inclut l'absence de `guest` et les ports 4369 et 25672 sur EKS.
- **XE4 étendu par ME8, et refus de « réserver » selon le jeton** : bloquants, comme proposé.
- **MFA de l'admin en non bloquant** : accepté, la console n'étant jamais routée (ME8 la protège).

**3. Sécurité du bac à sable**

- **Données** : uniquement fictives ; comptes Keycloak de test à mots de passe générés par session ; aucune donnée personnelle, aucune réutilisation d'un mot de passe réel de Martin.
- **Exposition** : l'ALB du bac à sable porte Keycloak et des services de test sans durcissement complet. Il reste restreint à l'adresse de Martin (ME5) pendant toute XP, ECS compris ; l'ouverture à Internet n'est pas nécessaire au tir, qui part du poste.
- **IMDS** : si X16 échoue, les identifiants obtenus ne doivent jamais être recopiés dans le protocole ou un journal ; seul le constat (code de retour, `AccessKeyId` tronqué) est consigné, puis la session est terminée par « End Lab ».
- **Sonde S8 (X15)** : la planification cible l'ARN exact d'un groupe de sécurité créé et étiqueté par la sonde, jamais un motif ni une liste ; exécution unique, planification supprimée après exécution. `LabRole` confié à Scheduler a des droits larges : une cible mal écrite supprimerait autre chose.
- **Destruction** : l'inventaire X5 du bac à sable couvre aussi les secrets de Secrets Manager (suppression sans délai de récupération), les groupes de journaux, les volumes EBS, les instantanés RDS finaux et les alias DNS ; seuls la zone et le certificat persistent (ME10 s'applique à la zone).

**4. Mesures du bac à sable**

| Id | Mesure | Exigences, menaces | Preuve |
|---|---|---|---|
| MX1 | Données et comptes de test fictifs, générés à chaque session | ES-06 | Revue du realm et des scripts de `sandbox/` |
| MX2 | Groupe de sécurité de l'ALB limité à l'adresse de Martin (/32) pendant toutes les XP, EKS et ECS | ES-07, M-LIV-03 | Inventaire en fin de session |
| MX3 | API du cluster de test limitée à l'adresse du poste (MS1), nœuds et RDS sans adresse publique | ES-07 | `kubectl` refusé hors plage, une fois en session 1 |
| MX4 | Aucun secret dans `sandbox/` : génération à l'exécution, état Terraform local ignoré par le dépôt, `manage_master_user_password` si S10 le permet | ES-06, M-LIV-02 | Recherche de secrets sur `sandbox/` et son historique avant chaque dépôt |
| MX5 | Aucune valeur de secret ni identifiant AWS dans les journaux de session, les sorties de script et les protocoles XP ; valeur piégée cherchée en fin de session | ES-06, ES-23 | Recherche de la valeur piégée dans les journaux conservés |
| MX6 | Planification S8 limitée à l'ARN d'une ressource étiquetée de la sonde, exécution unique | ES-38 | Planification et ressource absentes après le test |
| MX7 | Inventaire de destruction étendu (secrets, journaux, EBS, instantanés, alias DNS) | ES-07, ES-06 | Inventaire X5 vide hors couche persistante |
| MX8 | X17 et X20 inscrits en AC bloquants de la première US d'infrastructure, avant toute ouverture de l'ALB du POC à Internet | ES-06, ES-07 | AC du sprint 1 |

**Verdict** : avis favorable à l'option B, sous réserve de MX1 à MX8 et du reclassement de XC7 en bloquant. X16, X12, X19, XC2, XC7, XC9 et XE4 étendu restent des critères de bascule joués sur EKS ; X17 et X20 restent des conditions de l'avis favorable de D03, levées au sprint 1.

## Avis faisabilité

Fonction : DevOps — 2026-10-26. Références : D11 (Q3, allumage chronométré, effort), avis sécurité ci-dessus (MX1 à MX8), état du poste relevé le 2026-10-26.

**Synthèse.** Option B faisable, mais **le calendrier proposé ne tient pas** : la préparation réaliste est de 4 à 4,5 j, pas 3,5 j, et la session EKS 1 du 2026-10-29 suppose une IaC déjà rodée et une étape locale terminée la veille, le jour même où DevOps l'écrit. Calendrier recalé ci-dessous, réserve réduite à une demi-journée. Le double travail avec l'IaC du POC s'évite si l'IaC de `sandbox/` est écrite d'emblée à la forme de `infra/ephemere/`.

**1. Effort de préparation réestimé**

| Lot | D14 | Réaliste | Motif |
|---|---|---|---|
| IaC `sandbox/` : réseau, EKS (`aws_eks_cluster` et `aws_eks_node_group` directs, sans module), modèle de lancement IMDSv2, CNI avec politiques réseau, ALB | 2 j avec le reste | 1,25 j | Pas de module communautaire (`iam:GetRole`), ARN de `LabRole` construit ; ALB en groupe cible de type instance (NodePort), sans AWS Load Balancer Controller, pour ne pas dépendre d'un rôle IRSA |
| Variante ECS : cluster, définitions de tâche (avec `LabRole`, sans rôle pour G8), service, groupe cible de type ip sur le même ALB, groupes de sécurité par service, target tracking | inclus | 0,75 j | Réutilise le réseau et l'ALB de la variante EKS |
| Chart de test, service HTTP de test, script de mesure, inventaire X5/MX7 | inclus | 0,75 j | L'inventaire étendu (MX7) est celui de G2 de D11 : écrit une fois |
| XP02 : définitions RabbitMQ, producteur et consommateur de test | 0,5 j | 0,5 j | — |
| XP03 : realm, règles d'ALB, service vérifiant l'audience, proxy local | 1 j | 1 j | — |
| **Total** | **3,5 j** | **4,25 j** | Protocoles XP00 à XP03 (0,5 j) en sus, le 2026-10-27 |

**2. Calendrier recalé**

| Date | DevOps | Martin | Cloud |
|---|---|---|---|
| 2026-10-27 | Protocoles, critères figés, script de sonde, outil de tir épinglé | Session de configuration, XP00 ; délégation DNS en tout premier (propagation) | 1 h + 0,5 h d'`apply` du seul réseau de `sandbox/` pour roder l'IaC |
| 2026-10-28 | IaC EKS et ALB, chart de test, script de mesure | XP02 en local (docker compose) | — |
| 2026-10-29 | Variante ECS, realm, inventaire | XP01 à XP03 sur k3d | — |
| 2026-10-30 | Correctifs, revue MX4 | Session EKS 1 : G1 à G8, XP02 sur EKS, MX3, X5 | 4 h |
| 2026-10-31 | Correctifs | Session EKS 2 : G2 seconde mesure, XP03, G9, X5 | 4 h |
| 2026-11-01 | Résultats | Session ECS (G1 à G10), puis rédaction | 2 h |
| 2026-11-02 | Relectures | Verdicts, mise à jour de D03, D05, D06, D08 | Réserve : une demi-session de rejeu |

- Une bascule (EKS ou RabbitMQ) déborde sur 2026-11-03 : le sprint 1 démarre alors en WS-1, comme prévu par D14.
- **Session de 4 h** : `up` ≤ 45 min, `down` et inventaire ≈ 25 min, refus d'allumer sous 75 min restantes, aucun `apply` après la 3e heure : il reste ≈ 2 h 30 utiles. La session 1 est la plus chargée (G5 et G6 sous tir, XP02 complète) : liste ordonnée, bloquants d'abord ; densité et X14 coupés en premier.
- **Local puis AWS** : l'ordre est juste ; k3d ne prouve ni X16 ni G7 (réserve sécurité), il sert à déboguer chart, realm et définitions hors facturation.

**3. Réutilisation pour l'IaC du POC**

- Écrire `sandbox/XP01-orchestrateur/` avec le découpage par domaine de `infra/ephemere/` (réseau, EKS, ALB), mêmes étiquettes G3, mêmes versions épinglées. Après J3, le code est **copié** dans `infra/ephemere/` (le POC ne dépend jamais de `sandbox/`, qui reste figé comme preuve de reproductibilité C1). Gain estimé sur le lot « IaC des deux couches » de D11 : ≈ 1 j sur 2,5 à 3 j.
- La zone DNS et le certificat (S6, S7) sont créés par Terraform dans `infra/persistant/` dès XP00, pas en CLI : aucune importation à refaire au sprint 1.
- L'inventaire (X5, MX7) et le chronométrage sont écrits dans la bibliothèque commune de `tools/` et appelés par `sandbox/` : sens de dépendance autorisé, et repris tels quels par `down.sh`.

**4. Outillage de mesure**

- **Durées** (G2, G3, G5, G6) : un script unique horodate chaque étape (`date +%s`) dans un CSV par XP sous `docs/mesures/` ; G5 et G6 relèvent l'instant du seuil franchi et l'instant `Ready` (événements Kubernetes ; `runningCount` d'ECS interrogé toutes les 5 s).
- **Tir** : choisir l'outil avant le 2026-10-28, pas le 29 (D13), car il sert dès l'étape k3d ; à défaut, un outil HTTP simple épinglé donnant p95 et taux d'erreur.
- **Mémoire et pods** : `kubectl top` (metrics-server, déjà prévu), `OOMKilled` et pods `Pending` relevés par requête sur l'état des pods ; côté Fargate, métriques standard `MemoryUtilization` du service, sans Container Insights (payant).
- **G8** : IMDS depuis un pod non autorisé ; côté ECS, point de métadonnées de tâche (169.254.170.2) depuis la tâche sans rôle. Sortie tronquée (MX5).
- **G10** : comptage des fichiers et lignes de chaque variante par script, non à la main.

**5. Dépendance à la session de configuration AWS**

- Relevé du poste : Terraform, AWS CLI, kubectl, helm et Docker présents ; **k3d absent**, à installer avant le 2026-10-29. Versions à relever dans le protocole XP00.
- Prérequis restant à Martin : accès au Learner Lab, identifiants de session en profil nommé (MI6), accès au registrar du domaine pour la délégation. Tout retard de cette session décale l'ensemble d'autant : aucune XP cloud n'est possible sans S2, S3 et S4.

**Risques.** (1) Premier `apply` EKS défaillant consommant la session 1 : atténué par le rodage du réseau le 2026-10-27 et le parcours k3d. (2) Expiration en plein `apply` : inventaire avant tout nouvel `apply` (D11). (3) Propagation DNS au-delà d'un jour : XP03 passe en session ECS ou en réserve. (4) Charge de Martin : 11 h de sessions et 1,5 j local sur six jours.

**Verdict** : faisable sous réserve du calendrier recalé, de la copie de l'IaC de `sandbox/` vers `infra/ephemere/` et de l'outil de tir arrêté avant le 2026-10-28. Effort : ≈ 4,75 j DevOps (protocoles compris), coût inchangé (≈ 4,65 $ plus 0,10 $ de rodage du réseau).

## Décision de Martin

> **« B, calendrier recalé »** : sonde du compte XP00, puis XP01 (EKS contre ECS Fargate sur la grille commune G1 à G10), XP02 (RabbitMQ dans l'orchestrateur) et XP03 (Keycloak, ALB, ACM et Route 53), d'abord en local sur k3d puis sur AWS. Critères bloquants : X16 (élargi à la lecture des Secrets d'un autre namespace), X12/X19, XC2, XC7, XC9, XE4/ME8. X17 et X20 deviennent des critères d'acceptation bloquants de la première US d'infrastructure, avant toute ouverture de l'ALB du POC. Mesures du bac à sable MX1 à MX8 (données fictives, ALB restreint à l'adresse de Martin, inventaire étendu).
>
> Calendrier : sonde et préparation du 2026-10-27 au 2026-10-29, sessions EKS le 2026-10-30 et le 2026-10-31, session ECS le 2026-11-01, demi-session de réserve et verdicts le 2026-11-02. L'IaC du bac à sable est écrite au format d'`infra/ephemere/` pour être reprise par le POC.

Date : 2026-10-26

## Conséquences

- **XP** : protocoles `docs/experimentations/XP00-sonde-compte.md`, `XP01-orchestrateur.md`, `XP02-canal-rabbitmq.md`, `XP03-identite-exposition.md`, scripts dans `sandbox/` du même nom.
- **Statuts** : D03, D05, D06 et D08 passent en `Candidate (XP01)`, `(XP02)`, `(XP03)` ; l'index est mis à jour.
- **Sprint 1** (option B) : les critères de recette deviennent des AC des US d'infrastructure ; X5, X15 et l'inventaire y sont rejoués sur l'IaC du POC.
- **D13** : l'outil de tir doit être choisi avant le 2026-10-29, ou un tir HTTP simple épinglé dans le protocole de XP01.
- **D11** : budget J3 (≈ 4,65 $) imputé à l'enveloppe ; la ligne « expérimentation comparative ECS » est confirmée.
- **Budget de 50 $** (S1) : l'option B reste tenable ; la réserve de rejeu devient la variable d'ajustement.

## À savoir défendre en soutenance

1. **« Pourquoi ces expérimentations et pas d'autres ? »**
   - Chacune porte une technologie que la consigne cite en exemple : orchestrateur, broker dans un orchestrateur, certificats sur un point d'entrée.
   - Chacune peut changer une décision : ses critères de bascule sont écrits avant l'exécution.
2. **« Comment avez-vous rendu la comparaison EKS / Fargate équitable ? »**
   - Même charge, même tir, mêmes seuils, IaC de même périmètre ; critères G1 à G10 fixés avant.
   - Limite déclarée : la mise à l'échelle sur la file n'est mesurée que sur EKS ; son coût sur ECS est chiffré.
3. **« Pourquoi certains contrôles de sécurité ne sont-ils pas dans le bac à sable ? »**
   - Ils vérifient la configuration du POC, pas le choix d'une technologie ; les tester sur une IaC jetable aurait obligé à les refaire.
   - Ils sont des AC du sprint 1, vérifiés sur l'environnement de démonstration.
4. **« Combien ont coûté les expérimentations ? »**
   - Estimation ≈ 4,65 $, puis coût mesuré (G9) ; extinction et inventaire après chaque session.
