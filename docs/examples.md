# Example configurations of mercury

Below is a list of example configurations suiting your purpose

## Adding Mercury as Global Loadbalancer to existing Loadbalance solutions
If you are using multiple cloud environment you'll notice that they have their 'internal' Global loadbalance mechanisms across their datacenter locations. But very few have the option to add nodes outside of their network. Which results in the need of a extra layer, using DNS to loadbalance across multiple providers or datacenter

```
[settings]
  manage_network_interfaces = "yes"
  enable_proxy = "no"
[cluster]
  name = "MY_GLB_POOL"

  [cluster.binding]
  name = "loadbalancer1.example.com"
  addr = "1.2.3.4:9000"
  authkey = "test"

  [[cluster.nodes]]
  name = "loadbalancer2.example.com"
  addr = "1.2.3.5:9000"
  authkey = "test"

[dns]
  binding = "loadbalancer1.example.com"
  port = 53
  [dns.domains."example.com"]
    ttl = 11
  [dns.domains."example.com".soa]

[loadbalancer.settings]
  default_balance_method = "roundrobin"

[loadbalancer.pools.INTERNAL_VIP.listener]

  [loadbalancer.pools.INTERNAL_VIP.backends.myapp.dnsentry]
    domain = "example.com"
    hostnames = "www"
    ttl = 30
  [loadbalancer.pools.INTERNAL_VIP.backends.myapp.healthcheck]
    type = "tcpconnect"
  [[loadbalancer.pools.INTERNAL_VIP.backends.myapp.nodes]]
    hostname = "webserver1"
    ip = "1.2.3.4"
    port = 80
  [[loadbalancer.pools.INTERNAL_VIP.backends.myapp.nodes]]
    hostname = "webserver2"
    ip = "2.3.4.5"
    port = 80
```
The above creates a cluster of the servers: loadbalancer1 and loadbalancer2, both located at different datacenters and providing DNS.
When a client requests a DNS entry, it will go to one of the loadbalancers, and they will redirect the client to the ip of webserver1 or webserver 2 based on "roundrobin" balance method. Should one of the servers or loadbalancers fail, all traffic will be redirected to the remaining loadbalancer and webserver.
Note that the IP of the webserver in this case does not have to be the Webserver its self, but can also be the existing loadbalancer at your existing datacenter, which in turn contains many more servers for local Loadbalancing.

## Adding Mercury as Global Loadbalancer with its internal Loadbalance solution (with SSL offloading)
If you are using multiple cloud environment you'll notice that they have their 'internal' Global loadbalance mechanisms across their datacenter locations. But very few have the option to add nodes outside of their network. Which results in the need of a extra layer, using DNS to loadbalance across multiple providers or datacenter. This option also replaces any existing loadbalancer, making Mercury your primary entry point for all traffic.

```
[cluster]
  name = "MY_GLB_POOL"

  [cluster.binding]
  name = "loadbalancer1.example.com"
  addr = "1.2.3.4:9000"
  authkey = "test"

  [[cluster.nodes]]
  name = "loadbalancer2.example.com"
  addr = "1.2.3.5:9000"
  authkey = "test"

  [cluster.settings.tls]
    certificatekey = "build/test/ssl/self_signed_certificate.key"
    certificatefile = "build/test/ssl/self_signed_certificate.crt"
    insecureskipverify = true

[web]
  [web.tls]
    certificatekey = "build/test/ssl/self_signed_certificate.key"
    certificatefile = "build/test/ssl/self_signed_certificate.crt"

[dns]
  binding = "loadbalancer1.example.com"
  port = 53
  [dns.domains."example.com"]
    ttl = 11
  [dns.domains."example.com".soa]

[loadbalancer.settings]
  default_balance_method = "roundrobin"

[loadbalancer.pools.INTERNAL_VIP.backends.myapp]
  [loadbalancer.pools.INTERNAL_VIP.listener]
    ip = "127.0.0.1"
    port = 8080
    mode = "https"
    [loadbalancer.pools.INTERNAL_VIP.listener.tls]
      certificatekey = "build/test/ssl/self_signed_certificate.key"
      certificatefile = "build/test/ssl/self_signed_certificate.crt"

  [loadbalancer.pools.INTERNAL_VIP.backends.myapp.dnsentry]
    hostnames = ["default"]
    connectmode="http"
  [loadbalancer.pools.INTERNAL_VIP.backends.myapp.dnsentry]
    domain = "example.com"
    hostnames = "www"
    ttl = 30
  [loadbalancer.pools.INTERNAL_VIP.backends.myapp.healthcheck]
    type = "tcpconnect"
  [[loadbalancer.pools.INTERNAL_VIP.backends.myapp.nodes]]
    hostname = "webserver1"
    ip = "1.2.3.4"
    port = 80
  [[loadbalancer.pools.INTERNAL_VIP.backends.myapp.nodes]]
    hostname = "webserver2"
    ip = "2.3.4.5"
    port = 80
```
In this example we still enable global loadbalancing using DNS, however we now add a listener IP which will accept incomming connections on HTTPS using the provided SSL certificates. This listener exists on both loadbalancers, and once a client connects to this listener on the port specified (8080) the loadbalancer will create a new connection to the local node, and forward the request.

## Adding Mercury as (Global) Loadbalancer serving multiple Hostnames
Quite often are you loadbalancing multiple domains which point to different servers. If this is the case you can specify the hostname which the backend serves. Mercury will look at the requested host header, and forward the request to the backend which has this host header configured.

```
[cluster]
  name = "MY_GLB_POOL"
  [cluster.binding]
  name = "localhost1"
  addr = "127.0.0.1:9000"
  authkey = "test"
  [[cluster.nodes]]
  name = "localhost2"
  addr = "127.0.0.1:10000"
  authkey = "test"

[dns]
  binding = "localhost"
  port = 15353
  [dns.domains."example.com"]
    ttl = 11
  [dns.domains."example.com".soa]

[loadbalancer.settings]
  default_balance_method = "roundrobin"

[loadbalancer.pools.INTERNAL_VIP.listener]
    ip = "127.0.0.1"
    port = 8080
    mode = "http"
  [loadbalancer.pools.INTERNAL_VIP.backends.myapp]
    hostnames = ["www.example.com"]
    connectmode="http"
  [loadbalancer.pools.INTERNAL_VIP.backends.myapp.dnsentry]
    domain = "example.com"
    hostnames = "www"
  [loadbalancer.pools.INTERNAL_VIP.backends.myapp.healthcheck]
    type = "tcpconnect"
  [[loadbalancer.pools.INTERNAL_VIP.backends.myapp.nodes]]
    hostname = "webserver1"
    ip = "1.2.3.4"
    port = 80
  [[loadbalancer.pools.INTERNAL_VIP.backends.myapp.nodes]]
    hostname = "webserver2"
    ip = "2.3.4.5"
    port = 80

  [loadbalancer.pools.INTERNAL_VIP.backends.myimageapp]
    hostnames = ["image.example.com"]
    connectmode="http"
  [loadbalancer.pools.INTERNAL_VIP.backends.myimageapp.dnsentry]
    domain = "example.com"
    hostnames = "image"
  [loadbalancer.pools.INTERNAL_VIP.backends.myimageapp.healthcheck]
    type = "tcpconnect"
  [[loadbalancer.pools.INTERNAL_VIP.backends.myimageapp.nodes]]
    hostname = "webserver3"
    ip = "3.4.5.6"
    port = 80
  [[loadbalancer.pools.INTERNAL_VIP.backends.myimageapp.nodes]]
    hostname = "webserver4"
    ip = "4.5.6.7"
    port = 80
```
In this example we have 2 domains: www.example.com and image.example.com, requests made to www will be forwarded to webserver1+2 and requests made to images.example.com will be forwarded to webserver3+4
