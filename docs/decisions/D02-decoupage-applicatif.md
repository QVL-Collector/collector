# D02 — Architecture cible V1, découpage applicatif, périmètre du POC

| Champ | Valeur |
|---|---|
| Statut | Décidée |
| Date | 2026-10-09 |
| Décideur | Martin |
| Dépend de | D01 |
| Impacte | D03, D04, D05, D06, D07, D08, D09, D10, D11, D13, [architecture-technique.md](../architecture/architecture-technique.md), [PILOTAGE.md](../PILOTAGE.md) § 3.4 (selon le sous-point), sprint 1 |
| Critères grille | C2, C10, C5, C9 |

## Contexte et besoin

La décision porte sur trois points :

1. le **découpage applicatif de la V1 cible**, c'est-à-dire le produit, pas le POC ;
2. le **périmètre du POC**, tranche de cette V1, dérivé de [D01](D01-fonctionnalite-poc.md) ;
3. un sous-point soumis à Martin : le contenu du **walking skeleton** du sprint 1.

Le schéma qui en découle est dans [architecture-technique.md](../architecture/architecture-technique.md) (vue logique, sans technologie).

### Exigences

| Source | Exigence | Conséquence pour le découpage |
|---|---|---|
| ENF-21 | « Découpage par domaines métier : ajouter une évolution (EF-48 à EF-51) ne demande pas de refonte des domaines existants. » Contexte : « l'architecture de l'application doit faciliter la mise à jour des fonctionnalités ou l'ajout rapide de nouvelles fonctionnalités (par exemple un système d'enchère, des évènements collectors avec vente d'objets en streaming direct, bot service avant-vente, ajout de services supports tel que l'analyse des ventes pour la direction) ». | Frontières par domaine, données possédées par un seul domaine, échanges par contrats explicites. |
| ENF-17 | L'outil de détection des fraudes, « interne ou acheté », se branche via un contrat documenté et versionné ; « le remplacer ne modifie aucune autre fonctionnalité ». | Le point de branchement est un composant séparé, derrière un contrat. |
| ENF-12 | « Aucun événement métier perdu (soumission d'article, variation de prix, notification) si son destinataire est momentanément indisponible. » | Canal durable entre composants ; aucun événement perdu entre l'écriture d'un état et sa transmission. |
| ENF-18, ENF-19 | Paiement et envoi d'emails isolés derrière une interface propre ; API documentées et versionnées. | Interfaces vers les tiers, substituables en test ; contrats versionnés. |
| ENF-10, ENF-11, ENF-15, ENF-16 | Disponibilité de 99,5 % ; pas de point unique de défaillance sur la consultation et l'achat ; capacité ajoutée automatiquement au pic ; retour au minimum hors pic. | Composants répliqués, mise à l'échelle selon le profil de charge de chacun. |
| ES-02, ES-03 | Authentification déléguée à un serveur d'autorisation dédié ; « aucun service métier ne stocke ni ne vérifie de mot de passe » ; rôles fournis par ce seul serveur. | Serveur d'autorisation hors des domaines métier. |
| ES-07, ES-14, ES-15, ES-17, ES-20, ES-31 | Entrées prévues pour D02 ([exigences-securite.md](../securite/exigences-securite.md) § 5.5) : exposition minimale ; paiement vérifié et séquestre ; photos isolées et privées jusqu'à la publication ; back-office séparé de l'espace public ; contrôle fermé par défaut. | Zones de sécurité et frontières de confiance dans le schéma. |
| Contexte | « L'application incluant des transactions financières, la sécurité est une exigence de premier plan. » | Le rayon d'impact d'une compromission est un critère de choix. |
| D01 (décidée) | POC « A enrichie » : deux composants (annonces, catalogue compris ; contrôle de publication) reliés par un canal asynchrone point à point, contrat versionné du contrôle, revalidation après modification, contrôle fermé par défaut, upload sécurisé. | Le POC doit être une tranche représentative de la V1 retenue, et non un prototype à part. |
| Grille, C2 niveau 5 | « L'implémentation technique du POC valide la faisabilité des points clés de l'architecture de la solution applicative à développer (découpage applicatif, communication interservice, sécurité, hébergement, orchestration, etc.), elle est documentée. » | Tout point clé de la cible que le POC n'exerce pas affaiblit C2. |

### Contraintes

- **Équipe du contexte** : un lead developer et deux développeurs confirmés de 5 ans d'expérience ; « de nouvelles embauches sont envisagées ». C'est la base de la cartographie des compétences (C9).
- **Réalité de l'exercice** : Martin réalise seul le POC. Sprint 1 = walking skeleton ; sprint 2 = fonctionnalité métier, charge et autoscaling ([PILOTAGE.md](../PILOTAGE.md), jalons J4 et J5). Le périmètre de l'option P2, retenu dans le complément, demande un sprint de plus : découpage indicatif dans « Conséquences ».
- **Hébergement** : AWS Academy Learner Lab, crédits inférieurs à 100 $, environnement allumé puis éteint. Contraintes probables, à vérifier : rôle d'accès unique imposé, sessions limitées ([PILOTAGE.md](../PILOTAGE.md) § 1).
- **Aucune technologie choisie** : D02 fixe le découpage logique. Les solutions viennent de D03 à D11.

### Domaines métier de la V1

Issus de l'index rôle × domaine d'[exigences.md](../00-cadrage/exigences.md) § 2.13.
La colonne « Service » applique la décision de Martin (option B). Pour la réservation, elle remplace le texte de l'option B, conservé comme historique : voir « Arbitrages de cohérence ».

| Domaine | Service (option B) | Exigences | Données centrales |
|---|---|---|---|
| Annonces et catalogue | Place de marché | EF-01 à EF-05, EF-11 à EF-14 | Catégories, boutiques, articles, révisions, photos, décisions de contrôle, historique des prix, **disponibilité de l'article** (disponible, réservé jusqu'à une échéance, vendu) |
| Comptes et profils | Place de marché | EF-06 à EF-10, EF-20 (suspension) | Profil, centres d'intérêt, statut vendeur. L'identité et les mots de passe restent au serveur d'autorisation (ES-02). |
| Notation | Place de marché | EF-46, EF-47 | Notes |
| Recommandations | Place de marché | EF-39, EF-40 (V2) | Calculées à partir des intérêts et du catalogue |
| Alertes de fraude | Place de marché | EF-43 | Alertes reçues, décisions de l'admin |
| Commandes et paiement | Transactions | EF-22, EF-24 à EF-28 ; EF-23 avec la Place de marché | Commandes, montants, commission, état du séquestre, reversements. Cartes et fonds restent chez le prestataire (ES-12, ES-15). |
| Notifications | Interactions | EF-15, EF-29 à EF-34 | Suivis, alertes, préférences, notifications, copie des centres d'intérêt |
| Messagerie | Interactions | EF-35 à EF-38 | Conversations, signalements |
| Contrôle de publication | Contrôle de publication | EF-16, EF-17, EF-36 (annonces), EF-42 (partiel) | Aucune donnée propre : règles et plages de prix en configuration (D01) |
| Relais vers l'outil de fraude | Contrôle de publication (adaptateur, V1) | EF-41, EF-42 | Aucune donnée propre |
| Publicités partenaires (V2) | À créer en V2 | EF-44, EF-45 | Consentements, flux |

Le back-office n'est pas un domaine. C'est un front distinct (ES-20) ; chaque fonction d'administration est portée par le domaine propriétaire de la donnée. Les décisions de l'admin sur les articles « à vérifier » (EF-19) relèvent donc des annonces, pas du contrôle, qui reste sans état et remplaçable.

#### Arbitrages de cohérence (option B)

1. **Disponibilité et réservation de l'article.**
   - *Écart* : l'option B écrit « Transactions possède la réservation » ; l'avis faisabilité recommande de laisser la disponibilité aux annonces.
   - *Arbitrage* : la **disponibilité de l'article appartient à la Place de marché**. Transactions la réserve par l'appel synchrone authentifié déjà prévu par B (ES-13), pour une durée bornée (échéance). La Place de marché réserve de façon atomique dans ses propres données et renvoie le prix de référence ; Transactions crée alors la commande. Aucune transaction commune aux deux services.
   - *Sorties* : paiement confirmé → Transactions émet « commande payée » → la Place de marché passe l'article « vendu ». Échéance dépassée sans confirmation → la réservation tombe d'elle-même, sans message de compensation.
   - *Justification* :
     - une seule source de vérité pour « un objet unique n'est vendu qu'une fois » (EF-23, M-PAY-05) ;
     - le catalogue affiche « réservé » sans délai. Seul « vendu » arrive avec retard, alors que l'article n'est déjà plus achetable ;
     - la Place de marché sait que l'article est réservé : elle refuse au vendeur de le modifier ou de le retirer pendant la réservation. Avec une réservation tenue par Transactions, elle l'ignorerait ;
     - le prix est figé pendant la réservation : le montant payé est celui que la Place de marché a renvoyé (ES-13).
   - *Cas limite* : une confirmation de paiement qui arrive après l'échéance (prestataire en retard, ou « commande payée » livrée en retard à la Place de marché), alors qu'un autre acheteur a réservé l'article, produit une double vente (M-RES-01) et donne lieu à un remboursement tracé. L'échéance est donc fixée au-delà de la durée de la session de paiement et du délai de livraison des événements, à préciser avec le prestataire retenu (ES-14).
2. **Outil de fraude.**
   - Un outil externe ne se connecte jamais au canal interne. Un **adaptateur**, porté par le service Contrôle de publication, relaie les échanges : contrat du contrôle ; événements prix modifié, nouveau vendeur et transaction ; verdicts et alertes en retour.
   - Le Contrôle n'expose aucune API publique et ne décode aucun fichier : c'est le service le moins exposé pour détenir les identifiants de l'outil.
   - Les alertes sont conservées par la Place de marché, où l'admin les traite (EF-43). Transactions les reçoit aussi, pour suspendre un reversement (ES-15).
   - Réception des alertes de l'outil : mode fixé en D05, de préférence sans nouveau point d'entrée public.
3. **Comptes minimal.** Dans B, Comptes et profils est un module de la Place de marché. Dans le POC, le « module Comptes minimal » devient donc un module interne de la Place de marché : statut vendeur fictif, en lecture seule, consulté par le module Annonces à la soumission, sans API ni jointure (avis faisabilité § 3). Il ne prouve plus le découpage retenu, qui se joue entre services ; il garde la donnée à sa place, pour moins d'une demi-journée.

### Critères de choix

Fixés avant la comparaison des options.

| Critère | Ce qui est regardé |
|---|---|
| Évolutivité (ENF-21) | Coût d'ajout d'une évolution ; risque de refonte |
| Remplaçabilité et intégrations (ENF-17, ENF-18) | Branchement de l'outil de fraude et des tiers sans toucher au reste |
| Fiabilité et disponibilité (ENF-10 à ENF-12) | Événements sans perte ; pas de point unique de défaillance |
| Sécurité (« premier plan ») | Surface exposée, flux internes à authentifier, rayon d'impact d'une compromission |
| Équipe (C9) | Charge et compétences pour trois développeurs |
| Coût d'exploitation (ENF-16) | Nombre de déployables, d'instances au repos, de bases, de pipelines |
| Représentativité du POC (C2) et démonstrations (C5) | Part des points clés de la cible réellement exercés par le POC |

## Options

### Option A — Monolithe modulaire, avec le contrôle de publication extrait

- **Description** :
  - *Composants* :
    - une **plateforme** : un seul déployable, répliqué, découpé en modules par domaine (Annonces et catalogue, Comptes et profils, Commandes et paiement, Notifications, Messagerie, Notation, Recommandations, Fraude) ;
    - le **contrôle de publication**, extrait en composant séparé, sans état, derrière un contrat versionné. C'est le point de branchement de l'outil de fraude (ENF-17) ;
    - un **canal asynchrone durable**, point à point ;
    - un socle commun : serveur d'autorisation (ES-02), point d'entrée HTTPS, base de données, stockage de fichiers, front public et front back-office.
  - *Propriété des données* : une base, avec un espace de données par module. Un module est seul à lire et écrire ses données ; les autres passent par son interface. Le contrôle ne possède aucune donnée.
  - *Communication* :
    - front → plateforme : synchrone, par des API versionnées (ENF-19) ;
    - entre modules : appels internes par une interface explicite ; les événements métier sont écrits dans la même transaction que le changement d'état, puis traités, pour n'en perdre aucun (ENF-12, mécanisme fixé en D05) ;
    - plateforme ↔ contrôle : asynchrone, point à point, par le canal (demande de contrôle, résultat) ;
    - plateforme ↔ outil de fraude : contrat du contrôle, plus les événements variation de prix, nouveau vendeur et transaction (ES-21) ;
    - plateforme → prestataire de paiement et envoi d'emails : derrière une interface interne (ENF-18).
  - *Règle d'extraction* : un module ne devient un composant séparé que si au moins un critère est constaté et nommé.
    - **E1, remplaçabilité** : le module doit pouvoir être remplacé par un outil tiers ou acheté.
    - **E2, profil d'exécution** : charge, latence ou disponibilité mesurablement différentes du reste, qui obligeraient à dimensionner toute la plateforme pour ce module.
    - **E3, frontière de sécurité** : secrets ou données dont l'exposition doit être isolée du reste.
    - **E4, autonomie d'équipe** : une équipe dédiée qui livre à son propre rythme.
  - *Application à la V1* : seul le contrôle remplit un critère (E1, ENF-17). Candidats suivis : Commandes et paiement (E3, selon l'avis sécurité), Messagerie (E2, si le temps réel devient une exigence), Enchères à leur arrivée (E2).
  - *Évolutions (ENF-21)* : enchères, vente en direct, bot avant-vente et analyse des ventes se branchent comme nouveaux modules ou composants, sur les interfaces et les événements existants. Détail : [architecture-technique.md](../architecture/architecture-technique.md) § 7.
- **Avantages** :
  - La plus simple qui couvre les exigences : deux déployables applicatifs, un canal, une base.
  - Aucune transaction répartie : la réservation d'un objet unique et la création de la commande tiennent dans une transaction locale (EF-23, ES-14).
  - Aucune diffusion d'événements à plusieurs consommateurs en V1 : la variation de prix part vers les notifications en interne, et vers l'outil de fraude par le canal.
  - Surface interne minimale : un seul échange interservice à authentifier entre composants de la plateforme (plateforme ↔ canal ↔ contrôle), plus l'outil de fraude.
  - Extraction future peu coûteuse : les modules possèdent déjà leurs données et communiquent par interface ou par événement.
  - Adaptée à trois développeurs : un dépôt, deux pipelines, une conception partagée.
- **Inconvénients** :
  - **Rayon d'impact** : une faille dans un module (traitement des photos, messagerie) expose le processus entier, y compris les secrets du prestataire de paiement. À compenser par des mesures (droits restreints de la clé du prestataire, fonds en séquestre chez le prestataire), ou par l'extraction du module Commandes et paiement (critère E3).
  - Mise à l'échelle globale : un pic de consultation réplique toute la plateforme, pas seulement le catalogue.
  - Les frontières entre modules ne tiennent que si elles sont vérifiées : règle de dépendance contrôlée en CI et revue d'architecture de fin de sprint. Sans cela, la plateforme dérive vers un bloc indistinct.
  - Un déploiement de la plateforme livre tous les modules à la fois.
- **Coût AWS et effort** :
  - V1 cible : 2 déployables applicatifs. Au repos, environ 3 instances : 2 pour la plateforme, qui porte la consultation et l'achat (ENF-11), et 1 pour le contrôle, dont l'arrêt ne fait que retarder les publications (ES-31). S'y ajoutent une base, un stockage, un canal, un serveur d'autorisation, un point d'entrée, et 2 pipelines. Estimation de principe ; chiffrage en D03 et D11.
  - POC : 2 composants applicatifs, conformes à D01. Effort conforme à D01 (sprints 1 et 2).
- **Complexité ajoutée** : un composant extrait, un canal, un contrat versionné. Chacun est justifié par ENF-17 et ENF-12, et déjà décidé dans D01.
- **Couverture grille** :
  - **C2** : découpage par domaine (modules) et extraction justifiée par une exigence (contrôle) ; communication interservice asynchrone par contrat ; sécurité par zones ; hébergement et orchestration de deux profils de charge. Tous ces points sont exercés par le POC.
  - **C5** : deux signaux de mise à l'échelle distincts (requêtes pour la plateforme, messages en attente pour le contrôle) ; perte d'une instance de la plateforme pendant le tir de charge.
  - **C9** : compétences réalistes pour des développeurs confirmés : conception modulaire par domaine, messagerie asynchrone fiable (idempotence, rejeu), sécurité applicative. Action de formation candidate : conception par domaines et échanges asynchrones fiables.
- **Tranche POC** : la plateforme avec le module Annonces et catalogue (et un module Comptes minimal, voir « Périmètre du POC »), le contrôle extrait, le canal et le socle. Le POC exerce le motif central de A : module de la plateforme, composant extrait, contrat versionné, canal durable, contrôle fermé par défaut. Il n'exerce pas les échanges entre modules métier riches ni les intégrations tierces, qui restent documentés.

### Option B — Services à gros grain par grand domaine

- **Description** :
  - *Composants* : quatre services, chacun avec ses données, et le même socle que A :
    - **Place de marché** : annonces et catalogue, boutiques, comptes et profils, notation, recommandations ;
    - **Transactions** : commandes, réservation, paiement, commission, séquestre ;
    - **Interactions** : notifications, messagerie et sa modération ;
    - **Contrôle de publication** : comme dans A.
  - *Propriété des données* : une base, ou un espace isolé avec son propre compte d'accès, par service.
  - *Communication* :
    - front → services : synchrone, par le point d'entrée ;
    - entre services : asynchrone par le canal, avec **diffusion** : la variation de prix part vers Interactions et vers l'outil de fraude ; la commande payée vers Interactions, la notation et l'outil de fraude ;
    - synchrone interne : Transactions lit le prix de référence et l'état de l'article auprès de la Place de marché (ES-13), avec une authentification de service à service ;
    - Transactions possède la réservation ; le catalogue affiche « réservé » ou « vendu » avec une cohérence différée.
- **Avantages** :
  - Périmètre financier isolé : secrets du prestataire, point de réception de ses notifications et données de commande sont dans un service à part. Rayon d'impact réduit, revue renforcée possible.
  - Profil temps réel de la messagerie isolé.
  - Déploiements indépendants ; environ un service par développeur.
- **Inconvénients** :
  - Diffusion d'événements, cohérence différée et authentification de service à service dès la V1.
  - Le diagnostic d'un parcours qui traverse plusieurs services pousse vers les traces distribuées (ENF-25).
  - Au moins quatre contrats internes, chacun avec ses tests de contrat.
  - Regroupements discutables : la notation dépend des transactions, les notifications des intérêts tenus par la Place de marché. Des allers-retours entre services apparaissent.
- **Coût AWS et effort** :
  - V1 cible : 4 déployables. Au repos, environ 6 instances (2 Place de marché, 2 Transactions, 1 Interactions, 1 Contrôle), 3 à 4 espaces de données, 4 pipelines, un canal qui diffuse.
  - POC : identique à A, la tranche étant la même. Même coût dans le Learner Lab.
- **Complexité ajoutée** : diffusion, cohérence différée, appels synchrones authentifiés entre services, suivi distribué. Le principal gain, l'isolement du périmètre financier (E3), peut s'obtenir seul (voir Recommandation).
- **Couverture grille** :
  - **C2** : découpage plus visible. En revanche, les points clés propres à B (appel synchrone entre services, diffusion, cohérence différée, authentification de service à service) ne sont pas exercés par le POC.
  - **C5** : identique à A pour le POC.
  - **C9** : toute l'équipe a besoin de compétences en systèmes répartis ; formation plus lourde.
- **Tranche POC** : la partie annonces de la Place de marché et le contrôle. Représentative du motif « service, contrat, canal », pas des échanges entre les trois grands services.

### Option C — Services par domaine fin

- **Description** :
  - *Composants* : un service par domaine, soit une dizaine : identité et profils, boutiques et annonces, catalogue (lecture et recherche, alimenté par événements), contrôle, commandes et paiement, notifications, messagerie, notation, recommandations, fraude. S'y ajoutent une passerelle d'API et le socle commun.
  - *Propriété des données* : une base par service ; copies de lecture alimentées par événements (catalogue, recommandations).
  - *Communication* : majoritairement par événements, avec diffusion ; enchaînement réservation → paiement → confirmation ou compensation entre services (saga) ; appels synchrones via la passerelle.
- **Avantages** :
  - Indépendance maximale de déploiement et de mise à l'échelle par domaine.
  - Rayon d'impact minimal par service.
  - Chaque évolution est un nouveau service.
- **Inconvénients** :
  - Disproportionné pour trois développeurs : plus de trois services chacun, avec leurs pipelines, leurs bases et leur exploitation.
  - Compensation entre services pour l'achat d'un objet unique : la complexité que D01 a relevée pour l'achat (« annulation compensatoire »).
  - Séparer le catalogue des annonces recopie la même donnée sans besoin, ce que D01 a jugé « forcé ».
  - Surface interne maximale : chaque flux est à authentifier et à observer.
- **Coût AWS et effort** :
  - V1 cible : une dizaine de déployables plus la passerelle. Au repos, environ 15 instances (2 sur les chemins de consultation et d'achat, 1 ailleurs) ; une base par service ; une dizaine de pipelines.
  - POC : pour être représentatif, il devrait inclure la passerelle et la séparation catalogue / annonces, soit au moins quatre composants. Cela dépasse le sprint 2 et l'enveloppe de D01.
- **Complexité ajoutée** : la plus élevée (sagas, copies de lecture, passerelle, traces distribuées), sans exigence présente qui la justifie.
- **Couverture grille** :
  - **C2** : découpage spectaculaire, mais le POC n'en exerce qu'une faible part, ou bien il déborde du planning.
  - **C5** : plus de points de mise à l'échelle, coût plus élevé.
  - **C9** : demande des profils plateforme et fiabilité à recruter ; risque de profils irréalistes (« moutons à 5 pattes ») dans une équipe de trois.
- **Tranche POC** : faible, sauf à élargir le POC.

### Option écartée d'emblée — Monolithe sans composant extrait

Le contrôle resterait un module interne de la plateforme. C'est la solution la plus simple, mais elle contredit D01 (deux composants et un canal) et ne valide pas la communication interservice citée par C2.
Elle sert de référence de simplicité : chaque élément de A au-delà d'elle est justifié par ENF-17 et ENF-12.

### Synthèse

| Critère | A — Modulaire, contrôle extrait | B — Gros grain | C — Domaines fins |
|---|---|---|---|
| Évolutivité (ENF-21) | Bonne : modules et règle d'extraction | Bonne | Maximale, au coût le plus élevé |
| Fraude remplaçable (ENF-17) | Oui | Oui | Oui |
| Fiabilité (ENF-12) | Canal durable, événements écrits avec l'état | Idem, plus la diffusion | Idem, plus la compensation |
| Sécurité | Surface interne minimale ; rayon d'impact du déployable unique | Périmètre financier isolé ; plus de flux internes | Rayon d'impact minimal ; surface interne maximale |
| Équipe de trois (C9) | Adaptée | Tendue | Inadaptée sans recrutement |
| Instances au repos (estimation, hors socle) | ≈ 3 | ≈ 6 | ≈ 15 |
| Pipelines | 2 | 4 | ≈ 10 |
| POC représentatif (C2) | Forte | Partielle | Faible, ou POC élargi |
| Démonstrations C5 | Oui | Oui (POC identique à A) | Oui, plus coûteuses |

## Recommandation

**Recommandation initiale (option A) non retenue : voir Décision de Martin.**

**Option A — monolithe modulaire, avec le contrôle de publication extrait.**

1. **C'est la plus simple qui satisfait le besoin** : ENF-21 par les modules et la règle d'extraction ; ENF-17 par le contrôle extrait et son contrat ; ENF-12 par le canal durable et les événements écrits avec l'état. Aucune exigence présente ne demande davantage.
2. **Elle est proportionnée à l'équipe** : trois développeurs, deux déployables applicatifs, ni transaction répartie ni diffusion en V1 (C9).
3. **Le POC en est une tranche fidèle (C2)** : il exerce le motif central de la cible (module, composant extrait, contrat, canal, contrôle fermé par défaut), et rien n'y est jetable. B et C auraient des points clés que le POC ne touche pas.
4. **Sa croissance est écrite** : la règle d'extraction dit quand et pourquoi un module deviendra un service. C'est la réponse à « et si le marché change ? » (ENF-21).

**Variante à soumettre à l'avis sécurité** : si le rayon d'impact du déployable unique est jugé inacceptable pour le périmètre financier, extraire le **seul module Commandes et paiement** (critère E3) plutôt que passer à B. Cette variante isole les secrets du prestataire et le point de réception de ses notifications, sans découper le reste. Elle est hors POC : elle ne change ni le POC ni son coût.

**Points soumis aux avis** :
- Sécurité :
  - rayon d'impact du déployable unique, et variante ci-dessus ;
  - ES-18 ne cite ni le titre ni le nom de boutique parmi les modifications qui renvoient au contrôle, alors qu'ES-16 y recherche les coordonnées : à confirmer ;
  - moindre privilège entre composants (canal, stockage, secrets) si l'environnement de formation impose un rôle d'accès unique ([PILOTAGE.md](../PILOTAGE.md) § 1).
- Faisabilité :
  - contenu du walking skeleton (sous-point ci-dessous) ;
  - module Comptes minimal dans le POC.

## Périmètre du POC

Dérivé de D01 (« A enrichie »), de la décision de Martin (option B, WS-2) et de sa décision sur le complément : **option P2, les quatre services de l'option B**.
- La **fonctionnalité métier démontrée** reste la mise en vente contrôlée de D01 (C3) : soumission, contrôle, publication, consultation.
- **Interactions minimal** reçoit la « décision de publication » par diffusion et notifie le vendeur : diffusion et cohérence différée (K1, K4).
- La **réservation** est la première étape de l'achat de la V1, construite avec le code de la V1. Elle exerce dans le POC l'appel synchrone authentifié Transactions → Place de marché et la structure du périmètre financier (K2, K3, K5). Le paiement et la suite de l'achat restent documentés.

Schéma : [architecture-technique.md](../architecture/architecture-technique.md) § 3. Séquence de la réservation : § 4.3.

### Construit et déployé

| Élément | Contenu dans le POC | Exigences |
|---|---|---|
| Front web | Catalogue public (filtre par catégorie, fiche article, mention « réservé » sur un article réservé) ; espace vendeur (créer, modifier et soumettre un article avec photos, voir son état et le motif, lister ses notifications) ; espace acheteur (réserver un article publié, voir sa réservation, son montant et son échéance) ; écran admin « à vérifier » (publier, ou refuser avec motif). Textes en français et en anglais ; accessibilité contrôlée en CI. Politique de sécurité du contenu et neutralisation des sorties, y compris sur l'écran admin. | EF-01, EF-02, EF-04, EF-12, EF-18, EF-19, EF-23 (partielle), EF-33 et EF-34 (partielles) ; ENF-26 et ENF-30 en version minimale ; ES-07, ES-08 |
| Service Place de marché, module Annonces et catalogue | Articles et leurs états (brouillon, en contrôle, publié, refusé, à vérifier) ; numéro de révision ; **toute modification d'un article publié, en contrôle ou à vérifier le renvoie au contrôle, sans seuil** ; décision de l'admin portant la révision examinée, refusée pour conflit (409) si l'article a changé entre-temps ; dépôt des photos ; API publique de lecture, API vendeur, API admin ; journal des décisions ; contenu encodé à l'affichage. Émission de « décision de publication », diffusée une fois la décision appliquée. **Disponibilité de l'article** (disponible, réservé jusqu'à une échéance) : appel synchrone « réserver » ouvert au seul service Transactions, absent du point d'entrée public ; réservation atomique ; échéance fixée et plafonnée par la Place de marché ; une seule réservation active par acheteur ; un nouvel essai de « réserver » pour le même acheteur et le même article renvoie la réservation existante, avec la même échéance, au lieu d'en créer une seconde ou d'être refusé ; prix de référence et frais de port renvoyés ; modification et retrait refusés au vendeur pendant la réservation. | EF-12, EF-16, EF-18, EF-19, EF-23 (partielle), ES-03, ES-04, ES-08, ES-13, ES-14 (réservation seulement), ES-17, ES-18, ES-31, ES-38 ; M-SYN-01, M-RES-01, M-RES-02 |
| Service Place de marché, module Comptes minimal | Statut des vendeurs (ancienneté, suspension), fictif, en lecture seule ; interface interne appelée par Annonces à la soumission ; ni API, ni jointure, ni événement. | EF-17 (règle « statut du vendeur ») |
| Service Contrôle de publication | Sprint 1 (WS-2) : une seule règle, la complétude du titre, de la description, de la catégorie et du prix. Sprint 2 : le critère photo s'ajoute à la complétude avec une nouvelle version des règles, sans changer le contrat v1 ; puis coordonnées personnelles, plage de prix de la catégorie, statut du vendeur. Une catégorie inconnue du contrôle donne « à vérifier », jamais « publié ». Sans état, idempotent, contrat v1 vérifié par un test de contrat. Pas d'adaptateur de fraude dans le POC. | EF-17, EF-36 (annonces), EF-42 (partiel), ENF-17, ES-16 |
| Service Interactions minimal | Une seule fonction : recevoir « décision de publication » et créer la notification du vendeur, une seule fois par décision, listée dans son espace. Texte construit à partir de la catégorie de motif, en français et en anglais, sans contenu saisi par un utilisateur. Contrat validé à la réception ; un message invalide est écarté après un nombre borné d'essais. Liste filtrée par l'identité du jeton. Ni email, ni préférences, ni suivis, ni messagerie. | EF-33, EF-34 (partielles), ENF-12, ES-04 ; M-NOT-02, M-CAN-01, M-CAN-03 |
| Service Transactions, réduit à la réservation | Un acheteur authentifié réserve un article publié. Transactions obtient un jeton de service (audience « Place de marché », droit limité à « réserver », durée courte), appelle la Place de marché en propageant le jeton de l'acheteur, puis enregistre une commande « en attente de paiement », une seule par réservation (un nouvel essai n'en crée pas de seconde) : article, acheteur, échéance, prix de référence, frais de port, commission, montants calculés côté serveur au centime. Passée l'échéance, la commande est présentée « expirée ». Ni prestataire, ni paiement, ni événement de commande. | EF-23 (partielle), EF-24, ENF-01, ES-03, ES-04, ES-13, ES-38 ; M-MOD-02, M-PAY-05, M-RES-01, M-SYN-01 |
| Canal asynchrone | Durable. Point à point : demandes vers le Contrôle, résultats vers la Place de marché. Diffusion : « décision de publication » vers Interactions et vers un abonnement de test, qui prouve un second abonné. **Une identité par service présent sur le canal** (Place de marché, Contrôle, Interactions), avec des droits limités : seule la Place de marché écrit les demandes et publie la décision, seul le Contrôle écrit les résultats, Interactions ne fait que lire (M-CAN-01, M-CAN-05, avis sécurité § 3). Aucun service ne peut créer d'abonnement (M-CAN-06). Transactions n'utilise pas le canal dans le POC. | ENF-12, ES-31, ES-38 |
| Base de données | Un espace et un compte d'accès par service : Place de marché, Interactions, Transactions ; le Contrôle n'en a pas. Un accès direct d'un service aux données d'un autre est refusé par la base. | ES-07 ; M-MOD-01 (avis sécurité § 3) |
| Stockage de fichiers | Photos ré-encodées, privées jusqu'à la publication de l'article. | ES-17 |
| Serveur d'autorisation | Connexion, jetons, rôles acheteur, vendeur et admin ; comptes fictifs. Identité du service Transactions et jetons de service à audience et droit limités. | ES-02, ES-03 |
| Point d'entrée public | HTTPS seul, routage vers la Place de marché, Interactions et Transactions, en-têtes de sécurité ; limitation de débit recommandée. | ES-01, ES-07 ; ES-29 recommandée |
| Socle d'exploitation | Au moins une composante d'observabilité (D09) ; chaîne CI/CD des quatre services avec scans bloquants et détection de secrets (D10) ; allumage et extinction depuis zéro (D11). | ENF-22, ENF-25, ENF-28, ES-05, ES-06 |

Points de cohérence intégrés au POC :
- **Revalidation sans seuil**, retenue par défaut : toute modification renvoie l'article au contrôle (avis faisabilité § 2). La question du titre (écart ES-18 / ES-16) est réglée, et l'extension « prix sous le seuil » de D01 devient sans objet. ES-18 est à réviser dans [exigences-securite.md](../securite/exigences-securite.md).
- **Décision de l'admin sur une révision** : sans cela, l'admin pourrait publier une version qu'il n'a pas vue (M-ANN-05).
- **ES-07 et ES-08 intégrées**, et non plus recommandées : l'écran « à vérifier » partage l'origine du contenu public, et ces deux mesures sont la seule barrière contre un script injecté dans une annonce et exécuté dans la session de l'admin (M-ANN-03, puis M-BO-01). Preuve : un titre contenant du balisage s'affiche comme du texte, au catalogue et chez l'admin.
- **Règle unique du sprint 1** : le contrat v1 porte dès le sprint 1 le nombre de photos et la version des règles. Le passage aux règles v2 au sprint 2 démontre l'usage de ce champ sans nouvelle version du contrat.
- **Parades de l'option P2, issues de l'avis sécurité sur le complément**, dans les AC (ES-14, ES-38) : jeton de service à audience et droit limités ; jeton de l'acheteur revérifié par la Place de marché, qui lit le prix dans ses propres données (ES-13, M-MOD-02) ; réservation atomique, prouvée par un test de concurrence (M-PAY-05, M-RES-01) ; une seule réservation active par acheteur ; échéance plafonnée. Les deux dernières traitent aussi la réservation abusive relevée par D01 (M-RES-02).
- **Nouvel essai de la réservation**, retenu par défaut par Martin : un nouvel essai de l'appel « réserver » pour le même acheteur et le même article (délai dépassé, double clic) renvoie la réservation existante, avec la même échéance, au lieu d'en créer une seconde ou d'être refusé ; Transactions ne crée qu'une commande par réservation (idempotence, ES-14, M-RES-01). Sans cette règle, la limite d'une réservation active par acheteur refuserait le nouvel essai alors que la réservation a bien été faite.
- **Mesures 1 à 3 du point 4 de l'avis sécurité sur le complément**, dans les AC : seule la Place de marché peut publier « décision de publication » ; Interactions valide le contrat à la réception ; le cloisonnement des données est vérifié par la base.
- **Réservation et cycle de publication** : seul un article publié et disponible se réserve. Pendant la réservation, le vendeur ne peut ni modifier ni retirer l'article : sinon la revalidation le retirerait du catalogue et le prix réservé pourrait changer (ES-13).

### Fourni par des données fictives ou par configuration

- Catégories et plages de prix par catégorie, issues d'une seule source de données fictives (la gestion des catégories, EF-05, est hors POC).
- Comptes vendeur, acheteur et admin dans le serveur d'autorisation ; boutiques (EF-11 hors POC) ; statut des vendeurs.
- Photos de test, dont une portant des coordonnées GPS fictives, pour prouver leur suppression (ES-17).
- Le tout chargé à chaque allumage par un script rejouable sans effet de bord (D01, « Avis faisabilité », risque de démo n° 4).
- Échéance de réservation courte (quelques minutes), en configuration de la Place de marché, pour montrer son expiration pendant la démonstration. En V1, elle est fixée au-delà de la durée de la session de paiement (« Arbitrages de cohérence »).

### Simulé ou substitué

- **Rien n'est simulé dans le chemin critique** de la mise en vente : aucun tiers n'intervient (D01).
- **Paiement** : ni construit ni simulé. Une réservation n'est jamais payée dans le POC : elle tombe à l'échéance, comme en V1 quand le paiement n'aboutit pas. Aucun faux prestataire, aucune notification de paiement simulée (D01 : un simulateur affaiblit la preuve et pousse à accepter des notifications non vérifiées).
- **Outil de fraude** : ni construit ni simulé, adaptateur compris. Son branchement est prouvé par le contrat versionné du contrôle et son test de contrat (D01).
- **Second abonné de la diffusion** : en V1, l'adaptateur de fraude. Dans le POC, un abonnement de test, posé par l'exploitation pour le test d'intégration (aucun service ne crée d'abonnement, ES-38), lu avec une identité en lecture seule puis retiré, reçoit le même événement, y compris quand Interactions est arrêté.

### Hors POC, documenté

- Services et domaines : dans Transactions, le paiement, la confirmation du prestataire, le passage à « vendu », le séquestre, les reversements et les événements de commande ; dans Interactions, les emails, les préférences, les suivis et la messagerie ; notation ; recommandations ; alertes de fraude et adaptateur ; gestion des catégories, des boutiques et des comptes ; inscription en libre-service.
- Sécurité : back-office sur une origine distincte (ES-20) ; second facteur admin (ES-10), sauf s'il s'obtient par simple paramétrage ; paiement délégué (ES-12), confirmation de paiement vérifiée et idempotente (ES-14, hors réservation atomique), séquestre (ES-15) ; exigences listées dans [exigences-securite.md](../securite/exigences-securite.md) § 6.4.
- Publicités partenaires (V2) ; évolutions EF-48 à EF-51.

**Écarts assumés entre le POC et la V1**, documentés et repris dans l'audit V1 :
- l'écran « à vérifier » est servi par le front web, protégé par le rôle admin, ES-07 et ES-08 ; en V1, le back-office a sa propre origine (ES-20) ;
- tous les composants partagent l'identité d'exécution imposée par l'environnement de formation ; la V1 exige une identité d'exécution par service (avis sécurité § 5). Conséquence : l'isolement du périmètre financier n'est démontré que dans sa structure (service, données, compte d'accès et jeton de service propres à Transactions), sans prestataire ni secret financier (avis sécurité sur le complément) ;
- l'échéance de réservation est courte pour la démonstration ;
- Transactions et Interactions tournent à une instance ; la V1 prévoit au moins deux instances pour Transactions (ENF-11).

## Sous-point soumis à Martin — contenu du walking skeleton (sprint 1)

[PILOTAGE.md](../PILOTAGE.md) § 3.4 : « Le sprint 1 est un walking skeleton : un service minimal déployé de bout en bout sur l'environnement managé (CI, HTTPS, authentification, allumage et extinction depuis zéro), avant toute fonctionnalité métier. »
D01 (« Avis faisabilité ») a relevé l'alternative et l'a renvoyée à l'arbitrage de Martin. Les deux variantes portent une tranche minimale de la fonctionnalité, réutilisée telle quelle au sprint 2 : « avant toute fonctionnalité métier » s'entend donc ici comme « avant les règles et les écrans du sprint 2 ».

### WS-1 — Un seul composant (PILOTAGE tel qu'écrit)

- **Contenu** : la Place de marché avec le module Annonces : lecture publique du catalogue (articles publiés fictifs) et création d'un article par un vendeur authentifié. CI, HTTPS, authentification, allumage et extinction depuis zéro. Le contrôle et le canal arrivent au sprint 2.
- **Avantages** :
  - Conforme à PILOTAGE.
  - Sprint 1 plus léger, concentré sur les risques de plateforme : environnement managé, TLS, authentification, IaC.
- **Inconvénients** :
  - Le risque d'intégration le plus élevé (canal sur l'environnement managé, deux composants, contrat) glisse au sprint 2, déjà le plus chargé du projet (D01).
  - Un article créé ne peut pas être publié sans contrôle (ES-31). Le parcours de bout en bout n'est donc pas traversé avant le sprint 2, sauf à ajouter une publication directe temporaire, qui contredirait le contrôle fermé par défaut.
  - La communication interservice (C2) n'est prouvée qu'en fin de projet.

### WS-2 — Les deux composants et le canal, avec une seule règle

- **Contenu** : WS-1, plus le contrôle réduit à une règle (complétude), le canal, et le contrat v1 avec son test de contrat. Un article soumis passe « en contrôle », puis « publié » ou « refusé », et apparaît au catalogue s'il est publié. Pas de photos (sprint 2).
- **Avantages** :
  - Lève au sprint 1 le risque d'intégration le plus élevé.
  - Le parcours soumission → contrôle → publication → consultation existe dès J4 : preuve précoce pour C2, et base de démonstration de secours.
  - Le sprint 2 n'ajoute plus d'infrastructure : règles, photos, écran admin, écrans, charge et autoscaling.
  - Aucun raccourci temporaire : le contrôle fermé par défaut s'applique dès le premier jour.
- **Inconvénients** :
  - Écart à PILOTAGE § 3.4 (« un service minimal ») : la règle serait à amender par Martin.
  - Sprint 1 plus chargé : deux déployables, le canal dans l'IaC, deux pipelines.
  - Dépend de l'expérimentation du canal (J3) : si elle n'est pas concluante sur l'environnement managé, WS-2 est impossible au sprint 1.

### Recommandation sur le walking skeleton

**WS-2, à condition que l'expérimentation du canal soit retenue (D14) et concluante sur l'environnement managé avant le début du sprint 1.** À défaut, WS-1, et le canal devient le premier élément du sprint 2.
L'écart d'effort est faible (un composant réduit à une règle, un canal déjà expérimenté), alors que le risque levé est le plus élevé du projet.

## Avis sécurité

Fonction : Sécurité — 2026-10-09. Référence : [exigences-securite.md](../securite/exigences-securite.md) (menaces M-xxx, exigences ES-xx, frontières FC1 à FC10).

### 1. Comparaison des options

La **surface externe est la même pour les trois options** : point d'entrée HTTPS, origine des photos publiées, pages du serveur d'autorisation, point de réception des notifications du prestataire (FC1, FC4). Le découpage ne change que la surface interne, le rayon d'impact et la gestion des identités entre composants.

| | A — Modulaire, contrôle extrait | B — Quatre services | C — Domaines fins |
|---|---|---|---|
| Flux internes à authentifier | Deux composants sur le canal (plateforme, contrôle), plus l'outil de fraude. Aucun appel synchrone entre composants. Les appels entre modules restent dans le processus. | Canal avec diffusion (plusieurs abonnés par événement), plus un appel synchrone Transactions → Place de marché. Au moins quatre identités de service. | Une dizaine d'identités, une passerelle, des enchaînements compensés : chaque flux est à authentifier, autoriser et observer. |
| Rayon d'impact si le composant le plus exposé est compromis (décodage des photos, contenu utilisateur) | Tout le déployable : données de tous les modules (profils, conversations, commandes) et tous les secrets d'intégration (clé du prestataire, clé d'envoi d'emails, accès à la base, au stockage et au canal). Cartes et fonds restent hors d'atteinte (ES-12, ES-15). | Place de marché : catalogue, comptes et profils, notation. Transactions (clé du prestataire) et Interactions (conversations, clé d'emails) restent hors d'atteinte. | Le seul service touché. Mais secrets et droits sont répartis sur une dizaine de services : c'est l'option où une erreur de configuration est la plus probable (M-LIV-02, M-LIV-03). |
| Frontière financière, PCI-DSS | Clé du prestataire et point de réception des notifications dans le même processus que les autres modules. | Isolés dans Transactions. | Isolés dans le service Commandes et paiement. |
| Identités de service, moindre privilège | Le plus simple à rendre correct : deux identités sur le canal, aux droits séparés. **Dans la plateforme, la frontière entre modules n'est pas une frontière de sécurité**, sauf compte d'accès à la base distinct par module. | Identités de service dès la V1, droits par sujet de diffusion, authentification mutuelle sur l'appel synchrone. | Idem, multiplié par dix, plus la propagation de l'identité de l'utilisateur derrière la passerelle : chaque service doit revérifier le jeton. |
| Bilan | Surface interne minimale, rayon d'impact large : acceptable avec les mesures du § 2. | Paiement mieux isolé, au prix de nouveaux flux internes et d'une réservation répartie entre deux services (M-PAY-05). | Rayon d'impact théorique minimal, mais le risque réel le plus élevé pour une équipe de trois (configuration, correctifs, observabilité). |

**PCI-DSS** : la réduction du périmètre vient d'ES-12 (carte saisie chez le prestataire), pas du découpage. Dans les trois options, le front qui lance la redirection vers le prestataire reste dans le périmètre, et le questionnaire applicable est le même, le plus léger. Point à confirmer avec le prestataire retenu.

**Point clé** : les cartes et les fonds sont déjà hors de la plateforme. Le vecteur de compromission le plus probable est le décodage de fichiers non fiables et les dépendances vulnérables (M-ANN-01, M-LIV-01). Isoler ce vecteur réduit le rayon d'impact à moindre coût qu'isoler le paiement.

### 2. Variante « extraire Commandes et paiement dès la V1 » : utile plus tard, pas nécessaire en V1

- **Le niveau de risque ne baisse pas** : même après l'extraction, une compromission de la plateforme reste d'impact 4 (fuite massive des profils et des conversations). La variante supprime une conséquence (fraude financière par la clé du prestataire), pas le niveau du risque.
- **Elle ajoute des risques** : un appel synchrone authentifié pour le prix de référence (ES-13), et une réservation de l'objet unique répartie entre deux services, qui réintroduit le double achat (M-PAY-05) et la compensation que D01 a jugée complexe.
- **La conséquence financière peut être bornée autrement.** Quatre mesures compensatoires, à rendre **Must V1** si la variante n'est pas retenue :
  1. **Décodage isolé des images** : décodage et ré-encodage dans un processus isolé, sans secret, sans accès réseau ni à la base, avec des limites de ressources (complément d'ES-17).
  2. **Clé du prestataire restreinte** aux opérations nécessaires, avec plafonds et alertes côté prestataire sur les remboursements et les reversements ; reversements seulement vers des vendeurs vérifiés par le prestataire, avec délai de carence sur de nouvelles coordonnées (ES-11) (complément d'ES-12).
  3. **Compte d'accès à la base distinct** pour le module Commandes et paiement, au minimum, avec des droits limités à son espace de données.
  4. **Journaux de sécurité envoyés hors du processus**, en ajout seul, pour qu'un attaquant ne puisse pas effacer ses traces (ES-23).

**Critère E3 à déclencher** dès que l'un de ces faits est constaté :
- le prestataire retenu ne permet ni de restreindre la clé, ni de plafonner les remboursements et les reversements ;
- le montant en séquestre dépasse un seuil d'exposition fixé par la direction ;
- la plateforme doit traiter de nouveaux contenus non fiables complexes (pièces jointes du chat, vidéo de la vente en direct EF-49) ;
- les enchères arrivent (EF-48) : elles réutilisent le paiement, et l'extraction se fait alors naturellement avec E2 ;
- un incident ou un constat d'audit critique d'exécution de code sur la plateforme ;
- le prestataire ou le questionnaire PCI-DSS applicable impose de séparer les composants liés au paiement.

La variante est hors POC : la décision ne change ni le POC ni son coût.

### 3. Cohérence des frontières de confiance et nouvelles menaces

Les frontières FC1 à FC10 du schéma reprennent fidèlement l'analyse de menaces, et les contrôles attendus correspondent aux exigences ES. Cinq écarts :
1. **Les frontières entre modules ne sont pas des frontières de confiance** dans A : même processus, mêmes secrets. « Un module est seul à lire et écrire ses données » est une règle de conception vérifiée en CI, pas un contrôle de sécurité. La mention de FC8 « accès par le seul composant propriétaire » n'est vraie qu'entre composants ; entre modules, il faut un compte de base par module.
2. **Le canal entre deux déployables est une frontière absente de l'analyse.** À ajouter comme FC11 : plateforme ↔ canal ↔ contrôle de publication, dans la zone applicative privée.
3. **L'outil de fraude est relié directement au canal** dans le schéma. Un outil acheté, hors de la zone privée, ne doit jamais accéder au canal interne : le module Fraude, ou un adaptateur, relaie les échanges. À rendre systématique pour tout outil externe.
4. **POC : l'écran « à vérifier » est servi sur la même origine que le contenu public.** Un script injecté dans une annonce s'exécute alors dans la session de l'admin (M-ANN-03 puis M-BO-01). Dans le POC, la neutralisation des sorties (ES-08) et la politique de sécurité du contenu (ES-07) sont la seule barrière : à intégrer au POC plutôt que laissées « recommandées ».
5. **Les notifications du prestataire entrent par le point d'entrée public.** Le routage doit les limiter au seul point de réception du module Commandes, vérifié par signature et sans session utilisateur (ES-14). À préciser dans D08.

Nouvelles menaces, à ajouter lors de la révision d'exigences-securite.md (risque brut) :

| ID proposé | STRIDE | Scénario | V | I | Risque | Mesure proposée |
|---|---|---|---|---|---|---|
| M-CAN-01 | S, T | Un composant ou un acteur qui peut écrire sur le canal y dépose un faux résultat « publié » (accès partagé, droits trop larges, outil externe connecté au canal). | 2 | 3 | 6 Modéré | Une identité par composant. Seul le contrôle écrit les résultats, seule la plateforme écrit les demandes. Contrat validé à la réception. |
| M-CAN-02 | T | Un résultat périmé ou rejoué est appliqué à une nouvelle révision de l'article. | 2 | 3 | 6 Modéré | Déjà traité par le numéro de révision et l'idempotence (architecture § 4.2). À couvrir par un test automatisé. |
| M-CAN-03 | D | Un message malformé fait échouer le contrôle en boucle et bloque la file, ou un afflux de soumissions sature le canal. Le contrôle étant fermé par défaut, plus rien n'est publié. | 3 | 2 | 6 Modéré | Validation du contrat, mise à l'écart d'un message après un nombre borné d'échecs, alerte sur la file en attente, limite de soumissions par vendeur (ES-29). |
| M-CAN-04 | I | Les messages durables, files d'erreurs comprises, conservent sans limite du contenu saisi par les utilisateurs. | 2 | 2 | 4 Modéré | Contrat minimal (déjà prévu), durée de conservation des messages, chiffrement au repos. |
| M-MOD-01 | I, E | Avec un compte d'accès à la base commun, une injection ou un défaut dans un module (recherche du catalogue) lit ou modifie les données des autres (commandes, profils). | 2 | 4 | 8 Élevé | Un compte de base par module, limité à son espace ; au minimum un compte distinct pour Commandes et paiement. |
| M-MOD-02 | E | Une interface interne ne revérifie pas l'autorisation : l'API publique d'un autre module l'appelle avec des paramètres choisis par l'utilisateur (adjoint confus), accès indirect à la ressource d'autrui. | 2 | 3 | 6 Modéré | Les interfaces internes reçoivent l'identité de l'appelant et appliquent ES-04. Tests d'autorisation négatifs sur les chemins indirects. |
| M-MOD-03 | I, E, T | Exécution de code dans le déployable unique (décodage d'une image, dépendance vulnérable) : accès à toutes les données et tous les secrets (remboursements et reversements par la clé du prestataire, hameçonnage depuis le vrai domaine par la clé d'emails, effacement des journaux). | 2 | 4 | 8 Élevé | Mesures compensatoires du § 2 ; plafonds d'envoi d'emails (ES-25) ; critère E3. |

### 4. Écart ES-18 / ES-16 (titre et nom de boutique)

Écart confirmé. Correction proposée, reportée à la prochaine révision d'exigences-securite.md :
- **Remplacer la liste par un principe** : « Toute modification d'un champ examiné par une règle du contrôle renvoie l'article au contrôle. » Cela couvre le titre, la description, les photos, la catégorie et le prix au-delà d'un seuil, et suit l'ajout de nouvelles règles.
- **Nom de boutique** : il appartient à la boutique, pas à l'article. Le renommage ne renvoie pas tous les articles au contrôle ; le nouveau nom n'est affiché qu'après le même filtrage des coordonnées (ES-16), fermé par défaut (ES-31). Même règle pour les champs de profil visibles des autres.
- **Tests** : modifier le seul titre d'un article publié le renvoie « en contrôle » ; un nom de boutique contenant un numéro n'est pas affiché.
- **POC** : les boutiques sont fictives (EF-11 hors POC), seul le titre compte. Le § 4.2 de l'architecture et les AC des US doivent ajouter le titre.

### 5. Rôle d'accès unique imposé par l'environnement de formation

**Impact** : la plateforme, le contrôle et sans doute la chaîne de livraison s'exécutent sous la même identité. Le POC ne peut pas démontrer le cloisonnement par l'hébergeur entre composants (qui lit quel secret, qui écrit dans quelle file, qui accède à quel stockage).

**Ce qui reste démontrable** (à vérifier pendant l'expérimentation J3) : identifiants propres à chaque composant auprès du canal si le canal retenu en D05 a sa propre authentification ; comptes de base distincts gérés par la base ; chaque composant ne reçoit au déploiement que ses propres secrets (cloisonnement par configuration) ; photos privées jusqu'à la publication ; credentials temporaires de session conservés dans le stockage de secrets de la forge et renouvelés à chaque session (ES-06).

**Écart à documenter** (documentation d'exploitation et audit V1) :
> « Dans le POC, tous les composants partagent l'identité d'exécution imposée par l'environnement de formation. Un composant compromis peut atteindre les ressources des autres (canal, base, stockage, secrets). La V1 exige une identité d'exécution par composant, limitée à ses propres ressources. »

**Aucun impact sur le critère C8** : les cinq bonnes pratiques décidées dans D01 sont applicatives. En soutenance, le moindre privilège au niveau de l'hébergeur est présenté comme un écart connu, avec sa remédiation V1.

### 6. Verdict

Avis favorable à l'option A sans extraction du module Commandes et paiement en V1, à condition de rendre Must V1 les quatre mesures compensatoires du § 2 (décodage isolé des images, clé du prestataire restreinte et plafonnée, compte de base distinct pour Commandes et paiement, journaux hors du processus), d'inscrire dans la règle E3 les déclencheurs d'extraction listés, et d'ajouter la frontière FC11 (canal) avec ses menaces.

## Avis faisabilité

Fonction : Développement back — 2026-10-09. Périmètre : effort, modèle métier, contrats, frontières de modules, tests, idempotence. Estimations en jours de développement back, hors front, infrastructure et tests : ordres de grandeur, à confirmer après D04 et D05.

### 1. Effort de réalisation

**POC.** La tranche est la même pour A et B ; C l'élargit.

| Poste | A — Modulaire, contrôle extrait | B — Gros grain | C — Domaines fins |
|---|---|---|---|
| Modélisation | Article, révision, décision, photo, statut vendeur ; un propriétaire par donnée (§ 2) | Identique à A | A, plus une copie de lecture du catalogue à modéliser et à tenir à jour |
| Contrats et API | Une API, trois publics (lecture publique, vendeur, admin), un format d'erreur ; contrat v1 du contrôle : deux messages | Identique à A | A, plus la passerelle, un contrat d'événements annonces → catalogue et un accès réseau au statut vendeur |
| Frontières de modules | Deux modules dans un déployable : interface interne, données privées à chaque module, règle de dépendance en CI | Non exercées : Annonces et Comptes sont dans le même service, rien de propre à B n'est prouvé | Frontières réseau entre au moins quatre composants, chacune authentifiée |
| Tests | Règles en fonctions pures, machine à états, un test de contrat, acceptation sur deux composants et le canal | Identique à A | A, plus des tests de la copie de lecture, et des AC qui tolèrent son retard |
| Idempotence côté consommateur | Un seul consommateur avec effet : le résultat du contrôle, idempotent grâce à la garde d'état et de révision (§ 2) | Identique à A | A, plus la copie du catalogue, sensible à l'ordre des événements |
| Effort back | ≈ 15 à 20 j, tient dans S1 + S2 | ≈ 15 à 20 j | ≈ 25 à 30 j, déborde S2 |

**Point bloquant propre à C** : la copie de lecture du catalogue est mise à jour avec retard. Un article publié puis modifié reste visible dans la copie tant que l'événement de retrait n'est pas traité, ce qui ouvre la fenêtre de contournement qu'ES-18 et ES-31 interdisent (M-ANN-05). La fermer impose soit de vérifier en direct auprès des annonces (ce qui ôte tout intérêt à la copie), soit d'ordonner les événements par révision avec une copie qui refuse tout retour en arrière.

**Détail de l'option A pour le POC :**

| Poste back | Jours |
|---|---|
| Socle de la plateforme : structure modulaire, vérification des jetons et des rôles, format d'erreur, configuration externalisée, journaux | 2 à 3 |
| Module Annonces : modèle, machine à états, révision, API publique, vendeur et admin, décisions | 4 à 6 |
| Upload sécurisé (ES-17) : contenu vérifié, taille, ré-encodage, nom généré par le serveur, stockage privé, publication | 2 à 3 |
| Module Comptes minimal | ≤ 0,5 |
| Contrôle : quatre règles, détection des coordonnées à coût borné, plages de prix en configuration | 2 à 3 |
| Partie asynchrone : envoi sans perte, réception du résultat, garde de révision | 2 à 3 |
| Contrat v1 et prise en charge du test de contrat | 1 |
| Données fictives rejouables | 1 |
| **Total** | **≈ 15 à 20** |

**V1 cible.** Le code métier est le même dans les trois options ; seul le coût d'intégration change.

| Poste | A | B | C |
|---|---|---|---|
| Contrats | ≈ 6 interfaces internes et ≈ 11 types d'événements internes, dans le même processus (architecture § 2.2) ; un contrat externe (contrôle, étendu à la fraude) | Au moins 4 contrats réseau entre services, avec diffusion et authentification de service à service | Environ 10 services, quelques dizaines de contrats, une saga pour l'achat |
| Cohérence des API | Une API versionnée, un seul style par construction | Quatre API à aligner : une convention commune à écrire et à faire respecter | Une dizaine d'API derrière une passerelle : styles divergents probables sans règle commune |
| Frontières | Tenues par la CI et la revue ; risque principal : l'érosion | Tenues par le réseau ; certains regroupements imposent des allers-retours (notation ↔ transactions, notifications ↔ intérêts) | Tenues par le réseau, au prix de l'exploitation |
| Données | Une base, un espace par module ; ni clé étrangère ni jointure entre espaces | Un espace par service ; « réservé » et « vendu » mis à jour avec retard | Une base par service ; copies de lecture à reconstruire |
| Tests | Tests de module dans le même processus ; un test de contrat par contrat externe | Un test de contrat par paire producteur-consommateur ; environnement d'intégration à quatre services | Sagas, compensations, copies de lecture ; environnement à une dizaine de services |
| Idempotence | Traitements d'événements à effet externe (notifications, emails), contrôle, fraude | Chaque consommateur de chaque service, plus l'ordre d'arrivée (« réservé » puis « vendu ») | Comme B, plus chaque étape de saga et chaque compensation |
| Surcoût d'intégration (ordre de grandeur) | Référence | ≈ × 2 | ≈ × 4 ; devient le poste principal |

### 2. Cohérence du modèle

**Frontières nettes pour l'essentiel.** Trois choix sont cohérents : le contrôle sans état, les décisions de l'admin portées par Annonces, le back-office sans domaine propre. Cinq points sont à reprendre :

1. **Propriété de la réservation (V1)** : les documents se contredisent (tableau des domaines et architecture § 5.1 : Commandes ; schéma § 2.2 : interface d'Annonces ; option B : Transactions). Recommandation : la **disponibilité** de l'article (disponible, réservé jusqu'à une échéance, vendu) appartient à Annonces. Commandes l'appelle pour réserver, puis crée la commande ; une réservation non confirmée expire d'elle-même. Les dépendances restent sans cycle et le catalogue lit la disponibilité dans ses propres données.
2. **« Transaction locale » réservation + commande (V1)** : elle couvrirait les données de deux modules, contraire à la règle de propriété des données, et disparaît si Commandes et paiement est extrait (variante E3), ce qui ramène la compensation que D02 veut éviter. La réservation à échéance du point 1 n'a besoin d'aucune transaction commune et reste valable après extraction.
3. **La catégorie est répartie entre deux composants** : catégories dans Annonces, plages de prix dans la configuration du contrôle. En V1, dès que l'admin crée une catégorie (EF-05), les deux peuvent diverger. Le contrat v1 fixe donc qu'une catégorie inconnue du contrôle donne « à vérifier », jamais « publié » (ES-31). Dans le POC, catégories et plages viennent d'une seule source de données fictives.
4. **Trois notions « vendeur » à ne pas confondre** : le rôle vendeur appartient au serveur d'autorisation (ES-03) ; la propriété d'une boutique appartient à Annonces (ES-04) ; le statut vendeur appartient à Comptes et sert une règle du contrôle. Les trois partagent l'identifiant opaque de compte émis par le serveur d'autorisation, déjà pseudonyme (architecture § 8).
5. **Chemin de lecture du catalogue** : dans le POC, la fiche ne lit aucune donnée d'un autre module (« vendeur particulier » est un libellé fixe, EF-11). En V1, la note moyenne du vendeur (Notation) est recopiée dans Annonces par événement, pour éviter un appel par article (ENF-14).

**Entités de la tranche POC**

| Entité | Propriétaire | Contenu essentiel | Remarque |
|---|---|---|---|
| Catégorie | Annonces (fictive) | Identifiant, libellés français et anglais, active | Libellés bilingues dès les données fictives (ENF-30) |
| Boutique | Annonces (fictive) | Identifiant, identifiant du vendeur, nom | Base du contrôle de propriété (ES-04) |
| Article | Annonces | Identifiant non prévisible, boutique, état de publication, révision courante, titre, description, catégorie, prix, frais de port | Montants en centimes avec leur devise (ENF-01, ENF-31) |
| Photo | Annonces | Identifiant généré par le serveur, article, rang, référence de stockage | Publique seulement quand l'article est publié (ES-17) |
| Décision de contrôle | Annonces, ajout seul | Article, révision, issue, catégorie de motif, version des règles, auteur (contrôle ou admin), date | Source du motif montré au vendeur (EF-18) ; distincte du journal d'audit (ES-23) |
| Statut vendeur | Comptes (fictif, lecture seule) | Identifiant du vendeur, inscrit depuis, suspendu | Lu à la soumission seulement |
| Demande et résultat de contrôle v1 | Contrat ; le contrôle ne les conserve pas | Demande : version du contrat, article, révision, données utiles aux règles (dont nombre de photos et dates utiles). Résultat : issue, catégorie de motif, version des règles | Le contrôle ne lit ni base, ni stockage, ni horloge : règles déterministes et rejouables |

**La révision est un compteur, pas une copie du contenu.** Elle augmente à chaque envoi au contrôle. Toute modification d'un article en contrôle, à vérifier ou publié le renvoie au contrôle : le contenu examiné est toujours le contenu courant. Le POC n'a besoin d'aucune table de versions ; l'historique des prix (EF-13) reste en V1.

**États et transitions de l'article (POC)**

| État | Au catalogue | Modification par le vendeur | Sortie |
|---|---|---|---|
| Brouillon | Non | Libre | Soumission → en contrôle |
| En contrôle | Non | Nouvelle révision, nouvelle demande, reste en contrôle | Résultat de la révision courante → publié, refusé ou à vérifier |
| À vérifier | Non | Nouvelle révision → en contrôle | Décision admin sur la révision courante → publié ou refusé |
| Refusé | Non | Libre | Soumission → en contrôle |
| Publié | Oui | Nouvelle révision → en contrôle (ES-18) | — |

La disponibilité commerciale (réservé, vendu ; V1) est une dimension distincte de l'état de publication.

**Règles de garde** (idempotence sans table de messages traités) :
- Un résultat n'est appliqué que si l'article est « en contrôle » et que la révision du résultat est la révision courante ; sinon il est ignoré sans erreur. Cette seule règle traite doublons, résultats périmés et livraisons tardives.
- La décision de l'admin porte la révision examinée ; si l'article a changé entre-temps, elle est refusée pour conflit (409) et l'admin recharge la page. Sans cela, l'admin peut publier une version qu'il n'a pas vue (M-ANN-05).
- Les doublons étant sans effet, les instances de la plateforme n'ont pas à se coordonner : chacune peut recevoir les résultats et renvoyer les demandes restées sans réponse.

**Simplification proposée** : dans le POC, **toute** modification d'un article publié le renvoie au contrôle, sans seuil de prix. Gains : un paramètre en moins, la question du titre (écart ES-18) est réglée, l'extension « prix sous le seuil » de D01 devient inutile. Coût : un article retouché quitte le catalogue le temps du contrôle (quelques secondes).

**Contrats d'API du POC**
- Une seule ressource « article », vue de trois façons : publique (articles publiés seulement), vendeur (ses articles, tous états), admin (articles « à vérifier »).
- La soumission répond « accepté » (202) avec l'état « en contrôle » : le résultat arrive de façon asynchrone.
- L'article d'un autre vendeur (ES-04) et un article non publié en lecture publique (ES-31) répondent « introuvable » (404) plutôt qu'« interdit » (403), pour ne pas confirmer leur existence. Les AC de refus attendent ce code.
- La décision admin exige la révision ; si elle est périmée, réponse « conflit » (409).
- Un seul format d'erreur et un même préfixe de version pour les trois vues (ENF-19).

**Risques de couplage dans le POC, et parades**

| Risque | Parade |
|---|---|
| Le contrôle lit les photos pour vérifier la complétude | Le message porte le nombre de photos ; les fichiers sont validés au dépôt côté Annonces (ES-17) ; le contrôle n'a aucun accès au stockage. |
| Le contrôle lit une base ou l'horloge (ancienneté du vendeur) | Le message porte les dates utiles ; le contrôle reste une fonction pure, testable et rejouable. |
| Les deux déployables partagent du code pour les messages | La référence est le contrat versionné ; un code commun éventuel se limite aux types de messages, sans logique ni accès aux données. |
| À la publication, la base et le stockage sont écrits hors d'une même transaction | L'ouverture des photos est une étape rejouable et idempotente, relancée jusqu'au succès ; autre solution : décider la visibilité à la lecture, d'après l'état de l'article (D07). |
| Appel à Comptes sur le chemin de lecture | Comptes n'est appelé qu'à la soumission ; le tir de charge ne traverse qu'Annonces. |
| Clé étrangère ou jointure entre Annonces et Comptes | Interdites ; l'identifiant du vendeur est la seule référence. |

### 3. Module « Comptes minimal »

**Utile, à garder, à trois conditions.**
- La règle « statut du vendeur » (EF-17) a besoin d'une donnée, et les autres emplacements sont moins bons : dans Annonces (mauvais propriétaire, donnée à migrer en V1), dans la configuration du contrôle (le contrôle deviendrait une copie des comptes, ce qu'un outil acheté n'aurait pas, ENF-17), dans le jeton (le serveur d'autorisation fournit l'identité et les rôles, pas un statut métier).
- C'est la seule preuve du monolithe modulaire dans le POC : avec un module unique, la règle de dépendance en CI et l'interdiction d'accès direct aux données d'un autre module n'auraient rien à vérifier (C2, ENF-21).
- Coût : moins d'une demi-journée.

Conditions : (1) aucune API exposée, une interface interne en lecture seule appelée par Annonces à la soumission ; (2) ni clé étrangère ni jointure avec Annonces, ni événement ni écriture dans le POC ; (3) règle précisée dans l'US (par exemple : vendeur suspendu → refusé ; vendeur récent et prix au-delà d'un seuil → à vérifier).

Option à faible coût qui renforce la preuve : un compte d'accès à la base par module ; la base refuse elle-même un accès direct d'Annonces aux données de Comptes, démontrable par un test négatif.

### 4. Walking skeleton : WS-1 ou WS-2

| Critère | WS-1 | WS-2 |
|---|---|---|
| Effort back S1 | ≈ 5 à 7 j : socle, création d'article, lecture publique, données fictives | ≈ 9 à 12 j : WS-1, plus le contrôle à une règle, l'envoi sans perte, la réception du résultat et le contrat v1 |
| Effort back S2 | ≈ 10 à 13 j | ≈ 6 à 9 j |
| Effort total | Identique : rien n'est jetable, WS-2 avance du travail sans en ajouter | Identique |
| Risque back | La partie asynchrone (perte au redémarrage, doublons, résultats périmés) est construite au sprint le plus chargé, juste avant la charge et la mise à l'échelle | La partie asynchrone sert pendant tout S2 ; S2 n'ajoute que des règles, des photos et des écrans |
| À trancher avant J4 | D04 | D04, expérimentation du canal concluante, D05 tranchée sur trois points |
| Tentation de raccourci | Publier directement, à titre temporaire : à proscrire (ES-31) ; les articles « publiés » fictifs suffisent au catalogue de S1 | Aucune |

**Conditions back de WS-2 :**
1. **D05 tranchée avant J4** sur trois points : mode de retour du résultat, mécanisme anti-perte, contenu du contrat v1. Préférence pour l'anti-perte : renvoyer au contrôle les articles restés « en contrôle » au-delà d'un délai ; la demande se reconstruit entièrement à partir de l'article, un horodatage du dernier envoi suffit, les doublons sont neutralisés par la garde de révision.
2. **Contrat v1 complet dès S1** : les trois issues, les catégories de motif, le nombre de photos, les attributs du vendeur, la version des règles. Ajouter en S2 une issue ou un champ obligatoire casserait un consommateur strict et imposerait une v2 en milieu de projet.
3. **Règle unique de S1** : la complétude porte sur le titre, la description, la catégorie et le prix ; le critère photo (EF-12) s'ajoute en S2 avec une nouvelle version des règles, ce qui démontre l'usage de ce champ du contrat.

**Recommandation back : WS-2**, aux conditions de D02 et aux trois conditions ci-dessus. À défaut, WS-1 sans publication directe temporaire ; la partie asynchrone devient alors le premier poste de S2.

### 5. Verdict

Option A : faisable seul pour le POC (≈ 15 à 20 jours de développement back, dans l'enveloppe de D01) et adaptée à trois développeurs en V1, si la CI fait tenir les frontières de modules et si la propriété de la réservation est clarifiée ; garder le module Comptes minimal en interface interne en lecture seule, et retenir WS-2 si D05 et l'expérimentation du canal sont prêtes avant J4.

## Décision de Martin

> **Découpage : « B : 4 services »** : Place de marché, Transactions, Interactions, Contrôle de publication, chacun propriétaire de ses données.
>
> **Commandes et paiement : « Oui, dès la V1 »** : le périmètre financier (commandes, réservation, paiement, commission, séquestre) est un service à part entière dès la V1, ce que l'option B réalise avec le service Transactions.
>
> **Walking skeleton : « WS-2 : 2 composants + canal »**, aux conditions de la recommandation (expérimentation du canal concluante sur l'environnement managé et D05 tranchée avant J4 ; à défaut, WS-1). [PILOTAGE.md](../PILOTAGE.md) § 3.4 est amendé en conséquence.
>
> **Raisons de Martin** :
> - **Isoler la finance** : paiement, séquestre et commission ont leur propre service, leurs secrets, leurs données et leurs déploiements ; une compromission du catalogue n'atteint pas l'argent.
> - **Évolutivité du marché** : enchères, vente en direct, bot avant-vente et analytics se branchent sur des services et des événements existants sans toucher au cœur (« réagir à l'évolution du marché »).
> - **Mise à l'échelle par domaine** : catalogue (lecture intensive), transactions et messagerie ont des profils de charge différents, dimensionnés séparément.
>
> Analyse de menaces [exigences-securite.md](../securite/exigences-securite.md) validée par Martin le même jour comme référence de sécurité du projet.

Date : 2026-10-09

**Complément à instruire** : la recommandation et les avis ont été rédigés pour l'option A. Avec B, les points clés propres à l'architecture retenue (diffusion d'événements, authentification de service à service, appel synchrone interne, cohérence différée) doivent être exercés par le POC ou par le bac à sable pour tenir C2 au niveau 5. Le périmètre du POC adapté à B est soumis à Martin dans la section suivante.

## Complément — périmètre du POC pour l'option B (tranché par Martin : option P2)

### Problème

- L'option B le dit elle-même : « les points clés propres à B (appel synchrone entre services, diffusion, cohérence différée, authentification de service à service) ne sont pas exercés par le POC ».
- Or le niveau 5 de C2 exige que « l'implémentation technique du POC valide la faisabilité des points clés de l'architecture de la solution applicative à développer (découpage applicatif, communication interservice, sécurité, hébergement, orchestration, etc.) ». L'architecture à développer est désormais B.
- Le socle commun (Place de marché, Contrôle et canal point à point, repris dans « Périmètre du POC ») exerce déjà : deux services, un échange asynchrone point à point par contrat versionné, l'idempotence, le contrôle fermé par défaut, une identité par service sur le canal, deux profils de mise à l'échelle.

Points clés propres à B :

| Code | Point clé | Où il apparaît en V1 | Couvert par le socle |
|---|---|---|---|
| K1 | Diffusion : un même événement reçu par plusieurs consommateurs indépendants | Décision de publication, commande payée, alerte de fraude | Non |
| K2 | Authentification de service à service | Identités sur le canal ; appel synchrone Transactions → Place de marché | Sur le canal seulement |
| K3 | Appel synchrone entre services | Réservation d'un article par Transactions | Non |
| K4 | Cohérence différée | Notifications, passage à « vendu », copie des centres d'intérêt | Non |
| K5 | Isolement du périmètre financier (première raison de Martin) | Service Transactions | Non |

### Option P1 — Service Interactions minimal avec diffusion réelle ; appel synchrone validé en bac à sable

- **Contenu** :
  - La Place de marché émet l'événement « décision de publication » (article, révision, issue, catégorie de motif, vendeur) **après** avoir appliqué un résultat du contrôle ou une décision de l'admin. L'événement est diffusé.
  - Un service Interactions réduit à une fonction le consomme : il crée une notification au vendeur, listée dans son espace (EF-34 et EF-33, partiels). Le texte est construit à partir de la catégorie de motif, en français et en anglais, sans reprendre de contenu saisi par un utilisateur (M-NOT-02).
  - Pourquoi cet événement plutôt que le résultat du contrôle : il est émis par le propriétaire de l'article, seulement quand la décision est appliquée, et il couvre aussi les décisions de l'admin. Diffuser le résultat brut ferait notifier au vendeur des résultats périmés, que la Place de marché ignore.
  - Second abonné : en V1, l'adaptateur de fraude (EF-42). Dans le POC, la diffusion est prouvée par un second abonnement posé par le test d'intégration : il reçoit le même événement, y compris quand Interactions est arrêté.
  - Garde-fous : ni email, ni préférences, ni suivis, ni messagerie.
  - L'appel synchrone authentifié (K3, et K2 sur ce flux) est validé dans le bac à sable, entre deux services de test, dans l'expérimentation du serveur d'autorisation ou du canal (D14).
- **Points clés exercés dans le POC** : K1 (un consommateur métier et un abonnement de test), K2 sur le canal (trois identités, droits par flux), K4 (notification différée, délai borné dans les AC), découpage à trois services déployés. Hors POC : K3 en bac à sable, K5 documenté.
- **Surcoût** : environ 5 à 7 jours (back 3 à 4, front 1, infrastructure 1 à 2) ; un déployable de plus, à une instance ; un espace de données et un compte d'accès ; un pipeline ou un déclencheur de plus.
- **Impact sur WS-2** : contenu inchangé (deux composants et le canal) ; Interactions arrive au sprint 2. Le mode de diffusion doit être tranché en D05 avant J4 et testé dans l'expérimentation du canal, pour ne pas changer de canal au sprint 2.
- **Impact sur la charge (C5)** : tir sur la consultation inchangé. Troisième profil de mise à l'échelle (messages en attente). Démonstration possible : Interactions arrêté pendant une vague de soumissions, publications non affectées, notifications rattrapées au retour (ENF-12).
- **Impact sur les tests (C7)** : un test de contrat de plus (Place de marché → Interactions) ; un AC « le vendeur voit le résultat dans son espace dans un délai borné » ; un test « Interactions arrêté puis relancé, aucune notification perdue » ; un test négatif par objet (un vendeur ne lit pas les notifications d'un autre, ES-04) ; un test négatif sur le canal (Interactions ne peut pas écrire un résultat de contrôle, M-CAN-01).
- **Risques** : le sprint 2, déjà le plus chargé, prend environ une semaine de plus ; dérive vers des notifications complètes (garde-fous ci-dessus) ; K3 prouvé hors du POC pèse moins pour C2 qu'une preuve dans le POC.

### Option P2 — P1, plus un service Transactions réduit à la réservation

- **Contenu** : P1, plus un service Transactions minimal. Un acheteur authentifié réserve un article publié. Transactions appelle la Place de marché de façon synchrone et authentifiée (« réserver jusqu'à une échéance »), reçoit le prix de référence, et crée une commande « en attente de paiement » avec la commission calculée (EF-24, ENF-01). Pas de prestataire de paiement : la réservation expire à l'échéance.
- **Points clés exercés dans le POC** : K1 à K5 ; quatre services déployés.
- **Surcoût** : environ 12 à 15 jours (back 8 à 10, front 2, infrastructure 2 à 3) ; deux déployables de plus que le socle ; deux espaces de données.
- **Impact sur WS-2** : contenu inchangé. En revanche, le sprint 2 ne peut pas absorber P2 sans repousser la charge et l'autoscaling, ou sans un sprint supplémentaire.
- **Impact sur la charge (C5)** : un second scénario devient possible (rafale de réservations), mais chaque réservation consomme un objet unique : les données sont à reconstruire à chaque tir, le défaut relevé par D01 pour l'achat.
- **Impact sur les tests (C7)** : test de contrat de l'appel synchrone ; double réservation concurrente refusée ; expiration à l'échéance ; tests négatifs de service à service (un service non autorisé est refusé, une identité d'acheteur forgée est refusée).
- **Risques** : effort hors de l'enveloppe de D01 ; une « réservation sans paiement » est une fonctionnalité incomplète (C3), qui rouvre le périmètre d'achat écarté par D01 ; démonstration plus longue dans les 20 minutes ; coût accru dans le Learner Lab.

### Option P3 — POC limité au socle commun ; points propres à B validés en bac à sable

- **Contenu** : le POC reste à deux services. Diffusion, appel synchrone authentifié et cohérence différée sont validés dans les expérimentations du canal et du serveur d'autorisation (C1), entre services de test.
- **Points clés exercés dans le POC** : K2 sur le canal seulement. K1, K3 et K4 en bac à sable ; K5 documenté.
- **Surcoût** : aucun composant ; environ 1 à 2 jours ajoutés aux expérimentations.
- **Impact sur WS-2, la charge et les tests** : aucun.
- **Risques** : le descripteur de C2 porte sur l'implémentation du POC. Avec trois points clés sur quatre validés hors du POC, le jury peut estimer que le POC ne valide pas l'architecture retenue, et C2 n'atteint pas le niveau 5. Le bac à sable sert C1, il ne remplace pas C2.

### Synthèse du complément

| | P1 — Interactions minimal | P2 — Interactions et Transactions minimaux | P3 — Bac à sable |
|---|---|---|---|
| Services déployés dans le POC | 3 | 4 | 2 |
| K1 Diffusion | POC | POC | Bac à sable |
| K2 Service à service | POC (canal) ; bac à sable (appel synchrone) | POC | POC (canal) ; bac à sable (appel synchrone) |
| K3 Appel synchrone | Bac à sable | POC | Bac à sable |
| K4 Cohérence différée | POC | POC | Bac à sable |
| K5 Périmètre financier | Documenté | POC, sans prestataire | Documenté |
| Surcoût | ≈ 5 à 7 j, 1 déployable | ≈ 12 à 15 j, 2 déployables | ≈ 1 à 2 j, aucun déployable |
| Tient dans le sprint 2 | Oui, serré | Non | Oui |
| Risque pour C2 | Faible | Très faible, mais risque de planning | Élevé |

### Recommandation sur le complément

**Recommandation initiale (P1) non retenue : voir la décision de Martin sur le complément.**

**P1.**

1. Trois des quatre points clés de communication propres à B sont exercés dans le POC, pour environ une semaine de travail et un petit déployable, sans rouvrir l'achat écarté par D01.
2. Le seul point laissé au bac à sable, l'appel synchrone authentifié, est le plus classique et le moins risqué. Il ne dépend que du serveur d'autorisation, déjà présent dans le POC.
3. Rien n'est jetable : l'événement « décision de publication » et le service Interactions sont ceux de la V1.
4. L'isolement du périmètre financier (K5) se démontre par l'architecture et les menaces qu'il traite, pas par un service sans prestataire : le prouver sans paiement réel affaiblirait la preuve, comme D01 l'a relevé pour l'achat.

Conditions :
- D05 tranche avant J4 le mode de diffusion et les identités par service ; l'expérimentation du canal (D14) teste la diffusion et les droits par identité.
- L'expérimentation du serveur d'autorisation (D14) inclut un appel synchrone authentifié entre deux services.
- Les garde-fous d'Interactions sont tenus : une fonction, ni email ni préférences.
- Ordre de coupe si le sprint 2 déborde : d'abord les coupes déjà prévues par D01 (vague de soumissions en charge, confort de l'écran admin, photos multiples), puis Interactions. Dans ce dernier cas, repli sur P3, assumé et documenté.

#### Avis sécurité sur le complément

Fonction : Sécurité, 2026-10-09. Références : FC11 ([architecture-technique.md](../architecture/architecture-technique.md) § 2.3), menaces M-CAN-01 à M-CAN-04 et M-MOD-01 à M-MOD-02 (Avis sécurité § 3), § 6 d'[exigences-securite.md](../securite/exigences-securite.md).

**Condition commune.** Avec le rôle d'exécution unique, une identité par service ne se démontre que si le composant concerné authentifie lui-même ses clients : la base (un compte d'accès par service), le serveur d'autorisation (jetons de service), le canal (s'il a sa propre authentification avec des droits par identité). Si D05 retient un canal autorisé seulement par les rôles de l'hébergeur, les trois services y partagent une même identité et les droits par sujet passent en « documenté », quelle que soit l'option. À vérifier dans l'expérimentation du canal, avant J4.

**1. Ce que chaque option rend démontrable**

| Point de sécurité propre à B | P1 | P2 | P3 |
|---|---|---|---|
| Identité par service sur le canal | POC, 3 identités | POC, 3 identités | POC, 2 identités |
| Droits par sujet de diffusion | POC : seule la Place de marché publie la décision, Interactions ne fait que la lire | POC | Non (point à point seulement) ; bac à sable |
| Appel synchrone authentifié (jeton de service, audience, droit limité, identité de l'acheteur revérifiée) | Bac à sable | POC | Bac à sable |
| Données cloisonnées par service, accès refusé par la base | POC, 2 espaces | POC, 3 espaces | Réduit à « le Contrôle n'a aucun accès » |
| Jeton revérifié et accès par propriétaire sur un 2e service (BP2, BP5) | POC (notifications) | POC (notifications, réservations) | Non |
| Isolement du périmètre financier (K5) | Documenté | Structure seulement : ni prestataire, ni secret financier, identité d'exécution partagée | Documenté |

Documenté dans les trois options : une identité d'exécution par service (Avis sécurité § 5), l'authentification de l'outil de fraude (FC5, ES-21), la confirmation de paiement (ES-14), les emails (ES-24).

**2. Risques nouveaux et parades minimales**

P1 :
- **Faux événement « décision de publication »** (S, T ; extension de M-CAN-01) : notification trompeuse au vendeur dans le POC ; outil de fraude et acheteurs intéressés trompés en V1. Parade : seule l'identité de la Place de marché publie sur ce sujet ; Interactions valide le contrat à la réception (version, champs requis, issue et catégorie de motif dans des listes fermées) ; un message invalide est écarté après un nombre borné d'essais (M-CAN-03).
- **Notification qui révèle des données** (I) : un vendeur lit les notifications d'un autre (ES-04), ou le texte reprend du contenu saisi (M-NOT-02, FC10). Parade : destinataire pris dans l'événement ; liste filtrée par l'identité du jeton, jamais par un paramètre ; texte issu de la seule catégorie de motif ; événement minimal (identifiant opaque du vendeur), conservé pour une durée bornée (M-CAN-04).
- **Nouvelle API exposée** : Interactions vérifie lui-même le jeton (signature, expiration, émetteur, audience) ; le test BP2 est rejoué sur cette API.
- **Secrets supplémentaires** (identifiant de canal, compte de base) livrés au seul service Interactions et couverts par BP3 ; sous l'identité partagée, un service compromis peut lire les secrets des autres (écart documenté, Avis sécurité § 5).
- **Abonnement de test** : identité de test en lecture seule, abonnement retiré après le test ; aucun service n'a le droit de créer un abonnement.

P2, en plus de P1 :
- **Jeton de service trop large ou volé** : réservation de n'importe quel article. Parade : audience « Place de marché », droit limité à « réserver », durée courte ; échéance plafonnée ; une seule réservation active par acheteur.
- **Adjoint confus** (M-MOD-02) : la Place de marché revérifie le jeton de l'acheteur propagé et lit le prix de référence chez elle (ES-13).
- **Double réservation concurrente** (M-PAY-05) : réservation atomique, test de concurrence.
- **Secrets** : un identifiant de Transactions auprès du serveur d'autorisation et un compte de base ; sous l'identité partagée, l'isolement financier présenté serait surévalué, à dire explicitement.

P3 : aucun risque nouveau dans le POC ; le bac à sable passe la même détection de secrets et n'est pas déployé dans l'environnement du POC.

**3. Option qui sert le mieux C2 et C8**

- C8 est atteint dans les trois options par les cinq bonnes pratiques ; P1 et P2 les étendent à un second service et rendent démontrable le moindre privilège entre services (droits par sujet, refus par la base).
- P2 ajoute l'authentification de service à service dans le POC, mais coûte 12 à 15 jours, ne tient pas dans le sprint 2, et son gain sur K5 est annulé par l'identité d'exécution partagée.
- P3 laisse au bac à sable les droits par sujet et ne démontre pas le cloisonnement des données entre services.
- **P1 offre le meilleur rapport preuve / coût** : sous le rôle unique, il démontre dans le POC tout ce qui relève de l'application, sauf l'appel synchrone.

**4. Mesures à très faible coût pour P1** (environ 1 jour, à inscrire dans les AC du sprint 2) :
1. Tests négatifs sur le nouveau sujet : le Contrôle et Interactions ne peuvent pas publier de « décision de publication ».
2. Validation du contrat à la réception par Interactions, avec un test : un message mal formé est écarté sans bloquer les suivants.
3. Test de cloisonnement par la base : le compte d'Interactions est refusé sur l'espace de la Place de marché, et inversement.
4. Bac à sable de l'appel synchrone : tests négatifs (service non autorisé, mauvaise audience, jeton expiré, identité d'acheteur forgée), rapport archivé comme preuve pour C2.

**5. Verdict** : avis favorable à P1, à condition que le canal retenu en D05 authentifie lui-même chaque service et que les mesures 1 à 3 du point 4 figurent dans les AC du sprint 2 ; à défaut de ce canal, les droits par sujet ne sont démontrables dans aucune option.

**Décision de Martin sur le complément** (2026-10-09) :

> **« P2 : P1 + Transactions »** : le POC déploie les quatre services de l'option B. La Place de marché, le Contrôle de publication et un service Interactions minimal (diffusion réelle de « décision de publication », notification au vendeur), plus un service Transactions réduit à la réservation de l'article, qui exerce dans le POC l'appel synchrone authentifié Transactions → Place de marché.

Conséquences immédiates à reporter : schéma du POC de l'architecture (v0.1 dessiné selon P1) ; parades P2 de l'avis sécurité ci-dessus intégrées aux AC (jeton de service à audience et droit limités, revérification du jeton de l'acheteur, réservation atomique avec test de concurrence, une réservation active par acheteur, échéance plafonnée) ; mesures 1 à 3 du point 4 dans les AC ; effort supplémentaire (≈ 12 à 15 jours au total pour le complément) à absorber par le découpage en sprints ; scénario de charge maintenu sur la consultation du catalogue (les réservations consomment des objets uniques).

## Complément — repli si le sprint 3 ne tient pas (tranché par Martin le 2026-10-26)

> **« Oui, repli accepté »** : la cible reste les 4 services déployés (P2). Si le sprint 3 ne peut pas être tenu, le POC se replie sur P1 : Interactions et Transactions sont documentés (contrats, conception, tests prévus) mais non déployés. La démo reste complète sur le parcours soumission → contrôle → publication → consultation. Le repli est décidé à la revue du sprint 2 et consigné dans le bilan de sprint.

## Conséquences

- **Architecture** : [architecture-technique.md](../architecture/architecture-technique.md) passe en v0.2 : schéma du POC selon l'option P2 (§ 3), séquence de la réservation du POC (§ 4.3), distincte de l'achat V1 documenté (§ 4.4) ; FC11, règles d'architecture et réservation alignées sur la révision de sécurité (ES-38, M-RES-01).
- **Effort et découpage en sprints** : l'option P2 ajoute environ 12 à 15 jours (back 8 à 10, front 2, infrastructure 2 à 3) à l'enveloppe de D01 (≈ 15 à 20 jours de développement back). Le sprint 2 ne peut pas l'absorber sans repousser la charge et l'autoscaling : un sprint de plus est nécessaire (découpage indicatif ci-dessous).
- **D03** : quatre déployables à orchestrer dans le POC : deux au sprint 1 (WS-2), quatre au sprint 3 ; Transactions et Interactions à une instance. En V1 : Place de marché (au moins deux instances, consultation), Transactions (au moins deux instances, achat), Interactions et Contrôle (mis à l'échelle selon les messages en attente). Le moindre privilège entre services est limité par le rôle d'accès unique de l'environnement de formation (avis sécurité § 5).
- **D04** : une stack commune aux services, sauf justification service par service ; une convention d'API commune (format d'erreur, versionnement) à écrire et à faire respecter (avis faisabilité § 1).
- **D05** : canal à deux modes, point à point pour la demande et le résultat de contrôle, diffusion pour les événements métier ; contrat versionné de chaque message ; une identité par service, avec des droits par flux (trois identités dans le POC : Place de marché, Contrôle, Interactions ; Transactions n'utilise pas le canal) ; idempotence côté consommateur ; mécanisme anti-perte ; mode de retour du résultat ; réception des alertes de l'outil de fraude par l'adaptateur. À trancher avant J4 (condition de WS-2) : le canal du sprint 1 doit déjà permettre la diffusion du sprint 3 et authentifier lui-même chaque service, faute de quoi les droits par sujet restent documentés (avis sécurité sur le complément). **À confirmer en D05, deux règles posées par ES-38** : un événement venu d'un autre service peut suspendre un achat ou un reversement, jamais déclencher un mouvement de fonds ; aucun service ne crée d'abonnement sur le canal, opération réservée à l'exploitation, abonnement de test du POC compris (M-CAN-05, M-CAN-06 ; architecture § 6, règles 6 et 11).
- **D06** : authentification de service à service **exercée dans le POC** (sprint 3) pour l'appel Transactions → Place de marché : identité du service appelant, audience, droit limité à la réservation, durée courte, avec propagation de l'identité de l'acheteur, que la Place de marché revérifie (ES-04, M-MOD-02). Le service appelé est lui-même authentifié par Transactions (ES-38, M-SYN-01). Jetons des utilisateurs vérifiés par chaque service, Interactions et Transactions compris. À trancher avant le sprint 3.
- **D07** : une base, ou un espace avec son propre compte d'accès, par service : trois dans le POC (Place de marché, Interactions, Transactions). La disponibilité de l'article est dans les données de la Place de marché, avec un mécanisme de réservation atomique, la limite d'une réservation active par acheteur et le nouvel essai qui renvoie la réservation existante ; photos privées jusqu'à la publication.
- **D08** : point d'entrée unique avec routage par service (Place de marché, Interactions et Transactions dans le POC) ; route « réserver » de la Place de marché absente du point d'entrée public (ES-38, M-SYN-01), vérifiée par un AC de l'US de réservation sur l'inventaire des points d'entrée exposés ; route dédiée aux notifications du prestataire, vers Transactions seul et sans session utilisateur (avis sécurité § 3, point 5), en V1 ; origine distincte pour les photos ; origine du back-office en V1.
- **D09** : traces distribuées à considérer pour suivre un parcours à travers plusieurs services et le canal ; au minimum, un identifiant de corrélation propagé dans les appels et les messages. Dans le POC, la réservation traverse déjà Transactions et la Place de marché.
- **D10** : un pipeline par service, ou un monorepo avec des déclencheurs par chemin (le dépôt est déjà un monorepo, [PILOTAGE.md](../PILOTAGE.md) § 1) : quatre services dans le POC.
- **D11** : allumage et extinction de quatre déployables, du canal et de trois espaces de données ; garde-fou de coût dimensionné en conséquence (deux déployables de plus que le socle dans l'environnement de formation).
- **D13** :
  - un test de contrat par paire producteur-consommateur : Place de marché ↔ Contrôle ; Place de marché → Interactions ; Transactions → Place de marché (appel synchrone) ;
  - **test de concurrence de la réservation** : deux réservations simultanées du même article, une seule réussit (M-PAY-05, M-RES-01) ;
  - expiration à l'échéance ; une seule réservation active par acheteur ; nouvel essai de « réserver » pour le même acheteur et le même article : réservation existante renvoyée, même échéance, une seule commande (M-RES-01) ; modification refusée pendant la réservation ; montants et commission au centime (ENF-01) ;
  - tests négatifs de service à service (ES-38) : appel « réserver » sans jeton de service, avec le jeton d'un autre service ou d'un utilisateur, avec une mauvaise audience ou un jeton expiré, avec une identité d'acheteur forgée ;
  - mesures 1 à 3 du point 4 de l'avis sécurité sur le complément ;
  - tir de charge de démonstration sur la consultation (décision sur le complément) ; la double réservation est vérifiée par le test de concurrence, pas par un tir de charge.
- **D14** : l'expérimentation du canal couvre la diffusion et les identités par service ; celle du serveur d'autorisation inclut l'émission et la vérification d'un jeton de service, pour lever ce risque avant le sprint 3. La preuve elle-même est dans le POC.
- **PILOTAGE** : § 3.4 amendé pour WS-2 (décision de Martin) ; l'amendement reste à reporter dans [PILOTAGE.md](../PILOTAGE.md). Jalons à revoir pour le sprint 3 (découpage ci-dessus).
- **Sécurité** : révision d'[exigences-securite.md](../securite/exigences-securite.md) pour l'option B et l'option P2 du complément (historique en son § 8) : frontière FC11, menaces propres à B (M-CAN-05, M-CAN-06, M-SYN-01, M-RES-01, M-RES-02), identités de service (ES-38), revalidation sur toute modification (ES-18), statut avec B des mesures compensatoires proposées pour l'option A ; ES-13 et ES-14 partielles dans le POC (commande en attente, réservation atomique) ; parades de l'option P2 et mesures 1 à 3 de l'avis sécurité sur le complément parmi les mesures intégrées au POC, y compris le refus de modifier ou de retirer un article réservé ; le paiement et sa confirmation restent hors POC.
- **Compétences (C9)** : diffusion, idempotence, cohérence différée, authentification de service à service et traces distribuées entrent dans la cartographie des compétences et dans l'action de formation.
- **Backlog** : US du walking skeleton (S1 : WS-2, règle de complétude sans photo) ; US de la mise en vente avec les AC de révision (409), de revalidation sans seuil, et d'ES-07 et ES-08 sur l'écran admin (S2) ; US d'Interactions minimal et US de réservation (S3), avec les AC des parades de l'option P2 (dont le nouvel essai de la réservation et la route « réserver » absente du point d'entrée public) et des mesures 1 à 3 de l'avis sécurité sur le complément.
- **Index** : D02 à mettre à jour dans [INDEX.md](INDEX.md) (statut et lien).

### Découpage indicatif en sprints (option P2)

À arbitrer par Martin dans [PILOTAGE.md](../PILOTAGE.md) (jalons J4 à J6).

| Sprint | Contenu | Charge et autoscaling |
|---|---|---|
| S1 (J4) | WS-2 : Place de marché (module Annonces) et Contrôle à une règle, canal point à point, contrat v1 et son test ; CI, HTTPS, authentification, allumage et extinction depuis zéro. | Aucune démonstration : pas encore de parcours complet à charger. |
| S2 (J5) | Mise en vente contrôlée complète (D01) : quatre règles, photos, revalidation sans seuil, écran « à vérifier » et décision sur une révision (409), module Comptes minimal, ES-07 et ES-08. | Tir sur la consultation du catalogue ; mise à l'échelle de la Place de marché sur les requêtes et du Contrôle sur les messages en attente ; perte d'une instance de la Place de marché pendant le tir (ENF-11). Le catalogue est complet et le tir ne traverse ni Interactions ni Transactions : rien ne justifie d'attendre S3. |
| S3 (à ajouter avant J6) | Diffusion de « décision de publication » vers Interactions minimal et notification au vendeur ; Transactions réduit à la réservation, appel synchrone authentifié, espace acheteur ; tests de contrat, de concurrence et négatifs de service à service. | Interactions arrêté puis relancé : aucune notification perdue (ENF-12) ; troisième profil de mise à l'échelle (messages en attente d'Interactions) ; tir sur la consultation rejoué en non-régression avec les quatre services. Le tir de démonstration reste sur la consultation : chaque réservation consomme un objet unique ; la double réservation est vérifiée par le test de concurrence. |

**Ordre de coupe** : en S2, celui de D01 (vague de soumissions en charge, confort de l'écran admin, photos multiples) ; en S3, d'abord le confort de l'espace acheteur, puis le troisième profil de mise à l'échelle ; en dernier recours, repli sur l'option P1 (Transactions documenté, appel synchrone validé en bac à sable), assumé et documenté. Le parcours soumission → contrôle → publication → consultation n'est jamais coupé.

## À savoir défendre en soutenance

Questions préparées pour l'option retenue (B, quatre services) et le périmètre du POC (option P2).

1. **« Pourquoi 4 services pour 3 développeurs, et pas un monolithe modulaire ? »**
   - Trois raisons métier, pas une mode :
     - *isoler la finance* : paiement, séquestre et commission ont leur propre service, leurs secrets, leurs données et leurs déploiements. Le service le plus exposé, la Place de marché (contenu public, photos déposées), peut être compromis sans que l'attaquant atteigne la clé du prestataire ;
     - *réagir au marché* : enchères, vente en direct, bot avant-vente et analyse des ventes s'ajoutent comme nouveaux services ou nouveaux abonnés aux événements existants, sans toucher au cœur (ENF-21) ;
     - *mettre à l'échelle par domaine* : la consultation du catalogue, l'achat et la messagerie ont des profils de charge différents, dimensionnés séparément.
   - Quatre gros services, pas dix : un par grand domaine, et un contrôle sans état. Le découpage fin a été écarté (copies du catalogue, compensations entre services, exploitation hors de portée d'une équipe de trois).
   - Le coût est assumé et outillé : diffusion, cohérence différée et authentification de service à service ont chacune leur mécanisme et leur test.
2. **« Comment gérez-vous la cohérence entre la réservation et le catalogue ? »**
   - La disponibilité de l'article appartient à la Place de marché. Transactions la réserve par un seul appel synchrone authentifié, avec une échéance. La réservation est atomique dans les données de la Place de marché : un objet unique ne peut pas être réservé deux fois (EF-23). Un nouvel essai de l'appel renvoie la réservation existante, sans en créer une seconde (M-RES-01).
   - Aucune transaction commune entre services. « Réservé » est visible sans délai. Seul « vendu » arrive par un événement, alors que l'article n'est déjà plus achetable.
   - Pendant la réservation, le vendeur ne peut ni modifier ni retirer l'article, et le prix payé est celui que la Place de marché a renvoyé (ES-13).
   - Sans paiement confirmé avant l'échéance, la réservation tombe d'elle-même, sans message de compensation. Une confirmation tardive, sur un article réservé entre-temps par un autre acheteur, produirait une double vente (M-RES-01) : l'échéance couvre la session de paiement et le délai de livraison des événements pour la rendre rare, et un remboursement tracé la règle (ES-14).
3. **« Comment les services s'authentifient-ils entre eux ? »**
   - Sur le canal : une identité par service et des droits par flux. Seul le Contrôle écrit les résultats ; Interactions ne fait que lire. Un faux « publié » est refusé (M-CAN-01). Chaque message est validé contre son contrat à la réception. Un service ne publie que sur ses propres sujets et ne crée aucun abonnement ; un événement venu d'un autre service peut suspendre, jamais déclencher un mouvement de fonds (ES-38, M-CAN-05, M-CAN-06).
   - Sur l'appel synchrone : jeton de service émis par le serveur d'autorisation, vérifié par la Place de marché (signature, expiration, audience, droit limité à la réservation). L'identité de l'acheteur est propagée et revérifiée, pour qu'un service ne devienne pas un intermédiaire abusé (M-MOD-02).
   - Un outil de fraude externe ne touche jamais le canal : il passe par un adaptateur.
   - Limite connue : dans l'environnement de formation, tous les services partagent la même identité d'exécution. C'est un écart documenté, avec sa remédiation en V1 (avis sécurité § 5).
4. **« Votre POC prouve-t-il vraiment cette architecture ? »**
   - Il déploie les quatre services de l'option B : Place de marché, Contrôle, Interactions minimal, Transactions réduit à la réservation.
   - Il exerce les points clés propres à B : la diffusion d'un événement métier (K1) ; l'authentification de service à service, sur le canal et sur l'appel synchrone (K2) ; l'appel synchrone de réservation (K3) ; la cohérence différée, avec un délai borné vérifié par les tests (K4) ; la structure du périmètre financier (K5, voir la question 7).
   - S'y ajoutent l'échange point à point par contrat versionné, une identité par service avec des tests négatifs, le cloisonnement des données vérifié par la base, le contrôle fermé par défaut et trois profils de mise à l'échelle.
   - Ce qui n'est pas prouvé (prestataire de paiement, confirmation, séquestre, reversements) est documenté et classé dans le plan de remédiation. Le prouver contre un simulateur aurait affaibli la preuve (D01).
5. **« Que se passe-t-il si un service tombe ? »**
   - Contrôle arrêté : les articles soumis restent invisibles (fermé par défaut, ES-31), les demandes attendent dans le canal et sont traitées à son retour (ENF-12).
   - Interactions arrêté : publications et catalogue ne sont pas affectés ; les notifications sont créées à son retour.
   - Transactions arrêté : la réservation est impossible ; consultation et mise en vente ne sont pas affectées.
   - Place de marché indisponible : Transactions refuse la réservation plutôt que de créer une commande sans article réservé. C'est le couplage assumé du seul appel synchrone entre services (architecture § 6, règle 3).
   - Perte d'une instance de la Place de marché pendant le tir de charge : le service continue sur les autres instances (ENF-11).
6. **« Une réservation sans paiement, n'est-ce pas une fonctionnalité incomplète ? »**
   - Comme achat, oui, et c'est assumé : le POC n'encaisse rien. La fonctionnalité métier démontrée (C3) reste la mise en vente contrôlée, complète de bout en bout (D01).
   - La réservation est la première étape de l'achat de la V1, et elle est complète en elle-même : l'acheteur réserve un article publié et voit son montant et son échéance ; les autres le voient « réservé » ; le vendeur ne peut plus le modifier ; la réservation tombe à l'échéance. C'est le code de la V1 : le paiement viendra s'insérer entre la réservation et l'échéance (architecture § 4.4).
   - Elle porte ce que le paiement n'apporterait pas sans prestataire : appel synchrone authentifié entre services, identité de l'acheteur revérifiée, prix lu chez le propriétaire de l'article (ES-13), réservation atomique d'un objet unique prouvée par un test de concurrence (M-PAY-05).
   - Le paiement n'est pas simulé : un faux prestataire aurait prouvé un faux parcours et poussé à accepter des notifications non vérifiées (D01). Il est documenté, avec sa confirmation vérifiée et idempotente (ES-14), et classé dans le plan de remédiation.
7. **« Votre POC prouve-t-il l'isolement financier ? »**
   - En partie seulement, et nous le disons. Il en prouve la **structure** : Transactions est un service à part, avec ses données, son compte d'accès (refusé par la base sur les données des autres) et sa propre identité auprès du serveur d'autorisation. La Place de marché n'accepte « réserver » que de lui, avec un droit limité.
   - Il ne prouve pas l'isolement à l'exécution. Dans l'environnement de formation, tous les services partagent la même identité d'exécution : un service compromis pourrait y atteindre les ressources et les secrets de Transactions. Et le POC n'a ni prestataire ni secret financier à protéger.
   - Remédiation V1 : une identité d'exécution par service, limitée à ses propres ressources ; secrets du prestataire accessibles au seul service Transactions ; notifications du prestataire reçues par une route dédiée (avis sécurité § 5, D08). L'écart est documenté et repris dans l'audit V1.
