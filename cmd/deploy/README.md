# Golem Deploy Command

This command deploys and updates AWS ECS service task definitions.

## Implementation Plan

### 1. CLI Argument Parsing
* Uses `nullprogram.com/x/optparse` to define and parse options.
* Extends parsing to aggregate multi-value flags (like multiple `-e`, `-s`, `-i`, `-c` instances).
* Positional arguments: `<cluster>` and `<service>`.

### 2. Task Definition Management
* Retrieves the current active task definition of the ECS service (or from a specific baseline via `--task`).
* Clones the container definitions, volumes, and execution roles.
* Applies updates based on CLI arguments (images, tags, commands, environments, secrets, port mappings, resource limits, etc.).
* Supports "exclusive" mode for environment variables, secrets, ports, docker labels, and mounts (clearing pre-existing configurations first).
* Registers the newly built task definition revision with AWS ECS.

### 3. ECS Service Deployment
* Updates the ECS service with the newly registered task definition ARN.
* Triggers a new deployment (`forceNewDeployment = true` if specified).

### 4. Status Check & Polling
* Continuously polls ECS service deployments until the new revision is fully rolled out.
* The deployment is considered complete when:
  1. The primary deployment rollout state is `COMPLETED` (or successful).
  2. The only remaining deployment is the primary one (`len(deployments) == 1`).
  3. The number of running tasks for the new task definition equals the service's `desiredCount`.
* Monitors service events for placement or resource errors.

### 5. Rollback & Deregistration
* On deployment failure, if `--rollback` is set, updates the service back to the previous task definition revision and waits for completion.
* On deployment success, if `--deregister` is set, deregisters the older task definition to avoid cluttering the AWS account.
