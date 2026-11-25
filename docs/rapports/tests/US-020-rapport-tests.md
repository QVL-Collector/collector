# Rapport de tests — US-020 Ajouter une photo à un article, en toute sécurité

| Champ | Valeur |
|---|---|
| US | [US-020](../../../backlog/US-020-photo-securisee.md) |
| Branche | `feat/US-020-photo-securisee` |
| Date | 2026-11-23 ; lot 2 (corrections de revue) 2026-11-25 |
| Fonction | Test |
| Périmètre | Back : Place de marché (dépôt, ré-encodage, S3, URL présignées, droits) et Contrôle (règles v4) ; Front : dépôt, aperçu, retrait, erreurs, rafraîchissement d'URL, accessibilité ; AC8 : partie vérifiable dans le code seulement |
| Référentiel | AC de l'US, README de `services/place-de-marche` (§ Photos, variables `PHOTOS_*`, S3 local), contrat [`v1.openapi.yaml`](../../../contracts/api/place-de-marche/v1.openapi.yaml), README de `web` |
| Verdict | **Conforme** : AC1 à AC7 et AC9 couverts et conformes ; aucune anomalie, lot 2 compris (route admin MD7, bornes de configuration, orientation EXIF, journal d'objet orphelin, limites front configurables, photo de l'examen admin). AC8 reporté à l'inventaire après déploiement (DevOps/AWS), partie code conforme. Une clarification ouverte |

## 1. Livrables de test

### Back

| Fichier | Contenu |
|---|---|
| `services/place-de-marche/internal/annonces/photo/photoessai/photoessai.go` | Générateurs de fichiers de test : JPEG et PNG, JPEG avec EXIF GPS fictif (APP1 `Exif`, IFD GPS, latitude 48° 51′ 29,59″ N, longitude 2° 17′ 40,12″ E), PNG avec `eXIf` GPS et `tEXt` de lieu, GIF, WebP, SVG avec `onload`, texte nommé `.jpg`, polyglottes (signature JPEG + HTML, GIF + ZIP), JPEG suivi d'un script, JPEG tronqué, en-têtes géants (PNG, JPEG), bombe PNG de 100 mégapixels compressée (≈ 100 Kio), lecture des segments JPEG |
| `services/place-de-marche/internal/annonces/photo/sanitizer_test.go` | Tests unitaires du nettoyage (AC1, AC3, AC5, AC6) et fuzzing `FuzzUS020DecodageNeCedeJamais` |
| `services/place-de-marche/cmd/place-de-marche/photos_banc_test.go` | SeaweedFS `3.80` (épinglé par empreinte) en conteneur testcontainers-go, partagé par le processus de test, un bucket aléatoire par banc ; variables `PHOTOS_*` posées sur chaque lancement de la Place de marché ; client S3 de test (inventaire, lecture d'objet) ; `assurerUnePhoto` |
| `services/place-de-marche/cmd/place-de-marche/photos_test.go` | `TestUS020PhotosSurPostgreSQLSeaweedFSEtControle` : PostgreSQL + RabbitMQ + SeaweedFS + Contrôle réel (binaire), 19 scénarios ordonnés |
| `services/controle/internal/regles/photo_test.go`, `services/controle/internal/canal/photo_test.go` | AC2 côté Contrôle : règle de complétude photo, par la règle et par une demande du contrat v1 |

### Front

| Fichier | Contenu |
|---|---|
| `web/src/acceptance/US-020/photoHarness.ts` | Faux `XMLHttpRequest` (le dépôt passe par XHR pour la progression) branché sur la place de marché simulée d'US-013 ; envois retenus pour observer l'aperçu ; réponses 413, 415, 422, 409 scriptées ; fichiers JPEG, PNG, GIF, WebP, SVG, faux `.jpg` ; complément jsdom (`Blob.arrayBuffer`, `URL.createObjectURL`, absents de jsdom, présents dans les navigateurs) |
| `web/src/acceptance/US-020/publicPhotoHarness.ts` | Route `GET /marche/v1/articles/{id}/photos` simulée au-dessus du harnais public d'US-023, compteur de relectures |
| `web/src/acceptance/US-020/photoWording.ts` | Libellés FR/EN attendus |
| `web/src/acceptance/US-020/AC1-depot-apercu-retrait.test.tsx` | 5 tests |
| `web/src/acceptance/US-020/AC5-AC6-erreurs-de-depot.test.tsx` | 11 tests |
| `web/src/acceptance/US-020/AC4-url-presignee-rafraichie.test.tsx` | 5 tests |
| `web/src/acceptance/US-020/AC9-accessibilite-clavier-et-texte-alternatif.test.tsx` | 11 tests (dont axe FR et EN) |
| `web/e2e/US-020/photos.ts` | Libellés FR/EN, fichiers, création d'un brouillon de démonstration |
| `web/e2e/US-020/AC9-depot-photo-clavier.spec.ts` | Dépôt et retrait au clavier seul (sélecteur de fichier ouvert par Entrée), fichiers refusés |
| `web/e2e/US-020/AC4-AC9-photos-axe.spec.ts` | Fiche publiée avec photo chargée et texte alternatif, axe-core WCAG 2.1 A/AA sur la fiche et sur le dépôt |

### Lot 2 (revue [US-020-revue-code.md](../revues/US-020-revue-code.md))

| Fichier | Contenu |
|---|---|
| `services/place-de-marche/cmd/place-de-marche/photos_test.go` | + 4 scénarios : route admin MD7 (200, 403, 401, 404) et JPEG d'orientation 6 publié redressé |
| `services/place-de-marche/internal/config/photos_test.go` | `config.LoadPhotos` : défauts, bornes acceptées et refusées de chaque variable `PHOTOS_*`, bucket obligatoire et nom valide, adresse S3, erreurs réunies (majeure 3) |
| `services/place-de-marche/internal/annonces/photo/orientation_test.go` | Orientation EXIF 1 à 8 en petit et gros boutisme, orientation hors norme, PNG non pivoté, segments EXIF mal formés, longueur d'APP1 incohérente ; fuzzing `FuzzUS020LecteurEXIF` |
| `services/place-de-marche/internal/annonces/photo/photoessai/photoessai.go` | + image à quatre quarts colorés, image stockée pour chaque orientation, TIFF d'orientation II/MM, APP1 à longueur déclarée libre, lecture des quarts d'une sortie |
| `services/place-de-marche/internal/annonces/photo_orpheline_test.go` | Journal `annonces.photo_orpheline` (dépôt et retrait) avec doublures des ports `PhotoRecords`, `PhotoSanitizer`, `PhotoObjects` |
| `web/src/acceptance/US-020/limites-configurables.test.tsx` | 11 tests : `photos.maxBytes`, `maxPixels`, `maxPerArticle` de `config.json` |
| `web/src/acceptance/US-020/AC4-image-conservee-pendant-le-rafraichissement.test.tsx` | 1 test : relecture avant expiration, image précédente conservée |
| `web/src/acceptance/US-020/MD7-photo-examen-admin.test.tsx` | 6 tests : photo de la carte d'examen admin |
| `web/src/acceptance/US-022/adminReviewHarness.ts` | Route `GET …/admin/articles/{id}/photos` simulée ; `listReads()` exclut les lectures de photos |

## 2. Réparation des tests existants

| Cassure | Réparation | Ce qui reste prouvé |
|---|---|---|
| `publicHandler` prend `api.PhotoRoutes` (`environnement_test.go`) | `api.PhotoRoutes{}` pour les bancs en mémoire (aucune route photo appelée) ; vrai service photo sur les bancs complets via `serve` | Tous les scénarios US-009 à US-032 en mémoire |
| `serve` exige `PHOTOS_BUCKET` | `lancerPlaceDeMarche` et `lancerPlaceDeMarcheSurKeycloak` créent un bucket SeaweedFS et posent `PHOTOS_BUCKET`, `PHOTOS_S3_ADRESSE`, `PHOTOS_S3_STYLE_CHEMIN`, `AWS_*` (README, § S3 en local) | Bancs complets US-013, US-018, US-019, US-021, US-024, US-028, US-030, US-031, US-032 |
| Le Contrôle refuse désormais un article sans photo (règles v4) | `soumettreAvecSucces` dépose une photo JPEG si l'article n'en a pas, sur les bancs à service photo réel (`environnement.avecPhotos`) ; les scénarios AC2 d'US-020 soumettent sans passer par ce raccourci | Tous les parcours qui attendaient `publie` |
| Version des règles 3 attendue | `versionDesReglesV4 = 4` (bout en bout US-013), `TestUS020AC2VersionDesReglesEgaleA4`, `exigerResultat` en v4 (canal du Contrôle) | Chaque résultat porte la version courante et respecte le contrat v1 |
| Fixtures du Contrôle sans photo | `annonceComplete()` et les annonces construites à la main portent `PhotoCount: 1` | Tous les verdicts US-013, US-018, US-019 inchangés |
| Droits minimaux : table `photo` nouvelle | `photo` ajoutée aux tables du service ; `DELETE FROM photo` et `UPDATE photo SET cle_objet` refusés | Aucune table hors service, aucun objet possédé |
| Lot 2, harnais US-022 : la carte d'examen lit désormais `GET …/admin/articles/{id}/photos` (404 du faux serveur, et lecture comptée par `listReads()`) | Route photo explicite ; `listReads()` filtre `/photos` | Comptes de relecture de la liste US-022 inchangés |
| Lot 2, instabilité sous charge (mineure 6) | Délais explicites `screenTimeout` (15 s) et `announcementTimeout` sur toutes les attentes des tests US-020, comme les harnais US-010, US-023, US-028, US-030 ; `vi.setConfig({ testTimeout: 30000 })` ajouté aux deux fichiers clavier qui tombaient au délai de 5 s par défaut (US-008 AC10, US-025 AC5) ; délais explicites sur les attentes de US-009 AC11 | Voir § 3 bis |
| Harnais US-013 : `GET …/photos` renvoyait l'article | Route explicite `GET /marche/v1/vendeur/articles/{id}/photos` (liste), `DELETE …/photos/{photoId}` (204 ou 404), traitée avant les réponses scriptées des articles | Les 62 tests US-013 verts, sans réponse d'article consommée par la section photo |

## 3. Exécutions

Lot 2 (2026-11-25), après les corrections des développeurs :

| Commande | Résultat |
|---|---|
| `go test ./... -count=1` (`services/place-de-marche`) | Vert, tous paquets, bancs Docker compris (`cmd/place-de-marche` 807 s, `annonces`, `annonces/photo`, `config`) |
| `go test ./... -count=1` (`services/controle`) | Vert (`canal`, `coordonnees`, `regles`) |
| Fuzz `FuzzUS020DecodageNeCedeJamais`, 60 s | Vert : 1 231 591 exécutions, 81 entrées intéressantes |
| Fuzz `FuzzUS020LecteurEXIF`, 60 s | Vert : 264 915 exécutions, 61 entrées intéressantes |
| `gofmt -l`, `go vet ./...` (les deux services) | Aucun constat |
| `golangci-lint run --config <racine>/.golangci.yml ./...` (les deux services) | 0 constat |
| `npx vitest run` | 53 fichiers, 590 tests (dont 50 US-020) ; voir § 3 bis |
| `npx playwright test` (FR + bascule) | 74 verts |
| `E2E_LANGUES=fr,en npx playwright test` | 145 verts (dont 10 US-020) |
| `npx tsc -b` | Aucune erreur |
| `npx eslint src e2e` | 0 constat |
| `npx prettier --check .` | Conforme |

Lot 1 (2026-11-23) : mêmes commandes vertes (Vitest 572, Playwright 74 et 145, fuzz du décodage 2 214 120 exécutions).

### 3 bis. Instables et relances

- **Back** : première exécution complète du lot 1 : 3 échecs, tous des cassures dues à la livraison (version de règles 3 attendue, table `photo` absente de la liste des droits), corrigées au § 2. Aucun instable observé sur les bancs Go.
- **Front, mineure 6** : le rapport du lot 1 disait « aucun instable observé » ; la revue a constaté sous charge des échecs US-020 au délai de 1 s par défaut des `findBy*`. Toutes les attentes des tests US-020 portent désormais un délai explicite (15 s). Suite complète lancée **20 fois** en parallèle (4 séries de 5 passes, dont une série où deux suites tournaient en même temps) : **aucun échec US-020**.
- Hors US-020, la même charge fait tomber d'autres tests existants au délai par défaut : US-025 AC5 et US-008 AC10 (délai de test de 5 s) et US-009 AC11 (`findByRole('alert')` à 1 s). Délais explicites ajoutés à ces trois fichiers, selon les conventions du lot des instables. Dernière série de 5 passes : 4 entièrement vertes (590/590) ; la 3e a eu 11 échecs groupés en US-008 AC1/AC2, US-010 AC1, US-023 AC1 et US-028 AC2 (aucun US-020), pendant qu'un cluster k3d d'une autre session tournait sur le poste. Ces fichiers n'ont pas de délai explicite partout ; je ne les ai pas modifiés hors du périmètre de cette US.

## 4. Résultats par AC

### AC1 — Brouillon : JPEG ou PNG ré-encodé, stocké dans le bucket privé sous un nom généré, affiché

| Scénario (Given / When / Then) | Niveau | Attendu (AC) | Réel | Statut |
|---|---|---|---|---|
| Brouillon ; JPEG envoyé avec `Content-Type: application/octet-stream` et `filename="maison-de-vacances-gps.jpg"` | Banc complet | Ré-encodé, stocké, nom généré, affiché | 201 ; un seul objet ajouté ; clé sans l'identifiant d'article, de photo ni le nom du fichier ; objet `image/jpeg` différent de l'original, sans segment APPn ; listé dans l'espace du vendeur ; l'URL sert exactement l'objet stocké | Conforme |
| PNG transparent | Banc complet | Ré-encodé | Stocké en `image/jpeg`, 40 × 20, sans métadonnée | Conforme |
| Cinq dépôts sur cinq articles | Banc complet | Nom généré par le serveur | Cinq clés distinctes ; partie variable ≥ 20 caractères | Conforme |
| Seconde photo (plafond 1) | Banc complet | (contrat) | 409 `photos-limite-atteinte`, bucket inchangé | Conforme |
| Retrait | Banc complet | (contrat) | 204, objet supprimé du bucket, liste vide, second retrait 404, nouveau dépôt 201 | Conforme |
| Dépôt sur un article publié | Banc complet | (contrat) | 409 `article-non-modifiable`, aucun objet écrit | Conforme |
| JPEG, PNG, PNG transparent, 1 × 1, JPEG suivi d'un script | Unitaire | Ré-encodé | JPEG décodable, dimensions conservées, octets différents de l'original, ni `Exif`, ni `GPS`, ni script | Conforme |
| Dépôt JPEG puis PNG depuis l'espace vendeur | Vitest | Affiché dans son espace | Corps brut envoyé en `POST …/photos` avec `Bearer` et `Content-Type` du fichier ; photo affichée depuis l'URL présignée ; « Photo ajoutée. » annoncé | Conforme |
| Envoi retenu | Vitest | Aperçu | Aperçu `blob:` avec texte alternatif « Aperçu de la photo en cours d'envoi », remplacé par la photo enregistrée | Conforme |
| Retrait depuis l'espace vendeur | Vitest | — | `DELETE`, photo retirée, « Photo retirée. » annoncé, dépôt de nouveau proposé | Conforme |
| Brouillon de démonstration, dépôt et retrait au clavier | Playwright FR/EN | Affiché dans son espace | Photo chargée (`naturalWidth > 0`), annonces, retrait | Conforme |

### AC2 — Article sans photo contrôlé : « refusé », motif « annonce incomplète » (contrat v1 inchangé)

| Scénario | Niveau | Attendu | Réel | Statut |
|---|---|---|---|---|
| Brouillon complet sans photo soumis | Banc complet (Contrôle réel) | `refuse` / `annonce_incomplete` | `refuse` / `annonce_incomplete` ; photo ajoutée sur l'article refusé puis resoumis : `publie` | Conforme |
| Photo déposée puis retirée avant soumission | Banc complet | `refuse` / `annonce_incomplete` | Idem | Conforme |
| Annonce sans photo dans chacune des 7 catégories | Unitaire Contrôle | Refusée incomplète | Refusée incomplète | Conforme |
| 1, 2, 10 photos | Unitaire Contrôle | Publiée | Publiée | Conforme |
| Sans photo et catégorie inconnue, prix hors plage, prix au-dessus de la vigilance | Unitaire Contrôle | Refusée incomplète | Refusée incomplète (la complétude prime) | Conforme |
| Vendeur suspendu sans photo | Unitaire Contrôle | (US-019 AC2) | `compte_non_autorise` conservé | Conforme |
| Exemple du contrat `annonce-complete.json` avec `photo_count` 0, puis 1 | Canal du Contrôle | Refusé incomplet / publié | Idem, résultat conforme au contrat v1 | Conforme |
| Toute réponse photo du banc | Banc complet | Contrat inchangé | Validée par `kin-openapi` contre `v1.openapi.yaml` | Conforme |

### AC3 — Photo de test avec GPS fictif : la photo servie ne contient plus aucune localisation

| Scénario | Niveau | Attendu | Réel | Statut |
|---|---|---|---|---|
| JPEG portant un APP1 EXIF avec IFD GPS fictif, publié, téléchargé par un visiteur via l'URL publique | Banc complet | Aucune métadonnée de localisation | JPEG servi sans segment APP1 à APP15, sans `Exif`, sans `GPS`, sans les octets de la latitude fictive | Conforme |
| PNG portant `eXIf` GPS et `tEXt` « lieu: 48.8566N 2.3522E », publié | Banc complet | Idem | Idem, commentaire de lieu absent | Conforme |
| Mêmes fichiers passés au nettoyage | Unitaire | Idem | Idem (précondition vérifiée : l'entrée porte bien la latitude fictive) | Conforme |

### AC4 — Article publié : photos servies par une URL présignée de courte durée

| Scénario | Niveau | Attendu | Réel | Statut |
|---|---|---|---|---|
| Visiteur sur la fiche publiée puis `GET /marche/v1/articles/{id}/photos` | Banc complet | URL présignée, courte durée | URL avec `X-Amz-Signature`, `X-Amz-Credential`, `X-Amz-Expires` entre 1 et 900 s ; `expireLe` dans le futur, à 15 min au plus ; l'URL sert la photo | Conforme |
| Fiche publiée | Vitest | Photo chargée depuis l'URL de l'API publique | `src` = URL fournie, une seule lecture | Conforme |
| Image en échec (URL expirée) | Vitest | Rafraîchie | Liste relue une fois, nouvelle URL affichée | Conforme |
| URL toujours en échec, 10 échecs successifs | Vitest | Sans boucle | Relectures ≤ 3 puis arrêt (aucune relecture pendant les 2 s suivantes), « Photo momentanément indisponible » | Conforme |
| API renvoyant toujours des URL déjà expirées, 3 s d'affichage | Vitest | Sans boucle | Relectures ≤ 3 | Conforme |
| Fiche publiée de démonstration | Playwright FR/EN | Photo servie | Photo chargée, texte alternatif non vide | Conforme |

Limite : SeaweedFS lancé sans identités ne vérifie ni la signature ni l'expiration (README, § S3 en local). Le refus d'une URL expirée ou altérée par le vrai bucket relève de l'inventaire après déploiement (AC8).

### AC5 — Contenu non image avec extension `.jpg`, GIF, WebP, SVG : refusé (415)

| Scénario | Niveau | Attendu | Réel | Statut |
|---|---|---|---|---|
| Texte nommé `photo.jpg`, GIF, WebP, SVG, polyglotte signature JPEG + HTML, polyglotte GIF + ZIP, fichier vide, JPEG tronqué, tous envoyés en `image/jpeg` | Banc complet | 415 | 415 `photo-non-prise-en-charge`, aucun objet écrit, liste vide | Conforme |
| Mêmes fichiers et signature PNG seule | Unitaire | Refus | `ErrUnsupportedPhoto` | Conforme |
| GIF, WebP, SVG, texte `.jpg` choisis dans l'espace vendeur | Vitest | Refus expliqué | Message « Ce fichier n'est pas une image JPEG ou PNG valide… », aucune photo | Conforme |
| Réponse 415 du serveur sur un JPEG d'apparence valide, puis JPEG valide | Vitest | Refus expliqué, nouvel essai | Message, puis photo affichée et message effacé | Conforme |
| Texte `.jpg` et SVG en démonstration | Playwright FR/EN | Refus expliqué | Message, aucune photo | Conforme |

Hors AC, constaté : un JPEG valide suivi d'un script (polyglotte qui reste une image) est accepté mais le script ne survit pas au ré-encodage.

### AC6 — Au-delà de la taille maximale : 413 ; bombe de décompression refusée sans saturer la mémoire

| Scénario | Niveau | Attendu | Réel | Statut |
|---|---|---|---|---|
| 5 Mio + 1 octet, puis 20 Mio | Banc complet | 413 | 413 `photo-trop-volumineuse`, aucun objet écrit | Conforme |
| Exactement 5 Mio (borne) | Banc complet | Accepté | 201 | Conforme |
| Bombe PNG 10 000 × 10 000 (≈ 100 Kio compressés), en-tête PNG 100 000 × 100 000, en-tête JPEG 65 535 × 65 535 | Banc complet | Refusée | 422 `photo-dimensions-excessives` en moins de 3 s, aucun objet, dépôt suivant accepté | Conforme |
| Bombe de 100 mégapixels | Unitaire | Sans saturer la mémoire | `ErrPhotoTooManyPixels` en ≈ 0,2 s, moins de 32 Mio alloués (un décodage complet en exigerait ≈ 100 Mio) | Conforme |
| 16 bombes simultanées | Unitaire | Idem | Toutes refusées, moins de 32 Mio alloués au total | Conforme |
| PNG de 70 000 × 1, plafond de pixels exact et dépassé d'un pixel | Unitaire | Limites | Côté excessif refusé ; plafond inclus accepté, plafond + 1 refusé | Conforme |
| Fuzzing du décodage, 60 s | Fuzz | Aucun plantage | 2,2 M exécutions, toute sortie acceptée est un JPEG lisible sous le plafond | Conforme |
| Réponses 413 et 422 ; fichier de plus de 5 Mo choisi | Vitest | Refus expliqué | « Fichier trop lourd : 5 Mo au plus. », « Image trop grande : 16 mégapixels au plus… », aucune photo | Conforme |

### AC7 — Photo d'un article non publié de A demandée par B ou un visiteur : 404

| Scénario | Niveau | Attendu | Réel | Statut |
|---|---|---|---|---|
| Brouillon de A avec photo : B lit la liste vendeur, le visiteur et B lisent la liste publique, B retire la photo, B dépose une photo | Banc complet | 404 | 404 pour chaque appel, Problem Details identique à celui d'un article inexistant, sans URL ni identifiant de photo ; photo de A intacte | Conforme |
| Article refusé de A | Banc complet | 404 public | 404 public ; A voit toujours sa photo | Conforme |
| Article non publié | Vitest | Aucune photo | Aucune image sur la réponse 404 | Conforme |

### AC8 — Bucket : accès public bloqué, chiffrement, TLS exigé ; CSP `img-src` ; aucune URL présignée journalisée

**Reporté** à l'inventaire après déploiement (DevOps/AWS, `tools/inventaire.sh`, CloudWatch). Partie vérifiable dans le code :

| Vérification | Moyen | Réel | Statut |
|---|---|---|---|
| Aucune URL présignée dans les journaux du service | Banc complet : journal de la Place de marché relu après tous les scénarios (précondition : des requêtes `/photos` y figurent) | Ni `X-Amz-Signature`, ni `X-Amz-Credential`, ni chemin d'objet du bucket | Conforme |
| Accès public bloqué, chiffrement, refus hors TLS | Lecture de `infra/persistant/photos.tf` | `aws_s3_bucket_public_access_block` (4 blocages), SSE `AES256`, politique `Deny` si `aws:SecureTransport = false`, propriété `BucketOwnerEnforced` | Conforme dans le code, à prouver sur le bucket |
| CSP `img-src` | Lecture de `web/nginx/security-headers.inc.template` | `img-src 'self' blob: ${COLLECTOR_PHOTOS_ORIGIN}` ; défaut du `Dockerfile` : `'self'` | Voir clarification C2 |
| Rôle d'exécution sur `photo` | Banc complet | `DELETE`, `TRUNCATE`, `UPDATE` de `cle_objet`, `article_id`, `largeur`, `hauteur`, `taille_octets`, `cree_le` refusés (42501) ; `UPDATE retiree_le` accepté | Conforme |

### AC9 — Dépôt opérable au clavier, chaque photo affichée a un texte alternatif

| Scénario | Niveau | Attendu | Réel | Statut |
|---|---|---|---|---|
| Tabulation jusqu'à « Choisir une photo », Entrée ou Espace | Vitest | Opérable | Le sélecteur de fichier s'ouvre (clic sur le champ fichier) ; le champ caché n'est pas tabulable | Conforme |
| Tabulation jusqu'à « Retirer la photo », Entrée | Vitest | Opérable | Photo retirée, focus conservé dans la page | Conforme |
| Parcours complet au clavier seul : sélecteur ouvert par Entrée, dépôt, retrait | Playwright FR/EN | Opérable | Focus visible à chaque tabulation, chaque étape aboutit | Conforme |
| Photo du brouillon (FR, EN), photo de la fiche publiée | Vitest | Texte alternatif | Reprend le titre de l'article | Conforme |
| Brouillon sans titre | Vitest | Texte alternatif | « Photo de l'article » | Conforme |
| Fiche publiée de démonstration | Playwright FR/EN | Texte alternatif | Toutes les images de la fiche ont un `alt` non vide | Conforme |
| axe-core : brouillon sans photo, brouillon avec photo (FR, EN) | Vitest | Aucune violation sérieuse ou critique | Aucune | Conforme |
| axe-core WCAG 2.1 A/AA : fiche publiée et dépôt avec photo | Playwright FR/EN | Aucune violation sérieuse ou critique | Aucune | Conforme |

## 4 bis. Lot 2 — corrections de la revue

### Majeure 2 — `GET /marche/v1/admin/articles/{id}/photos` (MD7)

| Scénario | Niveau | Attendu | Réel | Statut |
|---|---|---|---|---|
| Article `a_verifier` (catégorie inconnue du Contrôle réel) avec photo, jeton admin | Banc complet | 200 avec URL | 200, la photo déposée, URL présignée qui sert le JPEG | Conforme |
| Brouillon avec photo, jeton admin | Banc complet | 200 (tout état) | 200, une photo | Conforme |
| Vendeur propriétaire, autre vendeur, acheteur | Banc complet | 403 | 403 Problem Details, sans URL | Conforme |
| Sans jeton | Banc complet | 401 | 401 | Conforme |
| Article inconnu, jeton admin | Banc complet | 404 | 404 | Conforme |
| Toutes ces réponses | Banc complet | Contrat | Validées contre `v1.openapi.yaml` | Conforme |

### Majeure 3 — `config.LoadPhotos`

| Variable | Acceptées | Refusées (erreur nommant la variable) | Statut |
|---|---|---|---|
| Défauts | bucket seul posé : `us-east-1`, 10 min, 5 Mio, 16 Mpx, 1 photo, 2 décodages, adressage virtuel | — | Conforme |
| `PHOTOS_URL_DUREE` | `1s`, `15m` | `0s`, `999ms`, `15m1s`, `1h`, `-1m`, `dix` | Conforme |
| `PHOTOS_DECODAGES_SIMULTANES` | 1, 8 | 0, 9, -1, `deux` | Conforme |
| `PHOTOS_TAILLE_MAX_OCTETS` | 1024, 20 971 520 | 1023, 20 971 521, 0, `5Mo` | Conforme |
| `PHOTOS_PIXELS_MAX` | 1, 40 000 000 | 0, 40 000 001 | Conforme |
| `PHOTOS_PAR_ARTICLE_MAX` | 1, 10 | 0, 11 | Conforme |
| `PHOTOS_BUCKET` | `abc`, 63 caractères | absent, majuscules, 64 caractères, 2 caractères, tiret initial ou final, `_` | Conforme |
| `PHOTOS_S3_ADRESSE` | `http://127.0.0.1:8333`, `https://s3.local.test` | sans schéma, `ftp://`, `http://` sans hôte, chemin relatif | Conforme |
| `PHOTOS_S3_STYLE_CHEMIN` | `true` | `oui`, `2` | Conforme |
| Quatre variables invalides à la fois | — | Un seul message qui nomme les quatre | Conforme |

### Orientation EXIF appliquée

| Scénario | Niveau | Attendu | Réel | Statut |
|---|---|---|---|---|
| JPEG d'orientation 1 à 8, TIFF petit boutisme (II) et gros boutisme (MM) ; pixels stockés de sorte que l'image redressée fasse 48 × 32 à quatre quarts colorés | Unitaire (16 cas) | Sortie redressée (côtés rétablis pour 5 à 8), aucun EXIF | 48 × 32, chaque quart à sa couleur, aucun segment APPn, ni `Exif` ni `GPS` | Conforme |
| Orientation 0, 9, 65 535 | Unitaire | Ignorée | Dimensions d'origine, sortie propre | Conforme |
| PNG portant `eXIf` d'orientation 6 | Unitaire | PNG non pivoté | Dimensions stockées conservées | Conforme |
| TIFF tronqué, réduit à l'en-tête, offset d'IFD hors bornes, marque de boutisme inconnue, segment Exif vide, APP1 sans préfixe, nombre d'entrées démesuré | Unitaire | Sans effet ni plantage | Photo acceptée sans rotation, ou 415 | Conforme |
| Longueur d'APP1 déclarée 0, 1, 2, 8, 65 535 | Unitaire | Sans plantage | Accepté ou 415 | Conforme |
| `FuzzUS020LecteurEXIF` (segment APP1 et longueur déclarée fuzzés), 60 s | Fuzz | Aucun plantage ; sortie 24 × 16 ou 16 × 24, sans EXIF | 264 915 exécutions, 61 entrées intéressantes, vert | Conforme |
| JPEG d'orientation 6 déposé, publié, téléchargé par un visiteur | Banc complet | Redressé, sans métadonnée | 48 × 32, quarts à leur place, aucun segment APPn | Conforme |

### Journal `annonces.photo_orpheline`

| Scénario | Attendu | Réel | Statut |
|---|---|---|---|
| Objet écrit, enregistrement refusé (article non modifiable), suppression S3 en échec | Erreur métier rendue ; Warn `annonces.photo_orpheline` avec `article_id`, sans la clé | `ErrArticleNotEditable` seule (l'erreur S3 ne remonte pas), un Warn avec `article_id`, clé absente du journal, une tentative de suppression | Conforme |
| Enregistrement refusé (plafond), suppression réussie | Aucun journal orphelin | Une suppression, aucun journal | Conforme |
| Enregistrement réussi | Ni suppression ni journal | Idem | Conforme |
| Retrait dont la suppression échoue | Warn avec `photo_id`, sans la clé | Idem | Conforme |

### Front

| Scénario | Attendu | Réel | Statut |
|---|---|---|---|
| `photos.maxBytes` 2 Kio : JPEG de 3 Kio, puis de 2 Kio exactement | Refus local sans envoi ; borne incluse | « Fichier trop lourd… », aucun XHR ; 2 Kio envoyé et affiché | Conforme |
| `photos.maxPixels` 20 : JPEG 8 × 6 | Refus local sans envoi | « Image trop grande… », aucun XHR | Conforme |
| `photos.maxBytes` 1 Mio, réponse 413 | Message citant la limite configurée | « Fichier trop lourd : 1 Mo au plus. » | Conforme |
| `photos.maxPerArticle` 2 : une photo, puis deux | Dépôt proposé, consigne au pluriel ; puis consigne de retrait à la place du dépôt | Idem | Conforme |
| Sans `photos` dans `config.json`, une photo | Plafond 1 par défaut | « Une seule photo par article… », pas de dépôt | Conforme |
| `maxPerArticle` 0 ou 11, `maxBytes` 0, `maxPixels` négatif | Échec de démarrage | « L'application n'a pas pu démarrer » | Conforme |
| URL expirant dans 61 s | Relue avant expiration, image précédente affichée jusqu'au chargement de la nouvelle | Liste relue ; l'ancienne image reste visible (décorative, `aria-hidden`), la nouvelle à opacité 0 jusqu'à son `load`, puis seule nommée | Conforme |
| Carte d'examen admin avec photo | Lue sur la route admin avec le jeton, `alt` = titre | Idem | Conforme |
| Carte sans photo ; lecture en échec (500) | Photo masquée, carte examinable | Aucune image, « Publier » actif | Conforme |
| Image de la carte en échec répété | Masquée, sans boucle | 3 lectures au plus puis arrêt, image retirée | Conforme |
| Écran d'examen avec photo | axe sans violation sérieuse | Aucune | Conforme |
| 409 `article-non-modifiable` au dépôt | Message dédié | « Les photos de cet article ne sont pas modifiables… » (table des 409, couverte dès le lot 1) | Conforme |

## 5. Anomalies

Aucune.

## 6. Clarifications

| N° | Sujet | Constat | Question |
|---|---|---|---|
| C1 | AC2, « règles v2 » | L'US cite « règles v2 » ; les règles en vigueur avant US-020 étaient en v3 (US-019) ; le Contrôle publie désormais la version 4 (README de la Place de marché). | Confirmer que « v2 » dans l'AC signifie « version suivante » et que la v4 est attendue. Tests alignés sur v4. |
| C2 | AC8, « `img-src` limité au bucket » | **Close** : acceptée par la revue (`blob:` nécessaire à l'aperçu, hôte exact du bucket posé au déploiement, documenté dans `web/README.md`). Reste à constater dans l'inventaire AC8. | — |

## 7. Observations (hors AC)

| N° | Constat |
|---|---|
| O1 | **Close** au lot 2 : `web/README.md` décrit désormais les photos (§ Photos, limites `photos.*`, CSP). |
| O2 | `jsdom` n'implémente ni `Blob.arrayBuffer` ni `URL.createObjectURL` ; le harnais Vitest les complète. Sans ce complément, le dépôt échoue en test avec « L'envoi a échoué », ce qui ne se produit pas dans un navigateur (parcours Playwright verts). |

## 8. Verdict

Tous les AC testables (AC1 à AC7, AC9) sont couverts et conformes, lot 2 compris ; aucune anomalie. Les deux trous de test de la revue (majeures 2 et 3) sont comblés, l'orientation EXIF et le journal d'objet orphelin sont prouvés, la mineure 6 est traitée (§ 3 bis). AC8 : partie code conforme ; inventaire du bucket, de la CSP déployée et des journaux CloudWatch reporté après déploiement.
