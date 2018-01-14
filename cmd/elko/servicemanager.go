// Public Domain (-) 2018-present, The Elko Authors.
// See the Elko UNLICENSE file for details.

package main

import (
	"github.com/tav/elko/pkg/servicemanager"
	"github.com/tav/golly/log"
)

func cmdServiceManager(argv []string, usage string) {

	opts := createOpts("service-manager [OPTIONS]", usage)

	callTimeout := opts.Flags("--call-timeout").Label("DURATION").Duration(
		"the default timeout duration for connections and service calls [10s]")

	clusterEndpoints := opts.Flags("--cluster-endpoints").Label("LIST").String(
		"comma-delimited list of endpoints for the cluster metadata server(s)")

	clusterID := opts.Flags("--cluster-id").Label("ID").String(
		"the cluster ID")

	clusterKey := opts.Flags("--cluster-key").Label("KEY").String(
		"the access key for the cluster metadata server(s)")

	clusterType := opts.Flags("--cluster-type").Label("TYPE").String(
		"the type of the cluster metadata server(s), e.g. consul, etcd, gcd, etc.")

	heartbeat := opts.Flags("--heartbeat").Label("DURATION").Duration(
		"the default duration of service heartbeats [10s]")

	hostMetadata := opts.Flags("--host-metadata").Label("TYPE").String(
		"the type of the host metadata server, e.g. aws, azure, gcp, etc.")

	leaseDuration := opts.Flags("--lease-duration").Label("DURATION").Duration(
		"the duration of the node lease [7s]")

	port := opts.Flags("--port").Label("PORT").Int("the port to listen on [9000]")

	productionMode := opts.Flags("--production-mode").Bool(
		"enable production mode [false]")

	shutdownTimeout := opts.Flags("--shutdown-timeout").Label("DURATION").Duration(
		"the duration of the service shutdown timeout [30m]")

	opts.Parse(argv)

	server, err := servicemanager.New(&servicemanager.Config{
		CallTimeout:      *callTimeout,
		ClusterEndpoints: *clusterEndpoints,
		ClusterID:        *clusterID,
		ClusterKey:       *clusterKey,
		ClusterType:      *clusterType,
		Heartbeat:        *heartbeat,
		HostMetadata:     *hostMetadata,
		LeaseDuration:    *leaseDuration,
		Port:             *port,
		ProductionMode:   *productionMode,
		ShutdownTimeout:  *shutdownTimeout,
	})
	if err != nil {
		log.Fatal(err)
	}

	err = server.Run()
	if err != nil {
		log.Fatal(err)
	}

}
