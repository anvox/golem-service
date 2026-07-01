package main

import (
	"context"
	"fmt"
	"os"
	"sync"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
)

type uploadTask struct {
	LocalPath    string
	S3Key        string
	CacheControl string
	ContentType  string
}

type copyTask struct {
	SrcKey       string
	DstKey       string
	CacheControl string
	ContentType  string
}

func startUploadWorkers(ctx context.Context, s3Client *s3.Client, bucket string, tasks chan uploadTask, verbose bool, logFile *os.File) (chan error, func()) {
	var wg sync.WaitGroup
	errChan := make(chan error, 1)
	var errOnce sync.Once

	var logMu sync.Mutex
	logMsg := func(format string, a ...interface{}) {
		logMu.Lock()
		defer logMu.Unlock()
		msg := fmt.Sprintf(format, a...)
		if logFile != nil {
			_, _ = fmt.Fprint(logFile, msg)
		}
		if logFile == nil || verbose {
			fmt.Fprint(os.Stdout, msg)
		}
	}

	numWorkers := 10
	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for task := range tasks {
				file, err := os.Open(task.LocalPath)
				if err != nil {
					errOnce.Do(func() { errChan <- fmt.Errorf("failed to open file %s: %w", task.LocalPath, err) })
					return
				}

				_, err = s3Client.PutObject(ctx, &s3.PutObjectInput{
					Bucket:       aws.String(bucket),
					Key:          aws.String(task.S3Key),
					Body:         file,
					CacheControl: aws.String(task.CacheControl),
					ContentType:  aws.String(task.ContentType),
				})
				file.Close()
				if err != nil {
					errOnce.Do(func() { errChan <- fmt.Errorf("failed to upload %s to s3://%s/%s: %w", task.LocalPath, bucket, task.S3Key, err) })
					return
				}

				logMsg("upload: s3://%s/%s\n", bucket, task.S3Key)
			}
		}()
	}

	cleanup := func() {
		wg.Wait()
		close(errChan)
	}

	return errChan, cleanup
}

func startCopyWorkers(ctx context.Context, s3Client *s3.Client, bucket string, tasks chan copyTask, verbose bool, logFile *os.File) (chan error, func()) {
	var wg sync.WaitGroup
	errChan := make(chan error, 1)
	var errOnce sync.Once

	var logMu sync.Mutex
	logMsg := func(format string, a ...interface{}) {
		logMu.Lock()
		defer logMu.Unlock()
		msg := fmt.Sprintf(format, a...)
		if logFile != nil {
			_, _ = fmt.Fprint(logFile, msg)
		}
		if logFile == nil || verbose {
			fmt.Fprint(os.Stdout, msg)
		}
	}

	numWorkers := 10
	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for task := range tasks {
				copySource := fmt.Sprintf("%s/%s", bucket, task.SrcKey)
				input := &s3.CopyObjectInput{
					Bucket:            aws.String(bucket),
					CopySource:        aws.String(copySource),
					Key:               aws.String(task.DstKey),
					CacheControl:      aws.String(task.CacheControl),
					MetadataDirective: types.MetadataDirectiveReplace,
				}
				if task.ContentType != "" {
					input.ContentType = aws.String(task.ContentType)
				}

				_, err := s3Client.CopyObject(ctx, input)
				if err != nil {
					errOnce.Do(func() { errChan <- fmt.Errorf("failed to copy s3://%s to s3://%s/%s: %w", copySource, bucket, task.DstKey, err) })
					return
				}

				logMsg("copy: s3://%s -> s3://%s/%s\n", copySource, bucket, task.DstKey)
			}
		}()
	}

	cleanup := func() {
		wg.Wait()
		close(errChan)
	}

	return errChan, cleanup
}

func listS3Keys(ctx context.Context, s3Client *s3.Client, bucket, prefix string) ([]string, error) {
	var keys []string
	paginator := s3.NewListObjectsV2Paginator(s3Client, &s3.ListObjectsV2Input{
		Bucket: aws.String(bucket),
		Prefix: aws.String(prefix),
	})

	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to list keys under prefix %s: %w", prefix, err)
		}

		for _, obj := range page.Contents {
			key := aws.ToString(obj.Key)
			if key == "pages/" {
				continue
			}
			keys = append(keys, key)
		}
	}

	return keys, nil
}

func deleteObsoleteKeys(ctx context.Context, s3Client *s3.Client, bucket string, obsoleteKeys []string, verbose bool, logFile *os.File) error {
	if len(obsoleteKeys) == 0 {
		return nil
	}

	var logMu sync.Mutex
	logMsg := func(format string, a ...interface{}) {
		logMu.Lock()
		defer logMu.Unlock()
		msg := fmt.Sprintf(format, a...)
		if logFile != nil {
			_, _ = fmt.Fprint(logFile, msg)
		}
		if logFile == nil || verbose {
			fmt.Fprint(os.Stdout, msg)
		}
	}

	const batchSize = 1000
	for i := 0; i < len(obsoleteKeys); i += batchSize {
		end := i + batchSize
		if end > len(obsoleteKeys) {
			end = len(obsoleteKeys)
		}

		batch := obsoleteKeys[i:end]
		var objectIds []types.ObjectIdentifier
		for _, key := range batch {
			objectIds = append(objectIds, types.ObjectIdentifier{
				Key: aws.String(key),
			})
		}

		_, err := s3Client.DeleteObjects(ctx, &s3.DeleteObjectsInput{
			Bucket: aws.String(bucket),
			Delete: &types.Delete{
				Objects: objectIds,
				Quiet:   aws.Bool(true),
			},
		})
		if err != nil {
			return fmt.Errorf("failed to delete batch of obsolete keys: %w", err)
		}

		for _, key := range batch {
			logMsg("delete: s3://%s/%s\n", bucket, key)
		}
	}

	return nil
}
