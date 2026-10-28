# Service Contrôle de publication

Applique les règles de publication à chaque demande de contrôle et publie le résultat. Sans base, sans stockage, sans horloge métier : la règle est une fonction pure de la demande ([D02](../../docs/decisions/D02-decoupage-applicatif.md), [D05](../../docs/decisions/D05-communication-interservice.md)). Go, bibliothèque standard, `libs/platform` ([D04](../../docs/decisions/D04-stack.md)). Contrat : messages [`controle.demande` et `controle.resultat` v1](../../contracts/README.md).

## Fonctionnement

1. Consomme la file `controle.demandes` (acquittement manuel, préchargement 10).
2. Écarte sans remise en file (vers `.ecartes`) et journalise tout message dont le `user-id` n'est pas `CANAL_EMETTEUR_ATTENDU`, ou non conforme au contrat (taille, clé dupliquée, schéma).
3. Évalue les règles, publie `controle.resultat` v1 sur l'échange `controle.resultats`, clé `v1`, avec `user-id` = `CANAL_UTILISATEUR` et l'identifiant de corrélation de la demande, attend la confirmation du broker, puis seulement acquitte la demande. Une panne entre les deux produit un doublon, jamais une perte ; la Place de marché l'absorbe par sa garde de révision.
4. Une publication refusée ou non confirmée n'acquitte pas la demande : nouvel essai avec une attente croissante.

Le service ne déclare aucune file ni aucun échange : la topologie est posée par l'IaC. Une file ou un échange absent, ou un droit refusé, arrête le service en erreur.

## Règles, version 3

Évaluées dans cet ordre, la première qui s'applique fixe l'issue ; les refus passent avant les « à vérifier » :

| Condition | Issue | Motif | Fichier |
|---|---|---|---|
| Vendeur suspendu (US-019) | `refuse` | `compte_non_autorise` | `compte.go` |
| Titre, description ou catégorie vides (espaces seuls compris), ou prix absent (US-013) | `refuse` | `annonce_incomplete` | `completude.go` |
| Titre ou description contenant des coordonnées (US-018) | `refuse` | `coordonnees_detectees` | `coordonnees.go` |
| Catégorie absente du référentiel du Contrôle (US-013) | `a_verifier` | `categorie_inconnue` | `categorie.go` |
| Prix hors de la plage de la catégorie, bornes incluses dans la plage (US-019) | `a_verifier` | `prix_a_confirmer` | `prix.go` |
| Ancienneté du vendeur sous le seuil et prix strictement au-dessus du seuil de vigilance de la catégorie (US-019) | `a_verifier` | `vendeur_a_verifier` | `vendeur.go` |
| Sinon | `publie` | — | |

Chaque résultat porte `rules_version: 3`, contrat inchangé en v1. Les règles sont des fonctions pures de la demande, seuils injectés par `regles.New(catalogue, politique d'ancienneté)`.

### Seuils

Fichier versionné `internal/regles/referentiel.go`, et non variables d'environnement : un seuil modifié change l'issue des règles, il passe donc par une revue et une nouvelle `rules_version`. Les seuils ne figurent jamais dans le résultat (motif fermé, US-012 AC5). Catégories alignées sur le jeu amorcé par la Place de marché.

| Catégorie | Plage acceptée | Vigilance (vendeur récent) |
|---|---|---|
| `bandes-dessinees` | 1 € à 5 000 € | 300 € |
| `cartes-a-collectionner` | 0,50 € à 10 000 € | 500 € |
| `figurines` | 1 € à 3 000 € | 300 € |
| `jouets-anciens` | 2 € à 5 000 € | 400 € |
| `monnaies` | 1 € à 20 000 € | 1 000 € |
| `montres` | 20 € à 50 000 € | 2 000 € |
| `vinyles` | 1 € à 2 000 € | 200 € |

- Plage : un prix très bas sert d'appât vers un contact hors plateforme, un prix très élevé de blanchiment par auto-achat (M-FRA-02). Les bornes encadrent les prix courants du jeu de données fictif ; au-delà, l'admin confirme.
- Vigilance : montant à partir duquel une fraude d'un vendeur sans historique coûte cher en remboursement (M-FRA-01), sans soumettre à l'admin les petites ventes d'un nouveau venu.
- Ancienneté minimale : 30 jours, calculés en jours entiers UTC entre `seller.member_since` et la date de `submitted_at` de la demande ; le Contrôle ne lit pas l'horloge. Un mois laisse le temps aux premières ventes d'un compte d'être livrées et contestées avant qu'il vende cher sans relecture.

Photos : US-020.

Détection des coordonnées (`internal/coordonnees`) : texte normalisé (NFKD, accents et caractères invisibles retirés, minuscules), puis expressions RE2 à coût linéaire, compilées une fois. Elle repère les emails (`@`, « arobase », « (at) », « point », « dot »), les liens (`http://`, `www`, domaine suivi d'une extension connue), les pseudos `@…` et les noms de réseaux sociaux, et les numéros français (`0X` + 8 chiffres, `+33`, `0033`), chiffres séparés par espaces, points, tirets ou parenthèses, ou écrits en toutes lettres, ainsi que les paires séparées par `/` (`06/12/34/56/78`). Les dates complètes (`12.05.1987`, `1987-05-12`) sont écartées avant la recherche de numéros. Le résultat ne cite ni la règle ni le texte détecté.

### Limites connues

- Faux positifs : numéros de cartes ou de tomes en suite (`01 02 03 04 05`), référence de 10 chiffres au format d'un numéro (`n°0102030405`), nom de réseau seul (« Messenger » dans un titre), point collé suivi d'une extension (`France.Net`, `Prince.Fr`), date écrite avec des espaces suivie de chiffres (`01 02 2003 12`).
- Non couverts : chiffres arabes-indiens et autres systèmes de chiffres, homoglyphes cyrilliques ou grecs, « O » pour zéro, numéro accolé à une date complète (`06.12.2019.78`), adresse dont le point est collé au mot qui précède puis suivi d'une espace (`site. fr`).

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
