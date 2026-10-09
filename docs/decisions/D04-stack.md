# D04 — Stack back et front

| Champ | Valeur |
|---|---|
| Statut | Décidée |
| Date | 2026-10-09 |
| Décideur | Martin |
| Dépend de | D02 |
| Impacte | D05, D06, D07, D08, D09, D10, D12, D13, [architecture-technique.md](../architecture/architecture-technique.md) § 9 et § 10, `docs/processus/competences-formation.md` (à venir), US du backlog |
| Critères grille | C2, C9 ; contribue à C7, C8, C10 |

## Contexte et besoin

La décision porte sur deux points :

1. le langage, et le cas échéant les frameworks, des **quatre services** du POC : Place de marché, Contrôle de publication, Interactions minimal, Transactions réduit à la réservation ([D02](D02-decoupage-applicatif.md), option P2) ;
2. le framework du **front web** : catalogue public, espace vendeur et ses notifications, espace acheteur (réservation), écran admin « à vérifier ».

- **Exigences** :
  - Consigne, § 1.2.1 : un « schéma d'architecture technique intégrant la sécurité et les solutions retenues (base de données, protocole, framework…) » (M8, C2).
  - D02, « Conséquences » : « D04 : une stack commune aux services, sauf justification service par service ; une convention d'API commune (format d'erreur, versionnement) à écrire et à faire respecter ».
  - C9, niveau 5 : « Les compétences de l'équipe sont recensées. Les expertises techniques à acquérir sont identifiées. Au moins une action de formation est proposée. » La stack fixe les compétences dont l'équipe aura besoin.
  - Exigences qui contraignent directement la stack ([exigences.md](../00-cadrage/exigences.md)) : ENF-19 (API documentées et versionnées), ENF-26 (accessibilité, contrôle automatisé en CI dans le POC), ENF-29 (même artefact en local et dans l'environnement managé), ENF-30 et ENF-31 (textes en français et en anglais ; dates, nombres et montants au format de la langue), ES-07 et ES-08 (CSP, neutralisation des sorties, y compris sur l'écran « à vérifier »).
- **Contraintes** :
  - Martin réalise seul. Estimations de D02, établies sans langage : ≈ 23 à 30 jours de développement back pour le POC (15 à 20 pour le socle, 8 à 10 pour l'option P2), à confirmer après D04.
  - AWS Academy Learner Lab, crédits inférieurs à 100 $, environnement allumé puis éteint ; quatre déployables ; mise à l'échelle automatique à démontrer (C5).
  - **Compétences déclarées par Martin** :

    | | Back | Front |
    |---|---|---|
    | Martin, lead developer | Java/Kotlin (Spring), TypeScript (Node), Go, Rust | Angular, Vue ; pas React |
    | Experts de l'équipe | Rust, Go | React + TypeScript, bibliothèque de composants construite sur MUI |

  - **Priorité déclarée par Martin** : rester dans le périmètre des experts de l'équipe.
  - **Tension à traiter** : côté back, le périmètre des experts et la maîtrise de Martin se recouvrent (Go, Rust). Côté front, ils sont disjoints : les experts font du React, Martin maîtrise Angular et Vue. Or Martin défend la stack à l'oral, avec un vocabulaire précis (C10).
  - **Pas d'expérimentation** : « le test d'un langage de programmation » et « l'implémentation d'un composant avec un framework » ne sont pas des expérimentations valides ([contexte-et-grille.md](../00-cadrage/contexte-et-grille.md) § 3.5). D04 se tranche sans bac à sable : elle passe de « Proposée » à « Décidée », sans statut « Candidate ».

### Besoins techniques à couvrir

| Besoin | Source | Où |
|---|---|---|
| API HTTP JSON versionnées, format d'erreur commun | ENF-19, D02 | Place de marché, Interactions, Transactions |
| Vérification des jetons OIDC/JWT : signature par le jeu de clés publié, émetteur, audience, expiration | ES-02, ES-03 | Place de marché, Interactions, Transactions |
| Jeton de service (flux « client credentials ») et propagation du jeton de l'acheteur | ES-38 ; D02, K2 et K3 | Transactions, Place de marché |
| Client du canal : point à point et diffusion, une identité par service | ENF-12, ES-38, D05 | Place de marché, Contrôle, Interactions |
| Base relationnelle : requêtes paramétrées, migrations versionnées, réservation atomique | ES-08, D07 | Place de marché, Interactions, Transactions |
| Stockage objet : photos privées jusqu'à la publication | ES-17, D07 | Place de marché |
| Décodage et ré-encodage d'images non fiables, bornes de taille et de dimensions | ES-17, M-ANN-01, M-ANN-07 | Place de marché |
| Détection des coordonnées à coût linéaire | ES-16, M-DIS-02 | Contrôle ; Interactions en V1 (chat) |
| Montants au centime | ENF-01 | Transactions |
| Validation des entrées et des messages contre leur contrat | ES-08, ES-38 | Tous les services |
| Tests unitaires, d'intégration, de contrat, d'acceptation | C7, D13 | Services et front |
| OpenTelemetry, identifiant de corrélation propagé dans les appels et les messages | ENF-25, D09 | Tous les services |
| Analyse des dépendances, du code et des images | ES-05, D10 | Services et front |
| i18n français/anglais, formats de la langue, accessibilité contrôlée en CI, CSP, neutralisation des sorties | ENF-26, ENF-30, ENF-31, ES-07, ES-08 | Front |

### Critères de choix

Fixés avant la comparaison des options.

| Critère | Ce qui est regardé | Poids |
|---|---|---|
| Périmètre des experts | Relecture de code, appui et maintenance par l'équipe ; impact concret si l'option en sort | Prioritaire (déclaré par Martin) |
| Adéquation aux besoins | Couverture du tableau ci-dessus par des bibliothèques maintenues ; garanties de sécurité du langage | Éliminatoire si un besoin reste sans réponse |
| Maîtrise par Martin | Productivité immédiate ; vocabulaire précis à l'oral (C10) | Fort |
| Productivité solo | Temps de compilation, outillage, nombre de pièces à assembler, boucle de test | Fort |
| Homogénéité | Nombre de langages et de chaînes d'outils pour quatre services | Fort (D02 : « stack commune, sauf justification ») |
| Recrutement et C9 | Profils disponibles, montée en compétence d'une embauche, réalisme de la cartographie | Moyen |
| Empreinte conteneur et démarrage | Taille d'image, mémoire, démarrage : atout pour l'autoscaling et le coût | Moyen : le délai d'ajout d'une instance dépend surtout de la plateforme |

## Options back

### Couverture des besoins par écosystème

Bibliothèques citées pour vérifier qu'aucun besoin n'est orphelin. Elles sont indicatives ; les versions sont figées au sprint 1.

| Besoin | Go | Rust | Kotlin/Java, Spring Boot | TypeScript, Node.js |
|---|---|---|---|---|
| Serveur HTTP | Bibliothèque standard (`net/http`, routage par méthode et motif) | `axum` | Spring Web | Fastify, NestJS |
| Jetons OIDC/JWT | `go-oidc` ou `jwx` ; `x/oauth2/clientcredentials` | `jsonwebtoken`, `openidconnect`, `oauth2` ; cache du jeu de clés à assembler | Spring Security : serveur de ressources, client OAuth2 | `jose`, `openid-client` |
| Canal, candidats de D05 | SDK AWS v2 (SQS, SNS), `amqp091-go`, `nats.go`, `franz-go` | SDK AWS, `lapin`, `async-nats`, `rdkafka` (liaison C) | Spring Cloud AWS, Spring AMQP, Spring Kafka | SDK AWS v3, `amqplib`, `nats`, `kafkajs` |
| Base et migrations | `pgx` ; `goose` ou `golang-migrate` | `sqlx` : migrations intégrées, requêtes vérifiées à la compilation | Spring Data ; Flyway ou Liquibase | `pg`, Kysely ou Drizzle, outil de migration |
| Images non fiables | Bibliothèque standard, décodeurs sans code C ; dimensions lues avant décodage | `image`, décodeurs sans code C, limites configurables | ImageIO | `sharp` (libvips, code C natif) |
| Expressions régulières | `regexp` : temps linéaire garanti | `regex` : temps linéaire garanti | `java.util.regex` : retour arrière (ReDoS) ; RE2/J en bibliothèque | Moteur V8 à retour arrière par défaut |
| Tests | `go test`, `httptest`, testcontainers-go | `cargo test`, testcontainers | JUnit 5, Testcontainers | Vitest ou Jest, testcontainers |
| OpenTelemetry | SDK Go : traces et métriques stables | SDK Rust plus jeune : statut à vérifier par signal | Instrumentation automatique de la JVM, la plus complète | SDK et instrumentation automatique |
| Analyse des dépendances | `govulncheck`, outil officiel : par défaut, ne signale que les vulnérabilités dont le code est appelé | `cargo audit`, `cargo deny` | Analyseurs Maven ou Gradle | `npm audit`, OSV-Scanner |

### Option Back-A — Go pour les quatre services

- **Description** :
  - Un seul langage, Go, pour les quatre services.
  - Aucun framework web : la bibliothèque standard couvre le serveur HTTP, le JSON et les journaux structurés (`log/slog`). SQL paramétré écrit à la main, sans ORM.
  - Chaque service est un binaire statique dans une image minimale, sans shell, exécutée sans privilège root. Configuration par variables d'environnement (ENF-29).
  - Le Contrôle est une fonction pure, testée par tables de cas.
- **Avantages** :
  - Dans le périmètre des experts et maîtrisé par Martin : aucune tension.
  - Tous les besoins couverts par la bibliothèque standard ou par des bibliothèques officielles ou très répandues. Peu de dépendances, donc peu d'alertes à traiter (ES-05).
  - Sécurité utile au POC :
    - langage à mémoire gérée ;
    - expressions régulières à temps linéaire : avec une longueur bornée, elles réalisent par construction la parade de M-DIS-02 ;
    - décodeurs d'images sans code C (M-ANN-01), dimensions lues avant le décodage complet (M-ANN-07) ;
    - détecteur d'accès concurrents non synchronisés intégré aux tests (`-race`), utile quand serveur HTTP et consommateur du canal partagent un processus.
  - Compilation en secondes : boucle de développement et CI rapides pour quatre services.
  - Langage volontairement petit : revue facile, montée en compétence rapide d'une embauche.
  - Images de quelques dizaines de Mo, démarrage en millisecondes (ordre de grandeur, à mesurer au sprint 1).
- **Inconvénients** :
  - Pas de types somme : le compilateur ne vérifie pas que chaque état de l'article et chaque issue du contrôle sont traités. Parade : linter d'exhaustivité en CI, tests de la machine à états.
  - Le décodeur JSON standard accepte une clé sans tenir compte de la casse, et des clés dupliquées. La validation d'un message contre son contrat (ES-38) ne peut donc pas reposer sur le seul décodage. Parade : validation par schéma avant décodage (D05).
  - Pointeur nul possible à l'exécution ; gestion des erreurs verbeuse.
- **Coût AWS et effort** :
  - Coût d'exécution minimal. Si l'hébergement retenu par D03 facture chaque conteneur selon le CPU et la mémoire réservés, chaque service tient dans la plus petite taille.
  - Effort : référence des estimations de D02, sans surcoût d'apprentissage.
- **Complexité ajoutée** : aucune au-delà du besoin. Un langage, une chaîne d'outils, un modèle d'image et de pipeline pour quatre services.
- **Couverture grille** :
  - **C2** : solutions homogènes ; la conformité des quatre services se vérifie sur un même modèle (M11).
  - **C7** : unitaires, intégration (conteneurs de test), contrat et acceptation avec l'outillage standard ; couverture mesurée nativement (`go test -cover`).
  - **C9** : compétence partagée par Martin et les experts, donc pas de dépendance à une seule personne. Profils « développeur back Go » réalistes.
  - **C10** : vocabulaire maîtrisé par Martin : goroutines, `context` pour les délais et l'annulation, interfaces implicites, erreurs comme valeurs, binaire statique.

### Option Back-B — Rust pour les quatre services

- **Description** : un seul langage, Rust : `axum` sur le moteur asynchrone `tokio`, `sqlx` pour la base. Binaires statiques, images minimales.
- **Avantages** :
  - Dans le périmètre des experts et maîtrisé par Martin.
  - Garanties les plus fortes à la compilation : pas de course de données, pas de nul, `match` exhaustif sur les énumérations (états de l'article, issues du contrôle), requêtes SQL vérifiées.
  - Désérialisation stricte : champ dupliqué refusé, casse respectée, champs inconnus refusables (ES-38).
  - Empreinte mémoire la plus faible, sans ramasse-miettes. Mêmes atouts que Go sur les expressions régulières et les décodeurs d'images.
- **Inconvénients** :
  - Compilation longue, en particulier avec le SDK AWS : CI plus lente sur quatre services, cache de compilation à mettre en place.
  - Davantage d'assemblage : cache du jeu de clés JWT, intégration OpenTelemetry moins mûre. La vérification des requêtes à la compilation exige une base, ou un mode hors ligne, dans la CI.
  - Programmation asynchrone (durées de vie, `Send`, `Sync`) : productivité solo plus faible, revue plus exigeante.
  - Profils rares à recruter ; montée en compétence longue pour une embauche.
  - **Ses gains ne répondent à aucun besoin présent.** La consultation du catalogue est limitée par la base et le réseau, pas par le CPU. Et la mémoire économisée ne réduit pas la facture quand chaque service tient déjà dans la plus petite taille.
- **Coût AWS et effort** : coût d'exécution identique à Back-A. Effort back supérieur : de l'ordre de +15 à 30 % (estimation de l'architecture, non mesurée).
- **Complexité ajoutée** : modèle asynchrone, réglages de compilation (cache, mode hors ligne de `sqlx`), sans besoin présent qui les justifie.
- **Couverture grille** :
  - **C2, C7** : équivalents à Back-A.
  - **C9** : moins bon : profils rares, montée en compétence longue.
  - **C10** : vocabulaire maîtrisé, mais le choix appelle la question « quel besoin de performance ? », sans réponse mesurée.

### Option Back-C — Go et Rust, répartis par service

- **Description** : Go pour trois services, Rust pour un service choisi selon un critère explicite. Deux critères possibles :
  - *calcul pur* : Rust pour le Contrôle, fonction pure sans base ;
  - *exposition* : Rust pour la Place de marché, qui reçoit le contenu public et décode les photos.
- **Avantages** :
  - Reste dans le périmètre des experts.
  - Illustre qu'un contrat versionné rend les services indépendants de leur langage.
- **Inconvénients** :
  - **Aucun gain mesurable.** Les règles du contrôle s'exécutent en quelques microsecondes dans les deux langages. Les deux ont la mémoire gérée, des expressions régulières linéaires, des décodeurs d'images sans code C. Aucun des deux critères ne nomme un problème que Go ne traite pas.
  - Deux chaînes d'outils à tenir : linters, analyse des dépendances, modèles d'image, intégration OpenTelemetry, clients du canal et de la base, conventions de code.
  - Règles de détection des coordonnées (ES-16) : en V1, le Contrôle (annonces) et Interactions (chat) les appliquent tous deux. Avec deux langages, deux implémentations à garder identiques, au risque qu'un contournement bloqué dans les annonces passe dans le chat (M-CHA-01).
  - Chaque développeur back doit maîtriser les deux langages pour relire et dépanner tous les services.
  - L'indépendance vis-à-vis du langage se prouve déjà par le test de contrat, qu'un outil de substitution doit passer (ENF-17). Elle ne demande pas un second langage.
- **Coût AWS et effort** : coût d'exécution identique. Mise en place d'une seconde chaîne, de l'ordre de 2 à 3 jours, puis un surcoût de maintenance permanent.
- **Complexité ajoutée** : la plus élevée des options du périmètre, sans besoin présent : **complexité gratuite à ce jour**. D02 exige une justification service par service ; aucune ne tient.
- **Couverture grille** :
  - **C2** : aucun gain.
  - **C7** : deux outillages de test.
  - **C9** : double compétence exigée de chaque développeur back.
  - **C10** : polyglottisme difficile à justifier devant un jury.

### Option Back-D — Kotlin ou Java avec Spring Boot (hors périmètre, pour comparaison)

- **Description** : Spring Boot pour les quatre services, en Kotlin ou en Java, sur la JVM.
- **Avantages** :
  - Maîtrisé par Martin.
  - L'écosystème le plus intégré pour ces besoins : serveur de ressources et client OAuth2 de Spring Security, intégrations du canal et de la base, migrations, Testcontainers, tests de contrat, instrumentation automatique OpenTelemetry, validation déclarative.
  - Profils abondants sur le marché.
- **Inconvénients** :
  - **Hors du périmètre des experts** : ni relecture ni appui sur le back ; Martin seul garant de quatre services.
  - Empreinte : images de l'ordre de 200 Mo et plus, mémoire de la JVM, démarrage de plusieurs secondes, plus long sur une petite allocation de CPU. La compilation native (GraalVM) réduit le démarrage, au prix de constructions longues et d'une configuration supplémentaire.
  - Moteur d'expressions régulières à retour arrière : la détection des coordonnées demande une bibliothèque linéaire ou des motifs vérifiés (M-DIS-02).
  - Plus de composants dans l'image (environnement d'exécution, couche système) : plus d'alertes à trier dans les scans d'images (ES-05).
- **Coût AWS et effort** : mémoire réservée probablement doublée par service si l'hébergement facture la mémoire réservée. Effort de code comparable à Go pour Martin.
- **Complexité ajoutée** : configuration automatique du framework ; réglage de la JVM ou de la compilation native.
- **Couverture grille** :
  - **C2, C7** : très bien outillés.
  - **C9** : expertise de l'équipe inemployée ; formation de l'équipe au framework.
  - **C10** : bien défendu par Martin, mais « pourquoi une stack que l'équipe ne connaît pas ? » reste sans bonne réponse.

### Option Back-E — TypeScript avec Node.js (hors périmètre back, pour comparaison)

- **Description** : Node.js et TypeScript pour les quatre services, avec un framework léger (Fastify). Un seul langage pour tout le dépôt, front compris.
- **Avantages** :
  - Maîtrisé par Martin ; langage connu des experts front, donc relecture partielle possible.
  - Un seul langage et un seul système de types pour le front et le back.
  - Écosystème complet ; démarrage rapide.
- **Inconvénients** :
  - **Hors du périmètre des experts back** : le modèle d'exécution (boucle d'événements), l'exploitation et les bibliothèques serveur ne sont pas couverts.
  - Décodage des images par `sharp`, bibliothèque C native : surface d'attaque en code non sûr pour la mémoire (M-ANN-01). L'alternative en JavaScript pur est lente et bloque la boucle d'événements.
  - Moteur d'expressions régulières à retour arrière par défaut (M-DIS-02).
  - Arbres de dépendances npm profonds : plus d'alertes, risque de chaîne d'approvisionnement plus élevé (ES-05).
  - Nombres en virgule flottante : les centimes restent exacts en entiers, mais une division mal placée introduit une erreur (ENF-01).
- **Coût AWS et effort** : coût d'exécution proche de Go ; images plus lourdes (environnement d'exécution et dépendances). Effort comparable.
- **Complexité ajoutée** : gestion des dépendances ; travail CPU à sortir de la boucle d'événements.
- **Couverture grille** :
  - **C2, C7** : correctement outillés.
  - **C9** : profils « full stack TypeScript » abondants, mais expertise back de l'équipe inemployée.
  - **C10** : défendable, sans réponse à « pourquoi pas la stack back de l'équipe ? ».

### Synthèse back

| Critère | Back-A Go | Back-B Rust | Back-C Go + Rust | Back-D Spring | Back-E Node |
|---|---|---|---|---|---|
| Périmètre des experts (prioritaire) | Oui | Oui | Oui | Non | Non, langage du front seulement |
| Adéquation aux besoins | Complète, peu de dépendances | Complète, plus d'assemblage | Complète, en double | Complète, la plus intégrée | Complète ; images en code C, regex à retour arrière |
| Maîtrise par Martin | Oui | Oui | Oui | Oui | Oui |
| Productivité solo | Forte | Moyenne | Moyenne | Forte en code, images lourdes | Forte |
| Homogénéité | Un langage | Un langage | Deux langages | Un langage | Un langage pour tout le dépôt |
| Recrutement et C9 | Profils courants, apprentissage rapide | Profils rares | Double compétence | Profils abondants, équipe inemployée | Profils abondants, équipe back inemployée |
| Empreinte et démarrage | Très faible | Très faible | Très faible | Élevée | Moyenne |
| Complexité ajoutée | Aucune | Asynchrone, compilation | Seconde chaîne, gratuite | Framework, JVM | Dépendances, code natif |

## Options front

### Points communs aux trois options

Ces points ne départagent pas les options :

- **Application monopage servie en fichiers statiques**, construite par Vite ou par la CLI Angular. Aucun serveur front à exploiter. Le rendu côté serveur n'a aucun besoin présent : pas d'exigence de référencement dans le POC.
- **Configuration lue à l'exécution** : adresse des API et émetteur OIDC dans un fichier servi par l'environnement, et non figés au build. Un seul artefact pour tous les environnements (ENF-29).
- **Contrôle d'accessibilité en CI** (ENF-26) : axe-core exécuté par Playwright sur le DOM rendu de chaque écran, dans les deux langues. L'outil est indépendant du framework. Il ne détecte qu'une partie des critères WCAG, d'où la vérification manuelle prévue par ENF-26.
- **CSP des scripts** : le build de production ne produit que des scripts en fichiers externes, compatibles avec `script-src 'self'`. À vérifier au sprint 1 par la CSP elle-même, active pendant les tests.
- **CSP des styles** : MUI (Emotion) et Angular injectent des balises `<style>` à l'exécution ; Vue extrait les styles de ses composants au build, mais une bibliothèque de composants à thème dynamique peut en injecter. La CSP des styles demande alors `'unsafe-inline'` ou un nonce. ES-08 vise les scripts : l'écart sur les styles est à faire valider par la sécurité.
- **Formats de la langue** (ENF-31) : API `Intl` du navigateur pour les dates, les nombres et les montants avec leur devise.
- **Connexion OIDC** : bibliothèque cliente indépendante du framework, flux « Authorization Code » avec PKCE. Le mode de conservation des jetons relève de D06.

### Option Front-A — React, TypeScript et MUI

- **Description** : React, TypeScript strict, MUI, Vite, React Router. Appels aux API dans des hooks, composants de présentation séparés. Aucune bibliothèque d'état global : l'état utile est celui du serveur, sur cinq écrans. MUI est utilisée directement : une bibliothèque de composants partagée n'a pas de besoin présent avec un seul front.
- **i18n** : react-i18next (fichiers de traduction, changement de langue sans rechargement) ; `Intl` pour les formats ; textes internes des composants MUI traduits par ses objets de langue français et anglais.
- **Accessibilité** : composants MUI qui suivent les motifs WAI-ARIA ; règles de lint `jsx-a11y` ; axe-core en CI.
- **Sécurité** : tout texte inséré en JSX est échappé par défaut. La seule échappatoire, `dangerouslySetInnerHTML`, est interdite par une règle de lint bloquante (`react/no-danger`) : aucun HTML libre (ES-08). Preuve : un titre contenant du balisage s'affiche comme du texte, au catalogue et chez l'admin (AC de D02).
- **Tests** : Vitest et Testing Library pour les composants et les hooks ; Playwright pour les parcours et l'accessibilité.
- **Avantages** :
  - Dans le périmètre des experts : relecture, appui, maintenance en V1.
  - MUI déjà pratiquée par l'équipe ; séparation logique et présentation par hooks, pratique de l'équipe.
  - L'écosystème front le plus large : bibliothèques, documentation, profils.
- **Inconvénients** :
  - **Martin ne maîtrise pas React.** Montée en compétence nécessaire pour coder seul et pour défendre à l'oral : modèle de rendu, état et props, hooks et leurs dépendances, effets, composants contrôlés, contexte.
  - Routage, i18n et OIDC sont des bibliothèques à assembler, là où Angular les intègre.
  - Styles injectés par Emotion (voir les points communs).
- **Coût AWS et effort** : hébergement statique, coût négligeable, identique pour les trois options. Montée en compétence de Martin de l'ordre de 2 à 4 jours (estimation de l'architecture), réduite par sa pratique de Vue, dont l'API de composition est proche des hooks.
- **Complexité ajoutée** : aucune au-delà du besoin, si les garde-fous sont tenus : ni rendu serveur, ni état global.
- **Couverture grille** :
  - **C2** : conforme à l'architecture ; front statique.
  - **C7** : unitaires et parcours outillés ; accessibilité en CI.
  - **C8** : échappement par défaut, échappatoire interdite en CI.
  - **C9** : expertise de l'équipe employée ; l'écart du lead devient une compétence à acquérir, identifiée.
  - **C10** : vocabulaire React à acquérir et à répéter avant la soutenance.

### Option Front-B — Angular

- **Description** : Angular, TypeScript strict, Angular Material et CDK.
- **i18n** : intégrée (`@angular/localize`), avec une construction par langue et un changement de langue par rechargement ; ou bibliothèque de traduction à l'exécution. Formats par les pipes intégrés.
- **Accessibilité** : CDK (gestion du focus, annonces aux lecteurs d'écran) et Angular Material ; règles d'accessibilité des gabarits dans angular-eslint ; axe-core en CI.
- **Sécurité** : la plus complète par défaut. Échappement des interpolations, **assainissement automatique** des liaisons HTML, URL et style, prise en charge de Trusted Types, protection CSRF intégrée au client HTTP (utile si D06 retient un cookie de session).
- **Tests** : runner de tests de la CLI ; Playwright.
- **Avantages** :
  - Maîtrisé par Martin : productivité immédiate, défense orale assurée.
  - Framework complet : routage, formulaires, client HTTP et i18n intégrés ; peu de bibliothèques tierces à choisir et à défendre.
  - Garanties de sécurité par défaut les plus fortes des trois.
- **Inconvénients** :
  - **Hors du périmètre des experts** : aucune relecture experte du front.
  - En V1, l'équipe doit apprendre le framework le plus riche en concepts : injection de dépendances, RxJS, signaux.
  - Changement de langue par rechargement avec l'i18n intégrée.
  - La pratique de MUI de l'équipe n'est pas réutilisée.
- **Coût AWS et effort** : hébergement statique. Aucun surcoût pour Martin ; coût reporté sur l'équipe (formation au framework).
- **Complexité ajoutée** : framework plus lourd, mais moins de dépendances tierces.
- **Couverture grille** :
  - **C8** : la meilleure par défaut.
  - **C9** : formation de l'équipe à Angular ; expertise React inemployée.
  - **C10** : très bien défendu.

### Option Front-C — Vue

- **Description** : Vue 3, TypeScript, Vite, vue-i18n, une bibliothèque de composants à choisir.
- **i18n** : vue-i18n, changement de langue à l'exécution ; `Intl` pour les formats.
- **Accessibilité** : dépend de la bibliothèque de composants retenue ; règles `vuejs-accessibility` ; axe-core en CI.
- **Sécurité** : échappement par défaut ; `v-html` interdit par une règle de lint (`vue/no-v-html`).
- **Tests** : Vitest et Vue Test Utils ; Playwright.
- **Avantages** :
  - Maîtrisé par Martin.
  - Framework léger, proche de React par son API de composition : apprentissage plus court qu'Angular pour les experts.
- **Inconvénients** :
  - **Hors du périmètre des experts.**
  - Bibliothèque de composants à choisir hors de MUI ; sa qualité d'accessibilité est à vérifier.
  - Écosystème et vivier de profils plus restreints que React.
- **Coût AWS et effort** : identiques à Front-B.
- **Complexité ajoutée** : faible ; un choix de bibliothèque de composants de plus.
- **Couverture grille** :
  - **C8** : équivalente à React.
  - **C9** : expertise React inemployée ; formation de l'équipe plus légère qu'avec Angular.
  - **C10** : bien défendu.

### Synthèse front

| Critère | Front-A React | Front-B Angular | Front-C Vue |
|---|---|---|---|
| Périmètre des experts (prioritaire) | Oui, avec MUI | Non | Non |
| Maîtrise par Martin | Non : à acquérir | Oui | Oui |
| i18n (ENF-30, ENF-31) | Bibliothèque et `Intl` | Intégrée, une construction par langue | Bibliothèque et `Intl` |
| Accessibilité (ENF-26) | MUI, lint, axe-core | CDK et Material, lint, axe-core | Selon la bibliothèque, lint, axe-core |
| Sécurité par défaut (ES-08) | Échappement ; échappatoire interdite par lint | Échappement et assainissement, Trusted Types, CSRF | Échappement ; échappatoire interdite par lint |
| Tests | Vitest, Testing Library, Playwright | CLI, Playwright | Vitest, Vue Test Utils, Playwright |
| Productivité solo | Moyenne au départ | Forte | Forte |
| Défense orale (C10) | Vocabulaire à acquérir | Assurée | Assurée |
| Recrutement et C9 | Vivier le plus large ; expertise employée | Vivier large ; formation de l'équipe | Vivier plus restreint ; formation de l'équipe |

## Recommandation

**Back-A (Go pour les quatre services) avec Front-A (React, TypeScript et MUI, en application statique).**

### Back : Go

1. **Périmètre et maîtrise** : Go est dans le périmètre des experts et maîtrisé par Martin. Aucune tension.
2. **Simplicité** : le plus petit nombre de pièces. Pas de framework web ni d'ORM, la bibliothèque standard suffit ; peu de dépendances à suivre.
3. **Sécurité** : mémoire gérée, expressions régulières linéaires (M-DIS-02), décodeurs d'images sans code C (M-ANN-01), analyse des vulnérabilités ciblée sur le code appelé (ES-05).
4. **Productivité solo** : compilation en secondes, une seule chaîne d'outils pour quatre services.
5. **Homogénéité** : un langage, comme le demande D02. Les règles de détection des coordonnées restent un seul code pour les annonces et, en V1, pour le chat.
6. **Rust** est aussi dans le périmètre, mais ses gains (performance, mémoire, garanties de compilation) ne répondent à aucun besoin présent, pour un coût certain en compilation, en productivité et en recrutement. **Back-C** est de la complexité gratuite. **Back-D et Back-E** sortent du périmètre des experts, priorité de Martin.

### Front : React

1. **Priorité de Martin** : React et MUI sont le périmètre des experts. Le code du front aura des relecteurs et des mainteneurs en V1.
2. **Les critères techniques départagent peu** : i18n, accessibilité en CI (axe-core est indépendant du framework), CSP et tests s'obtiennent dans les trois options. Angular est plus fort par défaut en sécurité, mais le POC n'affiche aucun HTML saisi : l'échappement par défaut de React et l'interdiction de son échappatoire en CI couvrent ES-08.
3. **La tension est traitée, pas ignorée** :
   - l'écart de Martin sur React est inscrit comme compétence à acquérir dans la cartographie (C9) ;
   - la montée en compétence a lieu pendant le sprint 1, surtout back et infrastructure, avant les écrans du sprint 2 ;
   - le code du front est relu par un expert ;
   - le vocabulaire à maîtriser pour l'oral est listé dans « À savoir défendre » (question 2).
4. **Repli** si Martin fait primer sa propre maîtrise sur le périmètre des experts : **Angular plutôt que Vue**. Sans relecture experte, mieux vaut le framework dont les garanties par défaut sont les plus fortes et qui demande le moins de bibliothèques tierces.

### Stack proposée

| Couche | Choix proposé | Remarque |
|---|---|---|
| Langage back | Go, dernière version stable au sprint 1, figée dans le dépôt | Quatre services |
| HTTP, JSON, journaux | Bibliothèque standard : `net/http`, `encoding/json`, `log/slog` | Aucun framework web |
| Jetons | `go-oidc` ou `jwx` ; `golang.org/x/oauth2/clientcredentials` | Détail en D06 |
| Base | Pilote du SGBD de D07 ; SQL paramétré écrit à la main ; outil de migrations versionnées | Pas d'ORM |
| Canal | Client officiel ou activement maintenu du canal de D05 | — |
| Tests back | `go test` (tables de cas, `-race`), testcontainers-go | Contrat : D13 |
| Qualité et sécurité back | golangci-lint (dont gosec et le linter d'exhaustivité), govulncheck | Portes : D10 |
| Images | Binaire statique, image minimale sans shell, utilisateur non root | Configuration par variables d'environnement |
| Front | React, TypeScript strict, MUI, Vite, React Router | Application statique |
| i18n | react-i18next ; `Intl` pour les formats | Français par défaut, anglais |
| Tests front | Vitest, Testing Library ; Playwright avec axe-core | Parcours et accessibilité dans les deux langues |
| Lint front | typescript-eslint, `jsx-a11y`, `react/no-danger` bloquante | — |

### Garde-fous de simplicité

- **Pas de module partagé entre services dans le POC** : chaque service possède ses types de messages ; le test de contrat garantit la compatibilité. Un module partagé n'apparaît qu'avec un second utilisateur réel.
- **Pas de rendu serveur, pas de bibliothèque d'état global** côté front ; des appels `fetch` dans des hooks suffisent pour cinq écrans.
- **Règle d'exception pour un second langage back** : il n'entre que pour un besoin mesuré (latence, mémoire, CPU) que Go ne tient pas, constaté par une mesure, et par une nouvelle décision.

### Convention d'API commune (demandée par D02)

- Préfixe de version `/v1` ; compatibilité ascendante au sein d'une version (ENF-19).
- Erreurs au format « Problem Details » (RFC 9457, `application/problem+json`), sans trace technique (ES-07).
- Une description OpenAPI par service, dans le dépôt, vérifiée en CI (D10).
- Codes déjà fixés par D02 : 202 à la soumission, 404 plutôt que 403 pour une ressource d'autrui ou non publiée, 409 sur une révision périmée.

### Point ouvert de l'architecture traité ici

[architecture-technique.md](../architecture/architecture-technique.md) § 10 renvoie à D04 le partage des règles de détection des coordonnées entre le Contrôle (annonces) et Interactions (chat, V1).
Proposition : un paquet Go de fonctions pures, sans dépendance, propriété du Contrôle dans le POC. En V1, Interactions l'importe depuis le monorepo au lieu de le réécrire. Rien n'est extrait dans le POC : Interactions minimal n'a pas de chat.

### Verdict simplicité

- **Solution la plus simple pour le besoin** : un seul langage back, sans framework web ; un front statique sur la stack des experts.
- **Écarts de complexité justifiés** : aucun n'est ajouté par la recommandation.
- **Complexité écartée** : second langage back (Back-C), modèle asynchrone de Rust (Back-B), JVM (Back-D), rendu serveur, état global, module partagé prématuré.

## Avis sécurité

**Synthèse.** Avis favorable à Back-A (Go) et à Front-A (React, TypeScript, MUI), sous les conditions ci-dessous, à reporter dans D06, D08 et D10. Aucune option écartée n'apporte un gain de sécurité qui justifierait de sortir du périmètre des experts.

**1. Back**
- **Mémoire** : Go et Rust excluent la corruption mémoire hors `unsafe` et hors code C lié ; la JVM et Node aussi, mais leur traitement d'image repose en partie sur du code C. Nuance Go : une course de données sur une interface, une tranche ou une chaîne peut corrompre la mémoire ; `-race` ne voit que les courses exercées par les tests.
- **Chaîne d'approvisionnement (ES-05, M-LIV-01)** : Go a l'arbre le plus court (quelques dizaines de modules par service, à mesurer au sprint 1 par la nomenclature logicielle), des sommes de contrôle publiques (`go.sum` et base de sommes) et n'exécute aucun code tiers au téléchargement ni à la construction. Rust : plusieurs centaines de crates, scripts `build.rs` et macros procédurales exécutés à la compilation. Node : arbre npm profond et scripts d'installation, vecteur des compromissions npm de 2025. Spring : arbre Maven large et image JVM avec couche système.
- **govulncheck** : l'analyse d'atteignabilité réduit le bruit, mais la base de vulnérabilités Go ne publie pas de gravité ; règle pour Go : **toute vulnérabilité atteignable bloque** ; le scan de l'image, noté en CVSS, couvre le reste. Les failles de la bibliothèque standard se corrigent par une version mineure de Go, figée par la directive `toolchain`.
- **Images (M-ANN-01, ES-17)** : en Go, l'absence de code C tient à `CGO_ENABLED=0`, vérifié en CI. Séquence : `http.MaxBytesReader`, `image.DecodeConfig`, format comparé à une liste blanche, borne en pixels, décodage, ré-encodage JPEG ou PNG (métadonnées supprimées par construction) ; SVG refusé d'office. Risque résiduel sans isolement : déni de service, pas exécution de code. Autres options : Rust, crate `image` sans fonctionnalités par défaut, `Limits` posées ; Spring, lecteur JPEG d'ImageIO natif ; Node, `sharp` embarque libvips et libwebp (CVE-2023-4863, exploitée).
- **Expressions régulières (ES-16, M-DIS-02)** : `regexp` (Go) et `regex` (Rust) linéaires par construction ; interdire en CI les moteurs à retour arrière (`regexp2`, `fancy-regex`). La borne de longueur d'ES-16 s'applique avant la normalisation des variantes masquées.
- **Jetons (ES-02, ES-38)** : `go-oidc` v3 ou `jwx` v2 et plus ; algorithmes asymétriques fixés (ni `none` ni HMAC), jeu de clés en cache, émetteur, audience, expiration, puis contrôle explicite du type de jeton (service ou utilisateur) et de la portée « réserver ». Options `InsecureSkipSignatureCheck`, `SkipIssuerCheck`, `SkipClientIDCheck`, `SkipExpiryCheck` interdites en CI. Si l'émetteur de D06 place `client_id` au lieu de `aud`, D06 fixe un contrôle équivalent, jamais une désactivation. Cette vérification a trois utilisateurs dès le POC : un paquet commun est justifié ; à défaut, mêmes tests négatifs dans chaque service.
- **Pièges de Go sans framework** : `http.Server` sans délai par défaut : poser `ReadHeaderTimeout`, `ReadTimeout`, `WriteTimeout`, `IdleTimeout`, et un délai sur chaque client sortant (`http.DefaultClient` n'en a pas). JSON : `DisallowUnknownFields` et schéma ; `encoding/json/v2` (refuse doublons et casse libre) à retenir s'il est stabilisé au sprint 1.

**2. Front**
- **Neutralisation (ES-08)** : l'échappement JSX suffit si les échappatoires sont fermées par un lint bloquant : `react/no-danger`, `react/jsx-no-script-url`, `no-unsanitized/property` et `no-unsanitized/method`, `no-eval`, `no-implied-eval`, `no-new-func`. Aucune donnée utilisateur dans `sx` ni `style` ; aucun lien saisi rendu en `href` (ES-16).
- **CSP, tranché : `'unsafe-inline'` sur `style-src` seulement, pas de nonce dans le POC.** Un nonce doit changer à chaque réponse, ce qui exige un composant qui réécrit `index.html` ; un nonce figé ne protège rien ; une empreinte est impossible sur des styles générés. Politique posée en en-tête HTTP (pas en balise `meta`, qui ignore `frame-ancestors`) : `default-src 'none'; script-src 'self'; style-src 'self' 'unsafe-inline'; img-src 'self' <origine des photos>; font-src 'self'; connect-src 'self' <API> <émetteur OIDC>; base-uri 'none'; form-action 'self'; frame-ancestors 'none'; object-src 'none'`. Polices servies par l'origine ; `build.assetsInlineLimit: 0` dans Vite.
- **Effet sur ES-07 et ES-08** : ES-08 est tenue (`script-src 'self'`, sans `'unsafe-inline'` ni `'unsafe-eval'`). Le volet CSP d'ES-07 est tenu avec un risque résiduel faible, à documenter : injecter un style suppose une injection HTML ou une donnée utilisateur dans `sx`, exclues ci-dessus, et l'exfiltration par CSS est bornée par `img-src`, `font-src` et `connect-src`. Le nonce redevient la cible si D06 ajoute un composant serveur. Les tests Playwright échouent à toute violation de CSP.
- **Jetons (ES-02)** : Authorization Code avec PKCE `S256`, client public sans secret, jamais de flux implicite. Jetons **en mémoire seulement** (`InMemoryWebStorage` au lieu du `sessionStorage` par défaut d'`oidc-client-ts`), jamais en `localStorage`. Jeton d'accès court ; rafraîchissement par jeton à rotation en mémoire, ou nouvelle redirection. Code retiré de l'URL ; `Referrer-Policy: no-referrer`. Déconnexion : mémoire vidée, jeton de rafraîchissement révoqué, session close chez l'émetteur. Jeton dans l'en-tête `Authorization` : pas de requête intersite forgée sur l'API ; CORS limité à l'origine du front, sans `credentials`. Variante plus forte (jetons hors de portée du JavaScript) : intermédiaire serveur à cookie `HttpOnly`, à arbitrer en D06.
- **npm** : `npm ci --ignore-scripts` (exceptions listées), versions exactes, délai de quarantaine des nouvelles versions dans l'outil de mise à jour, revue de toute nouvelle dépendance. Serveur de développement Vite jamais exposé. Configuration servie et variables `VITE_*` : valeurs publiques uniquement (ES-06).

**3. CI, quel que soit le choix (ES-05, ES-06, M-LIV-01)**
- **Dépendances, bloquant** : govulncheck (Go), `cargo audit` et `cargo deny` (Rust), `npm audit --omit=dev --audit-level=high` et OSV-Scanner (front) ; installation verrouillée (`go mod verify`, `cargo --locked`, `npm ci`) ; exceptions justifiées et datées dans le dépôt ; rapports conservés.
- **Images** : Trivy ou Grype sur l'image construite, bloquant en critique et élevée corrigeables ; nomenclature logicielle conservée ; image `scratch` ou distroless `nonroot`.
- **Secrets** : gitleaks sur tout l'historique, puis à chaque fusion ; scan des couches d'image ; masquage dans les journaux du pipeline.
- **SAST** : `go vet` et golangci-lint avec gosec ; `clippy -D warnings` et `#![forbid(unsafe_code)]` (Rust) ; ESLint avec les règles ci-dessus ; règles Semgrep pour les interdits de cet avis (options `Skip*` des vérificateurs de jetons, `import "C"`, moteurs de regex à retour arrière, `dangerouslySetInnerHTML`).
- **Configuration et pipeline** : analyse de la configuration décrite en code (Trivy ou Checkov) ; actions et images de CI figées par empreinte (cf. compromission de `tj-actions/changed-files`, 2025).
- **Fin de sprint** : ZAP en mode passif sur l'environnement déployé, qui contrôle aussi les en-têtes et la CSP (ES-05, ES-07).

**4. Verdict.** Go et React avec MUI forment un choix sûr pour le POC, Rust n'y ajoutant aucune garantie décisive et Spring ou Node en retirant (regex à retour arrière, décodage d'image en code C) : avis favorable, sous réserve de `CGO_ENABLED=0`, des délais du serveur HTTP, d'une vérification des jetons sans option de contournement, d'une CSP stricte sur les scripts et de jetons conservés en mémoire.

## Avis faisabilité

### Back

Fonction : Développement back — 2026-10-09. Périmètre : bibliothèques, pièges, organisation du code, effort. Estimations en jours de développement back, hors front, infrastructure et tests ; bibliothèques et versions figées au sprint 1.

**1. Go, Rust ou les deux, pour ce POC**

| Besoin | Go : choix | Piège à traiter | Rust : équivalent et écart |
|---|---|---|---|
| HTTP | `net/http` (routage par méthode et motif), `log/slog` ; `errgroup` pour servir l'API et consommer le canal dans un même processus | Par défaut aucun délai : poser `ReadHeaderTimeout` et borner les corps (`http.MaxBytesReader`). Arrêt propre sur SIGTERM (`signal.NotifyContext`, `Server.Shutdown`), sinon chaque réduction d'instances coupe des requêtes et des messages en cours | `axum`, `tower-http` : mûrs, équivalents |
| Jetons | `jwx` : jeu de clés en cache et rafraîchi, émetteur, audience, expiration, algorithme imposé ; `x/oauth2/clientcredentials` : jeton de service en cache, renouvelé à l'expiration | `go-oidc` vise le jeton d'identité, pas le jeton d'accès. Si D06 retient Cognito, ses jetons d'accès ne portent pas `aud` : audience vérifiée par le `scope`. Le transport de `x/oauth2` réécrit `Authorization` : le jeton de l'acheteur passe dans un second en-tête, convention à fixer en D06 | `jsonwebtoken`, cache des clés à écrire ; `oauth2` ne met pas le jeton en cache |
| Base et migrations | `pgx` v5 et `pgxpool` (si D07 retient PostgreSQL) ; `goose` v3 : migrations SQL embarquées dans le binaire, appliquées avant le déploiement par une sous-commande du même artefact (ENF-29) | Compte de migration (DDL) distinct du compte d'exécution (DML). « Une réservation active par acheteur » ne s'écrit pas en index unique partiel (`now()` interdit) : verrou transactionnel par acheteur ou contrainte d'exclusion, à trancher en D07 | `sqlx` : migrations intégrées, requêtes vérifiées à la compilation, mode hors ligne à tenir en CI |
| Canal | RabbitMQ : `amqp091-go` (officiel), diffusion par un échange `topic` et une file par abonné. NATS : `nats.go` et `jetstream`, un consommateur durable par abonné. Files et consommateurs posés par l'exploitation, jamais déclarés par le code (M-CAN-06). Anti-perte : événement écrit dans la même transaction que la décision, publié par un relais (`FOR UPDATE SKIP LOCKED`), à confirmer en D05 | `amqp091-go` ne se reconnecte pas seul ; confirmations de publication et `Qos` à écrire (sans `Qos`, une réplique du Contrôle accapare la file). JetStream : droits par les sujets `$JS.API.>`, délicats pour interdire la création de consommateurs, à prouver en J3. SQS/SNS : une seule identité sous le rôle imposé | `lapin` : reconnexion à écrire ; `async-nats` : officiel, solide |
| Validation des messages | Types Go propres à chaque service ; validation par JSON Schema avant décodage (`santhosh-tekuri/jsonschema` v6) | Le schéma ne voit pas les clés dupliquées : les refuser demande `encoding/json/v2` (expérimental depuis Go 1.25) ou une passe sur les jetons du décodeur, ≈ 0,5 j | `serde` : doublons refusés, `deny_unknown_fields` |
| Images | `image.DecodeConfig` (dimensions lues avant décodage), `image/jpeg`, `image/png`, `x/image/draw` ; ré-encodage JPEG sans métadonnées : coordonnées GPS supprimées par construction | Orientation EXIF à lire, sinon photo tournée. Plafond de pixels et décodages simultanés bornés (sémaphore). Pas d'encodeur WebP. Lire le dépôt par `MultipartReader`, borné, si le système de fichiers est en lecture seule | `image` avec `Limits` : plus complet d'origine |
| Coordonnées | `regexp` (RE2, temps linéaire garanti) après normalisation (`x/text` : NFKC, suppression des séparateurs) | Ni anticipation ni référence arrière ; champs bornés en longueur ; motifs compilés une fois | `regex` : même garantie |
| Montants | `int64` de centimes sous un type nommé ; commission en points de base | Règle d'arrondi explicite, testée au centime (ENF-01) | Identique |
| Tests | `go test` (tables de cas), `httptest`, `testcontainers-go` (PostgreSQL, RabbitMQ, NATS) ; fuzzing natif sur le dépôt de photos et la détection des coordonnées ; contrats vérifiés contre les schémas de `contracts/` (`kin-openapi`) | `-race` exige cgo dans l'image de CI. La réservation concurrente se prouve contre la vraie base par des appels simultanés. `pact-go` (bibliothèque native) écarté au profit des schémas (D13). Démon Docker requis sur l'exécuteur de CI | `cargo test`, `testcontainers` |
| OpenTelemetry | SDK Go (traces et métriques stables), `otelhttp`, `otelpgx` ; `slog` enrichi de l'identifiant de trace | Pas d'instrumentation officielle des clients du canal : porteur W3C Trace Context à écrire dans les en-têtes des messages, ≈ 0,5 j | Crates en 0.x, ruptures d'API fréquentes |

**Bilan** : Rust ne l'emporte que sur la désérialisation stricte et les limites du décodeur d'images, que Go traite en moins d'une journée. Il coûte ailleurs : asynchrone, cache des clés et du jeton, `sqlx` hors ligne, OpenTelemetry, compilation. Le mélange additionne les deux chaînes sans lever aucun de ces pièges ; les plus durs (canal, jetons) sont indépendants du langage.

**2. Organisation du monorepo**
- **Un module Go par service** : `services/<service>/` avec `go.mod`, `cmd/`, `internal/<module métier>/` et `migrations/` ; dépendances, `govulncheck`, image et pipeline propres à chaque service. `go.work` à la racine pour le poste de développement seulement ; la CI l'ignore.
- **Contrats hors du code** : `contracts/` porte les JSON Schema des messages et les OpenAPI des API, source unique lisible par un outil de substitution (ENF-17). Aucun type de contrat partagé en Go : producteur et consommateur évolueraient ensemble et le test de contrat ne prouverait plus rien.
- **Un seul module partagé, technique** : `libs/platform/` (vérification des jetons, Problem Details, démarrage OpenTelemetry, connexion au canal), sans type métier ni type de contrat. Ce code a trois ou quatre utilisateurs réels dès le POC ; dupliqué, une faille de vérification des jetons se corrigerait trois fois. En V1, le paquet des coordonnées le rejoint dans `libs/`, au second utilisateur.
- **Règle de dépendance vérifiée en CI** : chaque service construit et testé avec `GOWORK=off` ; `depguard` (golangci-lint) interdit l'import d'un autre service et tout import de `services/` depuis `libs/` ; `internal/` appliqué par le compilateur ; dans la Place de marché, Annonces limité à l'interface de Comptes.
- **Déclencheurs par chemin** : `services/<service>/**` déclenche son pipeline ; `libs/**` et `contracts/**` déclenchent les quatre.

**3. Effort solo et risques**

| Option | Effort back du POC | Écart avec Go |
|---|---|---|
| Go | 23 à 30 j (D02), plutôt au haut de la fourchette | Référence |
| Rust | ≈ 28 à 38 j | + 5 à 8 j : asynchrone, cache des clés et du jeton de service, `sqlx` hors ligne, OpenTelemetry instable, CI plus lente |
| Go + Rust | ≈ 27 à 35 j | + 4 à 5 j : seconde chaîne, puis ≈ 0,5 j par sprint ; détection des coordonnées écrite deux fois en V1 |

Risques back, par gravité : (1) client du canal (reconnexion, droits par identité, interdiction de créer un consommateur), à prouver en J3 ; (2) jetons (audience, transport du jeton de l'acheteur), à fixer en D06 avant le sprint 3 ; (3) validation stricte des messages, à acter en D05 ; (4) mémoire et orientation des photos, bornées dès l'US de dépôt. Les risques 1 et 2 se poseraient à l'identique en Rust.

**4. Verdict** : Go pour les quatre services est faisable seul dans l'enveloppe de 23 à 30 jours ; Rust coûterait 5 à 8 jours de plus sans besoin présent qui les justifie, et le mélange cumule les coûts sans gain : avis favorable à Back-A, sous trois réserves : clés dupliquées traitées en D05, un module technique partagé `libs/platform`, contrats portés par des schémas et non par des types Go partagés.

### Front

Fonction : Faisabilité front — 2026-10-09. Écrans du POC de D02 ; connexion OIDC par « Authorization Code » avec PKCE.

**1. Options, bibliothèques et pièges.** Les trois écosystèmes sont mûrs pour ces besoins ; ce qui les distingue, c'est la quantité de bibliothèques à assembler et le fait d'être ou non dans le périmètre des experts.
- **React + TS + MUI** : Vite ; React Router v7 en mode « data » (`createBrowserRouter`) ; `oidc-client-ts` avec `react-oidc-context` ; i18next et react-i18next ; React Hook Form et Zod (champs MUI par `Controller`) ; TanStack Query sur un client typé généré depuis l'OpenAPI de chaque service (`openapi-typescript`, `openapi-fetch`) ; Vitest, Testing Library et MSW ; Playwright avec `@axe-core/playwright`. Bibliothèques très répandues, mais six choix à assembler et à savoir justifier.
- **Angular** : routeur, formulaires réactifs typés, `HttpClient` et intercepteur du jeton intégrés ; `angular-auth-oidc-client` (certifiée OpenID) ; `@angular/localize` ou Transloco ; Vitest par la CLI ; Angular Material et CDK ; Playwright et axe. L'option qui demande le moins de choix tiers.
- **Vue** : Vue Router, `oidc-client-ts`, vue-i18n, VeeValidate et Zod, TanStack Query, Vitest et Vue Test Utils, Playwright et axe. Bibliothèque de composants à choisir (Vuetify), accessibilité à vérifier composant par composant ; assemblage comparable à React, sans l'appui des experts.
- **Pièges React, à fermer au sprint 1** :
  - Les appels réseau écrits à la main dans `useEffect` exposent à des courses entre réponses, à la double exécution en développement (`StrictMode`) et aux nettoyages oubliés : **TanStack Query est recommandé** (cache de l'état du serveur, pas un état global). Il couvre aussi le rafraîchissement des notifications, l'invalidation après soumission, le rechargement après un 409 et le nouvel essai de « réserver ».
  - `import.meta.env` est figé au build : interdit pour l'adresse des API et l'émetteur OIDC (ENF-29). Ces valeurs sont lues dans un `config.json` servi par l'environnement, chargé avant le routeur et le client OIDC.
  - Règles `react-hooks/rules-of-hooks` et `react-hooks/exhaustive-deps` en erreur, TypeScript en `strict`.

**2. Accessibilité (ENF-26, ENF-27) et i18n (ENF-30, ENF-31).**
- MUI implémente les motifs WAI-ARIA des composants complexes et relie libellé, aide et erreur au champ ; objets de langue `frFR` et `enUS`. Angular CDK offre la boîte à outils la plus complète. En Vue, tout dépend de la bibliothèque retenue. Aucune option n'assure seule la conformité.
- Ce qu'aucune option ne fait : titre de page et focus sur le `h1` à chaque navigation (WCAG 2.4.2, 2.4.3) ; attribut `lang` de `<html>` à jour (3.1.1) ; annonce des états asynchrones dans une zone `role="status"` (4.1.3) ; contraste de la palette vérifié (1.4.3). Trois petits hooks partagés suffisent.
- ENF-26/27 à moindre coût : un test Playwright paramétré par écran, par langue, sur « mobile » et « ordinateur » ; axe restreint à `wcag2a`, `wcag2aa`, `wcag21a`, `wcag21aa`, toute violation bloque ; points de rupture du thème ; requêtes Testing Library par rôle et nom accessible ; vérification manuelle clavier et lecteur d'écran (NVDA) une demi-journée par sprint, consignée. Piège : les jetons en mémoire ne se rejouent pas par `storageState` ; les tests réutilisent la session du serveur d'autorisation.
- ENF-30/31 à moindre coût : clés de traduction typées ; règle `i18next/no-literal-string` ; test d'égalité des clés français/anglais ; dates et montants par `Intl` avec la devise renvoyée par l'API, centimes convertis à l'affichage. Piège : espaces insécables de `Intl` en français (` `, ` `) à écrire explicitement dans les assertions.
- Point à aligner : si Interactions renvoie la catégorie de motif plutôt qu'un texte construit, le front la traduit comme le reste de l'interface et toutes les traductions restent à un seul endroit.

**3. Sécurité front.**
- **Neutralisation** : JSX échappe le texte et les attributs ; `react/no-danger`, `react/jsx-no-script-url`, toute écriture de `innerHTML` et `eval` en erreur au lint ; aucune URL construite à partir d'un contenu saisi. Preuve de D02 : un titre `<img src=x onerror=alert(1)>` s'affiche comme du texte au catalogue et sur l'écran « à vérifier ».
- **CSP et Emotion** : un nonce ne protège que s'il est imprévisible et renouvelé à chaque réponse, donc avec un serveur qui sert `index.html`. Pour un front purement statique : `style-src 'self' 'unsafe-inline'` et tout le reste strict (`default-src 'self'; script-src 'self'; object-src 'none'; base-uri 'none'; frame-ancestors 'none'; form-action 'self'; font-src 'self'`), `img-src` limité à `'self'`, à l'origine des photos et à `blob:`, `connect-src` limité aux API et à l'émetteur. Risque résiduel (exfiltration par CSS) borné par ces origines. Angular et Vuetify ont le même besoin d'un serveur pour un nonce : ce critère ne départage pas les options.
- **Pièges CSP** : police auto-hébergée (`@fontsource/roboto`) ; aperçu local des photos par `blob:` ; configuration en `config.json`, jamais en `<script>` en ligne ; CSP testée sur le build de production servi avec les mêmes en-têtes ; tout événement `securitypolicyviolation` fait échouer Playwright.
- **Jetons** : client public, PKCE `S256`, aucun secret dans le front, jamais de jeton dans une URL (ES-02) ; jetons en mémoire (`InMemoryWebStorage`), jamais en `localStorage` ; `code` et `state` retirés de l'URL au retour ; rechargement = nouvelle redirection sans saisie tant que la session du serveur d'autorisation est active ; pas de renouvellement silencieux par iframe ; jeton dans l'en-tête `Authorization` (pas de requête intersite forgée). Masquer l'interface selon le rôle n'est qu'un confort : le serveur décide (ES-03).
- **Argument pour D06** : un intermédiaire côté serveur (cookie `HttpOnly`, `SameSite`), modèle le plus sûr selon « OAuth 2.0 for Browser-Based Applications », empêcherait le vol du jeton (pas son usage pendant la session) et permettrait un nonce par réponse, donc la suppression de `'unsafe-inline'` ; il ajoute un composant serveur et la protection contre les requêtes intersites forgées.

**4. Montée en compétence React de Martin.**
- À maîtriser pour l'oral : rendu comme fonction pure des props et de l'état, réconciliation et `key` stable, état immuable ; règles des hooks, `useEffect` comme synchronisation avec un système extérieur, `useMemo`/`useCallback` seulement face à un coût constaté ; composants contrôlés (React Hook Form), contexte (session, langue), frontière d'erreur par route, ce que `dangerouslySetInnerHTML` contourne ; clé de requête, mutation et invalidation dans TanStack Query.
- Correspondances depuis Vue : `ref` → `useState` ; `computed` → valeur calculée au rendu ; `watch` → `useEffect` ; composable → hook personnalisé ; `v-model` → `value` et `onChange` ; slot → `children` ; `provide`/`inject` → contexte. Depuis Angular : service injecté → hook et contexte ; flux RxJS de requêtes → TanStack Query.
- Effort réaliste (estimation) : parcours « Learn » de react.dev (≈ 2 j), routeur, requêtes et formulaires (≈ 1 j), puis le catalogue, premier écran réel du POC relu par un expert (≈ 1 à 2 j) : **3 à 5 jours au sprint 1**, puis un ralentissement de 20 à 30 % sur le front du sprint 2.
- Organisation : `src/api/` (client typé, ajout du jeton, « Problem Details » traduits en erreurs typées dont le 409) ; `src/features/<écran>/` avec un hook par cas d'usage (`useCatalogue`, `useArticleEditor`, `useReviewDecision`, `useReservation`, `useSellerNotifications`) et des composants de présentation qui ne reçoivent que des props. Hooks testés par `renderHook` et MSW, composants par leurs props ; chaque fusion du front relue par un expert.

**5. Verdict** : React, TypeScript et MUI sont faisables pour ce POC et conformes à la priorité de Martin, à quatre conditions : 3 à 5 jours de montée en compétence au sprint 1, TanStack Query plutôt que des effets écrits à la main, `'unsafe-inline'` limité aux styles tant que le front reste statique (sauf si D06 retient un intermédiaire côté serveur), et relecture du front par les experts.

## Décision de Martin

> **Back : « Go »** pour les quatre services (Back-A).
>
> **Front : « React + TS + MUI »** (Front-A), application statique.
>
> Motif de Martin : rester dans le périmètre des experts de l'équipe (Go au back, React + TypeScript + MUI au front). La montée en compétence de Martin sur React (3 à 5 jours au sprint 1, relecture du front par un expert) est inscrite comme compétence à acquérir dans la cartographie (C9).
>
> Retenus par défaut, sans objection de Martin, sur avis back et sécurité : un seul module technique partagé `libs/platform` (vérification des jetons, Problem Details, démarrage OpenTelemetry, connexion au canal), sans type métier ni type de contrat ; contrats portés par des schémas (JSON Schema et OpenAPI dans `contracts/`) et non par des types Go partagés ; conditions de l'avis sécurité (`CGO_ENABLED=0`, délais du serveur HTTP, vérification des jetons sans option de contournement, CSP stricte sur les scripts avec `'unsafe-inline'` limité à `style-src`, jetons en mémoire) ; TanStack Query côté front.

Date : 2026-10-09

## Conséquences

Si la recommandation est retenue :

- **D05** : le client du canal retenu devrait exister en Go et être activement maintenu ; aucun candidat courant n'est exclu (SQS et SNS, RabbitMQ, NATS, Kafka). La validation d'un message à la réception devrait passer par un schéma, avant le décodage, puisque le décodeur JSON standard accepte des clés en casse libre et des clés dupliquées (ES-38).
- **D06** :
  - côté services, vérification des jetons avec le jeu de clés mis en cache, l'émetteur, l'audience et l'expiration (`go-oidc` ou `jwx`) ; jeton de service de Transactions par le flux « client credentials » ;
  - côté front, D06 trancherait le mode de conservation des jetons : jetons en mémoire de l'application, ou intermédiaire côté serveur qui garde les jetons et pose un cookie `HttpOnly` (ES-02). Le second ajouterait un composant serveur, à justifier.
- **D07** : aucun SGBD relationnel courant n'est exclu par Go. Les migrations seraient des fichiers SQL versionnés, appliqués par un outil dédié.
- **D08** : en-têtes de sécurité et CSP posés par le point d'entrée ou par l'hébergement statique. `script-src 'self'` strict ; la directive des styles demanderait `'unsafe-inline'` ou un nonce, à cause d'Emotion. Les autres directives (`img-src`, `font-src`, `connect-src`) seraient limitées aux origines de la plateforme.
- **D09** : SDK OpenTelemetry Go pour les traces et les métriques. Propagation du contexte de trace (W3C Trace Context) dans les en-têtes HTTP et dans les attributs des messages. Journaux `slog` en JSON, avec l'identifiant de trace. Aucune instrumentation du front dans le POC, sauf besoin exprimé par D09.
- **D10** :
  - back, par service : `go vet`, golangci-lint, `go test -race -cover`, govulncheck, construction d'une image minimale non root, scan de l'image ;
  - front : vérification des types, ESLint (accessibilité, `react/no-danger`), Vitest, Playwright avec axe-core, audit des dépendances npm ;
  - caches des modules Go et npm pour tenir les durées de pipeline.
- **D12** : couverture (`go test -cover`, Vitest), complexité (golangci-lint, ESLint) et vulnérabilités (govulncheck, audit npm) seraient mesurables nativement ; l'outil d'analyse multi-langage reste à choisir.
- **D13** :
  - unitaires : `go test` et Vitest ;
  - intégration : testcontainers-go (base, canal) ;
  - contrat : schéma ou Pact, avec du Go des deux côtés des trois paires (Place de marché ↔ Contrôle, Place de marché → Interactions, Transactions → Place de marché) ;
  - acceptation : API en Go ; parcours d'écran et accessibilité en Playwright, dans les deux langues ;
  - charge : outil indépendant du langage.
- **Compétences et formation (C9)**, dans `docs/processus/competences-formation.md` :
  - cartographie : Go partagé par Martin et les experts ; React, TypeScript et MUI portés par les experts, **à acquérir par Martin** ; Rust, atout de l'équipe hors stack ; Java/Kotlin, Node, Angular et Vue, compétences de Martin hors stack ;
  - rattachement de ces expertises aux profils du contexte (un lead developer, deux développeurs confirmés) : à fixer par Martin ;
  - expertises transverses requises par la stack, à recenser : OIDC et OAuth2 (jetons de service), OpenTelemetry, accessibilité (WCAG, RGAA), i18n ; s'y ajoutent celles de D02 (diffusion, idempotence, cohérence différée, traces distribuées) ;
  - actions de formation candidates : montée en compétence React du lead developer (binôme avec un expert front, relecture systématique) ; accessibilité RGAA pour les développeurs front ; échanges asynchrones fiables (D02). Le choix et le détail (objectifs, public, modalités, durée, évaluation) relèvent du document ;
  - recrutement : profils « développeur back Go » et « développeur front React/TypeScript », courants et réalistes.
- **Architecture** : [architecture-technique.md](../architecture/architecture-technique.md) § 9 compléterait la ligne D04 ; le point « partage des règles de détection des coordonnées » du § 10 serait clos.
- **Index et README** : statut et lien de D04 à mettre à jour lors de la décision.
- **Aucune expérimentation** : la consigne exclut le test d'un langage ou d'un framework (C1).

## À savoir défendre en soutenance

1. **« Pourquoi Go et pas Rust, plus rapide et plus sûr ? »**
   - Les deux sont dans le périmètre de l'équipe ; ce qui départage, c'est le besoin présent.
   - Performance : la consultation du catalogue est limitée par la base et le réseau, pas par le CPU. Mémoire : chaque service tient déjà dans la plus petite taille d'exécution.
   - Sécurité : Go a aussi la mémoire gérée, des expressions régulières linéaires et des décodeurs d'images sans code C. Ce que Rust ajoute (types somme, absence de nul, courses de données refusées à la compilation) est compensé par le linter d'exhaustivité, les tests et `-race`.
   - Coût de Rust : compilation longue, asynchrone exigeant, profils rares. Il reste possible par la règle d'exception, si une mesure le justifie.
2. **« Pourquoi React, alors que vous maîtrisez Angular et Vue ? »**
   - La stack est choisie pour l'équipe qui maintiendra le code, pas pour le lead. Les experts front font du React avec MUI : relecture, appui, maintenance, recrutement.
   - Les critères techniques départagent peu : i18n, accessibilité en CI (axe-core est indépendant du framework), tests. Angular assainit le HTML par défaut, mais le POC n'affiche aucun HTML saisi : l'échappement de React et l'interdiction de `dangerouslySetInnerHTML` en CI couvrent ES-08.
   - Mon écart est une compétence à acquérir, identifiée dans la cartographie (C9), traitée par binôme et relecture.
   - Vocabulaire à maîtriser : rendu et réconciliation, état et props, hooks et tableau de dépendances, effets et nettoyage, composants contrôlés, contexte.
3. **« Avec quatre services, pourquoi pas un langage par service ? »**
   - Le découpage en services permet le polyglottisme ; il ne l'impose pas.
   - Coût : deux chaînes d'outils, de scans et d'observabilité ; double compétence par développeur ; règles de détection des coordonnées dupliquées entre annonces et chat (M-CHA-01).
   - L'indépendance vis-à-vis du langage est prouvée par le contrat versionné et son test, qu'un outil de substitution doit passer (ENF-17).
   - Java avec Spring, que je maîtrise et dont l'écosystème est le plus intégré, a été écarté : hors du périmètre de l'équipe, empreinte de la JVM.
4. **« Comment le front garantit-il la neutralisation des sorties et la CSP ? »**
   - React échappe tout texte ; `dangerouslySetInnerHTML` est interdit par une règle de lint bloquante ; aucun HTML libre. Le back valide chaque entrée (ES-08).
   - `script-src 'self'` : le build ne produit aucun script en ligne, vérifié en test avec la CSP active.
   - Styles : Emotion les injecte à l'exécution, d'où `'unsafe-inline'` ou un nonce sur les seuls styles. Le risque résiduel (exfiltration par CSS) est plus faible que sur les scripts, sans être nul ; il est borné en limitant les origines des images, des polices et des appels.
   - Preuve : un titre contenant du balisage s'affiche comme du texte, au catalogue et sur l'écran « à vérifier ».
5. **« Vos images légères accélèrent-elles vraiment la montée en charge ? »**
   - Un peu : binaire statique, démarrage en millisecondes, images de quelques dizaines de Mo.
   - Le délai d'ajout d'une instance reste dominé par la plateforme : planification, tirage de l'image, sondes de santé. Il est mesuré au sprint 2 (C5).
   - Gains réels : allumage depuis zéro plus rapide, moins de composants à scanner, plus petite taille d'exécution.
