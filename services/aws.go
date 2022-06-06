package services

import (
	"context"
	"io"
	"os"

	logger "github.com/sirupsen/logrus"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/feature/s3/manager"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go/aws"
)

func UploadS3(key string, data io.Reader) error {
	cfg, err := config.LoadDefaultConfig(context.Background())
	if err != nil {
		logger.WithFields(logger.Fields{
			"key":    key,
			"bucket": os.Getenv("BACKUP_BUCKET"),
			"region": os.Getenv("AWS_REGION"),
		}).Error("upload config failed")
		return err
	}
	s3c := s3.NewFromConfig(cfg)
	uploader := manager.NewUploader(s3c, func(u *manager.Uploader) {
		// Define a strategy that will buffer 25 MiB in memory
		u.BufferProvider = manager.NewBufferedReadSeekerWriteToPool(25 * 1024 * 1024)
	})
	_, err = uploader.Upload(context.TODO(), &s3.PutObjectInput{
		Bucket: aws.String(os.Getenv("BACKUP_BUCKET")),
		Key:    aws.String(key),
		Body:   data,
	})
	if err != nil {
		logger.WithFields(logger.Fields{
			"key":    key,
			"bucket": os.Getenv("BACKUP_BUCKET"),
			"region": os.Getenv("AWS_REGION"),
		}).Error("upload failed")
		return err
	}
	logger.WithFields(logger.Fields{
		"key":    key,
		"bucket": os.Getenv("BACKUP_BUCKET"),
	}).Info("upload completed")
	return nil
}
