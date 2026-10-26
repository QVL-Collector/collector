# Rapport de tests — US-012 Contrat v1 du contrôle de publication

| Champ | Valeur |
|---|---|
| US | [US-012](../../../backlog/US-012-contrat-controle-v1.md) |
| Branche | `feat/US-012-contrat-controle-v1` |
| Date | 2026-10-26 |
| Fonction | Test |
| Référentiel | AC de l'US, [tests.md](../../processus/tests.md), [contracts/README.md](../../../contracts/README.md) |
| Verdict | **Conforme** : tous les AC sont couverts et conformes ; anomalie A1 (mineure) corrigée et revérifiée ; retours 4 et 5 de la [revue de code](../revues/US-012-revue-code.md) traités |

## 1. Livrables de test

| Fichier | Contenu |
|---|---|
| `libs/platform/message/support_test.go` | Chargement des exemples, construction du validateur, assertions de refus et de non-recopie des valeurs reçues |
| `libs/platform/message/contract_test.go` | Tests de contrat étiquetés `US012ACn` (table-driven) |
| `libs/platform/message/fuzz_test.go` | `FuzzValidate`, amorcé par tous les exemples |
| `contracts/messages/controle.resultat/exemples/v1/invalides/json_invalide/resultat-tronque.json` | Exemple ajouté : aucun `json_invalide` pour le résultat |
| `contracts/messages/controle.resultat/exemples/v1/invalides/cle_dupliquee/version-en-double.json` | Exemple ajouté : clé dupliquée dans l'enveloppe (`version` 2 puis 1) |
| `contracts/messages/controle.resultat/exemples/v1/invalides/schema_non_respecte/a-verifier-sans-motif.json` | Exemple ajouté : issue `a_verifier` sans catégorie |
| `contracts/messages/controle.resultat/exemples/v1/invalides/schema_non_respecte/email-vendeur-dans-le-resultat.json` | Exemple ajouté : email du vendeur dans le résultat |
| `contracts/messages/controle.demande/exemples/v1/invalides/contrat_inconnu/version-zero.json` | Exemple ajouté : version 0 |

Le validateur est construit sur `os.DirFS(contracts/)` (même arborescence que `contracts.Messages`), car `libs/platform` ne dépend pas du module `collector/contracts`.

## 2. Exécutions

| Commande (dans `libs/platform`) | Résultat |
|---|---|
| `go test ./... -race` | Non exécutable sur le poste : `-race requires cgo` (pas de gcc, `CGO_ENABLED=0`). À exécuter en CI |
| `go test ./... -count=1 -coverprofile=cover.out -v` | Campagne initiale : 175 verts, 4 rouges (A1). Après correction Back et retours de revue : **194 tests et sous-tests verts, 0 rouge** |
| `go test -run=^$ -fuzz=FuzzValidate -fuzztime=30s ./message` | **PASS** (initial : 1 843 811 exécutions ; relance après correction : 523 020 exécutions, 200 entrées intéressantes), aucune panique ni propriété violée |
| `go tool cover -func=cover.out` | **83,5 %** des instructions du paquet `message` (84,7 % avant la correction Back) |
| `go vet ./...` | Aucun constat |

Exemples chargés : `controle.demande` 3 valides, 15 invalides ; `controle.resultat` 3 valides, 13 invalides (dont 5 ajoutés).

## 3. Résultats par AC

### AC1 — Le contrat v1 contient les schémas de la demande et du résultat avec l'enveloppe commune

| Scénario (Given / When / Then) | Test | Résultat |
|---|---|---|
| Given `enveloppe-v1.schema.json` When on le lit Then `version` et `correlation_id` (avec `id`, `type`, `emitted_at`, `data`) sont requis, `additionalProperties: false` | `TestUS012AC1_EnveloppeCommuneDecritVersionEtCorrelation` | Conforme |
| Given le schéma de la demande Then `article_id`, `revision`, vendeur (`account_id` au format uuid, `member_since`, `suspended`), `photo_count` requis | `TestUS012AC1_DemandeDecritArticleRevisionCompteOpaqueStatutEtPhotos` | Conforme |
| Given le schéma du résultat Then `rules_version`, `issue` requis, `reason` décrit, issue = {publie, refuse, a_verifier}, motif = 6 catégories fermées | `TestUS012AC1_ResultatDecritVersionDesReglesIssueEtCategorieDeMotif` | Conforme |
| Given un résultat refusé ou à vérifier When chaque catégorie de motif est utilisée Then il est accepté (12 cas) | `TestUS012AC1_ChaqueIssueEtChaqueCategorieDeMotifSontAcceptees` | Conforme |
| Given un résultat valide When `Decode` Then l'enveloppe et les données sont restituées | `TestUS012AC1_DecodeRestitueLEnveloppeEtLesDonnees` | Conforme |

### AC2 — Chaque exemple valide est accepté, chaque invalide refusé avec la raison

| Scénario | Test | Résultat |
|---|---|---|
| Given les 34 exemples When `Validate` et `Decode` Then valides acceptés, invalides refusés avec la raison du dossier (`ReasonOf` et `RejectionError`) | `TestUS012AC2_ChaqueExempleEstAccepteOuRefuseAvecSaRaison` | Conforme pour l'acceptation et les raisons (34/34) |
| Même test : le détail du refus ne recopie aucune valeur reçue (chaînes et nombres de 3 caractères et plus) | idem | Conforme après correction (initialement non conforme pour 4 exemples `contrat_inconnu` : anomalie A1, corrigée) |
| Given une limite configurée When le message fait exactement la limite Then accepté ; When un octet de plus Then `taille_depassee` | `TestUS012AC2_MessageAuDelaDeLaTailleConfigureeEstRefuseTailleDepassee` | Conforme |
| Given la taille par défaut (64 Kio) When une description de 64 Kio Then `taille_depassee` | `TestUS012AC2_MessageAuDelaDeLaTailleParDefautEstRefuseTailleDepassee` | Conforme |
| Given un corps vide, `null`, tableau, `{}`, texte, UTF-8 invalide, deux messages Then refusé avec une raison de la liste | `TestUS012AC2_CorpsQuiNEstPasUnMessageJSONEstRefuse` | Conforme |
| Given un même message refusé deux fois Then raison et détail identiques ; `ReasonOf(nil)` sans raison | `TestUS012AC2_RefusEstStableEtRaisonLisibleParReasonOf` | Conforme |

### AC3 — Issue ou catégorie hors liste fermée, version inconnue : refusé

| Scénario | Test | Résultat |
|---|---|---|
| Issue `PUBLIE`, `publié`, vide, `refuse ` (espace), `null`, nombre, liste → `schema_non_respecte` | `TestUS012AC3_IssueHorsListeFermeeEstRefusee` | Conforme |
| Motif en majuscules, vide, `null`, `fraude_suspectee`, liste, `publie` → `schema_non_respecte` | `TestUS012AC3_CategorieDeMotifHorsListeFermeeEstRefusee` | Conforme |
| Version 0, -1, 2, 99999, 1.5, `"v1"`, `null`, `true`, pour la demande et le résultat → refusé | `TestUS012AC3_VersionInconnueEstRefusee` | Conforme |
| Type vide, casse différente, espace final, `controle`, `enveloppe-v1` → refusé | `TestUS012AC3_TypeInconnuEstRefuse` | Conforme |
| `Encode` d'un résultat à issue `approuve` → refusé ; issue `publie` → émis puis accepté | `TestUS012AC3_EncodeRefuseDEmettreUneIssueHorsListe` | Conforme |
| `Encode` d'un résultat en version 2 → refusé (un échec de `NewEnvelope` fait désormais échouer le test, retour de revue 5) | `TestUS012AC3_VersionInconnueRefuseeALEmission` | Conforme |
| Fuzzing : tout message accepté a type et version connus, issue et motif dans les listes | `FuzzValidate` | Conforme (1,8 M exécutions) |

### AC4 — Clé dupliquée, champ inconnu ou email de vendeur : refusé

| Scénario | Test | Résultat |
|---|---|---|
| Doublon de `version`, `type`, `data`, `reason` de même valeur, `reason` doublé via la clé `reason` dont le `r` est écrit en échappement JSON Unicode (U+0072) (retour de revue 4 : l'échappement manquait dans la version initiale), `account_id`, `photo_count` → `cle_dupliquee` | `TestUS012AC4_CleDupliqueeEstRefusee` | Conforme |
| Champ inconnu dans l'enveloppe, les données du résultat, de la demande, le vendeur, l'annonce, clé vide → `schema_non_respecte` | `TestUS012AC4_ChampInconnuEstRefuseAChaqueNiveau` | Conforme |
| Email dans le vendeur, les données, l'enveloppe, l'annonce, comme `account_id`, dans le résultat → `schema_non_respecte`, sans recopie de l'email | `TestUS012AC4_EmailDuVendeurEstRefuse` | Conforme |
| `emitted_at` en seconde intercalaire (`23:59:60Z`), `t`/`z` minuscules (ensemble ou séparément), demande et résultat → `schema_non_respecte` dès `Validate`, même raison au `Decode` | `TestUS012AC4_DateDEmissionHorsFormeStricteEstRefuseeDesLaValidation` | Conforme |
| `rules_version` à 2147483648, 2^63, 20 chiffres, `1e30` → `schema_non_respecte` dès `Validate`, même raison au `Decode` | `TestUS012AC4_VersionDesReglesAuDelaDuMaximumEstRefuseeParLeSchema` | Conforme |
| Fuzzing : aucune réponse hors des cinq raisons, aucun email de vendeur accepté, aucune panique | `FuzzValidate` | Conforme |

### AC5 — Le motif donne une catégorie, jamais la règle ni le seuil

| Scénario | Test | Résultat |
|---|---|---|
| Given les résultats valides Then `data` ne porte que les membres du contrat et le motif est une catégorie fermée | `TestUS012AC5_MotifDesResultatsValidesEstUneCategorieSansRegleNiSeuil` | Conforme |
| Les catégories ne contiennent ni chiffre ni opérateur de seuil | `TestUS012AC5_CategoriesDeMotifNeCitentNiRegleNiSeuil` | Conforme |
| Résultat avec `rule`, `threshold`, motif en texte libre, motif structuré, catégorie suffixée du seuil, `reason_detail` → `schema_non_respecte` | `TestUS012AC5_ResultatDivulguantRegleOuSeuilEstRefuse` | Conforme |
| `a_verifier` ou `refuse` sans motif, `publie` avec motif → refusé | `TestUS012AC5_IssueNonPublieeSansCategorieOuPublieeAvecCategorieEstRefusee` | Conforme |

## 4. Anomalies

### A1 — Le détail d'un refus `contrat_inconnu` recopie le type reçu — mineure — **corrigée**

- **Statut** : corrigée par le Back ; revérifiée le 2026-10-26, les 4 sous-tests concernés sont verts.

- **Référence** : [contracts/README.md](../../../contracts/README.md) § Validation à la réception : « un détail qui nomme l'emplacement et le mot-clé fautifs sans recopier la valeur reçue ». Aucun AC n'est violé (AC2 exige la raison, qui est correcte), d'où la sévérité mineure.
- **Attendu** : le détail ne contient aucune valeur du message reçu.
- **Réel observé** : `message refusé (contrat_inconnu) : aucun contrat pour le type "controle.demande.prioritaire" en version 1`. Avec un type forgé `vendeur.fictif@example.com\nNIVEAU=ERREUR faux journal`, le détail contient `"vendeur.fictif@example.com\nNIVEAU=ERREUR faux journal"` (valeur citée et échappée, tronquée à 64 caractères : pas d'injection de ligne, mais la donnée reçue est recopiée).
- **Reproduction** : `go test ./message -run TestUS012AC2_ChaqueExempleEstAccepteOuRefuseAvecSaRaison -v` dans `libs/platform` ; 4 sous-tests rouges (`contrat_inconnu/type-inconnu.json`, `version-2.json` ×2, `version-zero.json`).
- **Fichier en cause** : `libs/platform/message/validator.go` (construction du refus `contrat_inconnu`) ; à corriger par la fonction Back. Alternative à arbitrer par Martin : préciser dans le README que le type et la version peuvent être cités.

## 5. Points d'attention (non anomalies)

- `-race` non exécuté localement faute de cgo ; à prouver dans le job CI unitaire back.
- La limite de taille est inclusive (message de exactement `maxSize` octets accepté), cohérent avec « taille maximale ».

## 6. Conclusion

Tous les AC sont couverts et conformes. L'anomalie A1 (mineure) est corrigée et revérifiée. Les retours de revue côté Test (cas d'échappement Unicode réel, échec explicite de `NewEnvelope`) sont traités, et les cas `emitted_at` stricts et `rules_version` borné passent après la correction Back.
