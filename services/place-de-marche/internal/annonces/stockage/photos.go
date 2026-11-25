package stockage

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"collector/services/place-de-marche/internal/annonces"
)

const activePhoto = `retiree_le IS NULL`

const photoCount = `(SELECT count(*) FROM photo WHERE photo.article_id = article.id AND photo.` + activePhoto + `)`

const (
	photoStateQuery       = `SELECT etat, revision, ` + reservedOrSold + `, ` + photoCount + ` FROM article WHERE id = $1 AND vendeur_id = $2`
	lockedPhotoStateQuery = photoStateQuery + ` FOR UPDATE OF article`
	countPhotosQuery      = `SELECT count(*) FROM photo WHERE article_id = $1 AND ` + activePhoto
)

const insertPhotoQuery = `
INSERT INTO photo (article_id, cle_objet, largeur, hauteur, taille_octets)
VALUES ($1, $2, $3, $4, $5)
RETURNING id, cle_objet, largeur, hauteur`

const withdrawPhotoQuery = `
UPDATE photo
SET retiree_le = now()
WHERE id = $1 AND article_id = $2 AND ` + activePhoto + `
RETURNING id, cle_objet, largeur, hauteur`

const (
	photosOfSelectedArticle = `SELECT p.id, p.cle_objet, p.largeur, p.hauteur FROM (`
	selectedArticlePhotos   = `) AS a
LEFT JOIN photo AS p ON p.article_id = a.id AND p.` + activePhoto + `
ORDER BY p.cree_le, p.id`
	ownPhotosQuery       = photosOfSelectedArticle + `SELECT id FROM article WHERE id = $1 AND vendeur_id = $2` + selectedArticlePhotos
	publishedPhotosQuery = photosOfSelectedArticle + `SELECT a.id` + publishedArticles + ` AND a.id = $1` + selectedArticlePhotos
	articlePhotosQuery   = photosOfSelectedArticle + `SELECT id FROM article WHERE id = $1` + selectedArticlePhotos
)

func (s *Store) CheckPhotoAddable(ctx context.Context, seller annonces.SellerID, id annonces.ArticleID, limit int) error {
	state, err := photoState(ctx, s.pool, photoStateQuery, seller, id)
	if err != nil {
		return err
	}
	return state.VerifyAddable(limit)
}

func (s *Store) AddPhoto(ctx context.Context, seller annonces.SellerID, id annonces.ArticleID, photo annonces.StoredPhoto, limit int) (annonces.Photo, error) {
	var added annonces.Photo
	err := pgx.BeginFunc(ctx, s.pool, func(tx pgx.Tx) error {
		state, err := photoState(ctx, tx, lockedPhotoStateQuery, seller, id)
		if err != nil {
			return err
		}
		if err = state.VerifyAddable(limit); err != nil {
			return err
		}
		added, err = scanPhoto(tx.QueryRow(ctx, insertPhotoQuery,
			uuid.UUID(id), string(photo.Key), photo.Width, photo.Height, photo.Size,
		))
		return err
	})
	if err != nil {
		return annonces.Photo{}, fmt.Errorf("ajout de la photo : %w", err)
	}
	return added, nil
}

func (s *Store) RemovePhoto(ctx context.Context, seller annonces.SellerID, id annonces.ArticleID, photo annonces.PhotoID) (annonces.Photo, error) {
	var removed annonces.Photo
	err := pgx.BeginFunc(ctx, s.pool, func(tx pgx.Tx) error {
		state, err := photoState(ctx, tx, lockedPhotoStateQuery, seller, id)
		if err != nil {
			return err
		}
		if err = state.Article.VerifyPhotosEditable(); err != nil {
			return err
		}
		removed, err = scanPhoto(tx.QueryRow(ctx, withdrawPhotoQuery, uuid.UUID(photo), uuid.UUID(id)))
		if errors.Is(err, pgx.ErrNoRows) {
			return annonces.ErrPhotoNotFound
		}
		return err
	})
	if err != nil {
		return annonces.Photo{}, fmt.Errorf("retrait de la photo : %w", err)
	}
	return removed, nil
}

func (s *Store) OwnPhotos(ctx context.Context, seller annonces.SellerID, id annonces.ArticleID) ([]annonces.Photo, error) {
	return s.photosOf(ctx, ownPhotosQuery, uuid.UUID(id), uuid.UUID(seller))
}

func (s *Store) PublishedPhotos(ctx context.Context, id annonces.ArticleID) ([]annonces.Photo, error) {
	return s.photosOf(ctx, publishedPhotosQuery, uuid.UUID(id))
}

func (s *Store) ArticlePhotos(ctx context.Context, id annonces.ArticleID) ([]annonces.Photo, error) {
	return s.photosOf(ctx, articlePhotosQuery, uuid.UUID(id))
}

func (s *Store) photosOf(ctx context.Context, query string, args ...any) ([]annonces.Photo, error) {
	rows, err := s.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("liste des photos : %w", err)
	}
	selected, err := pgx.CollectRows(rows, func(row pgx.CollectableRow) (optionalPhotoRow, error) {
		var r optionalPhotoRow
		err := row.Scan(&r.id, &r.key, &r.width, &r.height)
		return r, err
	})
	if err != nil {
		return nil, fmt.Errorf("lecture des photos : %w", err)
	}
	if len(selected) == 0 {
		return nil, annonces.ErrArticleNotFound
	}
	return presentPhotos(selected), nil
}

func countPhotos(ctx context.Context, tx pgx.Tx, id annonces.ArticleID) (int, error) {
	var count int
	if err := tx.QueryRow(ctx, countPhotosQuery, uuid.UUID(id)).Scan(&count); err != nil {
		return 0, fmt.Errorf("décompte des photos : %w", err)
	}
	return count, nil
}

func photoState(ctx context.Context, querier rowQuerier, query string, seller annonces.SellerID, id annonces.ArticleID) (annonces.ArticlePhotoState, error) {
	var (
		status   string
		revision int
		reserved bool
		photos   int
	)
	err := querier.QueryRow(ctx, query, uuid.UUID(id), uuid.UUID(seller)).Scan(&status, &revision, &reserved, &photos)
	if errors.Is(err, pgx.ErrNoRows) {
		return annonces.ArticlePhotoState{}, annonces.ErrArticleNotFound
	}
	if err != nil {
		return annonces.ArticlePhotoState{}, fmt.Errorf("lecture de l'état des photos de l'article : %w", err)
	}
	return annonces.ArticlePhotoState{
		Article: annonces.ArticleState{Status: annonces.Status(status), Revision: annonces.Revision(revision), Reserved: reserved},
		Photos:  photos,
	}, nil
}

func scanPhoto(row pgx.Row) (annonces.Photo, error) {
	var (
		id     uuid.UUID
		key    string
		width  int
		height int
	)
	if err := row.Scan(&id, &key, &width, &height); err != nil {
		return annonces.Photo{}, err
	}
	return annonces.Photo{
		ID:         annonces.PhotoID(id),
		Key:        annonces.PhotoKey(key),
		Dimensions: annonces.Dimensions{Width: width, Height: height},
	}, nil
}

type optionalPhotoRow struct {
	id     *uuid.UUID
	key    *string
	width  *int
	height *int
}

func presentPhotos(selected []optionalPhotoRow) []annonces.Photo {
	photos := make([]annonces.Photo, 0, len(selected))
	for _, r := range selected {
		if r.id == nil || r.key == nil || r.width == nil || r.height == nil {
			continue
		}
		photos = append(photos, annonces.Photo{
			ID:         annonces.PhotoID(*r.id),
			Key:        annonces.PhotoKey(*r.key),
			Dimensions: annonces.Dimensions{Width: *r.width, Height: *r.height},
		})
	}
	return photos
}
