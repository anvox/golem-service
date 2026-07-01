package main

import (
	"fmt"
)

const VERSION = "v0.0.1"

const HELP_TEXT = `NAME
	golem-deploy - ECS Service Deployment tool. Version %s

SYNOPSIS
	golem-deploy <cluster> <service> [options]

DESCRIPTION
	Redeploy or modify an ECS service.
	When not giving any modification options, the task definition will just be duplicated
	(so that all container images will be pulled and redeployed).

ARGUMENTS
	<cluster>
		The name of your ECS cluster.
	<service>
		The name of your ECS service.

OPTIONS
	-t, --tag <tag>
		Changes the tag for ALL container images in the task definition.
	-i, --image <container> <image>
		Overwrites the image for a container. (Can be specified multiple times).
	-c, --command <container> <command>
		Overwrites the command in a container. (Can be specified multiple times).
	-e, --env <container> <name> <value>
		Adds or changes an environment variable in a container. (Can be specified multiple times).
	--env-file <container> <env_file_path>
		Loads environment variables from a local file for a container. (Can be specified multiple times).
	--s3-env-file <container> <s3_arn>
		Adds an environment file from S3 to a container in ARN format. (Can be specified multiple times).
	-s, --secret <container> <name> <parameter_name>
		Adds or changes a secret variable from AWS SSM Parameter Store. (Can be specified multiple times).
	--secrets-env-file <container> <env_file_path>
		Loads secrets from a local env file. (Can be specified multiple times).
	-d, --docker-label <container> <name> <value>
		Adds or changes a docker label in a container. (Can be specified multiple times).
	-u, --ulimit <container> <name> <soft_limit> <hard_limit>
		Adds or changes a ulimit in a container. (Can be specified multiple times).
	--system-control <container> <namespace> <value>
		Adds or changes a sysctl variable in a container. (Can be specified multiple times).
	-p, --port <container> <container_port> <host_port>
		Adds or changes a port mapping. (Can be specified multiple times).
	-m, --mount <container> <volume_name> <container_path>
		Adds or changes a mount point. (Can be specified multiple times).
	-l, --log <container> <log_driver> <option_name> <option_value>
		Adds or changes a log configuration option. (Can be specified multiple times).
	--cpu <container> <cpu>
		Overwrites the CPU value for a container. (Can be specified multiple times).
	--memory <container> <memory>
		Overwrites the memory value for a container. (Can be specified multiple times).
	--memoryreservation <container> <memory_reservation>
		Overwrites the memory reservation for a container. (Can be specified multiple times).
	--privileged <container> <true|false>
		Overwrites the privileged flag for a container. (Can be specified multiple times).
	--essential <container> <true|false>
		Overwrites the essential flag for a container. (Can be specified multiple times).
	--task-cpu <cpu>
		Overwrites the CPU value for the task.
	--task-memory <memory>
		Overwrites the memory value for the task.
	-r, --role <role_arn>
		Sets the task's role ARN.
	-x, --execution-role <execution_role_arn>
		Sets the task's execution role ARN.
	--runtime-platform <cpu_arch> <os_family>
		Overwrites runtime platform (e.g., X86_64 LINUX).
	--task <task_arn_or_family>
		Task definition to use as baseline. Defaults to the service's current task definition.
	--region <region>
		AWS region.
	--access-key-id <key_id>
		AWS access key ID.
	--secret-access-key <secret_key>
		AWS secret access key.
	--profile <profile_name>
		AWS profile name.
	--account <account_id>
		AWS account ID.
	--assume-role <role_name>
		IAM Role to assume in target account.
	--timeout <seconds>
		Seconds to wait for deployment before command fails (default: 300).
		Set to -1 for fire-and-forget (do not wait).
	--force-new-deployment
		Force ECS to start a new deployment.
	--ignore-warnings
		Do not fail deployment on placement/resource warnings.
	--sleep-time <seconds>
		Seconds to wait between deployment status checks (default: 1).
	--diff, --no-diff
		Print which values were changed (default: --diff).
	--deregister, --no-deregister
		Deregister the old task definition revision on success (default: --deregister).
	--rollback, --no-rollback
		Rollback to previous revision if deployment fails (default: --no-rollback).
	--exclusive-env
		Remove all pre-existing env variables from all containers.
	--exclusive-secrets
		Remove all pre-existing secrets from all containers.
	--exclusive-docker-labels
		Remove all pre-existing docker labels from all containers.
	--exclusive-s3-env-file
		Remove all pre-existing S3 env files from all containers.
	--exclusive-ulimits
		Remove all pre-existing ulimits.
	--exclusive-system-controls
		Remove all pre-existing system controls.
	--exclusive-ports
		Remove all pre-existing port mappings.
	--exclusive-mounts
		Remove all pre-existing mount points.
	--volume <volume_name> <host_path>
		Adds a volume mapping to the task definition (Can be specified multiple times).
	--add-container <container_name>
		Add a placeholder container to the task definition (Can be specified multiple times).
	--remove-container <container_name>
		Remove a container from the task definition (Can be specified multiple times).
	-h, --help
		Show this help text.
	-v, --version
		Show version.
`

func printHelp() {
	fmt.Printf(HELP_TEXT, VERSION)
}

func printVersion() {
	fmt.Printf("%s\n", VERSION)
}
