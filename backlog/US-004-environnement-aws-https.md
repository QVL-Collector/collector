# US-004 — Allumer, exposer en HTTPS et éteindre l'environnement AWS depuis zéro

| Champ | Valeur |
|---|---|
| Epic | E2 Plateforme AWS |
| Statut | À faire |
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

- **Choix EKS ou ECS (XP01)** : retirer la variante écartée de `infra/ephemere/` ; pour ECS, décrire tâches et services du front ; reporter durées et densité mesurées (`delegation_prefixes`, `imds_limite_sauts` si G4 échoue).
- **Couche persistante** : premier `plan` puis `apply` des imports (vérifier zéro remplacement) ; bucket des photos et groupes de journaux applicatif et d'audit (AC1, D09) ; `tools/bootstrap.sh`.
- **Preuve HTTPS (AC2, AC7, XP03)** : front nginx par `helm upgrade` ; HTTPS 200 en 45 min au plus ; rapport TLS (redirection 80, TLS 1.0/1.1 refusés, HSTS) archivé.
- **X17 (AC8) et X20 (AC9)** : script de contrôle après déploiement (balayage externe, `kubectl` hors plage, inventaire des load balancers et Services, liste blanche ME8, recherche des secrets dans `terraform show -json`, releases, journaux et images), rejoué par chaque `up` (AC11) ; ouverture de l'ALB (`alb_ouvert`) refusée tant qu'il n'est pas vert (AC10).
- **AC4 et AC5** : deux `up` et deux `down` chronométrés (X4, X5) ; affichage nœuds, pods et coût (`status.sh` de l'AC5).
