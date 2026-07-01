package main

import (
	log "github.com/sirupsen/logrus"

	"os"
	"os/signal"
	"syscall"
)

const VERSION = "v0.0.1"

type Configuration struct {
	// Identifier
	Environment string
	Stack       string
	Service     string

	Platform string
	Cluster  string
	ProcType string
}

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
	config := defaultConfiguration()

	config, err := fetchConfigFromEnvionment(config)
	if err != nil {
		log.Fatalf("Failed to fetch config: %+v\n", err)
	}
	log.Debugf("Running with config [%+v]\n", config)

	envVars, err := getEnvVars(config)
	if err != nil {
		log.Fatalf("Failed to Get env: %+v\n", err)
	}
	command, err := procCommand(config)
	if err != nil {
		log.Fatalf("Failed to get command: %+v\n", err)
	}

	runCommand(command, envVars, func(pid int) {
		if config.Platform != "local" && config.ProcType != "wait" {
			log.Debugf("Set cycling for PID[%d]\n", pid)
			go selfCycling(pid)
		}
		go func() {
			sigs := make(chan os.Signal)
			signal.Notify(sigs, syscall.SIGTERM, syscall.SIGKILL)
			for {
				select {
				case sig := <-sigs:
					switch sig {
					case syscall.SIGTERM, syscall.SIGKILL:
						log.Infof("Got signal [%+v]. Process to exit.\n", sig)
						signalKill(pid, sig)
					}
				}
			}
		}()
		if config.Platform == "local" {
			// Detect stack
			// Run update dependencies
			// go rubyDevDependency()
			// go nodeDevDependency()
		}
	})
}

func defaultConfiguration() Configuration {
	return Configuration{
		Environment: "development",
		Platform:    "local",
		ProcType:    "wait",
	}
}
