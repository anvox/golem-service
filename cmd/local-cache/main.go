package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"

	log "github.com/sirupsen/logrus"
	"nullprogram.com/x/optparse"
)

const VERSION = "v0.0.1"

func init() {
	log.SetFormatter(&log.JSONFormatter{})
	log.SetOutput(os.Stdout)
	if os.Getenv("DEBUG") == "1" {
		log.SetLevel(log.DebugLevel)
	} else {
		log.SetLevel(log.InfoLevel)
	}
}

func main() {
	options := []optparse.Option{
		{"help", 'h', optparse.KindNone},
		{"version", 'v', optparse.KindNone},
		{"key", 'k', optparse.KindRequired},
		{"path", 'p', optparse.KindRequired},
		{"cache-dir", 'c', optparse.KindRequired},
		{"restore-key", 'r', optparse.KindRequired},
	}

	knownFlagsWithValue := map[string]bool{
		"k":           true,
		"key":         true,
		"p":           true,
		"path":        true,
		"c":           true,
		"cache-dir":   true,
		"r":           true,
		"restore-key": true,
	}

	argsToParse := reorderArgs(os.Args, knownFlagsWithValue)
	results, rest, err := optparse.Parse(options, argsToParse)
	if err != nil {
		fmt.Printf("Cannot parse arguments: %v\n", err)
		printHelp()
		os.Exit(1)
	}

	var key string
	var restoreKey string
	var customCacheDir string
	var paths []string
	showHelp := false
	showVersion := false

	for _, result := range results {
		switch result.Long {
		case "help":
			showHelp = true
		case "version":
			showVersion = true
		case "key":
			key = result.Optarg
		case "restore-key":
			restoreKey = result.Optarg
		case "cache-dir":
			customCacheDir = result.Optarg
		case "path":
			if result.Optarg != "" {
				for _, p := range strings.Split(result.Optarg, ",") {
					p = strings.TrimSpace(p)
					if p != "" {
						paths = append(paths, p)
					}
				}
			}
		}
	}

	if showHelp {
		printHelp()
		os.Exit(0)
	}

	if showVersion {
		printVersion()
		os.Exit(0)
	}

	if len(rest) == 0 {
		fmt.Println("Error: subcommand (save|restore) is required.")
		printHelp()
		os.Exit(1)
	}

	cmd := rest[0]
	positionalArgs := rest[1:]

	for _, p := range positionalArgs {
		p = strings.TrimSpace(p)
		if p != "" {
			paths = append(paths, p)
		}
	}

	switch cmd {
	case "save":
		if key == "" {
			fmt.Println("Error: --key (-k) is required for save command.")
			printHelp()
			os.Exit(1)
		}
		if len(paths) == 0 {
			fmt.Println("Error: at least one directory path (--path or positional argument) is required for save command.")
			printHelp()
			os.Exit(1)
		}
		if err := runSave(key, paths, customCacheDir); err != nil {
			fmt.Fprintf(os.Stderr, "Error during save: %v\n", err)
			os.Exit(1)
		}

	case "restore":
		if key == "" {
			fmt.Println("Error: --key (-k) is required for restore command.")
			printHelp()
			os.Exit(1)
		}
		if err := runRestore(key, restoreKey, customCacheDir); err != nil {
			fmt.Fprintf(os.Stderr, "Error during restore: %v\n", err)
			os.Exit(1)
		}

	default:
		fmt.Printf("Error: unknown subcommand %q. Must be 'save' or 'restore'.\n", cmd)
		printHelp()
		os.Exit(1)
	}
}

func reorderArgs(args []string, knownFlagsWithValue map[string]bool) []string {
	if len(args) <= 1 {
		return args
	}

	bin := args[0]
	var flags []string
	var positionals []string

	i := 1
	for i < len(args) {
		arg := args[i]
		if strings.HasPrefix(arg, "-") {
			flags = append(flags, arg)
			flagName := strings.TrimLeft(arg, "-")
			if strings.Contains(flagName, "=") {
				i++
				continue
			}
			if knownFlagsWithValue[flagName] && i+1 < len(args) && !strings.HasPrefix(args[i+1], "-") {
				flags = append(flags, args[i+1])
				i++
			}
		} else {
			positionals = append(positionals, arg)
		}
		i++
	}

	result := make([]string, 0, len(args))
	result = append(result, bin)
	result = append(result, flags...)
	result = append(result, positionals...)
	return result
}

func getCacheDir(customDir string) string {
	dir := customDir
	if dir == "" {
		dir = os.Getenv("CACHE_DIR")
	}
	home := os.Getenv("HOME")
	if home == "" {
		home = "."
	}
	if dir == "" {
		repo := os.Getenv("GITHUB_REPOSITORY")
		if repo != "" {
			dir = filepath.Join(home, "runner-cache", repo)
		} else {
			dir = filepath.Join(home, "runner-cache")
		}
	}

	if strings.HasPrefix(dir, "~/") {
		dir = filepath.Join(home, dir[2:])
	} else if dir == "~" {
		dir = home
	}
	return dir
}

func runSave(key string, paths []string, customCacheDir string) error {
	cacheDir := getCacheDir(customCacheDir)

	if err := os.MkdirAll(cacheDir, 0755); err != nil {
		return fmt.Errorf("failed to create cache directory %s: %w", cacheDir, err)
	}

	exactCacheFilename := key
	if !strings.HasSuffix(exactCacheFilename, ".tar.zst") {
		exactCacheFilename += ".tar.zst"
	}
	exactCache := filepath.Join(cacheDir, exactCacheFilename)

	if fileExists(exactCache) {
		fmt.Printf("⚡ Cache file already exists for key %q. Skipping save.\n", key)
		return nil
	}

	// Clean, deduplicate, and validate existing paths
	seen := make(map[string]bool)
	var validPaths []string
	for _, p := range paths {
		cleaned := filepath.Clean(p)
		if seen[cleaned] {
			continue
		}
		seen[cleaned] = true

		if _, err := os.Stat(cleaned); err != nil {
			fmt.Printf("⚠️ Warning: cache path %q does not exist on disk, skipping\n", cleaned)
			continue
		}
		validPaths = append(validPaths, cleaned)
	}

	if len(validPaths) == 0 {
		fmt.Println("⚠️ None of the specified cache paths exist on disk. Nothing to save.")
		return nil
	}

	fmt.Println("💾 dependencies to local host disk...")

	tarArgs := append([]string{"--zstd", "-cf", exactCache}, validPaths...)
	cmd := exec.Command("tar", tarArgs...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("tar command failed: %w", err)
	}
	return nil
}

func runRestore(key string, restoreKey string, customCacheDir string) error {
	cacheDir := getCacheDir(customCacheDir)

	if err := os.MkdirAll(cacheDir, 0755); err != nil {
		return fmt.Errorf("failed to create cache directory %s: %w", cacheDir, err)
	}

	exactCacheFilename := key
	if !strings.HasSuffix(exactCacheFilename, ".tar.zst") {
		exactCacheFilename += ".tar.zst"
	}
	exactCache := filepath.Join(cacheDir, exactCacheFilename)

	if fileExists(exactCache) {
		fmt.Println("⚡ Exact cache hit! Restoring...")
		if err := extractTar(exactCache); err != nil {
			writeGithubOutput("cache-hit", "false")
			return fmt.Errorf("failed to extract exact cache: %w", err)
		}
		writeGithubOutput("cache-hit", "true")
		return nil
	}

	fmt.Println("🔄 Exact cache miss. Searching for latest available cache file...")

	prefix := restoreKey
	if prefix == "" {
		trimmedKey := strings.TrimSuffix(key, ".tar.zst")
		if lastIdx := strings.LastIndex(trimmedKey, "-"); lastIdx != -1 {
			prefix = trimmedKey[:lastIdx+1]
		} else {
			prefix = trimmedKey
		}
	}

	latestCache := findLatestCache(cacheDir, prefix)
	if latestCache != "" && fileExists(latestCache) {
		fmt.Printf("⚡ Fallback cache hit! Restoring latest: %s\n", filepath.Base(latestCache))
		if err := extractTar(latestCache); err != nil {
			writeGithubOutput("cache-hit", "false")
			return fmt.Errorf("failed to extract fallback cache: %w", err)
		}
	} else {
		fmt.Println("⚠️ No previous cache files found.")
	}
	writeGithubOutput("cache-hit", "false")
	return nil
}

func findLatestCache(cacheDir string, prefix string) string {
	entries, err := os.ReadDir(cacheDir)
	if err != nil {
		return ""
	}

	type cacheFile struct {
		path    string
		modTime int64
	}

	var matches []cacheFile
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if strings.HasPrefix(name, prefix) && strings.HasSuffix(name, ".tar.zst") {
			fullPath := filepath.Join(cacheDir, name)
			info, err := entry.Info()
			if err == nil {
				matches = append(matches, cacheFile{
					path:    fullPath,
					modTime: info.ModTime().UnixNano(),
				})
			}
		}
	}

	if len(matches) == 0 {
		return ""
	}

	sort.Slice(matches, func(i, j int) bool {
		return matches[i].modTime > matches[j].modTime
	})

	return matches[0].path
}

func fileExists(path string) bool {
	info, err := os.Stat(path)
	if os.IsNotExist(err) {
		return false
	}
	return err == nil && !info.IsDir()
}

func extractTar(archivePath string) error {
	cmd := exec.Command("tar", "--zstd", "-xf", archivePath)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func writeGithubOutput(key, value string) {
	githubOutput := os.Getenv("GITHUB_OUTPUT")
	if githubOutput == "" {
		return
	}
	f, err := os.OpenFile(githubOutput, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return
	}
	defer f.Close()
	fmt.Fprintf(f, "%s=%s\n", key, value)
}
