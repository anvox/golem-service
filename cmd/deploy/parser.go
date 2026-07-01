package main

import (
	"fmt"
	"strconv"
	"strings"
)

type ParsedArgs struct {
	Cluster                 string
	Service                 string
	Tag                     string
	Images                  []ImageOverride
	Commands                []CommandOverride
	CpuOverrides            []CpuOverride
	MemoryOverrides         []MemoryOverride
	MemReservationOverrides []MemoryReservationOverride
	PrivilegedOverrides     []PrivilegedOverride
	EssentialOverrides      []EssentialOverride
	EnvOverrides            []EnvVarOverride
	EnvFiles                []EnvFileArg
	S3EnvFiles              []S3EnvFileArg
	SecretOverrides         []SecretOverride
	SecretsEnvFiles         []EnvFileArg
	DockerLabels            []DockerLabelArg
	Ulimits                 []UlimitOverride
	Sysctls                 []SysctlOverride
	Ports                   []PortOverride
	Mounts                  []MountOverride
	Logs                    []LogOverride
	TaskCpu                 string
	TaskMemory              string
	Role                    string
	ExecutionRole           string
	RuntimePlatform         *RuntimePlatformArg
	Task                    string
	Region                  string
	AccessKeyId             string
	SecretAccessKey         string
	Profile                 string
	Account                 string
	AssumeRole              string
	Timeout                 int
	ForceNewDeployment      bool
	IgnoreWarnings          bool
	SleepTime               int
	Diff                    bool
	Deregister              bool
	Rollback                bool
	ExclusiveEnv            bool
	ExclusiveSecrets        bool
	ExclusiveDockerLabels   bool
	ExclusiveS3EnvFile      bool
	ExclusiveUlimits        bool
	ExclusiveSysctls        bool
	ExclusivePorts          bool
	ExclusiveMounts         bool
	Volumes                 []VolumeOverride
	AddContainers           []string
	RemoveContainers        []string
	Help                    bool
	Version                 bool
	HealthCheckOverrides    []HealthCheckOverride
}

type ImageOverride struct {
	Container string
	Image     string
}

type CommandOverride struct {
	Container string
	Command   string
}

type CpuOverride struct {
	Container string
	Cpu       int64
}

type MemoryOverride struct {
	Container string
	Memory    int64
}

type MemoryReservationOverride struct {
	Container         string
	MemoryReservation int64
}

type PrivilegedOverride struct {
	Container  string
	Privileged bool
}

type EssentialOverride struct {
	Container string
	Essential bool
}

type EnvVarOverride struct {
	Container string
	Name      string
	Value     string
}

type SecretOverride struct {
	Container string
	Name      string
	ValueFrom string
}

type PortOverride struct {
	Container     string
	ContainerPort int64
	HostPort      int64
}

type MountOverride struct {
	Container    string
	SourceVolume string
	Path         string
}

type LogOverride struct {
	Container string
	LogDriver string
	Name      string
	Value     string
}

type UlimitOverride struct {
	Container string
	Name      string
	SoftLimit int64
	HardLimit int64
}

type SysctlOverride struct {
	Container string
	Namespace string
	Value     string
}

type VolumeOverride struct {
	Name       string
	SourcePath string
}

type HealthCheckOverride struct {
	Container   string
	Command     string
	Interval    int64
	Timeout     int64
	Retries     int64
	StartPeriod int64
}

type EnvFileArg struct {
	Container string
	FilePath  string
}

type S3EnvFileArg struct {
	Container string
	S3Arn     string
}

type DockerLabelArg struct {
	Container string
	Name      string
	Value     string
}

type RuntimePlatformArg struct {
	CpuArch  string
	OsFamily string
}

func parseCLI(args []string) (*ParsedArgs, error) {
	parsed := &ParsedArgs{
		Timeout:    300,
		SleepTime:  1,
		Diff:       true,
		Deregister: true,
	}

	var positionals []string

	for i := 0; i < len(args); i++ {
		arg := args[i]

		// Handle --flag=value format
		if strings.HasPrefix(arg, "--") && strings.Contains(arg, "=") {
			parts := strings.SplitN(arg, "=", 2)
			flag := parts[0]
			val := parts[1]
			switch flag {
			case "--tag":
				parsed.Tag = val
			case "--task-cpu":
				parsed.TaskCpu = val
			case "--task-memory":
				parsed.TaskMemory = val
			case "--role":
				parsed.Role = val
			case "--execution-role":
				parsed.ExecutionRole = val
			case "--task":
				parsed.Task = val
			case "--region":
				parsed.Region = val
			case "--access-key-id":
				parsed.AccessKeyId = val
			case "--secret-access-key":
				parsed.SecretAccessKey = val
			case "--profile":
				parsed.Profile = val
			case "--account":
				parsed.Account = val
			case "--assume-role":
				parsed.AssumeRole = val
			case "--timeout":
				t, _ := strconv.Atoi(val)
				parsed.Timeout = t
			case "--sleep-time":
				s, _ := strconv.Atoi(val)
				parsed.SleepTime = s
			default:
				return nil, fmt.Errorf("unknown or unsupported flag format: %s", arg)
			}
			continue
		}

		if strings.HasPrefix(arg, "-") {
			// Check matching flags
			switch arg {
			case "-h", "--help":
				parsed.Help = true
			case "-v", "--version":
				parsed.Version = true
			case "-t", "--tag":
				if i+1 >= len(args) {
					return nil, fmt.Errorf("missing value for option %s", arg)
				}
				parsed.Tag = args[i+1]
				i++
			case "-i", "--image":
				if i+2 >= len(args) {
					return nil, fmt.Errorf("option %s requires 2 arguments: <container> <image>", arg)
				}
				parsed.Images = append(parsed.Images, ImageOverride{Container: args[i+1], Image: args[i+2]})
				i += 2
			case "-c", "--command":
				if i+2 >= len(args) {
					return nil, fmt.Errorf("option %s requires 2 arguments: <container> <command>", arg)
				}
				parsed.Commands = append(parsed.Commands, CommandOverride{Container: args[i+1], Command: args[i+2]})
				i += 2
			case "--health-check":
				if i+6 >= len(args) {
					return nil, fmt.Errorf("option %s requires 6 arguments: <container> <command> <interval> <timeout> <retries> <start_period>", arg)
				}
				interval, _ := strconv.ParseInt(args[i+3], 10, 64)
				timeout, _ := strconv.ParseInt(args[i+4], 10, 64)
				retries, _ := strconv.ParseInt(args[i+5], 10, 64)
				startPeriod, _ := strconv.ParseInt(args[i+6], 10, 64)
				parsed.HealthCheckOverrides = append(parsed.HealthCheckOverrides, HealthCheckOverride{
					Container:   args[i+1],
					Command:     args[i+2],
					Interval:    interval,
					Timeout:     timeout,
					Retries:     retries,
					StartPeriod: startPeriod,
				})
				i += 6
			case "--cpu":
				if i+2 >= len(args) {
					return nil, fmt.Errorf("option %s requires 2 arguments: <container> <cpu>", arg)
				}
				cpu, _ := strconv.ParseInt(args[i+2], 10, 64)
				parsed.CpuOverrides = append(parsed.CpuOverrides, CpuOverride{Container: args[i+1], Cpu: cpu})
				i += 2
			case "--memory":
				if i+2 >= len(args) {
					return nil, fmt.Errorf("option %s requires 2 arguments: <container> <memory>", arg)
				}
				mem, _ := strconv.ParseInt(args[i+2], 10, 64)
				parsed.MemoryOverrides = append(parsed.MemoryOverrides, MemoryOverride{Container: args[i+1], Memory: mem})
				i += 2
			case "--memoryreservation":
				if i+2 >= len(args) {
					return nil, fmt.Errorf("option %s requires 2 arguments: <container> <memoryreservation>", arg)
				}
				mem, _ := strconv.ParseInt(args[i+2], 10, 64)
				parsed.MemReservationOverrides = append(parsed.MemReservationOverrides, MemoryReservationOverride{Container: args[i+1], MemoryReservation: mem})
				i += 2
			case "--task-cpu":
				if i+1 >= len(args) {
					return nil, fmt.Errorf("missing value for option %s", arg)
				}
				parsed.TaskCpu = args[i+1]
				i++
			case "--task-memory":
				if i+1 >= len(args) {
					return nil, fmt.Errorf("missing value for option %s", arg)
				}
				parsed.TaskMemory = args[i+1]
				i++
			case "--privileged":
				if i+2 >= len(args) {
					return nil, fmt.Errorf("option %s requires 2 arguments: <container> <true|false>", arg)
				}
				priv := args[i+2] == "true"
				parsed.PrivilegedOverrides = append(parsed.PrivilegedOverrides, PrivilegedOverride{Container: args[i+1], Privileged: priv})
				i += 2
			case "--essential":
				if i+2 >= len(args) {
					return nil, fmt.Errorf("option %s requires 2 arguments: <container> <true|false>", arg)
				}
				ess := args[i+2] == "true"
				parsed.EssentialOverrides = append(parsed.EssentialOverrides, EssentialOverride{Container: args[i+1], Essential: ess})
				i += 2
			case "-e", "--env":
				if i+3 >= len(args) {
					return nil, fmt.Errorf("option %s requires 3 arguments: <container> <name> <value>", arg)
				}
				parsed.EnvOverrides = append(parsed.EnvOverrides, EnvVarOverride{Container: args[i+1], Name: args[i+2], Value: args[i+3]})
				i += 3
			case "--env-file":
				if i+2 >= len(args) {
					return nil, fmt.Errorf("option %s requires 2 arguments: <container> <file_path>", arg)
				}
				parsed.EnvFiles = append(parsed.EnvFiles, EnvFileArg{Container: args[i+1], FilePath: args[i+2]})
				i += 2
			case "--s3-env-file":
				if i+2 >= len(args) {
					return nil, fmt.Errorf("option %s requires 2 arguments: <container> <S3_ARN>", arg)
				}
				parsed.S3EnvFiles = append(parsed.S3EnvFiles, S3EnvFileArg{Container: args[i+1], S3Arn: args[i+2]})
				i += 2
			case "-s", "--secret":
				if i+3 >= len(args) {
					return nil, fmt.Errorf("option %s requires 3 arguments: <container> <name> <parameter_name>", arg)
				}
				parsed.SecretOverrides = append(parsed.SecretOverrides, SecretOverride{Container: args[i+1], Name: args[i+2], ValueFrom: args[i+3]})
				i += 3
			case "--secrets-env-file":
				if i+2 >= len(args) {
					return nil, fmt.Errorf("option %s requires 2 arguments: <container> <file_path>", arg)
				}
				parsed.SecretsEnvFiles = append(parsed.SecretsEnvFiles, EnvFileArg{Container: args[i+1], FilePath: args[i+2]})
				i += 2
			case "-d", "--docker-label":
				if i+3 >= len(args) {
					return nil, fmt.Errorf("option %s requires 3 arguments: <container> <name> <value>", arg)
				}
				parsed.DockerLabels = append(parsed.DockerLabels, DockerLabelArg{Container: args[i+1], Name: args[i+2], Value: args[i+3]})
				i += 3
			case "-u", "--ulimit":
				if i+4 >= len(args) {
					return nil, fmt.Errorf("option %s requires 4 arguments: <container> <name> <softLimit> <hardLimit>", arg)
				}
				soft, _ := strconv.ParseInt(args[i+3], 10, 64)
				hard, _ := strconv.ParseInt(args[i+4], 10, 64)
				parsed.Ulimits = append(parsed.Ulimits, UlimitOverride{Container: args[i+1], Name: args[i+2], SoftLimit: soft, HardLimit: hard})
				i += 4
			case "--system-control":
				if i+3 >= len(args) {
					return nil, fmt.Errorf("option %s requires 3 arguments: <container> <namespace> <value>", arg)
				}
				parsed.Sysctls = append(parsed.Sysctls, SysctlOverride{Container: args[i+1], Namespace: args[i+2], Value: args[i+3]})
				i += 3
			case "-p", "--port":
				if i+3 >= len(args) {
					return nil, fmt.Errorf("option %s requires 3 arguments: <container> <containerPort> <hostPort>", arg)
				}
				cPort, _ := strconv.ParseInt(args[i+2], 10, 64)
				hPort, _ := strconv.ParseInt(args[i+3], 10, 64)
				parsed.Ports = append(parsed.Ports, PortOverride{Container: args[i+1], ContainerPort: cPort, HostPort: hPort})
				i += 3
			case "-m", "--mount":
				if i+3 >= len(args) {
					return nil, fmt.Errorf("option %s requires 3 arguments: <container> <volumeName> <containerPath>", arg)
				}
				parsed.Mounts = append(parsed.Mounts, MountOverride{Container: args[i+1], SourceVolume: args[i+2], Path: args[i+3]})
				i += 3
			case "-l", "--log":
				if i+4 >= len(args) {
					return nil, fmt.Errorf("option %s requires 4 arguments: <container> <logDriver> <optionName> <optionValue>", arg)
				}
				parsed.Logs = append(parsed.Logs, LogOverride{Container: args[i+1], LogDriver: args[i+2], Name: args[i+3], Value: args[i+4]})
				i += 4
			case "-r", "--role":
				if i+1 >= len(args) {
					return nil, fmt.Errorf("missing value for option %s", arg)
				}
				parsed.Role = args[i+1]
				i++
			case "-x", "--execution-role":
				if i+1 >= len(args) {
					return nil, fmt.Errorf("missing value for option %s", arg)
				}
				parsed.ExecutionRole = args[i+1]
				i++
			case "--runtime-platform":
				if i+2 >= len(args) {
					return nil, fmt.Errorf("option %s requires 2 arguments: <cpuArchitecture> <operatingSystemFamily>", arg)
				}
				parsed.RuntimePlatform = &RuntimePlatformArg{CpuArch: args[i+1], OsFamily: args[i+2]}
				i += 2
			case "--task":
				if i+1 >= len(args) {
					return nil, fmt.Errorf("missing value for option %s", arg)
				}
				parsed.Task = args[i+1]
				i++
			case "--region":
				if i+1 >= len(args) {
					return nil, fmt.Errorf("missing value for option %s", arg)
				}
				parsed.Region = args[i+1]
				i++
			case "--access-key-id":
				if i+1 >= len(args) {
					return nil, fmt.Errorf("missing value for option %s", arg)
				}
				parsed.AccessKeyId = args[i+1]
				i++
			case "--secret-access-key":
				if i+1 >= len(args) {
					return nil, fmt.Errorf("missing value for option %s", arg)
				}
				parsed.SecretAccessKey = args[i+1]
				i++
			case "--profile":
				if i+1 >= len(args) {
					return nil, fmt.Errorf("missing value for option %s", arg)
				}
				parsed.Profile = args[i+1]
				i++
			case "--account":
				if i+1 >= len(args) {
					return nil, fmt.Errorf("missing value for option %s", arg)
				}
				parsed.Account = args[i+1]
				i++
			case "--assume-role":
				if i+1 >= len(args) {
					return nil, fmt.Errorf("missing value for option %s", arg)
				}
				parsed.AssumeRole = args[i+1]
				i++
			case "--timeout":
				if i+1 >= len(args) {
					return nil, fmt.Errorf("missing value for option %s", arg)
				}
				t, _ := strconv.Atoi(args[i+1])
				parsed.Timeout = t
				i++
			case "--force-new-deployment":
				parsed.ForceNewDeployment = true
			case "--ignore-warnings":
				parsed.IgnoreWarnings = true
			case "--sleep-time":
				if i+1 >= len(args) {
					return nil, fmt.Errorf("missing value for option %s", arg)
				}
				s, _ := strconv.Atoi(args[i+1])
				parsed.SleepTime = s
				i++
			case "--diff":
				parsed.Diff = true
			case "--no-diff":
				parsed.Diff = false
			case "--deregister":
				parsed.Deregister = true
			case "--no-deregister":
				parsed.Deregister = false
			case "--rollback":
				parsed.Rollback = true
			case "--no-rollback":
				parsed.Rollback = false
			case "--exclusive-env":
				parsed.ExclusiveEnv = true
			case "--exclusive-secrets":
				parsed.ExclusiveSecrets = true
			case "--exclusive-docker-labels":
				parsed.ExclusiveDockerLabels = true
			case "--exclusive-s3-env-file":
				parsed.ExclusiveS3EnvFile = true
			case "--exclusive-ulimits":
				parsed.ExclusiveUlimits = true
			case "--exclusive-system-controls":
				parsed.ExclusiveSysctls = true
			case "--exclusive-ports":
				parsed.ExclusivePorts = true
			case "--exclusive-mounts":
				parsed.ExclusiveMounts = true
			case "--volume":
				if i+2 >= len(args) {
					return nil, fmt.Errorf("option %s requires 2 arguments: <volumeName> <hostSourcePath>", arg)
				}
				parsed.Volumes = append(parsed.Volumes, VolumeOverride{Name: args[i+1], SourcePath: args[i+2]})
				i += 2
			case "--add-container":
				if i+1 >= len(args) {
					return nil, fmt.Errorf("missing value for option %s", arg)
				}
				parsed.AddContainers = append(parsed.AddContainers, args[i+1])
				i++
			case "--remove-container":
				if i+1 >= len(args) {
					return nil, fmt.Errorf("missing value for option %s", arg)
				}
				parsed.RemoveContainers = append(parsed.RemoveContainers, args[i+1])
				i++
			default:
				return nil, fmt.Errorf("unknown option: %s", arg)
			}
		} else {
			positionals = append(positionals, arg)
		}
	}

	if parsed.Help || parsed.Version {
		return parsed, nil
	}

	if len(positionals) < 2 {
		return nil, fmt.Errorf("missing required positional arguments: <cluster> <service>")
	}
	parsed.Cluster = positionals[0]
	parsed.Service = positionals[1]

	return parsed, nil
}
