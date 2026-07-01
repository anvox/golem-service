package main

import (
	"fmt"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ecs"
)

// mutateTaskDefinition applies mutations to the baseline task definition
func mutateTaskDefinition(baseTd *ecs.TaskDefinition, parsed *ParsedArgs, diffs *[]string) (*ecs.RegisterTaskDefinitionInput, error) {
	input := &ecs.RegisterTaskDefinitionInput{
		Family:                  baseTd.Family,
		TaskRoleArn:             baseTd.TaskRoleArn,
		ExecutionRoleArn:        baseTd.ExecutionRoleArn,
		NetworkMode:             baseTd.NetworkMode,
		Volumes:                 baseTd.Volumes,
		PlacementConstraints:    baseTd.PlacementConstraints,
		RequiresCompatibilities: baseTd.RequiresCompatibilities,
		Cpu:                     baseTd.Cpu,
		Memory:                  baseTd.Memory,
		RuntimePlatform:         baseTd.RuntimePlatform,
		ContainerDefinitions:    baseTd.ContainerDefinitions,
	}

	// 1. Add / Remove Containers
	if len(parsed.AddContainers) > 0 {
		var beforeList []string
		for _, c := range input.ContainerDefinitions {
			beforeList = append(beforeList, aws.StringValue(c.Name))
		}
		for _, addName := range parsed.AddContainers {
			exists := false
			for _, c := range input.ContainerDefinitions {
				if aws.StringValue(c.Name) == addName {
					exists = true
					break
				}
			}
			if !exists {
				input.ContainerDefinitions = append(input.ContainerDefinitions, &ecs.ContainerDefinition{
					Name:              aws.String(addName),
					Image:             aws.String("PLACEHOLDER"),
					Cpu:               aws.Int64(0),
					MemoryReservation: aws.Int64(128),
					Essential:         aws.Bool(true),
				})
			}
		}
		var afterList []string
		for _, c := range input.ContainerDefinitions {
			afterList = append(afterList, aws.StringValue(c.Name))
		}
		if fmt.Sprintf("%v", beforeList) != fmt.Sprintf("%v", afterList) {
			*diffs = append(*diffs, fmt.Sprintf("Changed containers to: %v (was: %v)", afterList, beforeList))
		}
	}

	if len(parsed.RemoveContainers) > 0 {
		var beforeList []string
		for _, c := range input.ContainerDefinitions {
			beforeList = append(beforeList, aws.StringValue(c.Name))
		}
		var filtered []*ecs.ContainerDefinition
		for _, c := range input.ContainerDefinitions {
			removed := false
			for _, rm := range parsed.RemoveContainers {
				if aws.StringValue(c.Name) == rm {
					removed = true
					break
				}
			}
			if !removed {
				filtered = append(filtered, c)
			}
		}
		input.ContainerDefinitions = filtered

		var afterList []string
		for _, c := range input.ContainerDefinitions {
			afterList = append(afterList, aws.StringValue(c.Name))
		}
		if fmt.Sprintf("%v", beforeList) != fmt.Sprintf("%v", afterList) {
			*diffs = append(*diffs, fmt.Sprintf("Changed containers to: %v (was: %v)", afterList, beforeList))
		}
	}

	// Validate container references in CLI overrides
	allContainerNames := make(map[string]bool)
	for _, c := range input.ContainerDefinitions {
		allContainerNames[aws.StringValue(c.Name)] = true
	}

	validateContainer := func(name string) error {
		if !allContainerNames[name] {
			return fmt.Errorf("Unknown container: %s", name)
		}
		return nil
	}

	// 2. Apply Container Property Updates
	for _, c := range input.ContainerDefinitions {
		cName := aws.StringValue(c.Name)

		// Image Updates
		for _, imgOver := range parsed.Images {
			if imgOver.Container == cName {
				if err := validateContainer(cName); err != nil {
					return nil, err
				}
				oldImg := aws.StringValue(c.Image)
				if oldImg != imgOver.Image {
					*diffs = append(*diffs, fmt.Sprintf("Changed image of container %q to: %q (was: %q)", cName, imgOver.Image, oldImg))
					c.Image = aws.String(imgOver.Image)
				}
			}
		}

		if parsed.Tag != "" {
			oldImg := aws.StringValue(c.Image)
			idx := strings.LastIndex(oldImg, ":")
			var newImg string
			if idx != -1 {
				newImg = oldImg[:idx] + ":" + parsed.Tag
			} else {
				newImg = oldImg + ":" + parsed.Tag
			}
			if oldImg != newImg {
				*diffs = append(*diffs, fmt.Sprintf("Changed image of container %q to: %q (was: %q)", cName, newImg, oldImg))
				c.Image = aws.String(newImg)
			}
		}

		// Command Updates
		for _, cmdOver := range parsed.Commands {
			if cmdOver.Container == cName {
				if err := validateContainer(cName); err != nil {
					return nil, err
				}
				newCmd := parseCommand(cmdOver.Command)
				oldCmdJoined := joinCommand(c.Command)
				newCmdJoined := joinCommand(newCmd)
				if oldCmdJoined != newCmdJoined {
					*diffs = append(*diffs, fmt.Sprintf("Changed command of container %q to: %q (was: %q)", cName, newCmdJoined, oldCmdJoined))
					c.Command = newCmd
				}
			}
		}

		// CPU & Memory Override
		for _, cpuOver := range parsed.CpuOverrides {
			if cpuOver.Container == cName {
				if err := validateContainer(cName); err != nil {
					return nil, err
				}
				oldCpu := aws.Int64Value(c.Cpu)
				if oldCpu != cpuOver.Cpu {
					*diffs = append(*diffs, fmt.Sprintf("Changed cpu of container %q to: %d (was: %d)", cName, cpuOver.Cpu, oldCpu))
					c.Cpu = aws.Int64(cpuOver.Cpu)
				}
			}
		}

		for _, memOver := range parsed.MemoryOverrides {
			if memOver.Container == cName {
				if err := validateContainer(cName); err != nil {
					return nil, err
				}
				oldMem := aws.Int64Value(c.Memory)
				if oldMem != memOver.Memory {
					*diffs = append(*diffs, fmt.Sprintf("Changed memory of container %q to: %d (was: %d)", cName, memOver.Memory, oldMem))
					c.Memory = aws.Int64(memOver.Memory)
				}
			}
		}

		for _, memResOver := range parsed.MemReservationOverrides {
			if memResOver.Container == cName {
				if err := validateContainer(cName); err != nil {
					return nil, err
				}
				oldMemRes := aws.Int64Value(c.MemoryReservation)
				if oldMemRes != memResOver.MemoryReservation {
					*diffs = append(*diffs, fmt.Sprintf("Changed memoryReservation of container %q to: %d (was: %d)", cName, memResOver.MemoryReservation, oldMemRes))
					c.MemoryReservation = aws.Int64(memResOver.MemoryReservation)
				}
			}
		}

		// Privileged & Essential
		for _, privOver := range parsed.PrivilegedOverrides {
			if privOver.Container == cName {
				if err := validateContainer(cName); err != nil {
					return nil, err
				}
				oldPriv := aws.BoolValue(c.Privileged)
				if oldPriv != privOver.Privileged {
					*diffs = append(*diffs, fmt.Sprintf("Changed privileged of container %q to: %t (was: %t)", cName, privOver.Privileged, oldPriv))
					c.Privileged = aws.Bool(privOver.Privileged)
				}
			}
		}

		for _, essOver := range parsed.EssentialOverrides {
			if essOver.Container == cName {
				if err := validateContainer(cName); err != nil {
					return nil, err
				}
				oldEss := aws.BoolValue(c.Essential)
				if oldEss != essOver.Essential {
					*diffs = append(*diffs, fmt.Sprintf("Changed essential of container %q to: %t (was: %t)", cName, essOver.Essential, oldEss))
					c.Essential = aws.Bool(essOver.Essential)
				}
			}
		}

		// HealthCheck
		for _, hcOver := range parsed.HealthCheckOverrides {
			if hcOver.Container == cName {
				if err := validateContainer(cName); err != nil {
					return nil, err
				}
				newHc := &ecs.HealthCheck{
					Command:     []*string{aws.String("CMD-SHELL"), aws.String(hcOver.Command)},
					Interval:    aws.Int64(hcOver.Interval),
					Timeout:     aws.Int64(hcOver.Timeout),
					Retries:     aws.Int64(hcOver.Retries),
					StartPeriod: aws.Int64(hcOver.StartPeriod),
				}
				*diffs = append(*diffs, fmt.Sprintf("Changed healthCheck of container %q to: Command %q (interval: %d, timeout: %d, retries: %d, start_period: %d)", cName, hcOver.Command, hcOver.Interval, hcOver.Timeout, hcOver.Retries, hcOver.StartPeriod))
				c.HealthCheck = newHc
			}
		}

		// Environment Variables
		err := applyEnvironment(c, parsed.EnvOverrides, parsed.EnvFiles, parsed.ExclusiveEnv, diffs)
		if err != nil {
			return nil, err
		}

		// Secrets
		err = applySecrets(c, parsed.SecretOverrides, parsed.SecretsEnvFiles, parsed.ExclusiveSecrets, diffs)
		if err != nil {
			return nil, err
		}

		// Docker Labels
		err = applyDockerLabels(c, parsed.DockerLabels, parsed.ExclusiveDockerLabels, diffs)
		if err != nil {
			return nil, err
		}

		// S3 Env Files
		err = applyS3EnvFiles(c, parsed.S3EnvFiles, parsed.ExclusiveS3EnvFile, diffs)
		if err != nil {
			return nil, err
		}

		// Port Mappings
		err = applyPortMappings(c, parsed.Ports, parsed.ExclusivePorts, diffs)
		if err != nil {
			return nil, err
		}

		// Mount Points
		err = applyMountPoints(c, parsed.Mounts, parsed.ExclusiveMounts, diffs)
		if err != nil {
			return nil, err
		}

		// Log Configuration
		err = applyLogConfiguration(c, parsed.Logs, diffs)
		if err != nil {
			return nil, err
		}

		// Ulimits
		err = applyUlimits(c, parsed.Ulimits, parsed.ExclusiveUlimits, diffs)
		if err != nil {
			return nil, err
		}

		// System Controls (Sysctls)
		err = applySysctls(c, parsed.Sysctls, parsed.ExclusiveSysctls, diffs)
		if err != nil {
			return nil, err
		}
	}

	// 3. Apply Task Level Property Updates
	if parsed.TaskCpu != "" {
		oldCpu := aws.StringValue(input.Cpu)
		if oldCpu != parsed.TaskCpu {
			*diffs = append(*diffs, fmt.Sprintf("Changed cpu to: %q (was: %q)", parsed.TaskCpu, oldCpu))
			input.Cpu = aws.String(parsed.TaskCpu)
		}
	}

	if parsed.TaskMemory != "" {
		oldMem := aws.StringValue(input.Memory)
		if oldMem != parsed.TaskMemory {
			*diffs = append(*diffs, fmt.Sprintf("Changed memory to: %q (was: %q)", parsed.TaskMemory, oldMem))
			input.Memory = aws.String(parsed.TaskMemory)
		}
	}

	if parsed.Role != "" {
		oldRole := aws.StringValue(input.TaskRoleArn)
		if oldRole != parsed.Role {
			*diffs = append(*diffs, fmt.Sprintf("Changed role_arn to: %q (was: %q)", parsed.Role, oldRole))
			input.TaskRoleArn = aws.String(parsed.Role)
		}
	}

	if parsed.ExecutionRole != "" {
		oldExecRole := aws.StringValue(input.ExecutionRoleArn)
		if oldExecRole != parsed.ExecutionRole {
			*diffs = append(*diffs, fmt.Sprintf("Changed execution_role_arn to: %q (was: %q)", parsed.ExecutionRole, oldExecRole))
			input.ExecutionRoleArn = aws.String(parsed.ExecutionRole)
		}
	}

	if parsed.RuntimePlatform != nil {
		oldPlatform := input.RuntimePlatform
		newPlatform := &ecs.RuntimePlatform{
			CpuArchitecture:       aws.String(parsed.RuntimePlatform.CpuArch),
			OperatingSystemFamily: aws.String(parsed.RuntimePlatform.OsFamily),
		}
		oldStr := ""
		if oldPlatform != nil {
			oldStr = fmt.Sprintf("%s %s", aws.StringValue(oldPlatform.CpuArchitecture), aws.StringValue(oldPlatform.OperatingSystemFamily))
		}
		newStr := fmt.Sprintf("%s %s", parsed.RuntimePlatform.CpuArch, parsed.RuntimePlatform.OsFamily)
		if oldStr != newStr {
			*diffs = append(*diffs, fmt.Sprintf("Changed runtimePlatform to: %q (was: %q)", newStr, oldStr))
			input.RuntimePlatform = newPlatform
		}
	}

	// Volume Updates
	if len(parsed.Volumes) > 0 {
		var newVolumes []*ecs.Volume
		for _, volOver := range parsed.Volumes {
			newVolumes = append(newVolumes, &ecs.Volume{
				Name: aws.String(volOver.Name),
				Host: &ecs.HostVolumeProperties{
					SourcePath: aws.String(volOver.SourcePath),
				},
			})
		}

		// Merge with existing volumes that are not overridden
		for _, existingVol := range input.Volumes {
			overridden := false
			for _, nv := range newVolumes {
				if aws.StringValue(nv.Name) == aws.StringValue(existingVol.Name) {
					overridden = true
					break
				}
			}
			if !overridden {
				newVolumes = append(newVolumes, existingVol)
			}
		}

		oldVolStr := fmt.Sprintf("%v", input.Volumes)
		newVolStr := fmt.Sprintf("%v", newVolumes)
		if oldVolStr != newVolStr {
			*diffs = append(*diffs, fmt.Sprintf("Changed volumes to: %v (was: %v)", newVolStr, oldVolStr))
			input.Volumes = newVolumes
		}
	}

	return input, nil
}

func parseCommand(cmd string) []*string {
	if strings.HasPrefix(cmd, "[") && strings.HasSuffix(cmd, "]") {
		// Clean brackets and split by comma or spaces
		inner := strings.Trim(cmd, "[]")
		parts := strings.Split(inner, ",")
		var res []*string
		for _, part := range parts {
			cleaned := strings.Trim(strings.TrimSpace(part), `"'`)
			res = append(res, aws.String(cleaned))
		}
		return res
	}
	parts := strings.Fields(cmd)
	var res []*string
	for _, part := range parts {
		res = append(res, aws.String(part))
	}
	return res
}

func joinCommand(cmd []*string) string {
	var parts []string
	for _, c := range cmd {
		parts = append(parts, aws.StringValue(c))
	}
	return strings.Join(parts, " ")
}

func applyEnvironment(container *ecs.ContainerDefinition, overrides []EnvVarOverride, envFiles []EnvFileArg, exclusive bool, diffs *[]string) error {
	var newEnv []*ecs.KeyValuePair

	cName := aws.StringValue(container.Name)

	for _, envFile := range envFiles {
		if envFile.Container == cName {
			vars, err := readEnvFile(envFile.FilePath)
			if err != nil {
				return err
			}
			for k, v := range vars {
				newEnv = append(newEnv, &ecs.KeyValuePair{
					Name:  aws.String(k),
					Value: aws.String(v),
				})
			}
		}
	}

	for _, over := range overrides {
		if over.Container == cName {
			var temp []*ecs.KeyValuePair
			for _, kv := range newEnv {
				if aws.StringValue(kv.Name) != over.Name {
					temp = append(temp, kv)
				}
			}
			newEnv = temp
			newEnv = append(newEnv, &ecs.KeyValuePair{
				Name:  aws.String(over.Name),
				Value: aws.String(over.Value),
			})
		}
	}

	var merged []*ecs.KeyValuePair
	if exclusive {
		merged = newEnv
	} else {
		for _, existing := range container.Environment {
			replaced := false
			for _, ne := range newEnv {
				if aws.StringValue(ne.Name) == aws.StringValue(existing.Name) {
					replaced = true
					break
				}
			}
			if !replaced {
				merged = append(merged, existing)
			}
		}
		merged = append(merged, newEnv...)
	}

	oldMap := make(map[string]string)
	for _, kv := range container.Environment {
		oldMap[aws.StringValue(kv.Name)] = aws.StringValue(kv.Value)
	}
	newMap := make(map[string]string)
	for _, kv := range merged {
		newMap[aws.StringValue(kv.Name)] = aws.StringValue(kv.Value)
	}

	for k, v := range newMap {
		oldV, exists := oldMap[k]
		if !exists || oldV != v {
			*diffs = append(*diffs, fmt.Sprintf("Changed environment %q of container %q to: %q (was: %q)", k, cName, v, oldV))
		}
	}
	for k, oldV := range oldMap {
		if _, exists := newMap[k]; !exists {
			*diffs = append(*diffs, fmt.Sprintf("Removed environment %q of container %q (was: %q)", k, cName, oldV))
		}
	}

	container.Environment = merged
	return nil
}

func applySecrets(container *ecs.ContainerDefinition, overrides []SecretOverride, secretsFiles []EnvFileArg, exclusive bool, diffs *[]string) error {
	var newSecrets []*ecs.Secret
	cName := aws.StringValue(container.Name)

	for _, sFile := range secretsFiles {
		if sFile.Container == cName {
			vars, err := readEnvFile(sFile.FilePath)
			if err != nil {
				return err
			}
			for k, v := range vars {
				newSecrets = append(newSecrets, &ecs.Secret{
					Name:      aws.String(k),
					ValueFrom: aws.String(v),
				})
			}
		}
	}

	for _, over := range overrides {
		if over.Container == cName {
			var temp []*ecs.Secret
			for _, s := range newSecrets {
				if aws.StringValue(s.Name) != over.Name {
					temp = append(temp, s)
				}
			}
			newSecrets = temp
			newSecrets = append(newSecrets, &ecs.Secret{
				Name:      aws.String(over.Name),
				ValueFrom: aws.String(over.ValueFrom),
			})
		}
	}

	var merged []*ecs.Secret
	if exclusive {
		merged = newSecrets
	} else {
		for _, existing := range container.Secrets {
			replaced := false
			for _, ns := range newSecrets {
				if aws.StringValue(ns.Name) == aws.StringValue(existing.Name) {
					replaced = true
					break
				}
			}
			if !replaced {
				merged = append(merged, existing)
			}
		}
		merged = append(merged, newSecrets...)
	}

	oldMap := make(map[string]string)
	for _, s := range container.Secrets {
		oldMap[aws.StringValue(s.Name)] = aws.StringValue(s.ValueFrom)
	}
	newMap := make(map[string]string)
	for _, s := range merged {
		newMap[aws.StringValue(s.Name)] = aws.StringValue(s.ValueFrom)
	}

	for k, v := range newMap {
		oldV, exists := oldMap[k]
		if !exists || oldV != v {
			*diffs = append(*diffs, fmt.Sprintf("Changed secret %q of container %q to: %q (was: %q)", k, cName, v, oldV))
		}
	}
	for k, oldV := range oldMap {
		if _, exists := newMap[k]; !exists {
			*diffs = append(*diffs, fmt.Sprintf("Removed secret %q of container %q (was: %q)", k, cName, oldV))
		}
	}

	container.Secrets = merged
	return nil
}

func applyDockerLabels(container *ecs.ContainerDefinition, overrides []DockerLabelArg, exclusive bool, diffs *[]string) error {
	cName := aws.StringValue(container.Name)
	newLabels := make(map[string]*string)

	for _, over := range overrides {
		if over.Container == cName {
			newLabels[over.Name] = aws.String(over.Value)
		}
	}

	var merged map[string]*string
	if exclusive {
		merged = newLabels
	} else {
		merged = make(map[string]*string)
		for k, v := range container.DockerLabels {
			merged[k] = v
		}
		for k, v := range newLabels {
			merged[k] = v
		}
	}

	oldMap := make(map[string]string)
	for k, v := range container.DockerLabels {
		oldMap[k] = aws.StringValue(v)
	}
	newMap := make(map[string]string)
	for k, v := range merged {
		newMap[k] = aws.StringValue(v)
	}

	for k, v := range newMap {
		oldV, exists := oldMap[k]
		if !exists || oldV != v {
			*diffs = append(*diffs, fmt.Sprintf("Changed dockerLabel %q of container %q to: %q (was: %q)", k, cName, v, oldV))
		}
	}
	for k, oldV := range oldMap {
		if _, exists := newMap[k]; !exists {
			*diffs = append(*diffs, fmt.Sprintf("Removed dockerLabel %q of container %q (was: %q)", k, cName, oldV))
		}
	}

	container.DockerLabels = merged
	return nil
}

func applyS3EnvFiles(container *ecs.ContainerDefinition, overrides []S3EnvFileArg, exclusive bool, diffs *[]string) error {
	cName := aws.StringValue(container.Name)
	var newFiles []*ecs.EnvironmentFile

	for _, over := range overrides {
		if over.Container == cName {
			newFiles = append(newFiles, &ecs.EnvironmentFile{
				Type:  aws.String("s3"),
				Value: aws.String(over.S3Arn),
			})
		}
	}

	var merged []*ecs.EnvironmentFile
	if exclusive {
		merged = newFiles
	} else {
		for _, existing := range container.EnvironmentFiles {
			replaced := false
			for _, nf := range newFiles {
				if aws.StringValue(nf.Value) == aws.StringValue(existing.Value) {
					replaced = true
					break
				}
			}
			if !replaced {
				merged = append(merged, existing)
			}
		}
		merged = append(merged, newFiles...)
	}

	oldStr := fmt.Sprintf("%v", container.EnvironmentFiles)
	newStr := fmt.Sprintf("%v", merged)
	if oldStr != newStr {
		*diffs = append(*diffs, fmt.Sprintf("Changed environmentFiles of container %q to: %s (was: %s)", cName, newStr, oldStr))
	}

	container.EnvironmentFiles = merged
	return nil
}

func applyPortMappings(container *ecs.ContainerDefinition, overrides []PortOverride, exclusive bool, diffs *[]string) error {
	cName := aws.StringValue(container.Name)
	var newPorts []*ecs.PortMapping

	for _, over := range overrides {
		if over.Container == cName {
			newPorts = append(newPorts, &ecs.PortMapping{
				ContainerPort: aws.Int64(over.ContainerPort),
				HostPort:      aws.Int64(over.HostPort),
				Protocol:      aws.String("tcp"),
			})
		}
	}

	var merged []*ecs.PortMapping
	if exclusive {
		merged = newPorts
	} else {
		for _, existing := range container.PortMappings {
			overridden := false
			for _, np := range newPorts {
				if aws.Int64Value(np.ContainerPort) == aws.Int64Value(existing.ContainerPort) {
					overridden = true
					break
				}
			}
			if !overridden {
				merged = append(merged, existing)
			}
		}
		merged = append(merged, newPorts...)
	}

	oldStr := fmt.Sprintf("%v", container.PortMappings)
	newStr := fmt.Sprintf("%v", merged)
	if oldStr != newStr {
		*diffs = append(*diffs, fmt.Sprintf("Changed portMappings of container %q to: %s (was: %s)", cName, newStr, oldStr))
	}

	container.PortMappings = merged
	return nil
}

func applyMountPoints(container *ecs.ContainerDefinition, overrides []MountOverride, exclusive bool, diffs *[]string) error {
	cName := aws.StringValue(container.Name)
	var newMounts []*ecs.MountPoint

	for _, over := range overrides {
		if over.Container == cName {
			newMounts = append(newMounts, &ecs.MountPoint{
				SourceVolume:  aws.String(over.SourceVolume),
				ContainerPath: aws.String(over.Path),
				ReadOnly:      aws.Bool(false),
			})
		}
	}

	var merged []*ecs.MountPoint
	if exclusive {
		merged = newMounts
	} else {
		for _, existing := range container.MountPoints {
			overridden := false
			for _, nm := range newMounts {
				if aws.StringValue(nm.SourceVolume) == aws.StringValue(existing.SourceVolume) {
					overridden = true
					break
				}
			}
			if !overridden {
				merged = append(merged, existing)
			}
		}
		merged = append(merged, newMounts...)
	}

	oldStr := fmt.Sprintf("%v", container.MountPoints)
	newStr := fmt.Sprintf("%v", merged)
	if oldStr != newStr {
		*diffs = append(*diffs, fmt.Sprintf("Changed mountPoints of container %q to: %s (was: %s)", cName, newStr, oldStr))
	}

	container.MountPoints = merged
	return nil
}

func applyLogConfiguration(container *ecs.ContainerDefinition, overrides []LogOverride, diffs *[]string) error {
	cName := aws.StringValue(container.Name)

	var logDrv string
	options := make(map[string]*string)

	hasOverride := false
	for _, over := range overrides {
		if over.Container == cName {
			hasOverride = true
			logDrv = over.LogDriver
			options[over.Name] = aws.String(over.Value)
		}
	}

	if !hasOverride {
		return nil
	}

	var oldLog *ecs.LogConfiguration
	if container.LogConfiguration != nil {
		oldLog = container.LogConfiguration
	} else {
		oldLog = &ecs.LogConfiguration{}
	}

	mergedOptions := make(map[string]*string)
	for k, v := range oldLog.Options {
		mergedOptions[k] = v
	}
	for k, v := range options {
		mergedOptions[k] = v
	}

	newLog := &ecs.LogConfiguration{
		LogDriver: aws.String(logDrv),
		Options:   mergedOptions,
	}

	oldStr := fmt.Sprintf("Driver: %s, Options: %v", aws.StringValue(oldLog.LogDriver), oldLog.Options)
	newStr := fmt.Sprintf("Driver: %s, Options: %v", logDrv, mergedOptions)
	if oldStr != newStr {
		*diffs = append(*diffs, fmt.Sprintf("Changed logConfiguration of container %q to: %s (was: %s)", cName, newStr, oldStr))
	}

	container.LogConfiguration = newLog
	return nil
}

func applyUlimits(container *ecs.ContainerDefinition, overrides []UlimitOverride, exclusive bool, diffs *[]string) error {
	cName := aws.StringValue(container.Name)
	var newUlimits []*ecs.Ulimit

	for _, over := range overrides {
		if over.Container == cName {
			newUlimits = append(newUlimits, &ecs.Ulimit{
				Name:      aws.String(over.Name),
				SoftLimit: aws.Int64(over.SoftLimit),
				HardLimit: aws.Int64(over.HardLimit),
			})
		}
	}

	var merged []*ecs.Ulimit
	if exclusive {
		merged = newUlimits
	} else {
		for _, existing := range container.Ulimits {
			overridden := false
			for _, nu := range newUlimits {
				if aws.StringValue(nu.Name) == aws.StringValue(existing.Name) {
					overridden = true
					break
				}
			}
			if !overridden {
				merged = append(merged, existing)
			}
		}
		merged = append(merged, newUlimits...)
	}

	oldStr := fmt.Sprintf("%v", container.Ulimits)
	newStr := fmt.Sprintf("%v", merged)
	if oldStr != newStr {
		*diffs = append(*diffs, fmt.Sprintf("Changed ulimits of container %q to: %s (was: %s)", cName, newStr, oldStr))
	}

	container.Ulimits = merged
	return nil
}

func applySysctls(container *ecs.ContainerDefinition, overrides []SysctlOverride, exclusive bool, diffs *[]string) error {
	cName := aws.StringValue(container.Name)
	var newSysctls []*ecs.SystemControl

	for _, over := range overrides {
		if over.Container == cName {
			newSysctls = append(newSysctls, &ecs.SystemControl{
				Namespace: aws.String(over.Namespace),
				Value:     aws.String(over.Value),
			})
		}
	}

	var merged []*ecs.SystemControl
	if exclusive {
		merged = newSysctls
	} else {
		for _, existing := range container.SystemControls {
			overridden := false
			for _, ns := range newSysctls {
				if aws.StringValue(ns.Namespace) == aws.StringValue(existing.Namespace) {
					overridden = true
					break
				}
			}
			if !overridden {
				merged = append(merged, existing)
			}
		}
		merged = append(merged, newSysctls...)
	}

	oldStr := fmt.Sprintf("%v", container.SystemControls)
	newStr := fmt.Sprintf("%v", merged)
	if oldStr != newStr {
		*diffs = append(*diffs, fmt.Sprintf("Changed systemControls of container %q to: %s (was: %s)", cName, newStr, oldStr))
	}

	container.SystemControls = merged
	return nil
}
