package photo_test

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"testing"

	"collector/services/place-de-marche/internal/annonces"
	"collector/services/place-de-marche/internal/annonces/photo/photoessai"
)

var boutismes = map[string]binary.AppendByteOrder{"petit boutisme (II)": binary.LittleEndian, "gros boutisme (MM)": binary.BigEndian}

func exigerRedressee(t *testing.T, sortie annonces.EncodedPhoto) {
	t.Helper()
	quarts, cadre, err := photoessai.QuartsDe(sortie.Content)
	if err != nil {
		t.Fatalf("sortie illisible : %v", err)
	}
	if cadre.Dx() != photoessai.LargeurRedressee || cadre.Dy() != photoessai.HauteurRedressee ||
		sortie.Width != photoessai.LargeurRedressee || sortie.Height != photoessai.HauteurRedressee {
		t.Fatalf("dimensions %dx%d (annoncées %dx%d), %dx%d attendues", cadre.Dx(), cadre.Dy(), sortie.Width, sortie.Height,
			photoessai.LargeurRedressee, photoessai.HauteurRedressee)
	}
	for rang, attendue := range photoessai.CouleursDesQuarts {
		if !photoessai.Proche(quarts[rang], attendue) {
			t.Fatalf("quart %d : %v, attendu %v (image non redressée)", rang, quarts[rang], attendue)
		}
	}
}

func TestUS020OrientationEXIFAppliqueeAuxHuitCas(t *testing.T) {
	for nomBoutisme, ordre := range boutismes {
		for orientation := uint16(1); orientation <= 8; orientation++ {
			t.Run(fmt.Sprintf("%s orientation %d", nomBoutisme, orientation), func(t *testing.T) {
				brut := photoessai.JPEGOriente(orientation, ordre)
				stockee := photoessai.ImageStockeePour(orientation).Bounds()
				if orientation >= 5 && stockee.Dx() != photoessai.HauteurRedressee {
					t.Fatalf("précondition : image stockée %v non couchée", stockee)
				}
				sortie, err := nettoyer(t, pixelsMaxParDefaut, brut)
				if err != nil {
					t.Fatalf("photo refusée : %v", err)
				}
				exigerRedressee(t, sortie)
				exigerJPEGPropre(t, sortie, photoessai.LargeurRedressee, photoessai.HauteurRedressee)
			})
		}
	}
}

func TestUS020OrientationHorsNormeIgnoree(t *testing.T) {
	for _, orientation := range []uint16{0, 9, 0xFFFF} {
		t.Run(fmt.Sprint(orientation), func(t *testing.T) {
			brut := photoessai.JPEGAvecEXIF(photoessai.JPEG(40, 20), photoessai.TIFFOrientation(orientation, binary.LittleEndian))
			sortie, err := nettoyer(t, pixelsMaxParDefaut, brut)
			if err != nil {
				t.Fatalf("photo refusée : %v", err)
			}
			exigerJPEGPropre(t, sortie, 40, 20)
		})
	}
}

func TestUS020OrientationDUnPNGNonAppliquee(t *testing.T) {
	brut := photoessai.PNGOriente(6)
	sortie, err := nettoyer(t, pixelsMaxParDefaut, brut)
	if err != nil {
		t.Fatalf("photo refusée : %v", err)
	}
	stockee := photoessai.ImageStockeePour(6).Bounds()
	exigerJPEGPropre(t, sortie, stockee.Dx(), stockee.Dy())
}

func TestUS020SegmentEXIFMalFormeSansEffetNiPlantage(t *testing.T) {
	base := photoessai.JPEG(40, 20)
	tiff := photoessai.TIFFOrientation(6, binary.LittleEndian)
	horsBornes := append([]byte("II*\x00"), 0xFF, 0xFF, 0xFF, 0x7F)
	cas := map[string][]byte{
		"TIFF tronqué dans l'entrée":  photoessai.JPEGAvecEXIF(base, tiff[:12]),
		"TIFF réduit à l'en-tête":     photoessai.JPEGAvecEXIF(base, tiff[:8]),
		"offset d'IFD hors bornes":    photoessai.JPEGAvecEXIF(base, horsBornes),
		"marque de boutisme inconnue": photoessai.JPEGAvecEXIF(base, append([]byte("XX"), tiff[2:]...)),
		"segment Exif vide":           photoessai.JPEGAvecEXIF(base, nil),
		"APP1 sans préfixe Exif":      photoessai.JPEGAvecAPP1(base, tiff, photoessai.LongueurDeSegment(tiff)),
		"nombre d'entrées démesuré":   photoessai.JPEGAvecEXIF(base, append(append([]byte{}, tiff[:8]...), 0xFF, 0xFF)),
	}
	for nom, brut := range cas {
		t.Run(nom, func(t *testing.T) {
			sortie, err := nettoyer(t, pixelsMaxParDefaut, brut)
			if err != nil {
				exigerErreur(t, err, annonces.ErrUnsupportedPhoto)
				return
			}
			exigerJPEGPropre(t, sortie, 40, 20)
		})
	}
}

func TestUS020LongueurDeSegmentAPP1Incoherente(t *testing.T) {
	base := photoessai.JPEG(40, 20)
	charge := append([]byte("Exif\x00\x00"), photoessai.TIFFOrientation(6, binary.BigEndian)...)
	for _, longueur := range []uint16{0, 1, 2, 8, 0xFFFF} {
		t.Run(fmt.Sprint(longueur), func(t *testing.T) {
			_, err := nettoyer(t, pixelsMaxParDefaut, photoessai.JPEGAvecAPP1(base, charge, longueur))
			if err != nil && !errors.Is(err, annonces.ErrUnsupportedPhoto) {
				t.Fatalf("erreur inattendue : %v", err)
			}
		})
	}
}

func grainesEXIF() [][]byte {
	var graines [][]byte
	for _, ordre := range boutismes {
		for orientation := uint16(1); orientation <= 8; orientation++ {
			graines = append(graines, append([]byte("Exif\x00\x00"), photoessai.TIFFOrientation(orientation, ordre)...))
		}
	}
	return append(graines,
		[]byte("Exif\x00\x00II*\x00\xff\xff\xff\x7f"),
		[]byte("Exif\x00\x00MM\x00*\x00\x00\x00\x08\xff\xff"),
		[]byte{},
	)
}

func exigerSortieDuFuzzEXIF(t *testing.T, sortie annonces.EncodedPhoto) {
	t.Helper()
	l, h, err := photoessai.Tailles(sortie.Content)
	droite := l == 24 && h == 16
	couchee := l == 16 && h == 24
	if err != nil || (!droite && !couchee) || l != sortie.Width || h != sortie.Height {
		t.Fatalf("sortie %dx%d (annoncée %dx%d) : %v", l, h, sortie.Width, sortie.Height, err)
	}
	if bytes.Contains(sortie.Content, []byte("Exif")) {
		t.Fatal("EXIF conservé en sortie")
	}
}

func FuzzUS020LecteurEXIF(f *testing.F) {
	base := photoessai.JPEG(24, 16)
	for _, charge := range grainesEXIF() {
		f.Add(charge, photoessai.LongueurDeSegment(charge))
		f.Add(charge[:len(charge)/2], photoessai.LongueurDeSegment(charge))
	}
	s := nettoyeurDuFuzz(f)
	f.Fuzz(func(t *testing.T, charge []byte, longueur uint16) {
		sortie, err := s.Sanitize(t.Context(), photoessai.JPEGAvecAPP1(base, charge, longueur))
		if err == nil {
			exigerSortieDuFuzzEXIF(t, sortie)
			return
		}
		if !errors.Is(err, annonces.ErrUnsupportedPhoto) && !errors.Is(err, annonces.ErrPhotoTooManyPixels) {
			t.Fatalf("erreur inattendue : %v", err)
		}
	})
}
