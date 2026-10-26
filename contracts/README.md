# Contrats d'échange

Source unique des contrats entre services, lisible par tout outil qui comprend JSON Schema 2020-12 : un outil de substitution, interne ou acheté, peut s'y conformer sans lire le code (ENF-17, ENF-19). Aucun type Go partagé : chaque service possède ses propres types, et le test de contrat vérifie qu'ils respectent ces schémas ([D04](../docs/decisions/D04-stack.md)).

## Organisation

```
contracts/
  contracts.go                       embarque les schémas (variable contracts.Messages)
  messages/
    enveloppe-v1.schema.json         enveloppe commune
    <type>/v<N>.schema.json          un schéma par type de message et par version
    <type>/exemples/v<N>/valides/    messages acceptés
    <type>/exemples/v<N>/invalides/<raison>/   messages refusés, rangés par raison attendue
```

Le nom du répertoire est le `type` du message, le nom du fichier sa `version` : `messages/controle.demande/v1.schema.json` s'applique aux messages `"type": "controle.demande", "version": 1`. Les schémas ne portent pas de `$id` : les `$ref` relatifs se résolvent par le chemin des fichiers.

## Enveloppe commune

| Membre | Contenu |
|---|---|
| `id` | UUID du message, conservé en cas de rejeu |
| `type` | Type du message |
| `version` | Version du contrat du type (entier) |
| `emitted_at` | Date d'émission, RFC 3339, en UTC |
| `correlation_id` | UUID de corrélation, repris de la demande dans le résultat |
| `data` | Objet propre au type |

Le contexte de trace W3C voyage dans les en-têtes du message, l'émetteur dans la propriété `user-id` (MC2) : ni l'un ni l'autre n'est dans le corps.

## Contrat v1 du contrôle de publication

| Message | Émetteur → destinataire | Échange, clé | Schéma |
|---|---|---|---|
| `controle.demande` | Place de marché → Contrôle | `controle.demandes`, `v1` | [v1](messages/controle.demande/v1.schema.json) |
| `controle.resultat` | Contrôle → Place de marché | `controle.resultats`, `v1` | [v1](messages/controle.resultat/v1.schema.json) |

- **Demande** : article (`article_id`, `revision`), date de soumission, vendeur réduit à son identifiant opaque et à ses attributs de statut (`member_since`, `suspended`), contenu soumis aux règles (titre, description, code de catégorie, prix en centimes, nombre de photos). Une saisie absente est un texte vide ou `null` : c'est le Contrôle qui la juge.
- **Résultat** : `article_id` et `revision` repris de la demande, `rules_version`, `issue` et, sauf issue `publie`, `reason`.
- **Issues** : `publie`, `refuse`, `a_verifier`.
- **Catégories de motif** : `annonce_incomplete`, `coordonnees_detectees`, `compte_non_autorise`, `prix_a_confirmer`, `vendeur_a_verifier`, `categorie_inconnue`. Le motif ne cite jamais la règle ni le seuil appliqués.
- **Données minimales** : `additionalProperties: false` à chaque niveau ; aucun email, nom ni coordonnée du vendeur ne peut y figurer.

## Versionnement

- Un nouveau jeu de règles change `rules_version`, jamais la version du contrat.
- Tout changement qui ferait refuser un message hier valide, ou accepter un membre inconnu hier refusé, crée une nouvelle version : nouveau fichier `v<N+1>.schema.json`, nouvelle clé de routage `v<N+1>`, ancienne version servie jusqu'à la migration du dernier consommateur.
- Ajouter une valeur à une liste fermée (issue, motif) est un tel changement.

## Validation à la réception

`collector/libs/platform/message` applique, dans cet ordre : taille maximale (64 Kio par défaut), UTF-8, rejet des clés dupliquées, choix du schéma par `type` et `version`, validation du schéma avec assertion des formats, puis décodage dans le type du service.

```go
validator, err := message.NewValidator(contracts.Messages, message.DefaultMaxSize)
envelope, err := validator.Decode(body, &demande)
reason, rejected := message.ReasonOf(err)
```

Un message refusé porte une raison parmi `taille_depassee`, `json_invalide`, `cle_dupliquee`, `contrat_inconnu`, `schema_non_respecte`, et un détail qui nomme l'emplacement et le mot-clé fautifs sans recopier la valeur reçue. Il est écarté sans nouvel essai (MC6). Côté émetteur, `Encode` refuse de sérialiser un message non conforme.

## Exemples

Chaque exemple de `valides/` doit être accepté ; chaque exemple de `invalides/<raison>/` doit être refusé avec cette raison. Le cas `taille_depassee` se construit dans le test, au-delà de la taille configurée.
