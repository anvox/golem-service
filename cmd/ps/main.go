package main

import (
	"github.com/olekukonko/tablewriter"
	log "github.com/sirupsen/logrus"
	"nullprogram.com/x/optparse"

	"fmt"
	"os"
	"strings"

	service_ps "github.com/anvox/golem-service/pkg/process"
)

const VERSION = "v0.0.1"

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
	options := []optparse.Option{
		{"help", 'h', optparse.KindNone},
		{"version", 'v', optparse.KindNone},
		{"environment", 'e', optparse.KindRequired},
		{"stack", 't', optparse.KindRequired},
		{"service", 's', optparse.KindRequired},
		{"process", 'p', optparse.KindOptional},
	}

	results, rest, err := optparse.Parse(options, os.Args)

	if err != nil {
		fmt.Println("Cannot parse arguments!")
		printHelp()
		os.Exit(1)
	}

	cmd := "list"
	var process []string
	id := service_ps.Identifier{}
	for _, result := range results {
		switch result.Long {
		case "help":
			printHelp()
			os.Exit(0)
		case "version":
			printVersion()
			os.Exit(0)
		case "environment":
			id.Environment = result.Optarg
		case "stack":
			id.Stack = result.Optarg
		case "service":
			id.Service = result.Optarg
		case "process":
			process = strings.Split(result.Optarg, ",")
		}
	}

	if id.Environment == "" || id.Stack == "" {
		fmt.Println("Environment and Stack are required. Please run with -h/--help for more info.")
		os.Exit(2)
	}

	if err := id.Validate(); err != nil {
		fmt.Printf("Error: %s\n", err)
		os.Exit(6)
	}

	if len(rest) >= 1 {
		cmd = rest[0]
		rest = rest[1:len(rest)]

		if cmd != "list" && cmd != "kill" {
			fmt.Println("Subcommand must be list|kill. Please run with -h/--help for more info.")
			os.Exit(3)
		}
	}

	if cmd == "list" && len(rest) > 0 {
		fmt.Println("Subcommand list requires no argument. Please run with -h/--help for more info.")
		os.Exit(4)
	} else if cmd == "kill" && (len(process) <= 0 && len(rest) != 1) {
		fmt.Println("Subcommand kill requires a proccess type or a process id. Please run with -h/--help for more info.")
		os.Exit(5)
	}

	log.Debugf("Identifier: %+v", id)
	log.Debugf("Command: %s", cmd)
	for _, configArg := range rest {
		log.Debugf("Arg: %+v", configArg)
	}

	// TODO: [AV] Support scale command
	switch cmd {
	case "list":
		processInfos, err := service_ps.List(id, process)
		if err != nil {
			fmt.Printf("\nError while getting list of processes: %+v\n", err)
		} else {
			fmt.Printf("\nProcess info of [%s-%s]\n", id.Environment, id.Stack)
			printProcessInfo(processInfos)
		}
	case "kill":
		if len(rest) == 1 {
			service_ps.KillOne(id, rest[0])
			fmt.Println("Process is queueing to stop.")
		} else if process != nil {
			service_ps.Kill(id, process)
		}
	default:
		return
	}
}

func printProcessInfo(processInfos []service_ps.ProcessInfo) {
	table := tablewriter.NewWriter(os.Stdout)
	table.SetHeader([]string{"ID", "Service", "ProcType", "Status", "Started At"})
	table.SetColMinWidth(0, 40)
	table.SetAutoWrapText(false)
	table.SetBorder(false)
	for _, row := range processInfos {
		table.Append([]string{
			row.Id,
			row.Service,
			row.ProcType,
			row.Status,
			row.StartedAt,
		})
	}
	table.Render()
}
