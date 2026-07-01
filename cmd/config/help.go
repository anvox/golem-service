package main

import (
	"fmt"
)

const HELP_TEXT = `NAME
	golem-config - Simple service toolkit. Version %s
		Manage service configuration using environment variables.

SYNOPSIS
	golem-config [ -h | --help ] [ -v | --version ]
	[< -e <environment> > < -t <stack> > [ -s <service> ] [subcommand]]

DESCRIPTION
	This command used to manage services' environment variables, which uses AWS Parameter Store under.
	Manage through golem-config to apply our convention in resources management as default.
	AWS_REGION environment variable is required to fetch info from AWS.

	Subcommands:
	list
		Default command, list all environment variables of service/cluster with value
	get
		List environment variables with provided name.
		i.e. get DATABASE_URL AUTH_URL
	set
		Update environment variables
		i.e. set DATABASE_URL=postgresql://db.com AUTH_URL=https://auth.com
		To unset environment variables, just set to empty
		i.e. set DATABASE_URL= AUTH_URL=
		Note:
			For AWS ECS platform, we have 2 layers of environment variabe: cluster and service
			If service environment variable is not set, system will try cluster environemnt variable.
			Ref golem docker endpoint for detail. (golem-service/cmd/entrypoint for now)

	The options are as follows:
	-e <environment>, --environment <environment>
		Environment of service to perform on. Ref golem-tf.
	-t <stack>, --stack <stack>
		Stack of service to perform on. Ref golem-tf.
	-s <service>, --service <service>
		Service to perform on. Ref golem-tf-stack.
	--value-only
		Get option. Used to get a single value to feed to command.
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
