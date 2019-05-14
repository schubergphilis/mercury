package core

import (
	"errors"
	"fmt"
	"reflect"
	"time"

	"github.com/schubergphilis/mercury.v2/internal/logging"
	"github.com/schubergphilis/mercury.v2/pkg/cluster"
	"github.com/schubergphilis/mercury.v2/pkg/tlsconfig"
)

// Cluster contains the cluster settings
type ClusterConfig struct {
	Binding   ClusterConfigNode     `toml:"binding" json:"binding"`
	Nodes     []ClusterConfigNode   `toml:"nodes" json:"nodes"`
	Settings  ClusterConfigSettings `toml:"settings" json:"settings"`
	TLSConfig tlsconfig.TLSConfig   `toml:"tls" json:"tls"`
}

// ClusterNode contains the connection details of the cluster node
type ClusterConfigNode struct {
	Name    string `toml:"name" json:"name"`
	Addr    string `toml:"addr" json:"addr"`
	AuthKey string `toml:"authkey" json:"authkey"`
}

type ClusterConfigSettings struct {
	PingInterval    time.Duration `toml:"ping_interval" json:"ping_interval"`       // how over to ping a node
	JoinDelay       time.Duration `toml:"join_delay" json:"join_delay"`             // delay before announcing node (done to prevent duplicate join messages on simultainious connects) (must be shorter than ping timeout)
	ReadTimeout     time.Duration `toml:"read_timeout" json:"read_timeout"`         // timeout when to discard a node as broken if not read anything before this
	ConnectInterval time.Duration `toml:"connect_interval" json:"connect_interval"` // how often we try to reconnect to lost cluster nodes
	ConnectTimeout  time.Duration `toml:"connect_timeout" json:"connect_timeout"`   // how long to try to connect to a node
	LogLevel        string        `toml:"log_level" json:"log_level"`               // log level for cluster events
}

func (h *Handler) startCluster() {
	// only start if we have a config
	if h.config.ClusterConfig == nil {
		h.log.Warnf("custer not started", "error", "no cluster config provided")
	}

	h.setClusterLog()

	// only start if we're not already started
	if h.runningConfig.ClusterConfig != nil {
		h.log.Warnf("custer not started", "error", "cluster is already started")
	}

	tlsConfig, err := tlsconfig.LoadCertificate(h.config.ClusterConfig.TLSConfig)
	if err != nil {
		h.log.Warnf("custer failed to load ssl config", "error", err)
	}

	h.cluster.WithKey(h.config.ClusterConfig.Binding.AuthKey)
	h.cluster.WithName(h.config.ClusterConfig.Binding.Name)
	h.cluster.WithAddr(h.config.ClusterConfig.Binding.Addr)
	h.cluster.WithTLS(tlsConfig)

	// start listener
	h.cluster.Start()

	// add nodes on initial start
	for _, node := range h.config.ClusterConfig.Nodes {
		h.cluster.AddNode(node.Name, node.Addr)
	}

	// tracing only when debug is set
	if h.config.ClusterConfig.Settings.LogLevel == "debug" {
		go h.enableClusterTracing()
	}

	h.runningConfig.ClusterConfig = h.config.ClusterConfig
}

func (h *Handler) stopCluster() {
	if h.runningConfig.ClusterConfig == nil {
		h.log.Warnf("custer not stopped", "error", "no cluster config provided")
	}
	h.cluster.Stop()
	h.runningConfig.ClusterConfig = nil
}

func (h *Handler) disableClusterTracing() {
	cluster.LogTraffic = false
}

func (h *Handler) enableClusterTracing() {
	cluster.LogTraffic = true
}

func (h *Handler) reloadCluster() {
	// if any of these things change, we need to restart the cluster part
	if h.runningConfig.ClusterConfig.Binding.Addr != h.config.ClusterConfig.Binding.Addr ||
		h.runningConfig.ClusterConfig.Binding.Name != h.config.ClusterConfig.Binding.Name ||
		h.runningConfig.ClusterConfig.Binding.AuthKey != h.config.ClusterConfig.Binding.AuthKey ||
		h.runningConfig.ClusterConfig.Settings.ConnectInterval != h.config.ClusterConfig.Settings.ConnectInterval ||
		h.runningConfig.ClusterConfig.Settings.ConnectTimeout != h.config.ClusterConfig.Settings.ConnectTimeout ||
		h.runningConfig.ClusterConfig.Settings.JoinDelay != h.config.ClusterConfig.Settings.JoinDelay ||
		h.runningConfig.ClusterConfig.Settings.PingInterval != h.config.ClusterConfig.Settings.PingInterval ||
		h.runningConfig.ClusterConfig.Settings.ReadTimeout != h.config.ClusterConfig.Settings.ReadTimeout ||
		!reflect.DeepEqual(h.runningConfig.ClusterConfig.TLSConfig, h.config.ClusterConfig.TLSConfig) {

		h.stopCluster()
		h.startCluster()
		return
	}

	// if node number changes, collect the changes
	old := []string{}
	for _, node := range h.runningConfig.ClusterConfig.Nodes {
		old = append(old, node.uuid())
	}

	new := []string{}
	for _, node := range h.config.ClusterConfig.Nodes {
		new = append(new, node.uuid())
	}

	added, deleted := sliceAddedAndDeleted(old, new)
	// delete old nodes first (will also delet enodes who's ip changed)
	for _, uuid := range deleted {
		if node, err := h.runningConfig.ClusterConfig.NodeByUUID(uuid); err != nil {
			h.cluster.RemoveNode(node.Name)
		}
	}

	// add new nodes
	for _, uuid := range added {
		if node, err := h.config.ClusterConfig.NodeByUUID(uuid); err != nil {
			h.cluster.AddNode(node.Name, node.Addr)
		}
	}

	// TODO: enable/disable tracing
	if h.config.LoggingConfig.ClusterLevel != h.runningConfig.LoggingConfig.ClusterLevel {
		h.setClusterLog()
	}

}

func (n *ClusterConfigNode) uuid() string {
	return fmt.Sprintf("%s__%s__%s", n.Name, n.Addr, n.AuthKey)
}

func (c *ClusterConfig) NodeByUUID(uuid string) (*ClusterConfigNode, error) {
	for _, n := range c.Nodes {
		if n.uuid() == uuid {
			return &n, nil
		}
	}
	return nil, errors.New("node not found")
}

func (h *Handler) setClusterLog() {
	// set log level
	logLevel, err := logging.ToLevel(h.config.LoggingConfig.ClusterLevel)
	if err != nil {
		h.log.Fatalf("unkown log level configured for cluster", "level", h.config.LoggingConfig.ClusterLevel, "error", err)
	}
	h.log.Infof("cluster log level", "level", h.config.LoggingConfig.ClusterLevel)
	var prefix []interface{}
	prefix = append(prefix, "func")
	prefix = append(prefix, "cluster")
	h.cluster.WithLogger(&logging.Wrapper{Log: h.LogProvider, Level: logLevel, Prefix: prefix})
	h.runningConfig.LoggingConfig.ClusterLevel = h.config.LoggingConfig.ClusterLevel
}
