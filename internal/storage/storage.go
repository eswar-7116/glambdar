package storage

import (
	"bytes"
	"context"
	"io"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsCfg "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

type S3Config struct {
	Endpoint        string `json:"endpoint"`
	Region          string `json:"region"`
	Bucket          string `json:"bucket"`
	AccessKeyID     string `json:"access_key_id"`
	SecretAccessKey string `json:"secret_access_key"`
	SessionToken    string `json:"session_token"`
	ForcePathStyle  bool   `json:"force_path_style"`
}

type S3Storage struct {
	client *s3.Client
	bucket string
}

type Storage interface {
	Upload(ctx context.Context, key string, file io.Reader) error
	Download(ctx context.Context, key string) (io.ReadCloser, error)
}

func NewS3Storage(conf S3Config) (*S3Storage, error) {
	cfg, err := awsCfg.LoadDefaultConfig(
		context.Background(),
		awsCfg.WithRegion(conf.Region),
		awsCfg.WithRequestChecksumCalculation(aws.RequestChecksumCalculationWhenRequired),
		awsCfg.WithCredentialsProvider(
			credentials.NewStaticCredentialsProvider(
				conf.AccessKeyID,
				conf.SecretAccessKey,
				conf.SessionToken,
			),
		),
	)
	if err != nil {
		return nil, err
	}

	client := s3.NewFromConfig(cfg, func(o *s3.Options) {
		if conf.Endpoint != "" {
			o.BaseEndpoint = aws.String(conf.Endpoint)
		}

		o.UsePathStyle = conf.ForcePathStyle
	})

	return &S3Storage{
		client: client,
		bucket: conf.Bucket,
	}, nil
}

func (s *S3Storage) Upload(
	ctx context.Context,
	filename string,
	file io.Reader,
) error {
	var body io.Reader = file
	if rs, ok := file.(io.ReadSeeker); ok {
		body = rs
	} else {
		data, err := io.ReadAll(file)
		if err != nil {
			return err
		}
		body = bytes.NewReader(data)
	}

	_, err := s.client.PutObject(ctx, &s3.PutObjectInput{
		Bucket:      aws.String(s.bucket),
		Key:         aws.String(filename),
		Body:        body,
		ContentType: aws.String("application/zip"),
	})

	return err
}

func (s *S3Storage) Download(
	ctx context.Context,
	key string,
) (io.ReadCloser, error) {
	result, err := s.client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return nil, err
	}

	return result.Body, nil
}
