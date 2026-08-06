package main

import (
	"fmt"
)

const HELP_TEXT = `NAME
	golem-local-cache - Local disk caching tool. Version %s
		Save and restore cache directories on local disk.

SYNOPSIS
	golem-local-cache [ -h | --help ] [ -v | --version ]
	golem-local-cache save -k <key> [ -c <cache-dir> ] [ -p <path> ... ] [<path> ...]
	golem-local-cache restore -k <key> [ -r <restore-key> ] [ -c <cache-dir> ]

DESCRIPTION
	Save and restore build cache directories (such as node_modules, yarn cache, etc.)
	on local disk for CI/CD runners and local builds.

SUBCOMMANDS
	save
		Save specified directories into a compressed archive (.tar.zst) using the given key.

	restore
		Restore cache archive matching key or fallback to latest matching restore key.

OPTIONS
	-k <key>, --key <key>
		Cache key to save or restore (e.g. yarn-cache-123456). Required.

	-p <path>, --path <path>
		Path or directory to cache. Can be specified multiple times or comma-separated.

	-c <dir>, --cache-dir <dir>
		Directory where cache files are stored.
		Defaults to ${HOME}/runner-cache/${GITHUB_REPOSITORY}.

	-r <key>, --restore-key <key>
		Fallback prefix/key for restore if exact key miss occurs (e.g. yarn-cache-).
		If omitted, automatically derived from --key.

	-h, --help
		Show this help message.

	-v, --version
		Show version information.
`

func printHelp() {
	fmt.Printf(HELP_TEXT, VERSION)
}

func printVersion() {
	fmt.Println(VERSION)
}
