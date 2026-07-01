package main

import (
	"fmt"
)

const HELP_TEXT = `NAME
nextjs-s3-deploy - deploy an exported standalone nextjs app to S3.

USAGE:
  nextjs-s3-deploy <working-dir> --s3-bucket <bucket-name> [--log <log-file>] [--verbose]

ARGUMENTS:
  <working-dir>              Working directory containing Next.js build (e.g. dist/apps/kds)

FLAGS:
  --s3-bucket <bucket-name>  Target S3 bucket name (required)
  --log <log-file>           Path to the upload log file (optional)
  -v, --verbose              Print out syncing file list, along with writing to log file
  -h, --help                 Show this help message
`

func printHelp() {
	fmt.Print(HELP_TEXT)
}
