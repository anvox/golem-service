package main

import (
	"strings"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ecs/types"
)

func createBaseTaskDefinition(containerNames ...string) *types.TaskDefinition {
	var containers []types.ContainerDefinition
	for _, name := range containerNames {
		containers = append(containers, types.ContainerDefinition{
			Name:  aws.String(name),
			Image: aws.String("my-repo/" + name + ":latest"),
		})
	}
	return &types.TaskDefinition{
		Family:               aws.String("test-family"),
		ContainerDefinitions: containers,
	}
}

func TestMutateTaskDefinition_UnknownContainer_Env(t *testing.T) {
	baseTd := createBaseTaskDefinition("app")
	parsed := &ParsedArgs{
		EnvOverrides: []EnvVarOverride{
			{Container: "containerApp", Name: "COMMIT_ID", Value: "xxxxx"},
		},
	}
	var diffs []string
	_, err := mutateTaskDefinition(baseTd, parsed, &diffs)
	if err == nil {
		t.Fatalf("expected error for unknown container 'containerApp', got nil")
	}

	expectedErr := `❌ container "containerApp" not found in task definition`
	if !strings.Contains(err.Error(), expectedErr) {
		t.Errorf("expected error message to contain %q, got %q", expectedErr, err.Error())
	}
}

func TestMutateTaskDefinition_UnknownContainer_AllOverrides(t *testing.T) {
	tests := []struct {
		name   string
		modify func(*ParsedArgs)
	}{
		{
			name: "ImageOverride",
			modify: func(p *ParsedArgs) {
				p.Images = []ImageOverride{{Container: "ghost", Image: "img:1"}}
			},
		},
		{
			name: "CommandOverride",
			modify: func(p *ParsedArgs) {
				p.Commands = []CommandOverride{{Container: "ghost", Command: "echo 1"}}
			},
		},
		{
			name: "CpuOverride",
			modify: func(p *ParsedArgs) {
				p.CpuOverrides = []CpuOverride{{Container: "ghost", Cpu: 256}}
			},
		},
		{
			name: "MemoryOverride",
			modify: func(p *ParsedArgs) {
				p.MemoryOverrides = []MemoryOverride{{Container: "ghost", Memory: 512}}
			},
		},
		{
			name: "MemReservationOverride",
			modify: func(p *ParsedArgs) {
				p.MemReservationOverrides = []MemoryReservationOverride{{Container: "ghost", MemoryReservation: 256}}
			},
		},
		{
			name: "PrivilegedOverride",
			modify: func(p *ParsedArgs) {
				p.PrivilegedOverrides = []PrivilegedOverride{{Container: "ghost", Privileged: true}}
			},
		},
		{
			name: "EssentialOverride",
			modify: func(p *ParsedArgs) {
				p.EssentialOverrides = []EssentialOverride{{Container: "ghost", Essential: false}}
			},
		},
		{
			name: "HealthCheckOverride",
			modify: func(p *ParsedArgs) {
				p.HealthCheckOverrides = []HealthCheckOverride{{Container: "ghost", Command: "curl localhost"}}
			},
		},
		{
			name: "EnvVarOverride",
			modify: func(p *ParsedArgs) {
				p.EnvOverrides = []EnvVarOverride{{Container: "ghost", Name: "K", Value: "V"}}
			},
		},
		{
			name: "EnvFileArg",
			modify: func(p *ParsedArgs) {
				p.EnvFiles = []EnvFileArg{{Container: "ghost", FilePath: "/tmp/fake.env"}}
			},
		},
		{
			name: "S3EnvFileArg",
			modify: func(p *ParsedArgs) {
				p.S3EnvFiles = []S3EnvFileArg{{Container: "ghost", S3Arn: "arn:aws:s3:::bucket/env"}}
			},
		},
		{
			name: "SecretOverride",
			modify: func(p *ParsedArgs) {
				p.SecretOverrides = []SecretOverride{{Container: "ghost", Name: "SEC", ValueFrom: "param"}}
			},
		},
		{
			name: "SecretsEnvFiles",
			modify: func(p *ParsedArgs) {
				p.SecretsEnvFiles = []EnvFileArg{{Container: "ghost", FilePath: "/tmp/fake.env"}}
			},
		},
		{
			name: "DockerLabelArg",
			modify: func(p *ParsedArgs) {
				p.DockerLabels = []DockerLabelArg{{Container: "ghost", Name: "label", Value: "val"}}
			},
		},
		{
			name: "UlimitOverride",
			modify: func(p *ParsedArgs) {
				p.Ulimits = []UlimitOverride{{Container: "ghost", Name: "nofile", SoftLimit: 1024, HardLimit: 2048}}
			},
		},
		{
			name: "SysctlOverride",
			modify: func(p *ParsedArgs) {
				p.Sysctls = []SysctlOverride{{Container: "ghost", Namespace: "net.core.somaxconn", Value: "1024"}}
			},
		},
		{
			name: "PortOverride",
			modify: func(p *ParsedArgs) {
				p.Ports = []PortOverride{{Container: "ghost", ContainerPort: 80, HostPort: 80}}
			},
		},
		{
			name: "MountOverride",
			modify: func(p *ParsedArgs) {
				p.Mounts = []MountOverride{{Container: "ghost", SourceVolume: "data", Path: "/data"}}
			},
		},
		{
			name: "LogOverride",
			modify: func(p *ParsedArgs) {
				p.Logs = []LogOverride{{Container: "ghost", LogDriver: "awslogs", Name: "awslogs-group", Value: "/ecs/app"}}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			baseTd := createBaseTaskDefinition("app")
			parsed := &ParsedArgs{}
			tt.modify(parsed)

			var diffs []string
			_, err := mutateTaskDefinition(baseTd, parsed, &diffs)
			if err == nil {
				t.Fatalf("expected error for unknown container 'ghost', got nil")
			}

			expectedErr := `❌ container "ghost" not found in task definition`
			if !strings.Contains(err.Error(), expectedErr) {
				t.Errorf("expected error message to contain %q, got %q", expectedErr, err.Error())
			}
		})
	}
}

func TestMutateTaskDefinition_RemoveContainer_NotFound(t *testing.T) {
	baseTd := createBaseTaskDefinition("app")
	parsed := &ParsedArgs{
		RemoveContainers: []string{"nonexistent"},
	}
	var diffs []string
	_, err := mutateTaskDefinition(baseTd, parsed, &diffs)
	if err == nil {
		t.Fatalf("expected error for removing nonexistent container, got nil")
	}

	expectedErr := `❌ container "nonexistent" not found in task definition`
	if !strings.Contains(err.Error(), expectedErr) {
		t.Errorf("expected error message to contain %q, got %q", expectedErr, err.Error())
	}
}

func TestMutateTaskDefinition_AddContainer_Success(t *testing.T) {
	baseTd := createBaseTaskDefinition("app")
	parsed := &ParsedArgs{
		AddContainers: []string{"worker"},
		EnvOverrides: []EnvVarOverride{
			{Container: "worker", Name: "WORKER_VAR", Value: "123"},
		},
	}
	var diffs []string
	result, err := mutateTaskDefinition(baseTd, parsed, &diffs)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(result.ContainerDefinitions) != 2 {
		t.Fatalf("expected 2 container definitions, got %d", len(result.ContainerDefinitions))
	}

	var workerContainer *types.ContainerDefinition
	for i := range result.ContainerDefinitions {
		if aws.ToString(result.ContainerDefinitions[i].Name) == "worker" {
			workerContainer = &result.ContainerDefinitions[i]
		}
	}
	if workerContainer == nil {
		t.Fatalf("worker container was not added")
	}

	foundEnv := false
	for _, kv := range workerContainer.Environment {
		if aws.ToString(kv.Name) == "WORKER_VAR" && aws.ToString(kv.Value) == "123" {
			foundEnv = true
			break
		}
	}
	if !foundEnv {
		t.Errorf("expected WORKER_VAR=123 in worker container environment")
	}
}

func TestMutateTaskDefinition_ValidContainer_Success(t *testing.T) {
	baseTd := createBaseTaskDefinition("app")
	parsed := &ParsedArgs{
		EnvOverrides: []EnvVarOverride{
			{Container: "app", Name: "COMMIT_ID", Value: "abcde"},
		},
	}
	var diffs []string
	result, err := mutateTaskDefinition(baseTd, parsed, &diffs)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(result.ContainerDefinitions) != 1 {
		t.Fatalf("expected 1 container definition, got %d", len(result.ContainerDefinitions))
	}

	foundEnv := false
	for _, kv := range result.ContainerDefinitions[0].Environment {
		if aws.ToString(kv.Name) == "COMMIT_ID" && aws.ToString(kv.Value) == "abcde" {
			foundEnv = true
			break
		}
	}
	if !foundEnv {
		t.Errorf("expected COMMIT_ID=abcde in app container environment")
	}
	if len(diffs) == 0 {
		t.Errorf("expected diff to be recorded for COMMIT_ID change")
	}
}

func TestMutateTaskDefinition_RemoveContainer_ThenOverrideFails(t *testing.T) {
	baseTd := createBaseTaskDefinition("app", "worker")
	parsed := &ParsedArgs{
		RemoveContainers: []string{"worker"},
		EnvOverrides: []EnvVarOverride{
			{Container: "worker", Name: "K", Value: "V"},
		},
	}
	var diffs []string
	_, err := mutateTaskDefinition(baseTd, parsed, &diffs)
	if err == nil {
		t.Fatalf("expected error overriding removed container, got nil")
	}
	expectedErr := `❌ container "worker" not found in task definition`
	if !strings.Contains(err.Error(), expectedErr) {
		t.Errorf("expected error message to contain %q, got %q", expectedErr, err.Error())
	}
}
