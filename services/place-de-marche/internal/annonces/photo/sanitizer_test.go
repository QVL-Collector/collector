package photo_test

import (
	"bytes"
	"errors"
	"image/jpeg"
	"runtime"
	"testing"
	"time"

	"collector/services/place-de-marche/internal/annonces"
	"collector/services/place-de-marche/internal/annonces/photo"
	"collector/services/place-de-marche/internal/annonces/photo/photoessai"
)

const (
	pixelsMaxParDefaut  = 16_000_000
	plafondMemoireBombe = 32 << 20
	delaiMaxBombe       = 2 * time.Second
)

func nettoyeur(t *testing.T, pixelsMax int) *photo.Sanitizer {
	t.Helper()
	s, err := photo.NewSanitizer(photo.Limits{MaxPixels: pixelsMax, ConcurrentDecodes: 2})
	if err != nil {
		t.Fatalf("nettoyeur : %v", err)
	}
	return s
}

func nettoyer(t *testing.T, pixelsMax int, brut []byte) (annonces.EncodedPhoto, error) {
	t.Helper()
	return nettoyeur(t, pixelsMax).Sanitize(t.Context(), brut)
}

func exigerJPEGPropre(t *testing.T, sortie annonces.EncodedPhoto, largeur, hauteur int) {
	t.Helper()
	if !photoessai.EstUnJPEG(sortie.Content) {
		t.Fatal("la photo ré-encodée n'est pas un JPEG")
	}
	if _, err := jpeg.Decode(bytes.NewReader(sortie.Content)); err != nil {
		t.Fatalf("photo ré-encodée illisible : %v", err)
	}
	l, h, err := photoessai.Tailles(sortie.Content)
	if err != nil || l != largeur || h != hauteur || sortie.Width != largeur || sortie.Height != hauteur {
		t.Fatalf("dimensions %dx%d (annoncées %dx%d), attendues %dx%d : %v", l, h, sortie.Width, sortie.Height, largeur, hauteur, err)
	}
	for _, m := range photoessai.SegmentsAvantImage(sortie.Content) {
		if m >= 0xE1 && m <= 0xEF {
			t.Fatalf("segment de métadonnées APP%d conservé", m-0xE0)
		}
	}
	for _, trace := range [][]byte{[]byte("Exif"), []byte("GPS"), photoessai.LatitudeFictive(), []byte(photoessai.CommentaireDeLieu), []byte(photoessai.ScriptPiege)} {
		if bytes.Contains(sortie.Content, trace) {
			t.Fatalf("la photo ré-encodée contient encore %q", trace)
		}
	}
}

func exigerErreur(t *testing.T, err, attendue error) {
	t.Helper()
	if !errors.Is(err, attendue) {
		t.Fatalf("erreur %v, attendue %v", err, attendue)
	}
}

func TestUS020AC1JPEGEtPNGAcceptesEtReencodesEnJPEG(t *testing.T) {
	cas := map[string]struct {
		brut             []byte
		largeur, hauteur int
	}{
		"JPEG":             {photoessai.JPEG(64, 48), 64, 48},
		"PNG":              {photoessai.PNG(50, 20), 50, 20},
		"PNG transparent":  {photoessai.PNGTransparent(30, 30), 30, 30},
		"JPEG d'un pixel":  {photoessai.JPEG(1, 1), 1, 1},
		"JPEG avec script": {photoessai.JPEGSuiviDUnScript(), 24, 16},
	}
	for nom, c := range cas {
		t.Run(nom, func(t *testing.T) {
			sortie, err := nettoyer(t, pixelsMaxParDefaut, c.brut)
			if err != nil {
				t.Fatalf("photo refusée : %v", err)
			}
			if bytes.Equal(sortie.Content, c.brut) {
				t.Fatal("photo stockée telle quelle, sans ré-encodage")
			}
			exigerJPEGPropre(t, sortie, c.largeur, c.hauteur)
		})
	}
}

func TestUS020AC3LocalisationAbsenteApresReencodage(t *testing.T) {
	cas := map[string]struct {
		brut             []byte
		largeur, hauteur int
	}{
		"JPEG avec EXIF GPS":        {photoessai.JPEGAvecGPS(), 48, 32},
		"PNG avec eXIf GPS et tEXt": {photoessai.PNGAvecGPS(), 40, 30},
	}
	for nom, c := range cas {
		t.Run(nom, func(t *testing.T) {
			if !bytes.Contains(c.brut, photoessai.LatitudeFictive()) {
				t.Fatal("la photo de test ne porte pas les coordonnées fictives")
			}
			sortie, err := nettoyer(t, pixelsMaxParDefaut, c.brut)
			if err != nil {
				t.Fatalf("photo refusée : %v", err)
			}
			exigerJPEGPropre(t, sortie, c.largeur, c.hauteur)
		})
	}
}

func TestUS020AC5FichiersNonImageOuHorsJPEGPNGRefuses(t *testing.T) {
	cas := map[string][]byte{
		"GIF":                               photoessai.GIF(),
		"WebP":                              photoessai.WebP(),
		"SVG":                               photoessai.SVG(),
		"texte nommé .jpg":                  photoessai.FauxJPEG(),
		"polyglotte signature JPEG et HTML": photoessai.PolyglotteJPEGHTML(),
		"polyglotte GIF et ZIP":             photoessai.PolyglotteGIFZip(),
		"fichier vide":                      {},
		"JPEG tronqué":                      photoessai.JPEGTronque(),
		"signature PNG seule":               []byte("\x89PNG\r\n\x1a\n"),
	}
	for nom, brut := range cas {
		t.Run(nom, func(t *testing.T) {
			_, err := nettoyer(t, pixelsMaxParDefaut, brut)
			exigerErreur(t, err, annonces.ErrUnsupportedPhoto)
		})
	}
}

func TestUS020AC6DimensionsExcessivesRefuseesAvantDecodage(t *testing.T) {
	cas := map[string][]byte{
		"en-tête PNG de 100 000 x 100 000": photoessai.EnTetePNGGeant(100_000, 100_000),
		"en-tête JPEG de 65 535 x 65 535":  photoessai.JPEGAuxDimensionsAnnoncees(65_535, 65_535),
		"PNG au côté de 70 000 pixels":     photoessai.EnTetePNGGeant(70_000, 1),
	}
	for nom, brut := range cas {
		t.Run(nom, func(t *testing.T) {
			_, err := nettoyer(t, pixelsMaxParDefaut, brut)
			exigerErreur(t, err, annonces.ErrPhotoTooManyPixels)
		})
	}
}

func TestUS020AC6PlafondDePixelsInclus(t *testing.T) {
	if _, err := nettoyer(t, 64*48, photoessai.JPEG(64, 48)); err != nil {
		t.Fatalf("photo exactement au plafond refusée : %v", err)
	}
	_, err := nettoyer(t, 64*48-1, photoessai.JPEG(64, 48))
	exigerErreur(t, err, annonces.ErrPhotoTooManyPixels)
}

func TestUS020AC6BombeDeDecompressionRefuseeSansSaturerLaMemoire(t *testing.T) {
	bombe := photoessai.BombePNG(10_000, 10_000)
	if len(bombe) > 1<<20 {
		t.Fatalf("bombe de %d octets, attendue compacte", len(bombe))
	}
	s := nettoyeur(t, pixelsMaxParDefaut)
	runtime.GC()
	var avant, apres runtime.MemStats
	runtime.ReadMemStats(&avant)
	debut := time.Now()
	_, err := s.Sanitize(t.Context(), bombe)
	duree := time.Since(debut)
	runtime.ReadMemStats(&apres)
	exigerErreur(t, err, annonces.ErrPhotoTooManyPixels)
	if alloue := apres.TotalAlloc - avant.TotalAlloc; alloue > plafondMemoireBombe {
		t.Fatalf("%d octets alloués pour refuser une bombe de 100 mégapixels", alloue)
	}
	if duree > delaiMaxBombe {
		t.Fatalf("bombe refusée en %s, décodage complet probable", duree)
	}
}

func TestUS020AC6BombesSimultaneesRefuseesSansSaturerLaMemoire(t *testing.T) {
	bombe := photoessai.BombePNG(10_000, 10_000)
	s := nettoyeur(t, pixelsMaxParDefaut)
	runtime.GC()
	var avant, apres runtime.MemStats
	runtime.ReadMemStats(&avant)
	erreurs := make(chan error, 16)
	for range cap(erreurs) {
		go func() {
			_, err := s.Sanitize(t.Context(), bombe)
			erreurs <- err
		}()
	}
	for range cap(erreurs) {
		exigerErreur(t, <-erreurs, annonces.ErrPhotoTooManyPixels)
	}
	runtime.ReadMemStats(&apres)
	if alloue := apres.TotalAlloc - avant.TotalAlloc; alloue > plafondMemoireBombe {
		t.Fatalf("%d octets alloués pour refuser 16 bombes simultanées", alloue)
	}
}

const pixelsMaxDuFuzz = 1 << 20

func nettoyeurDuFuzz(f *testing.F) *photo.Sanitizer {
	f.Helper()
	s, err := photo.NewSanitizer(photo.Limits{MaxPixels: pixelsMaxDuFuzz, ConcurrentDecodes: 1})
	if err != nil {
		f.Fatalf("nettoyeur : %v", err)
	}
	return s
}

func FuzzUS020DecodageNeCedeJamais(f *testing.F) {
	for _, graine := range [][]byte{
		photoessai.JPEG(8, 8), photoessai.PNG(8, 8), photoessai.JPEGAvecGPS(), photoessai.PNGAvecGPS(),
		photoessai.GIF(), photoessai.WebP(), photoessai.SVG(), photoessai.FauxJPEG(), photoessai.PolyglotteJPEGHTML(),
		photoessai.JPEGTronque(), photoessai.EnTetePNGGeant(100_000, 100_000), photoessai.JPEGAuxDimensionsAnnoncees(9000, 9000), {},
	} {
		f.Add(graine)
	}
	s := nettoyeurDuFuzz(f)
	f.Fuzz(func(t *testing.T, brut []byte) {
		sortie, err := s.Sanitize(t.Context(), brut)
		if err != nil {
			if !errors.Is(err, annonces.ErrUnsupportedPhoto) && !errors.Is(err, annonces.ErrPhotoTooManyPixels) {
				t.Fatalf("erreur inattendue : %v", err)
			}
			return
		}
		l, h, err := photoessai.Tailles(sortie.Content)
		if err != nil || l*h > pixelsMaxDuFuzz || l != sortie.Width || h != sortie.Height {
			t.Fatalf("sortie acceptée invalide : %dx%d (%dx%d annoncés) : %v", l, h, sortie.Width, sortie.Height, err)
		}
	})
}
