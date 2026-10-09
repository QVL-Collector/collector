# Exigences de Collector.shop

Reformulation des exigences du contexte (`contexte_Collector.pdf`, V1 – 04/03/2025). Synthèse du contexte et de la grille : [contexte-et-grille.md](contexte-et-grille.md).
Chaque exigence cite sa source, ou porte la mention « Ajout justifié » : le sujet autorise des exigences ajoutées « pour répondre aux consignes ou les enrichir », en restant cohérent avec le contexte.
Ce document est rattaché à [D01](../decisions/D01-fonctionnalite-poc.md), décidée le 2026-10-09. Les points tranchés par Martin sont au § 4.

## 1. Conventions

### Priorités

| Priorité | Signification |
|---|---|
| V1 | Périmètre de la première version du produit |
| V2 | Version suivante : citée comme telle par le contexte, ou repoussée avec justification |
| Évolution | Évolution annoncée par la direction ; l'architecture V1 doit la rendre possible sans refonte (ENF-21) |

Le **POC n'est pas la V1** : il implémente la seule fonctionnalité retenue par D01, la mise en vente d'un article avec contrôle automatisé de publication. Les autres exigences V1 restent au backlog produit.

### Rôles

| Rôle | Définition |
|---|---|
| Visiteur | Personne non authentifiée |
| Acheteur | Utilisateur inscrit qui achète ; peut aussi être vendeur |
| Vendeur | Utilisateur inscrit qui vend, identifié comme vendeur particulier |
| Admin | Gestionnaire du back-office ; assure aussi la modération (pas de rôle modérateur distinct en V1) |
| Système | Traitement automatisé de la plateforme (pas un rôle utilisateur) |

## 2. Exigences fonctionnelles

### 2.1 Catalogue

| ID | Rôle | Exigence | Priorité | Source |
|---|---|---|---|---|
| EF-01 | Visiteur | Parcourir le catalogue des articles publiés sans être authentifié. | V1 | « le parcours du catalogue ne nécessite pas d'être authentifié » |
| EF-02 | Visiteur | Filtrer le catalogue par catégorie. | V1 | Ajout justifié : les catégories créées par l'admin servent à classer et filtrer le catalogue. |
| EF-03 | Visiteur | Rechercher un article par mots-clés. | V1 | Ajout justifié : un catalogue d'objets uniques est impraticable sans recherche. |
| EF-04 | Visiteur | Consulter la fiche d'un article : photos, description, prix, frais de port, boutique, vendeur identifié comme particulier, note du vendeur. | V1 | « Chaque article publié doit avoir des photos et une description précise… sans oublier le prix et les éventuels frais de ports » |
| EF-05 | Admin | Créer, renommer et désactiver les catégories ; aucun autre rôle ne peut en créer. | V1 | « Les catégories du site sont donc uniquement créées par l'admin du site » |

### 2.2 Compte et espace personnel

| ID | Rôle | Exigence | Priorité | Source |
|---|---|---|---|---|
| EF-06 | Visiteur | S'inscrire ; l'inscription est obligatoire pour acheter ou vendre. | V1 | « L'Inscription est obligatoire pour les clients qui désirent vendre ou acheter un objet » |
| EF-07 | Acheteur, Vendeur | S'authentifier avec un compte unique qui peut cumuler les rôles acheteur et vendeur. | V1 | « Un acheteur peut aussi avoir le rôle de vendeur » |
| EF-08 | Acheteur, Vendeur | Disposer d'un espace dédié : achats et ventes en cours, historique, notifications, accès au chat. | V1 | « ont un espace dédié sur la plateforme dans lequel ils peuvent suivre les achats ou vente en cours » |
| EF-09 | Acheteur | Paramétrer ses centres d'intérêt depuis son profil. | V1 | « Il peut depuis son profil, paramétrer ses centres d'intérêt » |
| EF-10 | Acheteur, Vendeur | Consulter, rectifier et supprimer ses données personnelles et son compte. | V1 | Ajout justifié : plateforme française traitant des données personnelles (RGPD). |

### 2.3 Boutique et article

| ID | Rôle | Exigence | Priorité | Source |
|---|---|---|---|---|
| EF-11 | Vendeur | Créer et gérer plusieurs boutiques virtuelles, en restant identifié comme vendeur particulier. | V1 | « Un vendeur peut créer plusieurs boutiques virtuelles selon ses articles mais il doit clairement être identifié en tant que vendeur particulier » |
| EF-12 | Vendeur | Créer un article dans une de ses boutiques : titre, catégorie, description précise, au moins une photo, prix, frais de port éventuels. L'article est alors soumis au contrôle (EF-16). | V1 | « Chaque article publié doit avoir des photos et une description précise… sans oublier le prix et les éventuels frais de ports » |
| EF-13 | Vendeur | Modifier un article ; toute variation de prix est historisée (ancien prix, nouveau prix, date). | V1 | « En cas de variation du prix, l'information doit être collectée » |
| EF-14 | Vendeur | Retirer un article de la vente tant qu'il n'est pas vendu. | V1 | Ajout justifié : cycle de vie minimal d'une annonce. |
| EF-15 | Acheteur | Suivre un article pour être informé de ses évolutions. | V1 | Ajout justifié : moyen explicite d'exprimer « l'intérêt pour cet article » cité par le contexte. |

### 2.4 Contrôle de publication

| ID | Rôle | Exigence | Priorité | Source |
|---|---|---|---|---|
| EF-16 | Système | Ne proposer un article à la vente qu'après contrôle ; un article non contrôlé n'apparaît pas au catalogue. | V1 | « La mise en ligne d'un article est proposé à la vente qu'après contrôle » |
| EF-17 | Système | Contrôler automatiquement chaque article soumis : complétude de l'annonce, absence de coordonnées personnelles, cohérence du prix avec la catégorie, statut du vendeur. Conclure « publié », « refusé » avec motif, ou « à vérifier ». | V1 | « Le contrôle doit pouvoir être automatisé le plus possible pour éviter l'intervention humaine » |
| EF-18 | Vendeur | Connaître le résultat du contrôle et le motif d'un refus ; corriger et soumettre à nouveau. | V1 | Ajout justifié : sans motif, chaque refus devient une sollicitation manuelle. |
| EF-19 | Admin | Traiter les articles « à vérifier » : publier, ou refuser avec motif. | V1 | Ajout justifié : le contrôle est automatisé « le plus possible », il reste donc un reliquat humain. |
| EF-20 | Admin | Supprimer un article ou suspendre un vendeur qui ne respecte pas la charte de la plateforme. | V1 | « suppressions d'articles ou vendeurs ne respectant pas la charte de la plateforme » |
| EF-21 | Acheteur | Signaler un article non conforme. | V2 | Ajout justifié : complète le contrôle automatisé par la communauté. |

### 2.5 Achat, paiement et commission

| ID | Rôle | Exigence | Priorité | Source |
|---|---|---|---|---|
| EF-22 | Acheteur | Acheter un article publié et le payer par carte bancaire sur la plateforme ; aucun paiement direct entre particuliers. | V1 | « Le paiement se fait à travers le site. Les paiements directs entre acheteurs et vendeurs sont proscrits. Pour la v1 seul le mode de paiement par carte bancaire est prévu » |
| EF-23 | Système | Réserver l'article pendant le paiement ; un objet unique n'est vendu qu'une fois. | V1 | Ajout justifié : objets « rares ou uniques ». |
| EF-24 | Système | Prélever une commission de 5 % sur chaque transaction, calculée sur le total payé (prix de l'article + frais de port) ; le montant dû au vendeur est le total payé moins la commission. | V1 | « L'entreprise prend une commission de 5 % sur chaque transaction » ; base de calcul : D01 (§ 4, point 2) |
| EF-25 | Acheteur, Vendeur | Suivre l'état de chaque transaction (payée, expédiée, reçue) : le vendeur déclare l'expédition, l'acheteur confirme la réception. | V1 | « suivre les achats ou vente en cours » |
| EF-26 | Système | Séquestre : les fonds restent chez le prestataire de paiement jusqu'à la confirmation de réception par l'acheteur, puis le montant dû est reversé au vendeur ; libération automatique après un délai sans signalement. | V1 | « Collector.shop se charge de la garantie de la qualité du vendeur, ses produits » ; séquestre : D01 (§ 4, point 3) |
| EF-27 | Acheteur, Admin | Ouvrir un litige sur une transaction ; l'admin l'arbitre. | V2 | « Collector.shop se charge de la garantie de la qualité du vendeur, ses produits » (interprétation : voir § 4) |
| EF-28 | Acheteur | Payer par un autre moyen que la carte bancaire. | V2 | « Pour la v1 seul le mode de paiement par carte bancaire est prévu » |

### 2.6 Notifications

| ID | Rôle | Exigence | Priorité | Source |
|---|---|---|---|---|
| EF-29 | Acheteur | Être notifié de la mise en ligne d'un nouvel article correspondant à un de ses centres d'intérêt. | V1 | « d'un nouvel article correspondant à un centre d'intérêt de l'acheteur » |
| EF-30 | Acheteur | Créer une alerte sur un article précis recherché et être notifié de sa mise en ligne. | V1 | « la mise en ligne d'un nouvel article particulier » (interprétation : voir § 4) |
| EF-31 | Acheteur | Être notifié d'une variation de prix d'un article suivi (EF-15). | V1 | « Les acheteurs ayant de l'intérêt pour cet article doivent en être informés » |
| EF-32 | Acheteur | Choisir les types de notifications reçus et leur envoi éventuel par email. | V1 | « L'acheteur doit pouvoir régler le type de notification à recevoir » ; « peuvent être aussi envoyées à son adresse email » |
| EF-33 | Acheteur, Vendeur | Consulter la liste de ses notifications dans son espace. | V1 | « Ces notifications sont listées dans son espace particulier » |
| EF-34 | Acheteur, Vendeur | Être notifié des événements de ses transactions et de ses annonces (paiement, expédition, résultat du contrôle). | V1 | Ajout justifié : suivi des achats et ventes en cours. |

### 2.7 Chat modéré

| ID | Rôle | Exigence | Priorité | Source |
|---|---|---|---|---|
| EF-35 | Acheteur, Vendeur | Échanger par chat depuis son espace, au sujet d'un article. | V1 | « Depuis cet espace, un chat est accessible pour discuter entre vendeurs et acheteurs » |
| EF-36 | Système | Détecter et bloquer les coordonnées personnelles (email, numéro de téléphone…) dans le chat et dans les annonces. | V1 | « Le système doit donc empêcher l'échange d'informations personnelles telles que l'email, un numéro de téléphone » |
| EF-37 | Acheteur, Vendeur | Signaler un message. | V1 | Ajout justifié : déclencheur de la modération, évite une lecture exhaustive par l'admin. |
| EF-38 | Admin | Modérer le chat : consulter les conversations signalées, supprimer un message, suspendre un utilisateur. | V1 | « L'admin du site peut avoir un rôle de modération » |

### 2.8 Recommandations

| ID | Rôle | Exigence | Priorité | Source |
|---|---|---|---|---|
| EF-39 | Acheteur | Recevoir des recommandations d'articles selon ses centres d'intérêt. | V1 | « L'acheteur authentifié a accès à des recommandations en fonction de ses centres d'intérêts » |
| EF-40 | Acheteur | Recevoir des recommandations qui tiennent aussi compte de son parcours du catalogue. | V2 | « dans la V2 les recommandations s'appuient aussi sur le parcours du catalogue » |

### 2.9 Fraude et anomalies

| ID | Rôle | Exigence | Priorité | Source |
|---|---|---|---|---|
| EF-41 | Système | Transmettre chaque variation de prix au composant back-office de détection des fraudes. | V1 | « La notification de changement de prix doit être aussi reçu par un composant back-office de détection des fraudes » |
| EF-42 | Système | Soumettre articles, prix et vendeurs à un outil de détection automatique des fraudes et anomalies (prix trop élevé, vendeur suspect). | V1 | « la v1 de l'application supporte l'intégration avec un outil d'automatisation de détection de fraudes ou d'anomalie » |
| EF-43 | Admin | Consulter les alertes de fraude et décider d'une action (retrait d'article, suspension, classement sans suite). | V1 | Ajout justifié : une alerte sans traitement reste sans effet. |

### 2.10 Publicités partenaires

| ID | Rôle | Exigence | Priorité | Source |
|---|---|---|---|---|
| EF-44 | Système | Diffuser automatiquement des publicités ciblées (articles, catégories) sur des sites partenaires. | V2 | « la direction souhaite la capacité d'automatiser autant que possible l'intégration de publicités ciblées sur des sites partenaires » ; priorité V2 : D01 (§ 4, point 4) |
| EF-45 | Visiteur, Acheteur | Accepter ou refuser le ciblage publicitaire. | V2 | Ajout justifié : le ciblage est un traitement de données personnelles soumis au consentement. |

### 2.11 Notation

| ID | Rôle | Exigence | Priorité | Source |
|---|---|---|---|---|
| EF-46 | Acheteur, Vendeur | Noter l'autre partie à l'issue d'une transaction, une seule fois par transaction. | V1 | « notez le vendeur ou l'acheteur » |
| EF-47 | Visiteur | Voir la note moyenne d'un vendeur sur ses boutiques et ses articles. | V1 | Ajout justifié : la note sert la confiance de l'acheteur et la garantie de qualité du vendeur. |

### 2.12 Évolutions annoncées

| ID | Rôle | Exigence | Priorité | Source |
|---|---|---|---|---|
| EF-48 | Acheteur, Vendeur | Vendre et acheter aux enchères. | Évolution | « par exemple un système d'enchère » |
| EF-49 | Acheteur, Vendeur | Vendre des objets lors d'événements collectors diffusés en streaming direct. | Évolution | « des évènements collectors avec vente d'objets en streaming direct » |
| EF-50 | Visiteur, Acheteur | Obtenir des réponses d'un bot de service avant-vente. | Évolution | « bot service avant-vente » |
| EF-51 | Admin (direction) | Analyser les ventes. | Évolution | « ajout de services supports tel que l'analyse des ventes pour la direction » |

### 2.13 Index rôle × domaine

| Domaine | Visiteur | Acheteur | Vendeur | Admin | Système |
|---|---|---|---|---|---|
| Catalogue | EF-01 à EF-04 | — | — | EF-05 | — |
| Compte et espace | EF-06 | EF-07 à EF-10 | EF-07, EF-08, EF-10 | — | — |
| Boutique et article | — | EF-15 | EF-11 à EF-14 | — | — |
| Contrôle de publication | — | EF-21 | EF-18 | EF-19, EF-20 | EF-16, EF-17 |
| Paiement et commission | — | EF-22, EF-25, EF-27, EF-28 | EF-25 | EF-27 | EF-23, EF-24, EF-26 |
| Notifications | — | EF-29 à EF-34 | EF-33, EF-34 | — | — |
| Chat modéré | — | EF-35, EF-37 | EF-35, EF-37 | EF-38 | EF-36 |
| Recommandations | — | EF-39, EF-40 | — | — | — |
| Fraude et anomalies | — | — | — | EF-43 | EF-41, EF-42 |
| Publicités partenaires | EF-45 | EF-45 | — | — | EF-44 |
| Notation | EF-47 | EF-46 | EF-46 | — | — |
| Évolutions | EF-50 | EF-48 à EF-50 | EF-48, EF-49 | EF-51 | — |

## 3. Exigences non fonctionnelles

Rattachement aux caractéristiques et sous-caractéristiques d'**ISO/IEC 25010:2011**, référence citée par la consigne. La révision de 2023 renomme notamment Utilisabilité en « Capacité d'interaction » et Portabilité en « Flexibilité » ; le rattachement reste valable.
Les seuils chiffrés sont retenus comme cibles par D01, à confirmer par la mesure.

### 3.1 Aptitude fonctionnelle

| ID | Sous-caractéristique | Exigence | Vérification | Priorité | Source |
|---|---|---|---|---|---|
| ENF-01 | Exactitude | Prix, frais de port, commission de 5 % et montant dû au vendeur sont calculés au centime, avec une règle d'arrondi unique et documentée. | Tests automatisés sur les cas d'arrondi ; commission + montant dû = montant payé. | V1 | Ajout justifié : transactions financières. |

### 3.2 Sécurité

| ID | Sous-caractéristique | Exigence | Vérification | Priorité | Source |
|---|---|---|---|---|---|
| ENF-02 | Confidentialité, intégrité | Tout échange exposé passe en HTTPS (TLS) ; aucune page ni API en clair. | Requête HTTP redirigée ou refusée ; certificat valide ; configuration TLS analysée. | V1 | Consigne : « HTTPS » ; contexte : « la sécurité est une exigence de premier plan » |
| ENF-03 | Authenticité, confidentialité | Authentification requise pour toute action autre que la consultation du catalogue. Autorisation par rôle (visiteur, acheteur, vendeur, admin) et par propriété : un vendeur n'agit que sur ses boutiques et ses articles. | Tests d'autorisation négatifs automatisés (anonyme, autre vendeur, non-admin) : refus systématique. | V1 | Consigne : « authentification/autorisation » ; « L'Inscription est obligatoire pour les clients qui désirent vendre ou acheter » |
| ENF-04 | Confidentialité | Aucune donnée de carte bancaire n'est saisie, stockée ni transmise par les composants de la plateforme ; le paiement est délégué à un prestataire certifié PCI DSS. | Revue d'architecture ; recherche de données de carte dans les bases et journaux : aucune. | V1 | Ajout justifié : réduit le périmètre de conformité des paiements par carte. |
| ENF-05 | Intégrité, confidentialité | Scan de vulnérabilités automatisé du code, des dépendances et des artefacts déployés à chaque changement, bloquant au-delà d'un seuil de sévérité fixé. | Rapport de scan à chaque exécution ; aucune vulnérabilité critique non traitée à la livraison. | V1 | Consigne : « scan de vulnérabilités » |
| ENF-06 | Confidentialité | Aucun secret (mot de passe, clé, jeton) dans le code, le dépôt ou les journaux. | Détection automatisée de secrets à chaque changement : aucun résultat. | V1 | Ajout justifié : une fuite de secret vaut compromission (voir [PILOTAGE.md](../PILOTAGE.md) § 7). |
| ENF-07 | Confidentialité | Données personnelles minimisées ; mots de passe stockés uniquement sous forme hachée ; effacement sur demande. | Revue des données stockées ; test de suppression de compte. | V1 | Ajout justifié : RGPD. |
| ENF-08 | Non-répudiation, imputabilité | Actions sensibles journalisées avec auteur et horodatage (connexion, paiement, changement de prix, décision de contrôle, modération), sans modification possible par l'utilisateur. | Chaque action sensible produit une entrée de journal vérifiable. | V1 | Ajout justifié : enquêtes de fraude. |
| ENF-09 | Authenticité | Nombre de tentatives d'authentification et débit des API publiques limités. | Au-delà du seuil fixé, les requêtes sont refusées. | V1 | Ajout justifié : plateforme publique exposée au bourrage d'identifiants et à l'aspiration du catalogue. |

### 3.3 Fiabilité et disponibilité

| ID | Sous-caractéristique | Exigence | Vérification | Priorité | Source |
|---|---|---|---|---|---|
| ENF-10 | Disponibilité | Disponibilité mensuelle de 99,5 % pour la consultation du catalogue et l'achat. | Sonde de disponibilité externe. | V1 | Consigne : « garantir la disponibilité et la montée en charge » ; seuil : ajout justifié. |
| ENF-11 | Tolérance aux pannes | Pas de point unique de défaillance sur la consultation et l'achat : la perte d'une instance d'un composant n'interrompt pas le service. | Arrêt volontaire d'une instance pendant un test de charge : service maintenu, erreurs sous le seuil d'ENF-15. | V1 | Consigne : « garantir la disponibilité » |
| ENF-12 | Tolérance aux pannes, récupérabilité | Aucun événement métier perdu (soumission d'article, variation de prix, notification) si son destinataire est momentanément indisponible ; le traitement reprend à son retour. | Destinataire arrêté puis relancé : tous les événements sont traités. | V1 | « La notification de changement de prix doit être aussi reçu par un composant back-office de détection des fraudes » |
| ENF-13 | Récupérabilité | Données sauvegardées ; perte maximale de 24 h et reprise en moins de 4 h. | Restauration testée. | V1 | Ajout justifié : données de transactions financières. |

### 3.4 Efficacité de performance

| ID | Sous-caractéristique | Exigence | Vérification | Priorité | Source |
|---|---|---|---|---|---|
| ENF-14 | Comportement temporel | Consultation du catalogue et d'une fiche article en moins de 500 ms au 95e centile, sous charge nominale. | Test de charge : 95e centile mesuré. | V1 | Ajout justifié : la consigne exige de démontrer la montée en charge. |
| ENF-15 | Capacité | Un pic de trafic (sortie de salon, campagne marketing) est absorbé par ajout automatique de capacité, sans dépasser le seuil d'ENF-14 ni 1 % d'erreurs ; la capacité redescend après le pic. | Test de charge progressif : capacité ajoutée puis retirée, seuils respectés. | V1 | Consigne : « montée en charge » ; contexte : activité événementielle de Collector. |
| ENF-16 | Utilisation des ressources | Hors pic, les ressources allouées reviennent au minimum défini. | Ressources allouées au repos comparées au minimum défini. | V1 | Ajout justifié : start-up financée par financement participatif. |

### 3.5 Compatibilité et interopérabilité

| ID | Sous-caractéristique | Exigence | Vérification | Priorité | Source |
|---|---|---|---|---|---|
| ENF-17 | Interopérabilité, remplaçabilité | L'outil de détection des fraudes, interne ou acheté, se branche via un contrat d'interface documenté et versionné (événements et API) ; le remplacer ne modifie aucune autre fonctionnalité. | Contrat publié ; un outil de substitution passe les tests de contrat. | V1 | « Il n'est pour l'instant pas décidé si cet outil sera développé en interne ou « acheté » » |
| ENF-18 | Interopérabilité, remplaçabilité | Les services externes (paiement, envoi d'emails) sont isolés derrière une interface propre à la plateforme et remplaçables sans impact sur le métier. | Tests avec un substitut de chaque service externe. | V1 | Ajout justifié : maîtrise de la dépendance aux fournisseurs. |
| ENF-19 | Interopérabilité | Les API exposées sont documentées et versionnées ; compatibilité ascendante au sein d'une version. | Contrat d'API publié ; tests de non-régression du contrat. | V1 | Ajout justifié : intégrations de l'outil de fraude et des partenaires. |
| ENF-20 | Interopérabilité | Les échanges avec les sites partenaires utilisent un format de flux standard. | Flux validé par un partenaire. | V2 | « intégration de publicités ciblées sur des sites partenaires » |

### 3.6 Maintenabilité et évolutivité

| ID | Sous-caractéristique | Exigence | Vérification | Priorité | Source |
|---|---|---|---|---|---|
| ENF-21 | Modularité, modifiabilité | Découpage par domaines métier : ajouter une évolution (EF-48 à EF-51) ne demande pas de refonte des domaines existants. | Revue d'architecture : impact d'une évolution limité à de nouveaux composants et aux contrats existants. | V1 | « l'architecture de l'application doit faciliter la mise à jour des fonctionnalités ou l'ajout rapide de nouvelles fonctionnalités » |
| ENF-22 | Modifiabilité | Chaque changement passe par une chaîne d'intégration et de livraison continues avec tests automatisés ; un déploiement peut être annulé. | Historique de la chaîne ; retour arrière démontré. | V1 | Consigne : « Un pipeline CI/CD avec tests automatisés » |
| ENF-23 | Testabilité | Chaque fonctionnalité est livrée avec des tests automatisés dérivés de ses critères d'acceptation. | Chaque AC est relié à un test vert en CI. | V1 | Consigne : « tests d'appels voire des tests d'acceptation » |
| ENF-24 | Analysabilité | La dette technique est mesurée par au moins 4 indicateurs relevés à chaque sprint. | Relevé par sprint ; tendance analysée. | V1 | Consigne, § 1.1.1 : « quatre indicateurs ou métriques » |
| ENF-25 | Analysabilité | Au moins une composante d'observabilité (journaux centralisés, métriques ou traces distribuées) permet de diagnostiquer un incident. | Un incident provoqué est diagnostiqué à partir de cette composante. | V1 | Consigne : « une des composantes de l'observabilité » |

### 3.7 Utilisabilité et accessibilité

Dans le POC, ENF-26 s'applique aux écrans en version minimale : contrôle automatisé d'accessibilité en CI (D01).

| ID | Sous-caractéristique | Exigence | Vérification | Priorité | Source |
|---|---|---|---|---|---|
| ENF-26 | Accessibilité | Interfaces conformes au niveau AA des WCAG 2.1, base du référentiel RGAA. | Audit automatisé et vérification manuelle des parcours principaux : aucune non-conformité bloquante. | V1 | « Collector.shop doit aussi supporter l'internationalisation et l'accessibilité » ; niveau AA : ajout justifié. |
| ENF-27 | Opérabilité | Interfaces utilisables sur mobile et sur ordinateur. | Parcours principaux vérifiés sur deux tailles d'écran. | V1 | Ajout justifié : public de particuliers. |

### 3.8 Portabilité

| ID | Sous-caractéristique | Exigence | Vérification | Priorité | Source |
|---|---|---|---|---|---|
| ENF-28 | Installabilité | Infrastructure et déploiement décrits sous forme de code : l'environnement complet se crée depuis zéro et s'éteint par commande. | Création puis destruction complètes, sans action manuelle. | V1 | Ajout justifié : exploitation allumée puis éteinte, reproductibilité. |
| ENF-29 | Adaptabilité | Un même artefact s'exécute en local et dans l'environnement managé ; seule la configuration externalisée change. | Même artefact déployé dans deux environnements. | V1 | Ajout justifié : supprimer les écarts entre environnements. |

### 3.9 Internationalisation

Rattachement ISO 25010 : portabilité (adaptabilité) et utilisabilité.

Dans le POC, ENF-30 s'applique aux écrans en version minimale : textes externalisés en français et en anglais (D01).

| ID | Sous-caractéristique | Exigence | Vérification | Priorité | Source |
|---|---|---|---|---|---|
| ENF-30 | Adaptabilité | Textes d'interface externalisés et traduisibles ; français par défaut, anglais en seconde langue. | Changement de langue sans modification de code ; aucun texte en dur. | V1 | « Collector.shop doit aussi supporter l'internationalisation » ; seconde langue : ajout justifié. |
| ENF-31 | Adaptabilité | Dates, nombres et montants affichés au format de la langue ; montants stockés avec leur devise ; contenus saisis en Unicode. | Tests d'affichage dans les deux langues ; saisie de caractères non latins. | V1 | « Collector.shop doit aussi supporter l'internationalisation » |

## 4. Points clarifiés

Validé par Martin le 2026-10-09 ([D01](../decisions/D01-fonctionnalite-poc.md)).
« Choix de Martin » : tranché explicitement. « Retenu par défaut » : interprétation proposée, retenue sans objection de Martin.

| # | Point | Ambiguïté de la source | Résolution | Origine |
|---|---|---|---|---|
| 1 | « nouvel article particulier » | Article précis attendu par l'acheteur, ou article d'un vendeur particulier ? | Alerte sur un article précis recherché (EF-30). | Retenu par défaut |
| 2 | Commission de 5 % | Base de calcul (prix seul ou prix + frais de port) et partie qui la supporte non précisées. | Calculée sur le **total payé**, prix de l'article + frais de port ; le montant dû au vendeur est le total payé moins la commission (EF-24). | Choix de Martin |
| 3 | Garantie | « Collector.shop se charge de la garantie de la qualité du vendeur, ses produits » : périmètre non défini. | V1 : contrôle de publication, notation, et **séquestre**, hors POC : fonds conservés chez le prestataire de paiement jusqu'à la confirmation de réception, libération automatique après un délai sans signalement (EF-16, EF-26, EF-46). V2 : litiges outillés (EF-27). | Retenu par défaut ; séquestre : choix de Martin |
| 4 | Publicités partenaires | Aucune version indiquée. | **V2 confirmée**, avec une architecture V1 qui ne l'empêche pas (ENF-21). Leur activation suppose ES-28 ([exigences-securite.md](../securite/exigences-securite.md)). | Retenu par défaut |
| 5 | Rôles | « 2 profils utilisateurs », puis un rôle admin et un parcours sans authentification. | Quatre rôles : visiteur, acheteur, vendeur, admin. L'admin assure aussi la modération : pas de rôle modérateur distinct en V1. | Retenu par défaut |
| 6 | Outil de fraude | Interne ou acheté : non décidé par la direction. | Exigence d'interopérabilité (ENF-17), sans préjuger du choix. Dans le POC, le contrat versionné du contrôle de publication sert de point de branchement. | Retenu par défaut |
| 7 | Seuils chiffrés | Aucun seuil dans le contexte. | ENF-10, ENF-13, ENF-14, ENF-15 : cibles retenues, à confirmer par la mesure. | Retenu par défaut |
| 8 | Accessibilité et internationalisation dans le POC | ENF-26 et ENF-30 s'appliquent-elles aux écrans du POC ? | Oui, en version minimale : contrôle automatisé d'accessibilité en CI (ENF-26) ; textes externalisés en français et en anglais (ENF-30). | Choix de Martin |
