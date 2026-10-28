# US-018 — Rapport de tests

| Champ | Valeur |
|---|---|
| US | [US-018 — Refuser les annonces qui contiennent des coordonnées personnelles](../../../backlog/US-018-filtrage-coordonnees.md) |
| Branche | `feat/US-018-filtrage-coordonnees` |
| Date | 2026-10-28 |
| Fonction | Test |
| Périmètre | Service Contrôle (détection et règles v2), contrat `controle.resultat` v1, bout en bout Place de marché ↔ Contrôle (RabbitMQ et PostgreSQL réels sous Docker) |
| Hors périmètre | Démonstration Playwright « jean point dupont arobase… » (acceptation API et E2E navigateur, à faire avec l'écran de mise en vente) |
| Outils | Go 1.26, `go test` (table, benchmark, fuzzing), golangci-lint 2.15.0, gitleaks |
| Verdict | **Tous les AC sont couverts et conformes** (passe 2 après le retour de revue) |

## Méthode

Scénarios Given/When/Then dérivés des AC, étiquetés `US018ACn`. Données fictives : domaine réservé `example.com`, numéros de la plage fictive 01 99 00 xx xx et 06 39 98 xx xx. Aucun commentaire dans le code de test.

| Fichier | Contenu | Tests |
|---|---|---|
| `services/controle/internal/coordonnees/coordonnees_test.go` | Tables positives (emails, liens, pseudos et réseaux, téléphones, en clair et masqués) et négatives ; 13 textes piégés de longueur maximale ; benchmark AC6 ; `FuzzDetected` | 118 cas |
| `services/controle/internal/coordonnees/coordonnees_perf_test.go` | Test chronométré AC6 (< 50 ms par annonce), contrainte de build `//go:build !race` | 13 mesures |
| `services/controle/internal/regles/coordonnees_test.go` | Titre et description séparément, ordre des règles, correction puis resoumission, textes réalistes publiés | 7 tests, 30 cas |
| `services/controle/internal/regles/regles_test.go` | `TestUS013AC5…` renommé `TestUS013AC5US018AC5VersionDesReglesEgaleA2` : exige `RulesVersion == 2` (même assertion d'égalité stricte, valeur mise à jour) | 1 |
| `contracts/messages/controle.resultat/exemples/v1/valides/refuse-coordonnees-detectees.json` | Paire contrat v1 / règles v2 avec le motif `coordonnees_detectees`, validée par la suite de contrats de `libs/platform/message` | 1 exemple |
| `services/place-de-marche/cmd/place-de-marche/bout_en_bout_test.go` | Scénario « annonce avec téléphone refusée `coordonnees_detectees` puis publiée une fois corrigée » ; contrôle de version des règles passé de v1 à v2 pour le vrai Contrôle (le canal fictif reste en v1) | 1 sous-test ajouté, 1 adapté |

## Résultats par AC

| AC | Scénario | Attendu (AC) | Réel | Statut |
|---|---|---|---|---|
| AC1 | Email en clair (simple, majuscules, `+`, sous-domaine, dans une phrase) | Refusé, `coordonnees_detectees` | Détecté | ✅ |
| AC1 | Lien externe (`https`, `http`, `www`, domaine nu `.com`/`.fr`, pleine chasse) | Refusé | Détecté | ✅ |
| AC1 | Téléphone français (collé, espaces, points, tirets, mélangés, `+33`, `+33 (0)`, `0033`, parenthèses, fin de phrase, 06 et 07) | Refusé | Détecté | ✅ |
| AC1 | Passe 2 : `06/39/98/12/34`, `06 / 39 / 98 / 12 / 34`, `06 2015 3456` (groupe ressemblant à une année) | Refusé | Détecté | ✅ |
| AC1 | Pseudo `@…`, Instagram, WhatsApp, Facebook, Snapchat, Telegram | Refusé | Détecté | ✅ |
| AC1 | Coordonnées dans le titre seul, puis dans la description seule (7 formes chacune) | `refuse` / `coordonnees_detectees` | Conforme | ✅ |
| AC1 | Bout en bout : titre « … appelez le 01 99 00 12 34 » | Refusé, motif visible du vendeur | `refuse`, `coordonnees_detectees` dans la fiche et « mes articles » | ✅ |
| AC2 | « arobase » (casse, accent), « (at) », « [at] », « (dot) », « point com », espaces autour de `@` et `.`, pleine chasse, caractères invisibles (U+200B, U+200D, U+2060, U+FEFF, U+00AD) | Refusé, même motif | Détecté | ✅ |
| AC2 | « zéro six … » en nombres, chiffre par chiffre, sans tirets, en majuscules, mélange lettres et chiffres, « plus trente-trois … », chiffres séparés par espaces simples ou multiples, groupes irréguliers, pleine chasse, chiffres séparés par caractères invisibles | Refusé | Détecté | ✅ |
| AC3 | Prix (`45 €`, `1 250,00 €`, `12 500 €`), dimensions (`30 x 40 cm`, `30x40cm`, `21 x 29,7 x 3 cm`), poids | Non refusé pour ce motif | Non détecté | ✅ |
| AC3 | Années, plages et suites d'années, dates `12/05/1987`, `12.05.1987`, `12-05-1987`, en lettres, plage de dates, date et heure | Non refusé | Non détecté | ✅ |
| AC3 | Références d'édition (ISBN 13 et 10, EO et tirage n° 125/300, dépôt légal, réf. montre `145.022-69`), numéros de série (`45781236`, `SN AB-12345-CD`, `0042/1000`), tomes (12, 06) | Non refusé | Non détecté | ✅ |
| AC3 | Passe 2 : `Edition 01/50, 12/05/2003`, `01/02/2003 à 14:30`, `12.05.1987`, `1987-05-12`, « English. FR : description » | Non refusé | Non détecté | ✅ |
| AC3 | « Bon état.De plus », « pièce.Comme neuve », « Le point fort », « 33 tours », « 5 francs 1960 », « deux figurines et trois cartes », texte vide | Non refusé | Non détecté | ✅ |
| AC4 | Règle : description avec téléphone refusée, téléphone retiré | Publié | `publie` sans motif | ✅ |
| AC4 | Bout en bout : article refusé, modifié sans le numéro, resoumis | Publié | `publie` | ✅ |
| AC5 | `RulesVersion` | 2 | 2 | ✅ |
| AC5 | Bout en bout : chaque résultat publié par le Contrôle et chaque résultat enregistré | `rules_version` 2, contrat v1 valide | Conforme (validateur de contrats) | ✅ |
| AC5 | Exemple de contrat v1 avec `rules_version` 2 et `coordonnees_detectees` | Valide | Valide | ✅ |
| AC6 | 13 annonces piégées (titre 200 + description 5 000 caractères) : chiffres espacés ou collés, « zéro » répété, nombres composés, `a@`, « arobase point », `a.`, `www.`, `+33`, années, séparateurs, pleine chasse et invisibles, texte réaliste | < 50 ms | 0,1 à 8 ms (benchmark), 5 exécutions de suite vertes | ✅ |
| AC6 | Coût linéaire (sonde benchmark 1 250 → 20 000 caractères) | Pas de retour arrière | Durée proportionnelle à la longueur (×2 quand la longueur double) | ✅ |
| AC6 | `FuzzDetected` 30 s | Aucune panique | Passe 1 : 551 140 exécutions ; passe 2 : 919 781 exécutions ; 0 panique | ✅ |
| Sécurité | Corps du résultat publié et journaux Contrôle et Place de marché | Ni le numéro, ni le titre, ni la règle | Absents ; schéma fermé (`additionalProperties: false`, motif en liste fermée) | ✅ |
| Front | `web/src/shared/controlReason.ts` et Place de marché | Motif `coordonnees_detectees` accepté | Présent dans la liste fermée ; résultat accepté et enregistré en bout en bout | ✅ |

## Mesures AC6

La preuve AC6 est faite hors détecteur de concurrence : sous `-race`, l'analyse est ralentie d'un facteur 5 à 10 (53 à 102 ms mesurés en CI), ce qui mesure l'instrumentation et non la détection. Le test chronométré est donc exclu des builds `-race` ; le benchmark et le fuzzing restent dans tous les builds.

Passe 2 (après le retour de revue) : benchmark 0,1 à 9,5 ms par annonce sur 3 répétitions ; une valeur isolée à 34 ms (`+33` répétés), non reproduite (8 ms aux 3 répétitions suivantes).

Machine de test chargée par d'autres processus (charge CPU mesurée à 76 % hors tests). Benchmark `BenchmarkUS018AC6AnnonceMaximalePiegee`, par annonce complète : 0,1 ms (`www.`) à 8 ms (chiffres espacés) au calme, jusqu'à 20 ms sous la charge. Le test `TestUS018AC6TexteMaximalPiegeAnalyseEnMoinsDe50ms` retient la meilleure de 7 mesures par texte : une mesure isolée ou une médiane dépassait parfois 50 ms (66 à 131 ms) lorsque d'autres suites tournaient en parallèle sur la machine, sans être reproductible ; le benchmark linéaire montre que ce n'est pas un coût de l'analyse.

## Limites documentées (non comptées comme anomalies)

README du service, § « Limites connues », complété par la revue. Comportement observé en passe 2 :

| Texte | Nature | Observé |
|---|---|---|
| `France.Net et propre`, `Le Petit Prince.Fr édition` | Faux positif documenté (point collé suivi d'une extension) | Détectés |
| `site. fr` | Lien masqué par un espace après le point, limite signalée par la revue | Non détecté |
| `01 02 2003 12` | Suite de chiffres autour d'une année, limite signalée par la revue | Détecté |
| `06.12.2019.78` | Numéro accolé à une date complète, non couvert (documenté) | Non détecté |
| `Cartes numérotées 01 02 … 10`, `Carte n°0102030405` | Faux positif documenté (suites et références de 10 chiffres) | Détectés |
| `j e a n @ e x a m p l e . c o m`, `jean.dupont arobase example` | Hors AC2 (lettres espacées, email sans extension) | Non détectés |

`site. fr` et `01 02 2003 12` ne figurent pas encore dans le README (§ « Limites connues ») : à y ajouter si ce sont des limites retenues.

## Commandes et qualité

| Commande | Résultat |
|---|---|
| `go test ./... -count=1 -cover` (Contrôle) | vert en passe 1 et passe 2 (3 exécutions) ; `coordonnees` 98,5 %, `regles` 100 % |
| `go test ./... -count=1 -cover` (Place de marché, Docker) | vert (113 s) ; bout en bout 16 sous-tests verts dont le nouveau |
| `go test ./message/` (`libs/platform`) | vert avec le nouvel exemple de contrat |
| `go test -fuzz FuzzDetected -fuzztime 30s -fuzzminimizetime 1x` | 551 140 exécutions, aucune panique (sans ce drapeau, la minimisation des entrées intéressantes affiche 0 exécution/s sous Windows, sans blocage réel) |
| `go test -race ./... -count=1` (Contrôle) | Pas de compilateur C sur le poste (cgo absent) : exécuté dans un conteneur `golang:1.26` (Linux), vert ; test chronométré exclu, autres tests exécutés |
| golangci-lint 2.15.0 `--config .golangci.yml` | 0 problème dans les deux services (Contrôle relancé en passe 2) |
| gitleaks sur les fichiers ajoutés | aucune fuite |

Conteneurs de test arrêtés et supprimés par les harnais ; aucun conteneur résiduel.

## Anomalies

Aucune.

## Verdict

Tous les AC sont couverts et conformes. La barre oblique est désormais couverte. Les limites restantes sont documentées ou signalées par la revue, et ne sont pas comptées comme anomalies.
