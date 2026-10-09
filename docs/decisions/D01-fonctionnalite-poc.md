# D01 — Exigences reformulées et fonctionnalité métier du POC

| Champ | Valeur |
|---|---|
| Statut | Décidée |
| Date | 2026-10-09 |
| Décideur | Martin |
| Dépend de | — |
| Impacte | D02, D05, D06, D07, D10, D13, US du backlog (à venir) |
| Critères grille | C3, C2, C7, C5, C8 |

## Contexte et besoin

La décision porte sur deux points :

1. **Valider la reformulation des exigences** et leurs priorités V1 / V2 / Évolution : [exigences.md](../00-cadrage/exigences.md), y compris les interprétations de son § 4.
2. **Choisir la fonctionnalité métier** implémentée par le POC.

- **Exigence** (consigne, § 1 et 1.2.3) :
  - « Vous devez cependant à minima implémenter une fonctionnalité métier. »
  - « La fonctionnalité implémentée dans « votre prototype » doit être clairement décrite sous forme d'un backlog (User story et critères d'acceptation) » (§ 1.2.1).
  - « Le respect de la fonctionnalité devra être présenté via des tests d'appels voire des tests d'acceptation. »
  - La montée en charge est présentée en soutenance par des tests de charge.
- **Attentes de la grille au niveau 5** (détail : [contexte-et-grille.md](../00-cadrage/contexte-et-grille.md) § 4.2) :
  - C3 : la fonctionnalité « répond aux besoins fonctionnels exprimés par la maîtrise d'ouvrage ».
  - C2 : le POC « valide la faisabilité des points clés de l'architecture » : découpage applicatif, communication interservice, sécurité, hébergement, orchestration.
  - C5 : « La disponibilité et la montée en charge de l'application sont démontrées ».
  - C7 : « Au moins deux types de tests sont appliqués au POC. Ils sont exécutés avec succès. »
- **Contraintes** :
  - Martin réalise seul ; soutenance dans plus de 6 semaines.
  - AWS Academy Learner Lab, crédits inférieurs à 100 $, environnement allumé seulement pour un usage précis.
  - Sprint 1 = walking skeleton ; sprint 2 = fonctionnalité métier, charge et autoscaling ([PILOTAGE.md](../PILOTAGE.md), jalons J4 et J5).
  - Démonstration dans les 20 minutes de présentation, test de charge compris.

### Critères de choix

Fixés avant la comparaison des options.

| Critère de choix | Ce qui est regardé |
|---|---|
| Valeur MOA (C3) | Exigence explicite du contexte ; place dans le cœur du métier |
| Points clés d'architecture (C2) | Occasion naturelle de démontrer découpage, communication interservice, sécurité, hébergement, orchestration, sans complexité artificielle |
| Charge réaliste (C5) | Scénario crédible pour Collector, rejouable en démonstration |
| Testabilité (C7) | AC Given/When/Then déterministes, automatisables en CI |
| Effort et risque solo | Tient dans le sprint 2 ; peu de dépendances |
| Robustesse de la démo (C10) | Aucun service tiers indispensable pendant la soutenance |

## Options

### Option A — Mise en vente d'un article avec contrôle automatisé de publication

- **Description** :
  - *Valeur métier* : un vendeur particulier met en vente un objet qui n'apparaît au catalogue public qu'après un contrôle automatisé, ce qui garantit la qualité des annonces sans intervention humaine.
  - *Parcours principal* :
    1. Le vendeur, authentifié, crée un article dans une de ses boutiques : titre, catégorie, description, photos, prix, frais de port.
    2. Il soumet l'article, qui passe « en contrôle » et reste invisible au catalogue.
    3. Le contrôle automatisé vérifie la complétude, l'absence de coordonnées personnelles, la cohérence du prix avec la catégorie et le statut du vendeur.
    4. Issue : « publié » ; « refusé » avec motif, que le vendeur corrige avant de soumettre à nouveau ; ou « à vérifier », tranché par l'admin.
    5. Un visiteur non authentifié trouve l'article par sa catégorie et consulte sa fiche, vendeur identifié comme particulier.
  - *Exigences couvertes* : EF-01, EF-02, EF-04, EF-12, EF-16 à EF-19, EF-36 (partie annonces), EF-42 (partiel : prix incohérent avec la catégorie, statut du vendeur), ENF-17 (contrat versionné du contrôle), revalidation après modification d'un article publié (aucune EF dédiée : ES-18), ENF-26 et ENF-30 (version minimale). Prérequis fournis par des données fictives : catégories (EF-05), compte vendeur et boutique (EF-11).
- **Avantages** :
  - Cœur du métier : sans article publié, ni vente, ni notification, ni recommandation. Toutes les autres fonctionnalités en dépendent.
  - Exigences explicites et propres au sujet : contrôle avant mise en vente, automatisé « le plus possible ».
  - Trois rôles et le système interviennent sur un même parcours : visiteur, vendeur, admin et contrôle automatisé. Le rôle acheteur est couvert par un AC négatif : un acheteur sans boutique ne peut pas créer d'article.
  - Aucun service tiers nécessaire.
- **Inconvénients** :
  - Le dépôt de photos ajoute un stockage de fichiers et une surface d'attaque (taille, type, contenu des fichiers).
  - Le résultat du contrôle est différé : les tests attendent ce résultat dans un délai borné.
  - Ne démontre ni le paiement ni la commission, qui portent le modèle économique.
  - Risque de dérive si le contrôle vise une détection « intelligente » : il doit rester un jeu de règles déterministes.
- **Coût AWS et effort** :
  - AWS : faible (stockage de quelques photos fictives).
  - Effort solo : moyen, compatible avec le sprint 2. Postes principaux : saisie d'article avec photos, règles de contrôle, catalogue public et fiche article, liste « à vérifier » de l'admin.
- **Complexité ajoutée** : un traitement automatisé découplé de la saisie ; un stockage de fichiers.
- **Couverture grille** :
  - **C2** :
    - *Découpage* : deux composants, les annonces (catalogue compris) et le contrôle de publication (traitement automatisé, remplaçable). Le détail relève de D02.
    - *Communication interservice* : la soumission déclenche un contrôle dont le résultat change l'état de l'article et revient au vendeur, un échange naturellement asynchrone ; la consultation est naturellement synchrone. Les deux modes sont démontrables si D05 les retient.
    - *Sécurité* : lecture anonyme, vendeur limité à ses propres annonces, admin seul à trancher les cas « à vérifier » : autorisation par rôle et par propriété démontrable. Blocage des coordonnées personnelles dans les annonces. Contrôle des fichiers déposés.
    - *Hébergement et orchestration* : deux profils de charge distincts (lecture publique, traitement des contrôles), ajustables indépendamment.
  - **C3** : couvre des besoins explicites de la MOA : photos, description, prix, frais de port, contrôle automatisé, boutiques, vendeur identifié comme particulier, catalogue sans authentification, rôle admin.
  - **C5** : scénario « afflux de visiteurs sur le catalogue après un salon Collector ou une campagne marketing ». Lecture anonyme, sans effet de bord, rejouable à volonté. Complément : une vague de soumissions qui montre l'absorption des contrôles.
  - **C7** : règles déterministes, donc AC nets et automatisables. Exemples : article sans photo refusé avec motif ; description contenant un email refusée ; prix hors plage « à vérifier » ; article valide visible sans authentification dans le délai fixé ; un vendeur ne peut pas modifier l'annonce d'un autre ; un visiteur ne peut pas créer d'article ; un acheteur sans boutique ne peut pas créer d'article.

### Option B — Achat d'un article avec paiement par carte et commission de 5 %

- **Description** :
  - *Valeur métier* : un acheteur achète un objet unique et le paie par carte sur la plateforme, qui prélève sa commission de 5 % et garantit que l'objet n'est vendu qu'une fois.
  - *Parcours principal* :
    1. L'acheteur, authentifié, consulte un article publié et lance l'achat.
    2. L'article est réservé : aucun autre acheteur ne peut l'acheter.
    3. L'acheteur paie par carte auprès du prestataire de paiement, en mode test.
    4. Paiement confirmé : commande enregistrée, article « vendu », commission de 5 % et montant dû au vendeur calculés ; acheteur et vendeur voient la transaction dans leur espace.
    5. Paiement refusé ou expiré : la réservation est levée et l'article remis en vente.
  - *Exigences couvertes* : EF-22, EF-23, EF-24, EF-25 (partiel), EF-34 (partiel), ENF-01, ENF-04. Prérequis : articles publiés, injectés comme données fictives sans passer par le contrôle.
- **Avantages** :
  - Porte le modèle économique : la commission est le revenu de Collector.
  - Sujet de sécurité de premier plan (transactions financières), riche pour l'audit.
  - Cohérence sous accès concurrents (objet unique) intéressante à démontrer.
- **Inconvénients** :
  - Dépend d'un prestataire de paiement externe : compte de test, secrets, retour asynchrone du prestataire à recevoir dans l'environnement managé.
  - Cohérence entre plusieurs étapes (réservation, paiement, annulation compensatoire) complexe à réaliser seul.
  - Articles injectés sans contrôle : récit métier amputé de son amont.
  - Démonstration exposée à la disponibilité d'un tiers (effet démo).
- **Coût AWS et effort** :
  - AWS : faible ; flux sortants vers le prestataire à vérifier dans le Learner Lab.
  - Effort solo : élevé, déborde probablement du sprint 2.
- **Complexité ajoutée** : intégration d'un tiers financier ; cohérence entre étapes avec compensation ; confirmations idempotentes.
- **Couverture grille** :
  - **C2** :
    - *Découpage* : frontière forte autour du périmètre financier.
    - *Communication interservice* : enchaînement réservation → paiement → confirmation ou annulation ; le plus démonstratif, mais le plus coûteux.
    - *Sécurité* : intégrité des montants, authenticité des retours du prestataire, aucune donnée de carte côté plateforme.
    - *Hébergement et orchestration* : appels sortants et secrets d'un tiers à gérer.
  - **C3** : valeur MOA très forte, mais fonctionnalité privée de l'amont (publication) : risque d'être jugée incomplète.
  - **C5** : scénario « vente flash d'une édition limitée » crédible. Mais le prestataire ne peut pas être sollicité en charge : un simulateur est nécessaire, ce qui réduit la représentativité. Chaque achat consomme un objet unique : jeu de données à reconstruire à chaque tir.
  - **C7** : AC clairs (commission exacte, double achat refusé, paiement refusé → article remis en vente), mais tests dépendants d'un substitut du prestataire et nombreux cas asynchrones.

### Option C — Alerte de variation de prix aux acheteurs intéressés et à la détection des fraudes

- **Description** :
  - *Valeur métier* : quand un vendeur change le prix d'un article, les acheteurs qui le suivent en sont informés selon leurs préférences, et la variation est transmise à la détection des fraudes.
  - *Parcours principal* :
    1. L'acheteur, authentifié, suit un article publié et choisit de recevoir les alertes de prix dans son espace et par email.
    2. Le vendeur modifie le prix de l'article.
    3. La variation est historisée (ancien prix, nouveau prix, date).
    4. L'acheteur reçoit une notification dans son espace, et par email s'il l'a choisi.
    5. La détection des fraudes reçoit la variation ; si elle est anormale, une alerte est créée et l'admin la consulte.
  - *Exigences couvertes* : EF-13, EF-15, EF-31, EF-32, EF-33, EF-41, EF-43 (partiel), ENF-12, ENF-17. Prérequis : articles publiés et comptes acheteurs, injectés comme données fictives.
- **Avantages** :
  - Exigence propre au sujet, citée deux fois (acheteurs intéressés, composant de fraude).
  - Un même événement servi à deux destinataires indépendants : démonstration très lisible du découplage.
  - Montre l'intégration d'un outil de fraude « interne ou acheté ».
- **Inconvénients** :
  - Fonctionnalité périphérique : suppose des articles et des acheteurs existants.
  - Envoi d'emails réels depuis le Learner Lab à vérifier ; à défaut, canal email simulé.
  - La détection de fraude doit rester un simple seuil, sinon dérive du périmètre.
- **Coût AWS et effort** :
  - AWS : faible ; service d'envoi d'emails à vérifier dans le Learner Lab.
  - Effort solo : moyen à élevé ; les prérequis (suivi d'article, préférences, deux destinataires, email) chargent le sprint 2.
- **Complexité ajoutée** : diffusion d'un événement à plusieurs destinataires ; gestion des préférences ; canal email.
- **Couverture grille** :
  - **C2** :
    - *Découpage* : frontières entre annonces, notifications et fraude.
    - *Communication interservice* : diffusion d'un événement à plusieurs consommateurs, la plus démonstrative des trois options pour l'asynchrone.
    - *Sécurité* : préférences et adresse email sont des données personnelles ; autorisation sur le suivi d'article.
    - *Hébergement et orchestration* : consommateurs ajustables indépendamment.
  - **C3** : exigence explicite, mais hors du parcours principal d'une marketplace : risque d'être jugée incomplète.
  - **C5** : scénario « baisse de prix d'un article très suivi » : pic de notifications à absorber. Crédible mais rare ; la charge réelle d'une marketplace porte sur la consultation.
  - **C7** : AC testables, mais asynchrones sur deux destinataires et un canal email (boîte de test nécessaire) : tests plus fragiles.

### Synthèse

| Critère de choix | A — Mise en vente contrôlée | B — Achat et commission | C — Alerte de prix |
|---|---|---|---|
| Valeur MOA (C3) | Forte : prérequis de tout le métier | Très forte, mais amont manquant | Moyenne : explicite mais périphérique |
| Points clés d'architecture (C2) | Forts, sans complexité artificielle | Très forts, au coût le plus élevé | Forts sur l'asynchrone |
| Charge réaliste (C5) | Forte : consultation après un salon, rejouable | Moyenne : paiement simulé, données consommées | Moyenne : pic rare |
| Testabilité (C7) | Forte : règles déterministes | Moyenne : substitut du prestataire | Moyenne : asynchrone multiple, email |
| Effort et risque solo | Moyen, tient dans le sprint 2 | Élevé, déborde | Moyen à élevé |
| Robustesse de la démo (C10) | Forte : aucun tiers | Faible : tiers de paiement | Moyenne : canal email |

## Recommandation

**Option A — Mise en vente d'un article avec contrôle automatisé de publication.**

1. C'est le cœur du métier : toutes les autres fonctionnalités en dépendent, et le besoin MOA est reconnu immédiatement (C3).
2. Elle offre une occasion naturelle, non artificielle, de valider les points clés d'architecture (C2) : lecture publique, traitement automatisé découplé, autorisation sur trois rôles et le système.
3. Le contrôle automatisé est le point d'entrée naturel de l'outil de fraude « interne ou acheté » (EF-42, ENF-17) : la V1 pourra le brancher sans refonte.
4. Son scénario de charge est le plus crédible pour une marketplace issue de l'événementiel : afflux sur le catalogue après un salon ou une campagne (C5), rejouable sans remise à zéro des données.
5. Ses règles déterministes donnent des tests d'acceptation stables en CI (C7), et aucun service tiers ne menace la démonstration (C10).
6. Son effort est compatible avec un travail seul dans le sprint 2.

Garde-fous proposés :

- Le contrôle reste un jeu de règles déterministes simples : complétude, coordonnées personnelles, plage de prix par catégorie, statut du vendeur. Aucune détection « intelligente » dans le POC.
- Catégories, comptes et boutique sont fournis par des données fictives ; leur gestion complète reste hors POC.
- Paiement, chat, notifications, recommandations et notation restent au backlog V1 ([exigences.md](../00-cadrage/exigences.md)), hors POC.
- Extension, seulement si le sprint 2 se termine en avance : renvoyer au contrôle un article dont le prix change, même sous le seuil de revalidation d'ES-18 (EF-13, EF-41 partiel). Elle réutilise le même canal et le même composant.

## Avis sécurité

Fonction : Sécurité — 2026-10-09. Référence : [exigences-securite.md](../securite/exigences-securite.md) (menaces M-xxx, exigences ES-xx).

Socle commun aux trois options, non repris ci-dessous : ES-01 (HTTPS), ES-02 et ES-03 (authentification, rôles), ES-05 et ES-06 (scan de vulnérabilités, secrets).

### 1. Surface de sécurité démontrable dans le POC

**A — Mise en vente contrôlée**
- Exigences : ES-04 (un vendeur n'agit que sur ses articles et brouillons), ES-08 (contenu public affiché au catalogue et chez l'admin), ES-10 (second facteur admin), ES-16 (coordonnées dans l'annonce), ES-17 (upload), ES-18 (contrôle avant publication), ES-23 (décisions de contrôle et d'admin journalisées avec motif), ES-29 (limitation de débit sur le catalogue et l'upload). ES-03 s'applique aux rôles du parcours.
- Menaces critiques : M-ACC-02 traitée ; M-BO-01 en partie (ES-08 coupe le vecteur « script », ES-10 si retenu) ; M-CHA-01 en partie (la détection de coordonnées des annonces est celle que réutilisera le chat) ; M-FRA-01 en partie (contrôle avant publication).
- Menaces élevées : M-ANN-01 à M-ANN-05, M-BO-02, M-FRA-02, M-LIV-08, M-DIS-01.
- Preuve : forte. Règles déterministes, tests négatifs automatisés en CI, aucun tiers.

**B — Achat et commission**
- Exigences : ES-04 (commande visible des seules parties), ES-12 (aucune donnée de carte), ES-13 (montants côté serveur), ES-14 (confirmation vérifiée, idempotente, réservation atomique), ES-23 (étapes de paiement), ES-29 (lancement du paiement).
- Menaces : critiques M-PAY-02 et M-ACC-02 ; élevées M-PAY-01, M-PAY-03, M-PAY-04, M-PAY-07.
- Preuve : affaiblie. La vérification des notifications et l'absence de données de carte se prouvent contre un simulateur, pas contre le prestataire réel.

**C — Alerte de variation de prix**
- Exigences : ES-04 (suivi et préférences), ES-18 (historisation des prix, en partie), ES-21 (événements fiables vers la fraude), ES-23, ES-24 (en partie), ES-25, ES-31.
- Menaces : aucune critique. Élevées : M-FRA-02, et M-NOT-01 seulement avec un vrai domaine d'envoi. Le reste est Modéré : M-FRA-04 à M-FRA-06, M-NOT-02, M-NOT-03.
- Preuve : moyenne. Asynchrone vers deux destinataires ; canal email probablement simulé, donc ES-24 non prouvée.

| | A | B | C |
|---|---|---|---|
| Exigences au-delà du socle | 8, toutes Must | 6, Must, dont 2 prouvées sur simulateur | 7, dont 3 Should et 2 partielles |
| Risques critiques touchés | 4 sur 5, dont 3 en partie | 2 sur 5 | 0 sur 5 |
| Qualité de la preuve | Forte | Affaiblie | Moyenne |

### 2. Risques introduits dans le POC lui-même

**A**
- **Upload** : fichier piégé ou image malformée visant le traitement d'image (M-ANN-01), image géante ou bombe de décompression (M-ANN-07), photos non validées accessibles par adresse prévisible (M-ANN-06), stockage de fichiers exposé par erreur de configuration (M-LIV-03), crédit de l'environnement consommé (M-DIS-03). Les photos du POC sont fictives : la fuite de géolocalisation (M-ANN-02) n'y a pas d'impact réel, mais son traitement doit être démontré.
- **Contenu non fiable relu par l'admin** : la liste « à vérifier » affiche par construction les annonces les plus suspectes. Un script dans une description y vise directement la session admin (M-ANN-03, puis M-BO-01).
- **Contournement du contrôle** : un article déjà publié est modifié sans nouveau contrôle (M-ANN-05).
- **Contrôle ouvert en cas de panne** : si le contrôle est indisponible ou en retard, l'article ne doit jamais devenir visible par défaut (ES-31).
- **Coût de la détection** : un texte très long ou construit pour cela peut rendre la recherche de coordonnées très coûteuse et bloquer le contrôle (M-DIS-02). Parade : longueur bornée, motifs de détection à coût linéaire.
- **Motif de refus trop précis** (risque faible) : un motif qui cite la règle exacte apprend au vendeur à la contourner. Préférer une catégorie (« coordonnées détectées »).

**B**
- **Confirmation forgée ou rejouée** (M-PAY-02, M-PAY-05) : le point d'entrée des notifications est public par nature. Avec un simulateur, accepter des notifications non vérifiées est tentant, et ce serait un constat critique à l'audit.
- **Montant falsifié** (M-PAY-01) et erreurs d'arrondi (ENF-01).
- **Réservation abusive** : un compte réserve en boucle les objets uniques sans payer et les retire de la vente (menace à ajouter au registre si B est retenue). Parade : expiration courte de la réservation, limite de réservations simultanées par compte.
- **Secrets du prestataire**, même de test, exposés dans le dépôt ou le pipeline (M-LIV-02) ; point d'entrée public supplémentaire (M-LIV-03).
- **Simulateur** : il ne doit jamais imiter un formulaire de carte ni recevoir de donnée qui ressemble à un numéro de carte.

**C**
- **Données personnelles dans les notifications** : identité ou email des acheteurs qui suivent un article visibles du vendeur, notification lisible par un autre utilisateur (M-ACC-02), envoi vers une adresse non vérifiée (M-NOT-03).
- **Titre d'annonce repris tel quel dans l'email** (M-NOT-02).
- **Interface fraude** : faux événement ou faux verdict (M-FRA-05), données transmises au-delà du nécessaire (M-FRA-06).
- **Variations abusives** : baisses et hausses répétées pour inonder les suiveurs (M-NOT-03), ou petites variations successives sous le seuil pour échapper à la détection (M-FRA-02). Parade : seuil cumulé sur une période.
- **Canal email** : identifiants d'envoi à protéger (M-LIV-02) ; domaine d'envoi non authentifié si l'environnement ne le permet pas.

### 3. Option qui sert le mieux le critère C8 et le plan de remédiation

**Option A.**
1. Elle démontre le plus d'exigences au-delà du socle et touche le plus de risques critiques. Ses preuves sont automatisées et ne dépendent d'aucun tiers.
2. Elle alimente le mieux le plan de remédiation. Ses surfaces réelles (upload, contenu public relu par l'admin, contrôle asynchrone, catalogue public sous charge) produisent de vrais constats de scans et de tests de charge. Les risques qu'elle ne couvre pas, à commencer par M-PAY-02, restent documentés et deviennent des actions P1 justifiées du plan.
3. B a la plus forte valeur de sécurité sur le papier, mais sa preuve repose sur un simulateur, et son effort menace le sprint 2 : le POC risque de finir sans aucune bonne pratique métier démontrée. C ne touche aucun risque critique.

Bonnes pratiques métier à démontrer dans A, en plus du socle :
- **ES-16, filtrage des coordonnées** dans le titre, la description et le nom de boutique, variantes masquées comprises. C'est déjà une règle du contrôle (EF-17, EF-36). Elle est propre à Collector et traite le mécanisme du risque critique M-CHA-01.
- **ES-04 avec ES-03, contrôle par propriété et par rôle** : un vendeur ne modifie pas l'article d'un autre, un vendeur ne tranche pas un « à vérifier », un visiteur ne crée pas d'article. Traite M-ACC-02 et M-BO-02 ; ces cas figurent déjà dans les AC d'exemple.
- **ES-17, upload sécurisé**, n'est pas optionnel : c'est la condition pour que A n'introduise pas de risque Élevé. Version minimale : format vérifié sur le contenu, taille maximale, ré-encodage (qui supprime la géolocalisation), nom de fichier généré par le serveur, stockage privé jusqu'à la publication.

Preuves à prévoir dans les AC : fichier non image refusé ; photo publiée sans géolocalisation ; description contenant « jean point dupont arobase… » refusée ; balisage dans un titre affiché comme du texte, au catalogue et chez l'admin ; vendeur B refusé sur l'article du vendeur A ; vendeur refusé sur la décision admin.

### 4. Variantes à faible coût

Le filtrage des coordonnées (ES-16) fait déjà partie de A (EF-17). Deux ajouts apportent un vrai gain pour un coût faible :
- **Revalidation après modification (ES-18)** : toute modification d'un article publié (photos, description, catégorie, prix au-delà d'un seuil) le renvoie « en contrôle ». Le parcours est le même que la soumission, donc le coût est faible. Sans cela, le contrôle se contourne en une étape (M-ANN-05), et l'audit le relèverait aussitôt.
- **Contrôle fermé par défaut (ES-31)** : un article n'est visible qu'après une décision « publié ». Contrôle indisponible ou en retard veut dire article invisible. C'est une règle de conception, sans coût.
- **En option**, si le serveur d'autorisation le permet par simple paramétrage : **second facteur pour l'admin (ES-10)**, qui traite le risque critique M-BO-01.

### 5. Verdict

Avis favorable à l'option A, à condition d'y intégrer l'upload sécurisé minimal (ES-17) et la revalidation après modification (ES-18), et d'y démontrer comme bonnes pratiques métier le filtrage des coordonnées (ES-16) et le contrôle d'accès par propriété et par rôle (ES-04, ES-03).

## Avis faisabilité

### Points clés d'architecture par candidate

Légende : **Naturel** = découle du besoin MOA · **Coûteux** = découle du besoin, mais cher pour une personne seule · **Forcé** = n'existerait que pour cocher un critère.

| Point clé | A — Mise en vente contrôlée | B — Achat et commission | C — Alerte de prix |
|---|---|---|---|
| Découpage applicatif | **Naturel** : deux composants, les annonces (propriétaires de l'article, catalogue compris) et le contrôle de publication (automatisé, remplaçable par l'outil de fraude « interne ou acheté »). **Forcé** : séparer le catalogue des annonces (même donnée, recopiée sans besoin). | **Naturel** : frontière autour des commandes et du paiement. **Coûteux** : il faut quand même des annonces et un catalogue pour avoir un article à acheter, soit trois périmètres plus un tiers. | **Naturel** : annonces et notifications. **Forcé** : un composant fraude réduit à un seuil, c'est-à-dire un service déployable dont la logique tient en une comparaison. |
| Communication interservice asynchrone | **Naturel** : soumission → contrôle → résultat, en point à point. Justifié non par la durée des règles mais par trois raisons : contrôle remplaçable (ENF-17), aucune soumission perdue (ENF-12), absorption d'une vague de soumissions. | **Coûteux** : chaîne réservation → paiement → confirmation ou compensation, retour asynchrone du prestataire, confirmations idempotentes. | **Naturel** : un même événement diffusé à deux consommateurs. Le cas le plus démonstratif des trois. |
| Communication interservice synchrone | **Facultatif, non forcé** : le résultat peut revenir par message ou par un appel interne vers les annonces (choix de D05). Un seul mode bien justifié suffit pour C2. | **Naturel** : l'appel au prestataire de paiement, mais c'est un tiers externe, pas un échange entre composants de la plateforme. | **Absent** : le besoin n'appelle aucun échange synchrone entre composants. |
| Sécurité et autorisation | **Naturel** : lecture anonyme, écriture authentifiée, rôles vendeur et admin, contrôle de propriété (un vendeur n'agit que sur ses articles), validation des fichiers déposés, blocage des coordonnées personnelles dans les annonces, accès au canal limité aux deux composants. | **Naturel et riche** : intégrité des montants, authenticité des retours du prestataire, aucune donnée de carte (ENF-04). **Coûteux** : secrets d'un tiers et point d'entrée public pour ses retours. | **Naturel** : données personnelles (email, préférences) et propriété du suivi. Plus pauvre sur l'autorisation par rôle. |
| Hébergement | **Naturel** : composants répliqués, stockage de fichiers, canal de messages, base, fournisseur d'identité. Aucun tiers. | **Coûteux** : flux sortants vers le prestataire et point d'entrée stable pour ses retours, alors que l'environnement est recréé à chaque allumage. | **Coûteux** : service d'envoi d'emails, restrictions de l'environnement de formation à vérifier. |
| Orchestration | **Naturel** : deux signaux de mise à l'échelle différents, le trafic de lecture pour les annonces et les messages en attente pour le contrôle. | **Forcé** : la mise à l'échelle du paiement ne se montre qu'avec un simulateur du prestataire. | **Naturel** : consommateurs ajustés selon les messages en attente. |
| Montée en charge (C5) | **Naturel** : consultation anonyme du catalogue, sans effet de bord, rejouable sans remise à zéro des données. | **Forcé** : simulateur du prestataire nécessaire, et chaque achat consomme un objet unique (données à reconstruire à chaque tir). | **Forcé** : une rafale de changements de prix par un même vendeur n'est pas réaliste ; la charge réelle d'une marketplace n'est pas là. |
| Disponibilité (C5) | **Naturel** : perte d'une instance des annonces pendant le tir sans interruption ; contrôle arrêté, soumissions conservées puis traitées à son retour (ENF-12). | **Coûteux** : perte d'une instance au milieu d'un paiement, compensation à démontrer. | **Naturel** : consommateur arrêté, événements traités à son retour. |
| Tests (C7) | **Naturel** : unitaires sur des règles déterministes, acceptation sur les AC, autorisation négatifs, charge. | **Coûteux** : dépendent d'un substitut du prestataire, cas concurrents et asynchrones. | **Coûteux** : asynchrones sur deux destinataires, boîte mail de test. |

### Faisabilité

- **A est faisable seul si le périmètre tient.**
  - Sprint 1 (walking skeleton) : la première tranche de A sert de squelette, rien n'est jetable : composant annonces avec lecture publique du catalogue et création d'article authentifiée, déployé avec CI, HTTPS, authentification, allumage et extinction depuis zéro.
  - Sprint 2 : composant contrôle et canal asynchrone, quatre règles, photos, liste « à vérifier », écrans, tests d'acceptation, tir de charge et mise à l'échelle automatique. C'est le sprint le plus chargé du projet.
  - Réduction du risque : si l'expérimentation du canal de messages (J3) ne l'a pas encore déployé sur l'environnement managé, le squelette du sprint 1 peut déjà traverser les deux composants et le canal avec une seule règle, ce qui lève le risque d'intégration le plus élevé avant le sprint 2. Écart à [PILOTAGE.md](../PILOTAGE.md) § 3.4 (« un service minimal ») : arbitrage de Martin.
  - Ordre de coupe si le sprint 2 déborde : d'abord la vague de soumissions en charge (le tir sur la consultation suffit pour C5), puis le confort de l'écran admin (une page minimale suffit), puis les photos multiples (une seule suffit). Le parcours soumission → contrôle → publication → consultation n'est jamais coupé.
- **B ne tient pas dans le sprint 2 sans risque majeur** : prestataire externe (compte de test, secrets, retours asynchrones vers un point d'entrée stable dans un environnement recréé à chaque allumage), chaîne avec compensation et expiration de la réservation, simulateur pour la charge, catalogue à construire quand même.
- **C est faisable mais dépasse le sprint 2** : suivi, préférences, diffusion, canal email et composant fraude, en plus des prérequis injectés ; tests asynchrones sur deux destinataires plus fragiles.
- **Principaux risques de la démo avec A, et parades** :
  1. Délai de mise à l'échelle automatique dans 20 minutes de présentation : tir calibré et répété, seuils de déclenchement fixés à l'avance, vidéo de secours.
  2. Délai du contrôle visible à l'écran (« en contrôle » puis « publié ») : délai borné inscrit dans les AC, présenté comme une propriété voulue du système.
  3. Contraintes de l'environnement de formation AWS (rôle imposé, sessions limitées, crédits) sur le stockage de fichiers, le canal et le fournisseur d'identité : à vérifier en J3, avant tout développement.
  4. Allumage depuis zéro : catégories, plages de prix, comptes et boutiques fictifs recréés par un script rejouable sans effet de bord ; durée d'allumage mesurée avant la soutenance.
  5. Tir de charge faussé par les photos : le tir vise les pages et les données du catalogue, les fichiers ne transitent pas par le composant annonces (à cadrer en D07 et D13).
  6. Coût caché du front (quatre à cinq écrans) : à trancher, les exigences d'accessibilité et d'internationalisation (ENF-26, ENF-30) s'appliquent-elles aux écrans du POC ?
  7. Dérive de périmètre (contrôle « intelligent », traitement d'images, back-office riche) : garde-fous de la recommandation.

### Simplicité

- **A est la candidate la plus simple qui couvre C2, C3, C5 et C7.** C'est le plus petit ensemble qui valide C2 : un front, deux composants applicatifs (annonces, contrôle), un canal asynchrone point à point, une base dont seules les annonces sont propriétaires, un stockage de fichiers, un fournisseur d'identité. Aucun tiers dans le chemin critique, aucune diffusion à plusieurs consommateurs, aucune transaction répartie avec compensation. B ajoute un tiers financier, une chaîne avec compensation et un simulateur de charge ; C ajoute une diffusion, un canal email et un troisième composant presque vide.
- **Pourquoi pas plus simple** : un seul composant avec le contrôle en interne couvrirait le besoin fonctionnel, mais ne validerait pas la communication interservice que C2 cite explicitement. Deux composants est le minimum qui satisfait à la fois le besoin et la grille.
- **Justification de l'asynchrone** : des règles déterministes s'exécutent presque instantanément, donc leur durée ne justifie pas l'asynchrone. Ce qui le justifie : le contrôle est remplaçable par un outil de fraude interne ou acheté, de latence et de disponibilité inconnues (ENF-17) ; aucune soumission n'est perdue si le contrôle est arrêté (ENF-12) ; une vague de soumissions ne dégrade pas la consultation.
- **Complexité à ne pas introduire avec A** (aucun besoin présent) : un composant catalogue séparé des annonces avec copie des données (la lecture se met à l'échelle en répliquant les annonces) ; une base propre au contrôle ou partagée entre les deux composants (le contrôle est sans état, ses règles et plages de prix sont dans sa configuration, le message de soumission porte les données utiles) ; une diffusion à plusieurs consommateurs ; un moteur de règles générique, un moteur de workflow ou un historique construit à partir des événements (quatre règles codées et un état de l'article suffisent : brouillon, en contrôle, publié, refusé, à vérifier) ; une notification temps réel au vendeur ; un traitement d'image (seuls le type et la taille sont vérifiés) ; un back-office au-delà de la liste « à vérifier » et de deux actions.
- **Complexité réelle et peu coûteuse à garder** : un traitement idempotent, car le canal peut livrer deux fois le même message.
- **Variante recommandée : A avec un contrat explicite pour le contrôle.** Les règles « cohérence du prix avec la catégorie » et « statut du vendeur » sont déjà de la détection d'anomalies au sens d'EF-42 (« prix trop élevé, vendeur suspect »). Les messages d'entrée et de sortie du contrôle sont documentés et versionnés comme point de branchement de l'outil de fraude, et un test de contrat vérifie que les deux composants les respectent. EF-42 (partiel) et ENF-17 rejoignent les exigences couvertes. Gain : C3 gagne une exigence MOA explicite sans fonctionnalité nouvelle ; le découpage et l'asynchrone de C2 sont justifiés par une exigence MOA (réponse anticipée à « n'est-ce pas de la sur-architecture ? ») ; C7 gagne un type de test. Coût faible : aucun composant ni écran ajouté.
- **Aucune variante qui ajoute un élément de C n'est utile d'emblée.** La faisabilité de la diffusion, dont la V1 aura besoin (variation de prix → acheteurs et fraude), se valide à moindre coût dans l'expérimentation du canal de messages (C1, D14), hors du POC. Si le sprint 2 finit en avance, l'extension la moins coûteuse est le renvoi au contrôle d'un article dont le prix change (EF-13, EF-41 partiel) : même canal, même composant.

**Verdict architecture** : option A, sous la forme de deux composants applicatifs (les annonces, catalogue compris, et le contrôle de publication) reliés par un canal asynchrone point à point, avec un contrat explicite qui fait du contrôle le point de branchement de l'outil de fraude.

## Décision de Martin

> **« A enrichie »** : mise en vente d'un article avec contrôle automatisé de publication, sous la forme de deux composants (annonces, catalogue compris ; contrôle de publication) reliés par un canal asynchrone point à point, avec :
> - un contrat explicite et versionné pour le contrôle, point de branchement de l'outil de fraude (EF-42 partiel, ENF-17), vérifié par un test de contrat ;
> - la revalidation après modification d'un article publié (ES-18) ;
> - le contrôle fermé par défaut : un article n'est visible qu'après une décision « publié » (ES-31) ;
> - l'upload sécurisé minimal (ES-17).
>
> Choix associés tranchés le même jour :
> - Commission de 5 % : **« Total payé »**, soit prix de l'article + frais de port (ES-13).
> - Séquestre : **« Oui, en V1 »** : fonds conservés chez le prestataire de paiement jusqu'à la confirmation de réception, libération automatique après un délai sans signalement (ES-15, EF-26). Hors POC.
> - i18n et accessibilité dans les écrans du POC : **« Oui, version minimale »** : textes externalisés français/anglais (ENF-30) et contrôle automatisé d'accessibilité en CI (ENF-26).
>
> Retenus par défaut, sans objection de Martin : publicités partenaires en V2 (ES-28 devient un prérequis de leur activation) ; interprétations 1, 3, 5, 6 et 7 du § 4 d'[exigences.md](../00-cadrage/exigences.md), l'admin assurant le rôle de modérateur ; cinq bonnes pratiques de sécurité démontrées dans le POC (HTTPS, authentification et rôles, scan bloquant avec détection de secrets, filtrage des coordonnées ES-16, contrôle d'accès par propriété et par rôle ES-04/ES-03).

Date : 2026-10-09

## Conséquences

- **D02** : part de deux composants applicatifs (annonces, catalogue compris ; contrôle de publication) reliés par un canal asynchrone point à point.
- **D05** : porte le contrat versionné du contrôle (messages d'entrée et de sortie, point de branchement de l'outil de fraude) et le traitement idempotent des messages.
- **D07** : couvre le stockage privé des photos jusqu'à la publication de l'annonce (ES-17).
- **D13** : couvre le test de contrat, le contrôle d'accessibilité automatisé en CI (ENF-26) et le scénario de charge sur la consultation du catalogue.
- **Backlog** : les US de la fonctionnalité (`backlog/US-xxx-*.md`) contiennent les AC de preuve proposés par la sécurité (« Avis sécurité », § 3, « Preuves à prévoir dans les AC »), ainsi que l'AC négatif « un acheteur sans boutique ne peut pas créer d'article ».
- **Exigences** : les choix tranchés sont reportés dans [exigences.md](../00-cadrage/exigences.md) : commission sur le total payé (EF-24), séquestre en V1 hors POC (EF-26), ENF-26 et ENF-30 en version minimale dans le POC, § 4 validé.

## À savoir défendre en soutenance

Questions préparées pour l'option retenue (« A enrichie »).

1. **« Pourquoi pas le paiement, qui porte le modèle économique ? »** La publication est le prérequis de tout le métier. Le paiement par carte repose surtout sur l'intégration d'un prestataire externe : il valide peu de points d'architecture propres à Collector et expose la démonstration à un tiers. Paiement et commission restent au backlog V1 (EF-22 à EF-24) ; leurs risques sont couverts par les exigences de sécurité (ENF-04).
2. **« Qu'est-ce qui est réellement automatisé dans le contrôle, et que se passe-t-il quand il ne sait pas trancher ? »** Des règles déterministes donnent trois issues : publié, refusé avec motif, à vérifier. Seul le reliquat « à vérifier » mobilise l'admin. La part d'articles traités sans intervention humaine se mesure.
3. **« Votre test de charge est-il représentatif ? »** Hypothèse, à étayer : la consultation anonyme domine le trafic d'une marketplace. Le pic modélise l'activité événementielle de Collector (sortie de salon, campagne). Les seuils (ENF-14, ENF-15) sont fixés avant le tir ; la montée est progressive.
4. **« Isoler le contrôle, n'est-ce pas de la sur-architecture ? »** Le besoin vient des exigences, pas d'un principe : contrôle automatisé « le plus possible » et outil de fraude « interne ou acheté », donc remplaçable. Le découpage effectivement retenu est justifié dans D02.
5. **« Comment prouvez-vous que la fonctionnalité répond au besoin de la MOA ? »** Par la traçabilité : citation du contexte → exigence EF → US → AC → test d'acceptation automatisé vert en CI → démonstration du parcours.
6. **« Vos règles s'exécutent en quelques millisecondes : pourquoi de l'asynchrone ? »** Leur durée ne le justifie pas, et ce n'est pas l'argument. Trois raisons : le contrôle est remplaçable par un outil de fraude interne ou acheté, de latence et de disponibilité inconnues (ENF-17) ; aucune soumission n'est perdue si le contrôle est arrêté (ENF-12) ; une vague de soumissions est absorbée sans dégrader la consultation.
7. **« Le paiement est votre premier risque critique et le POC ne le couvre pas ? »** Il est documenté : paiement délégué à un prestataire certifié, sans aucune donnée de carte chez Collector (ES-12) ; confirmation vérifiée et idempotente, réservation atomique de l'objet (ES-14). Il est classé P1 dans le plan de remédiation (`docs/securite/plan-remediation.md`). Le démontrer contre un simulateur aurait affaibli la preuve (« Avis sécurité », § 3).
