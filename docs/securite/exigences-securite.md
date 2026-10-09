# Exigences de sécurité — analyse de menaces du besoin

| | |
|---|---|
| Statut | Proposé — en attente de validation par Martin |
| Date | 2026-10-09 |
| Mise à jour | 2026-10-09 : alignement sur les décisions de D01 |
| Fonction | Sécurité |
| Critères servis | C8 (principal), C2 (architecture intégrant la sécurité), C7 (tests de sécurité) |
| Sources | Contexte Collector V1 (2025-03-04) ; consigne d'évaluation §1, §1.2.3, §1.3 ; grille individuelle, critère 8 |
| Décisions liées | [D01 — Exigences reformulées et fonctionnalité métier du POC](../decisions/D01-fonctionnalite-poc.md) (décidée le 2026-10-09) |

Ce document analyse les menaces **à partir du besoin**, avant tout choix technique. Il sert à :
1. fixer les exigences de sécurité de la V1 et leur priorité ;
2. préparer l'audit de la V1 et le plan de remédiation (phase 3).

Les points tranchés par Martin dans D01 sont reportés tels quels (voir Synthèse). Tout le reste est une **recommandation**, et les priorités restent à valider par Martin.

## Synthèse

- 57 menaces analysées : **5 Critiques**, 24 Élevées, 26 Modérées, 2 Faibles (risque brut, sans mesure).
- Les 5 risques critiques : fausse confirmation de paiement, accès aux données d'autrui (IDOR), compromission d'un compte admin, contournement de la plateforme via l'échange de coordonnées, vendeur fraudeur (§4.11).
- Deux des cinq sont des **abus métier**, pas des failles techniques : sur une place de marché, la sécurité porte d'abord sur l'intégrité de la transaction et le maintien du paiement sur la plateforme.
- 37 exigences : 23 Must V1, 8 Should V1, 6 Later. Chaque menace Critique ou Élevée applicable à la V1 est couverte par au moins une exigence Must (§5.4).
- Décisions de Martin du 2026-10-09 ([D01](../decisions/D01-fonctionnalite-poc.md)), reportées dans ce document :
  - fonctionnalité du POC « A enrichie » : mise en vente contrôlée, avec upload sécurisé (ES-17), revalidation après modification (ES-18) et contrôle fermé par défaut (ES-31) ;
  - commission calculée sur le total payé, prix + frais de port (ES-13) ;
  - séquestre en V1, hors POC (ES-15) ;
  - publicités partenaires en V2, ES-28 devenant le prérequis de leur activation ;
  - l'admin assure aussi la modération, sans rôle modérateur distinct en V1 (ES-03, ES-20) ;
  - cinq bonnes pratiques de sécurité démontrées dans le POC (§6.1).

## 1. Périmètre et méthode

### 1.1 Périmètre

- **Inclus** : les fonctionnalités V1 du contexte (catalogue, comptes, espaces acheteur et vendeur, boutiques, annonces et photos, contrôle de mise en ligne, chat, notifications et emails, paiement par carte et commission, notes, recommandations, back-office et modération, intégration de la détection de fraude), ainsi que la chaîne de livraison et l'hébergement. La publicité ciblée chez les partenaires est reportée en V2 (D01). Elle reste analysée pour préparer son activation.
- **Exclu** : le SI bureautique de l'entreprise, sauf comme vecteur d'attaque des comptes privilégiés. Les évolutions futures (enchères, streaming, bot avant-vente) seront analysées lors de leur cadrage.
- **V1 et POC** : les exigences visent la V1 mise en production. Le POC (mise en vente d'un article avec contrôle automatisé de publication, « A enrichie », D01) n'en couvre qu'une partie (§6). Les exigences non couvertes seront des constats de l'audit V1, puis des entrées du plan de remédiation (§7).

### 1.2 Méthode

**STRIDE**, appliqué aux frontières de confiance (§3), complété par des **cas d'abus métier**.

| Code | Catégorie | Propriété violée |
|---|---|---|
| S | Usurpation d'identité | Authentification |
| T | Altération | Intégrité |
| R | Répudiation | Traçabilité |
| I | Divulgation d'information | Confidentialité |
| D | Déni de service | Disponibilité |
| E | Élévation de privilège | Autorisation |
| Abus | Usage détourné d'une fonction légitime par un utilisateur authentifié | Règle métier |

Justification :
- STRIDE passe en revue les six propriétés de sécurité sans dépendre d'une technologie. La méthode convient donc à une analyse menée avant les choix techniques.
- STRIDE couvre mal les abus commis par des utilisateurs légitimes (contournement de commission, vendeur fraudeur, faux avis). Ce sont pourtant les risques propres à une place de marché, d'où la catégorie « Abus ».
- Méthodes écartées : une analyse de risque complète de type EBIOS RM serait trop lourde pour ce périmètre. Une checklist de vulnérabilités (OWASP Top 10) est orientée code. Elle servira à l'audit V1, pas à l'analyse du besoin.

Les références citées (STRIDE, CVSS, TLS, CSP, SPF, DKIM, DMARC, OpenID Connect) sont des normes ou des référentiels ouverts, pas des choix de produit.

### 1.3 Échelle de risque (réutilisable pour le plan de remédiation)

**Vraisemblance (V)**

| V | Niveau | Critère |
|---|---|---|
| 1 | Rare | Attaquant expert et ciblé, conditions exceptionnelles |
| 2 | Possible | Compétences moyennes ou condition particulière (accès préalable, erreur de conception précise) |
| 3 | Probable | Attaque courante, outillée, documentée publiquement sur ce type de plateforme |
| 4 | Quasi certaine | Se produira dès l'ouverture : comportement courant des utilisateurs ou attaque automatisée de masse |

**Impact (I)** : on retient le plus élevé des impacts financier, données personnelles, réglementaire, confiance et continuité.

| I | Niveau | Critère |
|---|---|---|
| 1 | Mineur | Gêne ponctuelle, pas de perte financière, pas de donnée personnelle exposée |
| 2 | Modéré | Quelques utilisateurs touchés, perte faible et récupérable, données peu sensibles d'un nombre limité de personnes, indisponibilité de moins d'une heure |
| 3 | Grave | Perte financière significative ou répétée, fuite de données d'un nombre important d'utilisateurs (notification CNIL probable), indisponibilité de plusieurs heures, atteinte visible à la confiance |
| 4 | Critique | Fraude ou fuite massive, compromission totale, sanction lourde ou perte du prestataire de paiement, menace sur la survie de la start-up |

**Matrice risque = V × I**

| V \ I | 1 Mineur | 2 Modéré | 3 Grave | 4 Critique |
|---|---|---|---|---|
| 4 Quasi certaine | 4 Modéré | 8 Élevé | 12 Critique | 16 Critique |
| 3 Probable | 3 Faible | 6 Modéré | 9 Élevé | 12 Critique |
| 2 Possible | 2 Faible | 4 Modéré | 6 Modéré | 8 Élevé |
| 1 Rare | 1 Faible | 2 Faible | 3 Faible | 4 Modéré |

**Niveaux et priorités de remédiation**

| Niveau | Score | Priorité de remédiation | Équivalent pour une vulnérabilité technique |
|---|---|---|---|
| Critique | 12 à 16 | P1 — bloquant avant ouverture publique | CVSS 9,0 à 10 sur un composant exposé |
| Élevé | 8 à 9 | P2 — avant mise en production V1 | CVSS 7,0 à 8,9 sur un composant exposé |
| Modéré | 4 à 6 | P3 — planifié dans les sprints suivants | CVSS 4,0 à 6,9 |
| Faible | 1 à 3 | P4 — accepté, surveillé, réévalué à chaque audit | CVSS inférieur à 4,0 |

Règles d'usage :
- Ce document donne le **risque brut**, c'est-à-dire sans mesure de sécurité. L'audit V1 calculera le **risque résiduel** en tenant compte des mesures constatées.
- Une mesure sans preuve (test, rapport de scan, configuration observée) ne réduit pas le risque.
- À score égal, l'impact le plus élevé passe devant.
- Le CVSS mesure une sévérité technique, pas un risque métier. Une vulnérabilité sur un composant non exposé, ou qui ne touche aucune donnée sensible, peut descendre d'un niveau, avec une justification écrite.
- Une non-conformité légale est au minimum P2.

## 2. Actifs et acteurs

### 2.1 Actifs

Besoins : C confidentialité, I intégrité, D disponibilité, T traçabilité.

| ID | Actif | Contenu | Besoins | Sensibilité |
|---|---|---|---|---|
| A01 | Données de carte bancaire | Numéro, date, cryptogramme | C, I | Critique — ne doivent jamais être traitées par Collector (ES-12) |
| A02 | Transactions et flux financiers | Commandes, montants, frais de port, commission, statuts de paiement, reversements, coordonnées de reversement | I, T | Critique |
| A03 | Revenu de commission | Exclusivité du canal de paiement (5 % par transaction) | I | Critique (seule source de revenu) |
| A04 | Comptes et identités | Identifiants, mots de passe, seconds facteurs, sessions, rôles | C, I | Critique |
| A05 | Données personnelles | Identité, email, téléphone, adresse de livraison, historiques, centres d'intérêt, préférences | C | Élevée (RGPD) |
| A06 | Messages du chat | Conversations privées acheteur-vendeur | C, I, T (preuve en cas de litige) | Élevée |
| A07 | Annonces et photos | Contenu, prix, frais de port, historique des prix, statut de validation, boutiques | I, D | Élevée |
| A08 | Réputation | Notes, avis, statut des vendeurs | I | Élevée (base de la garantie de qualité) |
| A09 | Back-office et règles de contrôle | Catégories, décisions de modération, règles de contrôle automatique, suspensions | I, T | Critique |
| A10 | Journaux et traces d'audit | Événements de sécurité et métier | I, T, C | Élevée |
| A11 | Secrets techniques | Clés des intégrations paiement, fraude et email, clés de signature, accès à l'hébergement, credentials temporaires de l'environnement de formation | C | Critique |
| A12 | Disponibilité du service | Catalogue, achat, chat | D | Élevée |
| A13 | Domaine et réputation d'envoi | Nom de domaine, réputation des emails | I | Élevée (vecteur d'hameçonnage) |
| A14 | Code source et chaîne de livraison | Dépôt, pipeline, artefacts, configuration décrite en code | I | Élevée |

### 2.2 Acteurs

| Acteur | Nature | Accès | Menace portée ou subie |
|---|---|---|---|
| Visiteur | Légitime, anonyme | Catalogue, recherche, inscription | Aspiration du catalogue, reconnaissance avant attaque |
| Acheteur | Légitime, authentifié | Achat, chat, notes, notifications, centres d'intérêt | Peut frauder (carte volée, contestation abusive) ; victime d'hameçonnage |
| Vendeur | Légitime, authentifié (acheteur ayant le rôle vendeur) | Boutiques, annonces, photos, chat, reversements | Victime de prise de compte (reversements détournés) |
| Vendeur fraudeur | Malveillant, avec compte | Droits d'un vendeur | Contrefaçon, non-livraison, contournement de commission, faux avis |
| Admin | Privilégié, interne ; assure aussi la modération (D01) | Back-office : catégories, comptes, articles « à vérifier », signalements, conversations signalées | Cible prioritaire (compte le plus puissant) ; peut devenir initié malveillant |
| Initié malveillant | Privilégié (admin, développeur, exploitant) | Back-office, dépôt, pipeline, hébergement | Abus de privilège, exfiltration, sabotage |
| Attaquant externe | Malveillant, anonyme ou avec compte | Toute surface exposée | Prise de comptes, injection, vol de données, fraude au paiement |
| Robots automatisés | Malveillants ou opportunistes | Points d'entrée publics | Bourrage d'identifiants, faux comptes, spam, aspiration, achat automatisé d'objets rares |
| Prestataire de paiement | Tiers certifié | Reçoit l'acheteur, notifie la plateforme | Ses notifications sont un point d'entrée à authentifier ; indisponibilité |
| Outil de détection de fraude | Tiers ou interne (non décidé) | Reçoit des événements, renvoie des verdicts | Interface à authentifier ; destinataire de données personnelles |
| Service d'envoi d'emails | Tiers | Achemine les notifications | Réputation d'envoi, usurpation du domaine |
| Sites partenaires publicitaires (V2) | Tiers, confiance limitée | Données de ciblage, scripts éventuels | Collecte non consentie, exécution de code tiers |
| Hébergeur | Tiers de confiance | Infrastructure | Erreurs de configuration, localisation des données |

## 3. Frontières de confiance

```
   Visiteurs, acheteurs, vendeurs, attaquants, robots          (Internet, non fiable)
                               |
 ============================ FC1 : entrée publique ============================
 |  Zone publique : catalogue, recherche, inscription, connexion               |
 |      -------------------- FC2 : authentification ---------------------      |
 |      |  Zone authentifiée : espaces, boutiques, annonces, chat, achat |      |
 |      ------------------------------------------------------------------      |
 |      -------------------- FC3 : rôle privilégié ----------------------      |
 |      |  Back-office : catégories, modération, gestion des comptes      |      |
 |      ------------------------------------------------------------------      |
 |  ------------------------- FC8 : zone de données -------------------------  |
 |  |  Bases, photos, journaux, sauvegardes                                  |  |
 |  -------------------------------------------------------------------------  |
 ===============================================================================
       | FC4              | FC5             | FC6                 | FC7
  Prestataire de     Outil de        Envoi d'emails       Partenaires
  paiement           fraude          vers les boîtes      publicitaires (V2)

 FC9  : chaîne de livraison (dépôt, pipeline) et accès d'exploitation -> exécution
 FC10 : contenu d'un utilisateur -> navigateur d'un autre (y compris le back-office)
```

| ID | Frontière | Ce qui la traverse | Principe de contrôle |
|---|---|---|---|
| FC1 | Internet → plateforme | Toutes les requêtes | Chiffrement, limitation de débit, validation des entrées |
| FC2 | Public → espace authentifié | Requêtes d'un utilisateur connecté | Authentification, session, contrôle d'accès par objet |
| FC3 | Utilisateur → back-office | Actions privilégiées | Rôle vérifié côté serveur, second facteur, traçabilité |
| FC4 | Plateforme ↔ prestataire de paiement | Redirection de l'acheteur, notifications de paiement, reversements | Carte jamais chez Collector, notification authentifiée, idempotence |
| FC5 | Plateforme ↔ outil de fraude | Événements (annonce, prix, vendeur, transaction), verdicts | Authentification mutuelle, données minimisées, livraison fiable |
| FC6 | Plateforme → emails | Notifications vers des boîtes externes | Domaine authentifié, contenu utilisateur neutralisé |
| FC7 | Plateforme ↔ partenaires publicitaires (V2) | Données de ciblage, scripts éventuels | Consentement, pseudonymisation, aucun script sur les pages sensibles |
| FC8 | Services → zone de données | Lectures et écritures | Zone non exposée, moindre privilège, chiffrement au repos |
| FC9 | Chaîne de livraison → exécution ; accès d'exploitation | Code, artefacts, configuration, secrets | Revue, scans bloquants, secrets hors dépôt, accès tracés |
| FC10 | Contenu utilisateur → autre utilisateur | Titres, descriptions, photos, messages, noms de boutique | Validation à l'entrée, neutralisation à l'affichage, filtrage des coordonnées |

## 4. Menaces majeures

Risque **brut** (sans mesure). V = vraisemblance, I = impact (§1.3). Les exigences qui traitent chaque menace sont en §5.

### 4.1 Paiement et commission

| ID | STRIDE | FC | Scénario | Impact métier | V | I | Risque |
|---|---|---|---|---|---|---|---|
| M-PAY-01 | T | FC1, FC2 | L'acheteur modifie le prix, les frais de port ou la commission envoyés par son navigateur et paie 1 € un objet affiché 300 €. | Perte financière directe, litige avec le vendeur. | 3 | 3 | 9 Élevé |
| M-PAY-02 | S, T | FC4 | Un attaquant forge le retour « paiement accepté » du navigateur, ou forge ou rejoue une notification de paiement. La commande passe « payée » sans encaissement et le vendeur expédie. | Objet perdu ou remboursé par Collector (garant), attaque répétable à grande échelle. | 3 | 4 | 12 Critique |
| M-PAY-03 | I | FC4, FC8 | Les données de carte transitent par Collector (formulaire maison), sont journalisées ou stockées, puis fuient. | Fraude sur les cartes des clients, conformité PCI-DSS complète exigée, perte du prestataire. | 2 | 4 | 8 Élevé |
| M-PAY-04 | T, I | FC1, FC7, FC10 | Un script injecté (contenu utilisateur, script tiers compromis) remplace le parcours de paiement par une fausse page de saisie de carte. | Vol de cartes sous la marque Collector. | 2 | 4 | 8 Élevé |
| M-PAY-05 | T | FC4 | Notification de paiement reçue deux fois, double clic, ou deux acheteurs qui paient au même moment le même objet unique. | Double débit, double reversement, objet vendu deux fois. | 3 | 2 | 6 Modéré |
| M-PAY-06 | Abus | FC2 | Le vendeur affiche un prix minimal et des frais de port gonflés pour réduire la commission, ou saisit des valeurs négatives ou aberrantes. | Érosion du revenu de commission. | 3 | 2 | 6 Modéré |
| M-PAY-07 | S | FC1, FC4 | Achats avec des cartes volées ; tests de cartes en rafale sur le parcours d'achat. | Impayés et rétrofacturations, frais, risque de résiliation par le prestataire. | 3 | 3 | 9 Élevé |
| M-PAY-08 | S, T | FC2, FC4 | Après prise de contrôle d'un compte vendeur, l'attaquant change les coordonnées de reversement. | Reversements détournés, Collector mis en cause. | 2 | 3 | 6 Modéré |
| M-PAY-09 | R | FC2, FC4 | Un acheteur nie un achat ou une réception, un vendeur nie une vente ; aucune trace horodatée ne permet de trancher. | Remboursements à perte, litiges. | 3 | 2 | 6 Modéré |

**Décision D01 (2026-10-09)** : la commission porte sur le total payé (prix + frais de port). Gonfler les frais de port ne réduit donc plus la commission : ce scénario de M-PAY-06 est **traité par la décision sur l'assiette**. Les valeurs négatives ou aberrantes restent traitées par les bornes d'ES-13.

### 4.2 Comptes et sessions

| ID | STRIDE | FC | Scénario | Impact métier | V | I | Risque |
|---|---|---|---|---|---|---|---|
| M-ACC-01 | S | FC1, FC2 | Bourrage d'identifiants (mots de passe fuités sur d'autres sites) ou force brute sur la connexion. | Prise de contrôle de comptes : achats, arnaques depuis un compte de bonne réputation, accès aux données. | 3 | 3 | 9 Élevé |
| M-ACC-02 | I, E | FC2 | Un utilisateur connecté change l'identifiant d'une commande, d'une conversation, d'un profil ou d'une annonce non publiée dans sa requête, puis lit ou modifie la ressource d'un autre (IDOR). | Fuite massive de données personnelles et de conversations par simple itération, notification CNIL, perte de confiance. | 3 | 4 | 12 Critique |
| M-ACC-03 | S | FC1, FC2 | Vol ou rejeu de session : jeton dans une URL ou un journal, session sans expiration ou non invalidée à la déconnexion, requête forgée depuis un site tiers. | Actions au nom de la victime. | 2 | 3 | 6 Modéré |
| M-ACC-04 | S | FC1, FC6 | Lien de réinitialisation du mot de passe prévisible, réutilisable ou sans expiration. | Prise de contrôle de comptes. | 2 | 3 | 6 Modéré |
| M-ACC-05 | I | FC8 | Vol de la base des comptes alors que les mots de passe sont mal protégés. | Réutilisation des mots de passe ailleurs, notification CNIL, image. | 2 | 4 | 8 Élevé |
| M-ACC-06 | I | FC1 | Les messages diffèrent selon que le compte existe ou non (inscription, connexion, réinitialisation). | Constitution d'une liste de clients pour un hameçonnage ciblé. | 3 | 1 | 3 Faible |
| M-ACC-07 | S, Abus | FC1 | Création automatisée de faux comptes. | Support aux arnaques, au spam et aux faux avis. | 3 | 2 | 6 Modéré |

### 4.3 Chat et fuite de coordonnées (contournement de la plateforme)

| ID | STRIDE | FC | Scénario | Impact métier | V | I | Risque |
|---|---|---|---|---|---|---|---|
| M-CHA-01 | I, Abus | FC2, FC10 | Acheteur et vendeur s'échangent email, téléphone ou pseudo de réseau social, en clair ou masqués (« zéro six… », « arobase », espaces, chiffres en lettres), puis concluent hors plateforme. | Perte de commission ; Collector ne peut plus garantir la transaction ; arnaques hors site imputées à Collector. Pratique explicitement proscrite par le besoin. | 4 | 3 | 12 Critique |
| M-CHA-02 | S | FC2, FC10 | Un fraudeur se présente comme « support Collector » et envoie un lien vers une fausse page de paiement ou de connexion. | Vol d'identifiants ou de cartes sous la marque Collector. | 3 | 3 | 9 Élevé |
| M-CHA-03 | Abus | FC2 | Harcèlement, menaces ou contenus illicites, sans moyen de signalement. | Victimes, responsabilité d'hébergeur, image. | 3 | 2 | 6 Modéré |
| M-CHA-04 | D, Abus | FC1, FC2 | Envoi automatisé de messages en masse à tous les vendeurs. | Spam, dégradation du service. | 3 | 2 | 6 Modéré |

Voir aussi : lecture des conversations d'autrui (M-ACC-02), accès abusif de l'admin dans son rôle de modération (M-BO-03), script injecté dans un message (M-ANN-03).

### 4.4 Upload de photos et contenu des annonces

| ID | STRIDE | FC | Scénario | Impact métier | V | I | Risque |
|---|---|---|---|---|---|---|---|
| M-ANN-01 | T, E | FC1, FC8 | Fichier piégé présenté comme une photo : contenu exécutable, image vectorielle contenant du script, image malformée qui vise le traitement d'image côté serveur. | Exécution de code sur la plateforme, ou script exécuté chez les visiteurs. | 3 | 3 | 9 Élevé |
| M-ANN-02 | I | FC10 | Les métadonnées des photos prises au téléphone contiennent les coordonnées GPS du domicile du vendeur. | Localisation publique de collectionneurs qui détiennent des objets de valeur (risque de cambriolage) ; fuite de donnée personnelle. | 3 | 3 | 9 Élevé |
| M-ANN-03 | T, S | FC10 | Script injecté dans un titre, une description, un nom de boutique, un profil ou un message, puis exécuté chez les lecteurs, admin compris (liste « à vérifier »). | Vol de session (y compris de comptes privilégiés), actions au nom de la victime, redirection vers une fausse page de paiement. | 3 | 3 | 9 Élevé |
| M-ANN-04 | I, Abus | FC10 | Coordonnées placées dans la description, le nom de boutique, ou incrustées dans une photo. | Même contournement que M-CHA-01, visible par tous les acheteurs. | 3 | 3 | 9 Élevé |
| M-ANN-05 | T | FC2 | Après validation, le vendeur remplace les photos, la description ou le prix (appât puis substitution). | Contrôle de mise en ligne contourné ; contrefaçon ou arnaque publiée sous la garantie Collector. | 3 | 3 | 9 Élevé |
| M-ANN-06 | I | FC8, FC10 | Les photos d'annonces refusées ou non validées restent accessibles par une adresse prévisible ; l'upload sert d'hébergement gratuit pour des contenus illicites. | Diffusion de contenus non contrôlés, responsabilité d'hébergeur. | 2 | 2 | 4 Modéré |
| M-ANN-07 | D | FC1, FC8 | Fichiers énormes ou images « bombes » de décompression. | Saturation du stockage et du calcul, surcoût. | 3 | 2 | 6 Modéré |

### 4.5 Back-office et modération

En V1, l'admin assure la gestion du back-office et la modération ; il n'y a pas de rôle modérateur distinct (D01).

| ID | STRIDE | FC | Scénario | Impact métier | V | I | Risque |
|---|---|---|---|---|---|---|---|
| M-BO-01 | S, E | FC3 | Compromission d'un compte admin, qui assure aussi la modération : hameçonnage, mot de passe réutilisé, poste ou wifi partagé de l'incubateur. | Contrôle de la plateforme : annonces frauduleuses validées, vendeurs supprimés, conversations lues, catégories modifiées. | 3 | 4 | 12 Critique |
| M-BO-02 | E | FC3 | Une fonction d'administration est appelable sans le rôle requis, ou l'utilisateur peut modifier son propre rôle via son profil. | Élévation de privilège jusqu'à l'administration. | 2 | 4 | 8 Élevé |
| M-BO-03 | E, I, R | FC3, FC8 | Un admin, dans son rôle de modération, supprime un concurrent, valide ses propres annonces, lit des conversations sans motif ou exporte des données. | Fraude interne, atteinte à la vie privée, litiges. | 2 | 3 | 6 Modéré |
| M-BO-04 | R | FC3 | Un vendeur suspendu ou une annonce retirée est contesté, et il n'existe ni trace ni motif. | Litiges, non-conformité (motivation des décisions de modération). | 3 | 2 | 6 Modéré |

### 4.6 Fraude

| ID | STRIDE | FC | Scénario | Impact métier | V | I | Risque |
|---|---|---|---|---|---|---|---|
| M-FRA-01 | S, R, Abus | FC2, FC4 | Un vendeur encaisse sans expédier, expédie une contrefaçon ou un objet non conforme, puis disparaît. | Collector, garant de la qualité, rembourse ; la garantie perd sa crédibilité. | 4 | 3 | 12 Critique |
| M-FRA-02 | Abus | FC2, FC5 | Prix anormal : très élevé (blanchiment par auto-achat avec une carte volée), très bas (appât vers un contact hors plateforme), ou variation brusque. | Fraude financière, rétrofacturations, image. | 3 | 3 | 9 Élevé |
| M-FRA-03 | S, T | FC2 | Faux comptes, auto-achats ou achats croisés pour gonfler des notes ; notes de représailles. | La réputation, base de la garantie, devient trompeuse. | 3 | 2 | 6 Modéré |
| M-FRA-04 | T, R, D | FC5 | Des événements (variation de prix, nouveau vendeur) sont perdus, non émis ou altérés, ou l'outil est indisponible. | Fraude non détectée, sans que personne ne s'en aperçoive. | 2 | 3 | 6 Modéré |
| M-FRA-05 | S, T | FC5 | Un attaquant envoie à la plateforme un faux verdict « vendeur sain » ou de faux événements. | Détection neutralisée. | 2 | 3 | 6 Modéré |
| M-FRA-06 | I | FC5 | Un outil acheté reçoit plus de données personnelles que nécessaire, ou les traite hors UE sans encadrement. | Non-conformité RGPD. | 2 | 3 | 6 Modéré |

### 4.7 Notifications et emails

| ID | STRIDE | FC | Scénario | Impact métier | V | I | Risque |
|---|---|---|---|---|---|---|---|
| M-NOT-01 | S | FC6 | Emails d'hameçonnage qui usurpent le domaine Collector ou un domaine ressemblant (« votre paiement a échoué, cliquez ici »). | Vol d'identifiants et de cartes, image. | 3 | 3 | 9 Élevé |
| M-NOT-02 | T | FC6, FC10 | Un titre d'annonce contenant un lien ou du balisage est repris tel quel dans la notification « nouvel article ». | Hameçonnage porté par un vrai email Collector. | 2 | 3 | 6 Modéré |
| M-NOT-03 | D, Abus | FC6 | Publication en masse dans une catégorie très suivie, abonnement d'adresses tierces non vérifiées, envois malgré les préférences de l'utilisateur. | Plaintes, domaine d'envoi mis en liste noire, emails de paiement et de réinitialisation non délivrés, non-conformité. | 3 | 2 | 6 Modéré |

### 4.8 Recommandations et publicités partenaires

| ID | STRIDE | FC | Scénario | Impact métier | V | I | Risque |
|---|---|---|---|---|---|---|---|
| M-PUB-01 | I | FC7 | Centres d'intérêt, historique ou email transmis aux partenaires, ou traceurs déposés, sans consentement valable ni information. | Sanction CNIL, atteinte à la vie privée. | 3 | 3 | 9 Élevé |
| M-PUB-02 | T, I | FC7, FC10 | Un script de partenaire est chargé dans les pages de la plateforme et s'exécute avec les droits de la session. | Lecture des données affichées, redirection, collecte non consentie, détournement du paiement (M-PAY-04). | 2 | 4 | 8 Élevé |
| M-PUB-03 | I | FC7 | Des centres d'intérêt révèlent une donnée sensible (objets religieux, politiques ou militaires). | Traitement de données sensibles au sens du RGPD. | 1 | 3 | 3 Faible |

**Décision D01 (2026-10-09)** : les publicités partenaires sont reportées en V2. M-PUB-01, pour sa partie partenaires et traceurs, et M-PUB-02 ne s'appliquent donc pas à la V1. ES-28 est le prérequis de leur activation. Les recommandations fondées sur les centres d'intérêt restent en V1 (M-PUB-03, traitée par ES-26).

### 4.9 Disponibilité

| ID | STRIDE | FC | Scénario | Impact métier | V | I | Risque |
|---|---|---|---|---|---|---|---|
| M-DIS-01 | D | FC1 | Pic de charge légitime : sortie d'une édition limitée, salon Collector, campagne marketing. | Indisponibilité au moment des ventes les plus rentables. | 3 | 3 | 9 Élevé |
| M-DIS-02 | D | FC1 | Déni de service malveillant, volumétrique ou applicatif (recherches coûteuses, uploads, connexions de chat maintenues ouvertes). | Indisponibilité, ventes perdues. | 2 | 3 | 6 Modéré |
| M-DIS-03 | D | FC1, FC9 | Mise à l'échelle sans plafond, déclenchée par une attaque, une boucle ou un oubli d'extinction. | Coût non maîtrisé. Pour le POC : crédit de moins de 100 $ épuisé et environnement coupé. | 2 | 3 | 6 Modéré |
| M-DIS-04 | D | FC4, FC5, FC6 | Indisponibilité du contrôle de publication, du prestataire de paiement, de l'outil de fraude ou de l'envoi d'emails. | Achats ou mises en ligne bloqués, ou validés sans contrôle. | 2 | 2 | 4 Modéré |
| M-DIS-05 | D, I | FC8 | Perte de données (erreur, suppression malveillante, rançongiciel) sans sauvegarde restaurable. | Perte des transactions et des comptes. | 2 | 4 | 8 Élevé |
| M-DIS-06 | Abus | FC1 | Des robots achètent les objets rares dès leur mise en ligne. | Iniquité, frustration des collectionneurs, image. | 2 | 2 | 4 Modéré |

### 4.10 Socle technique : hébergement, chaîne de livraison, journaux

| ID | STRIDE | FC | Scénario | Impact métier | V | I | Risque |
|---|---|---|---|---|---|---|---|
| M-LIV-01 | T, E | FC9 | Un composant tiers ou une image de base embarque une vulnérabilité connue, exploitable avec un outil public. | Compromission d'un service exposé. | 3 | 3 | 9 Élevé |
| M-LIV-02 | I, E | FC9 | Un secret est commité, intégré à une image ou affiché dans un journal du pipeline. | Usage frauduleux des intégrations (paiement, emails) ou de l'hébergement. | 3 | 3 | 9 Élevé |
| M-LIV-03 | I | FC8 | Une erreur de configuration expose à Internet une base, le stockage de fichiers ou un accès d'administration. | Fuite massive. | 2 | 4 | 8 Élevé |
| M-LIV-04 | I, S | FC1 | Trafic non chiffré ou mal chiffré, intercepté sur un réseau partagé (wifi public, incubateur). | Vol d'identifiants et de sessions. | 3 | 3 | 9 Élevé |
| M-LIV-05 | T | FC9 | Poussée directe sur la branche principale, pipeline modifié ou artefact substitué. | Code malveillant en production. | 1 | 4 | 4 Modéré |
| M-LIV-06 | R | Toutes | Les événements de sécurité ne sont pas journalisés. | Incident non détecté, impossible à investiguer ou à prouver. | 3 | 2 | 6 Modéré |
| M-LIV-07 | I | FC8 | Mots de passe, jetons, données de carte ou contenu des messages écrits dans les journaux. | Fuite par les journaux, accès élargi aux données. | 3 | 2 | 6 Modéré |
| M-LIV-08 | T, I, E | FC1, FC8 | Injection dans les requêtes vers les données via la recherche, les filtres ou les formulaires. | Lecture ou modification de la base. | 2 | 4 | 8 Élevé |

### 4.11 Synthèse des risques

| Niveau | Nombre |
|---|---|
| Critique | 5 |
| Élevé | 24 |
| Modéré | 26 |
| Faible | 2 |
| **Total** | **57** |

**Les 5 risques critiques** (score 12, classés à score égal par impact décroissant) :

| Rang | Menace | V × I | Pourquoi critique | Exigences Must qui le traitent |
|---|---|---|---|---|
| 1 | M-PAY-02 Fausse confirmation de paiement | 3 × 4 | Vol d'objets à grande échelle avec Collector comme garant | ES-14, ES-23 |
| 2 | M-ACC-02 Accès aux objets d'autrui (IDOR) | 3 × 4 | Fuite massive de données personnelles et de conversations | ES-04 |
| 3 | M-BO-01 Compromission d'un compte admin | 3 × 4 | Contrôle total de la modération et des comptes | ES-10, ES-03, ES-23 |
| 4 | M-CHA-01 Contournement via l'échange de coordonnées | 4 × 3 | Touche la seule source de revenu et la garantie | ES-16, ES-15 |
| 5 | M-FRA-01 Vendeur fraudeur | 4 × 3 | Remboursements à la charge de Collector, crédibilité de la garantie | ES-15, ES-18, ES-19 |

## 5. Exigences de sécurité V1

### 5.1 Règle de priorisation

- **Must V1** : l'exigence traite au moins une menace Critique ou Élevée applicable à la V1, ou répond à une exigence explicite de la consigne (HTTPS, authentification et autorisation, scan de vulnérabilités), ou à une obligation légale. Elle est requise avant l'ouverture publique de la V1.
- **Should V1** : l'exigence traite des menaces Modérées. Elle est prévue en V1 mais peut glisser d'un sprint sans bloquer l'ouverture.
- **Later** : risque Faible ou résiduel, ou fonctionnalité V2.
- Chaque exigence décrit le **quoi**, pas le **comment**. La solution sera choisie dans les décisions D02 à D13 (§5.5).
- **Colonne POC** :
  - « Démontrée (BPn) » : l'une des cinq bonnes pratiques décidées dans D01, présentée en soutenance (§6.1).
  - « Intégrée » : mesure décidée dans D01, ou déjà prévue par les règles du projet, présente et testée dans le POC sans être présentée comme bonne pratique (§6.2).
  - « Recommandée » : utile au POC, non tranchée (§6.3).
  - « Partielle » : le POC ne couvre qu'une partie de l'exigence.
  - « Documentée » : hors POC. Son absence sera un constat de l'audit V1 et une entrée du plan de remédiation (§6.4).

### 5.2 Exigences

**Socle (consigne et transverse)**

| ID | Exigence | Priorité | Menaces traitées | POC |
|---|---|---|---|---|
| ES-01 | **Chiffrement des échanges** : tout accès externe passe en HTTPS (redirection systématique depuis HTTP, TLS 1.2 minimum, certificat valide renouvelé automatiquement, en-tête HSTS). Les flux internes sont chiffrés ou confinés dans un réseau privé non exposé. | Must (consigne) | M-LIV-04, M-ACC-03 | Démontrée (BP1) |
| ES-02 | **Authentification déléguée et sessions** : authentification confiée à un serveur d'autorisation dédié, avec un protocole standard (type OpenID Connect). Aucun service métier ne stocke ni ne vérifie de mot de passe. Jetons signés, de courte durée, vérifiés par chaque service (signature, expiration, émetteur, audience). Session invalidée à la déconnexion et au changement de mot de passe. Cookies de session `Secure`, `HttpOnly`, `SameSite`. Protection contre les requêtes intersites forgées sur les actions qui modifient l'état. Jamais de jeton dans une URL. | Must (consigne) | M-ACC-01, M-ACC-03, M-ACC-05 | Démontrée (BP2) |
| ES-03 | **Autorisation par rôle côté serveur** : rôles visiteur, acheteur, vendeur et administrateur. L'administrateur assure aussi la modération (D01, [exigences.md](../00-cadrage/exigences.md)). Refus par défaut. Chaque fonction vérifie le rôle. Le rôle provient uniquement du serveur d'autorisation, jamais d'un champ modifiable par l'utilisateur. | Must (consigne) | M-BO-02, M-BO-01 | Démontrée (BP2, BP5) |
| ES-04 | **Contrôle d'accès au niveau objet** : chaque lecture ou modification d'une commande, conversation, profil, adresse ou annonce non publiée vérifie côté serveur que l'utilisateur y est partie prenante. Les identifiants sont non prévisibles, en défense en profondeur seulement (jamais à la place du contrôle). Un test automatisé « B ne peut pas accéder à la ressource de A » existe pour chaque type de ressource. | Must | M-ACC-02, M-ANN-06 | Démontrée (BP5) |
| ES-05 | **Analyse de vulnérabilités dans la chaîne de livraison** : analyse des dépendances, des images, du code (analyse statique orientée sécurité), de la configuration décrite en code, et détection de secrets. Une vulnérabilité critique ou élevée corrigeable bloque la fusion. Toute exception est justifiée et datée. Les rapports sont conservés. Analyse dynamique non intrusive de l'application déployée en fin de sprint. | Must (consigne) | M-LIV-01, M-LIV-02, M-LIV-03, M-LIV-08 | Démontrée (BP3) |
| ES-06 | **Gestion des secrets** : aucun secret dans le dépôt, les images, les journaux ou les sorties du pipeline. Secrets injectés à l'exécution depuis un stockage de secrets, avec des droits minimaux par secret. Procédure de révocation documentée (complète PILOTAGE §7). | Must | M-LIV-02 | Démontrée (BP3) |
| ES-07 | **Exposition minimale de l'hébergement** : seuls les points d'entrée HTTPS publics sont accessibles depuis Internet. Bases, stockage de fichiers et accès d'administration restent privés. Chiffrement au repos. En-têtes HTTP de sécurité (HSTS, CSP, interdiction de deviner le type de contenu, interdiction de l'intégration en cadre). Origines autorisées limitées à la plateforme. Messages d'erreur génériques, sans trace technique. | Must | M-LIV-03, M-LIV-04, M-ANN-03, M-PAY-04 | Recommandée |
| ES-08 | **Validation des entrées et neutralisation des sorties** : chaque entrée est validée côté serveur (type, longueur, format, bornes). Les requêtes vers les données sont paramétrées. Tout contenu utilisateur est encodé selon son contexte d'affichage, sans HTML libre. La politique de sécurité du contenu (CSP) n'autorise que les scripts de la plateforme. | Must | M-ANN-03, M-LIV-08, M-PAY-04, M-PUB-02 | Recommandée (catalogue, liste « à vérifier ») |

**Comptes**

| ID | Exigence | Priorité | Menaces traitées | POC |
|---|---|---|---|---|
| ES-09 | **Protection des parcours de compte** : nombre de tentatives de connexion limité par compte et par origine, avec un délai croissant. Pas de verrouillage définitif, qui permettrait un déni de service sur les comptes légitimes. Politique de mot de passe conforme à la recommandation CNIL, avec refus des mots de passe connus comme compromis. Message identique que le compte existe ou non. Adresse email vérifiée. Lien de réinitialisation aléatoire, à usage unique et de courte validité. Notification de tout changement de mot de passe ou d'email. | Must | M-ACC-01, M-ACC-04, M-ACC-06, M-ACC-07 | Partielle (paramétrage) |
| ES-10 | **Second facteur obligatoire** pour le rôle administrateur, qui assure aussi la modération ; nouvelle authentification avant les actions d'administration à fort impact. | Must | M-BO-01 | Recommandée (si simple paramétrage) |
| ES-11 | **Actions sensibles des vendeurs** : second facteur proposé. Nouvelle authentification et notification avant tout changement des coordonnées de reversement, de l'email ou du mot de passe. Délai de carence avant le premier reversement vers de nouvelles coordonnées. | Should | M-PAY-08, M-ACC-03 | Documentée |

**Paiement et commission**

| ID | Exigence | Priorité | Menaces traitées | POC |
|---|---|---|---|---|
| ES-12 | **Paiement délégué à un prestataire certifié PCI-DSS** : la carte est saisie uniquement sur la page ou le composant hébergé par le prestataire. Aucune donnée de carte ne transite chez Collector, n'y est stockée ni journalisée. Authentification forte du porteur (DSP2). Encaissement, commission et reversements passent par l'offre « place de marché » du prestataire, qui vérifie l'identité des vendeurs avant reversement. Aucun script tiers sur la page qui lance le paiement. | Must | M-PAY-03, M-PAY-04, M-PAY-07, M-FRA-02 | Documentée |
| ES-13 | **Montants calculés côté serveur** : prix, frais de port, total et commission sont recalculés côté serveur à partir des données de référence, jamais repris de la requête. Valeurs bornées (strictement positives, plafonds par catégorie). Assiette de la commission : **total payé, soit prix + frais de port** (décision D01 du 2026-10-09). | Must | M-PAY-01 ; M-PAY-06 (transfert vers les frais de port traité par la décision sur l'assiette, valeurs aberrantes par les bornes) | Documentée |
| ES-14 | **Confirmation de paiement vérifiée et idempotente** : une commande ne passe « payée » que sur une notification serveur à serveur du prestataire dont on a vérifié l'authenticité (signature) et le contenu (montant, devise, référence de commande). Le retour du navigateur n'a aucune valeur de preuve. Le traitement est idempotent : une notification rejouée est sans effet. Un objet unique est réservé de façon atomique pendant le paiement. | Must | M-PAY-02, M-PAY-05 | Documentée |
| ES-15 | **Séquestre** (décidé en V1, hors POC, D01) : les fonds restent chez le prestataire jusqu'à ce que l'acheteur confirme la réception. À défaut, ils sont libérés automatiquement après un délai sans signalement (EF-26). Le reversement est suspendu si l'outil de fraude ou un signalement vise le vendeur. La protection acheteur ne vaut que pour les paiements sur la plateforme, et l'interface le rappelle. | Must (décidé, D01) | M-FRA-01, M-CHA-01, M-PAY-09 | Documentée |

**Chat et annonces**

| ID | Exigence | Priorité | Menaces traitées | POC |
|---|---|---|---|---|
| ES-16 | **Filtrage des coordonnées** dans le chat, les titres, descriptions, noms de boutique et profils : détection des emails, numéros de téléphone, liens externes et variantes masquées (espaces, lettres, chiffres en toutes lettres), avec un coût de détection borné (longueur de texte limitée). Dans le chat, le passage détecté est masqué avec un avertissement plutôt que bloqué, ce qui limite l'effet des faux positifs. Dans une annonce, le contrôle de publication la refuse avec un motif générique (« coordonnées détectées »), que le vendeur corrige avant de soumettre à nouveau (D01, EF-17, EF-36). Récidive signalée à l'admin. Liens non cliquables. Chat limité au texte en V1. Utilisateurs informés de cette analyse automatique (transparence RGPD). | Must | M-CHA-01, M-CHA-02, M-ANN-04 | Démontrée (BP4) pour les annonces ; chat documenté |
| ES-17 | **Upload sécurisé** : seuls les formats d'image matriciels courants sont acceptés, vérifiés sur le contenu et non sur l'extension ; les formats vectoriels sont refusés. Taille et dimensions maximales. Ré-encodage systématique, qui supprime les métadonnées, dont la géolocalisation. Nom de fichier généré par le serveur. Fichiers servis comme contenu non exécutable, sans cookie de session. Une photo n'est publique qu'après validation de l'annonce. | Must | M-ANN-01, M-ANN-02, M-ANN-06, M-ANN-07 | Intégrée (D01) |
| ES-18 | **Contrôle de mise en ligne et revalidation** : toute annonce est contrôlée avant publication. Les règles automatiques passent d'abord (prix hors de la fourchette de la catégorie, vendeur récent avec des annonces de forte valeur, coordonnées détectées), avec une revue de l'admin en cas de doute. Toute modification substantielle (photos, description, catégorie, prix au-delà d'un seuil) repasse par le contrôle. Chaque variation de prix est historisée et transmise à la détection de fraude. | Must (exigence du besoin) | M-ANN-05, M-ANN-04, M-FRA-01, M-FRA-02 | Intégrée (D01), sauf historisation des prix et transmission à la fraude |

**Modération et fraude**

| ID | Exigence | Priorité | Menaces traitées | POC |
|---|---|---|---|---|
| ES-19 | **Signalement et motivation** : bouton de signalement sur les annonces, boutiques, profils et messages. Tout retrait ou suspension est motivé auprès de l'utilisateur concerné (obligations des hébergeurs, §5.3). | Must (obligation légale) | M-CHA-03, M-BO-04, M-FRA-01 | Documentée |
| ES-20 | **Moindre privilège au back-office** : en V1, l'admin assure la gestion et la modération, sans rôle modérateur distinct (D01). Les droits de modération (articles « à vérifier », signalements, conversations signalées) et d'administration (catégories, comptes) sont toutefois modélisés comme des permissions distinctes du rôle admin. La seule modération pourra ainsi être confiée à un futur salarié sans refonte. Une conversation privée n'est accessible que sur signalement, et cet accès est tracé. Motif obligatoire pour tout retrait. Back-office séparé de l'espace public et non référencé. | Should | M-BO-03, M-BO-01 | Documentée |
| ES-21 | **Intégration fraude fiable et authentifiée** : les événements (nouvelle annonce, variation de prix, nouveau vendeur, transaction) sont émis sans perte silencieuse et peuvent être rejoués. L'interface est authentifiée dans les deux sens, et seuls les verdicts de l'outil authentifié sont acceptés. Les données transmises sont minimisées et couvertes par un contrat de sous-traitance. Le comportement en cas d'indisponibilité de l'outil est défini (par défaut : mise en attente, jamais d'approbation automatique). | Should | M-FRA-04, M-FRA-05, M-FRA-06, M-DIS-04 | Documentée (le contrat versionné du contrôle, D01, la prépare) |
| ES-22 | **Réputation liée aux transactions** : une seule note par partie et par transaction réellement conclue. Les signaux de faux comptes (auto-achat, même moyen de paiement) sont transmis à la détection de fraude. | Should | M-FRA-03, M-ACC-07 | Documentée |

**Traçabilité**

| ID | Exigence | Priorité | Menaces traitées | POC |
|---|---|---|---|---|
| ES-23 | **Journal d'audit** : événements horodatés, avec leur auteur, pour les connexions réussies et échouées, les changements de rôle, les actions d'administration et de modération (avec motif), les décisions du contrôle de publication, les étapes de commande et de paiement, les variations de prix et les changements de coordonnées de reversement. Journal centralisé, non modifiable par les acteurs journalisés. Ni mot de passe, ni jeton, ni donnée de carte, ni contenu de message. Durée de conservation définie (repère CNIL : 6 mois à 1 an pour les journaux de sécurité). | Must | M-PAY-02, M-PAY-09, M-BO-01, M-BO-03, M-BO-04, M-LIV-06, M-LIV-07 | Recommandée (composante d'observabilité exigée) |

**Notifications**

| ID | Exigence | Priorité | Menaces traitées | POC |
|---|---|---|---|---|
| ES-24 | **Emails authentifiés et contenu sûr** : domaine d'envoi authentifié par les mécanismes standards (SPF, DKIM, DMARC avec une politique durcie progressivement). Contenu utilisateur échappé et tronqué dans les gabarits, sans aucun lien fourni par un utilisateur. Les emails ne demandent jamais de mot de passe ni de paiement, et les utilisateurs en sont informés. Envoi uniquement vers une adresse vérifiée. | Must | M-NOT-01, M-NOT-02, M-CHA-02 | Documentée |
| ES-25 | **Maîtrise des envois** : plafonds d'envoi par destinataire et au global, regroupement des notifications, respect des préférences, désinscription en un clic, consentement préalable pour les notifications à visée commerciale. | Should | M-NOT-03 | Documentée |

**Données personnelles et publicité**

| ID | Exigence | Priorité | Menaces traitées | POC |
|---|---|---|---|---|
| ES-26 | **Socle RGPD** : registre des traitements. Information des personnes, y compris sur l'analyse automatique des messages et les recommandations. Minimisation : l'adresse de livraison n'est communiquée au vendeur qu'après confirmation du paiement. Aucun traceur non essentiel en V1, les publicités partenaires étant en V2 (D01). Catégories d'intérêt choisies pour ne pas révéler de donnée sensible. Durées de conservation définies (comptes inactifs, messages, journaux). Contrats avec les sous-traitants (paiement, fraude, emails, hébergeur). Données hébergées dans l'UE, ou transfert encadré. Procédure de notification d'une violation à la CNIL sous 72 h. | Must (obligation légale) | M-FRA-06, M-PUB-01, M-PUB-03, M-LIV-07 | Documentée |
| ES-27 | **Droits des personnes outillés** : export des données ; suppression ou anonymisation du compte, en conservant les pièces comptables obligatoires. | Should | Conformité | Documentée |
| ES-28 | **Publicité ciblée et traceurs**, prérequis à l'activation des publicités partenaires, reportées en V2 (D01) : aucun traceur non essentiel et aucune transmission aux partenaires sans consentement préalable, qu'on peut refuser aussi simplement qu'accepter et retirer à tout moment. Les données transmises se limitent à des segments d'intérêt pseudonymisés, jamais l'email ni l'identité. Aucun script de partenaire sur les pages authentifiées, de paiement ou du back-office. | Later (V2, prérequis d'activation, D01) | M-PUB-01, M-PUB-02, M-PAY-04 | Documentée |

**Disponibilité**

| ID | Exigence | Priorité | Menaces traitées | POC |
|---|---|---|---|---|
| ES-29 | **Limitation de débit et maîtrise de la charge** : limites par origine et par compte sur la connexion, l'inscription, la recherche, l'upload, l'envoi de messages et le lancement d'un paiement. Tailles de requête et délais bornés. Mise à l'échelle automatique avec un minimum, un maximum et une alerte de coût. Tests de charge avec des objectifs chiffrés (débit, latence, taux d'erreur) avant la mise en production. | Must | M-DIS-01, M-DIS-02, M-DIS-03, M-PAY-07, M-CHA-04, M-ACC-01 | Recommandée (observable pendant les tests de charge) |
| ES-30 | **Sauvegardes** chiffrées, hors de portée des accès d'exploitation courants, avec une restauration testée avant l'ouverture. Perte de données maximale admissible et durée de reprise définies. | Must | M-DIS-05 | Documentée |
| ES-31 | **Mode dégradé et contrôle fermé par défaut** : un article n'est visible qu'après une décision « publié » (D01). Si le contrôle de publication, la détection de fraude, le paiement ou l'envoi d'emails est indisponible ou en retard, la plateforme met en attente (mise en ligne, commande, notification), puis reprend sans perte au retour du composant (ENF-12). Jamais de validation par défaut. | Should | M-DIS-04, M-FRA-04 | Intégrée (D01), pour le contrôle de publication |

**Chaîne de livraison**

| ID | Exigence | Priorité | Menaces traitées | POC |
|---|---|---|---|---|
| ES-32 | **Protection de la chaîne de livraison** : branche principale protégée, fusion seulement après revue et CI verte (déjà prévu, PILOTAGE §6). Droits du pipeline limités à son environnement cible. Artefact construit une seule fois, identifié par son empreinte et promu tel quel. | Should | M-LIV-05 | Intégrée (PILOTAGE §6) |

**Later**

| ID | Exigence | Priorité | Menaces traitées |
|---|---|---|---|
| ES-33 | Protection anti-robots adaptative sur l'inscription et l'achat : vérification humaine en cas de volume anormal, limite d'achats simultanés par compte sur les objets rares. | Later | M-ACC-07, M-DIS-06 |
| ES-34 | Détection avancée des contournements : texte incrusté dans les photos, apprentissage des nouvelles formes de masquage à partir des cas modérés. | Later | M-ANN-04, M-CHA-01 (risque résiduel) |
| ES-35 | Test d'intrusion externe avant l'ouverture à grande échelle ; point de contact public pour signaler une vulnérabilité. | Later | Transverse |
| ES-36 | Double validation des actions d'administration à fort impact (suppression d'un vendeur qui a des fonds en séquestre, changement des règles de commission). | Later | M-BO-03 |
| ES-37 | V2, recommandations fondées sur le parcours du catalogue : consentement spécifique, et analyse d'impact (AIPD) à évaluer avant le lancement. | Later | M-PUB-01 |

**Décompte** : 23 Must V1 (ES-01 à ES-10, ES-12 à ES-19, ES-23, ES-24, ES-26, ES-29, ES-30), 8 Should V1 (ES-11, ES-20, ES-21, ES-22, ES-25, ES-27, ES-31, ES-32), 6 Later (ES-28, ES-33 à ES-37).

### 5.3 Points réglementaires

Lecture volontairement minimale. **À confirmer avec un juriste avant toute mise en production réelle.**

| Texte | Pourquoi Collector est concerné | Traduction | Non retenu en V1 |
|---|---|---|---|
| RGPD | Données personnelles des acheteurs et vendeurs, profilage d'intérêts | ES-26, ES-27, ES-16 (transparence), ES-21 (sous-traitance) | AIPD formelle : à évaluer avant d'activer la publicité ciblée et avant la V2 (ES-37) |
| Traceurs (art. 82 de la loi Informatique et Libertés, lignes directrices CNIL) | Mesure d'audience ; publicité ciblée en V2 (D01) | V1 : aucun traceur non essentiel (ES-26). V2 : ES-28, prérequis d'activation des publicités | — |
| PCI-DSS | Paiement par carte | ES-12. Déléguer la saisie de la carte réduit le périmètre au questionnaire d'auto-évaluation le plus léger. **Le périmètre ne devient pas nul** : la plateforme reste responsable de l'intégrité de la page qui renvoie vers le prestataire (ES-07, ES-08 ; ES-28 quand les publicités seront activées). | Certification propre de Collector |
| DSP2 | Paiement par carte ; encaissement pour le compte des vendeurs, qui est un service de paiement réglementé | ES-12 : authentification forte, encaissement et reversements via l'offre place de marché d'un prestataire agréé ; ES-15 : fonds conservés chez le prestataire | Agrément propre d'établissement de paiement (sur-dimensionné) |
| DSA et LCEN | Hébergement de contenus de tiers (annonces, photos, messages) | ES-19 (signalement, motivation des décisions). ES-23 : conservation des données d'identification des auteurs de contenus, durée légale à confirmer | Obligations renforcées des plateformes de taille moyenne et plus (système interne de contestation, rapports de transparence) : exemption des petites entreprises à confirmer |
| Prospection électronique (CPCE, art. L34-5) | Notifications à visée commerciale | ES-25 | — |

### 5.4 Contrôle de couverture

- Chaque menace **Critique** ou **Élevée** applicable à la V1 est traitée par au moins une exigence **Must V1**. La vérification a été faite ligne à ligne entre les §4 et §5.2.
- **M-PUB-01** (partie partenaires et traceurs) et **M-PUB-02** sont sans objet en V1 : les publicités partenaires sont reportées en V2 (D01). ES-28 est le prérequis de leur activation. En V1, ES-26 (aucun traceur non essentiel, minimisation) couvre M-PUB-01, et ES-08 (scripts limités à ceux de la plateforme) couvre M-PUB-02.
- **M-PAY-06** : le scénario de transfert de valeur vers les frais de port est traité par la décision sur l'assiette de la commission (D01, ES-13). Les valeurs aberrantes restent couvertes par les bornes d'ES-13.
- Chaque menace **Modérée** est traitée par au moins une exigence Must ou Should.
- Les deux menaces **Faibles** sont couvertes par des Must : M-ACC-06 par ES-09, M-PUB-03 par ES-26.
- Risques résiduels connus, à suivre dans l'audit V1 :
  - le filtrage des coordonnées (ES-16) ne sera jamais exhaustif. Le séquestre (ES-15, décidé) rend le paiement hors plateforme moins intéressant, et ES-34 traite le résiduel ;
  - la fraude vendeur (M-FRA-01) reste possible, mais son coût est borné par le séquestre.

### 5.5 Entrées pour les décisions

| Décision | Exigences à prendre en compte |
|---|---|
| D01 Exigences et fonctionnalité du POC | **Décidée le 2026-10-09** ([D01](../decisions/D01-fonctionnalite-poc.md)) : « A enrichie ». Reportée dans ES-03, ES-13, ES-15, ES-16, ES-17, ES-18, ES-20, ES-26, ES-28, ES-31 et en §6 |
| D02 Architecture cible | Back-office séparé de l'espace public (ES-20), isolation du contenu utilisateur et des photos (ES-17), contrôle de publication fermé par défaut (ES-31), parcours de paiement et séquestre (ES-14, ES-15) |
| D03 Hébergement et orchestration | Exposition minimale (ES-07), localisation des données (ES-26), plafonds de mise à l'échelle (ES-29), sauvegardes (ES-30) |
| D05 Communication interservice | Canal asynchrone soumission → contrôle sans perte et fermé par défaut (ES-31, ENF-12), événements fiables vers la détection de fraude (ES-21), idempotence (ES-14) |
| D06 Authentification et autorisation | ES-02, ES-03, ES-04, ES-09, ES-10, ES-11 ; permissions de modération et d'administration au sein du rôle admin (ES-20) |
| D07 Données | Photos non publiques avant validation (ES-17), chiffrement au repos (ES-07), sauvegardes (ES-30) |
| D08 Exposition, TLS, domaine | ES-01, ES-07, domaine d'envoi des emails (ES-24) |
| D09 Observabilité | ES-23 |
| D10 CI/CD et portes DevSecOps | ES-05, ES-06, ES-32 |
| D11 IaC et garde-fou coût | Configuration analysée (ES-05, ES-07), plafonds et alerte de coût (ES-29) |
| D12 Indicateurs | Indicateurs sécurité candidats : vulnérabilités critiques et élevées ouvertes, délai de correction, part des exigences Must prouvées par un test automatisé |
| D13 Stratégie de test et charge | Tests d'autorisation négatifs (ES-03, ES-04), tests de charge (ES-29), vague de soumissions avec photos (ES-16, ES-17, ES-31) |

## 6. Sécurité intégrée au POC (critère 8)

Décision de Martin du 2026-10-09 ([D01](../decisions/D01-fonctionnalite-poc.md)) : la fonctionnalité du POC est la mise en vente d'un article avec contrôle automatisé de publication (« A enrichie »). Le POC démontre **cinq bonnes pratiques** de sécurité, là où le critère 8 en demande au moins deux.

### 6.1 Les cinq bonnes pratiques démontrées (D01)

Pour chacune : les AC à reprendre dans les US, le test automatisé en CI et la démonstration en soutenance. Les AC ci-dessous sont des pistes : leur rédaction définitive appartient aux US.

**BP1 — HTTPS (ES-01)**
- Menaces : M-LIV-04, M-ACC-03.
- AC : une requête HTTP vers une page ou une API est redirigée vers HTTPS ou refusée ; aucune page ni API n'est servie en clair (ENF-02).
- Test automatisé : contrôle après déploiement dans le pipeline. Il vérifie la redirection, le refus d'une version de TLS obsolète, la validité du certificat et la présence de l'en-tête HSTS. Le rapport d'analyse TLS est archivé.
- Démonstration : requête HTTP redirigée ; certificat et en-têtes affichés.

**BP2 — Authentification et rôles (ES-02, ES-03)**
- Menaces : M-ACC-01, M-ACC-03, M-ACC-05, M-BO-02.
- AC : la consultation du catalogue ne demande pas de compte ; toute autre action exige une authentification ; un visiteur ne peut pas créer d'article (ENF-03).
- Test automatisé : un appel sans jeton, avec un jeton expiré ou avec une signature invalide est refusé ; le même appel est accepté avec un jeton valide et le bon rôle.
- Démonstration : connexion par le serveur d'autorisation ; la même requête est refusée sans jeton, acceptée avec.

**BP3 — Scan bloquant avec détection de secrets (ES-05, ES-06)**
- Menaces : M-LIV-01, M-LIV-02, M-LIV-03, M-LIV-08.
- AC (US d'infrastructure) : une vulnérabilité critique ou élevée corrigeable, ou un secret détecté, fait échouer le pipeline et bloque la fusion (ENF-05, ENF-06).
- Test automatisé : dépendances, images, code, configuration et secrets analysés à chaque exécution ; rapports archivés.
- Démonstration : sur une branche de démonstration, une dépendance vulnérable connue et un secret fictif rendent le pipeline rouge, alors que `main` reste vert.

**BP4 — Filtrage des coordonnées dans les annonces (ES-16)**
- Menaces : M-ANN-04 ; mécanisme de M-CHA-01, que le chat réutilisera.
- AC : un article dont le titre, la description ou le nom de boutique contient un email, un numéro de téléphone ou un lien externe, en clair ou masqué (« zéro six… », « arobase », espaces), est refusé avec le motif « coordonnées détectées ». Une fois corrigé, il peut être soumis à nouveau (EF-17, EF-18, EF-36).
- Test automatisé : jeu de cas positifs (variantes masquées) et négatifs (dates, dimensions, références d'édition) ; temps de détection borné sur un texte de longueur maximale.
- Démonstration : une description « jean point dupont arobase… » est refusée avec son motif, puis l'article est publié après correction.

**BP5 — Contrôle d'accès par propriété et par rôle (ES-04 avec ES-03)**
- Menaces : M-ACC-02, M-ANN-06, M-BO-02.
- AC : un vendeur ne consulte ni ne modifie l'article ou le brouillon d'un autre vendeur ; seul l'admin tranche un article « à vérifier » ; un article non publié et ses photos sont invisibles des autres utilisateurs (ENF-03, EF-19).
- Test automatisé : tests négatifs par type de ressource et par fonction, tous refusés : vendeur B sur l'article du vendeur A, vendeur sur la décision admin, anonyme sur un brouillon ou une photo non publiée.
- Démonstration : la même requête est acceptée avec le jeton du vendeur A, puis refusée avec celui du vendeur B.

### 6.2 Mesures intégrées au POC (D01)

Ces mesures ne sont pas présentées comme bonnes pratiques. Elles traitent les risques que la fonctionnalité introduit dans le POC et sont testées en CI.

| Mesure | Exigence | Menaces | Preuve attendue (AC et test automatisé) |
|---|---|---|---|
| Upload sécurisé minimal | ES-17 | M-ANN-01, M-ANN-02, M-ANN-06, M-ANN-07 | Refus d'un fichier dont le contenu n'est pas une image, même avec une extension d'image ; refus d'une image vectorielle ; refus d'un fichier au-delà de la taille maximale ; une photo contenant des coordonnées GPS fictives est publiée sans métadonnées ; nom de fichier généré par le serveur ; photo d'un article non publié inaccessible sans droit |
| Contrôle avant publication et revalidation après modification | ES-18 | M-ANN-05, M-FRA-02, M-FRA-01 (en partie) | Règles déterministes : un prix hors de la plage de la catégorie donne « à vérifier ». Si les photos, la description, la catégorie ou le prix au-delà du seuil d'un article publié sont modifiés, l'article est retiré du catalogue et renvoyé « en contrôle » |
| Contrôle fermé par défaut | ES-31 | M-DIS-04, M-FRA-04 | Composant de contrôle arrêté : l'article soumis reste invisible. Au redémarrage, la soumission est traitée sans perte (ENF-12) |

Le contrat versionné du contrôle, prévu par D01 comme point de branchement de l'outil de fraude, prépare ES-21. Il ne prouve pas l'authentification de l'interface, qui restera à faire quand l'outil sera branché.

### 6.3 Compléments recommandés, non tranchés

- **ES-08, neutralisation du contenu affiché**, au catalogue et dans la liste « à vérifier » de l'admin. Cette liste affiche par construction les annonces les plus suspectes (M-ANN-03, vecteur de M-BO-01). Test : un titre contenant du balisage s'affiche comme du texte.
- **ES-10, second facteur pour l'admin**, si un simple paramétrage du serveur d'autorisation le permet (M-BO-01).
- **ES-07, exposition minimale** : photos et données privées, seuls les points d'entrée HTTPS publics ouverts (M-LIV-03).
- **ES-23, journal des décisions** du contrôle et de l'admin, avec leur motif, sur la composante d'observabilité exigée (M-BO-04, M-LIV-06).
- **ES-29, limitation de débit** sur le catalogue et l'upload, observable pendant les tests de charge (M-DIS-01, M-DIS-02).

### 6.4 Hors POC

Ces exigences sont documentées et seront auditées comme des écarts en phase 3 :
- paiement et commission (ES-12 à ES-14) ;
- séquestre (ES-15) ;
- filtrage dans le chat (partie chat d'ES-16) ;
- signalement (ES-19) ;
- emails (ES-24) ;
- socle RGPD (ES-26) ;
- sauvegardes (ES-30).

Le premier risque critique, **M-PAY-02**, n'est donc pas couvert par le POC. Il deviendra une action P1 justifiée du plan de remédiation.

## 7. Lien avec la phase 3 (audit V1 et plan de remédiation)

### 7.1 Preuves à collecter pendant le développement

| Preuve | Produite par | Quand | Exigences | Usage dans l'audit V1 |
|---|---|---|---|---|
| Rapports d'analyse des dépendances et des images | Pipeline | Chaque exécution, rapports conservés | ES-05 | Inventaire des vulnérabilités connues (CVSS), tendance, délai de correction |
| Rapport d'analyse statique orientée sécurité | Pipeline | Chaque exécution | ES-05, ES-08 | Failles de code (injection, neutralisation des sorties) |
| Détection de secrets, historique compris | Pipeline, plus un passage complet avant l'audit | Chaque exécution | ES-06 | Absence de secret, ou incident et révocation |
| Analyse de la configuration décrite en code | Pipeline | Chaque modification d'infrastructure | ES-07, ES-29 | Exposition, chiffrement au repos, plafonds de mise à l'échelle |
| Analyse dynamique non intrusive de l'application déployée | Environnement déployé | Fin de sprint | ES-01, ES-07, ES-08 | Configuration TLS, en-têtes, messages d'erreur, points d'entrée |
| Résultats des tests de sécurité automatisés | CI | Chaque exécution | ES-02, ES-03, ES-04, ES-16, ES-17, ES-18, ES-31 | Refus sans jeton, refus avec le mauvais rôle, refus sur la ressource d'autrui, coordonnées détectées, fichier non image refusé, métadonnées supprimées, revalidation après modification, contrôle fermé par défaut (§6.1, §6.2) |
| Résultats des tests de charge | Campagne dédiée | Sprint 2, puis avant l'audit | ES-29, ES-09, ES-31 | Voir §7.2 |
| Extraits de journaux | Observabilité | Fin de sprint | ES-23 | Présence des événements de sécurité, absence de données sensibles |
| Inventaire des points d'entrée exposés | Environnement déployé | Fin de sprint | ES-07 | Surface d'attaque réelle, comparée à la surface attendue |
| Registre des failles | `docs/audits/sprint-N-securite.md` | Fin de sprint | Toutes | Historique, statut, revérification |
| Matrice exigence → preuve → statut | Audit V1 | Jalon J6 | Toutes | Point de départ du plan de remédiation |

### 7.2 Ce que les tests de charge apportent à la sécurité

- **Point de saturation et ressource limitante** (base, stockage, connexions, calcul) : c'est la cible d'un déni de service applicatif (M-DIS-02).
- **Comportement au-delà du seuil** : refus propres (code de limitation) ou bien erreurs serveur, traces techniques exposées, dégradation durable (ES-07, ES-29).
- **Efficacité de la limitation de débit** sur la connexion, la recherche et l'upload (ES-29).
- **Rafale de connexions échouées** : la protection doit ralentir l'attaquant sans verrouiller les comptes légitimes (ES-09).
- **Vague de soumissions avec photos** (scénario complémentaire de D01) : coût du ré-encodage des photos et de la détection des coordonnées ; aucun article publié sans contrôle sous charge (ES-16, ES-17, ES-31).
- **Mise à l'échelle** : délai de réaction, plafond atteint, coût de la campagne extrapolé (M-DIS-03).
- Pour chaque observation, noter le seuil chiffré, joindre une capture et une interprétation. Chaque observation devient un constat de l'audit V1.

### 7.3 Déroulé de l'audit V1 et du plan de remédiation

1. **Statut de chaque exigence Must et Should** : Conforme, Partielle, Non conforme ou Hors POC, avec la preuve du §7.1. Sans preuve, l'exigence est Non conforme.
2. **Constats techniques** : vulnérabilités issues des scans, des tests et de la revue, chacune avec sa localisation et sa sévérité CVSS.
3. **Risque résiduel** de chaque menace (V × I), réévalué avec les seules mesures prouvées.
4. **Priorisation** selon le §1.3. À priorité égale, passer d'abord les actions qui réduisent le plus le risque pour le moindre effort.
5. **Plan de remédiation** : une ligne par action, avec les colonnes suivantes.

| ID action | Menaces | Exigences | Risque résiduel avant → après | Priorité | Action précise | Périmètre (composants) | Fonction responsable | Effort | Échéance | Critère de clôture (preuve attendue) |
|---|---|---|---|---|---|---|---|---|---|---|

6. **Revérification** : une action n'est close qu'une fois la preuve de clôture produite et vérifiée par la fonction Sécurité.

## 8. Hypothèses, limites, révision

- **Aucune technologie choisie** : les menaces propres aux choix techniques (communication interservice, orchestrateur, stockage) seront ajoutées après D02 à D07.
- **Notation à dire d'expert**, non mesurée. Elle sera recalibrée avec les constats de l'audit V1.
- **Points juridiques simplifiés**, à confirmer avec un juriste avant une mise en production réelle.
- **POC sur données fictives** (PILOTAGE §7), donc RGPD non engagé pour le POC. Les contraintes probables de l'environnement de formation (régions limitées, credentials temporaires, à vérifier selon PILOTAGE §1) sont acceptables pour le POC. Elles ne le seraient pas pour une V1 réelle (ES-26 : localisation des données ; ES-06 : secrets).
- **Révision** : après D02, après chaque audit de fin de sprint, et avant l'audit V1. Dernière révision : 2026-10-09, alignement sur les décisions de [D01](../decisions/D01-fonctionnalite-poc.md).
