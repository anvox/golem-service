package main

import (
	"fmt"
)

const HELP_TEXT = `NAME
	golem-ps - Simple service toolkit. Version %s
		Manage processes run in services

SYNOPSIS
	golem-ps [ -h | --help ] [ -v | --version ]
	[< -e <environment> > < -t <stack> > < -s <service> > [ -p <process> ] [subcommand]]

DESCRIPTION
	Manage processes run under a service.
	AWS_REGION environment variable is required to fetch info from AWS.

	Subcommands:
	list
		Default command, list all running processes of service/cluster with value
	kill
		Stop a process or a group of processes of service
		i.e.

		# Stop all web and sidekiq proc type of service
		golem-ps -e<environment> -t<stack> -s<service> --process=web,sidekiq kill

		# Stop a specific task from id get from ps
		golem-ps -e<environment> -t<stack> -s<service> kill <task-id>

	The options are as follows:
	-e <environment>, --environment <environment>
		Environment of service to perform on. Ref golem-tf.
	-t <stack>, --stack <stack>
		Stack of service to perform on. Ref golem-tf.
	-s <service>, --service=<service>
		Service to perform on. Ref golem-tf-stack.
	-p <process>, --process <process>
		Process type of processes to perform on. Ref golem-tf-stack.
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
