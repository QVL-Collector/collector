# D07 — Données (SGBD, stockage des photos)

| Champ | Valeur |
|---|---|
| Statut | Décidée |
| Date | 2026-10-26 |
| Décideur | Martin |
| Dépend de | [D02](D02-decoupage-applicatif.md), [D03](D03-hebergement-orchestration.md), [D04](D04-stack.md) |
| Impacte | D05, D06, D08, D10, D11, D13, D14, [architecture-technique.md](../architecture/architecture-technique.md) (§ 3, § 4.3), [exigences-securite.md](../securite/exigences-securite.md) (FC8, ES-14, ES-17, ES-30) |
| Critères grille | C2 (principal), C8, C5 |

## Contexte et besoin

D07 fixe où et comment les services stockent leurs données et les photos. Ce qui est déjà tranché n'est pas rediscuté :

- **D02** : trois services possèdent des données (Place de marché, Transactions, Interactions) ; le Contrôle n'en a aucune. Un accès direct d'un service aux données d'un autre est refusé par la base. La disponibilité de l'article appartient à la Place de marché : réservation atomique avec échéance, une seule réservation active par acheteur, un nouvel essai renvoie la réservation existante, une seule commande par réservation côté Transactions. La décision de l'admin porte la révision examinée ; refus 409 si elle est périmée.
- **D03** : RDS PostgreSQL db.t3.micro Single-AZ, hors du cluster, en sous-réseaux privés, dans la couche éphémère ; bucket S3 créé hors du provider Terraform ; accès des pods à S3 par le rôle du nœud (X6), IMDS réservé à la Place de marché (MS3) ; mesures MS6 (secrets) et MS7 (RDS chiffré, non public, `rds.force_ssl` ; bucket privé, chiffré, TLS imposé).
- **D04** : Go, `pgx` v5, migrations `goose` v3 embarquées dans le binaire, compte de migration distinct du compte d'exécution. D04 renvoie à D07 un piège : « une réservation active par acheteur » ne s'écrit pas en index unique partiel (`now()` interdit).
- **D06** (à instruire) : si le serveur d'autorisation retenu est un conteneur, il demande son propre espace dans RDS (D03, Conséquences).

### Exigences

| Exigence | Ce qu'elle impose à D07 |
|---|---|
| ES-07, FC8 | Base et stockage privés, chiffrement au repos, un compte d'accès par service limité à son espace |
| ES-38, M-MOD-01 | Aucun accès croisé entre services, vérifié par la base |
| ES-14, M-RES-01, M-RES-02, M-PAY-05 | Réservation atomique, échéance fixée et plafonnée par la Place de marché, une seule active par acheteur, nouvel essai sans seconde réservation, une commande par réservation |
| ES-18, M-ANN-05, M-CAN-02 | Décision et résultat de contrôle portant la révision ; résultat périmé ignoré, décision périmée refusée (409) |
| ES-17, M-ANN-01, 02, 06, 07 | Upload sécurisé : contenu vérifié, taille et dimensions bornées, ré-encodage sans métadonnées, nom généré, photo publique seulement après publication |
| ES-30, ENF-13 | Sauvegardes chiffrées et restauration testée : cible V1. Déjà jugée non démontrable dans le POC par l'avis sécurité de D03 |
| ENF-12, ES-31 | Aucun événement perdu : l'émission fiable des événements touche la base de l'émetteur (lien D05) |
| ENF-28, ENF-29 | Environnement recréé par commande ; même artefact en local et sur AWS |

### Contraintes de l'environnement

Source : [environnement-learner-lab.md](../00-cadrage/environnement-learner-lab.md).

| Fait | Conséquence | Niveau |
|---|---|---|
| RDS PostgreSQL db.t3.micro créé en 10 à 16 min ; Multi-AZ refusé | Une instance Single-AZ, créée en parallèle du cluster (15 à 20 min) : hors du chemin critique de l'allumage | [REX] ; Multi-AZ [?] |
| db.t3.micro ≈ 0,018 $/h ; stockage gp2 ≈ 0,115 $/Go-mois, 20 Go minimum | ≈ 0,085 $ par session de 4 h, ≈ 2,1 $ sur 25 sessions | [DOC], prix à vérifier |
| Une instance RDS arrêtée redémarre seule au bout de 7 jours ; le lab peut ne pas l'arrêter | Ne jamais compter sur l'arrêt : détruire ou payer | [DOC] |
| SCP S3 : `aws_s3_bucket` de Terraform échoue | Bucket créé en CLI, puis configuré par les ressources `aws_s3_bucket_*` | [REX] |
| Pas d'IRSA ni de Pod Identity ; pas de pilote EBS CSI | S3 par le rôle du nœud (`LabRole`) ; aucune base sur volume persistant dans le cluster | [REX] |
| Budget épuisé : compte désactivé, tout effacé | Aucune donnée ne doit être irremplaçable dans AWS | [DOC] |

## Options

Cinq sous-questions, chacune avec ses options et sa recommandation.

### Q1 — SGBD et persistance entre sessions

#### Option S1 — Une instance RDS PostgreSQL, une base et deux comptes par service

- **Description** : une instance db.t3.micro Single-AZ ; dans l'instance, une base logique par service (`marche`, `transactions`, `interactions`, plus une pour le serveur d'autorisation si D06 la demande).
- **Avantages** : managée (sauvegardes, correctifs, chiffrement) ; coût minimal ; une seule ressource à créer, détruire et inventorier ; isolement assuré par la base elle-même (Q2).
- **Inconvénients** : point unique de défaillance pour les trois services ; ressources partagées (une requête lourde d'un service ralentit les autres).
- **Coût AWS et effort** : ≈ 0,085 $ par session, ≈ 2,1 $ sur le projet ; ≈ 0,5 j (IaC et script d'amorçage).
- **Complexité ajoutée** : faible.
- **Couverture grille** : C2 (propriété des données démontrée par la base) ; C8 (FC8, MS7).

#### Option S2 — Une instance RDS par service

- **Description** : trois ou quatre instances db.t3.micro.
- **Avantages** : isolement physique ; panne ou saturation d'une base sans effet sur les autres ; proche d'une cible V1 à fort trafic.
- **Inconvénients** : trois à quatre fois les ressources à créer, inventorier et détruire ; plus de secrets maîtres ; aucun gain démontrable dans le POC, où la charge porte sur la consultation (Place de marché seule).
- **Coût AWS et effort** : ≈ 0,26 à 0,34 $ par session, ≈ 6,5 à 8,5 $ sur le projet ; quota d'instances RDS du lab inconnu [?].
- **Complexité ajoutée** : moyenne, sans besoin présent.
- **Couverture grille** : C2 identique à S1 : l'isolement logique suffit à prouver la propriété des données.

#### Option S3 — PostgreSQL en conteneur dans EKS

- **Description** : un StatefulSet PostgreSQL dans le cluster.
- **Avantages** : aucun coût RDS ; identique à l'environnement local.
- **Inconvénients** : sans pilote EBS CSI, les données disparaissent si le pod est replanifié, y compris pendant la démonstration de perte d'un nœud ; base non managée (sauvegardes, correctifs, TLS à la charge de Martin) ; mémoire prise sur les nœuds ; recul sur C5 (« environnement managé »).
- **Coût AWS et effort** : économie ≈ 0,085 $ par session ; +1 à 2 j d'exploitation.
- **Complexité ajoutée** : élevée pour un gain de quelques centimes.
- **Couverture grille** : affaiblit C5 et C8.

#### Persistance entre sessions

| Option | Principe | Coût entre sessions | Effet sur l'allumage et l'extinction | Risques |
|---|---|---|---|---|
| **P1 — Recréée vide, jeu de démonstration rejoué** | La base est détruite par `down` (`skip_final_snapshot`) et recréée par `up` ; migrations puis amorçage des données fictives (D02 : « données fictives rejouables ») | 0 $ | Allumage inchangé (RDS en parallèle du cluster) ; amorçage en quelques secondes | Aucune donnée de session conservée : sans enjeu, tout est fictif |
| **P2 — Instantané final, restauré à l'allumage** | `down` prend un instantané ; `up` restaure depuis le dernier | Quelques centimes (instantané de quelques Mo) | `down` +3 à 5 min [?] ; restauration d'une durée proche d'une création [?] | Instantané interrompu par l'expiration de la session ; identifiant d'instantané à gérer dans l'état Terraform ; instantanés orphelins à inventorier |
| **P3 — Instance arrêtée et conservée** | Arrêt en fin de session, redémarrage à la suivante | ≈ 2,3 $/mois de stockage, plus ≈ 0,43 $/jour quand elle redémarre seule après 7 jours | Allumage plus court de quelques minutes, sans effet sur le chemin critique | Impose de sortir le VPC et les sous-réseaux de la couche éphémère de D03 ; facturation silencieuse |

### Q2 — Isolation par service et migrations

#### Option I1 — Une base logique par service, deux rôles par service

- **Description** :
  - par service : une base, un rôle de migration propriétaire du schéma (DDL), un rôle d'exécution limité au DML sur ses tables ;
  - `REVOKE CONNECT ON DATABASE … FROM PUBLIC` sur chaque base : un rôle ne peut même pas se connecter à la base d'un autre service ;
  - un script d'amorçage idempotent (`psql`, compte maître) crée bases, rôles et extensions ; mots de passe générés à l'allumage, rangés dans Secrets Manager, livrés à chaque service seulement (MS6) ;
  - connexions en `sslmode=verify-full` avec le certificat d'autorité de RDS.
- **Avantages** : refus d'accès croisé garanti par PostgreSQL dès la connexion, simple à prouver par un test négatif (« le rôle de Transactions ne se connecte pas à `marche` ») ; aucune requête inter-bases possible, donc aucune tentation de jointure entre services.
- **Inconvénients** : un pool de connexions par base (sans effet : chaque service n'en ouvre qu'une).
- **Complexité ajoutée** : faible.

#### Option I2 — Une base commune, un schéma et un rôle par service

- **Description** : une seule base, un schéma par service, droits `USAGE` limités au schéma du service.
- **Avantages** : une seule base à créer ; jointures techniquement possibles pour une analyse ponctuelle.
- **Inconvénients** : l'isolement repose sur des droits plus fins (schéma `public`, droits par défaut, fonctions), donc plus faciles à mal régler (M-LIV-03) ; la possibilité de jointure entre services est justement ce que D02 interdit.
- **Complexité ajoutée** : faible, mais isolement plus fragile.

#### Migrations

- **Outil** : `goose` v3, déjà retenu par D04 ; migrations SQL embarquées, sous-commande `migrate` du même binaire (ENF-29). `golang-migrate` conviendrait aussi ; aucun motif de rouvrir D04.
- **Lancement** : un Job Kubernetes par service, crochet Helm `pre-install` et `pre-upgrade`, sous le rôle de migration. Une seule exécution par déploiement, contrairement à un conteneur d'initialisation lancé par chaque réplica ; le déploiement s'arrête si la migration échoue.
- **Règle** : migrations compatibles avec la version précédente du service (ajout d'abord, suppression à la version suivante), pour que la mise à jour progressive et le retour arrière (ENF-22) restent sûrs.

### Q3 — Réservation atomique et verrouillage optimiste

Données de la Place de marché. Seule l'horloge de la base fait foi (`now()`), jamais celle d'un service.

#### Option R1 — Disponibilité portée par la ligne de l'article, verrou consultatif par acheteur

- **Description** :
  - colonnes de l'article : `reservation_id`, `reserve_par`, `reserve_jusqu_a`, `vendu` ; une table `reservation` journalise chaque réservation (traçabilité, ENF-08) ;
  - « réserver », en une transaction :
    1. `pg_advisory_xact_lock` sur l'acheteur : sérialise les réservations d'un même acheteur ;
    2. une réservation active de cet acheteur sur un autre article → refus (409) ;
    3. mise à jour conditionnelle :
       ```sql
       UPDATE article
          SET reservation_id = CASE WHEN reserve_par = $acheteur AND reserve_jusqu_a > now()
                                    THEN reservation_id ELSE gen_random_uuid() END,
              reserve_jusqu_a = CASE WHEN reserve_par = $acheteur AND reserve_jusqu_a > now()
                                     THEN reserve_jusqu_a ELSE now() + $duree_plafonnee END,
              reserve_par = $acheteur
        WHERE id = $article AND statut = 'publie' AND NOT vendu
          AND (reserve_jusqu_a IS NULL OR reserve_jusqu_a <= now() OR reserve_par = $acheteur)
       RETURNING reservation_id, reserve_jusqu_a, prix, frais_port;
       ```
       Aucune ligne → article indisponible (409). Le nouvel essai du même acheteur renvoie la même réservation et la même échéance ;
  - modification ou retrait par le vendeur : `UPDATE … WHERE id = $article AND (reserve_jusqu_a IS NULL OR reserve_jusqu_a <= now())`. Le verrou de ligne de PostgreSQL ordonne ces deux écritures et relit la condition : aucune fenêtre de course ;
  - expiration : aucun traitement planifié ; « réservé » se lit `reserve_jusqu_a > now()`.
- **Avantages** : tout tient sur une ligne, en mode d'isolement par défaut ; une seule notion nouvelle (le verrou consultatif) ; le test de concurrence de D13 le prouve contre la vraie base.
- **Inconvénients** : la règle « une par acheteur » repose sur le code (verrou), pas sur une contrainte déclarée.
- **Complexité ajoutée** : faible.

#### Option R2 — Table de réservations avec contraintes d'exclusion

- **Description** : `reservation(article, acheteur, periode tstzrange)` avec deux contraintes `EXCLUDE USING gist` (`article WITH =, periode WITH &&` et `acheteur WITH =, periode WITH &&`), extension `btree_gist`. Lève directement le piège de D04.
- **Avantages** : les deux règles sont déclarées et garanties par la base.
- **Inconvénients** : le refus de modification pendant la réservation porte sur une autre table que l'article, donc verrou explicite de l'article (`FOR UPDATE`) dans les deux parcours ; fin anticipée d'une réservation à gérer en raccourcissant la période ; extension créée par le compte maître ; notion peu courante à défendre et à maintenir.
- **Complexité ajoutée** : moyenne.

#### Option R3 — Isolement `SERIALIZABLE` et nouvel essai automatique

- **Description** : requêtes simples, conflit détecté par PostgreSQL (erreur 40001), transaction rejouée.
- **Inconvénients** : boucle de rejeu à écrire et à tester partout où l'on écrit ; comportement sous charge moins prévisible ; correction moins lisible pour un relecteur.
- **Complexité ajoutée** : moyenne.

#### Idempotence et verrouillage optimiste (communs à R1, R2, R3)

- **Une commande par réservation** (Transactions) : contrainte `UNIQUE (reservation_id)` sur la commande ; `INSERT … ON CONFLICT (reservation_id) DO NOTHING`, puis lecture de la commande existante. Le double clic donne la même réservation, donc la même commande. Aucune clé d'idempotence supplémentaire n'est nécessaire : la paire (article, acheteur) en tient lieu.
- **Décision de l'admin** : `UPDATE article SET statut = $decision … WHERE id = $article AND revision = $revision_examinee AND statut = 'a_verifier'`. Aucune ligne et article existant → 409. Même garde pour appliquer un résultat de contrôle (révision courante et état « en contrôle »), ce qui rend le consommateur idempotent (M-CAN-02). Toute modification incrémente `revision`.

### Q4 — Photos

#### Option F1 — Dépôt par l'API de la Place de marché, stockage privé dans S3

- **Description** :
  - le front envoie la photo à l'API vendeur (multipart, corps borné par `http.MaxBytesReader`, nombre de photos plafonné) ;
  - la Place de marché vérifie le contenu (JPEG ou PNG, dimensions lues avant décodage, plafond de pixels), ré-encode en JPEG sans métadonnées, génère un nom aléatoire et écrit l'objet dans le bucket privé. L'original n'est jamais stocké ;
  - lecture : pour un article publié seulement, la Place de marché émet des URL présignées de lecture à courte durée (10 min par exemple) ; l'image vient alors directement de S3, origine distincte et sans cookie ; le tir de charge ne fait pas transiter les fichiers par le service (point laissé ouvert par D01) ;
  - bucket : Block Public Access, chiffrement SSE-S3, politique refusant les requêtes sans TLS, aucune règle CORS (le navigateur n'écrit jamais dans S3).
- **Avantages** : un seul chemin d'écriture, déjà prévu par D04 et par la séquence de l'architecture (§ 4.3) ; le ré-encodage neutralise le fichier piégé mieux qu'un antivirus, puisque seuls des pixels recodés sont conservés ; aucune nouvelle surface publique en écriture.
- **Inconvénients** : le trafic des dépôts traverse les pods (quelques Mo par photo, rare face à la consultation) ; une URL émise reste valable jusqu'à son expiration, même si l'article est retiré entre-temps (durée courte pour borner l'effet).
- **Coût AWS et effort** : stockage négligeable ; ≈ 1 j (dépôt, ré-encodage, présignature).
- **Complexité ajoutée** : faible.

#### Option F2 — Dépôt direct dans S3 par URL présignée, quarantaine puis traitement

- **Description** : la Place de marché émet une autorisation de dépôt (POST présigné avec type et `content-length-range`) vers un préfixe `quarantaine/` ; le front confirme le dépôt ; la Place de marché lit l'objet, le vérifie, le ré-encode vers `photos/` et supprime l'original ; règle de cycle de vie qui purge la quarantaine.
- **Avantages** : décharge les pods de l'envoi ; schéma courant à grande échelle.
- **Inconvénients** : le ré-encodage oblige de toute façon la Place de marché à relire chaque octet, donc le gain est faible ; nouvelle surface publique en écriture (S3 joignable par le navigateur, CORS) ; un état intermédiaire de plus (déposée, non traitée) et des orphelins à purger ; limitation de débit des dépôts (ES-29) plus difficile.
- **Complexité ajoutée** : moyenne, sans besoin présent.

#### Option F3 — Photos dans la base

- **Description** : colonne `bytea`.
- **Inconvénients** : base gonflée, sauvegardes alourdies ; chaque affichage traverse la Place de marché et la base, ce qui fausse le tir de charge sur la consultation (D01) ; pas d'origine distincte.
- **Complexité ajoutée** : faible au départ, coûteuse à l'usage. Écartée.

**Analyse antivirus** (ClamAV en conteneur) : non retenue par défaut. Pour des images ré-encodées, elle n'ajoute pas de protection mesurable, et coûte environ 1 Go de mémoire et des mises à jour de signatures. Soumis à l'avis sécurité.

**Accès des pods** : par le rôle du nœud (`LabRole`) via IMDSv2, réservé à la Place de marché (D03, MS3, X6). Aucun identifiant statique.

**Persistance du bucket** : bucket dans la couche persistante (créé une fois en CLI) ; `down` le vide, car la base repart vide (P1) et des photos sans article seraient des orphelines ; l'amorçage redépose les photos de test par le même chemin de dépôt, ce qui le teste à chaque session.

### Q5 — Outbox (lien D05, broker non tranché)

- **Description** : table `outbox` dans la base de l'émetteur (identifiant, sujet, clé, charge JSON, date de création, date de publication). L'événement est inséré dans la même transaction que le changement d'état (soumission, décision de publication). Un relais du même binaire lit les lignes non publiées (`FOR UPDATE SKIP LOCKED`, plusieurs réplicas sans doublon), publie avec confirmation du broker, puis marque la ligne. Purge des lignes publiées après une durée bornée (M-CAN-04).
- **Concerné** : la Place de marché (demandes de contrôle, décisions de publication). Le Contrôle n'a pas de base : il n'accuse réception d'une demande qu'après la confirmation de publication de son résultat. Transactions n'utilise pas le canal dans le POC.
- **Garantie** : au moins une fois. Les consommateurs restent idempotents (garde de révision ; identifiant de message unique côté Interactions).
- **Limite** : l'outbox ne couvre pas la perte, après confirmation, des messages d'un broker sans volume persistant (D03). Le renvoi des articles restés « en contrôle » au-delà d'un délai, ou un volume persistant, reste à trancher en D05.
- **Alternative** : publication après validation de la transaction, avec un balayage de rattrapage. Plus simple pour les demandes de contrôle, insuffisante pour les décisions diffusées. D05 tranche.

### Synthèse

| Sous-question | Options | Recommandée | Coût / effort |
|---|---|---|---|
| Q1 SGBD | S1 une instance RDS ; S2 une instance par service ; S3 PostgreSQL dans EKS | S1 | ≈ 0,085 $ par session ; 0,5 j |
| Q1 Persistance | P1 recréée et amorcée ; P2 instantané restauré ; P3 instance arrêtée | P1 | 0 $ entre sessions |
| Q2 Isolation | I1 base et rôles par service ; I2 schéma par service | I1, `goose` en Job Helm | ≈ 0,5 j |
| Q3 Réservation | R1 ligne de l'article et verrou par acheteur ; R2 contraintes d'exclusion ; R3 `SERIALIZABLE` | R1, plus `UNIQUE (reservation_id)` et garde de révision | Inclus dans les US |
| Q4 Photos | F1 dépôt par l'API ; F2 dépôt direct en quarantaine ; F3 en base | F1, lecture par URL présignée | ≈ 1 j |
| Q5 Outbox | Table outbox et relais ; publication après validation et rattrapage | Outbox, sous réserve de D05 | ≈ 1 j |

## Recommandation

**S1 + P1 + I1 + R1 + F1, outbox dans la base de l'émetteur, sous réserve de D05.**

1. **La base garantit elle-même la propriété des données** : une base logique et deux rôles par service. Le refus d'accès croisé se prouve par un test (ES-38, M-MOD-01), sans payer trois instances.
2. **La réservation tient sur une ligne** : une mise à jour conditionnelle suffit pour l'atomicité, le nouvel essai et le refus de modification ; un verrou par acheteur suffit pour la limite d'une réservation active. C'est la variante la plus courte à écrire et à expliquer. La contrainte d'unicité côté Transactions garantit une commande par réservation.
3. **Les photos ne prennent qu'un chemin**, par l'API qui les ré-encode : pas de nouvelle surface publique en écriture. La lecture en URL présignée sort les fichiers du tir de charge.
4. **Aucune donnée n'a de valeur entre deux sessions** : recréer et amorcer coûte 0 $, n'allonge pas l'allumage et applique ENF-28 à chaque session.

**Pourquoi pas S2 ou S3.** S2 triple la facture et l'exploitation sans rien prouver de plus pour C2. S3 économise quelques centimes et perd ses données à chaque replanification de pod.

**Pourquoi pas P2.** Restaurer un instantané démontrerait ES-30, mais l'avis sécurité de D03 l'a déjà jugée non démontrable : `LabRole` peut supprimer l'instantané. P2 allonge l'extinction et ajoute un état à gérer. Si Martin veut une preuve de restauration, un exercice unique et manuel, documenté, suffit (point soumis).

**Volontairement exclu** : réplica de lecture, cache, recherche plein texte externe, CDC, antivirus, CORS sur le bucket. Chaque ajout devra nommer le besoin qu'il sert.

### Points soumis aux avis

- **Sécurité** :
  - I1 : refus d'accès croisé à la connexion ; compte maître utilisé par le seul amorçage ; mots de passe générés à chaque allumage ;
  - F1 : une analyse antivirus est-elle exigée en plus du ré-encodage ? Durée des URL présignées de lecture ; S3 ne permet pas d'imposer `X-Content-Type-Options: nosniff` sur une URL présignée : risque résiduel acceptable pour des JPEG ré-encodés ?
  - accès à S3 par `LabRole` partagé : écart ES-38 déjà documenté ; politique de bucket limitée au préfixe `photos/` sans effet, le rôle étant commun ;
  - décodage des images dans le processus de la Place de marché (POC), le processus isolé d'ES-17 restant documenté pour la V1 ;
  - ES-30 : confirmer qu'elle reste documentée pour la V1, ou exiger un exercice de restauration unique.
- **Faisabilité** :
  - R1 : test de concurrence contre la vraie base (`testcontainers-go`) ; verrou consultatif et mode d'isolement par défaut ;
  - Job de migration en crochet Helm, ordre avec l'amorçage des bases ;
  - vidage du bucket dans `down`, amorçage des photos de test dans `up` ;
  - équivalent local de S3 : vérifier l'état de la distribution de MinIO, ou choisir un autre stockage compatible [?].

## Avis sécurité

Fonction : Sécurité — 2026-10-26. Références : [exigences-securite.md](../securite/exigences-securite.md) (FC8, ES-03, ES-04, ES-06, ES-07, ES-17, ES-29, ES-30, ES-38, §5.6 ; M-ANN-01, 02, 06, 07, M-DIS-05, M-LIV-03, 07, M-MOD-01, 03) ; avis sécurité de [D03](D03-hebergement-orchestration.md) (MS3, MS6, MS7, X16) et de [D04](D04-stack.md) (images, `CGO_ENABLED=0`). Avis sur dossier ; les points [?] relèvent des sondes de D14.

**Synthèse.** Avis favorable à S1 + P1 + I1 + R1 + F1 et à l'outbox, sous les mesures MD1 à MD10. Aucune option écartée n'apporte de gain de sécurité démontrable dans le POC. Le risque dominant n'est pas dans D07 : c'est la chaîne « exécution de code dans la Place de marché → IMDS → `LabRole` → tous les secrets, donc toutes les bases » (M-MOD-03), que D07 ne peut que borner.

**1. Points soumis**

- **I1 (bases et rôles)** : favorable. `REVOKE CONNECT` plus un rôle par base donne un refus vérifiable par la base, conforme à ES-38 et M-MOD-01. I2 est plus fragile (droits par défaut, schéma `public`) : écarté à raison. Trois compléments, faute desquels l'isolement est plus faible qu'annoncé :
  - le rôle d'exécution n'est propriétaire d'aucun objet et n'a aucun attribut (`NOSUPERUSER NOCREATEDB NOCREATEROLE`) ; ses droits passent par `ALTER DEFAULT PRIVILEGES FOR ROLE <migration>` ; `REVOKE CREATE ON SCHEMA public FROM PUBLIC` posé explicitement, la version de PostgreSQL n'étant pas figée ;
  - le secret du rôle de migration n'est livré qu'au Job de migration, jamais au Deployment : sinon une compromission du service donne le DDL de sa base ;
  - le lieu d'exécution de l'amorçage n'est pas fixé (le poste de Martin ne joint pas une base privée). S'il tourne en Job dans le cluster, le mot de passe maître y entre : Secret supprimé à la fin du Job, jamais en valeur Helm (MS6).
- **Antivirus en plus du ré-encodage** : non exigé. ES-17 ne le demande pas. L'original n'est ni stocké ni servi ; seuls des pixels recodés par l'encodeur de Go sont conservés, ce qui neutralise contenu exécutable, polyglotte et métadonnées mieux qu'une base de signatures. ClamAV analyserait l'original avant décodage, sans protéger le décodeur lui-même (le vrai vecteur, M-ANN-01), pour ≈ 1 Go de mémoire et une dépendance de plus à tenir (M-LIV-01). À réévaluer seulement si un jour un fichier est conservé ou servi tel quel (pièce jointe, document).
- **Durée des URL présignées** : 10 min accepté, plafond 15 min. Une URL émise reste valable après retrait de l'article : effet borné à cette durée, sur une image déjà publique, acceptable. Signature avec les identifiants temporaires du rôle : l'URL expire au plus tôt des deux échéances (sans effet de sécurité, à connaître pour les tests). Les URL complètes ne sont jamais journalisées (M-LIV-07) : la signature y figure.
- **Point manquant, contrôle d'accès à l'émission (ES-03, ES-04, M-ANN-06)** : « URL émise pour un article publié seulement » oublie deux lecteurs légitimes avant publication, le vendeur propriétaire et l'admin qui examine l'article « à vérifier ». L'émission pour un article non publié doit vérifier côté serveur que l'appelant est le vendeur de l'article ou un admin, sur une route distincte du catalogue, avec test négatif « B n'obtient pas l'URL d'une photo de l'article non publié de A ».
- **`nosniff` impossible sur URL présignée** : risque résiduel faible, accepté, à documenter. Justification : l'objet est écrit avec `Content-Type: image/jpeg` fixé par le serveur (jamais repris de la requête) ; son contenu est un JPEG produit par l'encodeur, sans HTML possible ; il est servi depuis l'origine du bucket, distincte de la plateforme et sans cookie, donc un contenu interprété n'y volerait aucune session. ES-17 (« non exécutable, sans cookie ») est tenue par ces trois faits ; l'en-tête reste une défense en profondeur. V1 : CloudFront avec politique d'en-têtes de réponse, qui rétablit `nosniff`. Corollaire pour D08 : `img-src` liste l'hôte exact du bucket, jamais `*.amazonaws.com`, qui autoriserait le bucket de n'importe qui.
- **`LabRole` partagé** : écart accepté pour le POC, déjà documenté (ES-38, D03 §MS3), sur données fictives. D07 doit l'écrire sans l'atténuer : `LabRole` lit aussi Secrets Manager, donc les mots de passe des autres services et le mot de passe maître ; l'isolement I1 tient contre une erreur de code, pas contre une compromission de la Place de marché. La seule borne réelle est MS3 (IMDS réservé, X16) ; F1 en dépend. Politique de bucket par préfixe : sans effet, inutile de la poser. V1 : une identité par service (IRSA ou Pod Identity), politique de bucket limitée au rôle de la Place de marché.
- **Décodage non isolé** : accepté pour le POC, isolement maintenu Must V1 (§5.6). Le risque est surtout un déni de service : avec `CGO_ENABLED=0`, les décodeurs Go sont sûrs pour la mémoire, leurs failles connues étant des épuisements de ressources ; l'exécution de code y est peu probable sans être nulle, et sa portée est aggravée par `LabRole`. Conditions : seuls `image/jpeg` et `image/png` sont enregistrés (aucun import de `image/gif` ni de `golang.org/x/image/...`, qui enregistrent d'autres formats auprès de `image.Decode`) ; plafond de pixels vérifié après `DecodeConfig` ; nombre de décodages simultanés borné ; limite mémoire du conteneur ; délai par dépôt ; test d'une bombe de décompression. Variante POC non exigée (≈ 0,5 à 1 j) : décodeur dans un Deployment distinct, sans IMDS, sans secret, sortie réseau interdite.
- **ES-30** : reste documentée pour la V1. Elle exige des sauvegardes hors de portée des accès courants, impossible ici puisque `LabRole` supprime tout instantané (avis de D03). Un exercice unique de restauration n'est pas exigé ; s'il est fait (≈ 1 h : instantané, restauration, comptage des lignes, durée mesurée), il prouve la procédure et donne une durée de reprise mesurée, sans démontrer ES-30. La V1 précise : sauvegardes automatiques chiffrées, copie dans un coffre verrouillé ou un autre compte, perte maximale et durée de reprise chiffrées.

**2. Points non soumis**

- **Chiffrement au repos** : `storage_encrypted = true` à la création (non activable après coup), clé gérée par AWS ; une clé client n'apporte rien tant que `LabRole` l'administre. Bucket : SSE-S3 et `BucketOwnerEnforced` (listes de contrôle d'accès désactivées) ; pas de versionnage, sinon `down` laisse des versions orphelines.
- **Chiffrement en transit** : `rds.force_ssl = 1` posé explicitement dans le groupe de paramètres ; `sslmode=verify-full` avec le paquet d'autorités RDS embarqué dans l'image et l'adresse de l'instance comme hôte (pas d'alias ni d'IP, qui font échouer la vérification). Test négatif : connexion sans TLS refusée.
- **Réseau** : groupe de sécurité de RDS ouvert sur 5432 depuis le seul groupe des nœuds ; tous les pods partageant ce groupe, politique réseau de sortie vers RDS pour les trois services propriétaires seulement (X12, X19) ; `publicly_accessible = false`.
- **Instance partagée** : une requête coûteuse ou une fuite de connexions d'un service bloque les deux autres (`max_connections` ≈ 80 sur db.t3.micro [?]). `CONNECTION LIMIT` et `statement_timeout` par rôle d'exécution, cohérents avec la taille des pools.
- **Secrets des rôles** : régénérés à chaque `up`, un secret par rôle et par service, lus par `kubectl` depuis l'entrée standard (MS6) ; aucune chaîne de connexion avec mot de passe dans un journal ou un message d'erreur (M-LIV-07).
- **Réservation et outbox** : R1 ne pose pas de problème de sécurité ; l'échéance plafonnée côté serveur traite M-RES-02. La charge de l'outbox ne contient aucune donnée personnelle au-delà du nécessaire, purge bornée (M-CAN-04).

**3. Mesures minimales, si la recommandation est retenue**

| Id | Mesure | Exigences, menaces | Preuve |
|---|---|---|---|
| MD1 | Rôle d'exécution sans attribut ni propriété, droits par `ALTER DEFAULT PRIVILEGES` ; `REVOKE CONNECT … FROM PUBLIC` ; `REVOKE CREATE ON SCHEMA public` | ES-38, FC8, M-MOD-01 | Tests négatifs : connexion à la base d'un autre service refusée ; `CREATE TABLE` refusé au rôle d'exécution |
| MD2 | Secret de migration livré au seul Job ; mot de passe maître jamais dans le cluster au-delà de l'amorçage ; secrets régénérés à chaque `up` | ES-06, MS6, M-LIV-02 | Inventaire des Secrets du Deployment ; aucun Secret maître après amorçage |
| MD3 | `storage_encrypted`, `rds.force_ssl = 1`, `verify-full` avec le paquet d'autorités RDS, instance non publique, entrée 5432 limitée | ES-01, ES-07, MS7, M-LIV-03, M-LIV-04 | Inventaire (X17) ; connexion sans TLS refusée |
| MD4 | `CONNECTION LIMIT` et `statement_timeout` par rôle d'exécution | ES-29, M-DIS-07 | Configuration des rôles relue par l'amorçage |
| MD5 | Décodeurs JPEG et PNG seuls enregistrés, plafond de pixels après `DecodeConfig`, décodages simultanés bornés, limite mémoire et délai | ES-17, M-ANN-01, M-ANN-07 | Tests : GIF, WebP et SVG refusés ; bombe de décompression refusée sans dépasser la mémoire du pod |
| MD6 | Objet écrit avec `Content-Type: image/jpeg` fixé par le serveur, nom aléatoire de 128 bits ; bucket privé, SSE-S3, TLS imposé, `BucketOwnerEnforced`, sans CORS ni versionnage | ES-07, ES-17, M-ANN-06, M-LIV-03 | Inventaire du bucket ; accès anonyme refusé |
| MD7 | URL présignées ≤ 15 min ; émission pour un article non publié réservée au vendeur propriétaire et à l'admin ; URL jamais journalisées | ES-03, ES-04, ES-17, M-ANN-06, M-LIV-07 | Test « B n'obtient pas l'URL d'une photo de l'article non publié de A » ; URL expirée refusée |
| MD8 | `img-src` limité à l'hôte exact du bucket (D08) | ES-07, ES-08 | Contrôle de la CSP par ZAP |
| MD9 | Écarts écrits dans D07 et §5.6 : `nosniff` absent sur les photos, `LabRole` lisant tous les secrets, décodage dans le processus ; F1 conditionnée à X16 | ES-17, ES-38, M-MOD-03 | Registre des écarts à jour |
| MD10 | ES-30 maintenue « Documentée » ; exercice de restauration facultatif, consigné s'il est fait | ES-30, M-DIS-05 | Procédure V1 rédigée |

**4. Menace à ajouter au registre** (à reporter dans exigences-securite.md, §4.9, par Martin)

| Id | STRIDE | Frontière | Menace | Impact | P | I | Risque | Couverture |
|---|---|---|---|---|---|---|---|---|
| M-DIS-07 | D | FC8 | Instance de base partagée : une requête coûteuse, une boucle ou une fuite de connexions dans un service épuise les connexions ou le calcul de l'instance. | Indisponibilité simultanée des trois services propriétaires de données. | 2 | 2 | 4 Modéré | MD4 (ES-29) ; V1 : instance dédiée pour le service qui en a besoin |

Aucune autre menace nouvelle : l'émission d'URL pour un article non publié relève de M-ANN-06 et M-ACC-02, la chaîne `LabRole` de M-MOD-03.

**Verdict.** Favorable, sous réserve de MD1 à MD9. Aucun point bloquant ; MD7 (contrôle d'accès à l'émission avant publication) et MD2 (secret de migration hors du Deployment) sont les deux corrections de fond à reporter dans la recommandation.

## Avis faisabilité

Fonction : Faisabilité — 2026-10-26. Références : [D04](D04-stack.md) (Go, `pgx` v5, `goose` v3, testcontainers-go, bibliothèques d'images), [D03](D03-hebergement-orchestration.md) (couches éphémère et persistante, MS3, MS6, local sur k3d ou minikube), avis sécurité ci-dessus (MD1 à MD10). Avis sur dossier ; les points [?] relèvent des sondes de D14.

**Synthèse.** Faisable avec la stack de D04, sans bibliothèque nouvelle hors un client S3 compatible en local. Favorable à S1 + P1 + I1 + R1 + F1 et à l'outbox. L'effort annoncé (≈ 3 j hors US) est sous-estimé : **≈ 5,5 à 7 j** une fois MD1 à MD7 intégrées. Deux points de fond ne sont pas encore fixés : le lieu d'exécution de l'amorçage (MD2) et le chemin d'amorçage des photos, qui dépend de D06.

### Développement

- **Réservation R1** : faisable en `READ COMMITTED`. PostgreSQL relit la condition `WHERE` sur la version verrouillée de la ligne, donc une seule des mises à jour concurrentes réussit. Trois précisions :
  - `pg_advisory_xact_lock` prend un entier : la clé vient de l'identifiant de l'acheteur par `hashtextextended(acheteur, 0)`. Une collision ne fait que sérialiser deux acheteurs, sans erreur de règle ;
  - `now()` vaut l'heure de début de transaction, figée pendant l'attente du verrou : la lecture est prudente (une réservation tout juste expirée paraît encore active), acceptable ; ouvrir la transaction juste avant le verrou ;
  - index partiel sur `article (reserve_par) WHERE reserve_par IS NOT NULL` pour l'étape 2.
- **Test de concurrence** : testcontainers-go contre un PostgreSQL de même version majeure que RDS (à figer, 16 ou 17 [?]), jamais un double. Cas : N goroutines et N acheteurs sur un article (une réussite exactement) ; N essais d'un même acheteur (même `reservation_id`, même échéance) ; un acheteur sur deux articles en parallèle (un refus) ; modification du vendeur contre réservation. Lancé avec `-race` et `-count` répété. Prérequis : Docker sur le poste et sur l'exécuteur de CI. Docker compose reste l'option de repli, moins isolée entre tests.
- **`goose` et rôles** :
  - la base est créée avec `OWNER` le rôle de migration. À partir de PostgreSQL 15, ce rôle possède alors le schéma `public`, et `REVOKE CREATE ON SCHEMA public FROM PUBLIC` (MD1) se pose sans le compte maître ;
  - sur RDS, le compte maître doit être membre du rôle pour créer une base qui lui appartient (`GRANT <migration> TO <maître>`), règle durcie en PostgreSQL 16 [?] ;
  - la première migration, exécutée par le rôle de migration, pose `ALTER DEFAULT PRIVILEGES` (tables et séquences) vers le rôle d'exécution, sans clause `FOR ROLE`. La table de version de `goose` appartient au rôle de migration ; le rôle d'exécution n'a aucun droit dessus.
- **Outbox `SKIP LOCKED`** : faisable, un goroutine relais par réplica. `SELECT … FOR UPDATE SKIP LOCKED LIMIT n`, publication avec délai borné, marquage, validation : la transaction reste ouverte pendant la publication. Il faut donc `idle_in_transaction_session_timeout` sur le rôle (MD4 ne couvre pas ce cas). L'ordre par clé n'est pas garanti entre réplicas ; les gardes de révision le rendent sans effet. Effort ≈ 1 j, à confirmer quand D05 aura choisi le broker.
- **Photos** : tout est en Go pur, compatible avec `CGO_ENABLED=0` (`image/jpeg`, `image/png`, `x/image/draw`, `aws-sdk-go-v2`).
  - Dépôt lu en mémoire sous `MaxBytesReader`, `DecodeConfig` sur ce tampon, refus au-delà d'un plafond de pixels (24 Mpx par exemple, ≈ 96 Mo en RGBA), puis décodage. Un sémaphore borne les décodages simultanés (2 par exemple) ; la limite mémoire du conteneur se dimensionne en conséquence (≈ 512 Mi).
  - Bombe de décompression (MD5) : un PNG de 20 000 × 20 000 pixels compressé en quelques centaines de Ko est refusé dès `DecodeConfig`, qui lit les dimensions de l'en-tête. Le test génère ce fichier au lieu d'en versionner un.
  - Orientation EXIF : la bibliothèque standard ne la lit pas. Il faut un petit lecteur du segment APP1 ou une dépendance de plus [?], à trancher en US.
  - Précision sur MD5 : `x/image/draw`, retenu par D04, n'enregistre aucun format. Seuls les sous-paquets de format (`webp`, `bmp`, `tiff`, `vp8`…) le font. La règle vérifiable est « aucun import de paquet de format hors `image/jpeg` et `image/png` », contrôlée par `go list -deps` en CI.
- **MD7, contrôle d'accès des URL** : ≈ 0,25 j. Une même fonction de présignature est appelée par deux routes distinctes du catalogue : vendeur (`/vendeur/articles/{id}/photos`) et admin. Le contrôle compare le sujet du jeton au vendeur de l'article, ou exige le rôle admin. Il dépend des claims fixés par D06.

### Exploitation

- **Ordre d'allumage** :
  1. Terraform, RDS en parallèle du cluster ;
  2. attente des deux ;
  3. génération des mots de passe et rangement dans Secrets Manager ;
  4. Job d'amorçage ;
  5. création des Secrets Kubernetes : exécution pour les Deployments, migration pour les Jobs seulement ;
  6. `helm install`, avec le crochet `pre-install` qui lance `migrate` ;
  7. Deployments ;
  8. amorçage des données.

  Le chemin critique devient max(cluster, RDS) + ≈ 2 min.
- **Lieu de l'amorçage (MD2)** : le poste ne joint pas la base privée, donc l'amorçage tourne en Job éphémère dans le cluster. Image officielle `postgres` (client `psql`) et script SQL idempotent en ConfigMap. Le mot de passe maître est injecté par un Secret créé depuis l'entrée standard, puis Job et Secret sont supprimés après `kubectl wait`. Ce Job doit être couvert par la politique réseau de sortie vers RDS. Ne pas loger l'amorçage dans le binaire d'un service : il porterait le compte maître.
- **Crochets Helm** : un crochet `pre-install` s'exécute avant les ressources ordinaires du chart. Secrets, ConfigMap du paquet d'autorités RDS et droit de sortie vers RDS doivent donc exister avant (script `up`, ou crochets de poids inférieur), sinon la première installation échoue.
- **Recréation RDS** : 10 à 16 min [REX]. `backup_retention_period = 0`, `delete_automated_backups = true` et `deletion_protection = false` accélèrent la création et la destruction et évitent des sauvegardes orphelines. La destruction prend ≈ 5 à 10 min [?] et doit tenir dans `down` avant l'expiration de la session.
- **Bucket S3** : `down` le vide par `aws s3 rm --recursive`, simple puisque le versionnage est désactivé. `up` le purge aussi avant l'amorçage, car une session expirée avant `down` laisse des orphelines. Les photos de test passent par le chemin de dépôt. Via HTTP, il faut un jeton vendeur, donc D06 doit être prêt. L'alternative est une sous-commande `seed` du binaire de la Place de marché, qui appelle le même cas d'usage (vérification, ré-encodage, écriture) sans HTTP, lancée en Job `post-install` avec l'accès IMDS de la Place de marché (MS3).
- **S3 en local** : MinIO n'est plus distribué en image maintenue pour l'édition communautaire depuis fin 2025, et son dépôt est passé en maintenance [?] (à vérifier). Alternatives compatibles avec les URL présignées SigV4 :
  - SeaweedFS (`weed server -s3`, Apache 2.0, un conteneur) : recommandé ;
  - Garage (AGPL, initialisation du plan de stockage à scripter) ;
  - une dernière image MinIO figée, tolérable en local seulement sur données fictives.

  Le client se configure par un point de terminaison surchargé et `UsePathStyle = true`, sans autre écart de code.

### Effort et risques

| Lot | Effort |
|---|---|
| Amorçage SQL, rôles, MD1, MD4, Job d'amorçage | 1 j |
| `goose`, Job en crochet, ordre des Secrets | 0,5 j |
| R1, idempotence, garde de révision, harnais testcontainers | 1,5 j |
| Photos F1, MD5, EXIF, MD7 | 1,5 à 2 j |
| Outbox et relais | 1 j |
| Purge et amorçage S3, S3 local | 0,5 à 1 j |
| **Total** | **≈ 5,5 à 7 j** |

Risques principaux :

1. Ordre amorçage, Secrets et crochet `pre-install` mal réglé : la première installation échoue à chaque session.
2. Amorçage des photos par HTTP bloqué tant que D06 n'est pas livré : prévoir la sous-commande `seed`.
3. Équivalent local de S3 à figer [?].
4. Orientation EXIF non couverte par la bibliothèque standard.
5. Durée de destruction de RDS dans `down` [?].

## Décision de Martin

> **« Paquet recommandé »** : S1 + P1 + I1 + R1 + F1 et outbox en table, avec les mesures de sécurité MD1 à MD9.
>
> - Une instance RDS PostgreSQL db.t3.micro Single-AZ, recréée vide à chaque session puis réamorcée avec le jeu de démo.
> - Une base logique par service, avec un rôle de migration et un rôle d'exécution chacun (`goose`).
> - Réservation par `UPDATE` conditionnel ; décision admin avec révision, 409 si elle est périmée.
> - Photos déposées par l'API de la Place de marché, ré-encodées, stockées dans un bucket S3 privé, lues par URL présignée courte. Pas d'antivirus. F1 reste conditionnée au critère X6 (accès S3 par le rôle du nœud, D14).

Date : 2026-10-26

## Conséquences

La recommandation est retenue ; ses conséquences :

- **D05 — Canal** : outbox dans la base de la Place de marché, relais `SKIP LOCKED`, purge bornée ; D05 tranche le broker et la couverture de sa perte de messages (renvoi des articles « en contrôle » ou volume persistant).
- **D06 — Autorisation** : si le serveur d'autorisation est un conteneur, il reçoit une base et des rôles dans la même instance, selon I1.
- **D08 — Exposition** : les photos publiées sont servies par URL présignée depuis S3 ; la CSP (`img-src`) autorise le domaine du bucket ; aucune route d'ALB pour les photos.
- **D10 — Livraison** : Job de migration en crochet Helm `pre-install` et `pre-upgrade` ; échec de migration = déploiement arrêté.
- **D11 — IaC** : RDS dans la couche éphémère avec `skip_final_snapshot` ; groupe de paramètres (`rds.force_ssl`) ; chiffrement au stockage ; script d'amorçage idempotent (bases, rôles, secrets) ; bucket dans la couche persistante, vidé par `down` ; inventaire des instantanés orphelins.
- **D13 — Tests** : test de concurrence de la réservation ; nouvel essai (même réservation, même échéance, une seule commande) ; limite par acheteur ; modification refusée pendant la réservation ; décision admin périmée → 409 ; test négatif d'accès croisé entre bases ; fichier non image refusé, photo publiée sans géolocalisation, photo non publiée inaccessible.
- **D14 — Expérimentations** : X6 (S3 par le rôle du nœud, présignature comprise) conditionne F1 ; sonde du compte : chiffrement RDS, groupe de paramètres, politique de bucket [?].
- **Architecture** : § 3 et § 4.3 d'[architecture-technique.md](../architecture/architecture-technique.md) précisés (base logique par service, URL présignées, outbox).
- **Sécurité** : [exigences-securite.md](../securite/exigences-securite.md) : FC8 précisée (rôle de migration, rôle d'exécution) ; ES-17 : mécanisme de lecture des photos publiées.
- **Coût** : ≈ 0,085 $ par session pour la base, ≈ 2,1 $ sur le projet ; S3 négligeable.
- **Index** : D07 à mettre à jour dans [INDEX.md](INDEX.md).

## À savoir défendre en soutenance

1. **« Quatre services, mais une seule base : où est l'isolement ? »**
   - Une instance, mais une base logique et des comptes distincts par service ; un service ne peut même pas se connecter à la base d'un autre, test à l'appui.
   - Une instance par service coûterait trois fois plus sans rien prouver de plus dans le POC. En V1, un service qui en a besoin passe sur sa propre instance sans changer de code : seule la chaîne de connexion change.
2. **« Comment empêchez-vous deux acheteurs de réserver le même objet ? »**
   - Une seule mise à jour conditionnelle sur la ligne de l'article : la base garantit qu'une seule réussit, prouvé par un test d'appels simultanés.
   - Le nouvel essai renvoie la même réservation ; la contrainte d'unicité sur la commande empêche une seconde commande ; l'échéance est fixée par la base, jamais par l'appelant.
3. **« Pourquoi pas de transaction répartie entre Transactions et la Place de marché ? »**
   - La disponibilité appartient à la Place de marché (D02) : la réservation est locale et atomique. L'échéance la libère d'elle-même, sans message de compensation.
4. **« Vos photos sont-elles sûres ? »**
   - Contenu vérifié, ré-encodage qui supprime les métadonnées et neutralise les fichiers piégés, nom aléatoire, bucket privé et chiffré, URL de lecture courtes, émises seulement pour un article publié.
   - Écart assumé : accès à S3 par le rôle partagé imposé par l'environnement de formation (ES-38) ; décodage isolé documenté pour la V1.
5. **« Vous perdez vos données à chaque session ? »**
   - Oui, volontairement : les données sont fictives et rejouées à chaque allumage, ce qui prouve que l'environnement se recrée depuis zéro (ENF-28) et coûte 0 $ entre sessions.
   - Les sauvegardes et la restauration testée (ES-30, ENF-13) sont une exigence V1, hors de portée d'un compte où le rôle unique peut supprimer un instantané.

## Sources

Consultées le 2026-10-26. Prix issus d'[environnement-learner-lab.md](../00-cadrage/environnement-learner-lab.md) (db.t3.micro par agrégateur, à vérifier dans le calculateur AWS).

- **Projet** : [D02](D02-decoupage-applicatif.md) (Arbitrages de cohérence, Périmètre du POC, Conséquences) ; [D03](D03-hebergement-orchestration.md) (Socle commun, MS3, MS6, MS7, Conséquences) ; [D04](D04-stack.md) (bibliothèques Go, base et migrations) ; [exigences-securite.md](../securite/exigences-securite.md) ; [exigences.md](../00-cadrage/exigences.md).
- **PostgreSQL** : verrous consultatifs, `UPDATE … RETURNING`, contraintes d'exclusion et `btree_gist`, `REVOKE CONNECT`, mode d'isolement par défaut et relecture de la condition sur ligne verrouillée : documentation officielle (https://www.postgresql.org/docs/current/).
- **AWS** : RDS PostgreSQL (`rds.force_ssl`, chiffrement, instantanés) : https://docs.aws.amazon.com/AmazonRDS/latest/UserGuide/CHAP_PostgreSQL.html ; URL présignées S3 : https://docs.aws.amazon.com/AmazonS3/latest/userguide/using-presigned-url.html ; Block Public Access : https://docs.aws.amazon.com/AmazonS3/latest/userguide/access-control-block-public-access.html
- **Outils** : `goose` : https://github.com/pressly/goose ; `pgx` : https://github.com/jackc/pgx
