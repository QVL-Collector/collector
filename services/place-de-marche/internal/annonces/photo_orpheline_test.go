package annonces_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"

	"collector/services/place-de-marche/internal/annonces"
)

const cleSecrete = annonces.PhotoKey("photos/cle-a-ne-jamais-journaliser.jpg")

var errStockageIndisponible = errors.New("S3 indisponible")

type registreDePhotos struct {
	refusAjout  error
	photoRetire annonces.Photo
}

func (r registreDePhotos) CheckPhotoAddable(context.Context, annonces.SellerID, annonces.ArticleID, int) error {
	return nil
}

func (r registreDePhotos) AddPhoto(_ context.Context, _ annonces.SellerID, _ annonces.ArticleID, photo annonces.StoredPhoto, _ int) (annonces.Photo, error) {
	if r.refusAjout != nil {
		return annonces.Photo{}, r.refusAjout
	}
	return annonces.Photo{ID: annonces.PhotoID(uuid.New()), Key: photo.Key, Dimensions: photo.Dimensions}, nil
}

func (r registreDePhotos) RemovePhoto(context.Context, annonces.SellerID, annonces.ArticleID, annonces.PhotoID) (annonces.Photo, error) {
	return r.photoRetire, nil
}

func (r registreDePhotos) OwnPhotos(context.Context, annonces.SellerID, annonces.ArticleID) ([]annonces.Photo, error) {
	return nil, nil
}

func (r registreDePhotos) PublishedPhotos(context.Context, annonces.ArticleID) ([]annonces.Photo, error) {
	return nil, nil
}

func (r registreDePhotos) ArticlePhotos(context.Context, annonces.ArticleID) ([]annonces.Photo, error) {
	return nil, nil
}

type nettoyeurNeutre struct{}

func (nettoyeurNeutre) Sanitize(context.Context, []byte) (annonces.EncodedPhoto, error) {
	return annonces.EncodedPhoto{Content: []byte{0xFF, 0xD8}, Dimensions: annonces.Dimensions{Width: 1, Height: 1}}, nil
}

type objetsDeTest struct {
	refusSuppression error
	supprimes        *[]annonces.PhotoKey
}

func (o objetsDeTest) NewKey() annonces.PhotoKey { return cleSecrete }

func (o objetsDeTest) Put(context.Context, annonces.PhotoKey, []byte) error { return nil }

func (o objetsDeTest) Delete(_ context.Context, cle annonces.PhotoKey) error {
	*o.supprimes = append(*o.supprimes, cle)
	return o.refusSuppression
}

func (o objetsDeTest) Link(context.Context, annonces.PhotoKey) (annonces.PhotoURL, error) {
	return annonces.PhotoURL{Address: "https://bucket/photo?X-Amz-Signature=s", ExpiresAt: time.Now().Add(time.Minute)}, nil
}

type scenarioOrphelin struct {
	journal   bytes.Buffer
	supprimes []annonces.PhotoKey
	article   annonces.ArticleID
}

func (s *scenarioOrphelin) service(registre registreDePhotos, refusSuppression error) *annonces.PhotoService {
	s.article = annonces.ArticleID(uuid.New())
	objets := objetsDeTest{refusSuppression: refusSuppression, supprimes: &s.supprimes}
	return annonces.NewPhotoService(registre, nettoyeurNeutre{}, objets, 1, slog.New(slog.NewJSONHandler(&s.journal, nil)))
}

func (s *scenarioOrphelin) evenements(t *testing.T) []map[string]any {
	t.Helper()
	var lus []map[string]any
	for _, ligne := range strings.Split(strings.TrimSpace(s.journal.String()), "\n") {
		if ligne == "" {
			continue
		}
		var evenement map[string]any
		if err := json.Unmarshal([]byte(ligne), &evenement); err != nil {
			t.Fatalf("journal illisible : %v : %s", err, ligne)
		}
		if evenement["event"] == "annonces.photo_orpheline" {
			lus = append(lus, evenement)
		}
	}
	return lus
}

func TestUS020EnregistrementEchoueEtSuppressionEchoueJournaliseLObjetOrphelinSansSaCle(t *testing.T) {
	var s scenarioOrphelin
	service := s.service(registreDePhotos{refusAjout: annonces.ErrArticleNotEditable}, errStockageIndisponible)

	_, err := service.AddPhoto(t.Context(), annonces.SellerID(uuid.New()), annonces.ArticleID(uuid.New()), []byte("brut"))

	if !errors.Is(err, annonces.ErrArticleNotEditable) {
		t.Fatalf("erreur %v, erreur métier attendue", err)
	}
	if errors.Is(err, errStockageIndisponible) {
		t.Fatalf("l'erreur S3 remonte à l'appelant : %v", err)
	}
	evenements := s.evenements(t)
	if len(evenements) != 1 || evenements[0]["level"] != "WARN" || evenements[0]["article_id"] == nil {
		t.Fatalf("journal orphelin : %v\n%s", evenements, s.journal.String())
	}
	if strings.Contains(s.journal.String(), string(cleSecrete)) || strings.Contains(s.journal.String(), "cle-a-ne-jamais") {
		t.Fatalf("la clé d'objet est journalisée : %s", s.journal.String())
	}
	if len(s.supprimes) != 1 || s.supprimes[0] != cleSecrete {
		t.Fatalf("suppressions tentées : %v", s.supprimes)
	}
}

func TestUS020EnregistrementEchoueEtSuppressionReussieSansJournalOrphelin(t *testing.T) {
	var s scenarioOrphelin
	service := s.service(registreDePhotos{refusAjout: annonces.ErrPhotoLimitReached}, nil)

	_, err := service.AddPhoto(t.Context(), annonces.SellerID(uuid.New()), s.article, []byte("brut"))

	if !errors.Is(err, annonces.ErrPhotoLimitReached) {
		t.Fatalf("erreur %v, plafond attendu", err)
	}
	if len(s.supprimes) != 1 || len(s.evenements(t)) != 0 {
		t.Fatalf("suppressions %v, journaux orphelins %v", s.supprimes, s.evenements(t))
	}
}

func TestUS020DepotEnregistreSansSuppressionNiJournalOrphelin(t *testing.T) {
	var s scenarioOrphelin
	service := s.service(registreDePhotos{}, errStockageIndisponible)

	if _, err := service.AddPhoto(t.Context(), annonces.SellerID(uuid.New()), s.article, []byte("brut")); err != nil {
		t.Fatalf("dépôt refusé : %v", err)
	}
	if len(s.supprimes) != 0 || len(s.evenements(t)) != 0 {
		t.Fatalf("suppressions %v, journaux orphelins %v", s.supprimes, s.evenements(t))
	}
}

func TestUS020RetraitDontLaSuppressionEchoueJournaliseLOrphelinSansSaCle(t *testing.T) {
	var s scenarioOrphelin
	retiree := annonces.Photo{ID: annonces.PhotoID(uuid.New()), Key: cleSecrete}
	service := s.service(registreDePhotos{photoRetire: retiree}, errStockageIndisponible)

	if err := service.RemovePhoto(t.Context(), annonces.SellerID(uuid.New()), s.article, retiree.ID); err != nil {
		t.Fatalf("retrait refusé : %v", err)
	}
	evenements := s.evenements(t)
	if len(evenements) != 1 || evenements[0]["photo_id"] != retiree.ID.String() {
		t.Fatalf("journal orphelin du retrait : %v", evenements)
	}
	if strings.Contains(s.journal.String(), string(cleSecrete)) {
		t.Fatalf("la clé d'objet est journalisée : %s", s.journal.String())
	}
}
