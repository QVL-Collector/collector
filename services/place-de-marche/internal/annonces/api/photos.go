package api

import (
	"context"
	"errors"
	"io"
	"net/http"
	"time"

	"collector/libs/platform/httpjson"
	"collector/libs/platform/problem"
	"collector/services/place-de-marche/internal/annonces"
)

const photoParameter = "photoId"

var photoLimitReached = problem.Typed("urn:collector:probleme:photos-limite-atteinte", "Nombre maximal de photos atteint",
	http.StatusConflict, "L'article porte déjà le nombre maximal de photos.")

var unsupportedPhoto = problem.Typed("urn:collector:probleme:photo-non-prise-en-charge", "Photo non prise en charge",
	http.StatusUnsupportedMediaType, "Seules les photos JPEG et PNG sont acceptées.")

var oversizedPhoto = problem.Typed("urn:collector:probleme:photo-trop-volumineuse", "Photo trop volumineuse",
	http.StatusRequestEntityTooLarge, "La photo dépasse la taille autorisée.")

var excessiveDimensions = problem.Typed("urn:collector:probleme:photo-dimensions-excessives", "Dimensions excessives",
	http.StatusUnprocessableEntity, "La photo dépasse les dimensions autorisées.")

var unreadablePhoto = problem.Typed("urn:collector:probleme:corps-invalide", "Corps de requête invalide",
	http.StatusBadRequest, "La photo n'a pas pu être lue.")

type PhotoService interface {
	AddPhoto(ctx context.Context, seller annonces.SellerID, id annonces.ArticleID, raw []byte) (annonces.PhotoLink, error)
	RemovePhoto(ctx context.Context, seller annonces.SellerID, id annonces.ArticleID, photo annonces.PhotoID) error
	OwnPhotos(ctx context.Context, seller annonces.SellerID, id annonces.ArticleID) ([]annonces.PhotoLink, error)
	PublishedPhotos(ctx context.Context, id annonces.ArticleID) ([]annonces.PhotoLink, error)
	ArticlePhotos(ctx context.Context, id annonces.ArticleID) ([]annonces.PhotoLink, error)
}

type PhotoRoutes struct {
	Service        PhotoService
	MaxUploadBytes int64
}

type photoResponse struct {
	ID        string    `json:"id"`
	URL       string    `json:"url"`
	ExpiresAt time.Time `json:"expireLe"`
	Width     int       `json:"largeur"`
	Height    int       `json:"hauteur"`
}

type photoListResponse struct {
	Photos []photoResponse `json:"elements"`
}

func (h handlers) addPhoto(w http.ResponseWriter, r *http.Request) {
	seller, id, found := ownedArticleOf(w, r)
	if !found {
		return
	}
	raw, rejection, rejected := h.readPhoto(w, r)
	if rejected {
		problem.Write(w, rejection)
		return
	}
	link, err := h.photos.Service.AddPhoto(r.Context(), seller, id, raw)
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	httpjson.Write(w, http.StatusCreated, toPhotoResponse(link))
}

func (h handlers) readPhoto(w http.ResponseWriter, r *http.Request) ([]byte, problem.Problem, bool) {
	raw, err := io.ReadAll(http.MaxBytesReader(w, r.Body, h.photos.MaxUploadBytes))
	if err == nil {
		return raw, problem.Problem{}, false
	}
	var tooLarge *http.MaxBytesError
	if errors.As(err, &tooLarge) {
		return nil, oversizedPhoto, true
	}
	return nil, unreadablePhoto, true
}

func (h handlers) removePhoto(w http.ResponseWriter, r *http.Request) {
	seller, id, found := ownedArticleOf(w, r)
	if !found {
		return
	}
	photo, err := annonces.ParsePhotoID(r.PathValue(photoParameter))
	if err != nil {
		problem.Write(w, problem.NotFound())
		return
	}
	if err = h.photos.Service.RemovePhoto(r.Context(), seller, id, photo); err != nil {
		h.writeError(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h handlers) ownPhotos(w http.ResponseWriter, r *http.Request) {
	seller, id, found := ownedArticleOf(w, r)
	if !found {
		return
	}
	links, err := h.photos.Service.OwnPhotos(r.Context(), seller, id)
	h.writePhotos(w, r, links, err)
}

func (h handlers) publishedPhotos(w http.ResponseWriter, r *http.Request) {
	id, found := articleIDOf(w, r)
	if !found {
		return
	}
	links, err := h.photos.Service.PublishedPhotos(r.Context(), id)
	h.writePhotos(w, r, links, err)
}

func (h handlers) articlePhotos(w http.ResponseWriter, r *http.Request) {
	id, found := articleIDOf(w, r)
	if !found {
		return
	}
	links, err := h.photos.Service.ArticlePhotos(r.Context(), id)
	h.writePhotos(w, r, links, err)
}

func (h handlers) writePhotos(w http.ResponseWriter, r *http.Request, links []annonces.PhotoLink, err error) {
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	httpjson.Write(w, http.StatusOK, toPhotoListResponse(links))
}

func articleIDOf(w http.ResponseWriter, r *http.Request) (annonces.ArticleID, bool) {
	id, err := annonces.ParseArticleID(r.PathValue("id"))
	if err != nil {
		problem.Write(w, problem.NotFound())
		return annonces.ArticleID{}, false
	}
	return id, true
}

func toPhotoResponse(link annonces.PhotoLink) photoResponse {
	return photoResponse{
		ID:        link.ID.String(),
		URL:       link.Address,
		ExpiresAt: link.ExpiresAt.UTC(),
		Width:     link.Width,
		Height:    link.Height,
	}
}

func toPhotoListResponse(links []annonces.PhotoLink) photoListResponse {
	photos := make([]photoResponse, 0, len(links))
	for _, link := range links {
		photos = append(photos, toPhotoResponse(link))
	}
	return photoListResponse{Photos: photos}
}
