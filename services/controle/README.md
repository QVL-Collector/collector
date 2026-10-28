# Service Contrôle de publication

Applique les règles de publication à chaque demande de contrôle et publie le résultat. Sans base, sans stockage, sans horloge métier : la règle est une fonction pure de la demande ([D02](../../docs/decisions/D02-decoupage-applicatif.md), [D05](../../docs/decisions/D05-communication-interservice.md)). Go, bibliothèque standard, `libs/platform` ([D04](../../docs/decisions/D04-stack.md)). Contrat : messages [`controle.demande` et `controle.resultat` v1](../../contracts/README.md).

## Fonctionnement

1. Consomme la file `controle.demandes` (acquittement manuel, préchargement 10).
2. Écarte sans remise en file (vers `.ecartes`) et journalise tout message dont le `user-id` n'est pas `CANAL_EMETTEUR_ATTENDU`, ou non conforme au contrat (taille, clé dupliquée, schéma).
3. Évalue les règles, publie `controle.resultat` v1 sur l'échange `controle.resultats`, clé `v1`, avec `user-id` = `CANAL_UTILISATEUR` et l'identifiant de corrélation de la demande, attend la confirmation du broker, puis seulement acquitte la demande. Une panne entre les deux produit un doublon, jamais une perte ; la Place de marché l'absorbe par sa garde de révision.
4. Une publication refusée ou non confirmée n'acquitte pas la demande : nouvel essai avec une attente croissante.

Le service ne déclare aucune file ni aucun échange : la topologie est posée par l'IaC. Une file ou un échange absent, ou un droit refusé, arrête le service en erreur.

## Règles, version 2

Évaluées dans cet ordre, la première qui s'applique fixe l'issue :

| Condition | Issue | Motif |
|---|---|---|
| Titre, description ou catégorie vides (espaces seuls compris), ou prix absent (US-013) | `refuse` | `annonce_incomplete` |
| Titre ou description contenant des coordonnées (US-018) | `refuse` | `coordonnees_detectees` |
| Catégorie absente du référentiel du Contrôle (US-013) | `a_verifier` | `categorie_inconnue` |
| Sinon | `publie` | — |

Chaque résultat porte `rules_version: 2`, contrat inchangé en v1. Référentiel des catégories : `internal/regles/referentiel.go`, aligné sur le jeu de catégories amorcé par la Place de marché. Prix, statut du vendeur et photos : US-019 et US-020.

Détection des coordonnées (`internal/coordonnees`) : texte normalisé (NFKD, accents et caractères invisibles retirés, minuscules), puis expressions RE2 à coût linéaire, compilées une fois. Elle repère les emails (`@`, « arobase », « (at) », « point », « dot »), les liens (`http://`, `www`, domaine suivi d'une extension connue), les pseudos `@…` et les noms de réseaux sociaux, et les numéros français (`0X` + 8 chiffres, `+33`, `0033`), chiffres séparés par espaces, points, tirets ou parenthèses, ou écrits en toutes lettres. Une année (19xx, 20xx) interrompt un numéro, pour ne pas confondre dates et téléphones. Le résultat ne cite ni la règle ni le texte détecté.

## Port

| Port | Contenu | Exposition |
|---|---|---|
| 9090 | `GET /sante` (vivant), `GET /disponibilite` (connecté au broker) | Jamais routé ; sondes Kubernetes |

## Variables d'environnement

| Variable | Défaut | Contenu |
|---|---|---|
| `CANAL_HOTE` | obligatoire | Hôte AMQP du broker |
| `CANAL_PORT` | `5672` | Port AMQP |
| `CANAL_VHOST` | `collector` | Vhost |
| `CANAL_UTILISATEUR` | obligatoire | Identité du Contrôle sur le canal, aussi posée en `user-id` des résultats |
| `CANAL_MOT_DE_PASSE` | obligatoire | Secret de cette identité, généré à chaque `up` (MC5) |
| `CANAL_EMETTEUR_ATTENDU` | `marche` | Seul `user-id` accepté sur la file des demandes (MC2) |
| `ADRESSE_GESTION` | `:9090` | Écoute de la santé |
| `JOURNAL_NIVEAU` | `INFO` | `DEBUG`, `INFO`, `WARN`, `ERROR` ; journaux JSON sur la sortie standard |

Les messages ne sont jamais journalisés en clair : seuls l'identifiant du message, l'article, la révision, l'issue et le motif le sont.
