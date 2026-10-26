# Revue de code — US-012 Contrat v1 du contrôle de publication

| Champ | Valeur |
|---|---|
| US | [US-012](../../../backlog/US-012-contrat-controle-v1.md) |
| Branche | `feat/US-012-contrat-controle-v1` |
| Périmètre | `a6d2018..5bc518d` (3 commits, 51 fichiers, +2179) |
| Date | 2026-10-26 |
| Référentiel | AC de l'US, [D04](../../decisions/D04-stack.md), [D05](../../decisions/D05-communication-interservice.md), [contracts/README.md](../../../contracts/README.md), [rapport de tests](../tests/US-012-rapport-tests.md) |
| Verdict | **OK avec réserves** : aucune issue bloquante ni majeure, 5 mineures, 3 suggestions |

Note sur la base : `main` a été réécrite (`632ef21`), `git diff main...HEAD` ne trouve plus de base commune. L'arbre de `main` est identique à celui de `a6d2018` (`git diff a6d2018 main` vide) : la revue porte donc sur `a6d2018..HEAD`. À rebaser sur `main` avant la PR.

## 1. Vérifications exécutées

| Commande | Résultat |
|---|---|
| `gofmt -l` (`libs/platform`, `contracts`) | Aucun fichier |
| `go vet ./...` (`libs/platform`, `contracts`) | Aucun constat |
| `go test ./... -count=1` (`libs/platform`) | Vert |
| `go test -run=^$ -fuzz=FuzzValidate -fuzztime=20s ./message` | PASS, 353 000 exécutions |
| golangci-lint, govulncheck | Non exécutés : outils absents du poste ; la CI (US-002, non fusionnée) devra couvrir le module `libs/platform` |
| Commentaires dans le code Go | Aucun (seule la directive `//go:embed`, autorisée) |
| Mention d'outil d'IA, d'assistant ou d'agent (code, docs, commits) | Aucune |

## 2. Points forts

- `libs/platform/message/validator.go:33-55` : pipeline de validation dans l'ordre documenté (taille, UTF-8, doublons, choix du contrat, schéma), chaque refus typé par une `Reason` fermée.
- `libs/platform/message/duplicates.go` : passe sur les jetons du décodeur conforme à D04 ; les clés échappées (`reason`) sont bien comparées après décodage (exemples `cle_dupliquee/*.json`).
- Correctif `5bc518d` : plus aucune valeur reçue dans les détails (type, version, clé) ; `schemaViolation` ne cite que l'emplacement et le mot-clé. Vérifié : avec `additionalProperties: false` partout, l'emplacement ne contient jamais de clé inconnue reçue.
- Schémas : `additionalProperties: false` à chaque niveau, listes fermées `issue` et `motif`, `if/then/else` qui impose le motif hors `publie` et l'interdit sinon, bornes de taille (`maxLength`, `maximum`, 64 Kio). AC1 à AC5 couverts.
- Pas de dépendance de `libs/platform` vers `contracts` ni vers `services/` (règle D04) ; le compilateur JSON Schema n'a pas de chargeur HTTP, `schemas.collector.example` n'est jamais résolu sur le réseau.
- API publique compacte et cohérente : `NewValidator`, `Validate`, `Decode`, `Encode`, `NewEnvelope`, `Contract`, `Envelope`, `Reason`, `RejectionError`, `ReasonOf`, `DefaultMaxSize`. Aucun code mort détecté.

## 3. Issues

### Bloquantes

Aucune.

### Majeures

Aucune.

### Mineures

1. **`libs/platform/message/validator.go:62` et `libs/platform/message/envelope.go:14`** — `Validate` accepte des `emitted_at` que `Decode` refuse ensuite.
   Constat reproduit : `2016-12-31T23:59:60Z` (seconde intercalaire) et `2026-11-02t09:15:00z` (minuscules) passent le format `date-time` du schéma, puis `json.Unmarshal` dans `time.Time` échoue : refus `json_invalide` « enveloppe non décodable ». Un outil tiers conforme au schéma verrait ses messages écartés sans nouvel essai.
   Correction : ajouter au schéma de l'enveloppe un `pattern` qui fixe la forme acceptée (`T` et `Z` majuscules, UTC comme l'annonce le README), ou traiter cette divergence en `schema_non_respecte`.
   → Back

2. **`contracts/messages/controle.resultat/v1.schema.json:28-32`** — `rules_version` n'a pas de `maximum` (contrairement à `revision`).
   Constat reproduit : `rules_version: 99999999999999999999` passe `Validate`, puis `Decode` dans un `int64` échoue en `json_invalide`. Même divergence contrat/décodage qu'au point 1.
   Correction : borner `rules_version` (`maximum: 2147483647`, comme `revision`).
   → Back

3. **`libs/platform/message/validator.go:65`** — `Decode` classe en `json_invalide` une erreur de l'appelant (cible non pointeur, `json.InvalidUnmarshalError`).
   Constat reproduit : `Decode(body, valeur)` renvoie « message refusé (json_invalide) ». Un bogue de programmation serait traité comme un message défectueux et écarté sans nouvel essai (MC6).
   Correction : renvoyer une erreur non `RejectionError` quand `errors.As(err, *json.InvalidUnmarshalError)`.
   → Back

4. **`libs/platform/message/contract_test.go:353`** — le cas `"cle echappee en unicode"` ne contient aucun échappement : c'est un doublon de `reason` en clair, identique en substance au cas de la ligne 352. Le nom promet une preuve que le test n'apporte pas (la preuve existe néanmoins dans les exemples `cle_dupliquee/*.json`).
   Correction : écrire la seconde clé `reason`, ou renommer le cas.
   → Test

5. **`libs/platform/message/contract_test.go:336-338`** — `if err != nil { return }` : si `NewEnvelope` échoue, le test passe en silence sans rien prouver.
   Correction : `t.Fatalf("NewEnvelope: %v", err)` comme dans le test précédent (lignes 314-316).
   → Test

### Suggestions

1. **`contracts/README.md` § Enveloppe commune** — `emitted_at` est annoncé « en UTC » mais le schéma (`enveloppe-v1.schema.json:25-29`) accepte tout décalage (`+02:00`). Aligner le README ou le schéma (se règle avec la mineure 1).
2. **`libs/platform/message/contract_test.go:278-307`** — `TestUS012AC3_VersionInconnueEstRefusee` et `TestUS012AC3_TypeInconnuEstRefuse` n'assertent que le refus, pas la raison. `expectRejectedWith(..., ReasonUnknownContract)` (ou `ReasonSchemaViolation` pour les valeurs qui ne sont pas des nombres entiers, selon le comportement voulu) figerait le contrat de raison.
3. **Nommage des schémas** — les `$defs` sont en français (`demande`, `vendeur`, `annonce`, `resultat`, `motif`) alors que les membres sont en anglais (`seller`, `listing`, `reason`). Sans effet sur le contrat publié ; à harmoniser si une v2 est écrite.

## 4. Régressions

Aucune : l'US ajoute deux modules neufs (`collector/contracts`, `collector/libs/platform`) ; aucun code existant n'en dépend ni n'est modifié (recherche des imports `collector/libs/platform` et `collector/contracts` : aucun appelant hors des tests du paquet).

## 5. Sécurité

| Point demandé | Constat |
|---|---|
| Aucune valeur reçue recopiée dans les refus | Conforme depuis `5bc518d` ; le détail `taille_depassee` cite la taille reçue et la limite, pas le contenu |
| Limites de taille | 64 Kio par défaut, vérifiée en premier ; `maxLength` et `maximum` sur les textes et entiers, sauf `rules_version` (mineure 2) |
| Clés dupliquées | Refusées à tout niveau, y compris via échappement Unicode, avant l'analyse par le schéma |
| Champs inconnus, email du vendeur | Refusés à chaque niveau par `additionalProperties: false` ; `account_id` au format `uuid` empêche un email en identifiant |
| Règle ou seuil dans le motif | Impossible : motif réduit à une énumération fermée, aucun autre membre admis |
