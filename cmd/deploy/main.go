package main

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/credentials/stscreds"
	"github.com/aws/aws-sdk-go-v2/service/ecs"
	"github.com/aws/aws-sdk-go-v2/service/ecs/types"
	"github.com/aws/aws-sdk-go-v2/service/sts"
	"github.com/fatih/color"
	log "github.com/sirupsen/logrus"
)

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
	parsed, err := parseCLI(os.Args[1:])
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		printHelp()
		os.Exit(1)
	}

	if parsed.Help {
		printHelp()
		os.Exit(0)
	}

	if parsed.Version {
		printVersion()
		os.Exit(0)
	}

	// 1. Initialize AWS ECS client
	ecsClient, err := createEcsClient(parsed)
	if err != nil {
		color.Red("Failed to initialize AWS client: %v", err)
		os.Exit(1)
	}

	// 2. Fetch baseline Task Definition
	var baseTaskArn string
	if parsed.Task != "" {
		baseTaskArn = parsed.Task
	} else {
		serviceOut, err := ecsClient.DescribeServices(context.TODO(), &ecs.DescribeServicesInput{
			Cluster:  aws.String(parsed.Cluster),
			Services: []string{parsed.Service},
		})
		if err != nil {
			color.Red("Failed to describe service %q: %v", parsed.Service, err)
			os.Exit(1)
		}
		if len(serviceOut.Services) == 0 {
			color.Red("Service %q not found in cluster %q", parsed.Service, parsed.Cluster)
			os.Exit(1)
		}
		baseTaskArn = aws.ToString(serviceOut.Services[0].TaskDefinition)
	}

	color.Blue("Fetching baseline task definition: %s", baseTaskArn)
	tdOut, err := ecsClient.DescribeTaskDefinition(context.TODO(), &ecs.DescribeTaskDefinitionInput{
		TaskDefinition: aws.String(baseTaskArn),
		Include:        []types.TaskDefinitionField{types.TaskDefinitionFieldTags},
	})
	if err != nil {
		color.Red("Failed to describe task definition: %v", err)
		os.Exit(1)
	}

	baseTd := tdOut.TaskDefinition

	// 3. Mutate Task Definition
	var diffs []string
	newTdInput, err := mutateTaskDefinition(baseTd, parsed, &diffs)
	if err != nil {
		color.Red("Failed to modify task definition: %v", err)
		os.Exit(1)
	}

	// Apply tags from the previous task definition if present
	if len(tdOut.Tags) > 0 {
		newTdInput.Tags = tdOut.Tags
	}

	// Print Diff
	if parsed.Diff && len(diffs) > 0 {
		color.Blue("Updating task definition changes:")
		for _, d := range diffs {
			color.Cyan("  %s", d)
		}
		fmt.Println()
	}

	// 4. Register new task definition revision
	color.Blue("Registering new task definition revision...")
	regOut, err := ecsClient.RegisterTaskDefinition(context.TODO(), newTdInput)
	if err != nil {
		color.Red("Failed to register new task definition: %v", err)
		os.Exit(1)
	}

	newTdArn := aws.ToString(regOut.TaskDefinition.TaskDefinitionArn)
	color.Green("Successfully created revision: %s (revision %d)\n", newTdArn, regOut.TaskDefinition.Revision)

	// 5. Update ECS Service
	color.Blue("Updating service %q to use task definition %s", parsed.Service, newTdArn)
	updateInput := &ecs.UpdateServiceInput{
		Cluster:        aws.String(parsed.Cluster),
		Service:        aws.String(parsed.Service),
		TaskDefinition: aws.String(newTdArn),
	}
	if parsed.ForceNewDeployment {
		updateInput.ForceNewDeployment = true
	}

	_, err = ecsClient.UpdateService(context.TODO(), updateInput)
	if err != nil {
		color.Red("Failed to update service: %v", err)
		os.Exit(1)
	}
	color.Green("Successfully changed task definition to: %s\n", newTdArn)

	// 6. Polling deployment status
	if parsed.Timeout == -1 {
		color.Green("Timeout is -1. Skipping deployment status check.")
		os.Exit(0)
	}

	color.Blue("Waiting for deployment to complete")
	startTime := time.Now()
	waitingTimeout := startTime.Add(time.Duration(parsed.Timeout) * time.Second)
	sleepDuration := time.Duration(parsed.SleepTime) * time.Second

	var lastInspectTime time.Time
	success := false

	for time.Now().Before(waitingTimeout) {
		fmt.Print(".")
		time.Sleep(sleepDuration)

		// Describe service to get current rollout/events
		svcOut, err := ecsClient.DescribeServices(context.TODO(), &ecs.DescribeServicesInput{
			Cluster:  aws.String(parsed.Cluster),
			Services: []string{parsed.Service},
		})
		if err != nil {
			log.Debugf("Error describing service: %v", err)
			continue
		}
		if len(svcOut.Services) == 0 {
			continue
		}

		svc := &svcOut.Services[0]

		// Inspect service events for errors/warnings
		inspectedTime, err := inspectErrors(svc, lastInspectTime, parsed.IgnoreWarnings)
		if err != nil {
			fmt.Println()
			color.Red("%v", err)
			handleFailure(ecsClient, parsed, baseTaskArn, newTdArn)
			os.Exit(1)
		}
		lastInspectTime = inspectedTime

		// Check if deployment is complete
		deployed, err := isDeployed(ecsClient, svc, newTdArn)
		if err != nil {
			fmt.Println()
			color.Red("%v", err)
			handleFailure(ecsClient, parsed, baseTaskArn, newTdArn)
			os.Exit(1)
		}

		if deployed {
			success = true
			break
		}
	}

	fmt.Println()

	if !success {
		color.Red("Deployment failed due to timeout.")
		handleFailure(ecsClient, parsed, baseTaskArn, newTdArn)
		os.Exit(1)
	}

	duration := int(time.Since(startTime).Seconds())
	color.Green("Deployment successful")
	color.Green("Duration: %d sec\n", duration)

	// 7. Deregister older task definition if requested
	if parsed.Deregister {
		color.Blue("Deregister task definition revision: %s", baseTaskArn)
		_, err = ecsClient.DeregisterTaskDefinition(context.TODO(), &ecs.DeregisterTaskDefinitionInput{
			TaskDefinition: aws.String(baseTaskArn),
		})
		if err != nil {
			color.Yellow("Warning: Failed to deregister old task definition: %v", err)
		} else {
			color.Green("Successfully deregistered revision: %s\n", baseTaskArn)
		}
	}
}

// handleFailure executes rollback and error handling
func handleFailure(client *ecs.Client, parsed *ParsedArgs, oldTdArn, newTdArn string) {
	if parsed.Rollback {
		color.Yellow("Rolling back to task definition: %s", oldTdArn)
		_, err := client.UpdateService(context.TODO(), &ecs.UpdateServiceInput{
			Cluster:        aws.String(parsed.Cluster),
			Service:        aws.String(parsed.Service),
			TaskDefinition: aws.String(oldTdArn),
		})
		if err != nil {
			color.Red("Failed to initiate rollback: %v", err)
			return
		}
		color.Yellow("Rollback service update triggered. Waiting for completion...")

		// Polling rollback status
		startTime := time.Now()
		waitingTimeout := startTime.Add(600 * time.Second)
		var lastInspectTime time.Time
		success := false

		for time.Now().Before(waitingTimeout) {
			fmt.Print(".")
			time.Sleep(time.Duration(parsed.SleepTime) * time.Second)

			svcOut, err := client.DescribeServices(context.TODO(), &ecs.DescribeServicesInput{
				Cluster:  aws.String(parsed.Cluster),
				Services: []string{parsed.Service},
			})
			if err != nil || len(svcOut.Services) == 0 {
				continue
			}
			svc := &svcOut.Services[0]

			inspectedTime, err := inspectErrors(svc, lastInspectTime, false)
			if err != nil {
				fmt.Println()
				color.Red("Rollback warning/error encountered: %v", err)
			}
			lastInspectTime = inspectedTime

			deployed, err := isDeployed(client, svc, oldTdArn)
			if err == nil && deployed {
				success = true
				break
			}
		}
		fmt.Println()
		if success {
			color.Yellow("Deployment failed, but service has been rolled back to previous task definition: %s", oldTdArn)
		} else {
			color.Red("Rollback failed or timed out. Please check AWS ECS Console.")
		}
	}
}

// isDeployed verifies if service runs only the target task definition and desiredCount is met
func isDeployed(client *ecs.Client, service *types.Service, targetTdArn string) (bool, error) {
	var primaryDeployment *types.Deployment
	for i := range service.Deployments {
		dep := &service.Deployments[i]
		if aws.ToString(dep.Status) == "PRIMARY" {
			primaryDeployment = dep
		}
	}

	if primaryDeployment != nil {
		if depState := primaryDeployment.RolloutState; depState == types.DeploymentRolloutStateFailed {
			return false, fmt.Errorf("Deployment Failed: %s", aws.ToString(primaryDeployment.RolloutStateReason))
		}
	}

	if len(service.Deployments) != 1 {
		return false, nil
	}

	// Verify primary deployment task definition matches
	if aws.ToString(primaryDeployment.TaskDefinition) != targetTdArn {
		return false, nil
	}

	// List and describe tasks
	var taskArns []string
	paginator := ecs.NewListTasksPaginator(client, &ecs.ListTasksInput{
		Cluster:     service.ClusterArn,
		ServiceName: service.ServiceName,
	})
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(context.TODO())
		if err != nil {
			return false, err
		}
		taskArns = append(taskArns, page.TaskArns...)
	}

	if len(taskArns) == 0 {
		return service.DesiredCount == 0, nil
	}

	descOut, err := client.DescribeTasks(context.TODO(), &ecs.DescribeTasksInput{
		Cluster: service.ClusterArn,
		Tasks:   taskArns,
	})
	if err != nil {
		return false, err
	}

	runningCount := 0
	for _, task := range descOut.Tasks {
		if aws.ToString(task.TaskDefinitionArn) == targetTdArn && aws.ToString(task.LastStatus) == "RUNNING" {
			runningCount++
		}
	}

	return runningCount == int(service.DesiredCount), nil
}

// inspectErrors checks service events for errors/warnings containing "unable"
func inspectErrors(service *types.Service, since time.Time, ignoreWarnings bool) (time.Time, error) {
	lastTime := since
	for _, event := range service.Events {
		if event.CreatedAt != nil && event.CreatedAt.After(since) {
			msg := aws.ToString(event.Message)
			if strings.Contains(strings.ToLower(msg), "unable") {
				if !ignoreWarnings {
					return *event.CreatedAt, fmt.Errorf("ERROR: %s", msg)
				}
				color.Yellow("\n%s\nWARNING: %s\nContinuing.", event.CreatedAt.Format(time.RFC3339), msg)
			}
			if event.CreatedAt.After(lastTime) {
				lastTime = *event.CreatedAt
			}
		}
	}
	return lastTime, nil
}

// createEcsClient initializes ecs.ECS client with standard session & STS option
func createEcsClient(parsed *ParsedArgs) (*ecs.Client, error) {
	var opts []func(*config.LoadOptions) error

	if parsed.Region != "" {
		opts = append(opts, config.WithRegion(parsed.Region))
	}

	if parsed.AccessKeyId != "" && parsed.SecretAccessKey != "" {
		opts = append(opts, config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(parsed.AccessKeyId, parsed.SecretAccessKey, "")))
	}

	if parsed.Profile != "" {
		opts = append(opts, config.WithSharedConfigProfile(parsed.Profile))
	}

	cfg, err := config.LoadDefaultConfig(context.TODO(), opts...)
	if err != nil {
		return nil, err
	}

	if parsed.AssumeRole != "" {
		roleArn := parsed.AssumeRole
		if !strings.HasPrefix(roleArn, "arn:aws:iam::") && parsed.Account != "" {
			roleArn = fmt.Sprintf("arn:aws:iam::%s:role/%s", parsed.Account, parsed.AssumeRole)
		}
		stsClient := sts.NewFromConfig(cfg)
		provider := stscreds.NewAssumeRoleProvider(stsClient, roleArn, func(o *stscreds.AssumeRoleOptions) {
			o.RoleSessionName = "golemDeploy"
		})
		cfg.Credentials = provider
	}

	return ecs.NewFromConfig(cfg), nil
}
