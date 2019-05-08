package core

import (
	"log"
	"os"
	"os/signal"
	"reflect"
	"syscall"
	"time"

	"github.com/rdoorn/cluster"
	"github.com/rdoorn/old/glbv2/pkg/tlsconfig"
	"github.com/schubergphilis/mercury.v2/internal/logging"
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

type Cluster struct {
	log     logging.SimpleLogger
	manager *cluster.Manager
	config  *ClusterConfig
	quit    chan struct{}
}

func NewCluster(config *ClusterConfig) *Cluster {
	cluster.ChannelBufferSize = 100
	manager := cluster.NewManager(config.Binding.Name, config.Binding.AuthKey)
	return &Cluster{
		config:  config,
		manager: manager,
		quit:    make(chan struct{}),
	}
}

func (c *Cluster) WithLogger(l logging.SimpleLogger) {
	c.log = l
}

func (c *Cluster) start() {
	c.log.Infof("starting cluster")
	// read ssl certificate
	tlsConfig, err := tlsconfig.LoadCertificate(c.config.TLSConfig)
	if err != nil {
		log.Fatal(err)
	}

	err = c.manager.ListenAndServeTLS(c.config.Binding.Addr, tlsConfig)
	if err != nil {
		log.Fatal(err)
	}

	// tracing only when debug is set
	if c.config.Settings.LogLevel == "debug" {
		go c.enableTracing()
	}
}

func (c *Cluster) connectNodes() {
	configured := c.manager.NodesConfigured()
	for _, node := range c.config.Nodes {
		if _, ok := configured[node.Name]; ok {
			delete(configured, node.Name)
		}
		// Add newly configured nodes
		if !c.manager.NodeConfigured(node.Name) {
			c.manager.AddNode(node.Name, node.Addr)
		}
	}

	//  remove old cluster nodes
	for name := range configured {
		c.manager.RemoveNode(name)
	}
}

func (c *Cluster) stop() {
	c.disableTracing()
	c.manager.Shutdown()
}

func (c *Cluster) reload(new *ClusterConfig) {
	if reflect.DeepEqual(c.config, new) {
		c.log.Infof("no cluster changes in reload")
	}

	needRestart := false
	needNodeUpdate := false
	needLogLevelUpdate := false
	// if the listener changed, we need to reconnect to all nodes
	if !reflect.DeepEqual(c.config.Binding, new.Binding) ||
		c.config.Settings.ConnectInterval != new.Settings.ConnectInterval ||
		c.config.Settings.ConnectTimeout != new.Settings.ConnectTimeout ||
		c.config.Settings.JoinDelay != new.Settings.JoinDelay ||
		c.config.Settings.PingInterval != new.Settings.PingInterval ||
		c.config.Settings.ReadTimeout != new.Settings.ReadTimeout {

		needRestart = true
	}

	if c.config.Settings.LogLevel != new.Settings.LogLevel {
		needLogLevelUpdate = true
	}

	if !reflect.DeepEqual(c.config.Nodes, new.Nodes) {
		needNodeUpdate = true
	}

	// update config and execute actions
	c.config = new
	if needRestart {
		c.stop()
		c.start()
		c.connectNodes()
		return
	}

	if needNodeUpdate {
		c.connectNodes()
	}

	if needLogLevelUpdate {
		c.enableTracing()
	}

}

func (c *Cluster) disableTracing() {
	cluster.LogTraffic = false
}

func (c *Cluster) enableTracing() {
	if c.config.Settings.LogLevel == "debug" {
		cluster.LogTraffic = true
		go func() {
			for {
				select {
				case logEntry := <-c.manager.Log:
					c.log.Debugf(logEntry)
				case <-c.quit:
					return
				}
			}
		}()
	}
}

func (c *Cluster) Handler() {
	c.log.Infof("cluster client handler started")
	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, syscall.SIGUSR1)

	for {
		select {
		case _ = <-signalChan:

		case node := <-c.manager.NodeJoin:
			// discard old config of the node
			// 	manager.dnsdiscard <- node

			// send request for remote config to node
			c.manager.ToNode <- cluster.NodeMessage{Node: node, Message: ClusterPacketConfigRequest{}}

			// send our config to the remote node
			// TODO

		case node := <-c.manager.NodeLeave:

			c.log.Warnf("node leaving", "node", node)
			// discard old config of the node
			// if forced offline (manually) then we discard
			// manager.dnsdiscard <- node

			// if failed (timeout, other) then we put offline
			// manager.dnsoffline <- node

		case packet := <-c.manager.FromCluster:
			switch packet.DataType {
			case "config.ClusterPacketConfigRequest":

				// send current config to remote node
				// go clusterDNSUpdateSingleBroadcastAll(cl, packet.Name)

			case "config.ClusterPacketGlobalDNSUpdate":
				// we received a dns update
				/*dnsupdate := &config.ClusterPacketGlobalDNSUpdate{}
				err := packet.Message(dnsupdate)
				if err != nil {
					continue
				}*/

				// update dns
				//manager.dnsupdates <- dnsupdate

			case "config.ClusterPacketGlobalDNSRemove":

				// remove a dns entry

			case "config.ClusterPacketGlbalDNSStatisticsUpdate":

				// update dns statistics

			case "config.ClusterPacketClearProxyStatistics":

				// clear proxy statistics

			default:
				c.log.Warnf("received an unknown cluster request", "node", packet.Name, "request", packet.DataType)
			}
		}
	}
}
