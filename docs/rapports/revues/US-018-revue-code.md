# Revue de code — US-018 Filtrage des coordonnées

Branche `feat/US-018-filtrage-coordonnees`, diff `origin/main...HEAD` (d17fc76, e8849cd). Date : 2026-10-28.

## Vérifications exécutées

| Contrôle | Résultat |
|---|---|
| `gofmt -l`, `go vet`, golangci-lint 2.15.0 (Contrôle) | 0 problème |
| `go test ./...` (Contrôle) | vert |
| Commentaires ajoutés dans le code Go | aucun |
| Mentions d'outils de génération dans le diff | aucune |
| Sondes manuelles sur une copie du paquet (hors dépôt) | voir constats |

## Points forts

- `internal/coordonnees` n'expose qu'une fonction, `Detected` (`coordonnees.go:10`) ; normalisation NFKD, retrait des `Mn` et `Cf` (accents, ZWSP, ZWJ, trait d'union conditionnel), espaces Unicode ramenés à l'espace, pleine chasse couverte (`0６ 12 34 56 78` détecté).
- Uniquement `regexp` (RE2), expressions compilées une fois ; entrée bornée par le contrat `controle.demande` (titre 200, description 5000) ; recherche de numéro bornée à 13 chiffres par départ (`telephone.go:50`) : coût linéaire.
- Ordre des règles correct et documenté : incomplète, puis coordonnées, puis catégorie (`regles.go:58-66`, README).
- `RulesVersion = 2`, contrat v1 inchangé, exemple `refuse-coordonnees-detectees.json` ajouté (AC5).
- Non-divulgation : le verdict ne porte que le motif ; le journal (`canal/responder.go:53-59`) n'écrit ni la règle ni le texte.
- `golang.org/x/text` passé en dépendance directe dans `go.mod` : correct.

## Constats

### Bloquante

Aucune.

### Majeure

Aucune.

### Mineure

1. `services/controle/internal/coordonnees/telephone.go:12` — la barre oblique n'est pas un séparateur : `06/39/98/12/34` passe. Contournement trivial, proche de l'AC2. Correctif sans risque pour les dates : une expression dédiée `\b0[1-9](?:\s?/\s?\d{2}){4}\b` (cinq paires de deux chiffres, préfixe 0[1-9]), plutôt qu'ajouter `/` à `digitRun` (qui accolerait `01/50 02/50 …`). Sondé : `Edition 01/50, 12/05/2003` reste non détecté avec cette forme. Ajouter les cas à la table de tests.
2. `services/controle/internal/coordonnees/telephone.go:46` — l'interruption par une année se contourne en regroupant les chiffres : `06 2015 3456` n'est pas détecté (`06 20 15 34 56` l'est). Concerne les numéros contenant une paire 19 ou 20 suivie de deux chiffres. Piste : n'interrompre sur une année que si le groupe précédent ou suivant est séparé par `/`, `-` ou `.`, ou accepter le numéro si la séquence complète forme exactement 10 chiffres commençant par 0[1-9]. À trancher, sinon l'écrire dans le README comme limite connue.
3. `services/controle/internal/coordonnees/adresses.go:30` — `spacedDot` accepte « mot. fr » : une annonce bilingue `Description in English. FR : carte rare` est refusée (sondé). Les faux positifs `France.Net`, `Prince.Fr` (`adresses.go:29`) sont rares en pratique (point collé sans espace, peu naturel en français). Celui en `. FR` est plus probable sur une marketplace internationale. Piste : retirer `fr` et `eu` de `spacedWebTLD` lorsqu'un espace suit le point, ou exiger l'absence d'espace après le point pour ces deux extensions.
4. `services/controle/internal/coordonnees/adresses.go:31` — les noms de réseaux seuls déclenchent un refus : `Figurine The Messenger` est refusé (sondé). Pour des objets de collection (jeux, goodies de marque), le risque existe mais reste faible ; le vendeur peut reformuler. À documenter comme limite, ou exiger un contexte (`insta :`, `sur snap`, suivi d'un pseudo).

### Suggestion

5. Faux positifs `Cartes numérotées 01 02 … 10` et `Carte n°0102030405` (`telephone.go:12`, `:62`) : indiscernables d'un 01 02 03 04 05. Risque réel mais limité : le refus est explicite, l'annonce se republie après reformulation (AC4), et l'AC3 vise dates, dimensions et références usuelles. Ne pas affaiblir la règle ; consigner ces cas dans le README comme limites acceptées.
6. Chiffres d'autres écritures (`٠٦١٢٣٤٥٦٧٨`, arabe-indien) et homoglyphes cyrilliques (`gmаil`) non couverts : NFKD ne les ramène pas à l'ASCII. Contournement marginal ; à noter comme limite, pas à traiter dans cette US.

## Verdict

Bloquante 0, majeure 0, mineure 4, suggestion 2. Prêt à fusionner : oui. Les mineures 1 et 2 sont recommandées avant livraison (contournements simples), les autres relèvent d'un choix produit à consigner dans le README.

Corrections à adresser au développeur back (Go, service Contrôle).
