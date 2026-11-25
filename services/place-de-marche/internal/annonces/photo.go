package annonces

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"
)

var (
	ErrPhotoNotFound              = errors.New("photo introuvable")
	ErrPhotoLimitReached          = errors.New("nombre maximal de photos atteint")
	ErrUnsupportedPhoto           = errors.New("photo ni JPEG ni PNG lisible")
	ErrPhotoTooManyPixels         = errors.New("dimensions de la photo excessives")
	ErrPhotoProcessingUnavailable = errors.New("traitement des photos saturé")
)

type PhotoID uuid.UUID

func ParsePhotoID(raw string) (PhotoID, error) {
	parsed, err := uuid.Parse(raw)
	if err != nil {
		return PhotoID{}, fmt.Errorf("identifiant de photo invalide : %w", err)
	}
	return PhotoID(parsed), nil
}

func (id PhotoID) String() string {
	return uuid.UUID(id).String()
}

type PhotoKey string

type Dimensions struct {
	Width  int
	Height int
}

type EncodedPhoto struct {
	Content []byte
	Dimensions
}

type Photo struct {
	ID  PhotoID
	Key PhotoKey
	Dimensions
}

type StoredPhoto struct {
	Key  PhotoKey
	Size int
	Dimensions
}

type PhotoURL struct {
	Address   string
	ExpiresAt time.Time
}

type PhotoLink struct {
	Photo
	PhotoURL
}

type ArticlePhotoState struct {
	Article ArticleState
	Photos  int
}

func (s ArticleState) VerifyPhotosEditable() error {
	if s.Reserved {
		return ErrArticleReserved
	}
	if !s.Status.Submittable() {
		return ErrArticleNotEditable
	}
	return nil
}

func (s ArticlePhotoState) VerifyAddable(limit int) error {
	if err := s.Article.VerifyPhotosEditable(); err != nil {
		return err
	}
	if s.Photos >= limit {
		return ErrPhotoLimitReached
	}
	return nil
}

type PhotoRecords interface {
	CheckPhotoAddable(ctx context.Context, seller SellerID, id ArticleID, limit int) error
	AddPhoto(ctx context.Context, seller SellerID, id ArticleID, photo StoredPhoto, limit int) (Photo, error)
	RemovePhoto(ctx context.Context, seller SellerID, id ArticleID, photo PhotoID) (Photo, error)
	OwnPhotos(ctx context.Context, seller SellerID, id ArticleID) ([]Photo, error)
	PublishedPhotos(ctx context.Context, id ArticleID) ([]Photo, error)
	ArticlePhotos(ctx context.Context, id ArticleID) ([]Photo, error)
}

type PhotoSanitizer interface {
	Sanitize(ctx context.Context, raw []byte) (EncodedPhoto, error)
}

type PhotoObjects interface {
	NewKey() PhotoKey
	Put(ctx context.Context, key PhotoKey, content []byte) error
	Delete(ctx context.Context, key PhotoKey) error
	Link(ctx context.Context, key PhotoKey) (PhotoURL, error)
}

type PhotoService struct {
	records    PhotoRecords
	sanitizer  PhotoSanitizer
	objects    PhotoObjects
	perArticle int
	logger     *slog.Logger
}

func NewPhotoService(records PhotoRecords, sanitizer PhotoSanitizer, objects PhotoObjects, perArticle int, logger *slog.Logger) *PhotoService {
	return &PhotoService{records: records, sanitizer: sanitizer, objects: objects, perArticle: perArticle, logger: logger}
}

func (s *PhotoService) AddPhoto(ctx context.Context, seller SellerID, id ArticleID, raw []byte) (PhotoLink, error) {
	if err := s.records.CheckPhotoAddable(ctx, seller, id, s.perArticle); err != nil {
		return PhotoLink{}, err
	}
	encoded, err := s.sanitizer.Sanitize(ctx, raw)
	if err != nil {
		return PhotoLink{}, err
	}
	stored := StoredPhoto{Key: s.objects.NewKey(), Size: len(encoded.Content), Dimensions: encoded.Dimensions}
	if err = s.objects.Put(ctx, stored.Key, encoded.Content); err != nil {
		return PhotoLink{}, fmt.Errorf("écriture de la photo : %w", err)
	}
	photo, err := s.records.AddPhoto(ctx, seller, id, stored, s.perArticle)
	if err != nil {
		s.deleteObject(ctx, stored.Key, slog.String("article_id", id.String()))
		return PhotoLink{}, err
	}
	return s.link(ctx, photo)
}

func (s *PhotoService) RemovePhoto(ctx context.Context, seller SellerID, id ArticleID, photoID PhotoID) error {
	removed, err := s.records.RemovePhoto(ctx, seller, id, photoID)
	if err != nil {
		return err
	}
	s.deleteObject(ctx, removed.Key, slog.String("photo_id", removed.ID.String()))
	return nil
}

func (s *PhotoService) OwnPhotos(ctx context.Context, seller SellerID, id ArticleID) ([]PhotoLink, error) {
	photos, err := s.records.OwnPhotos(ctx, seller, id)
	if err != nil {
		return nil, err
	}
	return s.links(ctx, photos)
}

func (s *PhotoService) PublishedPhotos(ctx context.Context, id ArticleID) ([]PhotoLink, error) {
	photos, err := s.records.PublishedPhotos(ctx, id)
	if err != nil {
		return nil, err
	}
	return s.links(ctx, photos)
}

func (s *PhotoService) ArticlePhotos(ctx context.Context, id ArticleID) ([]PhotoLink, error) {
	photos, err := s.records.ArticlePhotos(ctx, id)
	if err != nil {
		return nil, err
	}
	return s.links(ctx, photos)
}

func (s *PhotoService) links(ctx context.Context, photos []Photo) ([]PhotoLink, error) {
	links := make([]PhotoLink, 0, len(photos))
	for _, photo := range photos {
		link, err := s.link(ctx, photo)
		if err != nil {
			return nil, err
		}
		links = append(links, link)
	}
	return links, nil
}

func (s *PhotoService) link(ctx context.Context, photo Photo) (PhotoLink, error) {
	url, err := s.objects.Link(ctx, photo.Key)
	if err != nil {
		return PhotoLink{}, fmt.Errorf("présignature de la photo %s : %w", photo.ID, err)
	}
	return PhotoLink{Photo: photo, PhotoURL: url}, nil
}

func (s *PhotoService) deleteObject(ctx context.Context, key PhotoKey, subject slog.Attr) {
	err := s.objects.Delete(context.WithoutCancel(ctx), key)
	if err == nil {
		return
	}
	s.logger.WarnContext(ctx, "objet de photo non supprimé",
		slog.String("event", "annonces.photo_orpheline"),
		subject,
		slog.Any("erreur", err),
	)
}
