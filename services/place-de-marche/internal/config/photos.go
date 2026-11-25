package config

import (
	"errors"
	"fmt"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"

	"collector/services/place-de-marche/internal/annonces/objets"
	"collector/services/place-de-marche/internal/annonces/photo"
)

const (
	envPhotoBucket            = "PHOTOS_BUCKET"
	envPhotoRegion            = "PHOTOS_REGION"
	envPhotoEndpoint          = "PHOTOS_S3_ADRESSE"
	envPhotoPathStyle         = "PHOTOS_S3_STYLE_CHEMIN"
	envPhotoLinkLifetime      = "PHOTOS_URL_DUREE"
	envPhotoMaxBytes          = "PHOTOS_TAILLE_MAX_OCTETS"
	envPhotoMaxPixels         = "PHOTOS_PIXELS_MAX"
	envPhotosPerArticle       = "PHOTOS_PAR_ARTICLE_MAX"
	envPhotoConcurrentDecodes = "PHOTOS_DECODAGES_SIMULTANES"
	defaultPhotoRegion        = "us-east-1"
)

var bucketName = regexp.MustCompile(`^[a-z0-9][a-z0-9.-]{1,61}[a-z0-9]$`)

type integerRange struct {
	fallback int
	minimum  int
	maximum  int
}

type durationRange struct {
	fallback time.Duration
	minimum  time.Duration
	maximum  time.Duration
}

var (
	photoLinkLifetime      = durationRange{fallback: 10 * time.Minute, minimum: time.Second, maximum: 15 * time.Minute}
	photoMaxBytes          = integerRange{fallback: 5 << 20, minimum: 1 << 10, maximum: 20 << 20}
	photoMaxPixels         = integerRange{fallback: 16_000_000, minimum: 1, maximum: 40_000_000}
	photosPerArticle       = integerRange{fallback: 1, minimum: 1, maximum: 10}
	photoConcurrentDecodes = integerRange{fallback: 2, minimum: 1, maximum: 8}
)

type Photos struct {
	Storage    objets.Settings
	Limits     photo.Limits
	MaxBytes   int64
	PerArticle int
}

func LoadPhotos(lookup Lookup) (Photos, error) {
	settings := settingsReader{lookup: lookup}
	photos := Photos{
		Storage: objets.Settings{
			Bucket:       settings.bucket(envPhotoBucket),
			Region:       valueOrDefault(lookup(envPhotoRegion), defaultPhotoRegion),
			Endpoint:     settings.endpoint(envPhotoEndpoint),
			PathStyle:    settings.boolean(envPhotoPathStyle),
			LinkLifetime: settings.duration(envPhotoLinkLifetime, photoLinkLifetime),
		},
		Limits: photo.Limits{
			MaxPixels:         settings.integer(envPhotoMaxPixels, photoMaxPixels),
			ConcurrentDecodes: settings.integer(envPhotoConcurrentDecodes, photoConcurrentDecodes),
		},
		MaxBytes:   int64(settings.integer(envPhotoMaxBytes, photoMaxBytes)),
		PerArticle: settings.integer(envPhotosPerArticle, photosPerArticle),
	}
	if err := settings.err(); err != nil {
		return Photos{}, err
	}
	return photos, nil
}

type settingsReader struct {
	lookup   Lookup
	failures []error
}

func (s *settingsReader) err() error {
	return errors.Join(s.failures...)
}

func (s *settingsReader) fail(name, expectation string) {
	s.failures = append(s.failures, fmt.Errorf("%s : %s", name, expectation))
}

func (s *settingsReader) bucket(name string) string {
	value := strings.TrimSpace(s.lookup(name))
	if !bucketName.MatchString(value) {
		s.fail(name, "nom de bucket S3 obligatoire (3 à 63 caractères : minuscules, chiffres, points, tirets)")
	}
	return value
}

func (s *settingsReader) endpoint(name string) string {
	value := strings.TrimSpace(s.lookup(name))
	if value == "" {
		return ""
	}
	parsed, err := url.Parse(value)
	if err != nil || (parsed.Scheme != "https" && parsed.Scheme != "http") || parsed.Host == "" {
		s.fail(name, "adresse http(s) absolue attendue (ex. http://127.0.0.1:8333)")
	}
	return value
}

func (s *settingsReader) boolean(name string) bool {
	value := strings.TrimSpace(s.lookup(name))
	if value == "" {
		return false
	}
	parsed, err := strconv.ParseBool(value)
	if err != nil {
		s.fail(name, "true ou false attendu")
	}
	return parsed
}

func (s *settingsReader) integer(name string, bounds integerRange) int {
	value := strings.TrimSpace(s.lookup(name))
	if value == "" {
		return bounds.fallback
	}
	parsed, err := strconv.Atoi(value)
	if err != nil || parsed < bounds.minimum || parsed > bounds.maximum {
		s.fail(name, fmt.Sprintf("entier de %d à %d attendu", bounds.minimum, bounds.maximum))
	}
	return parsed
}

func (s *settingsReader) duration(name string, bounds durationRange) time.Duration {
	value := strings.TrimSpace(s.lookup(name))
	if value == "" {
		return bounds.fallback
	}
	parsed, err := time.ParseDuration(value)
	if err != nil || parsed < bounds.minimum || parsed > bounds.maximum {
		s.fail(name, fmt.Sprintf("durée de %s à %s attendue (ex. 10m)", bounds.minimum, bounds.maximum))
	}
	return parsed
}
