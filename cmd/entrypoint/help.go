package main

import (
	"fmt"
)

const HELP_TEXT = `NAME
	golem-entrypoint - Simple service toolkit. Version %s
		Entrypoint for golem services run on AWS ECS.

SYNOPSIS
	golem-entrypoint [ -h | --help ] [ -v | --version ] <any-supported-proc-type>

DESCRIPTION
	Entrypoint for golem services, supports:
		* Parameter Store as container environment variable
		* SSH
		* Local wait
		* Development .env environment file

	Supported proc types:
		* ssh
		* wait
		* Procfile
		* Procfile.development (for local development environment only)
		* bash (for local development environment only)

	-h, --help
		Show this help text.
	-v, --version
		Show version
`

func printHelp() {
	fmt.Printf(HELP_TEXT, VERSION)
}

func printVersion() {
	fmt.Printf("%s", VERSION)
}
