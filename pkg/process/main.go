package service_process

import (
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ecs"
	"github.com/dustin/go-humanize"
	log "github.com/sirupsen/logrus"

	"fmt"
	"strings"
	"sync"
)

type Identifier struct {
	Environment string
	Stack       string
	Service     string
}

type PollingToken struct {
	ProcessIds []string
}

type ProcessInfo struct {
	Id        string
	Status    string
	Service   string
	ProcType  string
	StartedAt string
}

func (identifier Identifier) Validate() error {
	if EcsClusterName(identifier) == "" {
		return fmt.Errorf("Unknown ECS cluster")
	}

	return nil
}

func List(identifier Identifier, processType []string) ([]ProcessInfo, error) {
	ecsClient := ecs.New(session.New())
	taskArns := []*string{}
	err := ecsClient.ListTasksPages(&ecs.ListTasksInput{
		Cluster: aws.String(EcsClusterName(identifier)),
	}, func(page *ecs.ListTasksOutput, lastPage bool) bool {
		taskArns = append(taskArns, page.TaskArns...)
		return !lastPage
	})
	if err != nil {
		return []ProcessInfo{}, err
	}

	processes := []ProcessInfo{}
	for i := 0; i < len(taskArns); i += 100 {
		tail := i + 100
		if tail > len(taskArns) {
			tail = len(taskArns)
		}
		tasks := taskArns[i:tail]
		page, _ := ecsClient.DescribeTasks(&ecs.DescribeTasksInput{
			Cluster: aws.String(EcsClusterName(identifier)),
			Tasks:   tasks,
			Include: []*string{
				aws.String("TAGS"),
			},
		})
		for _, task := range page.Tasks {
			processInfo, err := parseTask(*task)
			if err == nil {
				if identifier.Service != "" && identifier.Service != processInfo.Service {
					continue
				}
				if len(processType) > 0 {
					filteredOut := true
					for _, filteredType := range processType {
						if filteredType == processInfo.ProcType {
							filteredOut = false
						}
					}
					if filteredOut {
						continue
					}
				}
				processes = append(processes, processInfo)
			}
		}
	}
	return processes, nil
}

func pollingKill(clusterName string, idPipe chan string, ecsClient *ecs.ECS, wg *sync.WaitGroup) {
	defer wg.Done()

MainLoop:
	for {
		select {
		case id := <-idPipe:
			if id == "" {
				break MainLoop
			}
			log.Infof("Stopping [%s]", id)
			_, error := ecsClient.StopTask(&ecs.StopTaskInput{
				Cluster: aws.String(clusterName),
				Task:    aws.String(id),
				Reason:  aws.String("[golem] Manually stop"),
			})
			if error != nil {
				log.Errorf("Failed to stop task[%s]: %+v", id, error)
			}
		}
	}
}

func Kill(identifier Identifier, processType []string) error {
	processes, err := List(identifier, processType)
	if err != nil {
		return err
	}

	pipes := make(map[string]chan (string))
	ecsClient := ecs.New(session.New())
	clusterName := EcsClusterName(identifier)
	var wg sync.WaitGroup
	for _, process := range processes {
		pipe, ok := pipes[process.Service]
		if !ok {
			pipe = make(chan (string))
			pipes[process.Service] = pipe
			wg.Add(1)
			go pollingKill(clusterName, pipe, ecsClient, &wg)
		}
		pipe <- process.Id
	}
	for _, pipe := range pipes {
		// Let channel know to end
		pipe <- ""
	}
	wg.Wait()

	return nil
}

func KillOne(identifier Identifier, processId string) error {
	ecsClient := ecs.New(session.New())
	_, error := ecsClient.StopTask(&ecs.StopTaskInput{
		Cluster: aws.String(EcsClusterName(identifier)),
		Task:    aws.String(processId),
		Reason:  aws.String("[golem] Manually stop"),
	})
	if error != nil {
		return error
	}

	return nil
}

func EcsClusterName(id Identifier) string {
	return id.Stack + "-" + id.Environment
}

func buildMatcher(identifier Identifier, processType string) func(string, string) bool {
	return func(clusterName string, serviceName string) bool {
		return true
	}
}

func parseTask(ecsTask ecs.Task) (ProcessInfo, error) {
	arnParts := strings.Split(aws.StringValue(ecsTask.TaskArn), "/")

	service := extractServiceName(ecsTask.Tags)
	if service == "" {
		service = extractApplicationName(ecsTask.Tags)
	}
	status := aws.StringValue(ecsTask.LastStatus)
	startedAt := ""
	if status == "RUNNING" {
		startedAt = humanize.Time(aws.TimeValue(ecsTask.StartedAt))
	}
	return ProcessInfo{
		Id:        arnParts[len(arnParts)-1],
		Status:    status,
		Service:   service,
		ProcType:  extractProcType(ecsTask, service),
		StartedAt: startedAt,
	}, nil
}

func extractApplicationName(ecsTags []*ecs.Tag) string {
	for _, tag := range ecsTags {
		if aws.StringValue(tag.Key) == "Application" {
			return aws.StringValue(tag.Value)
		}
	}

	return "Unknown"
}

func extractServiceName(ecsTags []*ecs.Tag) string {
	for _, tag := range ecsTags {
		if aws.StringValue(tag.Key) == "Service" {
			return aws.StringValue(tag.Value)
		}
	}

	return ""
}

func extractProcType(ecsTask ecs.Task, service string) string {
	// Group of service task has pattern
	// service:<somethins>-<service>-<proc>
	// If it doesn't match this pattern, recognize as other
	group := aws.StringValue(ecsTask.Group)
	if !strings.Contains(group, service) {
		return "other"
	}

	parts := strings.Split(group, service)
	procType := strings.Trim(parts[len(parts)-1], "-")
	if procType == "" {
		procType = "main"
	}

	return procType
}
