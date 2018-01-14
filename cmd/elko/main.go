// Public Domain (-) 2018-present, The Elko Authors.
// See the Elko UNLICENSE file for details.

package main

import (
	"github.com/tav/golly/optparse"
)

func createOpts(cmd string, usage string) *optparse.Parser {
	opt := optparse.New("Usage: elko " + cmd + "\n\n  " + usage + "\n")
	opt.HideHelpOpt = true
	return opt
}

func main() {

	commands := map[string]func([]string, string){
		"run":             cmdRun,
		"service-manager": cmdServiceManager,
	}

	usage := map[string]string{
		"run":             "Build and run the specified services in dev mode",
		"service-manager": "Run just the service manager component",
	}

	description := `	┏━╸╻  ╻┏ ┏━┓
	┣╸ ┃  ┣┻┓┃ ┃
	┗━╸┗━╸╹ ╹┗━┛

  Elko is an auto-scaling microservices framework.`

	optparse.Commands("elko", "elko 0.0.1", commands, usage, description)

}
