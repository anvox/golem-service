package main

import (
	"fmt"
	"strings"
)

type Options struct {
	WorkingDir string
	S3Bucket   string
	LogFile    string
	Verbose    bool
	Help       bool
}

func parseArgs(args []string) (*Options, error) {
	opts := &Options{}
	var positionals []string

	for i := 0; i < len(args); i++ {
		arg := args[i]
		if arg == "-h" || arg == "--help" {
			opts.Help = true
			continue
		}
		if arg == "-v" || arg == "--verbose" {
			opts.Verbose = true
			continue
		}
		if strings.HasPrefix(arg, "--s3-bucket=") {
			opts.S3Bucket = strings.TrimPrefix(arg, "--s3-bucket=")
			continue
		}
		if arg == "--s3-bucket" {
			if i+1 >= len(args) {
				return nil, fmt.Errorf("missing value for --s3-bucket")
			}
			opts.S3Bucket = args[i+1]
			i++
			continue
		}
		if strings.HasPrefix(arg, "--log=") {
			opts.LogFile = strings.TrimPrefix(arg, "--log=")
			continue
		}
		if arg == "--log" {
			if i+1 >= len(args) {
				return nil, fmt.Errorf("missing value for --log")
			}
			opts.LogFile = args[i+1]
			i++
			continue
		}
		if strings.HasPrefix(arg, "-") {
			return nil, fmt.Errorf("unknown flag: %s", arg)
		}
		positionals = append(positionals, arg)
	}

	if opts.Help {
		return opts, nil
	}

	if len(positionals) == 0 {
		return nil, fmt.Errorf("working directory is required")
	}
	if len(positionals) > 1 {
		return nil, fmt.Errorf("too many positional arguments (expected 1, got %d)", len(positionals))
	}
	opts.WorkingDir = positionals[0]

	if opts.S3Bucket == "" {
		return nil, fmt.Errorf("--s3-bucket is required")
	}

	return opts, nil
}
