# Exigences de sécurité — analyse de menaces du besoin

| | |
|---|---|
| Statut | Validée par Martin le 2026-10-09 |
| Date | 2026-10-09 |
| Mise à jour | 2026-10-09 : alignement sur D01, révision pour l'option B de D02, puis cohérence avec l'architecture v0.2 ; 2026-10-26 : consolidation après D03 à D14 (historique en §8) |
| Fonction | Sécurité |
| Critères servis | C8 (principal), C2 (architecture intégrant la sécurité), C7 (tests de sécurité) |
| Sources | Contexte Collector V1 (2025-03-04) ; consigne d'évaluation §1, §1.2.3, §1.3 ; grille individuelle, critère 8 ; [architecture technique](../architecture/architecture-technique.md) v0.2 (POC option P2) |
| Décisions liées | [D01 — Exigences reformulées et fonctionnalité métier du POC](../decisions/D01-fonctionnalite-poc.md) (décidée le 2026-10-09) ; [D02 — Architecture cible V1, découpage applicatif, périmètre du POC](../decisions/D02-decoupage-applicatif.md) (décidée le 2026-10-09 : option B, walking skeleton WS-2, complément du POC : option P2) ; D03 à D14 (décidées les 2026-10-09 et 2026-10-26, voir l'[index](../decisions/INDEX.md)) : mesures reportées en §5.7 |

Ce document analyse les menaces **à partir du besoin**, puis du **découpage applicatif retenu** (D02, option B), avant tout choix de technologie. Il sert à :
1. fixer les exigences de sécurité de la V1 et leur priorité ;
2. préparer l'audit de la V1 et le plan de remédiation (phase 3).

Martin l'a validé le 2026-10-09 comme **référence de sécurité du projet** (D02). Les points tranchés dans D01 et D02 sont reportés tels quels (voir Synthèse). La révision pour l'option B applique D02 et son avis sécurité ; ses ajouts (FC11, §4.11, ES-38, §5.6) suivent la règle de priorisation du §5.1.

## Synthèse

- 96 menaces analysées : **5 Critiques**, 32 Élevées, 56 Modérées, 3 Faibles (risque brut, sans mesure). Dont 12 propres au découpage retenu (§4.11) : 2 Élevées et 10 Modérées ; et 27 propres aux technologies retenues par D05 à D13 : 6 Élevées, 20 Modérées, 1 Faible (§4.2, §4.9 à §4.11).
- Les 5 risques critiques : fausse confirmation de paiement, accès aux données d'autrui (IDOR), compromission d'un compte admin, contournement de la plateforme via l'échange de coordonnées, vendeur fraudeur (§4.12). Le découpage en services n'en ajoute aucun : il ajoute des flux internes à authentifier.
- Deux des cinq sont des **abus métier**, pas des failles techniques : sur une place de marché, la sécurité porte d'abord sur l'intégrité de la transaction et le maintien du paiement sur la plateforme.
- 38 exigences : 25 Must V1, 7 Should V1, 6 Later (ES-32 passe Must après D10, §5.4). Chaque menace Critique ou Élevée applicable à la V1 est couverte par au moins une exigence Must (§5.4).
- Décisions de Martin du 2026-10-09 ([D01](../decisions/D01-fonctionnalite-poc.md)), reportées dans ce document :
  - fonctionnalité du POC « A enrichie » : mise en vente contrôlée, avec upload sécurisé (ES-17), revalidation après modification (ES-18) et contrôle fermé par défaut (ES-31) ;
  - commission calculée sur le total payé, prix + frais de port (ES-13) ;
  - séquestre en V1, hors POC (ES-15) ;
  - publicités partenaires en V2, ES-28 devenant le prérequis de leur activation ;
  - l'admin assure aussi la modération, sans rôle modérateur distinct en V1 (ES-03, ES-20) ;
  - cinq bonnes pratiques de sécurité démontrées dans le POC (§6.1).
- Décisions de Martin du 2026-10-09 ([D02](../decisions/D02-decoupage-applicatif.md)), reportées dans ce document :
  - ce document est validé comme référence de sécurité du projet ;
  - **option B** : quatre services propriétaires de leurs données (Place de marché, Transactions, Interactions, Contrôle de publication). Transactions isole le périmètre financier dès la V1. Les événements métier sont diffusés par le canal ; un seul appel synchrone authentifié, Transactions → Place de marché, réserve l'article avec une échéance ; l'adaptateur de l'outil de fraude est porté par le Contrôle. Conséquences : frontière FC11 (§3), 12 menaces (§4.11), exigence ES-38 ;
  - décisions par défaut retenues : toute modification d'un article publié le renvoie au contrôle, sans seuil (ES-18) ; la décision de l'admin porte la révision examinée et est refusée (409) si elle est périmée (ES-18) ; ES-07 et ES-08 intégrées au POC ;
  - walking skeleton WS-2 : deux services et le canal dès le sprint 1 ;
  - complément du POC : **option P2**. Le POC déploie les quatre services, dont Interactions minimal et Transactions réduit à la réservation. L'isolement du périmètre financier n'y est démontré que par la structure (§6).
- Les quatre mesures compensatoires de l'avis sécurité de D02, conçues pour l'option A, sont réévaluées pour B (§5.6) : trois restent Must (décodage isolé des images, compte d'accès par service, journaux hors du processus), une passe en Should (clé du prestataire restreinte et plafonnée).
- Décision par défaut retenue par Martin : le nouvel essai de l'appel « réserver » renvoie la réservation existante au lieu d'en créer une seconde (ES-14). Cette mesure et la route « réserver » absente du point d'entrée public (ES-38) passent des compléments recommandés (§6.3) aux mesures intégrées au POC (§6.2).
- Les ajouts de la révision option B (ES-38, menaces M-CAN-05/06, M-MOD-01 à 03, M-SYN-01, M-RES-01/02, verdict des mesures compensatoires) sont à prendre en compte par Martin lors de sa prochaine relecture.
- Décisions D03 à D14 (2026-10-09 et 2026-10-26) : les menaces proposées par leurs avis sécurité sont intégrées au registre avec leur cotation (§4), chaque exigence est reliée à ses mesures et à sa preuve attendue (§5.7), les écarts assumés du POC sont regroupés en §6.4. D03, D05, D06 et D08 restent Candidate jusqu'aux expérimentations XP01 à XP03 (D14, du 2026-10-27 au 2026-11-02) : si un critère de bascule échoue, le repli (ECS Fargate, NATS JetStream, Cognito) impose de réviser les mesures concernées. Ajouts à valider par Martin à sa prochaine relecture, dont le passage d'ES-32 en Must.

## 1. Périmètre et méthode

### 1.1 Périmètre

- **Inclus** : les fonctionnalités V1 du contexte (catalogue, comptes, espaces acheteur et vendeur, boutiques, annonces et photos, contrôle de mise en ligne, chat, notifications et emails, paiement par carte et commission, notes, recommandations, back-office et modération, intégration de la détection de fraude), ainsi que la chaîne de livraison et l'hébergement. La publicité ciblée chez les partenaires est reportée en V2 (D01). Elle reste analysée pour préparer son activation.
- **Architecture retenue** (D02, option B ; [architecture technique](../architecture/architecture-technique.md) v0.2, POC option P2) : quatre services, chacun propriétaire de ses données, dans une zone applicative privée. Place de marché : annonces et catalogue, disponibilité de l'article, comptes et profils, notation, recommandations, alertes de fraude. Transactions : commandes, paiement, commission, séquestre, reversements. Interactions : notifications, messagerie et sa modération. Contrôle de publication : règles sans état, adaptateur de l'outil de fraude. Les échanges entre services (canal avec diffusion, appel synchrone de réservation) entrent dans l'analyse (FC11, §4.11). Aucune technologie n'est choisie.
- **Exclu** : le SI bureautique de l'entreprise, sauf comme vecteur d'attaque des comptes privilégiés. Les évolutions futures (enchères, streaming, bot avant-vente) seront analysées lors de leur cadrage.
- **V1 et POC** : les exigences visent la V1 mise en production. Le POC (mise en vente d'un article avec contrôle automatisé de publication, « A enrichie », D01 ; quatre services déployés selon l'option P2 du complément D02, dont Interactions et Transactions réduits) n'en couvre qu'une partie (§6). Les exigences non couvertes seront des constats de l'audit V1, puis des entrées du plan de remédiation (§7).

### 1.2 Méthode

**STRIDE**, appliqué aux frontières de confiance (§3), complété par des **cas d'abus métier**. La révision pour l'option B l'applique aussi aux échanges entre services (FC11).

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
| A07 | Annonces et photos | Contenu, prix, frais de port, historique des prix, statut de validation, boutiques, disponibilité (disponible, réservé jusqu'à une échéance, vendu) | I, D | Élevée |
| A08 | Réputation | Notes, avis, statut des vendeurs | I | Élevée (base de la garantie de qualité) |
| A09 | Back-office et règles de contrôle | Catégories, décisions de modération, règles de contrôle automatique, suspensions | I, T | Critique |
| A10 | Journaux et traces d'audit | Événements de sécurité et métier | I, T, C | Élevée |
| A11 | Secrets techniques | Clés des intégrations paiement, fraude et email, clés de signature, accès à l'hébergement, credentials temporaires de l'environnement de formation, identités et jetons des services, identifiants d'accès de chaque service au canal et aux données | C | Critique |
| A12 | Disponibilité du service | Catalogue, achat, chat | D | Élevée |
| A13 | Domaine et réputation d'envoi | Nom de domaine, réputation des emails | I | Élevée (vecteur d'hameçonnage) |
| A14 | Code source et chaîne de livraison | Dépôt, pipeline, artefacts, configuration décrite en code | I | Élevée |
| A15 | Échanges entre services | Demandes et résultats de contrôle, événements diffusés (décision de publication, prix modifié, commandes, alertes, suspensions), files d'erreurs, appel de réservation | I, D, C | Élevée (un « publié » ou une « commande payée » falsifiés ont un effet métier direct) |

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
| Services internes | Composants de la plateforme (Place de marché, Transactions, Interactions, Contrôle de publication), chacun avec son identité | Canal (sujets autorisés), appel synchrone de réservation, son propre espace de données | Un service compromis ou mal configuré peut usurper un autre service, publier un faux événement, lire ce qui ne lui est pas destiné ou servir d'intermédiaire abusé (§4.11) |
| Prestataire de paiement | Tiers certifié | Reçoit l'acheteur, notifie le service Transactions | Ses notifications sont un point d'entrée à authentifier, par une route dédiée ; indisponibilité |
| Outil de détection de fraude | Tiers ou interne (non décidé) | Échange avec l'adaptateur du Contrôle de publication, jamais avec le canal : reçoit des événements, renvoie des verdicts et des alertes | Interface à authentifier ; destinataire de données personnelles |
| Service d'envoi d'emails | Tiers | Achemine les notifications émises par Interactions | Réputation d'envoi, usurpation du domaine |
| Sites partenaires publicitaires (V2) | Tiers, confiance limitée | Données de ciblage, scripts éventuels | Collecte non consentie, exécution de code tiers |
| Hébergeur | Tiers de confiance | Infrastructure ; dans l'environnement de formation, rôle d'exécution unique imposé à tous les services | Erreurs de configuration, localisation des données |

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
 |  ------------- Zone applicative privée, FC11 entre services --------------  |
 |  |  Place de marché, Transactions, Interactions, Contrôle                |  |
 |  |  Canal : point à point et diffusion                                   |  |
 |  |  Appel synchrone : Transactions -> Place de marché                    |  |
 |  -------------------------------------------------------------------------  |
 |  ------------------------- FC8 : zone de données -------------------------  |
 |  |  Un espace par service, photos, journaux, sauvegardes                 |  |
 |  -------------------------------------------------------------------------  |
 ===============================================================================
       | FC4                | FC5                | FC6                | FC7
  Prestataire de       Outil de fraude,     Envoi d'emails,      Partenaires
  paiement, relié      relié à l'adaptateur relié à              publicitaires (V2)
  à Transactions       du Contrôle          Interactions

 FC9  : chaîne de livraison (dépôt, pipeline) et accès d'exploitation -> exécution
 FC10 : contenu d'un utilisateur -> navigateur d'un autre (y compris le back-office)
 FC11 : service -> service dans la zone applicative privée (canal, appel synchrone)
```

| ID | Frontière | Ce qui la traverse | Principe de contrôle |
|---|---|---|---|
| FC1 | Internet → plateforme | Toutes les requêtes | Chiffrement, limitation de débit, validation des entrées |
| FC2 | Public → espace authentifié | Requêtes d'un utilisateur connecté | Authentification, session, contrôle d'accès par objet, vérifiés par chaque service |
| FC3 | Utilisateur → back-office | Actions privilégiées | Rôle vérifié côté serveur, second facteur, traçabilité |
| FC4 | Transactions ↔ prestataire de paiement | Redirection de l'acheteur, notifications de paiement, reversements | Carte jamais chez Collector, notification authentifiée reçue par une route dédiée vers Transactions seul, idempotence |
| FC5 | Adaptateur du Contrôle de publication ↔ outil de fraude | Événements (annonce, prix, vendeur, transaction), verdicts et alertes | Authentification mutuelle, données minimisées, livraison fiable, aucun accès de l'outil au canal |
| FC6 | Interactions → emails | Notifications vers des boîtes externes | Domaine authentifié, contenu utilisateur neutralisé |
| FC7 | Plateforme ↔ partenaires publicitaires (V2) | Données de ciblage, scripts éventuels | Consentement, pseudonymisation, aucun script sur les pages sensibles |
| FC8 | Services → zone de données | Lectures et écritures | Zone non exposée, un compte d'accès par service limité à son espace, moindre privilège, chiffrement au repos |
| FC9 | Chaîne de livraison → exécution ; accès d'exploitation | Code, artefacts, configuration, secrets | Revue, scans bloquants, secrets hors dépôt, accès tracés |
| FC10 | Contenu utilisateur → autre utilisateur | Titres, descriptions, photos, messages, noms de boutique, champs de profil, notifications | Validation à l'entrée, neutralisation à l'affichage, filtrage des coordonnées |
| FC11 | Service ↔ service, dans la zone applicative privée | Messages du canal : point à point (demande et résultat de contrôle) et diffusion (décision de publication, prix modifié, vendeur, intérêts, commandes, alertes de fraude) ; appel synchrone Transactions → Place de marché (réserver jusqu'à une échéance) | Une identité par service, droits par sujet ; jeton de service vérifié et identité de l'utilisateur revérifiée par le service appelé ; contrat validé à la réception ; idempotence ; messages minimisés, conservés pour une durée bornée ; aucun outil externe sur le canal (ES-38, ES-21, ES-31) |

Révision pour l'option B (D02) :
- les quatre services sont dans une zone applicative privée ; leurs échanges forment **FC11**, absente de la première analyse (avis sécurité de D02, § 3) ;
- **dans un service, la frontière entre modules n'est pas une frontière de confiance** (Annonces et Comptes dans la Place de marché, Notifications et Messagerie dans Interactions) : même processus, mêmes secrets, même compte d'accès aux données. La frontière entre services en est une, à condition qu'ES-38 soit appliquée ;
- écart connu : dans l'environnement de formation, tous les services partagent l'identité d'exécution imposée. Le cloisonnement par l'hébergeur n'y est pas démontrable (ES-38, §8).

## 4. Menaces majeures

Risque **brut** (sans mesure). V = vraisemblance, I = impact (§1.3). Les exigences qui traitent chaque menace sont en §5. Le §4.11 ajoute les menaces propres au découpage retenu (D02, option B).

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

**Option B (D02)** : la réservation de l'objet unique est tenue par la Place de marché et demandée par Transactions. Ses risques propres sont en §4.11 (M-RES-01, M-RES-02) ; M-PAY-05 reste la menace générale.

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
| M-IDP-01 | S, E | FC3 | La console d'administration du serveur d'autorisation, son realm `master` ou le compte d'administration initial sont joignables ou laissés actifs. | Prise de contrôle de toutes les identités, admin compris. | 2 | 4 | 8 Élevé |
| M-IDP-02 | E | FC2, FC11 | Confusion de jetons : jeton de service accepté sur une route d'utilisateur (rôles par défaut du compte de service), ou jeton d'utilisateur accepté sur « réserver ». | Actions d'acheteur sans acheteur réel, réservation hors Transactions. | 2 | 3 | 6 Modéré |
| M-IDP-03 | S | FC11 | Un service compromis rejoue vers les deux autres le jeton d'utilisateur, valable pour trois audiences, pendant sa durée de vie (5 min). | Actions au nom de l'utilisateur dans un autre service. | 2 | 3 | 6 Modéré |
| M-IDP-04 | S | FC1, FC3 | Comptes de test à mot de passe connu, dont un admin, sur l'environnement exposé. | Prise de compte admin triviale (M-BO-01). | 2 | 4 | 8 Élevé |
| M-IDP-05 | E | FC2 | Erreur de paramétrage du realm versionné (URI de redirection à joker, flux « mot de passe », portée trop large) introduite sans revue. | Vol de code d'autorisation ou de jeton, droits excessifs. | 2 | 3 | 6 Modéré |

**Décision D06 (2026-10-26)** : M-IDP-01 à M-IDP-05 viennent de l'avis sécurité de D06 (Keycloak, application statique à jetons en mémoire). Risque résiduel accepté pour le POC : usage ou vol du jeton d'accès par un script injecté pendant la session, borné par la CSP (D04) et la rotation du jeton de rafraîchissement (MA11) ; l'intermédiaire serveur (BFF) est la cible V1 (§6.4).

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
| M-DIS-07 | D | FC8 | Instance de base partagée : une requête coûteuse, une boucle ou une fuite de connexions dans un service épuise les connexions ou le calcul de l'instance. | Indisponibilité simultanée des trois services propriétaires de données. | 2 | 2 | 4 Modéré |

### 4.10 Socle technique : hébergement, chaîne de livraison, journaux

| ID | STRIDE | FC | Scénario | Impact métier | V | I | Risque |
|---|---|---|---|---|---|---|---|
| M-LIV-01 | T, E | FC9 | Un composant tiers ou une image de base embarque une vulnérabilité connue, exploitable avec un outil public. | Compromission d'un service exposé. | 3 | 3 | 9 Élevé |
| M-LIV-02 | I, E | FC9 | Un secret est commité, intégré à une image ou affiché dans un journal du pipeline. | Usage frauduleux des intégrations (paiement, emails) ou de l'hébergement. | 3 | 3 | 9 Élevé |
| M-LIV-03 | I | FC8 | Une erreur de configuration expose à Internet une base, le stockage de fichiers ou un accès d'administration, y compris l'API du cluster ou une console d'exploitation (métriques, tableaux de bord, broker) routée par erreur (D03, D05, D09). | Fuite massive. | 2 | 4 | 8 Élevé |
| M-LIV-04 | I, S | FC1 | Trafic non chiffré ou mal chiffré, intercepté sur un réseau partagé (wifi public, incubateur). | Vol d'identifiants et de sessions. | 3 | 3 | 9 Élevé |
| M-LIV-05 | T | FC9 | Poussée directe sur la branche principale, pipeline modifié ou artefact substitué. | Code malveillant en production. | 1 | 4 | 4 Modéré |
| M-LIV-06 | R | Toutes | Les événements de sécurité ne sont pas journalisés. | Incident non détecté, impossible à investiguer ou à prouver. | 3 | 2 | 6 Modéré |
| M-LIV-07 | I | FC8 | Mots de passe, jetons, données de carte ou contenu des messages écrits dans les journaux, y compris par le texte des erreurs (URL présignée, chaîne de connexion), les `panic` et les journaux des composants tiers (serveur d'autorisation, broker, collecteur) (D09). | Fuite par les journaux, accès élargi aux données. | 3 | 2 | 6 Modéré |
| M-LIV-08 | T, I, E | FC1, FC8 | Injection dans les requêtes vers les données via la recherche, les filtres ou les formulaires. | Lecture ou modification de la base. | 2 | 4 | 8 Élevé |
| M-LIV-09 | T, E | FC9 | Action tierce compromise ou injection dans un workflow de CI : le jeton du job (écriture du registre ou de `env/demo`) est détourné. | Image malveillante publiée, puis déployée par Flux. | 2 | 4 | 8 Élevé |
| M-LIV-10 | T | FC9 | Écriture illégitime de `env/demo` (jeton détourné, règle de branche absente) : empreinte d'une image étrangère ou manifeste arbitraire. | Code arbitraire exécuté dans le cluster de démonstration. | 2 | 3 | 6 Modéré |
| M-LIV-11 | E | FC9 | Contrôleur Flux laissé `cluster-admin` : tout manifeste réconcilié obtient les droits de tout le cluster. | Prise de contrôle du cluster, accès à tous les Secrets. | 2 | 4 | 8 Élevé |
| M-LIV-12 | I | FC9 | Dépôt public : un fichier interdit (état Terraform, `.env`, données réelles) ou une sortie de pipeline divulgue un secret ou une donnée. | Secret ou donnée exposés à tous, durablement (forks, caches). | 2 | 3 | 6 Modéré |
| M-LIV-13 | S, T | FC9 | Compte GitHub de Martin compromis. | Contrôle complet du code, de la CI et du déploiement. | 1 | 4 | 4 Modéré |
| M-LIV-14 | I | FC9 | Dépôt public : un résumé de CI, `qualite.json`, un rapport k6 ou ZAP ou un fichier de sprint publie une vulnérabilité ouverte (identifiant, composant, chemin atteignable), un paramètre d'authentification (`code`, `state`, jeton) ou l'adresse IP de Martin. | Faille divulguée avant son correctif, ciblage de l'environnement de démonstration. | 2 | 3 | 6 Modéré |

**Exposition (D08)**

| ID | STRIDE | FC | Scénario | Impact métier | V | I | Risque |
|---|---|---|---|---|---|---|---|
| M-EXP-01 | E, I | FC1, FC3 | Un chemin non normalisé ou encodé (`..`, `%2e%2e`, `%2f`) franchit la liste blanche de l'ALB et atteint la console ou le realm `master` du serveur d'autorisation. | Prise de contrôle des identités (M-IDP-01). | 2 | 4 | 8 Élevé |
| M-EXP-02 | S, R | FC1 | `X-Forwarded-For` forgé : limitation de débit contournée ou adresse fausse dans les journaux. | Force brute et déni de service non freinés, enquête faussée. | 3 | 2 | 6 Modéré |
| M-EXP-03 | D | FC1 | La table de limitation de débit, non bornée, est saturée par des adresses variées. | Déni de service mémoire par le mécanisme de protection lui-même. | 2 | 2 | 4 Modéré |
| M-EXP-04 | S, T | FC1 | Délégation DNS pendante après effacement du compte du lab : une zone tierce reprend le sous-domaine, certificat compris. | Hameçonnage sous le nom du projet (A13). | 1 | 3 | 3 Faible |

**Observabilité (D09)**

| ID | STRIDE | FC | Scénario | Impact métier | V | I | Risque |
|---|---|---|---|---|---|---|---|
| M-OBS-01 | S, R | FC9, FC11 | Un service compromis écrit sur sa sortie standard de faux événements d'audit attribués à un autre service ou à un autre utilisateur. | Audit trompeur, enquête faussée, répudiation. | 2 | 3 | 6 Modéré |
| M-OBS-02 | T, R | FC9 | Un pod détenteur de `LabRole` supprime un groupe de journaux d'audit ou réduit sa rétention. | Effacement des traces après compromission. | 2 | 3 | 6 Modéré |
| M-OBS-03 | I, E | FC9 | Compromission du collecteur de journaux (image vulnérable, configuration) : lecture des journaux de tout le nœud et identifiants de `LabRole`. | Fuite des journaux et de tous les secrets. | 1 | 4 | 4 Modéré |
| M-OBS-04 | D | FC1 | Explosion de cardinalité des métriques ou inondation de journaux provoquée de l'extérieur. | Supervision aveugle pendant l'attaque, coût d'ingestion. | 2 | 2 | 4 Modéré |

**Infrastructure décrite en code (D11)**

| ID | STRIDE | FC | Scénario | Impact métier | V | I | Risque |
|---|---|---|---|---|---|---|---|
| M-INF-01 | I | FC9 | Un secret entre dans l'état Terraform ou un plan (ressource aléatoire, version de secret, sortie), puis fuit par le poste, une sauvegarde ou un commit. | Accès à la base ou aux intégrations. | 2 | 3 | 6 Modéré |
| M-INF-02 | S, E | FC9 | Vol des identifiants de session AWS sur le poste (fichier, historique, journal). | Contrôle du compte du lab pendant la session. | 2 | 3 | 6 Modéré |
| M-INF-03 | D | FC9 | Planification de destruction mal ciblée ou restée armée. | Ressource persistante perdue, session suivante détruite. | 2 | 2 | 4 Modéré |
| M-INF-04 | I | FC1, FC8 | Ressource orpheline hors IaC restée exposée (load balancer, groupe ouvert, instantané). | Exposition non analysée, coût. | 2 | 3 | 6 Modéré |
| M-INF-05 | D, T | FC9 | Perte ou désynchronisation de l'état : ressources en double ou non gérées. | Orphelins exposés et facturés. | 2 | 2 | 4 Modéré |

### 4.11 Échanges entre services (option B, D02)

Traçabilité :
- **M-CAN-01 à M-CAN-04 et M-MOD-01 à M-MOD-03** viennent de l'avis sécurité de D02 (§ 3), rédigé pour l'option A. Ils gardent leur identifiant et sont reformulés pour B : la frontière entre services (FC11) remplace celle entre modules là où un service possède ses données ; dans un service à plusieurs modules, la frontière entre modules reste sans valeur de sécurité (§3). « M-MOD » vise donc désormais le cloisonnement entre services, et entre modules d'un même service.
- **M-CAN-05, M-CAN-06, M-SYN-01, M-RES-01 et M-RES-02** sont propres à B : diffusion, appel synchrone, cohérence différée de la réservation. Ils reprennent aussi les risques relevés par l'avis sécurité sur le complément de D02 (faux événement « décision de publication », jeton de service trop large ou volé, adjoint confus, double réservation) et la réservation abusive relevée par D01.
- Le découpage réduit le rayon d'impact de plusieurs menaces des §4.1 à §4.10 (clé du prestataire et notifications de paiement isolées dans Transactions, conversations dans Interactions). Leur risque brut est inchangé : l'effet sera mesuré par le risque résiduel de l'audit V1.

**Canal (FC11)**

| ID | STRIDE | FC | Scénario | Impact métier | V | I | Risque |
|---|---|---|---|---|---|---|---|
| M-CAN-01 | S, T | FC11, FC5 | Un service ou un acteur qui peut écrire sur le canal y dépose un faux résultat de contrôle « publié » : accès partagé entre services, droits trop larges, outil de fraude externe connecté au canal. | Contrôle de publication contourné : annonce frauduleuse ou porteuse de coordonnées publiée sous la garantie Collector. | 2 | 3 | 6 Modéré |
| M-CAN-02 | T | FC11 | Un résultat de contrôle périmé ou rejoué est appliqué à une nouvelle révision de l'article ; un événement diffusé livré deux fois ou dans le désordre ramène un état antérieur (article « vendu » repassé « disponible »). | Version non examinée publiée (M-ANN-05) ; états incohérents entre services. | 2 | 3 | 6 Modéré |
| M-CAN-03 | D | FC11 | Un message malformé fait échouer son consommateur en boucle et bloque la file ; un afflux de soumissions sature le canal ; un abonné arrêté accumule les événements sans limite. Le contrôle étant fermé par défaut, plus rien n'est publié. | Publications bloquées, notifications en retard, surcoût. | 3 | 2 | 6 Modéré |
| M-CAN-04 | I | FC11, FC8 | Les messages durables, copies remises à chaque abonné et files d'erreurs comprises, conservent sans limite du contenu saisi par les utilisateurs ou des données personnelles. | Données conservées hors de toute durée définie, lisibles par qui accède au canal. | 2 | 2 | 4 Modéré |
| M-CAN-05 | S, T | FC11 | Diffusion, faux événement (extension de M-CAN-01 aux sujets diffusés) : un service publie sur un sujet dont il n'est pas le producteur (service compromis, droits d'écriture trop larges). Fausse « décision de publication » : notification trompeuse au vendeur, outil de fraude et acheteurs intéressés trompés. Fausse « commande payée » : article passé « vendu », notation ouverte sans transaction réelle. Fausse alerte de fraude ou fausse suspension : reversements bloqués, vendeur écarté. | Ventes bloquées, réputation faussée (contourne ES-22), vendeurs écartés à tort ; brèche dans l'isolement de la finance voulu par B. | 2 | 3 | 6 Modéré |
| M-CAN-06 | I, D | FC11 | Diffusion, abonné non autorisé : un service ou un futur consommateur (analyse des ventes, partenaires V2) s'abonne à des événements qui ne lui sont pas destinés (centres d'intérêt, commandes, identifiants d'acheteurs) ; ou un consommateur non autorisé prend les messages d'une file point à point à la place du destinataire (demandes de contrôle détournées). | Fuite de données personnelles ; articles bloqués « en contrôle ». | 2 | 3 | 6 Modéré |
| M-CAN-07 | S, T, I, E | FC11 | Prise de contrôle du plan d'administration du broker : cookie Erlang ou identité d'administration exposés, port de gestion ou de distribution joignable depuis un pod, empreintes de mot de passe versionnées. | Liaisons créées, lecture et purge de toutes les files, faux résultats de contrôle (M-CAN-01). | 2 | 4 | 8 Élevé |
| M-CAN-08 | D | FC11 | Perte silencieuse par les bornes elles-mêmes : longueur maximale en `drop-head` ou durée de vie sans mise à l'écart, qui suppriment des événements sans alerte (contredit ENF-12). | Publications ou notifications perdues sans que personne ne s'en aperçoive. | 2 | 2 | 4 Modéré |

**Cloisonnement entre services et entre modules**

| ID | STRIDE | FC | Scénario | Impact métier | V | I | Risque |
|---|---|---|---|---|---|---|---|
| M-MOD-01 | I, E | FC8, FC11 | Avec un compte d'accès aux données commun à plusieurs services, une injection ou un défaut dans un service (recherche du catalogue de la Place de marché) lit ou modifie les données d'un autre (commandes de Transactions, conversations d'Interactions). | Fuite massive de données personnelles et de conversations ; altération des commandes. | 2 | 4 | 8 Élevé |
| M-MOD-02 | E | FC2, FC11 | Adjoint confus : un service appelé ne revérifie pas l'autorisation de l'utilisateur pour lequel il agit. Transactions transmet à « réserver » un identifiant d'article ou d'acheteur choisi par l'utilisateur, et la Place de marché s'y fie : réservation d'un article non publié et lecture de son prix, réservation au nom d'un autre acheteur. Même risque sur une interface interne entre modules d'un service. | Accès indirect à la ressource d'autrui, qui contourne ES-04. | 2 | 3 | 6 Modéré |
| M-MOD-03 | I, E, T | FC8, FC9, FC11 | Exécution de code dans un service (décodage d'une image dans la Place de marché, dépendance vulnérable) : accès à toutes ses données, à ses secrets et à son identité de service (faux événements, M-CAN-05 ; prix de référence falsifié renvoyé à Transactions), et à ceux des autres services si l'identité d'exécution est partagée ou les droits trop larges, ce qui est le cas dans l'environnement de formation. Variante propre à l'orchestrateur (D03, D05) : un pod obtient par l'IMDS le rôle du nœud, puis lit les Secrets des pods placés sur ce nœud, ce qui annule le cloisonnement des Secrets par service (X16 élargi, bloquant en D14). | Fuite massive des profils, effacement des traces ; avec une identité partagée, accès à la clé du prestataire et à la clé d'envoi d'emails. | 2 | 4 | 8 Élevé |

**Appel synchrone et réservation**

| ID | STRIDE | FC | Scénario | Impact métier | V | I | Risque |
|---|---|---|---|---|---|---|---|
| M-SYN-01 | S | FC1, FC11 | Usurpation sur l'appel synchrone : un appelant autre que Transactions invoque « réserver » (route interne exposée par le point d'entrée public, autre service, utilisateur muni de son propre jeton, jeton de service trop large ou volé) ; ou un faux service répond à Transactions à la place de la Place de marché. | Objets bloqués en boucle, ventes empêchées ; prix de référence altéré (M-PAY-01). | 2 | 3 | 6 Modéré |
| M-RES-01 | T | FC11, FC4 | Double réservation : deux réservations concurrentes du même objet unique acceptées (réservation non atomique) ; nouvel essai de l'appel après un délai dépassé, qui crée une seconde réservation ou une commande sans réservation ; « commande payée » reçue après l'échéance (canal en retard, Place de marché arrêtée) alors qu'un autre acheteur a réservé l'article. | Objet vendu deux fois : double encaissement, remboursement, litige (M-PAY-05). | 3 | 2 | 6 Modéré |
| M-RES-02 | T, Abus | FC2, FC11 | Échéance contournée et réservation abusive (relevée par D01) : échéance fixée ou prolongée par l'appelant ; réservations renouvelées en boucle, sans payer, par un acheteur ou des robots pour retirer de la vente les objets rares ; article modifié ou retiré par le vendeur pendant la réservation, si bien que l'acheteur paie un prix ou un article différents de ceux réservés. | Objets rares bloqués, ventes perdues ; appât puis substitution sur un article réservé (M-ANN-05). | 3 | 2 | 6 Modéré |

Écart connu de l'environnement de formation (avis sécurité de D02, § 5) : tous les services partagent l'identité d'exécution imposée. Un service compromis peut atteindre les ressources des autres (canal, données, stockage, secrets), ce qui aggrave M-MOD-03 dans le POC, sur données fictives. La V1 exige une identité d'exécution par service (ES-38).

**Menaces issues des décisions D03 à D14** (révision du 2026-10-26). Reprises des avis sécurité avec leur cotation et leur identifiant : M-CAN-07 et 08 (D05, §4.11), M-IDP-01 à 05 (D06, §4.2), M-DIS-07 (D07, §4.9), M-EXP-01 à 04 (D08), M-OBS-01 à 04 (D09), M-LIV-09 à 13 (D10), M-INF-01 à 05 (D11), en §4.10 sauf mention. M-LIV-14 est ajoutée par cette révision : les avis de D12 et D13 décrivent le risque (constats ouverts publiés dans un dépôt public) sans le numéroter. Compléments sans nouvel identifiant : M-MOD-03 (D03, D05), M-LIV-03 (D03, D09), M-LIV-07 (D09). D04 et D14 n'apportent pas de menace nouvelle : leurs risques relèvent de M-ANN-01, M-LIV-01 et, pour le bac à sable, M-LIV-03.

### 4.12 Synthèse des risques

| Niveau | Nombre |
|---|---|
| Critique | 5 |
| Élevé | 32 |
| Modéré | 56 |
| Faible | 3 |
| **Total** | **96** |

Révision pour l'option B : les 12 menaces du §4.11 (2 Élevées, 10 Modérées) ne modifient pas la liste des risques critiques. Révision D03 à D14 : les 27 menaces ajoutées (6 Élevées, 20 Modérées, 1 Faible) ne la modifient pas non plus.

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
- Chaque exigence décrit le **quoi**, pas le **comment**. La solution sera choisie dans les décisions D03 à D13 (§5.5) ; D02 a fixé le découpage applicatif.
- **Colonne POC** :
  - « Démontrée (BPn) » : l'une des cinq bonnes pratiques décidées dans D01, présentée en soutenance (§6.1).
  - « Intégrée » : mesure décidée dans D01 ou D02, retenue par défaut par Martin, ou déjà prévue par les règles du projet, présente et testée dans le POC sans être présentée comme bonne pratique (§6.2).
  - « Recommandée » : utile au POC, non tranchée (§6.3).
  - « Partielle » : le POC ne couvre qu'une partie de l'exigence.
  - « Documentée » : hors POC. Son absence sera un constat de l'audit V1 et une entrée du plan de remédiation (§6.4).

### 5.2 Exigences

**Socle (consigne et transverse)**

| ID | Exigence | Priorité | Menaces traitées | POC |
|---|---|---|---|---|
| ES-01 | **Chiffrement des échanges** : tout accès externe passe en HTTPS (redirection systématique depuis HTTP, TLS 1.2 minimum, certificat valide renouvelé automatiquement, en-tête HSTS). Les flux internes, dont le canal et l'appel synchrone entre services, sont chiffrés ou confinés dans un réseau privé non exposé. | Must (consigne) | M-LIV-04, M-ACC-03 | Démontrée (BP1) ; flux internes confinés sans TLS : écart assumé (§6.4) |
| ES-02 | **Authentification déléguée et sessions** : authentification confiée à un serveur d'autorisation dédié, avec un protocole standard (type OpenID Connect). Aucun service métier ne stocke ni ne vérifie de mot de passe. Jetons signés, de courte durée, vérifiés par chaque service (signature, expiration, émetteur, audience). Session invalidée à la déconnexion et au changement de mot de passe. Cookies de session `Secure`, `HttpOnly`, `SameSite`. Protection contre les requêtes intersites forgées sur les actions qui modifient l'état. Jamais de jeton dans une URL. | Must (consigne) | M-ACC-01, M-ACC-03, M-ACC-05, M-IDP-02 | Démontrée (BP2) ; application statique à jetons en mémoire, BFF en V1 : écart assumé (§6.4) |
| ES-03 | **Autorisation par rôle côté serveur** : rôles visiteur, acheteur, vendeur et administrateur. L'administrateur assure aussi la modération (D01, [exigences.md](../00-cadrage/exigences.md)). Refus par défaut. Chaque fonction vérifie le rôle. Le rôle provient uniquement du serveur d'autorisation, jamais d'un champ modifiable par l'utilisateur. | Must (consigne) | M-BO-02, M-BO-01 | Démontrée (BP2, BP5) |
| ES-04 | **Contrôle d'accès au niveau objet** : chaque lecture ou modification d'une commande, conversation, profil, adresse ou annonce non publiée vérifie côté serveur que l'utilisateur y est partie prenante. Les identifiants sont non prévisibles, en défense en profondeur seulement (jamais à la place du contrôle). Un test automatisé « B ne peut pas accéder à la ressource de A » existe pour chaque type de ressource. Le contrôle s'applique aussi derrière un appel entre services : le service appelé reçoit l'identité de l'utilisateur, la revérifie et applique lui-même ce contrôle, sans se fier aux identifiants transmis par l'appelant (ES-38). | Must | M-ACC-02, M-ANN-06, M-MOD-02 | Démontrée (BP5) |
| ES-05 | **Analyse de vulnérabilités dans la chaîne de livraison** : analyse des dépendances, des images, du code (analyse statique orientée sécurité), de la configuration décrite en code, et détection de secrets. Une vulnérabilité critique ou élevée corrigeable bloque la fusion. Toute exception est justifiée et datée. Les rapports sont conservés. Analyse dynamique non intrusive de l'application déployée en fin de sprint. Dans un dépôt public, les résultats publiés se limitent à des compteurs ; le détail d'une faille ouverte reste au registre jusqu'à sa correction (D12). | Must (consigne) | M-LIV-01, M-LIV-02, M-LIV-03, M-LIV-08, M-LIV-09, M-LIV-14, M-IDP-05, M-OBS-03 | Démontrée (BP3) |
| ES-06 | **Gestion des secrets** : aucun secret dans le dépôt, les images, les journaux ou les sorties du pipeline. Secrets injectés à l'exécution depuis un stockage de secrets, avec des droits minimaux par secret. Procédure de révocation documentée (complète PILOTAGE §7). | Must | M-LIV-02, M-LIV-12, M-INF-01, M-INF-02, M-IDP-04 | Démontrée (BP3) |
| ES-07 | **Exposition minimale de l'hébergement** : seuls les points d'entrée HTTPS publics sont accessibles depuis Internet. Bases, stockage de fichiers et accès d'administration restent privés. Chiffrement au repos. En-têtes HTTP de sécurité (HSTS, CSP, interdiction de deviner le type de contenu, interdiction de l'intégration en cadre). Origines autorisées limitées à la plateforme. Messages d'erreur génériques, sans trace technique. | Must | M-LIV-03, M-LIV-04, M-ANN-03, M-PAY-04, M-CAN-07, M-IDP-01, M-EXP-01, M-EXP-04, M-INF-04, M-INF-05 | Intégrée (D02, puis D03, D07, D08, D11, §5.7) : en-têtes de sécurité et CSP, y compris sur l'écran « à vérifier » servi sur l'origine publique ; seuls les points d'entrée HTTPS exposés |
| ES-08 | **Validation des entrées et neutralisation des sorties** : chaque entrée est validée côté serveur (type, longueur, format, bornes). Les requêtes vers les données sont paramétrées. Tout contenu utilisateur est encodé selon son contexte d'affichage, sans HTML libre. La politique de sécurité du contenu (CSP) n'autorise que les scripts de la plateforme. | Must | M-ANN-03, M-LIV-08, M-PAY-04, M-PUB-02, M-BO-01 | Intégrée (D02) : catalogue, espace vendeur, notifications et écran « à vérifier », servi sur l'origine publique |

**Identités de service et cloisonnement (option B, D02)**

| ID | Exigence | Priorité | Menaces traitées | POC |
|---|---|---|---|---|
| ES-38 | **Identités de service et cloisonnement** : chaque service a sa propre identité et n'accède qu'à ses propres ressources. **Canal** : une identité par service ; droits par sujet, en écriture comme en lecture. Seul le producteur d'un sujet y publie : le Contrôle pour les résultats de contrôle et les alertes de fraude, la Place de marché pour les demandes de contrôle et ses événements, Transactions pour les événements de commande. Chaque service ne s'abonne qu'aux sujets dont il a besoin ; aucun service n'a le droit de créer un abonnement, opération réservée à l'exploitation. Chaque message est validé contre son contrat à la réception, émetteur compris ; un message non conforme est écarté et signalé. **Données** : un compte d'accès par service, limité à son espace ; le Contrôle n'en a aucun ; la base refuse elle-même tout accès d'un service aux données d'un autre. **Secrets** : chaque service ne reçoit que les siens. **Appel synchrone** Transactions → Place de marché : jeton de service émis par le serveur d'autorisation, de courte durée, vérifié par la Place de marché (signature, expiration, émetteur, audience « Place de marché », droit limité à « réserver ») ; route absente du point d'entrée public ; service appelé authentifié par l'appelant ; flux confiné à la zone privée. **Interfaces internes**, entre services comme entre modules d'un service : elles reçoivent l'identité de l'utilisateur et appliquent ES-04 elles-mêmes ; sur la réservation, la Place de marché revérifie le jeton de l'acheteur. **Argent** : un événement reçu d'un autre service peut suspendre un achat ou un reversement, jamais déclencher un mouvement de fonds. **Hébergement** (V1) : une identité d'exécution par service, limitée à ses ressources (canal, données, stockage, secrets). Tests négatifs, tous refusés : écriture sur un sujet qui n'est pas le sien ; accès aux données d'un autre service ; appel « réserver » sans jeton de service, avec le jeton d'un autre service ou d'un utilisateur, ou avec une identité d'acheteur forgée. **Écart connu** : l'environnement de formation impose un rôle d'exécution unique ; le cloisonnement par l'hébergeur n'y est pas démontrable (§8). | Must (consigne) | M-MOD-01, M-MOD-02, M-MOD-03, M-CAN-01, M-CAN-05, M-CAN-06, M-CAN-07, M-SYN-01, M-IDP-02, M-IDP-03, M-EXP-01 | Partielle : identités et droits par sujet sur le canal, compte d'accès et secrets par service, appel synchrone authentifié, revérification du jeton de l'acheteur et route « réserver » absente du point d'entrée public intégrés (option P2). Le canal retenu (RabbitMQ, D05) authentifie chaque service : droits par sujet et émetteur vérifié intégrés (MC1, MC2), sous réserve de XC2. Identité d'exécution partagée : écart connu |

**Comptes**

| ID | Exigence | Priorité | Menaces traitées | POC |
|---|---|---|---|---|
| ES-09 | **Protection des parcours de compte** : nombre de tentatives de connexion limité par compte et par origine, avec un délai croissant. Pas de verrouillage définitif, qui permettrait un déni de service sur les comptes légitimes. Politique de mot de passe conforme à la recommandation CNIL, avec refus des mots de passe connus comme compromis. Message identique que le compte existe ou non. Adresse email vérifiée. Lien de réinitialisation aléatoire, à usage unique et de courte validité. Notification de tout changement de mot de passe ou d'email. | Must | M-ACC-01, M-ACC-04, M-ACC-06, M-ACC-07 | Partielle (D06 : MA13) ; pas de limite par origine sur la connexion : écart assumé (§6.4) |
| ES-10 | **Second facteur obligatoire** pour le rôle administrateur, qui assure aussi la modération ; nouvelle authentification avant les actions d'administration à fort impact. | Must | M-BO-01, M-IDP-01, M-IDP-04 | Intégrée (D06 : MA12) ; nouvelle authentification avant une action à fort impact : V1 |
| ES-11 | **Actions sensibles des vendeurs et maîtrise des reversements** : second facteur proposé. Nouvelle authentification et notification avant tout changement des coordonnées de reversement, de l'email ou du mot de passe. Délai de carence avant le premier reversement vers de nouvelles coordonnées. Clé du prestataire limitée aux opérations nécessaires, avec plafonds et alertes côté prestataire sur les remboursements et les reversements (mesure compensatoire de D02, ramenée à Should avec l'option B, §5.6). | Should | M-PAY-08, M-ACC-03, M-LIV-02, M-MOD-03 | Documentée |

**Paiement et commission**

| ID | Exigence | Priorité | Menaces traitées | POC |
|---|---|---|---|---|
| ES-12 | **Paiement délégué à un prestataire certifié PCI-DSS** : la carte est saisie uniquement sur la page ou le composant hébergé par le prestataire. Aucune donnée de carte ne transite chez Collector, n'y est stockée ni journalisée. Authentification forte du porteur (DSP2). Encaissement, commission et reversements passent par l'offre « place de marché » du prestataire, qui vérifie l'identité des vendeurs avant reversement. Aucun script tiers sur la page qui lance le paiement. | Must | M-PAY-03, M-PAY-04, M-PAY-07, M-FRA-02 | Documentée |
| ES-13 | **Montants calculés côté serveur** : prix, frais de port, total et commission sont recalculés côté serveur à partir des données de référence, jamais repris de la requête. Valeurs bornées (strictement positives, plafonds par catégorie). Assiette de la commission : **total payé, soit prix + frais de port** (décision D01 du 2026-10-09). Avec l'option B, prix de référence et frais de port sont ceux que renvoie la Place de marché à la réservation, figés jusqu'à l'échéance (D02). | Must | M-PAY-01 ; M-PAY-06 (transfert vers les frais de port traité par la décision sur l'assiette, valeurs aberrantes par les bornes) | Partielle (option P2) : commande en attente créée par Transactions au prix de référence renvoyé par la Place de marché, commission calculée côté serveur ; sans paiement |
| ES-14 | **Confirmation de paiement vérifiée et idempotente** : une commande ne passe « payée » que sur une notification serveur à serveur du prestataire dont on a vérifié l'authenticité (signature) et le contenu (montant, devise, référence de commande). Le retour du navigateur n'a aucune valeur de preuve. Le traitement est idempotent : une notification rejouée est sans effet. Un objet unique est réservé de façon atomique pendant le paiement. Avec l'option B (D02), la réservation est tenue par la Place de marché : atomique dans ses données ; échéance fixée par elle, plafonnée, non prolongeable par l'appelant, au-delà de la durée de session du prestataire et du délai de livraison des événements ; nombre de réservations actives limité par acheteur (une seule dans le POC) ; nouvel essai de l'appel sans seconde réservation, la réservation existante étant renvoyée ; article ni modifiable ni retirable pendant la réservation. Une confirmation de paiement reçue après l'échéance, sur un article réservé entre-temps par un autre acheteur, déclenche un remboursement tracé. | Must | M-PAY-02, M-PAY-05, M-RES-01, M-RES-02 | Partielle (option P2) : réservation atomique avec test de concurrence, nouvel essai sans seconde réservation, une seule réservation active par acheteur, échéance plafonnée, article figé pendant la réservation ; confirmation de paiement hors POC |
| ES-15 | **Séquestre** (décidé en V1, hors POC, D01) : les fonds restent chez le prestataire jusqu'à ce que l'acheteur confirme la réception. À défaut, ils sont libérés automatiquement après un délai sans signalement (EF-26). Le reversement est suspendu si l'outil de fraude ou un signalement vise le vendeur. La protection acheteur ne vaut que pour les paiements sur la plateforme, et l'interface le rappelle. | Must (décidé, D01) | M-FRA-01, M-CHA-01, M-PAY-09 | Documentée |

**Chat et annonces**

| ID | Exigence | Priorité | Menaces traitées | POC |
|---|---|---|---|---|
| ES-16 | **Filtrage des coordonnées** dans le chat, les titres, descriptions, noms de boutique et profils : détection des emails, numéros de téléphone, liens externes et variantes masquées (espaces, lettres, chiffres en toutes lettres), avec un coût de détection borné (longueur de texte limitée). Dans le chat, le passage détecté est masqué avec un avertissement plutôt que bloqué, ce qui limite l'effet des faux positifs. Dans une annonce, le contrôle de publication la refuse avec un motif générique (« coordonnées détectées »), que le vendeur corrige avant de soumettre à nouveau (D01, EF-17, EF-36). Récidive signalée à l'admin. Liens non cliquables. Chat limité au texte en V1. Utilisateurs informés de cette analyse automatique (transparence RGPD). | Must | M-CHA-01, M-CHA-02, M-ANN-04 | Démontrée (BP4) pour les annonces ; chat documenté |
| ES-17 | **Upload sécurisé** : seuls les formats d'image matriciels courants sont acceptés, vérifiés sur le contenu et non sur l'extension ; les formats vectoriels sont refusés. Taille et dimensions maximales. Ré-encodage systématique, qui supprime les métadonnées, dont la géolocalisation. Nom de fichier généré par le serveur. Fichiers servis comme contenu non exécutable, sans cookie de session. Une photo n'est publique qu'après validation de l'annonce. Décodage et ré-encodage dans un processus isolé : sans secret, sans accès réseau ni aux données, avec des limites de ressources (mesure compensatoire de D02, maintenue Must avec l'option B, §5.6). | Must | M-ANN-01, M-ANN-02, M-ANN-06, M-ANN-07, M-MOD-03 | Intégrée (D01, D07 : MD5 à MD7) ; décodage isolé documenté (V1) |
| ES-18 | **Contrôle de mise en ligne et revalidation** : toute annonce est contrôlée avant publication. Les règles automatiques passent d'abord (prix hors de la fourchette de la catégorie, vendeur récent avec des annonces de forte valeur, coordonnées détectées ; une catégorie inconnue du contrôle donne « à vérifier », jamais « publié »), avec une revue de l'admin en cas de doute. **Toute modification d'un article publié, quel que soit le champ, le renvoie au contrôle, sans seuil** (D02) ; de même pour un article en contrôle ou à vérifier, avec une nouvelle révision. Les résultats du contrôle et les décisions de l'admin portent la révision examinée : un résultat périmé est ignoré, une décision de l'admin sur une révision périmée est refusée pour conflit (409). Le nom de boutique et les champs de profil visibles des autres ne sont affichés qu'après le filtrage des coordonnées (ES-16), fermé par défaut (ES-31) ; leur modification ne renvoie pas les articles au contrôle. Chaque variation de prix est historisée et transmise à la détection de fraude. | Must (exigence du besoin) | M-ANN-05, M-ANN-04, M-FRA-01, M-FRA-02, M-CAN-02 | Intégrée (D01, D02), sauf historisation des prix, transmission à la fraude, boutiques et profils (fictifs dans le POC) |

**Modération et fraude**

| ID | Exigence | Priorité | Menaces traitées | POC |
|---|---|---|---|---|
| ES-19 | **Signalement et motivation** : bouton de signalement sur les annonces, boutiques, profils et messages. Tout retrait ou suspension est motivé auprès de l'utilisateur concerné (obligations des hébergeurs, §5.3). | Must (obligation légale) | M-CHA-03, M-BO-04, M-FRA-01 | Documentée |
| ES-20 | **Moindre privilège au back-office** : en V1, l'admin assure la gestion et la modération, sans rôle modérateur distinct (D01). Les droits de modération (articles « à vérifier », signalements, conversations signalées) et d'administration (catégories, comptes) sont toutefois modélisés comme des permissions distinctes du rôle admin. La seule modération pourra ainsi être confiée à un futur salarié sans refonte. Une conversation privée n'est accessible que sur signalement, et cet accès est tracé. Motif obligatoire pour tout retrait. Back-office séparé de l'espace public et non référencé. Dans le POC, l'écran « à vérifier » est servi sur l'origine publique : écart assumé, compensé par ES-07 et ES-08 (D02). | Should | M-BO-03, M-BO-01 | Documentée |
| ES-21 | **Intégration fraude fiable et authentifiée** : les événements (nouvelle annonce, variation de prix, nouveau vendeur, transaction) sont émis sans perte silencieuse et peuvent être rejoués. L'interface est authentifiée dans les deux sens, et seuls les verdicts de l'outil authentifié sont acceptés. Les données transmises sont minimisées et couvertes par un contrat de sous-traitance. Le comportement en cas d'indisponibilité de l'outil est défini (par défaut : mise en attente, jamais d'approbation automatique). Aucun outil externe ne se connecte au canal interne : un adaptateur, porté par le service Contrôle de publication (D02), relaie les échanges dans les deux sens, détient seul les identifiants de l'outil et ne publie sur le canal que les sujets qui lui sont attribués (alertes de fraude). La réception des alertes de l'outil n'ouvre, si possible, aucun nouveau point d'entrée public (D05). | Should | M-FRA-04, M-FRA-05, M-FRA-06, M-DIS-04, M-CAN-01 | Documentée (le contrat versionné du contrôle, D01, la prépare ; adaptateur hors POC, D02) |
| ES-22 | **Réputation liée aux transactions** : une seule note par partie et par transaction réellement conclue. Les signaux de faux comptes (auto-achat, même moyen de paiement) sont transmis à la détection de fraude. | Should | M-FRA-03, M-ACC-07 | Documentée |

**Traçabilité**

| ID | Exigence | Priorité | Menaces traitées | POC |
|---|---|---|---|---|
| ES-23 | **Journal d'audit** : événements horodatés, avec leur auteur, pour les connexions réussies et échouées, les changements de rôle, les actions d'administration et de modération (avec motif), les décisions du contrôle de publication, les étapes de commande et de paiement, les variations de prix et les changements de coordonnées de reversement. Journal centralisé, non modifiable par les acteurs journalisés. Ni mot de passe, ni jeton, ni donnée de carte, ni contenu de message. Journaux émis hors du processus de chaque service, vers un stockage en ajout seul : un service compromis ne peut ni modifier ni effacer ses propres traces (mesure compensatoire de D02, maintenue Must avec l'option B, §5.6). Identifiant de corrélation propagé dans les appels et les messages entre services. Durée de conservation définie (repère CNIL : 6 mois à 1 an pour les journaux de sécurité). | Must | M-PAY-02, M-PAY-09, M-BO-01, M-BO-03, M-BO-04, M-LIV-06, M-LIV-07, M-MOD-03, M-OBS-01, M-OBS-02 | Intégrée (D09 : MO1 à MO10) ; ajout seul non démontrable sous l'identité d'exécution partagée : écart assumé (§6.4) |

**Notifications**

| ID | Exigence | Priorité | Menaces traitées | POC |
|---|---|---|---|---|
| ES-24 | **Emails authentifiés et contenu sûr** : domaine d'envoi authentifié par les mécanismes standards (SPF, DKIM, DMARC avec une politique durcie progressivement). Contenu utilisateur échappé et tronqué dans les gabarits, sans aucun lien fourni par un utilisateur. Les emails ne demandent jamais de mot de passe ni de paiement, et les utilisateurs en sont informés. Envoi uniquement vers une adresse vérifiée. | Must | M-NOT-01, M-NOT-02, M-CHA-02 | Documentée |
| ES-25 | **Maîtrise des envois** : plafonds d'envoi par destinataire et au global, regroupement des notifications, respect des préférences, désinscription en un clic, consentement préalable pour les notifications à visée commerciale. | Should | M-NOT-03 | Documentée |

**Données personnelles et publicité**

| ID | Exigence | Priorité | Menaces traitées | POC |
|---|---|---|---|---|
| ES-26 | **Socle RGPD** : registre des traitements. Information des personnes, y compris sur l'analyse automatique des messages et les recommandations. Minimisation : l'adresse de livraison n'est communiquée au vendeur qu'après confirmation du paiement. Aucun traceur non essentiel en V1, les publicités partenaires étant en V2 (D01). Catégories d'intérêt choisies pour ne pas révéler de donnée sensible. Durées de conservation définies (comptes inactifs, messages, journaux, messages du canal et files d'erreurs). Contrats avec les sous-traitants (paiement, fraude, emails, hébergeur). Données hébergées dans l'UE, ou transfert encadré. Procédure de notification d'une violation à la CNIL sous 72 h. | Must (obligation légale) | M-FRA-06, M-PUB-01, M-PUB-03, M-LIV-07, M-CAN-04 | Documentée |
| ES-27 | **Droits des personnes outillés** : export des données ; suppression ou anonymisation du compte, en conservant les pièces comptables obligatoires. | Should | Conformité | Documentée |
| ES-28 | **Publicité ciblée et traceurs**, prérequis à l'activation des publicités partenaires, reportées en V2 (D01) : aucun traceur non essentiel et aucune transmission aux partenaires sans consentement préalable, qu'on peut refuser aussi simplement qu'accepter et retirer à tout moment. Les données transmises se limitent à des segments d'intérêt pseudonymisés, jamais l'email ni l'identité. Aucun script de partenaire sur les pages authentifiées, de paiement ou du back-office. | Later (V2, prérequis d'activation, D01) | M-PUB-01, M-PUB-02, M-PAY-04 | Documentée |

**Disponibilité**

| ID | Exigence | Priorité | Menaces traitées | POC |
|---|---|---|---|---|
| ES-29 | **Limitation de débit et maîtrise de la charge** : limites par origine et par compte sur la connexion, l'inscription, la recherche, l'upload, la soumission d'articles, la réservation, l'envoi de messages et le lancement d'un paiement. Tailles de requête et délais bornés. Mise à l'échelle automatique avec un minimum, un maximum et une alerte de coût. Tests de charge avec des objectifs chiffrés (débit, latence, taux d'erreur) avant la mise en production. | Must | M-DIS-01, M-DIS-02, M-DIS-03, M-PAY-07, M-CHA-04, M-ACC-01, M-CAN-03, M-RES-02, M-DIS-07, M-EXP-02, M-EXP-03, M-OBS-04 | Intégrée (D03, D07, D08, D09, D13, §5.7) ; limite par réplica : écart assumé (§6.4) |
| ES-30 | **Sauvegardes** chiffrées, hors de portée des accès d'exploitation courants, avec une restauration testée avant l'ouverture. Perte de données maximale admissible et durée de reprise définies. | Must | M-DIS-05 | Documentée |
| ES-31 | **Mode dégradé et contrôle fermé par défaut** : un article n'est visible qu'après une décision « publié » (D01). Si le contrôle de publication, la détection de fraude, le paiement ou l'envoi d'emails est indisponible ou en retard, la plateforme met en attente (mise en ligne, commande, notification), puis reprend sans perte au retour du composant (ENF-12). Jamais de validation par défaut. Chaque consommateur est idempotent ; un message rejoué ou en retard ne ramène pas un état antérieur. Un message en échec répété est mis à l'écart après un nombre borné de tentatives, sans bloquer la file ; une alerte signale tout message mis à l'écart et toute file en attente anormale ; le message écarté reste rejouable après correction, pendant une durée bornée (ES-26). | Should | M-DIS-04, M-FRA-04, M-CAN-02, M-CAN-03, M-CAN-08 | Intégrée (D01), pour le contrôle de publication ; option P2 : Interactions arrêté sans perte de notification, message invalide écarté par Interactions après un nombre borné d'essais ; mise à l'écart native chez tous les consommateurs et alerte intégrées (D05 : MC8 ; D09) |

**Chaîne de livraison**

| ID | Exigence | Priorité | Menaces traitées | POC |
|---|---|---|---|---|
| ES-32 | **Protection de la chaîne de livraison** : branche principale protégée sans contournement, fusion seulement après revue et CI verte (PILOTAGE §6). Droits du pipeline, de l'outil de déploiement et des automatismes d'exploitation limités à leur environnement cible. Artefact construit une seule fois, identifié par son empreinte, signé, vérifié avant déploiement et promu tel quel. Second facteur sur le compte de la forge. | Must (D10 : couvre M-LIV-09 et M-LIV-11, Élevées) | M-LIV-05, M-LIV-09, M-LIV-10, M-LIV-11, M-LIV-13, M-IDP-05, M-INF-03 | Intégrée (D10 : MP1 à MP6, MP10 ; D11 : MI5) |

**Later**

| ID | Exigence | Priorité | Menaces traitées |
|---|---|---|---|
| ES-33 | Protection anti-robots adaptative sur l'inscription et l'achat : vérification humaine en cas de volume anormal, limite d'achats simultanés par compte sur les objets rares. | Later | M-ACC-07, M-DIS-06 |
| ES-34 | Détection avancée des contournements : texte incrusté dans les photos, apprentissage des nouvelles formes de masquage à partir des cas modérés. | Later | M-ANN-04, M-CHA-01 (risque résiduel) |
| ES-35 | Test d'intrusion externe avant l'ouverture à grande échelle ; point de contact public pour signaler une vulnérabilité. | Later | Transverse |
| ES-36 | Double validation des actions d'administration à fort impact (suppression d'un vendeur qui a des fonds en séquestre, changement des règles de commission). | Later | M-BO-03 |
| ES-37 | V2, recommandations fondées sur le parcours du catalogue : consentement spécifique, et analyse d'impact (AIPD) à évaluer avant le lancement. | Later | M-PUB-01 |

**Décompte** : 25 Must V1 (ES-01 à ES-10, ES-12 à ES-19, ES-23, ES-24, ES-26, ES-29, ES-30, ES-32, ES-38), 7 Should V1 (ES-11, ES-20, ES-21, ES-22, ES-25, ES-27, ES-31), 6 Later (ES-28, ES-33 à ES-37).

### 5.3 Points réglementaires

Lecture volontairement minimale. **À confirmer avec un juriste avant toute mise en production réelle.**

| Texte | Pourquoi Collector est concerné | Traduction | Non retenu en V1 |
|---|---|---|---|
| RGPD | Données personnelles des acheteurs et vendeurs, profilage d'intérêts | ES-26, ES-27, ES-16 (transparence), ES-21 (sous-traitance) | AIPD formelle : à évaluer avant d'activer la publicité ciblée et avant la V2 (ES-37) |
| Traceurs (art. 82 de la loi Informatique et Libertés, lignes directrices CNIL) | Mesure d'audience ; publicité ciblée en V2 (D01) | V1 : aucun traceur non essentiel (ES-26). V2 : ES-28, prérequis d'activation des publicités | — |
| PCI-DSS | Paiement par carte | ES-12. Déléguer la saisie de la carte réduit le périmètre au questionnaire d'auto-évaluation le plus léger. **Le périmètre ne devient pas nul** : la plateforme reste responsable de l'intégrité de la page qui renvoie vers le prestataire (ES-07, ES-08 ; ES-28 quand les publicités seront activées). Avec l'option B, la clé du prestataire et la réception de ses notifications sont isolées dans Transactions : le rayon d'impact baisse, le questionnaire applicable ne change pas (à confirmer avec le prestataire, D02). | Certification propre de Collector |
| DSP2 | Paiement par carte ; encaissement pour le compte des vendeurs, qui est un service de paiement réglementé | ES-12 : authentification forte, encaissement et reversements via l'offre place de marché d'un prestataire agréé ; ES-15 : fonds conservés chez le prestataire | Agrément propre d'établissement de paiement (sur-dimensionné) |
| DSA et LCEN | Hébergement de contenus de tiers (annonces, photos, messages) | ES-19 (signalement, motivation des décisions). ES-23 : conservation des données d'identification des auteurs de contenus, durée légale à confirmer | Obligations renforcées des plateformes de taille moyenne et plus (système interne de contestation, rapports de transparence) : exemption des petites entreprises à confirmer |
| Prospection électronique (CPCE, art. L34-5) | Notifications à visée commerciale | ES-25 | — |

### 5.4 Contrôle de couverture

- Chaque menace **Critique** ou **Élevée** applicable à la V1 est traitée par au moins une exigence **Must V1**. La vérification a été faite ligne à ligne entre les §4 et §5.2, puis refaite pour la révision de l'option B. Les deux menaces Élevées ajoutées sont couvertes : M-MOD-01 par ES-38 ; M-MOD-03 par ES-38, ES-17 (décodage isolé) et ES-23 (journaux hors du processus).
- **M-PUB-01** (partie partenaires et traceurs) et **M-PUB-02** sont sans objet en V1 : les publicités partenaires sont reportées en V2 (D01). ES-28 est le prérequis de leur activation. En V1, ES-26 (aucun traceur non essentiel, minimisation) couvre M-PUB-01, et ES-08 (scripts limités à ceux de la plateforme) couvre M-PUB-02.
- **M-PAY-06** : le scénario de transfert de valeur vers les frais de port est traité par la décision sur l'assiette de la commission (D01, ES-13). Les valeurs aberrantes restent couvertes par les bornes d'ES-13.
- Chaque menace **Modérée** est traitée par au moins une exigence Must ou Should. Pour le §4.11 : M-CAN-01 (ES-38, ES-21), M-CAN-02 (ES-18, ES-31), M-CAN-03 (ES-29, ES-31), M-CAN-04 (ES-26), M-CAN-05 et M-CAN-06 (ES-38), M-MOD-02 (ES-04, ES-38), M-SYN-01 (ES-38), M-RES-01 (ES-14), M-RES-02 (ES-14, ES-29).
- **Révision D03 à D14.** Les six menaces Élevées ajoutées sont couvertes par des Must : M-CAN-07 (ES-38, ES-06, ES-07 ; MC3 à MC5), M-IDP-01 (ES-07, ES-10 ; MA7, MA9, ME8), M-IDP-04 (ES-06, ES-09, ES-10 ; MA10, MX1), M-EXP-01 (ES-07, ES-38 ; ME8), M-LIV-09 (ES-05, ES-32 ; MP2, MP3, MP6), M-LIV-11 (ES-32, ES-06 ; MP4). Pour tenir la règle du §5.1, **ES-32 passe de Should à Must**, M-LIV-09 et M-LIV-11 en dépendant. Modérées : M-CAN-08 (ES-31, ES-26 ; MC8), M-IDP-02 (ES-02, ES-38 ; MA4, MA5), M-IDP-03 (ES-38 ; résiduel accepté, D06), M-IDP-05 (ES-05, ES-32 ; MA1), M-DIS-07 (ES-29 ; MD4), M-EXP-02 et 03 (ES-29, ES-23 ; ME4), M-OBS-01 à 03 (ES-23, ES-05 ; MO1, MO2, MO7, MO10), M-OBS-04 (ES-29 ; MO5, MO8), M-LIV-10 et 13 (ES-32 ; MP5, MP6, MP10), M-LIV-12 (ES-06 ; MP8), M-LIV-14 (ES-05 ; MQ5, MT7), M-INF-01 et 02 (ES-06 ; MI1 à MI4, MI6), M-INF-03 (ES-32 ; MI5), M-INF-04 et 05 (ES-07 ; MI8).
- Les trois menaces **Faibles** sont couvertes par des Must : M-ACC-06 par ES-09, M-PUB-03 par ES-26, M-EXP-04 par ES-07 (ME10).
- Risques résiduels connus, à suivre dans l'audit V1 :
  - le filtrage des coordonnées (ES-16) ne sera jamais exhaustif. Le séquestre (ES-15, décidé) rend le paiement hors plateforme moins intéressant, et ES-34 traite le résiduel ;
  - la fraude vendeur (M-FRA-01) reste possible, mais son coût est borné par le séquestre ;
  - dans le POC, l'identité d'exécution partagée annule le cloisonnement par l'hébergeur entre services (ES-38) : écart connu, remédiation en V1 ;
  - l'isolement du périmètre financier n'est démontré que par la structure (§6) : sans prestataire, la protection de sa clé et de ses notifications reste à prouver en V1.

### 5.5 Entrées pour les décisions

Table d'entrée rédigée avant les décisions. D03 à D14 sont désormais tranchées (voir l'[index](../decisions/INDEX.md)) ; D03, D05, D06 et D08 restent Candidate jusqu'aux expérimentations XP01 à XP03 (D14). Leurs mesures sont reportées au §5.7.

| Décision | Exigences à prendre en compte |
|---|---|
| D01 Exigences et fonctionnalité du POC | **Décidée le 2026-10-09** ([D01](../decisions/D01-fonctionnalite-poc.md)) : « A enrichie ». Reportée dans ES-03, ES-13, ES-15, ES-16, ES-17, ES-18, ES-20, ES-26, ES-28, ES-31 et en §6 |
| D02 Architecture cible | **Décidée le 2026-10-09** ([D02](../decisions/D02-decoupage-applicatif.md)) : option B, walking skeleton WS-2, complément du POC option P2. Reportée en §1.1, §2, §3 (FC11), §4.11, ES-04, ES-07, ES-08, ES-11, ES-13, ES-14, ES-17, ES-18, ES-20, ES-21, ES-23, ES-31, ES-38, §5.6 et §6 |
| D03 Hébergement et orchestration | Exposition minimale (ES-07), localisation des données (ES-26), plafonds de mise à l'échelle (ES-29), sauvegardes (ES-30), identité d'exécution par service et écart du rôle unique (ES-38) |
| D05 Communication interservice | Canal asynchrone soumission → contrôle sans perte et fermé par défaut (ES-31, ENF-12), événements fiables vers la détection de fraude (ES-21), idempotence (ES-14) ; diffusion avec une identité par service et des droits par sujet (ES-38) ; mise à l'écart d'un message en échec répété et alerte sur la file (ES-31) ; conservation bornée des messages (ES-26) ; réception des alertes de l'outil par l'adaptateur (ES-21) |
| D06 Authentification et autorisation | ES-02, ES-03, ES-04, ES-09, ES-10, ES-11 ; permissions de modération et d'administration au sein du rôle admin (ES-20) ; jeton de service de l'appel de réservation (audience, droit limité, courte durée) et revérification du jeton de l'acheteur (ES-38) |
| D07 Données | Photos non publiques avant validation (ES-17), chiffrement au repos (ES-07), sauvegardes (ES-30), un compte d'accès par service (ES-38) |
| D08 Exposition, TLS, domaine | ES-01, ES-07, domaine d'envoi des emails (ES-24) ; route de réservation absente du point d'entrée public (ES-38) ; route dédiée aux notifications du prestataire, vers Transactions seul (ES-14) |
| D09 Observabilité | ES-23 : journaux hors du processus, en ajout seul ; identifiant de corrélation ; alertes sur les files (ES-31) |
| D10 CI/CD et portes DevSecOps | ES-05, ES-06, ES-32 |
| D11 IaC et garde-fou coût | Configuration analysée (ES-05, ES-07), plafonds et alerte de coût (ES-29) |
| D12 Indicateurs | Indicateurs sécurité candidats : vulnérabilités critiques et élevées ouvertes, délai de correction, part des exigences Must prouvées par un test automatisé |
| D13 Stratégie de test et charge | Tests d'autorisation négatifs (ES-03, ES-04), tests de charge (ES-29), vague de soumissions avec photos (ES-16, ES-17, ES-31) ; tests négatifs de service à service (ES-38) ; tests de concurrence et de nouvel essai sur la réservation (ES-14) ; un test de contrat par paire producteur-consommateur |

### 5.6 Mesures compensatoires de D02, réévaluées pour l'option B

L'avis sécurité de D02 (§ 2) proposait quatre mesures, à rendre Must V1 si le périmètre financier restait dans le déployable unique de l'option A. L'option B l'extrait (Transactions). Chaque mesure est réévaluée selon ce que B change à la menace qu'elle visait.

| Mesure (avis D02) | Ce que B change | Verdict | Où |
|---|---|---|---|
| 1. Décodage isolé des images | Rien sur le vecteur : le décodage de fichiers non fiables reste le point d'entrée le plus probable d'une exécution de code (M-ANN-01, M-MOD-03), dans la Place de marché, le service le plus exposé. B protège la finance, pas les profils, intérêts et notes que détient la Place de marché : l'impact reste une fuite massive. | **Must V1** | ES-17 (documentée pour le POC) |
| 2. Clé du prestataire restreinte et plafonnée | La clé n'est plus dans le service le plus exposé, mais dans Transactions seul. La mesure ne borne plus que la fuite de la clé elle-même (M-LIV-02) ou une compromission de Transactions, déjà traitées par des Must (ES-05, ES-06). Elle reste une défense en profondeur peu coûteuse, surtout de configuration chez le prestataire. | **Should V1** | ES-11 |
| 3. Compte d'accès aux données distinct pour Commandes et paiement | Intégrée à B et généralisée : un compte par service, limité à son espace (M-MOD-01). | **Must V1**, généralisée | ES-38 |
| 4. Journaux de sécurité hors du processus, en ajout seul | Inchangée : un service compromis, quel qu'il soit, ne doit pas pouvoir effacer ses traces (M-MOD-03, M-LIV-06). Avec quatre services, la centralisation est de toute façon nécessaire. | **Must V1** | ES-23 |

Les déclencheurs du critère E3 listés par l'avis deviennent sans objet : l'extraction du périmètre financier est faite dès la V1.

### 5.7 Correspondance exigence → mesures des décisions → preuve (révision du 2026-10-26)

Préfixes des mesures : MS (D03), MC (D05), MA (D06), MD (D07), ME (D08), MO (D09), MP (D10), MI (D11), MQ (D12), MT (D13), MX (D14, bac à sable seulement) ; les conditions de l'avis de D04 sont citées « D04 ». Preuves : **porte** = porte CI bloquante (D10) ; **XP** = critère d'expérimentation (D14), bloquant quand D14 le classe ainsi ; **AC** = test d'acceptation ou d'intégration en CI (D13) ; **inventaire** = `status.sh --strict` ou inventaire des points d'entrée (D11). Statut : vocabulaire du §5.1 ; un écart assumé renvoie au §6.4. Une mesure sans sa preuve ne réduit pas le risque (§1.3).

| Exigence | Mesures des décisions | Preuve attendue | Statut |
|---|---|---|---|
| ES-01 Chiffrement | MS2, MS7, MD3, ME1, ME7 ; flux internes confinés : MS4, MS5, MC3 | XE3 (TLS 1.2 minimum, 80 redirigé, HSTS) ; AC : connexion à la base sans TLS refusée ; XP X12, X19 (bloquants) | Démontrée (BP1) ; écart : TLS interne absent |
| ES-02 Authentification | D04 (PKCE, jetons en mémoire), MA1, MA2, MA4, MA11, ME6 | AC BP2 (sans jeton, expiré, signature invalide) ; AC : jeton de rafraîchissement rejoué refusé (MT2) ; porte : contrôle du realm (MA1), options `Skip*` des vérificateurs interdites (Semgrep) | Démontrée (BP2) ; écart : BFF en V1 |
| ES-03 Rôles | MA3, MA5, MA9 | AC négatifs de rôle (D13 §2) ; AC : jeton de `transactions` refusé sur une route d'acheteur | Démontrée (BP2, BP5) |
| ES-04 Objet | MA6, MD7, MT1 | AC BP5 ; AC : URL d'une photo non publiée refusée à un autre vendeur ; AC : adjoint confus (article non publié, acheteur substitué malgré un jeton de service valide) | Démontrée (BP5) |
| ES-05 Analyse de vulnérabilités | D04 (CI), MS9, MA14, MP3, MP7, MP9, MI7, MQ1 à MQ3, MQ5, MQ6 | Portes : govulncheck, Trivy (images et configuration), npm audit, OSV-Scanner, gosec, Semgrep, gitleaks, zizmor ou actionlint ; ZAP baseline par sprint (non bloquant) ; branche de démonstration rouge (BP3) | Démontrée (BP3) |
| ES-06 Secrets | MS6, MC4, MC5, MD2, MI1 à MI4, MI6, MP8, MT5, MX4, MX5 | Porte gitleaks (historique complet) ; X20, AC bloquant de la première US d'infrastructure ; `terraform show -json` sans secret (MI4) ; valeur piégée absente des journaux | Démontrée (BP3) ; écart : `LabRole` lit tous les secrets |
| ES-07 Exposition | MS1, MS2, MS5, MS7, MC3, MA7, MD6, MD8, ME1 à ME5, ME8, ME10, MO4, MI7 à MI10, MX2, MX3 | X17, AC bloquant avant toute ouverture de l'ALB ; XE4 étendu par ME8 (bloquant) ; X18 ; ZAP sur une page 200 et une 404 ; porte `trivy config` ; inventaire | Intégrée ; écart : `nosniff` absent des photos |
| ES-08 Entrées et sorties | D04 (lint front, CSP), ME2, MC6, MT3 | Porte ESLint (`react/no-danger` et règles associées) ; Playwright échoue à toute violation de CSP ; AC : clés dupliquées refusées sur les messages et les corps d'API | Intégrée |
| ES-09 Parcours de compte | MA13, ME4 | AC : rafale de connexions échouées freinée sans verrouillage (§7.2) ; message identique que le compte existe ou non | Partielle ; écart : pas de limite par origine |
| ES-10 Second facteur admin | MA9, MA12, MO6 | AC : OTP demandé à l'admin, pas au vendeur (XP03, non bloquant) | Intégrée ; nouvelle authentification : V1 |
| ES-13, ES-14 Montants, réservation | D07 (`UPDATE` conditionnel), MC10, MT1 | AC : concurrence, nouvel essai, échéance, article figé, montants au centime ; V1 : test d'architecture « aucun consommateur n'importe le client du prestataire » | Partielle (option P2) |
| ES-17 Upload | D04 (`CGO_ENABLED=0`), MD5 à MD7, MD9, MT3 | AC : GIF, WebP et SVG refusés ; bombe de décompression refusée ; 413 au-delà de la taille ; photo publiée sans géolocalisation ; inventaire du bucket | Intégrée ; écart : décodage non isolé |
| ES-18 Revalidation | MC2, MC6, MC9 | AC : résultat périmé ou redélivré sans effet (MT2) ; décision périmée refusée (409) | Intégrée |
| ES-23 Journal d'audit | MS8, MC9, MA6, MO1 à MO3, MO6, MO7, MO10 | AC : `audit=true, service=transactions` émis par la Place de marché rangé sous la Place de marché (MO2) ; X21 ; valeur piégée absente de Logs Insights ; inventaire des groupes et de leur rétention | Intégrée ; écart : « ajout seul » non démontrable |
| ES-26 RGPD | MC8 (identifiants opaques, bornes de conservation), MO7 | XC3, XC4 ; inventaire des rétentions | Documentée (données fictives) ; bornes du canal intégrées |
| ES-29 Débit et charge | MS5, MS10, MD4, ME4, MO5, MO8, MT4, MT6 | XE5 et MT4 (429, `X-Forwarded-For` forgé en valeur unique et en liste, 429 rejoué après le tir) ; tir k6 (D13) ; 10 000 chemins aléatoires sans nouvelle série | Intégrée ; écart : limite par réplica |
| ES-30 Sauvegardes | MD10 | Procédure V1 ; exercice de restauration facultatif | Documentée |
| ES-31 Mode dégradé | MC6, MC8, MC9 | XC3, XC4, XC5 ; alerte d'entrée en `.ecartes` en moins de 2 min | Intégrée |
| ES-32 Chaîne de livraison | MP1 à MP6, MP10, MI5 | Export des règles, PR rouge non fusionnable ; `cosign verify` dans `up.sh`, image non signée refusée ; `kubectl auth can-i --list` du compte de Flux | Intégrée |
| ES-38 Identités de service | MS3, MS4, MC1, MC2, MC7, MA3 à MA6, MD1, MO1, MO9 | XP bloquants : X16 élargi, X12, X19, XC2, XC7, XC9 ; AC : « réserver » refusé selon le jeton ; accès croisé entre bases et `CREATE TABLE` refusés | Partielle ; écart : `LabRole` partagé |

Sans mesure nouvelle dans D03 à D14, statut inchangé : ES-11, ES-12, ES-15, ES-16 (BP4 pour les annonces), ES-19 à ES-22, ES-24, ES-25, ES-27, ES-28, ES-33 à ES-37. Pour ES-21, D05 précise que l'adaptateur interrogera l'outil de fraude plutôt que d'ouvrir un point d'entrée public (V1).

## 6. Sécurité intégrée au POC (critère 8)

Décision de Martin du 2026-10-09 ([D01](../decisions/D01-fonctionnalite-poc.md)) : la fonctionnalité du POC est la mise en vente d'un article avec contrôle automatisé de publication (« A enrichie »). Le POC démontre **cinq bonnes pratiques** de sécurité, là où le critère 8 en demande au moins deux.

Décisions de Martin du 2026-10-09 ([D02](../decisions/D02-decoupage-applicatif.md)) : option B, walking skeleton WS-2 et, pour le complément du POC, **option P2**. Le POC déploie les quatre services : la Place de marché (annonces et catalogue, comptes minimal), le Contrôle de publication, Interactions minimal (diffusion de la « décision de publication », notification au vendeur) et Transactions réduit à la réservation, avec l'appel synchrone authentifié Transactions → Place de marché. Les mesures propres à B (§6.2) sont intégrées sous réserve du rôle d'exécution unique de l'environnement de formation ; le canal retenu en D05 (RabbitMQ) authentifie chaque service.

**Isolement du périmètre financier (point K5 de D02) : structure seulement.** Transactions est un service distinct, avec ses données, son identité et son compte d'accès. Sans prestataire de paiement, ni sa clé, ni la réception de ses notifications, ni le séquestre ne sont présents : le POC n'éprouve pas leur isolement, qui reste documenté (ES-12, ES-14, ES-15, §6.4). De plus, sous l'identité d'exécution partagée, un service compromis atteint les secrets de Transactions : présenter cet isolement comme acquis dans le POC le surévaluerait.

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
- Test automatisé : un appel sans jeton, avec un jeton expiré ou avec une signature invalide est refusé ; le même appel est accepté avec un jeton valide et le bon rôle. Rejoué sur les API d'Interactions et de Transactions, qui vérifient elles-mêmes le jeton (option P2).
- Démonstration : connexion par le serveur d'autorisation ; la même requête est refusée sans jeton, acceptée avec.

**BP3 — Scan bloquant avec détection de secrets (ES-05, ES-06)**
- Menaces : M-LIV-01, M-LIV-02, M-LIV-03, M-LIV-08.
- AC (US d'infrastructure) : une vulnérabilité critique ou élevée corrigeable, ou un secret détecté, fait échouer le pipeline et bloque la fusion (ENF-05, ENF-06).
- Test automatisé : dépendances, images, code, configuration et secrets analysés à chaque exécution ; rapports archivés.
- Démonstration : sur une branche de démonstration, une dépendance vulnérable connue et un secret fictif rendent le pipeline rouge, alors que `main` reste vert.

**BP4 — Filtrage des coordonnées dans les annonces (ES-16)**
- Menaces : M-ANN-04 ; mécanisme de M-CHA-01, que le chat réutilisera.
- AC : un article dont le titre ou la description contient un email, un numéro de téléphone ou un lien externe, en clair ou masqué (« zéro six… », « arobase », espaces), est refusé avec le motif « coordonnées détectées ». Une fois corrigé, il peut être soumis à nouveau (EF-17, EF-18, EF-36). Les boutiques sont fictives dans le POC (D02) : le filtrage du nom de boutique relève de la V1 (ES-18).
- Test automatisé : jeu de cas positifs (variantes masquées) et négatifs (dates, dimensions, références d'édition) ; temps de détection borné sur un texte de longueur maximale.
- Démonstration : une description « jean point dupont arobase… » est refusée avec son motif, puis l'article est publié après correction.

**BP5 — Contrôle d'accès par propriété et par rôle (ES-04 avec ES-03)**
- Menaces : M-ACC-02, M-ANN-06, M-BO-02, M-MOD-02.
- AC : un vendeur ne consulte ni ne modifie l'article ou le brouillon d'un autre vendeur ; seul l'admin tranche un article « à vérifier » ; un article non publié et ses photos sont invisibles des autres utilisateurs (ENF-03, EF-19).
- Test automatisé : tests négatifs par type de ressource et par fonction, tous refusés : vendeur B sur l'article du vendeur A, vendeur sur la décision admin, anonyme sur un brouillon ou une photo non publiée, vendeur B sur les notifications du vendeur A (Interactions), réservation transmise par Transactions avec une identité d'acheteur forgée (option P2).
- Démonstration : la même requête est acceptée avec le jeton du vendeur A, puis refusée avec celui du vendeur B.

### 6.2 Mesures intégrées au POC (D01, D02)

Ces mesures ne sont pas présentées comme bonnes pratiques. Elles traitent les risques que la fonctionnalité introduit dans le POC et sont testées en CI.

| Mesure | Exigence | Menaces | Preuve attendue (AC et test automatisé) |
|---|---|---|---|
| Upload sécurisé minimal | ES-17 | M-ANN-01, M-ANN-02, M-ANN-06, M-ANN-07 | Refus d'un fichier dont le contenu n'est pas une image, même avec une extension d'image ; refus d'une image vectorielle ; refus d'un fichier au-delà de la taille maximale ; une photo contenant des coordonnées GPS fictives est publiée sans métadonnées ; nom de fichier généré par le serveur ; photo d'un article non publié inaccessible sans droit |
| Contrôle avant publication, revalidation sans seuil, décision sur une révision | ES-18 | M-ANN-05, M-FRA-02, M-FRA-01 (en partie), M-CAN-02 | Règles déterministes : un prix hors de la plage de la catégorie donne « à vérifier », une catégorie inconnue du contrôle aussi. Toute modification d'un article publié, y compris du seul titre, le retire du catalogue et le renvoie « en contrôle », sans seuil. Une décision de l'admin sur une révision périmée est refusée (409). Un résultat de contrôle périmé ou livré deux fois est sans effet |
| Contrôle fermé par défaut et consommation sans blocage | ES-31 | M-DIS-04, M-FRA-04, M-CAN-03 | Composant de contrôle arrêté : l'article soumis reste invisible. Au redémarrage, la soumission est traitée sans perte (ENF-12). Interactions arrêté : publications non affectées, notifications créées à son retour, sans perte ni doublon ; un message invalide est écarté par Interactions après un nombre borné d'essais, sans bloquer les suivants (option P2) |
| Neutralisation des sorties et politique de sécurité du contenu | ES-07, ES-08 | M-ANN-03, M-BO-01, M-PAY-04, M-NOT-02 | Un titre contenant du balisage s'affiche comme du texte, au catalogue et sur l'écran « à vérifier », servi sur l'origine publique ; la politique de sécurité du contenu n'autorise que les scripts de la plateforme ; en-têtes de sécurité présents ; texte des notifications construit à partir de la catégorie de motif, sans contenu saisi par un utilisateur |
| Identités de service et cloisonnement | ES-38, ES-04 | M-CAN-01, M-CAN-05, M-CAN-06, M-MOD-01, M-MOD-02, M-SYN-01 | Canal : une écriture par une identité qui n'est pas le producteur du sujet est refusée (Interactions ne peut pas écrire un résultat de contrôle ; le Contrôle et Interactions ne peuvent pas publier « décision de publication ») ; Interactions valide le contrat à la réception (version, champs requis, issue et catégorie de motif dans des listes fermées) ; aucun service ne peut créer d'abonnement, l'abonnement de test a une identité en lecture seule et est retiré après le test. Données : la base refuse au compte d'Interactions l'espace de la Place de marché, et inversement ; de même pour Transactions ; le Contrôle n'a aucun accès. Appel « réserver » : refusé sans jeton de service, avec le jeton d'un autre service ou d'un utilisateur, avec un jeton expiré ou destiné à une autre audience ; jeton de l'acheteur revérifié par la Place de marché, identité d'acheteur forgée refusée, prix de référence lu par la Place de marché dans ses propres données. Point d'entrée public : la route « réserver » de la Place de marché est absente de l'inventaire des points d'entrée exposés (§7.1) ; un appel à cette route depuis Internet échoue faute de route. Sous réserve du rôle d'exécution unique ; canal retenu en D05 (RabbitMQ) avec identité par service et émetteur vérifié (MC1, MC2, XC2) |
| Réservation à échéance | ES-14, ES-13 | M-RES-01, M-RES-02, M-PAY-05, M-PAY-01 | Seul un article publié et disponible se réserve. Test de concurrence : deux réservations simultanées du même article, une seule acceptée. Nouvel essai de l'appel « réserver » pour le même acheteur et le même article : la réservation existante est renvoyée, sans seconde réservation ni seconde commande en attente (décision par défaut retenue par Martin). Une seule réservation active par acheteur. Échéance fixée par la Place de marché et plafonnée ; la réservation tombe à l'échéance. Modification et retrait de l'article refusés au vendeur pendant la réservation, prix figé (D02, ES-13). Commande en attente au prix de référence, montants et commission calculés côté serveur au centime |

Le contrat versionné du contrôle, prévu par D01 comme point de branchement de l'outil de fraude, prépare ES-21. Il ne prouve pas l'authentification de l'interface, qui restera à faire quand l'outil sera branché. L'adaptateur, porté par le Contrôle en V1 (D02), est hors POC.

### 6.3 Compléments recommandés, non tranchés

ES-07 et ES-08, recommandées dans une version précédente, sont intégrées (D02, §6.2). De même, la route « réserver » absente du point d'entrée public (ES-38, M-SYN-01) et le nouvel essai de « réserver » qui renvoie la réservation existante (ES-14, M-RES-01) sont intégrés (§6.2).

Révision du 2026-10-26 : les quatre compléments qui restaient recommandés sont tranchés et intégrés : ES-10 (D06 : MA12), ES-31 chez tous les consommateurs (D05 : MC8), ES-23 (D09 : MO1 à MO10), ES-29 (D08 : ME4 ; D13). Restent recommandés, non exigés :
- décodage des images dans un Deployment distinct, sans IMDS, sans secret ni sortie réseau (D07, environ 0,5 à 1 jour) ;
- artefact OCI signé et vérifié par Flux, en plus de la vérification par `up.sh` (D10).

### 6.4 Hors POC

Ces exigences sont documentées et seront auditées comme des écarts en phase 3 :
- paiement et commission (ES-12 ; ES-13 et ES-14 au-delà de la commande en attente et de la réservation, intégrées avec l'option P2) ;
- séquestre (ES-15) ;
- filtrage dans le chat (partie chat d'ES-16) ;
- signalement (ES-19) ;
- emails (ES-24) ;
- socle RGPD (ES-26) ;
- sauvegardes (ES-30) ;
- adaptateur et authentification de l'outil de fraude (ES-21) ;
- décodage isolé des images (complément d'ES-17) ;
- clé du prestataire restreinte et plafonnée (ES-11) ;
- identité d'exécution par service chez l'hébergeur (ES-38).

Le premier risque critique, **M-PAY-02**, n'est donc pas couvert par le POC. Il deviendra une action P1 justifiée du plan de remédiation.

**Écarts assumés entre le POC et la V1** (D02, complétés par D03 à D14), repris dans l'audit V1 avec leur cible :
- l'écran « à vérifier » est servi sur l'origine publique, protégé par le rôle admin, ES-07 et ES-08 ; en V1, le back-office a sa propre origine (ES-20) ;
- **`LabRole` partagé** : tous les services partagent l'identité d'exécution imposée par l'environnement de formation ; la Place de marché (S3) et le collecteur de journaux l'obtiennent par l'IMDS (MS3, MO1), et ce rôle lit tous les secrets, dont celui du client `transactions` et le mot de passe maître de la base. Les autres pods n'y accèdent pas si X16 réussit. V1 : une identité d'exécution par service, politique de bucket limitée à la Place de marché (ES-38) ;
- l'échéance de réservation est courte (quelques minutes) pour la démonstration ; en V1, elle couvre la durée de la session de paiement et le délai de livraison des événements (ES-14) ;
- l'isolement du périmètre financier n'est démontré que par la structure (§6) ;
- **TLS interne absent** : AMQP, « réserver » et le flux de l'ALB vers les nœuds circulent en clair dans le VPC, confinés par MS2, MS4, MS5 et MC3 ; Transactions n'authentifie pas la Place de marché (M-SYN-01). V1 : TLS interne et TLS jusqu'aux cibles de l'ALB (ES-01) ;
- **« ajout seul » des journaux non démontrable** : `LabRole` peut supprimer un groupe de journaux ou réduire sa rétention (M-OBS-02), borné par MS3 et contrôlé par l'inventaire (MO7). V1 : identité du collecteur limitée à l'écriture, refus de `logs:Delete*`, export verrouillé, clé KMS gérée par le client, audit conservé 1 an (ES-23, MO10) ;
- **BFF en V1** : jetons en mémoire de l'application statique ; un script injecté peut utiliser le jeton pendant la session, et un jeton d'accès reste valable 5 min après la déconnexion. Bornes : CSP stricte sur les scripts, jetons de 5 min, rotation avec détection de réutilisation (MA11). V1 : intermédiaire serveur à cookie `HttpOnly` (ES-02) ;
- **décodage des images dans le processus** de la Place de marché, avec les bornes de MD5. V1 : processus isolé (ES-17, §5.6) ;
- **`nosniff` absent des photos** servies par URL présignée, compensé par le type fixé par le serveur, le ré-encodage et une origine distincte sans cookie. V1 : CDN avec politique d'en-têtes (ES-17, MD9) ;
- **limitation de débit par réplica** (limite effective multipliée par 6 au plus), sans limite par origine sur la connexion ; adresse du client peut-être fausse dans les événements du serveur d'autorisation, à vérifier. V1 : règle WAF à base de débit (ES-09, ES-29, ME9) ;
- **sauvegardes** : base recréée à chaque session, instantanés supprimables sous `LabRole` ; ES-30 non démontrable (MD10) ;
- **dépôt public** (D10) : résultats publiés limités à des compteurs (MQ5, MT7). V1 : dépôt privé ;
- **repli de MS3** : s'il s'applique, le collecteur de journaux passe en `hostNetwork` et échappe aux politiques réseau (MO10) ;
- **signature des images** vérifiée par `up.sh`, sans contrôleur d'admission (MP6). V1 : vérification par le cluster.

## 7. Lien avec la phase 3 (audit V1 et plan de remédiation)

### 7.1 Preuves à collecter pendant le développement

| Preuve | Produite par | Quand | Exigences | Usage dans l'audit V1 |
|---|---|---|---|---|
| Rapports d'analyse des dépendances et des images | Pipeline | Chaque exécution, rapports conservés | ES-05 | Inventaire des vulnérabilités connues (CVSS), tendance, délai de correction |
| Rapport d'analyse statique orientée sécurité | Pipeline | Chaque exécution | ES-05, ES-08 | Failles de code (injection, neutralisation des sorties) |
| Détection de secrets, historique compris | Pipeline, plus un passage complet avant l'audit | Chaque exécution | ES-06 | Absence de secret, ou incident et révocation |
| Analyse de la configuration décrite en code | Pipeline | Chaque modification d'infrastructure | ES-07, ES-29 | Exposition, chiffrement au repos, plafonds de mise à l'échelle |
| Analyse dynamique non intrusive de l'application déployée | Environnement déployé | Fin de sprint | ES-01, ES-07, ES-08 | Configuration TLS, en-têtes, messages d'erreur, points d'entrée |
| Résultats des tests de sécurité automatisés | CI | Chaque exécution | ES-02, ES-03, ES-04, ES-07, ES-08, ES-14, ES-16, ES-17, ES-18, ES-31, ES-38 | Refus sans jeton, refus avec le mauvais rôle, refus sur la ressource d'autrui, coordonnées détectées, fichier non image refusé, métadonnées supprimées, balisage affiché comme du texte, revalidation sans seuil, décision sur une révision périmée refusée (409), contrôle fermé par défaut, Interactions arrêté sans perte, écriture refusée sur un sujet qui n'est pas le sien, accès croisé aux données refusé, appel « réserver » refusé sans jeton de service valide ou avec une identité d'acheteur forgée, double réservation concurrente refusée, nouvel essai de « réserver » sans seconde réservation, une seule réservation active par acheteur, échéance plafonnée, article figé pendant la réservation, message invalide écarté sans bloquer la file (§6.1, §6.2) |
| Résultats des tests de contrat | CI | Chaque exécution | ES-31, ES-38 | Un test par paire producteur-consommateur (Place de marché ↔ Contrôle, Place de marché → Interactions, Transactions → Place de marché) ; contrat et émetteur attendus |
| Matrice des droits par service | Environnement déployé, configuration décrite en code | Fin de sprint | ES-06, ES-38 | Droits attendus et constatés par service sur le canal (sujets en écriture et en lecture), les données, le stockage et les secrets ; écart du rôle d'exécution unique documenté |
| Résultats des expérimentations du canal et du serveur d'autorisation | Expérimentations (D14) | Avant J4 | ES-38 | Diffusion et droits par identité sur le canal retenu ; jeton de service et propagation de l'identité de l'acheteur |
| Résultats des tests de charge | Campagne dédiée | Sprint 2, puis avant l'audit | ES-29, ES-09, ES-31 | Voir §7.2 |
| Extraits de journaux | Observabilité | Fin de sprint | ES-23, ES-31 | Présence des événements de sécurité, absence de données sensibles, journaux émis hors du processus des services, identifiant de corrélation suivi d'un service à l'autre, alertes sur les files |
| Inventaire des points d'entrée exposés | Environnement déployé | Fin de sprint | ES-07, ES-38 | Surface d'attaque réelle, comparée à la surface attendue ; route « réserver » non exposée |
| Registre des failles | `docs/audits/sprint-N-securite.md` | Fin de sprint | Toutes | Historique, statut, revérification |
| Matrice exigence → preuve → statut | Audit V1 | Jalon J6 | Toutes | Point de départ du plan de remédiation |

### 7.2 Ce que les tests de charge apportent à la sécurité

- **Point de saturation et ressource limitante** (base, stockage, connexions, calcul) : c'est la cible d'un déni de service applicatif (M-DIS-02).
- **Comportement au-delà du seuil** : refus propres (code de limitation) ou bien erreurs serveur, traces techniques exposées, dégradation durable (ES-07, ES-29).
- **Efficacité de la limitation de débit** sur la connexion, la recherche et l'upload (ES-29).
- **Rafale de connexions échouées** : la protection doit ralentir l'attaquant sans verrouiller les comptes légitimes (ES-09).
- **Vague de soumissions avec photos** (scénario complémentaire de D01) : coût du ré-encodage des photos et de la détection des coordonnées ; aucun article publié sans contrôle sous charge (ES-16, ES-17, ES-31) ; file du contrôle non bloquée par un message en échec, alertes déclenchées (M-CAN-03).
- **Mise à l'échelle** : délai de réaction, plafond atteint, coût de la campagne extrapolé (M-DIS-03).
- Pour chaque observation, noter le seuil chiffré, joindre une capture et une interprétation. Chaque observation devient un constat de l'audit V1.

**Réservations concurrentes : test de concurrence, pas tir de charge.** Le tir de charge de démonstration (D02) porte sur la consultation du catalogue, car chaque réservation consomme un objet unique. La preuve de M-RES-01 est le **test de concurrence automatisé** en CI (ES-14, §6.2, §7.1) : deux réservations simultanées du même article, une seule acceptée. Une campagne de sécurité facultative, hors D02, peut compléter cette preuve par une rafale de réservations concurrentes, données rechargées avant chaque passage : aucune double réservation, refus propres.

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

- **Technologies retenues** (D03 à D14) : les menaces propres au découpage (D02, §4.11) et aux technologies (orchestrateur, canal, serveur d'autorisation, données, exposition, observabilité, chaîne de livraison, IaC) sont intégrées. Si une décision Candidate (D03, D05, D06, D08) bascule vers son repli à l'issue de D14, ce document est révisé pour les mesures concernées.
- **Notation à dire d'expert**, non mesurée. Elle sera recalibrée avec les constats de l'audit V1.
- **Points juridiques simplifiés**, à confirmer avec un juriste avant une mise en production réelle.
- **POC sur données fictives** (PILOTAGE §7), donc RGPD non engagé pour le POC. Les contraintes probables de l'environnement de formation (régions limitées, credentials temporaires, à vérifier selon PILOTAGE §1) sont acceptables pour le POC. Elles ne le seraient pas pour une V1 réelle (ES-26 : localisation des données ; ES-06 : secrets). Le rôle d'exécution unique imposé à tous les services est un écart connu du POC (ES-38).
- **Révision** : après chaque décision qui touche une frontière de confiance, après les expérimentations de D14, après chaque audit de fin de sprint, et avant l'audit V1.

**Historique des révisions**

| Date | Révision | Changement |
|---|---|---|
| 2026-10-09 | Initiale | Analyse de menaces du besoin, avant tout choix technique |
| 2026-10-09 | D01 | Alignement sur [D01](../decisions/D01-fonctionnalite-poc.md) : fonctionnalité du POC « A enrichie », assiette de la commission, séquestre, publicités en V2, admin modérateur, cinq bonnes pratiques |
| 2026-10-09 | Validation | Validée par Martin comme référence de sécurité du projet ([D02](../decisions/D02-decoupage-applicatif.md)) |
| 2026-10-09 | D02, option B | Périmètre, actifs (A15), acteurs et frontières alignés sur l'option B et l'architecture v0.2 ; FC11 ; 12 menaces (§4.11), dont M-CAN-01 à 04 et M-MOD-01 à 03 de l'avis sécurité de D02, reformulées ; ES-38 ; ES-18 sans seuil, décision sur une révision, filtrage des boutiques et profils ; ES-07 et ES-08 intégrées au POC ; mesures compensatoires réévaluées (§5.6) ; ES-01, ES-04, ES-11, ES-13, ES-14, ES-17, ES-20, ES-21, ES-23, ES-26, ES-29, ES-31 complétées ; complément du POC : option P2 et avis sécurité sur le complément, isolement financier démontré par la structure seulement ; 69 menaces, 38 exigences (24 Must, 8 Should, 6 Later) |
| 2026-10-09 | Cohérence architecture v0.2 | Références à l'architecture passées en v0.2 (POC option P2) ; intégrés au POC (§6.2) et retirés des recommandations (§6.3) : nouvel essai de « réserver » qui renvoie la réservation existante (ES-14, décision par défaut retenue par Martin) et route « réserver » absente du point d'entrée public, vérifiée sur l'inventaire des points d'entrée (ES-38) ; refus de modifier ou de retirer un article réservé, déjà intégré par D02, présent au seul §6.2 ; §7.2 : réservations concurrentes prouvées par le test de concurrence automatisé (M-RES-01), et non par un tir de charge de D02, campagne de sécurité facultative ; ajouts de l'option B signalés pour la prochaine relecture de Martin (Synthèse) ; décompte (69 menaces, 38 exigences) et contrôle de couverture inchangés |
| 2026-10-26 | D03 à D14 | Registre : 27 menaces des avis sécurité (M-CAN-07 et 08, M-IDP-01 à 05, M-DIS-07, M-EXP-01 à 04, M-OBS-01 à 04, M-LIV-09 à 13, M-INF-01 à 05) dont M-LIV-14, tirée de D12 et D13 ; compléments de M-MOD-03, M-LIV-03 et M-LIV-07 ; 96 menaces (5 Critiques, 32 Élevées, 56 Modérées, 3 Faibles). ES-32 passe Must (couverture de M-LIV-09 et M-LIV-11) : 25 Must, 7 Should, 6 Later. ES-10, ES-23, ES-29 et ES-31 intégrées ; ES-38 : canal authentifié par service. Nouveau §5.7 (exigence → mesures → preuve → statut) ; §6.3 et §6.4 (écarts assumés) mis à jour. Le statut « Démontrée » annoncé par D06 pour ES-10 est noté « Intégrée », « Démontrée » étant réservé aux cinq bonnes pratiques |

Date de révision : 2026-10-26.
