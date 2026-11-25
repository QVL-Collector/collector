package objets

import (
	"bytes"
	"context"
	"crypto/rand"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"

	"collector/services/place-de-marche/internal/annonces"
)

const (
	keyPrefix    = "photos/"
	keySuffix    = ".jpg"
	photoType    = "image/jpeg"
	photoCaching = "private, max-age=600, no-transform"
)

type Settings struct {
	Bucket       string
	Region       string
	Endpoint     string
	PathStyle    bool
	LinkLifetime time.Duration
}

type Store struct {
	client    *s3.Client
	presigner *s3.PresignClient
	bucket    string
	lifetime  time.Duration
}

func NewStore(ctx context.Context, settings Settings) (*Store, error) {
	shared, err := awsconfig.LoadDefaultConfig(ctx, awsconfig.WithRegion(settings.Region))
	if err != nil {
		return nil, fmt.Errorf("configuration du client S3 : %w", err)
	}
	client := s3.NewFromConfig(shared, func(options *s3.Options) {
		options.UsePathStyle = settings.PathStyle
		options.RequestChecksumCalculation = aws.RequestChecksumCalculationWhenRequired
		options.ResponseChecksumValidation = aws.ResponseChecksumValidationWhenRequired
		if settings.Endpoint != "" {
			options.BaseEndpoint = aws.String(settings.Endpoint)
		}
	})
	return &Store{
		client:    client,
		presigner: s3.NewPresignClient(client),
		bucket:    settings.Bucket,
		lifetime:  settings.LinkLifetime,
	}, nil
}

func (s *Store) NewKey() annonces.PhotoKey {
	return annonces.PhotoKey(keyPrefix + rand.Text() + keySuffix)
}

func (s *Store) Put(ctx context.Context, key annonces.PhotoKey, content []byte) error {
	_, err := s.client.PutObject(ctx, &s3.PutObjectInput{
		Bucket:        aws.String(s.bucket),
		Key:           aws.String(string(key)),
		Body:          bytes.NewReader(content),
		ContentLength: aws.Int64(int64(len(content))),
		ContentType:   aws.String(photoType),
		CacheControl:  aws.String(photoCaching),
	})
	if err != nil {
		return fmt.Errorf("dépôt de l'objet : %w", err)
	}
	return nil
}

func (s *Store) Delete(ctx context.Context, key annonces.PhotoKey) error {
	_, err := s.client.DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(string(key)),
	})
	if err != nil {
		return fmt.Errorf("suppression de l'objet : %w", err)
	}
	return nil
}

func (s *Store) Link(ctx context.Context, key annonces.PhotoKey) (annonces.PhotoURL, error) {
	issuedAt := time.Now()
	signed, err := s.presigner.PresignGetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(string(key)),
	}, s3.WithPresignExpires(s.lifetime))
	if err != nil {
		return annonces.PhotoURL{}, fmt.Errorf("présignature de l'objet : %w", err)
	}
	return annonces.PhotoURL{Address: signed.URL, ExpiresAt: issuedAt.Add(s.lifetime).UTC()}, nil
}
