# US-004 — Allumer, exposer en HTTPS et éteindre l'environnement AWS depuis zéro

| Champ | Valeur |
|---|---|
| Epic | E2 Plateforme AWS |
| Statut | En cours |
| Sprint | 1 (J4) |
| Engagement | Ferme — **première US d'infrastructure** : X17 et X20 y sont des AC bloquants (D14) |
| Estimation | 8 points |
| Dépend de | US-001 ; verdicts de XP01 et XP03 (2026-11-02) |
| Fonction(s) | DevOps |
| Service | Plateforme |
| Critères grille | C5, C2, C8 |
| Exigences | ENF-02, ENF-28, ENF-29 |
| Branche | `feat/US-004-environnement-aws-https` |

## Récit

En tant que Lead Developer, je veux créer par une seule commande l'environnement de démonstration sur AWS, joignable en HTTPS sur `collector.qvl-project.com`, puis le détruire sans résidu, afin de démontrer le POC dans un environnement managé tout en tenant le budget de 50 $ du Learner Lab.

## Critères d'acceptation

### Fonctionnels

- **AC1** — Given le compte AWS sans couche persistante, When `tools/bootstrap.sh` s'exécute une fois, Then la zone Route 53 de `collector.qvl-project.com` (déléguée depuis Cloudflare), le certificat ACM `collector.qvl-project.com` et `*.collector.qvl-project.com`, le bucket des photos et les groupes de journaux existent, et un second lancement ne change rien.
- **AC2** — Given la couche persistante présente et plus de 75 minutes de session restantes, When `tools/up.sh` s'exécute, Then la couche éphémère (VPC, NAT, EKS et node group, RDS, ALB, alias DNS) est créée et une page du front (nginx) répond 200 en HTTPS sur `collector.qvl-project.com` en 45 minutes au plus (X4), chaque phase étant horodatée dans le journal de session.
- **AC3** — Given moins de 75 minutes de session restantes, When `tools/up.sh` s'exécute, Then il refuse de démarrer avec un message explicite.
- **AC4** — Given l'environnement allumé, When `tools/down.sh` s'exécute, Then la couche éphémère est détruite en 20 minutes au plus et l'inventaire strict (`tools/status.sh --strict`) ne trouve aucune ressource facturable hors couche persistante (X5) ; sinon son code retour est non nul et bloque le `up` suivant.
- **AC5** — Given l'environnement allumé, When `tools/status.sh` s'exécute, Then il affiche les ressources présentes, nœuds et pods, temps de session restant et coût estimé de la session.
- **AC6** — Given les variables de l'IaC, When une valeur hors plafond est saisie (plus de 4 nœuds, version EKS inférieure à 1.35), Then `terraform plan` échoue avant toute création (G4).

### Non fonctionnels

- **AC7 (Sécurité, BP1)** — Given l'environnement allumé, When le contrôle après déploiement s'exécute, Then une requête HTTP est redirigée vers HTTPS, TLS 1.0 et 1.1 sont refusés, le certificat est valide et l'en-tête HSTS est présent ; le rapport TLS est archivé.
- **AC8 (Sécurité, X17, bloquant)** — Given l'environnement allumé, When le balayage des adresses publiques du VPC, un `kubectl` depuis une adresse hors plage et l'inventaire des load balancers, Services et règles de l'ALB sont exécutés, Then seuls 443 et 80 (redirection) répondent, nœuds, base et API du cluster sont injoignables, il n'existe qu'un load balancer et aucun Service `LoadBalancer` ; tout chemin hors liste blanche répond 404 (XE4).
- **AC9 (Sécurité, X20, bloquant)** — Given l'environnement allumé, When les valeurs des secrets de la session sont recherchées dans l'état Terraform (`terraform show -json`), les releases Helm, les journaux du déploiement et de la CI et les images, Then aucune occurrence n'est trouvée, chaque pod ne monte que ses Secrets et aucune vulnérabilité critique corrigeable n'est relevée sur les images, modules et AMI.
- **AC10 (Sécurité)** — Given l'environnement allumé et AC8 ou AC9 non vert, When l'ouverture de l'ALB à Internet est demandée, Then elle est refusée : l'ALB reste restreint à l'adresse de Martin (ME5).
- **AC11 (Exploitation)** — Given chaque `up`, When il se termine, Then X17 et la recherche X20 sont rejouées automatiquement par le script de contrôle après déploiement, de sorte que chaque US suivante en hérite.

## Sécurité

| Exigence | Comment l'US la sert | AC |
|---|---|---|
| ES-01 | HTTPS seul, TLS 1.2 minimum, HSTS | AC7 |
| ES-07 | Exposition minimale vérifiée vue d'Internet, ALB restreint hors démonstration | AC8, AC10 |
| ES-06 | Aucun secret dans l'état, les releases, les journaux ni les images | AC9 |
| ES-05 | Analyse des images, modules et AMI | AC9 |

Mesures : MS1, MS2, MS6, MS7, MS9, ME1 à ME5, MI1 à MI4. Frontières : point d'entrée public, hébergement.

## Tests attendus

| Type | Outil | Ce qui est prouvé | Fonction |
|---|---|---|---|
| Après déploiement | Script de contrôle (curl, testssl.sh), inventaire | AC2, AC7, AC8, AC11 | DevOps |
| Sécurité négative | Balayage externe, `kubectl` hors plage, recherche de valeurs de secrets | AC8, AC9, AC10 | Test, DevOps |
| Configuration en code | Trivy `config`, `terraform validate` | AC6 | DevOps |
| Contrôle chronométré | Journal de session (deux `up` et deux `down`) | AC2, AC4 | DevOps |

## Hors périmètre

- Destruction programmée G1 et journal des coûts : US-005.
- Politiques réseau, admission, IMDS : US-006.
- Déploiement par Flux : US-007 (cette US déploie le front par `helm upgrade` depuis le poste).
- Keycloak et RabbitMQ : US-008 et US-011.

## Décisions liées

- [D03 — Hébergement et orchestration](../docs/decisions/D03-hebergement-orchestration.md) (X4, X5, X17, X20)
- [D08 — Exposition, TLS, domaine](../docs/decisions/D08-exposition-tls-domaine.md)
- [D11 — IaC, allumage et extinction](../docs/decisions/D11-iac-allumage-cout.md)
- [D14 — Expérimentations](../docs/decisions/D14-experimentations.md) (X17 et X20, AC bloquants de la première US d'infrastructure)

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
| Validation Martin | AAAA-MM-JJ |
| Livraison (PR / merge) | |

### Avancement au 2026-10-26 (avant les verdicts XP)

Fait sur la branche, sans aucune action AWS (`init -backend=false`, `fmt`, `validate`, `shellcheck` verts) :

- `infra/persistant/` : zone, certificat, CNAME de validation et budget repris par blocs `import` ; CAA `amazon.com` (ME7) ajouté ; `persistant.auto.tfvars` à renseigner avant le premier `plan`.
- `infra/ephemere/` : tronc commun promu de `sandbox/XP01-orchestrateur/` (réseau, ALB avec HSTS, DNS, RDS) ; orchestrateur paramétrable `eks` ou `ecs` ; IMDS à 2 sauts par défaut (D03) ; plafonds G4 (AC6).
- `tools/up.sh`, `down.sh`, `inventaire.sh`, `lib.sh` : G1 à G5, refus sous 75 min (AC3), journal de session `docs/mesures/sessions.csv`, rappel de finir `down.sh` avant « End Lab » (XP00 S8b).

Reste à faire après les verdicts du 2026-11-02 :

- ~~**Choix EKS ou ECS (XP01)**~~ : EKS retenue (verdict XP01 du 2026-11-20, D03 Confirmée), variante ECS retirée (voir l'avancement du 2026-11-20). Reste : reporter durées et densité mesurées (`delegation_prefixes`, `imds_limite_sauts` si G4 échoue).
- **Couche persistante** : premier `plan` puis `apply` des imports (vérifier zéro remplacement) ; bucket des photos et groupes de journaux applicatif et d'audit (AC1, D09) ; `tools/bootstrap.sh`.
- **Preuve HTTPS (AC2, AC7, XP03)** : front nginx par `helm upgrade` ; HTTPS 200 en 45 min au plus ; rapport TLS (redirection 80, TLS 1.0/1.1 refusés, HSTS) archivé.
- **X17 (AC8) et X20 (AC9)** : script de contrôle après déploiement (balayage externe, `kubectl` hors plage, inventaire des load balancers et Services, liste blanche ME8, recherche des secrets dans `terraform show -json`, releases, journaux et images), rejoué par chaque `up` (AC11) ; ouverture de l'ALB (`alb_ouvert`) refusée tant qu'il n'est pas vert (AC10).
- **AC4 et AC5** : deux `up` et deux `down` chronométrés (X4, X5) ; affichage nœuds, pods et coût (`status.sh` de l'AC5).

### Avancement au 2026-11-20 (après la session EKS 1 de XP01, sans action AWS)

Branche rebasée sur `main` sans conflit. Vérifications hors AWS : `terraform fmt -check` vert ; `init -backend=false` et `validate` verts sur `persistant` et `ephemere` ; `shellcheck --severity=warning` vert sur `tools/*.sh` ; `helm lint infra/front` vert ; Trivy `config infra` : aucun constat bloquant après les décisions du 2026-11-20 (voir plus bas).

Correctifs du bac à sable XP01 et constats de la session 1 reportés :

- Planifications G1 : champs en casse attendue par EventBridge Scheduler (`ClusterName`, `NodegroupName`, `DbInstanceIdentifier`, `Name`).
- `lib.sh` : chemins Windows pour Terraform (`pwd -W`) ; `jq` enveloppé pour retirer le retour chariot de la version Windows (toutes les lectures, pas seulement `down.sh`).
- Descriptions de règles de groupes de sécurité sans apostrophe (`eks.tf`).
- Politique de sortie sans `169.254.169.254/32` dans chaque namespace applicatif, `default` compris (`infra/k8s/socle.yaml`, appliqué par `up.sh`), vérifiée par le contrôle après déploiement.
- Descente de l'HPA en 60 s dans les valeurs AWS du front (`infra/front/valeurs-aws.yaml`, X7 bis).
- `publicAccessCidrs` au seul /32 du poste : déjà dans `eks.tf`, désormais vérifié à chaque contrôle ; « `kubectl` hors plage » joué par une sonde Lambda éphémère (CloudShell absent du lab).
- `up.sh --fin-lab HH:MM` : heure de fin lue sur le compteur du lab (X14 : le lab s'est arrêté avant l'échéance estimée).

Livré pour les AC :

- **AC1** : `tools/bootstrap.sh` (bucket des photos en CLI, puis `infra/persistant` ; plan refusé s'il détruit ou remplace ; second lancement « aucun changement ») ; `infra/persistant/photos.tf` (accès public bloqué, propriétaire imposé, SSE-S3, refus hors TLS, dépôts inachevés purgés) et `journaux.tf` (`/collector/app` 7 j, `/collector/audit` 30 j).
- **AC4, AC5** : `tools/status.sh` (session, temps restant, coût de la session et cumul, couche persistante, ressources, nœuds, pods, HPA, dernier verdict) ; `--strict` = inventaire strict + dérive de la couche persistante, appelé par `up.sh` (état vide) et `down.sh`.
- **AC7 à AC9, AC11** : `tools/controle-apres-deploiement.sh`, appelé à la fin de `up.sh` (code retour non nul si rouge), rapports masqués archivés dans `docs/mesures/US-004/` (`tls-<session>.txt`, `controle-<session>.txt`).
- **AC10** : `tools/alb.sh ouvrir|fermer` (verdict vert exigé pour la session et la série d'état courantes, contrôle rejoué une fois ouvert, fermeture s'il rougit) ; précondition Terraform `alb_ouvert` ⇒ `preuve_controle == session` ; `up.sh` force `alb_ouvert=false` par `-var`.
- **AC2** : chart `infra/front` (nginx non root, NodePort 30080, HPA 2 à 6), installé par défaut par `up.sh` dans le namespace `front`.

Variante ECS retirée (EKS confirmée par le verdict XP01 du 2026-11-20, D03 « Confirmée ») : `infra/ephemere/ecs.tf` supprimé ; variables `orchestrateur` et `port_front`, `local.eks`/`local.ecs` et `port_cible` supprimés, ressources EKS sans `count` ni index ; groupe cible de l'ALB en `instance` sur le NodePort ; sorties `orchestrateur`, `nom_cluster_ecs` et `groupe_securite_front_ecs` supprimées. Outils : `up.sh --fin-lab HH:MM` sans argument d'orchestrateur, chemin EKS seul (kubectl, Helm, version EKS toujours contrôlés) ; `down.sh` retire toujours les charges du cluster ; coût horaire unique `COUT_HORAIRE_USD` (0,34 $/h) dans `lib.sh` ; colonne `orchestrateur` retirée de `sessions.csv` et `couts.csv` (aucune ligne existante). L'inventaire garde `q_ecs` (orphelins). Vérifications : `fmt -check`, `validate` (`ephemere`, `persistant`), `shellcheck --severity=warning` verts ; Trivy : AWS-0104 disparu.

Trivy `config` : constats bloquants traités (décisions de Martin du 2026-11-20) :

- AWS-0053 (ALB public) et AWS-0040 (API EKS publique) : exceptions dans `.trivyignore` (échéance 2026-12-20), voulues, restreintes au /32 (ME5, MS1), MX3 vérifié hors plage, ni VPN ni bastion dans le Learner Lab ; tracées dans D08 (Conséquences). Validation Sécurité à confirmer.
- AWS-0039 (chiffrement des Secrets EKS) : clé KMS persistante `alias/collector-secrets-eks` (`infra/persistant/cle-secrets-eks.tf` : rotation activée, fenêtre de suppression 7 j, politique limitée au compte et à `LabRole`), exposée en sortie `cle_secrets_eks` et lue en data par `infra/ephemere` (`encryption_config` sur `secrets`). **À tester à la prochaine session (`bootstrap.sh`)** ; si le lab refuse KMS (création de la clé ou octroi à EKS), repli en exception documentée dans `.trivyignore`.

Reste à faire en session AWS :

1. `bootstrap.sh` deux fois (AC1) : premier `apply` sans remplacement, second sans changement ; délégation NS dans Cloudflare si non constatée.
2. Deux `up.sh --fin-lab HH:MM` et deux `down.sh` chronométrés (AC2 ≤ 45 min, AC4 ≤ 20 min), `status.sh` allumé (AC5), `up.sh` sous 75 min refusé (AC3), `terraform plan` avec `noeuds_max = 5` puis `version_eks = "1.34"` en échec (AC6).
3. Premier contrôle réel : création de la sonde Lambda sous `LabRole` (runtime python3.13), poignées TLS 1.0/1.1, HSTS sur la 404 fixe, réponses de nginx aux chemins encodés, analyse Trivy des images, AMI recommandée, lecture des journaux de la CI par `gh`.
4. `alb.sh ouvrir` refusé sur verdict rouge, accepté sur verdict vert, puis `alb.sh fermer` (AC10) ; ME4 et ME8 avant toute démonstration publique.
5. Remplacer l'image nginx de `valeurs-aws.yaml` par l'image du front publiée par la CI (D10), épinglée par empreinte.
6. Après la session 2 de XP01 : `delegation_prefixes` selon la densité mesurée.

### Session AWS du 2026-11-20 (couche réelle)

| AC | Résultat |
|---|---|
| AC1 | `bootstrap.sh` : zone, certificat `ISSUED`, bucket photos, journaux, clé KMS des Secrets EKS ; second lancement sans changement ; délégation Cloudflare constatée |
| AC2 | HTTPS 200 en 16 min (1 mesure sur 2) |
| AC3 | Refus à 55 min restantes |
| AC4 | `down.sh` : destruction en 9 min, inventaire strict vide (1 mesure sur 2) ; session 58 min ≈ 0,33 $ |
| AC5 | `status.sh` : ressources, nœuds, pods, temps restant, coût estimé |
| AC6 | `terraform plan` refusé par la validation des variables avec `noeuds_max=5` et `version_eks=1.34` ; plan témoin valide |
| AC7 | Redirection 301, TLS 1.0/1.1 refusés, 1.2 accepté, certificat valide, HSTS (contrôle corrigé pour Git Bash : `/dev/null` et `/tmp` natifs, CRLF de `curl`, `grep -i`) |
| AC8 | X17 vert ; XE4 : 400 émis par l'ALB accepté comme refus (décision du 2026-11-20, XP03) |
| AC9 | Rouge : image nginx provisoire (1.31.7-alpine, 13 vulnérabilités corrigeables). **Décision du 2026-11-20** : le front déployé sera l'image construite, analysée et publiée par la CI (US-007, publication GHCR signée) ; AC9 se rejoue avec elle |
| AC10 | Ouverture de l'ALB refusée sur contrôle rouge |
| AC11 | Contrôle rejoué par `up.sh` |

Reste : AC2 et AC4 (seconde mesure), AC9 avec l'image de la CI.
