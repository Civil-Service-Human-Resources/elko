// Public Domain (-) 2018-present, The Elko Authors.
// See the Elko UNLICENSE file for details.

package servicemanager

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/binary"
	"errors"
	"fmt"
	"net"
	"strings"
	"sync"
	"time"

	"github.com/tav/elko/pkg/servicemanager/protocol"
	"github.com/tav/golly/log"
)

type serviceMap struct {
	sync.RWMutex
	instances map[uint64]*service
	lastID    uint64
	services  map[string][]*service
}

// Server represents a service manager instance.
type Server struct {
	cluster interface {
		Maintain()
	}
	config     *Config
	nodeID     string
	queues     map[string][]*protocol.ClientRequest
	serviceMap *serviceMap
}

func (s *Server) handle(c net.Conn) {
	c.SetReadDeadline(time.Now().Add(s.config.CallTimeout))
	data := make([]byte, 1)
	for {
		n, err := c.Read(data)
		if err != nil {
			log.Errorf("servicemanager: couldn't read from fresh connection: %s", err)
			c.Close()
			return
		}
		if n == 0 {
			continue
		}
		break
	}
	switch data[0] {
	case 1:
		go handleService(s, c)
	case 2:
		// handle node connection
	default:
		log.Errorf("servicemanager: unknown connection type: %q", data[0])
		c.Close()
	}
}

// Run binds the service manager to the configured port and starts handling
// requests.
func (s *Server) Run() error {
	l, err := net.Listen("tcp", fmt.Sprintf(":%d", s.config.Port))
	if err != nil {
		return err
	}
	defer l.Close()
	log.Infof("Service Manager is listening on port %d", s.config.Port)
	// go s.removeDeadServices()
	for {
		c, err := l.Accept()
		if err != nil {
			return err
		}
		go s.handle(c)
	}
}

// New instantiates a service manager with the given config.
func New(cfg *Config) (*Server, error) {
	s := &Server{
		config: cfg,
	}
	switch cfg.ClusterType {
	case "":
		s.cluster = &SoloCluster{}
	case "consul":
		consul := &ConsulCluster{Servers: []string{}}
		if cfg.ClusterEndpoints == "" {
			return nil, errors.New("servicemanager: missing --cluster-endpoints value")
		}
		if cfg.ClusterID == "" {
			return nil, errors.New("servicemanager: missing --cluster-id value")
		}
		if cfg.ClusterKey == "" {
			return nil, errors.New("servicemanager: missing --cluster-key value")
		}
		for _, server := range strings.Split(cfg.ClusterEndpoints, ",") {
			server := strings.TrimSpace(server)
			if server != "" {
				consul.Servers = append(consul.Servers, server)
			}
		}
		if len(consul.Servers) == 0 {
			return nil, errors.New("servicemanager: empty list specified in --cluster-endpoints")
		}
		s.cluster = consul
	default:
		return nil, fmt.Errorf("servicemanager: unknown cluster type: %q", cfg.ClusterType)
	}
	idPrefix := ""
	switch cfg.HostMetadata {
	case "":
		idPrefix = "dev"
	case "azure":
		prefix, err := getAzureInstanceID()
		if err != nil {
			return nil, err
		}
		idPrefix = prefix
	default:
		return nil, fmt.Errorf("servicemanager: unknown metadata server type: %q", cfg.HostMetadata)
	}
	id, err := genNodeID(idPrefix)
	if err != nil {
		return nil, err
	}
	s.nodeID = id
	s.queues = map[string][]*protocol.ClientRequest{}
	s.serviceMap = &serviceMap{
		instances: map[uint64]*service{},
		services:  map[string][]*service{},
	}
	return s, nil
}

func genNodeID(prefix string) (string, error) {
	suffix := make([]byte, 9)
	binary.BigEndian.PutUint32(suffix[:4], uint32(time.Now().Unix()))
	n, err := rand.Read(suffix[4:9])
	if err != nil {
		return "", err
	}
	if n != 5 {
		return "", errors.New("servicemanager: unable to generate the random portion of the node id")
	}
	return prefix + "-" + base64.RawURLEncoding.EncodeToString(suffix), nil
}
