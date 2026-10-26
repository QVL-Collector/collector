# XP01 — Orchestrateur : EKS comparé à ECS Fargate

| Champ | Valeur |
|---|---|
| Décision testée | [D03](../decisions/D03-hebergement-orchestration.md) (statut `Candidate (XP01)`) |
| Dates prévues | Local k3d 2026-10-29 ; session EKS 1 2026-10-30 ; session EKS 2 2026-10-31 ; session ECS 2026-11-01 ; réserve 2026-11-02 |
| Exécutée par | Martin |
| Scripts | `sandbox/XP01-orchestrateur/` (découpage d'`infra/ephemere/` : réseau, EKS, ALB ; variante ECS) |
| Dépend de | [XP00](XP00-sonde-compte.md) : S2, S3, S4, S5, S7, S10, S11 |
| Partage | Cluster EKS de test commun avec [XP02](XP02-canal-rabbitmq.md) (session 1) et [XP03](XP03-identite-exposition.md) (session 2) |

## Objectif et critères de succès

Objectif : confirmer ou rejeter EKS sur ses critères de bascule (D03), et produire une comparaison mesurée avec ECS Fargate sur la même grille, la même charge et le même tir (D14).

Hypothèse : EKS (node group géré de deux t3.medium) s'allume en 45 min au plus, réserve l'IMDS et cloisonne les flux par les politiques réseau du CNI du VPC ; Fargate s'allume plus vite mais réagit plus lentement à la charge.

Règle de verdict, inchangée (D03) : EKS est rejeté si G1, G4 sans repli viable, G5, G6, G7 (politiques inopérantes) ou G8 échouent, ou si G2 dépasse 60 min. La comparaison ne remplace pas cette règle : elle argumente le rejet de l'option non retenue. Le rejet n'est pas joué d'avance.

### Grille commune G1 à G10 (bloquants d'abord)

| # | Critère | Mesure | EKS | ECS Fargate | Seuil | Bloquant | Si échec |
|---|---|---|---|---|---|---|---|
| G1 (X1) | Disponibilité | Création par l'IaC de `sandbox/` | Cluster 1.35 + node group | Cluster + service | Créés, `ACTIVE` | **Oui** | EKS : bascule vers ECS. ECS : comparatif documentaire |
| G2 (X4) | Allumage depuis zéro | Du lancement de `terraform apply` au premier HTTPS 200 par l'ALB sur `collector.qvl-project.com` | 2 mesures (sessions 1 et 2) | 1 mesure | ≤ 45 min, deux fois de suite | **Oui** (EKS) | 45 à 60 min : optimisation requise, non éliminatoire. > 60 min : bascule vers ECS |
| G3 (X5) | Extinction et inventaire | Du lancement du `destroy` à l'inventaire vide (MX7) | Chaque session | Chaque session | ≤ 20 min ; aucune ressource facturable hors couche persistante | **Oui** | IaC corrigée avant toute autre session |
| G4 (X6) | S3 sans identifiant statique | Écriture, lecture, URL présignée depuis le service autorisé | Rôle du nœud par IMDSv2 | Rôle de tâche `LabRole` | 3 opérations réussies | **Oui** | Repli : identifiants de session injectés, documenté. Sans repli viable : bascule vers ECS |
| G5 (X7) | Montée en charge | Délai entre le seuil franchi et la nouvelle capacité prête ; p95 et erreurs une fois la capacité ajoutée | HPA (CPU 60 %, 2 à 6 réplicas) | Target tracking (CPU 60 %, 2 à 6 tâches) | ≤ 90 s ; p95 < 500 ms ; < 1 % d'erreurs | **Oui** | EKS : bascule vers ECS si le délai est imprévisible (écart > 2× entre deux tirs) |
| G6 (X10) | Perte d'une instance sous tir | Suppression forcée d'un pod / arrêt d'une tâche, 3 réplicas | `kubectl delete pod --force` | `ecs stop-task` | < 1 % d'erreurs sur la fenêtre ; remplaçant prêt ≤ 60 s | **Oui** | EKS : bascule vers ECS |
| G7 (X12, X19) | Cloisonnement | Flux interdit refusé, flux autorisé fonctionnel, par paire | Politiques réseau du CNI du VPC, application activée et constatée | Un groupe de sécurité par service | 100 % des refus constatés, 100 % des flux autorisés passants | **Oui** | Politique mal réglée : corrigée et rejouée. Politiques inopérantes : bascule vers ECS, motif sécurité |
| G8 (X16 élargi) | Identifiants AWS hors composant autorisé | IMDSv2 et IMDSv1 depuis un pod non autorisé ; lecture des Secrets d'un autre namespace | IMDS depuis un pod non autorisé | Point de métadonnées 169.254.170.2 depuis une tâche sans rôle | Refus dans 100 % des cas ; succès pour le seul pod autorisé (G4) | **Oui** (sécurité) | Repli de MS3 (limite de sauts à 1, composants système en `hostNetwork`, X18 joué en local) ; sans repli : bascule vers ECS, motif sécurité |
| G9 (X13) | Coût horaire | Heures-ressources × tarifs us-east-1 | Sessions 1 et 2 | Session ECS | ≤ 0,40 $/h | Non | Revue du dimensionnement |
| G10 | Effort et complexité | Fichiers et lignes d'IaC et de manifests, composants à exploiter, comptés par script | Variante EKS | Variante ECS | Relevé | Non | — |

### Détail des tests de G7 et G8

| Test | Source | Cible | Attendu | Flux autorisé témoin |
|---|---|---|---|---|
| G7-1 (X12) | Pod quelconque (`ns-intrus`) | Port interne « réserver » du service de test `pdm-test` | Refus (délai expiré) | `transactions-test` → « réserver » : 200 |
| G7-2 (X12, XC7) | Pod quelconque | Broker, 5672 et 15672 | Refus | Joué dans XP02 |
| G7-3 (X19) | `controle-test` | RDS 5432 | Refus | `pdm-test` → RDS : connexion TLS établie |
| G7-4 (X19) | Pod quelconque | Console Keycloak (8080 `/admin/`, 9000) | Refus | Joué dans XP03, session 2 |
| G7-5 | Constat | Add-on `vpc-cni` | `enableNetworkPolicy` à `true`, conteneur de politique réseau présent sur chaque nœud | — |
| G8-1 | `ns-intrus` | `PUT http://169.254.169.254/latest/api/token` puis `GET .../iam/security-credentials/` | Refus ou délai expiré | `pdm-test` : jeton et identifiants obtenus (G4) |
| G8-2 | Tout pod | IMDSv1 `GET http://169.254.169.254/latest/meta-data/` | 401 ou délai expiré | — |
| G8-3 | `ns-intrus` | API du cluster : lecture d'un Secret de `pdm-test` (jeton de compte de service absent, puis identité de nœud si G8-1 a fourni `LabRole`) | Refus (403 ou aucun identifiant) | — |
| G8-4 (constat) | `pdm-test` (autorisé, porte `LabRole`) | Lecture, comme un nœud, d'un Secret d'un autre namespace | Relevé, non bloquant : écart résiduel de MS3 documenté (ES-38) | — |

### Critères propres à EKS, non bloquants

| # | Mesure | Seuil | Si échec |
|---|---|---|---|
| X2 | Même chart en local (k3d) et sur EKS | Seul le fichier de valeurs change | Dérive corrigée |
| X3 | HPA local sous tir | 2 → au moins 4 réplicas, puis retour à 2 | Corrigé avant la session 1 |
| X7 bis | Retour au minimum après le tir (ENF-16) | ≤ 10 min | Valeurs « démonstration » ajustées |
| Densité | Pods et mémoire par nœud au repos et au pic, avec RabbitMQ et Keycloak ; essai de la délégation de préfixes (`ENABLE_PREFIX_DELEGATION`) | Relevé ; aucun pod `Pending` | Troisième nœud ou délégation (D06, D09) |
| X14 | « Start Lab » relancé pendant que la pile tourne | `kubectl` rétabli ≤ 5 min, pile intacte | Procédure de soutenance adaptée |
| MX3 | `kubectl` depuis une adresse hors plage (CloudShell) | Refus (délai expiré) | IaC corrigée avant la session 2 |

Limite déclarée : la mise à l'échelle sur la file (X8, XC6) n'est mesurée que sur EKS (XP02) ; son coût sur ECS (publieur de métrique, 0,5 à 1 j) figure dans la comparaison comme estimation.

## Environnement

### Local (2026-10-29)

- Poste de Martin, Windows 11, Docker 28.5.1, k3d (version XP00), cluster k3d à 1 serveur et 2 nœuds de travail ; k3s applique les politiques réseau (contrôleur intégré).
- Sert à déboguer chart, politiques, HPA et scripts de mesure ; ne prouve ni G7 ni G8 (pas de CNI du VPC, pas d'IMDS).

### AWS

- us-east-1, VPC dédié sur deux zones, une NAT gérée, ALB créé par Terraform (groupes cibles de type instance vers les NodePorts, sans AWS Load Balancer Controller), RDS PostgreSQL db.t3.micro Single-AZ privée, bucket S3 créé en CLI.
- EKS : `aws_eks_cluster` et `aws_eks_node_group` directs, 2 t3.medium (max 4), modèle de lancement IMDSv2 obligatoire et limite de sauts à 2, API publique restreinte au /32 du poste (MS1).
- ECS : cluster Fargate, tâches 0,25 vCPU / 0,5 Gio, 2 à 6 tâches, groupe cible de type ip sur le même ALB, un groupe de sécurité par service.
- Groupe de sécurité de l'ALB restreint à l'adresse de Martin (/32) pendant toute l'XP (MX2).

### Versions à épingler

| Élément | Version | État |
|---|---|---|
| Terraform | 1.14.5 | Fixée |
| Provider AWS | à fixer avant exécution (fichier de verrouillage versionné) | À fixer |
| AWS CLI | 2.33.24 | Fixée |
| kubectl | 1.34.1 (aligner sur 1.35 si disponible) | Fixée |
| Helm | 4.1.3 | Fixée |
| k3d / image k3s | relevées en XP00 ; image k3s 1.35.x par empreinte | À fixer avant exécution |
| EKS | 1.35 (S3 de XP00) | À confirmer par XP00 |
| Add-ons `vpc-cni`, `coredns`, `kube-proxy`, `metrics-server` | versions par défaut relevées en S3 | À fixer avant exécution |
| AMI des nœuds | Amazon Linux 2023 optimisée EKS, identifiant relevé en S3 | À fixer avant exécution |
| PostgreSQL RDS | version par défaut relevée en S10 | À fixer avant exécution |
| Service HTTP de test | image consommant de la CPU par requête, épinglée par empreinte | À fixer avant exécution |
| k6 | relevée en XP00 | À fixer avant exécution |
| Image de test des sondes réseau | image `curl` officielle épinglée par empreinte | À fixer avant exécution |

## Technos et interactions

Poste (Terraform, kubectl, k6) → ALB (HTTPS, certificat ACM, `collector.qvl-project.com`) → NodePort des nœuds EKS ou tâches Fargate → service HTTP de test. Pods `pdm-test`, `controle-test`, `transactions-test`, `ns-intrus` dans des namespaces distincts ; RDS en sous-réseaux privés ; S3 par le rôle du nœud (IMDS) ou le rôle de tâche. Schéma à lier dans `docs/architecture/schemas/` à la rédaction des résultats.

## Sécurité du bac à sable

| Mesure | Application dans XP01 |
|---|---|
| MX1 | Données fictives ; mots de passe RDS et comptes générés à l'exécution |
| MX2 | Groupe de sécurité de l'ALB limité au /32 de Martin, EKS et ECS ; vérifié à l'inventaire |
| MX3 | API du cluster au /32 du poste ; nœuds et RDS sans adresse publique ; test depuis CloudShell en session 1 |
| MX4 | Aucun secret dans `sandbox/` ; état Terraform local ignoré par le dépôt ; `manage_master_user_password` si XP00 S10 le permet ; recherche de secrets avant chaque dépôt |
| MX5 | Si G8 échoue, seul le constat (code de retour, `AccessKeyId` tronqué aux 4 premiers caractères) est consigné, puis « End Lab » ; valeur piégée cherchée dans les journaux en fin de session |
| MX7 | Inventaire étendu : EKS, EC2 (arrêtées comprises), NAT, Elastic IP, ELBv2, groupes cibles, RDS et instantanés, volumes et ENI `available`, groupes de sécurité, secrets, groupes de journaux, planifications, alias DNS, buckets `collector-xp01-*` |
| MX8 | X17 et X20 inscrits en AC bloquants de la première US d'infrastructure ; non joués ici |

## Étapes reproductibles

### A. Local, k3d (2026-10-29, ≈ 0,5 j)

1. `k3d cluster create` (1 serveur, 2 nœuds de travail, image k3s épinglée). Attendu : 3 nœuds `Ready`.
2. Installation du chart de test avec les valeurs locales. Attendu : pods `Ready` ; X2 noté (diff des seules valeurs).
3. Politiques réseau en refus par défaut, puis tests G7-1 et G7-3 (base PostgreSQL locale). Attendu : refus et flux témoins conformes ; sert au débogage seulement.
4. Tir k6 (profil de G5 réduit). Attendu : X3, 2 → au moins 4 → 2 réplicas.
5. Si G8 ne peut tenir que par le repli de MS3 : admission refusant `hostNetwork`, `privileged`, `hostPath`, Service `LoadBalancer` dans les namespaces applicatifs (X18). Attendu : 4 refus.
6. `k3d cluster delete`. Attendu : aucun conteneur restant.

### B. Session EKS 1 (2026-10-30, 4 h) — bloquants d'abord

Règles de session (D14) : « Start Lab » juste avant ; refus d'allumer sous 75 min restantes ; aucun `apply` après la 3e heure ; densité et X14 coupés en premier si le temps manque.

1. Contrôles : identité, région, budget, inventaire vide, adresse du poste relevée (non consignée).
2. **G2 n° 1** : `apply` horodaté phase par phase (réseau, cluster et RDS, node group, add-ons, chart), jusqu'au premier `curl -s -o /dev/null -w '%{http_code}' https://collector.qvl-project.com/` = 200 (réessais 2 min pour le DNS). **G1** relevé au passage.
3. **MX3** : depuis CloudShell, `aws eks update-kubeconfig` puis `kubectl get ns`. Attendu : délai expiré.
4. **G7-5** puis **G8-1 à G8-4** depuis des pods de test éphémères. Attendu : tableau ci-dessus.
5. **G4** : depuis `pdm-test`, `aws s3 cp` (écriture, lecture) et URL présignée lue par `curl`. Attendu : 3 succès.
6. **G7-1, G7-3**, puis XP02 sur EKS (dont G7-2).
7. **G5** : tir k6 en paliers (montée 2 min, plateau 5 min, descente) ; script de mesure interrogeant toutes les 5 s l'HPA et les pods. Attendu : délai ≤ 90 s, p95 < 500 ms, < 1 % d'erreurs ; X7 bis relevé.
8. **G6** : tir constant, 3 réplicas, suppression forcée d'un pod au milieu de la fenêtre. Attendu : < 1 % d'erreurs, remplaçant `Ready` ≤ 60 s.
9. Densité : `kubectl top nodes/pods`, pods par nœud, `Pending` et `OOMKilled` (avec RabbitMQ chargé).
10. **G3** : `destroy` horodaté, puis inventaire MX7. Attendu : ≤ 20 min, inventaire vide hors zone, certificat et budget.

### C. Session EKS 2 (2026-10-31, 4 h)

1. **G2 n° 2** comme B.2.
2. XP03 complète (dont G7-4).
3. Si le temps le permet : délégation de préfixes (densité), **X14** (« Start Lab » relancé, `kubectl` rétabli).
4. **G9** : calcul sur les heures des sessions 1 et 2.
5. **G3** n° 2 et inventaire.

### D. Session ECS (2026-11-01, 2 h)

1. **G2** ECS : `apply` de la variante ECS jusqu'au HTTPS 200 ; **G1** relevé.
2. **G8** : depuis une tâche sans rôle, `curl http://169.254.170.2$AWS_CONTAINER_CREDENTIALS_RELATIVE_URI`. Attendu : variable absente, aucun identifiant.
3. **G4** depuis la tâche `pdm-test` (rôle `LabRole`).
4. **G7** : mêmes paires que G7-1 et G7-3, avec les groupes de sécurité par service.
5. **G5** : même profil k6 ; délai mesuré de la transition de l'alarme (`describe-alarm-history`) à `runningCount` augmenté et cible saine (interrogation toutes les 5 s).
6. **G6** : `aws ecs stop-task` sous tir constant, 3 tâches.
7. **G9**, **G10** (script de comptage sur les deux variantes).
8. **G3** : `destroy` et inventaire MX7.

### E. Réserve (2026-11-02)

Demi-session de rejeu d'un critère bloquant en échec corrigé, ou rejeu de XP02/XP03 sur ECS en cas de bascule.

## Durée et coût

- **Préparation** (DevOps) : IaC EKS et ALB 1,25 j ; variante ECS 0,75 j ; chart de test, script de mesure, inventaire 0,75 j.
- **Exécution** : 0,5 j local ; deux sessions EKS de 4 h (partagées avec XP02, XP03) ; une session ECS de 2 h.
- **Coût estimé** : ≈ 2,70 $ (EKS, deux sessions) + ≈ 1,25 $ (ECS). Coût constaté : G9, et `docs/mesures/couts.csv`.

## Données collectées

- **CSV** : `docs/mesures/XP01-orchestrateur.csv`, colonnes `horodatage_utc;xp;session;plateforme;critere;etape;valeur;unite;seuil;atteint;commentaire`. Une ligne par phase horodatée (`date +%s`) et par mesure.
- **Tir** : résumés k6 exportés en JSON dans `docs/mesures/XP01-orchestrateur/` (p95, taux d'erreur, débit), un fichier par tir et par plateforme.
- **Captures** : `kubectl get hpa,pods -w` pendant G5 et G6, configuration de l'add-on `vpc-cni`, console ECS pendant G5, inventaire final.
- **Journal de session** : identifiants de ressources seulement (MX5).

## Résultats

| # | EKS | ECS Fargate | Atteint EKS | Atteint ECS |
|---|---|---|---|---|
| G1 | | | | |
| G2 | | | | |
| G3 | | | | |
| G4 | | | | |
| G5 | | | | |
| G6 | | | | |
| G7 | | | | |
| G8 | | | | |
| G9 | | | | |
| G10 | | | | |

## Difficultés

À compléter par Martin pendant l'exécution, sans réécriture.

## Limites

- Charge de test synthétique, pas le code du POC.
- X8 et XC6 mesurés sur EKS seulement ; X9 (Cluster Autoscaler) et X11 reportés au sprint 2 ; X17, X18 (hors repli MS3), X20, X21 reportés en AC du sprint 1 (option B de D14).

## Conclusion

- **Relecture architecture** :
- **Relecture sécurité** (IAM, TLS) :
- **Verdict de Martin** :
- **Mise à jour de D03** :
