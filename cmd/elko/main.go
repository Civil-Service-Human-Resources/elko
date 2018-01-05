// Public Domain (-) 2018-present, The Elko Authors.
// See the Elko UNLICENSE file for details.

package main

import (
	"os"

	"github.com/tav/golly/optparse"
)

func main() {

	// Handle the hidden `elko run-node` command.
	if len(os.Args) > 1 && os.Args[1] == "run-node" {
		runNode()
	}

	commands := map[string]func([]string, string){
		"build":    cmdBuild,
		"clean":    cmdClean,
		"deploy":   cmdDeploy,
		"exec":     cmdExec,
		"history":  cmdHistory,
		"info":     cmdInfo,
		"logs":     cmdLogs,
		"node":     cmdNode,
		"nuke":     cmdNuke,
		"rollback": cmdRollback,
		"run":      cmdRun,
		"test":     cmdTest,
		"unlock":   cmdUnlock,
	}

	usage := map[string]string{
		"build":    "Build the docker images for services",
		"clean":    "Remove the meta directory and auto-generated files",
		"deploy":   "Build the services and deploy the latest version",
		"exec":     "Build the container for a service and run it as an executable",
		"history":  "Display the deployment history",
		"info":     "Display system info for debugging",
		"logs":     "Stream logs from an elko server",
		"node":     "Run an elko node",
		"nuke":     "Remove docker images and containers",
		"rollback": "Rollback to a previous deployment",
		"run":      "Build the services and deploy without pushing to the registry",
		"test":     "Build and test services",
		"unlock":   "Release the deploy lock",
	}

	description := `  Elko builds on top of Docker and Consul to deliver a highly automated
  setup for managing dev environments and deployment clusters.`

	optparse.Commands("elko", "elko 0.1", commands, usage, description)

}
