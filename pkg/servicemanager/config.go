// Public Domain (-) 2018-present, The Elko Authors.
// See the Elko UNLICENSE file for details.

package servicemanager

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"time"
)

type Config struct {
	CallTimeout      time.Duration
	ClusterEndpoints string
	ClusterID        string
	ClusterKey       string
	ClusterType      string
	Heartbeat        time.Duration
	HostMetadata     string
	LeaseDuration    time.Duration
	Port             int
	ProductionMode   bool
	ShutdownTimeout  time.Duration
}

type ConsulCluster struct {
	Key     string
	Servers []string
}

func (c *ConsulCluster) Maintain() {
}

type SoloCluster struct {
}

func (c *SoloCluster) Maintain() {
}

func getAzureInstanceID() (string, error) {
	req, err := http.NewRequest("GET",
		"http://169.254.169.254/metadata/instance/compute/vmId?api-version=2017-04-02&format=text", nil)
	if err != nil {
		return "", err
	}
	req.Header.Add("Metadata", "True")
	client := &http.Client{
		Timeout: 10 * time.Second,
	}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return "", fmt.Errorf("servicemanager: got %d response code from the Azure metadata service",
			resp.StatusCode)
	}
	id, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	return string(id), nil
}
