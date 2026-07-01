package main

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/credentials/stscreds"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ecs"
	"github.com/aws/aws-sdk-go/service/sts"
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
		serviceOut, err := ecsClient.DescribeServices(&ecs.DescribeServicesInput{
			Cluster:  aws.String(parsed.Cluster),
			Services: []*string{aws.String(parsed.Service)},
		})
		if err != nil {
			color.Red("Failed to describe service %q: %v", parsed.Service, err)
			os.Exit(1)
		}
		if len(serviceOut.Services) == 0 {
			color.Red("Service %q not found in cluster %q", parsed.Service, parsed.Cluster)
			os.Exit(1)
		}
		baseTaskArn = aws.StringValue(serviceOut.Services[0].TaskDefinition)
	}

	color.Blue("Fetching baseline task definition: %s", baseTaskArn)
	tdOut, err := ecsClient.DescribeTaskDefinition(&ecs.DescribeTaskDefinitionInput{
		TaskDefinition: aws.String(baseTaskArn),
		Include:        []*string{aws.String("TAGS")},
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
	regOut, err := ecsClient.RegisterTaskDefinition(newTdInput)
	if err != nil {
		color.Red("Failed to register new task definition: %v", err)
		os.Exit(1)
	}

	newTdArn := aws.StringValue(regOut.TaskDefinition.TaskDefinitionArn)
	color.Green("Successfully created revision: %s (revision %d)\n", newTdArn, aws.Int64Value(regOut.TaskDefinition.Revision))

	// 5. Update ECS Service
	color.Blue("Updating service %q to use task definition %s", parsed.Service, newTdArn)
	updateInput := &ecs.UpdateServiceInput{
		Cluster:        aws.String(parsed.Cluster),
		Service:        aws.String(parsed.Service),
		TaskDefinition: aws.String(newTdArn),
	}
	if parsed.ForceNewDeployment {
		updateInput.ForceNewDeployment = aws.Bool(true)
	}

	_, err = ecsClient.UpdateService(updateInput)
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
		svcOut, err := ecsClient.DescribeServices(&ecs.DescribeServicesInput{
			Cluster:  aws.String(parsed.Cluster),
			Services: []*string{aws.String(parsed.Service)},
		})
		if err != nil {
			log.Debugf("Error describing service: %v", err)
			continue
		}
		if len(svcOut.Services) == 0 {
			continue
		}

		svc := svcOut.Services[0]

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
		_, err = ecsClient.DeregisterTaskDefinition(&ecs.DeregisterTaskDefinitionInput{
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
func handleFailure(client *ecs.ECS, parsed *ParsedArgs, oldTdArn, newTdArn string) {
	if parsed.Rollback {
		color.Yellow("Rolling back to task definition: %s", oldTdArn)
		_, err := client.UpdateService(&ecs.UpdateServiceInput{
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

			svcOut, err := client.DescribeServices(&ecs.DescribeServicesInput{
				Cluster:  aws.String(parsed.Cluster),
				Services: []*string{aws.String(parsed.Service)},
			})
			if err != nil || len(svcOut.Services) == 0 {
				continue
			}
			svc := svcOut.Services[0]

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
func isDeployed(client *ecs.ECS, service *ecs.Service, targetTdArn string) (bool, error) {
	var primaryDeployment *ecs.Deployment
	for _, dep := range service.Deployments {
		if aws.StringValue(dep.Status) == "PRIMARY" {
			primaryDeployment = dep
		}
	}

	if primaryDeployment != nil {
		if aws.StringValue(primaryDeployment.RolloutState) == "FAILED" {
			return false, fmt.Errorf("Deployment Failed: %s", aws.StringValue(primaryDeployment.RolloutStateReason))
		}
	}

	if len(service.Deployments) != 1 {
		return false, nil
	}

	// Verify primary deployment task definition matches
	if aws.StringValue(primaryDeployment.TaskDefinition) != targetTdArn {
		return false, nil
	}

	// List and describe tasks
	var taskArns []*string
	err := client.ListTasksPages(&ecs.ListTasksInput{
		Cluster:     service.ClusterArn,
		ServiceName: service.ServiceName,
	}, func(page *ecs.ListTasksOutput, lastPage bool) bool {
		taskArns = append(taskArns, page.TaskArns...)
		return !lastPage
	})
	if err != nil {
		return false, err
	}

	if len(taskArns) == 0 {
		return aws.Int64Value(service.DesiredCount) == 0, nil
	}

	descOut, err := client.DescribeTasks(&ecs.DescribeTasksInput{
		Cluster: service.ClusterArn,
		Tasks:   taskArns,
	})
	if err != nil {
		return false, err
	}

	runningCount := 0
	for _, task := range descOut.Tasks {
		if aws.StringValue(task.TaskDefinitionArn) == targetTdArn && aws.StringValue(task.LastStatus) == "RUNNING" {
			runningCount++
		}
	}

	return runningCount == int(aws.Int64Value(service.DesiredCount)), nil
}

// inspectErrors checks service events for errors/warnings containing "unable"
func inspectErrors(service *ecs.Service, since time.Time, ignoreWarnings bool) (time.Time, error) {
	lastTime := since
	for _, event := range service.Events {
		if event.CreatedAt != nil && event.CreatedAt.After(since) {
			msg := aws.StringValue(event.Message)
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
func createEcsClient(parsed *ParsedArgs) (*ecs.ECS, error) {
	awsConfig := aws.NewConfig()
	if parsed.Region != "" {
		awsConfig.WithRegion(parsed.Region)
	}

	if parsed.AccessKeyId != "" && parsed.SecretAccessKey != "" {
		awsConfig.WithCredentials(credentials.NewStaticCredentials(parsed.AccessKeyId, parsed.SecretAccessKey, ""))
	}

	opts := session.Options{
		Config:            *awsConfig,
		SharedConfigState: session.SharedConfigEnable,
	}
	if parsed.Profile != "" {
		opts.Profile = parsed.Profile
	}

	sess, err := session.NewSessionWithOptions(opts)
	if err != nil {
		return nil, err
	}

	if parsed.AssumeRole != "" {
		roleArn := parsed.AssumeRole
		if !strings.HasPrefix(roleArn, "arn:aws:iam::") && parsed.Account != "" {
			roleArn = fmt.Sprintf("arn:aws:iam::%s:role/%s", parsed.Account, parsed.AssumeRole)
		}
		stsClient := sts.New(sess)
		creds := stscreds.NewCredentialsWithClient(stsClient, roleArn, func(p *stscreds.AssumeRoleProvider) {
			p.RoleSessionName = "golemDeploy"
		})
		sess.Config.Credentials = creds
	}

	return ecs.New(sess), nil
}
