package config_test

import (
	"strings"
	"testing"
	"time"

	"collector/services/place-de-marche/internal/config"
)

const bucketValide = "collector-photos-local"

func environnementPhotos(valeurs map[string]string) config.Lookup {
	complet := map[string]string{"PHOTOS_BUCKET": bucketValide}
	for nom, valeur := range valeurs {
		complet[nom] = valeur
	}
	return func(nom string) string { return complet[nom] }
}

func chargerPhotos(t *testing.T, valeurs map[string]string) config.Photos {
	t.Helper()
	photos, err := config.LoadPhotos(environnementPhotos(valeurs))
	if err != nil {
		t.Fatalf("configuration %v refusée : %v", valeurs, err)
	}
	return photos
}

func exigerRefusNommant(t *testing.T, valeurs map[string]string, variables ...string) {
	t.Helper()
	_, err := config.LoadPhotos(environnementPhotos(valeurs))
	if err == nil {
		t.Fatalf("configuration %v acceptée", valeurs)
	}
	for _, variable := range variables {
		if !strings.Contains(err.Error(), variable) {
			t.Fatalf("erreur %q sans la variable %s", err, variable)
		}
	}
}

func TestUS020ConfigurationPhotosValeursParDefaut(t *testing.T) {
	photos := chargerPhotos(t, nil)
	if photos.Storage.Bucket != bucketValide || photos.Storage.Region != "us-east-1" || photos.Storage.Endpoint != "" || photos.Storage.PathStyle {
		t.Fatalf("stockage par défaut : %+v", photos.Storage)
	}
	if photos.Storage.LinkLifetime != 10*time.Minute || photos.MaxBytes != 5<<20 || photos.Limits.MaxPixels != 16_000_000 ||
		photos.PerArticle != 1 || photos.Limits.ConcurrentDecodes != 2 {
		t.Fatalf("limites par défaut : %+v", photos)
	}
}

func TestUS020ConfigurationPhotosBornesAcceptees(t *testing.T) {
	cas := []struct {
		variable, valeur string
		verifier         func(config.Photos) bool
	}{
		{"PHOTOS_URL_DUREE", "1s", func(p config.Photos) bool { return p.Storage.LinkLifetime == time.Second }},
		{"PHOTOS_URL_DUREE", "15m", func(p config.Photos) bool { return p.Storage.LinkLifetime == 15*time.Minute }},
		{"PHOTOS_DECODAGES_SIMULTANES", "1", func(p config.Photos) bool { return p.Limits.ConcurrentDecodes == 1 }},
		{"PHOTOS_DECODAGES_SIMULTANES", "8", func(p config.Photos) bool { return p.Limits.ConcurrentDecodes == 8 }},
		{"PHOTOS_TAILLE_MAX_OCTETS", "1024", func(p config.Photos) bool { return p.MaxBytes == 1024 }},
		{"PHOTOS_TAILLE_MAX_OCTETS", "20971520", func(p config.Photos) bool { return p.MaxBytes == 20<<20 }},
		{"PHOTOS_PIXELS_MAX", "1", func(p config.Photos) bool { return p.Limits.MaxPixels == 1 }},
		{"PHOTOS_PIXELS_MAX", "40000000", func(p config.Photos) bool { return p.Limits.MaxPixels == 40_000_000 }},
		{"PHOTOS_PAR_ARTICLE_MAX", "1", func(p config.Photos) bool { return p.PerArticle == 1 }},
		{"PHOTOS_PAR_ARTICLE_MAX", "10", func(p config.Photos) bool { return p.PerArticle == 10 }},
		{"PHOTOS_S3_ADRESSE", "http://127.0.0.1:8333", func(p config.Photos) bool { return p.Storage.Endpoint == "http://127.0.0.1:8333" }},
		{"PHOTOS_S3_ADRESSE", "https://s3.local.test", func(p config.Photos) bool { return p.Storage.Endpoint == "https://s3.local.test" }},
		{"PHOTOS_S3_STYLE_CHEMIN", "true", func(p config.Photos) bool { return p.Storage.PathStyle }},
		{"PHOTOS_REGION", "eu-west-3", func(p config.Photos) bool { return p.Storage.Region == "eu-west-3" }},
		{"PHOTOS_BUCKET", "abc", func(p config.Photos) bool { return p.Storage.Bucket == "abc" }},
		{"PHOTOS_BUCKET", strings.Repeat("a", 63), func(p config.Photos) bool { return len(p.Storage.Bucket) == 63 }},
	}
	for _, c := range cas {
		t.Run(c.variable+"="+c.valeur, func(t *testing.T) {
			if photos := chargerPhotos(t, map[string]string{c.variable: c.valeur}); !c.verifier(photos) {
				t.Fatalf("%s=%s mal lu : %+v", c.variable, c.valeur, photos)
			}
		})
	}
}

func TestUS020ConfigurationPhotosHorsBornesRefusee(t *testing.T) {
	cas := map[string][]string{
		"PHOTOS_URL_DUREE":            {"0s", "999ms", "15m1s", "1h", "dix", "-1m"},
		"PHOTOS_DECODAGES_SIMULTANES": {"0", "9", "-1", "deux"},
		"PHOTOS_TAILLE_MAX_OCTETS":    {"1023", "20971521", "0", "5Mo"},
		"PHOTOS_PIXELS_MAX":           {"0", "40000001"},
		"PHOTOS_PAR_ARTICLE_MAX":      {"0", "11"},
		"PHOTOS_BUCKET":               {"Collector-Photos", strings.Repeat("a", 64), "ab", "-photos", "photos-", "photos_locales"},
		"PHOTOS_S3_ADRESSE":           {"127.0.0.1:8333", "ftp://127.0.0.1:8333", "http://", "/chemin"},
		"PHOTOS_S3_STYLE_CHEMIN":      {"oui", "2"},
	}
	for variable, valeurs := range cas {
		for _, valeur := range valeurs {
			t.Run(variable+"="+valeur, func(t *testing.T) {
				exigerRefusNommant(t, map[string]string{variable: valeur}, variable)
			})
		}
	}
}

func TestUS020ConfigurationPhotosBucketObligatoire(t *testing.T) {
	exigerRefusNommant(t, map[string]string{"PHOTOS_BUCKET": ""}, "PHOTOS_BUCKET")
}

func TestUS020ConfigurationPhotosErreursReuniesDansUnSeulMessage(t *testing.T) {
	exigerRefusNommant(t, map[string]string{
		"PHOTOS_BUCKET":               "",
		"PHOTOS_URL_DUREE":            "16m",
		"PHOTOS_DECODAGES_SIMULTANES": "9",
		"PHOTOS_S3_ADRESSE":           "ftp://s3",
	}, "PHOTOS_BUCKET", "PHOTOS_URL_DUREE", "PHOTOS_DECODAGES_SIMULTANES", "PHOTOS_S3_ADRESSE")
}
