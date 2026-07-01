package main

import (
	"fmt"
)

const HELP_TEXT = `NAME
	golem-exec-remote - Simple service toolkit. Version %s
		Open SSH connection to a service support SSH by golem-entrypoint

SYNOPSIS
	golem-exec-remote [ -h | --help ] [ -v | --version ]

DESCRIPTION
	Open SSH connection to a service support SSH by golem-entrypoint

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
