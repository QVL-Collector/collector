# Service Place de marché

Module Annonces : catalogue public, brouillons du vendeur, catégories, soumission au contrôle de publication et application de son résultat. Module Comptes minimal : statut des vendeurs (inscription, suspension). Go, bibliothèque standard, `pgx`, `goose`, `libs/platform` ([D04](../../docs/decisions/D04-stack.md)). Contrats : [`contracts/api/place-de-marche/v1.openapi.yaml`](../../contracts/api/place-de-marche/v1.openapi.yaml), messages [`controle.demande` et `controle.resultat` v1](../../contracts/README.md).

## Sous-commandes et ports

| Commande | Rôle | Compte de base attendu |
|---|---|---|
| `place-de-marche serve` (par défaut) | API publique sur `ADRESSE_API`, santé sur `ADRESSE_GESTION`, relais de l'outbox et consommateur des résultats de contrôle | `marche_execution` |
| `place-de-marche migrate` | Applique les migrations embarquées, puis s'arrête (Job Helm `pre-install`, `pre-upgrade`) | `marche_migration` |
| `place-de-marche suspendre <compte-id>`, `place-de-marche retablir <compte-id>` | Suspend ou rétablit un compte vendeur, puis s'arrête (voir « Comptes ») | `marche_migration` |
| `place-de-marche rejouer --depuis <RFC3339> [--jusqu-a <RFC3339>]` | Rejoue l'outbox sur une fenêtre, puis s'arrête (voir « Contrôle fermé par défaut ») | `marche_migration` |

| Port | Contenu | Exposition |
|---|---|---|
| 8080 | `/marche/v1/...` | Routé par l'ALB |
| 9090 | `GET /sante` (vivant), `GET /disponibilite` (base joignable ; ni Keycloak ni le broker interrogés) | Jamais routé ; sondes Kubernetes |

## Catalogue public (US-010, US-023)

Sans jeton : `GET /marche/v1/articles?categorie&page&taille` (du plus récemment publié au plus ancien) et `GET /marche/v1/articles/{id}`. Seuls les articles `publie` sont lus ; un article inexistant ou non publié répond le même 404. Aucune donnée du vendeur n'est exposée. Paramètre répété, vide ou hors bornes : 400 ; catégorie inconnue : 400 `categorie_inconnue`. Liste lue en transaction `READ ONLY` `REPEATABLE READ` (total et page cohérents), fiche en lecture directe ; index partiels `etat = 'publie'` (migration 00004).

## Cycle de soumission (US-013)

1. `POST /marche/v1/vendeur/articles/{id}/soumission` avec `{"revision": n}` : article brouillon ou refusé du vendeur, révision lue exigée (409 si périmée, 404 si l'article est à un autre vendeur). Dans une transaction : état `en_controle`, révision + 1, motif effacé, demande `controle.demande` v1 écrite dans la table `outbox`. Réponse 202.
2. Relais de l'outbox (une goroutine par réplica, `FOR UPDATE SKIP LOCKED`, lots de 50 toutes les 500 ms) : publie sur l'échange `controle.demandes`, clé `v1`, propriété `user-id` = `CANAL_UTILISATEUR`, attend la confirmation du broker, puis renseigne `publie_le`. Dans un lot, les messages routés sont marqués publiés ; un message sans file de destination ou un lot refusé par le broker reste en attente, avec un journal `outbox.essai_differe` et une attente croissante entre essais (jusqu'à 10 s). Rejeu, purge et filet de relance : voir « Contrôle fermé par défaut ».
   La demande porte `seller.account_id` (sujet du jeton), `seller.member_since` (date UTC d'inscription) et `seller.suspended`, lus dans la même transaction par l'interface interne du module Comptes (voir « Comptes »), sans email ni nom. Même lecture pour la revalidation (US-021).
3. Consommateur de la file `pdm.resultats-controle` (acquittement manuel, préchargement 10) : un message dont le `user-id` n'est pas `CANAL_EMETTEUR_ATTENDU`, ou non conforme au contrat, est rejeté sans remise en file (vers `pdm.resultats-controle.ecartes`) et journalisé (`canal.message_ecarte`, voir « Contrôle fermé par défaut »). Un résultat n'est appliqué que si l'article est `en_controle` à la révision du résultat ; sinon (doublon, révision périmée) il est acquitté sans effet. L'application écrit l'état, le motif, et une ligne dans `resultat_controle` (journal des résultats, une issue par révision).
4. Une erreur transitoire (base indisponible) ne rejette jamais le message : le consommateur réessaie avec une attente croissante, sans acquitter.

## Contrôle fermé par défaut (US-024)

- **Jamais publié sans contrôle** : seuls deux chemins mènent à `publie`, l'application d'un résultat `publie` du Contrôle sur un article `en_controle` à la même révision, et la décision d'un admin sur un article `a_verifier` (lui-même issu d'un résultat). La soumission n'attend pas le broker : l'article reste `en_controle`, hors du catalogue, tant qu'aucun résultat n'est appliqué, Contrôle arrêté ou non.
- **Reprise sans perte ni doublon** : la demande attend dans l'outbox ou dans la file durable `controle.demandes` ; au retour du Contrôle, le résultat est appliqué une fois. Un doublon, ou un résultat pour une révision dépassée, touche zéro ligne : acquitté sans effet, journal `controle.resultat_ignore`.
- **Mise à l'écart** : un message refusé par le consommateur (émetteur inattendu, JSON illisible, champ manquant, type ou version inconnus) part vers `<file>.ecartes` sans nouvel essai, sans bloquer les suivants ; un message relivré sans fin (plantage pendant son traitement) y part après 5 livraisons (`delivery-limit` 4, [infra/canal](../../infra/canal/README.md)). Chaque mise à l'écart par le consommateur produit un journal Warn `event=canal.message_ecarte` (`file`, `file_ecartes`, `message_id`, `emetteur`, `motif`, `relivre`), base de l'alerte d'US-015 ; les mises à l'écart faites par le broker se lisent sur la longueur des files `.ecartes`.
- **Filet de relance** ([D05](../../docs/decisions/D05-communication-interservice.md), [D02](../../docs/decisions/D02-decoupage-applicatif.md)) : `article.demande_controle_le` date le dernier envoi au contrôle (posée par la soumission et le recontrôle, migration 00007). Toutes les 30 s, chaque réplica relance les articles `en_controle` dont le dernier envoi date de plus de 2 min (lots de 50, `FOR UPDATE SKIP LOCKED`) : dans une transaction, date d'envoi remise à maintenant et nouvelle `controle.demande` v1 dans l'outbox, reconstruite à partir de l'article, pour la **même révision**. Journal Warn `event=controle.demande_relancee` (`article_id`, `revision`) ; échec : `controle.relance_en_echec`, nouvel essai au balayage suivant. Pas de plafond de relances (non prévu par D05) : tant que le Contrôle est arrêté, une demande par article toutes les 2 min environ. Les doublons de résultat sont absorbés par la garde de révision et `resultat_controle_une_issue_par_revision`.
- **Rejeu de l'outbox (MC9)** : `place-de-marche rejouer --depuis <RFC3339> [--jusqu-a <RFC3339>]`, lancé en Job par l'exploitation, même configuration que `suspendre`. Remet `publie_le` à `NULL` pour les seules lignes publiées dans `[depuis, jusqu-a[` (`jusqu-a` vaut maintenant par défaut) ; aucune autre colonne ni ligne n'est touchée, le relais les republie avec leur identifiant d'origine. Fenêtre refusée si le début ne précède pas la fin, si la fin est dans le futur, ou si le début remonte au-delà de la conservation (7 jours). Trace : journal Info `event=outbox.rejeu` (`correlation_id`, `depuis`, `jusqu_a`, `lignes`), plus la création du Job dans le journal du plan de contrôle. Sans risque de double effet : les consommateurs sont idempotents (garde de révision).
- **Purge de l'outbox (MC8)** : chaque réplica appelle, au démarrage puis toutes les heures, la fonction `purger_outbox_publiee()` (migration 00007), qui supprime les lignes publiées depuis plus de 7 jours. `marche_execution` n'a pas `DELETE` sur `outbox` : seulement `EXECUTE` sur cette fonction `SECURITY DEFINER`, qui ne peut jamais supprimer une ligne non publiée. Journaux `outbox.purge` (`lignes`) et `outbox.purge_en_echec`.
- **Comptage I3** ([D12](../../docs/decisions/D12-indicateurs-qualite.md)), sur la base `marche`, après le tir :

```sql
WITH demande AS (
    SELECT (convert_from(o.corps, 'UTF8')::jsonb #>> '{data,article_id}')::uuid AS article_id,
           (convert_from(o.corps, 'UTF8')::jsonb #>> '{data,revision}')::integer AS revision,
           count(*) AS envois,
           bool_and(o.publie_le IS NULL) AS jamais_publiee
    FROM outbox AS o
    WHERE o.type_message = 'controle.demande'
    GROUP BY 1, 2
)
SELECT count(*)                                                                 AS demandes_emises,
       coalesce(sum(d.envois), 0) - count(*)                                    AS envois_relances,
       count(*) FILTER (WHERE d.jamais_publiee)                                 AS demandes_non_publiees,
       count(r.message_id)                                                      AS resultats_appliques,
       count(*) FILTER (WHERE r.message_id IS NULL AND a.revision > d.revision) AS demandes_depassees,
       count(*) FILTER (WHERE r.message_id IS NULL AND a.revision = d.revision) AS demandes_sans_resultat
FROM demande AS d
JOIN article AS a ON a.id = d.article_id
LEFT JOIN resultat_controle AS r ON r.article_id = d.article_id AND r.revision = d.revision;
```

  Une demande est comptée une fois par article et révision, quels que soient ses envois (relances du filet dans `envois_relances`). I3 tenu si `demandes_sans_resultat = 0` et `demandes_emises = resultats_appliques + demandes_depassees` (une demande dépassée est celle d'une révision remplacée par une modification avant son résultat, US-021). Sans modification pendant le tir, `demandes_emises = resultats_appliques`. Lisible par `marche_execution` comme par `marche_migration`. Le comptage ne vaut que dans la fenêtre conservée par la purge (7 jours) : une demande purgée n'est plus comptée, son résultat non plus.

## Recontrôle après modification (US-021)

`PUT /marche/v1/vendeur/articles/{id}` sur un article `publie`, `en_controle` ou `a_verifier` : dans une transaction qui verrouille la ligne (`SELECT … FOR UPDATE`), contenu remplacé, révision + 1, état `en_controle`, motif effacé et nouvelle demande `controle.demande` v1 dans l'outbox. L'article quitte le catalogue sans seuil. Tout résultat portant sur l'ancienne révision est acquitté sans effet (étape 3). Brouillon et refusé : modification sans contrôle, comme avant.

Le service ne déclare aucune file ni aucun échange : la topologie est posée par l'IaC ([D05](../../docs/decisions/D05-communication-interservice.md)). Une file ou un échange absent, ou un droit refusé, arrête le service en erreur.

## Articles à vérifier (US-022)

Routes réservées au rôle `admin` (403 pour tout autre rôle, 401 sans jeton valide) :

- `GET /marche/v1/admin/articles?page&taille` : articles `a_verifier`, du plus ancien au plus récent (index partiel `article_a_verifier_idx`), avec le contenu de la révision examinée, le motif du contrôle et `vendeurId`.
- `POST /marche/v1/admin/articles/{id}/decision` avec `{"revision": n, "decision": "publier" | "refuser", "motif"?}` : `publier` donne `publie`, `refuser` donne `refuse` avec un motif de la liste fermée (`annonce_incomplete`, `coordonnees_detectees`, `prix_abusif`, `article_interdit`, `vendeur_non_fiable`). La révision ne change pas. 409 `revision-perimee` si l'article est passé à une autre révision, 409 `article-non-a-verifier` s'il n'est plus à vérifier, 422 `saisie-invalide` sur le motif (absent pour refuser, présent pour publier, hors liste).

Dans la même transaction que la décision, une ligne est ajoutée à `decision_admin` (admin, article, révision, décision, motif, date ; une seule décision par révision). Le rôle d'exécution peut seulement y insérer et lire : ni `UPDATE` ni `DELETE`. Après le commit, un journal Info `event=annonces.decision_admin` reprend les mêmes champs.

## Comptes (US-019)

Module `internal/comptes`, séparé d'Annonces. Table `compte` (migration 00006) : `id` (sujet du jeton), `inscrit_le`, `suspendu`, `suspendu_le` (renseigné si et seulement si suspendu). Aucune jointure ni clé étrangère avec `article`.

- **Inscription** : la création d'un brouillon et toute demande de contrôle (soumission, revalidation) inscrivent le vendeur dans leur transaction (`INSERT … ON CONFLICT DO NOTHING`, `inscrit_le = now()`). La migration inscrit les vendeurs existants avec la date de leur premier article.
- **Interface interne** lue par Annonces (`stockage.SellerAccounts`) : inscription et statut (date d'inscription, suspension), dans la transaction de l'appelant ; rien d'autre.
- **Pas d'API** : conformément à [D02](../../docs/decisions/D02-decoupage-applicatif.md) (module Comptes minimal, fictif, en lecture seule), aucune route HTTP ne lit ni ne modifie un compte.
- **Suspension, par l'exploitation** : `place-de-marche suspendre <compte-id>` et `place-de-marche retablir <compte-id>`, avec la même configuration de base que `migrate` (`BASE_DE_DONNEES_URL`, rôle `marche_migration`). Idempotent (la date d'une suspension déjà posée est conservée) ; identifiant invalide ou compte inconnu : arrêt en erreur. Journal Info `event=comptes.suspension` avec `compte_id`, `suspendu`. Un vendeur suspendu garde l'accès à ses brouillons ; le Contrôle refuse ses soumissions (`compte_non_autorise`).

## Limitation de débit (US-027)

Intergiciel `libs/platform/ratelimit` (seau de jetons en mémoire), posé après l'authentification et le rôle, sur les deux routes qui déclenchent un contrôle : `POST /marche/v1/vendeur/articles/{id}/soumission` et `PUT /marche/v1/vendeur/articles/{id}` (recontrôle, US-021). Le dépôt de photo (US-020) s'y branchera de la même façon. Le catalogue et les autres routes vendeur ne sont pas limités.

- Deux seaux par requête, débités ensemble : l'adresse du client et le compte (sujet du jeton). Le premier vide refuse : 429 Problem Details, `Retry-After` en secondes arrondies au supérieur, journal Info `event=http.debit_depasse` avec `adresse_client` (adresse lue), `cle` (clé du seau d'adresse, /64 en IPv6) et `limite` (`adresse` ou `compte`).
- Adresse du client : dernière valeur du dernier `X-Forwarded-For`, celle ajoutée par l'ALB en mode `append` ; à défaut, adresse source de la connexion. Une adresse IPv6 compte pour son /64. La confiance dans l'en-tête repose sur les NodePorts joignables depuis l'ALB seul (MS2, MS4).
- Adresse de tir : `LIMITE_DEBIT_TIR_ADRESSE`, injectée à chaque session par `up.sh`, jamais versionnée ; ses requêtes (adresse et compte) suivent la limite de tir, toutes les autres la limite nominale. Aucune route ni en-tête de contournement (ME4, MT6).
- Mémoire bornée : seaux pleins purgés chaque minute, 50 000 clés au plus (éviction d'une clé quelconque au-delà).
- Écart V1 : limite par réplica, non partagée ; la limite effective est multipliée par le nombre de réplicas (architecture § 9.8).

## Variables d'environnement

| Variable | Commande | Défaut | Contenu |
|---|---|---|---|
| `BASE_DE_DONNEES_URL` | toutes | obligatoire | `postgres://<rôle>:<mot de passe>@<hôte>:5432/marche?sslmode=verify-full&sslrootcert=<paquet RDS>` ; secret du rôle d'exécution pour `serve`, du rôle de migration pour `migrate`, `suspendre` et `retablir` |
| `BASE_DE_DONNEES_TLS_EXIGE` | toutes | `true` | `false` seulement en local : autorise un autre `sslmode` que `verify-full` |
| `OIDC_EMETTEUR` | `serve` | obligatoire | Émetteur exact, ex. `https://auth.<domaine>/realms/collector` |
| `OIDC_JEU_DE_CLES_URL` | `serve` | `<émetteur>/protocol/openid-connect/certs` | Adresse interne du jeu de clés si l'émetteur public n'est pas joignable depuis le cluster |
| `OIDC_AUDIENCE` | `serve` | `place-de-marche` | Audience exigée dans le jeton |
| `OIDC_CLIENT_AUTORISE` | `serve` | `collector-web` | Valeur exigée de `azp` |
| `CANAL_HOTE` | `serve` | obligatoire | Hôte AMQP du broker |
| `CANAL_PORT` | `serve` | `5672` | Port AMQP |
| `CANAL_VHOST` | `serve` | `collector` | Vhost |
| `CANAL_UTILISATEUR` | `serve` | obligatoire | Identité de la Place de marché sur le canal, aussi posée en `user-id` des messages publiés |
| `CANAL_MOT_DE_PASSE` | `serve` | obligatoire | Secret de cette identité, généré à chaque `up` (MC5) |
| `CANAL_EMETTEUR_ATTENDU` | `serve` | `controle` | Seul `user-id` accepté sur la file des résultats (MC2) |
| `LIMITE_DEBIT_PAR_MINUTE` | `serve` | `30` | Débit nominal par adresse et par compte, en requêtes par minute (entier ≥ 1) |
| `LIMITE_DEBIT_RAFALE` | `serve` | `10` | Rafale nominale (taille du seau, entier ≥ 1) |
| `LIMITE_DEBIT_TIR_ADRESSE` | `serve` | aucune | Adresse IP du poste de tir, injectée à l'exécution ; vide : aucune limite relevée |
| `LIMITE_DEBIT_TIR_PAR_MINUTE` | `serve` | `6000` | Débit relevé pour l'adresse de tir, en requêtes par minute |
| `LIMITE_DEBIT_TIR_RAFALE` | `serve` | `200` | Rafale relevée pour l'adresse de tir |
| `ADRESSE_API` | `serve` | `:8080` | Écoute de l'API |
| `ADRESSE_GESTION` | `serve` | `:9090` | Écoute de la santé |
| `JOURNAL_NIVEAU` | toutes | `INFO` | `DEBUG`, `INFO`, `WARN`, `ERROR` ; journaux JSON sur la sortie standard |

## Rôles PostgreSQL attendus

Créés par l'amorçage de la base `marche`, avant toute migration ([D07](../../docs/decisions/D07-donnees.md), MD1, MD2) :

- `marche_migration` : propriétaire de la base et du schéma `public`, seul à exécuter du DDL ; secret livré au seul Job de migration.
- `marche_execution` : `NOSUPERUSER NOCREATEDB NOCREATEROLE`, propriétaire d'aucun objet, sans `CREATE` sur `public` ; nom fixe, car la première migration lui accorde `SELECT`, `INSERT`, `UPDATE` sur les tables et `USAGE`, `SELECT` sur les séquences par `ALTER DEFAULT PRIVILEGES`. Aucun `DELETE`, aucun droit sur la table de version de `goose`. Le relais garde sa transaction ouverte pendant la publication : `idle_in_transaction_session_timeout` est à poser sur ce rôle.

Une migration exécutée sans que `marche_execution` existe échoue.

## Limites d'entrée

Corps JSON de 32 Kio au plus (413 au-delà) ; titre 200 caractères, description 5 000 ; prix jusqu'à 1 000 000 €, frais de port jusqu'à 1 000 €. Messages du canal : 64 Kio au plus, validés contre leur schéma avant décodage (MC6).
