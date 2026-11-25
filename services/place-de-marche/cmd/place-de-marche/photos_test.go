package main

import (
	"bytes"
	"encoding/binary"
	"io"
	"net/http"
	"net/url"
	"slices"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"collector/libs/platform/database"
	"collector/services/place-de-marche/internal/annonces/photo/photoessai"
	"collector/services/place-de-marche/internal/migrations"
)

const (
	tailleMaxParDefaut        = 5 << 20
	dureeMaxURL               = 15 * time.Minute
	delaiDuControle           = 30 * time.Second
	typePhotoNonPriseEnCharge = "urn:collector:probleme:photo-non-prise-en-charge"
	typePhotoTropVolumineuse  = "urn:collector:probleme:photo-trop-volumineuse"
	typeDimensionsExcessives  = "urn:collector:probleme:photo-dimensions-excessives"
	typeLimitePhotos          = "urn:collector:probleme:photos-limite-atteinte"
	typeArticleNonModifiable  = "urn:collector:probleme:article-non-modifiable"
	nomDeFichierDuClient      = "maison-de-vacances-gps.jpg"
)

type bancPhotos struct {
	*environnement
	base    basePostgreSQL
	bucket  bucketDeTest
	journal *journalPartage
	pool    *pgxpool.Pool
}

func demarrerBancPhotos(t *testing.T) *bancPhotos {
	t.Helper()
	ctx := contexteBorne(t, 3*time.Minute)
	base := amorcer(ctx, t, demarrerPostgreSQL(t))
	if err := migrations.Apply(ctx, journalMuet(), base.migration); err != nil {
		t.Fatalf("migrations : %v", err)
	}
	canal := demarrerRabbitMQ(t)
	bucket := nouveauBucket(t)
	e, journal := lancerPlaceDeMarcheSurLeBucket(t, canal, base.execution, nouvelEmetteurDeJetons(t), bucket)
	lancerControle(t, construireControle(t), canal)
	pool, err := database.Connect(ctx, base.execution)
	if err != nil {
		t.Fatalf("connexion en %s : %v", roleExecution, err)
	}
	t.Cleanup(pool.Close)
	return &bancPhotos{environnement: e, base: base, bucket: bucket, journal: journal, pool: pool}
}

func (b *bancPhotos) brouillon(t *testing.T, jeton, titre string) articleRecu {
	t.Helper()
	return b.creer(t, jeton, corpsComplet(titre, categorieSure, 4500, 600))
}

func (b *bancPhotos) deposer(t *testing.T, jeton, id string, contenu []byte) photoRecue {
	t.Helper()
	var recue photoRecue
	b.deposerPhoto(t, jeton, id, contenu).exiger(t, http.StatusCreated).lire(t, &recue)
	return recue
}

func (b *bancPhotos) attendreEtatDe(t *testing.T, jeton, id, etat string) articleRecu {
	t.Helper()
	var dernier articleRecu
	attendre(t, delaiDuControle, "article "+id+" "+etat, func() bool {
		dernier = b.relire(t, jeton, id)
		return dernier.Etat == etat
	})
	return dernier
}

func (b *bancPhotos) publierAvec(t *testing.T, jeton, titre string, contenu []byte) articleRecu {
	t.Helper()
	article := b.brouillon(t, jeton, titre)
	b.deposer(t, jeton, article.ID, contenu)
	b.soumettreAvecSucces(t, jeton, article)
	return b.attendreEtatDe(t, jeton, article.ID, etatPublie)
}

func (b *bancPhotos) photosPubliques(t *testing.T, id string) reponse {
	t.Helper()
	return b.envoyer(t, http.MethodGet, cheminPhotosPubliques(id), "")
}

func (b *bancPhotos) photosDAutrui(t *testing.T, jeton, id string) reponse {
	t.Helper()
	return b.envoyer(t, http.MethodGet, cheminPhotosVendeur(id), jeton)
}

func (b *bancPhotos) retirer(t *testing.T, jeton, id, photoID string) reponse {
	t.Helper()
	return b.envoyer(t, http.MethodDelete, cheminPhotosVendeur(id)+"/"+photoID, jeton)
}

func telecharger(t *testing.T, adresse string) []byte {
	t.Helper()
	requete, err := http.NewRequestWithContext(t.Context(), http.MethodGet, adresse, nil)
	if err != nil {
		t.Fatalf("requête de la photo : %v", err)
	}
	reponse, err := http.DefaultClient.Do(requete)
	if err != nil {
		t.Fatalf("téléchargement de la photo : %v", err)
	}
	defer func() { _ = reponse.Body.Close() }()
	contenu, err := io.ReadAll(reponse.Body)
	if err != nil || reponse.StatusCode != http.StatusOK {
		t.Fatalf("téléchargement de la photo : statut %d : %v", reponse.StatusCode, err)
	}
	return contenu
}

func nouvellesCles(avant, apres []string) []string {
	var nouvelles []string
	for _, cle := range apres {
		if !slices.Contains(avant, cle) {
			nouvelles = append(nouvelles, cle)
		}
	}
	return nouvelles
}

func exigerSansMetadonnee(t *testing.T, contenu []byte) {
	t.Helper()
	if !photoessai.EstUnJPEG(contenu) {
		t.Fatal("photo servie autre qu'un JPEG")
	}
	for _, m := range photoessai.SegmentsAvantImage(contenu) {
		if m >= 0xE1 && m <= 0xEF {
			t.Fatalf("segment de métadonnées APP%d servi", m-0xE0)
		}
	}
	for _, trace := range [][]byte{[]byte("Exif"), []byte("GPS"), photoessai.LatitudeFictive(), []byte(photoessai.CommentaireDeLieu)} {
		if bytes.Contains(contenu, trace) {
			t.Fatalf("la photo servie contient encore %q", trace)
		}
	}
}

func exigerProblemePhoto(t *testing.T, r reponse, statut int, typ string) {
	t.Helper()
	exigerTypeDeProbleme(t, r.probleme(t, statut), typ)
}

func TestUS020PhotosSurPostgreSQLSeaweedFSEtControle(t *testing.T) {
	b := demarrerBancPhotos(t)
	t.Run("AC1 JPEG déposé sur un brouillon ré-encodé, stocké sous un nom généré et affiché", b.ac1JPEGDeposeReencodeStockeEtAffiche)
	t.Run("AC1 PNG déposé stocké en JPEG ré-encodé", b.ac1PNGDeposeStockeEnJPEG)
	t.Run("AC1 noms d'objets générés par le serveur et non devinables", b.ac1NomsDObjetsNonDevinables)
	t.Run("AC1 seconde photo refusée au-delà du plafond, rien d'écrit", b.ac1PlafondDUnePhoto)
	t.Run("AC1 retrait de la photo : objet supprimé, liste vide, nouveau dépôt possible", b.ac1RetraitDeLaPhoto)
	t.Run("AC1 dépôt refusé sur un article publié", b.ac1DepotRefuseSurArticlePublie)
	t.Run("AC2 article sans photo refusé pour annonce incomplète puis publié avec une photo", b.ac2SansPhotoRefusePuisPublie)
	t.Run("AC2 photo retirée avant soumission refusée pour annonce incomplète", b.ac2PhotoRetireeAvantSoumissionRefuse)
	t.Run("AC3 photo JPEG avec GPS publiée sans aucune métadonnée de localisation", b.ac3JPEGAvecGPSPublieSansLocalisation)
	t.Run("AC3 photo PNG avec GPS publiée sans aucune métadonnée de localisation", b.ac3PNGAvecGPSPublieSansLocalisation)
	t.Run("Orientation EXIF 6 d'un JPEG de téléphone : photo publiée redressée, sans EXIF", b.orientationAppliqueeALaPhotoPubliee)
	t.Run("AC4 fiche publiée servie par une URL présignée de courte durée", b.ac4URLPresigneeDeCourteDuree)
	t.Run("AC5 fichiers piégés ou hors JPEG PNG refusés en 415 sans rien stocker", b.ac5FichiersPiegesRefuses)
	t.Run("AC6 fichier au-delà de la taille maximale refusé en 413", b.ac6TailleMaximale)
	t.Run("AC6 bombe de décompression refusée en 422, service toujours disponible", b.ac6BombeRefusee)
	t.Run("AC7 photo d'un article non publié introuvable pour un autre vendeur ou un visiteur", b.ac7PhotoNonPublieeIntrouvable)
	t.Run("AC7 article refusé : photos publiques introuvables, visibles du seul vendeur", b.ac7EtatsNonPublies)
	t.Run("MD7 route admin : photos d'un article à vérifier servies à l'admin seul", b.md7AdminLitLesPhotosDUnArticleAVerifier)
	t.Run("MD7 route admin : vendeur propriétaire, autre vendeur et acheteur refusés en 403, visiteur en 401", b.md7RouteAdminRefuseeAuxAutresRoles)
	t.Run("MD7 route admin : article inconnu introuvable", b.md7RouteAdminArticleInconnu)
	t.Run("Droits du rôle d'exécution sur photo : ni DELETE ni UPDATE hors retiree_le", b.droitsDuRoleDExecution)
	t.Run("AC8 aucune URL présignée dans les journaux du service", b.ac8AucuneURLPresigneeJournalisee)
	t.Run("Réponses photos conformes au contrat OpenAPI", b.reponsesConformesAuContrat)
}

func (b *bancPhotos) ac1JPEGDeposeReencodeStockeEtAffiche(t *testing.T) {
	jeton := b.vendeur()
	article := b.brouillon(t, jeton, "Figurine Gaston")
	original := photoessai.JPEG(64, 48)
	avant := b.bucket.cles(t)
	requete := b.envoyer(t, http.MethodPost, cheminPhotosVendeur(article.ID), jeton, corpsPhoto(original, "application/octet-stream"), avecNomDeFichier(nomDeFichierDuClient))
	var recue photoRecue
	requete.exiger(t, http.StatusCreated).lire(t, &recue)
	if _, err := uuid.Parse(recue.ID); err != nil || recue.Largeur != 64 || recue.Hauteur != 48 || recue.URL == "" {
		t.Fatalf("photo reçue %+v", recue)
	}
	cles := nouvellesCles(avant, b.bucket.cles(t))
	if len(cles) != 1 {
		t.Fatalf("%d objet(s) ajouté(s) au bucket, 1 attendu : %v", len(cles), cles)
	}
	for _, fragment := range []string{article.ID, recue.ID, "maison", "vacances", "gps"} {
		if strings.Contains(strings.ToLower(cles[0]), fragment) {
			t.Fatalf("clé %q dérivée de %q", cles[0], fragment)
		}
	}
	stocke, typ, present := b.bucket.objet(t, cles[0])
	if !present || typ != typePhotoJPEG || bytes.Equal(stocke, original) {
		t.Fatalf("objet présent %v, type %q, identique à l'original %v", present, typ, bytes.Equal(stocke, original))
	}
	exigerSansMetadonnee(t, stocke)
	liste := b.photosDuVendeur(t, jeton, article.ID)
	if len(liste.Elements) != 1 || liste.Elements[0].ID != recue.ID {
		t.Fatalf("espace du vendeur : %+v", liste)
	}
	if !bytes.Equal(telecharger(t, liste.Elements[0].URL), stocke) {
		t.Fatal("l'URL affichée ne sert pas la photo stockée")
	}
}

func avecNomDeFichier(nom string) option {
	return func(e *envoi) {
		e.nomDeFichier = nom
	}
}

func (b *bancPhotos) ac1PNGDeposeStockeEnJPEG(t *testing.T) {
	jeton := b.vendeur()
	article := b.brouillon(t, jeton, "Vinyle rare")
	avant := b.bucket.cles(t)
	recue := b.deposer(t, jeton, article.ID, photoessai.PNGTransparent(40, 20))
	cles := nouvellesCles(avant, b.bucket.cles(t))
	if len(cles) != 1 || recue.Largeur != 40 || recue.Hauteur != 20 {
		t.Fatalf("objets ajoutés %v, photo %+v", cles, recue)
	}
	stocke, typ, _ := b.bucket.objet(t, cles[0])
	if typ != typePhotoJPEG {
		t.Fatalf("type stocké %q", typ)
	}
	exigerSansMetadonnee(t, stocke)
}

func (b *bancPhotos) ac1NomsDObjetsNonDevinables(t *testing.T) {
	jeton := b.vendeur()
	avant := b.bucket.cles(t)
	for i := range 5 {
		b.deposer(t, jeton, b.brouillon(t, jeton, "Montre "+strconv.Itoa(i)).ID, photoessai.JPEG(16, 16))
	}
	cles := nouvellesCles(avant, b.bucket.cles(t))
	if len(cles) != 5 {
		t.Fatalf("%d objets ajoutés, 5 attendus", len(cles))
	}
	prefixe, suffixe := cles[0], cles[0]
	for _, cle := range cles[1:] {
		for !strings.HasPrefix(cle, prefixe) {
			prefixe = prefixe[:len(prefixe)-1]
		}
		for !strings.HasSuffix(cle, suffixe) {
			suffixe = suffixe[1:]
		}
	}
	for _, cle := range cles {
		variable := strings.TrimSuffix(strings.TrimPrefix(cle, prefixe), suffixe)
		if len(variable) < 20 {
			t.Fatalf("partie aléatoire de %q trop courte (%q) : devinable", cle, variable)
		}
	}
}

func (b *bancPhotos) ac1PlafondDUnePhoto(t *testing.T) {
	jeton := b.vendeur()
	article := b.brouillon(t, jeton, "Figurine Lagaffe")
	b.deposer(t, jeton, article.ID, photoessai.JPEG(16, 16))
	avant := b.bucket.cles(t)
	exigerProblemePhoto(t, b.deposerPhoto(t, jeton, article.ID, photoessai.JPEG(20, 20)), http.StatusConflict, typeLimitePhotos)
	if apres := b.bucket.cles(t); len(nouvellesCles(avant, apres)) != 0 || len(apres) != len(avant) {
		t.Fatalf("bucket modifié par un dépôt refusé : %v", nouvellesCles(avant, apres))
	}
	if n := len(b.photosDuVendeur(t, jeton, article.ID).Elements); n != 1 {
		t.Fatalf("%d photos après un dépôt refusé", n)
	}
}

func (b *bancPhotos) ac1RetraitDeLaPhoto(t *testing.T) {
	jeton := b.vendeur()
	article := b.brouillon(t, jeton, "Pièce de 1 franc")
	avant := b.bucket.cles(t)
	recue := b.deposer(t, jeton, article.ID, photoessai.JPEG(16, 16))
	cle := nouvellesCles(avant, b.bucket.cles(t))[0]
	b.retirer(t, jeton, article.ID, recue.ID).exiger(t, http.StatusNoContent)
	if _, _, present := b.bucket.objet(t, cle); present {
		t.Fatalf("objet %s toujours présent après retrait", cle)
	}
	if n := len(b.photosDuVendeur(t, jeton, article.ID).Elements); n != 0 {
		t.Fatalf("%d photo(s) listée(s) après retrait", n)
	}
	b.retirer(t, jeton, article.ID, recue.ID).probleme(t, http.StatusNotFound)
	b.retirer(t, jeton, article.ID, uuid.NewString()).probleme(t, http.StatusNotFound)
	b.deposer(t, jeton, article.ID, photoessai.JPEG(16, 16))
}

func (b *bancPhotos) ac1DepotRefuseSurArticlePublie(t *testing.T) {
	jeton := b.vendeur()
	publie := b.publierAvec(t, jeton, "Figurine publiée", photoessai.JPEG(16, 16))
	avant := b.bucket.cles(t)
	r := b.deposerPhoto(t, jeton, publie.ID, photoessai.JPEG(16, 16))
	if r.statut != http.StatusConflict {
		t.Fatalf("dépôt sur un article publié : %d %s", r.statut, r.corps)
	}
	exigerProblemePhoto(t, r, http.StatusConflict, typeArticleNonModifiable)
	if len(nouvellesCles(avant, b.bucket.cles(t))) != 0 {
		t.Fatal("objet écrit pour un dépôt refusé")
	}
}

func (b *bancPhotos) ac2SansPhotoRefusePuisPublie(t *testing.T) {
	jeton := b.vendeur()
	article := b.brouillon(t, jeton, "Carte Pokémon sans photo")
	b.soumettre(t, jeton, article.ID, article.Revision).exiger(t, http.StatusAccepted)
	refuse := b.attendreEtatDe(t, jeton, article.ID, etatRefuse)
	exigerEtat(t, refuse, etatRefuse, motifIncomplet)
	b.deposer(t, jeton, article.ID, photoessai.JPEG(16, 16))
	b.soumettreAvecSucces(t, jeton, b.relire(t, jeton, article.ID))
	exigerEtat(t, b.attendreEtatDe(t, jeton, article.ID, etatPublie), etatPublie, "")
}

func (b *bancPhotos) ac2PhotoRetireeAvantSoumissionRefuse(t *testing.T) {
	jeton := b.vendeur()
	article := b.brouillon(t, jeton, "Figurine retirée")
	recue := b.deposer(t, jeton, article.ID, photoessai.JPEG(16, 16))
	b.retirer(t, jeton, article.ID, recue.ID).exiger(t, http.StatusNoContent)
	b.soumettre(t, jeton, article.ID, article.Revision).exiger(t, http.StatusAccepted)
	exigerEtat(t, b.attendreEtatDe(t, jeton, article.ID, etatRefuse), etatRefuse, motifIncomplet)
}

func (b *bancPhotos) photoServieAuVisiteur(t *testing.T, id string) []byte {
	t.Helper()
	var liste listePhotos
	b.photosPubliques(t, id).exiger(t, http.StatusOK).lire(t, &liste)
	if len(liste.Elements) != 1 {
		t.Fatalf("%d photo(s) publique(s), 1 attendue", len(liste.Elements))
	}
	return telecharger(t, liste.Elements[0].URL)
}

func (b *bancPhotos) ac3JPEGAvecGPSPublieSansLocalisation(t *testing.T) {
	jeton := b.vendeur()
	publie := b.publierAvec(t, jeton, "Figurine géolocalisée", photoessai.JPEGAvecGPS())
	exigerSansMetadonnee(t, b.photoServieAuVisiteur(t, publie.ID))
}

func (b *bancPhotos) ac3PNGAvecGPSPublieSansLocalisation(t *testing.T) {
	jeton := b.vendeur()
	publie := b.publierAvec(t, jeton, "Vinyle géolocalisé", photoessai.PNGAvecGPS())
	exigerSansMetadonnee(t, b.photoServieAuVisiteur(t, publie.ID))
}

func (b *bancPhotos) orientationAppliqueeALaPhotoPubliee(t *testing.T) {
	jeton := b.vendeur()
	publie := b.publierAvec(t, jeton, "Montre en portrait", photoessai.JPEGOriente(6, binary.LittleEndian))
	servie := b.photoServieAuVisiteur(t, publie.ID)
	quarts, cadre, err := photoessai.QuartsDe(servie)
	if err != nil || cadre.Dx() != photoessai.LargeurRedressee || cadre.Dy() != photoessai.HauteurRedressee {
		t.Fatalf("photo servie %v : %v", cadre, err)
	}
	for rang, attendue := range photoessai.CouleursDesQuarts {
		if !photoessai.Proche(quarts[rang], attendue) {
			t.Fatalf("quart %d : %v, attendu %v", rang, quarts[rang], attendue)
		}
	}
	exigerSansMetadonnee(t, servie)
}

func (b *bancPhotos) ac4URLPresigneeDeCourteDuree(t *testing.T) {
	jeton := b.vendeur()
	publie := b.publierAvec(t, jeton, "Montre ancienne", photoessai.JPEG(32, 32))
	b.fiche(t, publie.ID).exiger(t, http.StatusOK)
	debut := time.Now()
	var liste listePhotos
	b.photosPubliques(t, publie.ID).exiger(t, http.StatusOK).lire(t, &liste)
	if len(liste.Elements) != 1 {
		t.Fatalf("%d photo(s) sur la fiche publiée", len(liste.Elements))
	}
	photo := liste.Elements[0]
	adresse, err := url.Parse(photo.URL)
	if err != nil {
		t.Fatalf("URL illisible : %v", err)
	}
	requete := adresse.Query()
	duree, err := strconv.Atoi(requete.Get("X-Amz-Expires"))
	if requete.Get("X-Amz-Signature") == "" || requete.Get("X-Amz-Credential") == "" || err != nil {
		t.Fatalf("URL non présignée : %s", photo.URL)
	}
	if duree <= 0 || time.Duration(duree)*time.Second > dureeMaxURL {
		t.Fatalf("URL valable %d s, au plus %s attendu", duree, dureeMaxURL)
	}
	expire, err := time.Parse(time.RFC3339, photo.ExpireLe)
	if err != nil || !expire.After(debut) || expire.Sub(debut) > dureeMaxURL+time.Minute {
		t.Fatalf("expireLe %q pour une demande à %s : %v", photo.ExpireLe, debut.Format(time.RFC3339), err)
	}
	if !photoessai.EstUnJPEG(telecharger(t, photo.URL)) {
		t.Fatal("l'URL présignée ne sert pas la photo")
	}
}

func (b *bancPhotos) ac5FichiersPiegesRefuses(t *testing.T) {
	jeton := b.vendeur()
	article := b.brouillon(t, jeton, "Article piégé")
	cas := map[string][]byte{
		"texte nommé .jpg":                  photoessai.FauxJPEG(),
		"GIF":                               photoessai.GIF(),
		"WebP":                              photoessai.WebP(),
		"SVG":                               photoessai.SVG(),
		"polyglotte signature JPEG et HTML": photoessai.PolyglotteJPEGHTML(),
		"polyglotte GIF et ZIP":             photoessai.PolyglotteGIFZip(),
		"fichier vide":                      {},
		"JPEG tronqué":                      photoessai.JPEGTronque(),
	}
	avant := b.bucket.cles(t)
	for nom, contenu := range cas {
		t.Run(nom, func(t *testing.T) {
			r := b.envoyer(t, http.MethodPost, cheminPhotosVendeur(article.ID), jeton, corpsPhoto(contenu, typePhotoJPEG), avecNomDeFichier("photo.jpg"))
			exigerProblemePhoto(t, r, http.StatusUnsupportedMediaType, typePhotoNonPriseEnCharge)
		})
	}
	if len(nouvellesCles(avant, b.bucket.cles(t))) != 0 || len(b.photosDuVendeur(t, jeton, article.ID).Elements) != 0 {
		t.Fatal("un fichier refusé a été stocké")
	}
}

func jpegDeTaille(taille int) []byte {
	contenu := photoessai.JPEG(16, 16)
	return append(contenu, make([]byte, taille-len(contenu))...)
}

func (b *bancPhotos) ac6TailleMaximale(t *testing.T) {
	jeton := b.vendeur()
	avant := b.bucket.cles(t)
	refuse := b.brouillon(t, jeton, "Photo trop lourde")
	exigerProblemePhoto(t, b.deposerPhoto(t, jeton, refuse.ID, jpegDeTaille(tailleMaxParDefaut+1)), http.StatusRequestEntityTooLarge, typePhotoTropVolumineuse)
	exigerProblemePhoto(t, b.deposerPhoto(t, jeton, refuse.ID, jpegDeTaille(4*tailleMaxParDefaut)), http.StatusRequestEntityTooLarge, typePhotoTropVolumineuse)
	if len(nouvellesCles(avant, b.bucket.cles(t))) != 0 {
		t.Fatal("objet écrit pour un fichier trop volumineux")
	}
	b.deposer(t, jeton, b.brouillon(t, jeton, "Photo à la taille maximale").ID, jpegDeTaille(tailleMaxParDefaut))
}

func (b *bancPhotos) ac6BombeRefusee(t *testing.T) {
	jeton := b.vendeur()
	article := b.brouillon(t, jeton, "Bombe")
	avant := b.bucket.cles(t)
	cas := map[string][]byte{
		"PNG de 100 mégapixels compressé":  photoessai.BombePNG(10_000, 10_000),
		"en-tête PNG de 100 000 x 100 000": photoessai.EnTetePNGGeant(100_000, 100_000),
		"en-tête JPEG de 65 535 x 65 535":  photoessai.JPEGAuxDimensionsAnnoncees(65_535, 65_535),
	}
	for nom, contenu := range cas {
		t.Run(nom, func(t *testing.T) {
			debut := time.Now()
			exigerProblemePhoto(t, b.deposerPhoto(t, jeton, article.ID, contenu), http.StatusUnprocessableEntity, typeDimensionsExcessives)
			if duree := time.Since(debut); duree > 3*time.Second {
				t.Fatalf("bombe refusée en %s", duree)
			}
		})
	}
	if len(nouvellesCles(avant, b.bucket.cles(t))) != 0 {
		t.Fatal("objet écrit pour une bombe")
	}
	b.deposer(t, jeton, article.ID, photoessai.JPEG(16, 16))
}

func (b *bancPhotos) ac7PhotoNonPublieeIntrouvable(t *testing.T) {
	vendeurA, vendeurB := b.vendeur(), b.vendeur()
	article := b.brouillon(t, vendeurA, "Article privé de A")
	photo := b.deposer(t, vendeurA, article.ID, photoessai.JPEG(16, 16))
	inexistant := uuid.NewString()
	reference := b.photosDAutrui(t, vendeurB, inexistant).probleme(t, http.StatusNotFound)
	for nom, r := range map[string]reponse{
		"vendeur B, liste du vendeur":  b.photosDAutrui(t, vendeurB, article.ID),
		"visiteur, liste publique":     b.photosPubliques(t, article.ID),
		"vendeur B, liste publique":    b.envoyer(t, http.MethodGet, cheminPhotosPubliques(article.ID), vendeurB),
		"vendeur B, retrait":           b.retirer(t, vendeurB, article.ID, photo.ID),
		"vendeur B, dépôt":             b.deposerPhoto(t, vendeurB, article.ID, photoessai.JPEG(16, 16)),
		"visiteur, article inexistant": b.photosPubliques(t, inexistant),
	} {
		t.Run(nom, func(t *testing.T) {
			p := r.probleme(t, http.StatusNotFound)
			if p.Type != reference.Type || p.Title != reference.Title || p.Detail != reference.Detail {
				t.Fatalf("404 distinguable d'un article inexistant : %+v contre %+v", p, reference)
			}
			for _, fuite := range []string{"X-Amz", photo.ID, "url"} {
				if strings.Contains(string(r.corps), fuite) {
					t.Fatalf("la réponse divulgue %q : %s", fuite, r.corps)
				}
			}
		})
	}
	if liste := b.photosDuVendeur(t, vendeurA, article.ID); len(liste.Elements) != 1 || liste.Elements[0].ID != photo.ID {
		t.Fatalf("photo de A altérée par B : %+v", liste)
	}
}

func (b *bancPhotos) ac7EtatsNonPublies(t *testing.T) {
	jeton := b.vendeur()
	refuse := b.brouillon(t, jeton, "Refusé")
	b.deposer(t, jeton, refuse.ID, photoessai.JPEG(16, 16))
	b.modifier(t, jeton, refuse.ID, `{"revision":1,"description":""}`).exiger(t, http.StatusOK)
	soumis := b.soumettreAvecSucces(t, jeton, b.relire(t, jeton, refuse.ID))
	b.attendreEtatDe(t, jeton, soumis.ID, etatRefuse)
	b.photosPubliques(t, refuse.ID).probleme(t, http.StatusNotFound)
	if n := len(b.photosDuVendeur(t, jeton, refuse.ID).Elements); n != 1 {
		t.Fatalf("le vendeur voit %d photo(s) de son article refusé", n)
	}
}

func cheminPhotosAdmin(id string) string {
	return "/marche/v1/admin/articles/" + id + prefixeDesPhotosURL
}

func (b *bancPhotos) articleAVerifierAvecPhoto(t *testing.T, jeton string) (articleRecu, photoRecue) {
	t.Helper()
	if _, err := b.base.maitre.Exec(t.Context(), `INSERT INTO categorie (id, libelle) VALUES ($1, 'Timbres') ON CONFLICT DO NOTHING`, categorieNonReconnue); err != nil {
		t.Fatalf("catégorie inconnue du Contrôle : %v", err)
	}
	article := b.creer(t, jeton, corpsComplet("Timbre à examiner", categorieNonReconnue, 900, 0))
	photo := b.deposer(t, jeton, article.ID, photoessai.JPEG(16, 16))
	b.soumettreAvecSucces(t, jeton, article)
	return b.attendreEtatDe(t, jeton, article.ID, etatAVerifier), photo
}

func (b *bancPhotos) md7AdminLitLesPhotosDUnArticleAVerifier(t *testing.T) {
	jeton := b.vendeur()
	article, photo := b.articleAVerifierAvecPhoto(t, jeton)
	admin := b.administrateur()
	var liste listePhotos
	b.envoyer(t, http.MethodGet, cheminPhotosAdmin(article.ID), admin.jeton).exiger(t, http.StatusOK).lire(t, &liste)
	if len(liste.Elements) != 1 || liste.Elements[0].ID != photo.ID || !strings.Contains(liste.Elements[0].URL, "X-Amz-Signature") {
		t.Fatalf("photos de l'article à vérifier vues par l'admin : %+v", liste)
	}
	if !photoessai.EstUnJPEG(telecharger(t, liste.Elements[0].URL)) {
		t.Fatal("l'URL remise à l'admin ne sert pas la photo")
	}
	brouillon := b.brouillon(t, jeton, "Brouillon examiné")
	b.deposer(t, jeton, brouillon.ID, photoessai.JPEG(16, 16))
	b.envoyer(t, http.MethodGet, cheminPhotosAdmin(brouillon.ID), admin.jeton).exiger(t, http.StatusOK).lire(t, &liste)
	if len(liste.Elements) != 1 {
		t.Fatalf("admin, brouillon : %d photo(s)", len(liste.Elements))
	}
}

func (b *bancPhotos) md7RouteAdminRefuseeAuxAutresRoles(t *testing.T) {
	jeton := b.vendeur()
	article, _ := b.articleAVerifierAvecPhoto(t, jeton)
	for nom, cas := range map[string]struct {
		jeton  string
		statut int
	}{
		"vendeur propriétaire": {jeton, http.StatusForbidden},
		"autre vendeur":        {b.vendeur(), http.StatusForbidden},
		"acheteur":             {bearer(b.jetons.acheteur(uuid.NewString())), http.StatusForbidden},
		"visiteur":             {"", http.StatusUnauthorized},
	} {
		t.Run(nom, func(t *testing.T) {
			r := b.envoyer(t, http.MethodGet, cheminPhotosAdmin(article.ID), cas.jeton)
			r.probleme(t, cas.statut)
			if strings.Contains(string(r.corps), "X-Amz") {
				t.Fatalf("la réponse refusée divulgue une URL : %s", r.corps)
			}
		})
	}
}

func (b *bancPhotos) md7RouteAdminArticleInconnu(t *testing.T) {
	b.envoyer(t, http.MethodGet, cheminPhotosAdmin(uuid.NewString()), b.administrateur().jeton).probleme(t, http.StatusNotFound)
}

func (b *bancPhotos) droitsDuRoleDExecution(t *testing.T) {
	exigerRefusDeDroit(t, b.pool, "DELETE FROM photo")
	exigerRefusDeDroit(t, b.pool, "TRUNCATE photo")
	exigerRefusDeDroit(t, b.pool, "UPDATE photo SET cle_objet = cle_objet")
	exigerRefusDeDroit(t, b.pool, "UPDATE photo SET article_id = article_id")
	exigerRefusDeDroit(t, b.pool, "UPDATE photo SET largeur = largeur, hauteur = hauteur")
	exigerRefusDeDroit(t, b.pool, "UPDATE photo SET taille_octets = taille_octets")
	exigerRefusDeDroit(t, b.pool, "UPDATE photo SET cree_le = cree_le")
	if _, err := b.pool.Exec(t.Context(), "UPDATE photo SET retiree_le = retiree_le WHERE false"); err != nil {
		t.Fatalf("UPDATE de retiree_le refusé au rôle d'exécution : %v", err)
	}
}

func (b *bancPhotos) ac8AucuneURLPresigneeJournalisee(t *testing.T) {
	if !b.journal.contient(prefixeDesPhotosURL) {
		t.Fatal("aucune requête de photo journalisée : vérification sans objet")
	}
	for _, trace := range []string{"X-Amz-Signature", "X-Amz-Credential", "x-amz-signature", b.bucket.nom + "/photos/"} {
		if b.journal.contient(trace) {
			t.Errorf("le journal du service contient %q", trace)
		}
	}
}

func (b *bancPhotos) reponsesConformesAuContrat(t *testing.T) {
	routeur := chargerContrat(t, b.adresse)
	for _, ech := range b.historique() {
		if strings.Contains(ech.requete.URL.Path, prefixeDesPhotosURL) {
			verifierEchange(t, routeur, ech)
		}
	}
}
