package storage

import (
	"bytes"
	"context"
	"errors"
	"io"
	"strings"

	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
)

// S3Config holds configuration for the S3 client.
type S3Config struct {
	Endpoint        string
	Region          string
	Bucket          string
	AccessKeyID     string
	SecretAccessKey string
}

// S3Store implements Store using an S3-compatible backend.
type S3Store struct {
	client *s3.Client
	bucket string
}

// NewS3Store creates a new S3-backed store.
func NewS3Store(cfg S3Config) *S3Store {
	region := cfg.Region
	if region == "" {
		region = "auto"
	}

	client := s3.New(s3.Options{
		Region: region,
		BaseEndpoint: &cfg.Endpoint,
		Credentials:  credentials.NewStaticCredentialsProvider(cfg.AccessKeyID, cfg.SecretAccessKey, ""),
	})

	return &S3Store{
		client: client,
		bucket: cfg.Bucket,
	}
}

func (s *S3Store) Get(ctx context.Context, key string) ([]byte, string, error) {
	out, err := s.client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: &s.bucket,
		Key:    &key,
	})
	if err != nil {
		if isNotFound(err) {
			return nil, "", ErrNotFound
		}
		return nil, "", err
	}
	defer out.Body.Close()

	data, err := io.ReadAll(out.Body)
	if err != nil {
		return nil, "", err
	}

	etag := ""
	if out.ETag != nil {
		etag = *out.ETag
	}
	return data, etag, nil
}

func (s *S3Store) Put(ctx context.Context, key string, data []byte, ifMatchETag string) (string, error) {
	input := &s3.PutObjectInput{
		Bucket: &s.bucket,
		Key:    &key,
		Body:   bytes.NewReader(data),
	}

	if ifMatchETag != "" {
		input.IfMatch = &ifMatchETag
	}

	out, err := s.client.PutObject(ctx, input)
	if err != nil {
		if isPreconditionFailed(err) {
			return "", ErrConflict
		}
		return "", err
	}

	etag := ""
	if out.ETag != nil {
		etag = *out.ETag
	}
	return etag, nil
}

func (s *S3Store) Head(ctx context.Context, key string) (string, error) {
	out, err := s.client.HeadObject(ctx, &s3.HeadObjectInput{
		Bucket: &s.bucket,
		Key:    &key,
	})
	if err != nil {
		if isNotFound(err) {
			return "", ErrNotFound
		}
		return "", err
	}

	etag := ""
	if out.ETag != nil {
		etag = *out.ETag
	}
	return etag, nil
}

func (s *S3Store) List(ctx context.Context, prefix string) ([]string, error) {
	var keys []string

	paginator := s3.NewListObjectsV2Paginator(s.client, &s3.ListObjectsV2Input{
		Bucket: &s.bucket,
		Prefix: &prefix,
	})

	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, err
		}
		for _, obj := range page.Contents {
			if obj.Key != nil {
				keys = append(keys, *obj.Key)
			}
		}
	}

	return keys, nil
}

func (s *S3Store) Delete(ctx context.Context, key string) error {
	_, err := s.client.DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket: &s.bucket,
		Key:    &key,
	})
	return err
}

func isNotFound(err error) bool {
	var nsk *types.NoSuchKey
	if errors.As(err, &nsk) {
		return true
	}
	var nsb *types.NotFound
	if errors.As(err, &nsb) {
		return true
	}
	return strings.Contains(err.Error(), "NoSuchKey") || strings.Contains(err.Error(), "404")
}

func isPreconditionFailed(err error) bool {
	return strings.Contains(err.Error(), "PreconditionFailed") || strings.Contains(err.Error(), "412")
}
