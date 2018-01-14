// Public Domain (-) 2018-present, The Elko Authors.
// See the Elko UNLICENSE file for details.

package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"path/filepath"

	"github.com/tav/elko/pkg/config"
	"gopkg.in/yaml.v2"
)

func cmdRun(argv []string, usage string) {

	opts := createOpts("run [SERVICE ...] [OPTIONS]",
		`Build and run the specified services in dev mode. If no services are
  specified, it defaults to running all available services.`)

	port := opts.Flags("-p", "--port").Label("PORT").Int("the port to listen on (defaults to any available)")

	services := opts.Parse(argv)

	_ = port
	_ = services

	root, err := config.GetRoot()
	if err != nil {
		log.Fatal(err)
	}

	cfgFile, err := ioutil.ReadFile(filepath.Join(root, ".elko", "config.yaml"))
	if err != nil {
		log.Fatal(err)
	}

	cfg := &config.Elko{}
	err = yaml.UnmarshalStrict(cfgFile, cfg)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("%#v\n", cfg)

}
