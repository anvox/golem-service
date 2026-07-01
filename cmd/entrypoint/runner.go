package main

import (
	log "github.com/sirupsen/logrus"

	"fmt"
	"math/rand"
	"os"
	"os/exec"
	"syscall"
	"time"
)

func runCommand(command string, envVars map[string]string, afterRunHook func(int)) {
	switch command {
	case "wait":
		log.Debugln("Starting wait runner")
		runWait(afterRunHook)
	case "ssh":
		log.Debugln("Starting ssh runner")
		runSsh(envVars, afterRunHook)
	case "bash":
		log.Debugln("Starting bash runner")
		runBash(envVars, afterRunHook)
	default:
		log.Debugf("Starting runner with command [%s]\n", command)
		runProcType(command, envVars, afterRunHook)
	}
}

func runWait(_ func(int)) {
	time.Sleep((216 + 24*60) * time.Minute)
}

// TODO: [AV] Need test
func runBash(envVars map[string]string, _ func(int)) {
	binary, lookErr := exec.LookPath("bash")
	if lookErr != nil {
		log.Infof("Error while looking for bash: %+v. Trying sh", lookErr)
		binary, lookErr = exec.LookPath("sh")
		if lookErr != nil {
			log.Infof("Error while looking for sh: %+v. Exiting", lookErr)
			return
		}
	}
	syscall.Exec(binary, []string{}, append(os.Environ(), formatEnv(envVars)...))
}

func runProcType(command string,
	envVars map[string]string,
	afterRunHook func(int)) error {

	binary, lookErr := exec.LookPath("sh")
	if lookErr != nil {
		log.Infof("Error while looking for sh: %+v\n", lookErr)
		return lookErr
	}
	cmd := exec.Command(binary, "-c", command)
	cmd.Dir = "/app"
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Env = append(os.Environ(), formatEnv(envVars)...)

	err := cmd.Start()
	if err != nil {
		log.Infof("Failed to start command [%s]: %+v\n", command, err)
		return err
	}
	afterRunHook(cmd.Process.Pid)

	return cmd.Wait()
}

func signalKill(pid int, sig os.Signal) {
	if sig == syscall.SIGTERM {
		syscall.Kill(pid, syscall.SIGTERM)
	} else {
		syscall.Kill(pid, syscall.SIGKILL)
	}
}

func selfCycling(pid int) {
	rand.Seed(time.Now().UnixNano())
	lifetimeInMinute := rand.Intn(216) + 24*60
	log.Infof("Self-cycling after [%d] minutes\n", lifetimeInMinute)
	select {
	case <-time.After(time.Duration(lifetimeInMinute) * time.Minute):
		log.Infof("SIGKILL [%d]\n", pid)
		syscall.Kill(pid, syscall.SIGKILL)
	case <-time.After(time.Duration(lifetimeInMinute)*time.Minute + 30*time.Second):
		log.Infof("Force SIGTERM [%d]\n", pid)
		syscall.Kill(pid, syscall.SIGTERM)
	}
}

func formatEnv(envVars map[string]string) []string {
	result := []string{}
	for key, value := range envVars {
		result = append(result, fmt.Sprintf("%s=%s", key, value))
	}

	return result
}
