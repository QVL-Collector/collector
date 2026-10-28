# Revue de code — US-019 Prix incohérents et vendeurs à risque (partie Contrôle)

Branche `feat/US-019-regles-prix-statut-vendeur`, diff `origin/main...HEAD` (217faa9, 737e028). Date : 2026-10-28.

## Vérifications exécutées

| Contrôle | Résultat |
|---|---|
| `gofmt -l`, `go vet` (Contrôle, Place de marché) | 0 problème |
| golangci-lint 2.15.0 `--config ../../.golangci.yml` (Contrôle, Place de marché) | 0 issue |
| `go test ./...` (Contrôle) | vert (`canal`, `coordonnees`, `regles`) |
| Commentaires ajoutés dans le code Go | aucun |
| Mentions d'outils de génération dans le diff | aucune |
| Usages de `regles.New` / `Rules.Evaluate` recherchés | `canal/responder.go:44`, `canal/paire_test.go`, tests `regles` : tous mis à jour |

## Points forts

- Une règle par fichier : `compte.go` (suspension), `completude.go`, `coordonnees.go`, `categorie.go` (catalogue), `prix.go` (plage, vigilance), `vendeur.go` (ancienneté) ; `regles.go` ne garde que les types et l'orchestration.
- Ordre conforme : suspension → complétude → coordonnées (`regles.go:74-81`), puis catégorie → prix → vendeur (`regles.go:84-91`). Les déréférencements `*listing.CategoryCode` et `*listing.PriceCents` (`regles.go:82-83`) sont sûrs : `complete` les garantit non nuls.
- Seuils injectés par `New(Catalogue, SeniorityPolicy)` et copiés à la construction (`categorie.go:12`, `regles.go:68-70`) ; valeurs dans `referentiel.go`, versionnées avec `RulesVersion = 3` ; README à jour (§ Règles v3, § Seuils).
- Ancienneté en jours UTC entiers sur `submitted_at` (`vendeur.go:15-18`) : `Truncate` opère sur le temps absolu, donc aligné sur minuit UTC ; aucune lecture d'horloge dans `regles`. Dates extrêmes : `Sub` sature, pas de panique.
- `member_since` invalide : rejetée par le validateur (`compiler.AssertFormat()`, `libs/platform/message/catalog.go:33`), puis par `calendarDate.UnmarshalJSON` (`canal/contract.go:54-65`) ; dans les deux cas `Decode` échoue et `responder.go:38-40` écarte le message sans évaluer.
- Non-divulgation : le verdict ne porte qu'issue et motif ; motifs déjà présents dans l'énumération fermée de `controle.resultat/v1.schema.json:57-59` ; tests de non-divulgation au niveau règles, paire et bout en bout.
- Pas de régression US-013 / US-018 : les tests existants passent par `evaluer` (vendeur en règle) ; `TestUS013AC1TexteMinimalSuffit` passé à 45 € pour rester un test de complétude ; bout en bout « chaque catégorie » passé à 25 € (15 € hors plage `montres`), justifié.
- Noms clairs (`authorizedToSell`, `isNewcomer`, `exceedsVigilance`, `priceRangeOf`, `refused`, `toVerify`).

## Constats

### Bloquante

Aucune.

### Majeure

Aucune.

### Mineure

1. `services/controle/internal/canal/contract.go:54-65` — la branche d'erreur de `calendarDate.UnmarshalJSON` n'est couverte par aucun test ni exemple invalide (`contracts/messages/controle.demande/exemples/v1/invalides/schema_non_respecte/` n'a pas de `member_since` mal formée). Ajouter un exemple invalide (`"member_since": "2026-02-30"`) et un test `canal` prouvant que la demande est écartée sans résultat.

### Suggestion

1. `services/controle/internal/regles/regles.go:89` — la règle « vendeur à vérifier » combine `vendeur.go` et `prix.go` dans `Evaluate`, alors que le README l'attribue à `vendeur.go` seul. Une fonction `riskySeller(policy, priceRange, submission)` dans `vendeur.go` rendrait la correspondance une règle / un fichier exacte.
2. `services/controle/internal/canal/paire_test.go:64-65` — la paire appelle `rules.Evaluate` et `resultMessage` directement, sans passer par `handle` ; le câblage `responder.go:44` n'est prouvé que par le bout en bout. Acceptable, à garder en tête.

## Reste à faire pour clore l'US (hors lot)

- Place de marché : aucune source de suspension, `Suspended` n'est jamais renseigné (`services/place-de-marche/internal/annonces/stockage/postgres.go:158`) ; AC2 non jouable de bout en bout, AC4 « interface interne du module Comptes » absente.
- `services/place-de-marche/internal/annonces/controle/requests.go:59` : `MemberSince.Format(time.DateOnly)` sans `.UTC()`, alors que `SubmittedAt` est converti (ligne 56) ; à vérifier pour éviter un décalage d'un jour selon le fuseau de la connexion.
- Acceptation API Playwright `request` (AC1, AC2, AC5), validation de Martin, Suivi et DoD du backlog.

## Verdict

Bloquante : 0 · Majeure : 0 · Mineure : 1 · Suggestion : 2. Partie Contrôle prête à fusionner ; la mineure peut être traitée dans le lot Place de marché.
