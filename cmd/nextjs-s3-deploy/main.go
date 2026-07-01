package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

func main() {
	opts, err := parseArgs(os.Args[1:])
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n\n", err)
		printHelp()
		os.Exit(1)
	}

	if opts.Help {
		printHelp()
		os.Exit(0)
	}

	if err := runDeploy(opts); err != nil {
		fmt.Fprintf(os.Stderr, "Deployment failed: %v\n", err)
		os.Exit(1)
	}
}

func runDeploy(opts *Options) error {
	ctx := context.TODO()

	// Load default AWS SDK configuration
	cfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		return fmt.Errorf("failed to load AWS configuration: %w", err)
	}
	s3Client := s3.NewFromConfig(cfg)

	workingDir, err := filepath.Abs(opts.WorkingDir)
	if err != nil {
		return fmt.Errorf("failed to get absolute path of working directory: %w", err)
	}

	exportedDir := filepath.Join(workingDir, "exported")
	if _, err := os.Stat(exportedDir); err != nil {
		return fmt.Errorf("exported directory does not exist at %s: %w", exportedDir, err)
	}

	// Create temporary directory in system temp to preserve input content
	tempDir, err := os.MkdirTemp("", "nextjs-s3-deploy-*")
	if err != nil {
		return fmt.Errorf("failed to create temporary directory: %w", err)
	}
	defer os.RemoveAll(tempDir)

	deployDir := filepath.Join(tempDir, "deploy")
	tempExportedDir := filepath.Join(tempDir, "exported")

	var logFile *os.File
	if opts.LogFile != "" {
		absLogPath, err := filepath.Abs(opts.LogFile)
		if err != nil {
			return fmt.Errorf("failed to resolve log file path: %w", err)
		}

		if err := os.MkdirAll(filepath.Dir(absLogPath), 0755); err != nil {
			return fmt.Errorf("failed to create log file directory: %w", err)
		}

		logFile, err = os.OpenFile(absLogPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
		if err != nil {
			return fmt.Errorf("failed to open log file: %w", err)
		}
		defer logFile.Close()
	}

	fmt.Printf("📁 Directory        : %s\n", workingDir)
	fmt.Printf("📁 Deploy directory : %s (temporary)\n", deployDir)
	fmt.Printf("🪣 Target S3 bucket : %s\n", opts.S3Bucket)
	if logFile != nil {
		fmt.Printf("📝 AWS S3 log file  : %s\n", logFile.Name())
		_, _ = fmt.Fprintf(logFile, "Start deployment at %s\n", time.Now().Format(time.RFC3339))
		_, _ = fmt.Fprintf(logFile, "Directory        : %s\n", workingDir)
		_, _ = fmt.Fprintf(logFile, "Target S3 bucket : %s\n\n", opts.S3Bucket)
	}

	// Copy original exported dir to temporary directory
	if err := copyDir(exportedDir, tempExportedDir); err != nil {
		return fmt.Errorf("failed to copy exported directory to temp folder: %w", err)
	}

	if err := os.MkdirAll(deployDir, 0755); err != nil {
		return fmt.Errorf("failed to create deploy directory: %w", err)
	}

	pagesDirDst := filepath.Join(deployDir, "_pages")
	if err := os.MkdirAll(pagesDirDst, 0755); err != nil {
		return fmt.Errorf("failed to create _pages directory: %w", err)
	}

	// Move _next folder within tempDir
	nextSrc := filepath.Join(tempExportedDir, "_next")
	nextDst := filepath.Join(deployDir, "_next")
	if err := os.Rename(nextSrc, nextDst); err != nil {
		return fmt.Errorf("failed to move _next directory: %w", err)
	}

	// Move other files to _pages folder within tempDir
	entries, err := os.ReadDir(tempExportedDir)
	if err != nil {
		return fmt.Errorf("failed to read temp exported directory: %w", err)
	}
	for _, entry := range entries {
		name := entry.Name()
		if name == "_next" {
			continue
		}
		src := filepath.Join(tempExportedDir, name)
		dst := filepath.Join(pagesDirDst, name)
		if err := os.Rename(src, dst); err != nil {
			return fmt.Errorf("failed to move %s to _pages: %w", name, err)
		}
	}

	var uploadTasks []uploadTask
	var createdRelKeys []string
	var createdRelKeysMu sync.Mutex

	// Scan _next folder for upload
	nextDir := filepath.Join(deployDir, "_next")
	err = filepath.WalkDir(nextDir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		relPath, err := filepath.Rel(nextDir, path)
		if err != nil {
			return err
		}
		relPath = filepath.ToSlash(relPath)
		s3Key := fmt.Sprintf("_next/%s", relPath)
		uploadTasks = append(uploadTasks, uploadTask{
			LocalPath:    path,
			S3Key:        s3Key,
			CacheControl: "immutable,max-age=100000000,public",
			ContentType:  getContentType(path),
		})
		return nil
	})
	if err != nil {
		return fmt.Errorf("failed to scan _next directory: %w", err)
	}

	// Scan _pages folder for upload
	pagesDir := fmt.Sprintf("_pages_%s-%03d", time.Now().Format("20060102-150405"), time.Now().Nanosecond()/1000000)
	err = filepath.WalkDir(pagesDirDst, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		relPath, err := filepath.Rel(pagesDirDst, path)
		if err != nil {
			return err
		}
		relPath = filepath.ToSlash(relPath)
		s3Key := fmt.Sprintf("%s/%s", pagesDir, relPath)
		uploadTasks = append(uploadTasks, uploadTask{
			LocalPath:    path,
			S3Key:        s3Key,
			CacheControl: "max-age=0,no-cache",
			ContentType:  getContentType(path),
		})
		createdRelKeysMu.Lock()
		createdRelKeys = append(createdRelKeys, relPath)
		createdRelKeysMu.Unlock()
		return nil
	})
	if err != nil {
		return fmt.Errorf("failed to scan _pages directory: %w", err)
	}

	// Execute uploads concurrently
	if len(uploadTasks) > 0 {
		tasksChan := make(chan uploadTask, len(uploadTasks))
		for _, t := range uploadTasks {
			tasksChan <- t
		}
		close(tasksChan)

		errChan, cleanupWorkers := startUploadWorkers(ctx, s3Client, opts.S3Bucket, tasksChan, opts.Verbose, logFile)
		cleanupWorkers()
		if err := <-errChan; err != nil {
			return fmt.Errorf("upload failed: %w", err)
		}
	}
	fmt.Println("🫸 _next")
	fmt.Printf("🫸 _pages to %s\n", pagesDir)

	// Copy HTML pages without extensions parallelly
	var htmlCopyTasks []copyTask
	var newKeys []string
	for _, relPath := range createdRelKeys {
		if strings.HasSuffix(relPath, ".html") {
			withoutExt := strings.TrimSuffix(relPath, ".html")
			htmlCopyTasks = append(htmlCopyTasks, copyTask{
				SrcKey:       fmt.Sprintf("%s/%s", pagesDir, relPath),
				DstKey:       fmt.Sprintf("%s/%s", pagesDir, withoutExt),
				CacheControl: "max-age=0,no-cache",
				ContentType:  "text/html",
			})
			newKeys = append(newKeys, withoutExt)
		}
	}
	createdRelKeys = append(createdRelKeys, newKeys...)

	if len(htmlCopyTasks) > 0 {
		copyChan := make(chan copyTask, len(htmlCopyTasks))
		for _, task := range htmlCopyTasks {
			copyChan <- task
		}
		close(copyChan)

		errChan, cleanupCopies := startCopyWorkers(ctx, s3Client, opts.S3Bucket, copyChan, opts.Verbose, logFile)
		cleanupCopies()
		if err := <-errChan; err != nil {
			return fmt.Errorf("HTML copy without extension failed: %w", err)
		}
	}
	fmt.Println("🫸 html pages without extension")

	// Final sync to the pages directory
	var finalSyncTasks []copyTask
	for _, relKey := range createdRelKeys {
		contentType := getContentType(relKey)
		if !strings.Contains(relKey, ".") {
			contentType = "text/html"
		}
		finalSyncTasks = append(finalSyncTasks, copyTask{
			SrcKey:       fmt.Sprintf("%s/%s", pagesDir, relKey),
			DstKey:       fmt.Sprintf("pages/%s", relKey),
			CacheControl: "max-age=0,no-cache",
			ContentType:  contentType,
		})
	}

	if len(finalSyncTasks) > 0 {
		syncChan := make(chan copyTask, len(finalSyncTasks))
		for _, task := range finalSyncTasks {
			syncChan <- task
		}
		close(syncChan)

		errChan, cleanupSync := startCopyWorkers(ctx, s3Client, opts.S3Bucket, syncChan, opts.Verbose, logFile)
		cleanupSync()
		if err := <-errChan; err != nil {
			return fmt.Errorf("final sync failed: %w", err)
		}
	}

	// Delete obsolete keys under pages/
	allTargetKeys, err := listS3Keys(ctx, s3Client, opts.S3Bucket, "pages/")
	if err != nil {
		return fmt.Errorf("failed to list target keys under pages/: %w", err)
	}

	createdRelSet := make(map[string]bool)
	for _, k := range createdRelKeys {
		createdRelSet[k] = true
	}

	var obsoleteKeys []string
	for _, key := range allTargetKeys {
		relKey := strings.TrimPrefix(key, "pages/")
		if !createdRelSet[relKey] {
			obsoleteKeys = append(obsoleteKeys, key)
		}
	}

	if err := deleteObsoleteKeys(ctx, s3Client, opts.S3Bucket, obsoleteKeys, opts.Verbose, logFile); err != nil {
		return fmt.Errorf("failed to delete obsolete keys: %w", err)
	}

	fmt.Printf("🫸 📁%s -> 📁pages\n", pagesDir)
	fmt.Println("🚀🌍")
	return nil
}
