# Mercury Configuration

Mercury uses a [toml](https://github.com/toml-lang/toml) configuration file.
this documentation will go through each section to explain the configurable items of each section

## Global Settings
Settings are defined in the `[settings]` block.
options are:

Key | Option | Default | Values | Description
--- | --- | --- | --- | ---
[settings] | manage_network_interfaces | "yes" | "yes"/"no" | allow mercury to add vip's to the network interfaces - required for internal proxy or for haproxy who does not add vip's.
[settings] | enable_proxy              | "yes" | "yes"/"no" | use internal proxy for loadbalancing - not needed for external proxy programs, or dns only setup.

## Logging
Log settings are defined in the `[logging]` block.
options are:

Key | Option | Default | Values | Description
--- | --- | --- | --- | ---
[logging] | level | "info" | "(debug|info|warn|error)" | log level with debug being the most informative.
[logging] | output | "/var/log/mercury" | "(stdout|file)"| location to write the log information to Web Settings.


## Web
Web interface settings are defined in the `[web]` block.
options are:

Key | Option | Default | Values | Description
--- | --- | --- | --- | ---
[web] | binding | "0.0.0.0" | string | ip for the web interface to listen on
[web] | port | 9001 | int | port for the web interface to listen on
[web.tls] | tls | none | see TLS Attributes | TLS certificate information required for SSL


## Cluster
Cluster settings are defined in the `[cluster]` block.
options are:

Key | Option | Default | Values | Description
--- | --- | --- | --- | ---
[cluster.binding] | name | "" | string | Name of the cluster group
[cluster.binding] | addr | "" | string | ip to bind on for cluster communication
[cluster.binding] | authkey | "" | string | key required to connect to this cluster
[cluster.settings] | connection_timeout | 10 | int (seconds) | timeout connecting to remote cluster
[cluster.settings] | connection_retry_interval | 10 | int (seconds) | time in between retries connecting to the cluster
[cluster.settings] | ping_interval | 11 | int (seconds) |  how often to send a ping to the remote host
[cluster.settings] | ping_timeout | 10 | int (seconds) |  host long to wait for a ping timeout (generally 1 second less then interval)
[cluster.settings] | port | 9000 | int | port to listen on for cluster communication
[cluster.tls] | none | see TLS Attributes | TLS certificate information required for SSL
[[cluster.nodes]] | | | array of loadbalancer nodes to connect to and form a cluster
[[cluster.nodes]] | name | string | name of a cluster node
[[cluster.nodes]] | addr | string | address of a cluster node
[[cluster.nodes]] | authkey | string | key used to connect to this cluster node

## DNS
DNS settings are defined in the `[dns]` block.
options are:

Key | Option | Default | Values | Description
--- | --- | --- | --- | ---
[dns] | binding | "0.0.0.0" | string | binding ip for dns service
[dns] | port | 53 | int | binding port for dns service
[dns] | allow_forwarding | [] | ["ip/mask"] | array of cidrs to allow dns forwarding requests
[dns] | allow_requests | [ "A", "AAAA", "NS", "MX", "SOA", "TXT", "CAA", "ANY", "CNAME", "MB", "MG", "MR", "WKS", "PTR", "HINFO", "MINFO", "SPF" ] | ["types"] | array of dns requests types we respond to

## TLS Attributes
TLS attributes are appended to any of the TLS keys in the config.

Usable in the settings for `cluster`, `webserver`, `listener` and `backend`

Key | Option | Default | Values | Description
--- | --- | --- | --- | ---
[parent.tls] | minversion | "VersionTLS12" | string | Minimum TLS version required for this listener
[parent.tls] | maxversion | "" | string | Maximum TLS version required for this listener
[parent.tls] | ciphersuites | all | ["cipher"] | Cipher suites used by the listener (note that TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256 is required for HTTP/2 support. see https://golang.org/pkg/crypto/tls/#pkg-constants for details
[parent.tls] | curvepreferences | all | ["curve"] | Curve preference used by the listener. see https://golang.org/pkg/crypto/tls/#pkg-constants for details.
[parent.tls] | certificatekey | "" | "/path/to/file" | file containing your ssl key
[parent.tls] | certificatefile | "" | "/path/to/file" | file containing your ssl certificate file
[parent.tls] | insecureskipverify | false | true/false | to to true to ignore insecure certificates, usable for self-signed certificates

## TLS Attributes
ACL attributes can adjust headers, cookies or allow/deny clients based on ip/headers

To adjust headers towards the client a rule should be applied on the `outboundacl`

To allow/deny clients based on headers/ip's a rule should be applied on the `inboundacl`

the ACL attribute should be an `Array` of acl's, you can add multiple.

Key | Option | Default | Values | Description
--- | --- | --- | --- | ---
[[parent.inboundacl]] |  |  |  | inbound ACL's are applied on requests from loadbalancer to the backend - needs to be an array of ACL's executed top to bottom
[[parent.outboundacl]] |  |  |  | outbound ACL's are applied on requests from loadbalancer to the client - needs to be an array of ACL's executed top to bottom
[[(in|out)boundacl]] | action | "" | see acl actions below | action to do when matching
[[(in|out)boundacl]] | headerkey | "" | string | key of header (ex. "Content-Type")
[[(in|out)boundacl]] | headervalue | "" | string | value of the header (ex. "UTF8")
[[(in|out)boundacl]] | cookiekey | "" | string | key of the cookie
[[(in|out)boundacl]] | cookievalue | "" | string | value of the cookie
[[(in|out)boundacl]] | cookiepath | "" | string | path of the cookie
[[(in|out)boundacl]] | cookieexpire | "" | datetime | expire date of the cookie
[[(in|out)boundacl]] | cookiehttponly |  | bool | httponly cookie
[[(in|out)boundacl]] | cookiesecure |  | bool | secure cookie
[[(in|out)boundacl]] | conditiontype | "" | string | header/cookie status	type to match with regex
[[(in|out)boundacl]] | conditionmatch | "" | string | regex string to match
[[(in|out)boundacl]] | statuscode |  | int | status code to return to the client (e.g. 500)
[[(in|out)boundacl]] | action | "" | string | action to do when matching
[[(in|out)boundacl]] | action | "" | string | action to do when matching
[[(in|out)boundacl]] | action | "" | string | action to do when matching
[[(in|out)boundacl]] | action | "" | string | action to do when matching




## ACL Actions
Action | ACL Type | Result
--- | --- | ---
Allow | Inbound	| will deny a client if non of the allowed rules matches the client header/ip
Deny | Inbound	| will deny a client if any of the deny rules matches the client header/ip
Add | Inbound/Outbound | Adds a header/cookie given match. Only if it does not exist.
Replace | Inbound/Outbound | Replaces a header/cookie/status code given match. Only if it exists.
Remove | Inbound/Outbound | Removes a header/cookie given match Only if it exists

## ACL special keys
The following special keys are translated in the ACL to a value.
All values are placed between 3 hashes(#) on both sides. for example: ###NODE_ID###

Key | Value
--- | ---
NODE_ID	| returns the uuid of the backend node
NODE_IP	| returns the ip of the backend node
LB_IP | returns the ip of the listener
REQ_URL | returns the requested host + path
REQ_PATH | returns the requested path
REQ_HOST | returns the requested host
REQ_IP | returns the ip of the requested host
CLIENT_IP	| returns the remote addr of the client
UUID | returns a random UUID


### Examples
* deny all clients which user-agent specifies Macintosh
```
[[loadbalancer.pools.INTERNAL_VIP_LB.inboundacls]]
action = "deny"
header_key = "User-Agent"
header_value = ".*Macintosh.*"
```

* add a location header, effectively redirecting the user to the ssl if this came in on a http connection (see ACL Special keys)
```
[[loadbalancer.pools.INTERNAL_VIP_REDIRECT.backends.redirect.outboundacls]]
action = "add"
header_key = "Location"
header_value = "https://###REQ_HOST######REQ_PATH###"
```

*	allow only the local networks specified
```
[[loadbalancer.pools.INTERNAL_VIP_LB.inboundacls]]
action = "allow"
cidrs = ["10.10.0.197/32", "10.10.0.197/32"]
```


Stickyness Loadbalancing ACL

To use Stickyness you Must apply the following ACL. this will ensure that the correct cookie gets set to direct the client to its sticky backend node

```
[[loadbalancer.pools.INTERNAL_VIP_LB.outboundacls]]
action = "add"
cookie_expire = "24h"
cookie_httponly = false
cookie_key = "stky"
cookie_secure = true
cookie_value = "###NODE_ID###"
```
should the client be directed to another node that its initial sticky cookie, because its unavailable, we need to make sure that this new node is the sticky node for all future requests.

we do this by overwriting the node id with the ID of the new node.
```
[[loadbalancer.pools.INTERNAL_VIP_LB.outboundacls]]
action = "replace"
cookie_expire = "24h"
cookie_httponly = false
cookie_key = "stky"
cookie_secure = true
cookie_value = "###NODE_ID###"
```
adds a stky cookie with the node_id the client is connected to


## ErrorPage Attributes

An error page is shown when an error is generated by Mercury, or if configured, when a 500 or higher error code is given by the backend application.

   ...['errorpage']['file']

	string	 	Path to html file to serve if an error is generated

   ...['errorpage']['triggerthreshold']

	int	500

threshold to show error page, if the backend application reply is >= this value, it will show the error page.

set this to 600 or higher if you do not want the loadbalancer to show an error page if the application generates a 500+ error message


BackendDNS attributes

This specifies the dns entry for a backend, this will point to the loadbalancer serving the backend.

the dns entry will be balanced based across the loadbalancers based on the backend balance type.
{	 	 	 

   hostname:

	string	 	specifies the host entry for the dns record (e.g. "www")

   domain:

	string	 	specifies the domain for the dns record (e.g. "example.com")

   ip:

	string	 	

specified the IP for the record. If omitted the IP of the Pool listener is used.

You should specify this if the IP is different then the IP where mercury is listening on
}	 	 	 


Balance attributes

Loadbalancing is done based on the balance attributes. this applies to both global (dns) as internal (proxy) loadbalancing
{	 	 	 

method:

	string	"leastconnected"	This determains the type of loadbalancing to apply (See `Loadbalancing Methods` below)

local_topology:

	array of string	 	list of cidr's that defines the local network (e.g. [ "127.0.0.1/32" ])

preference:

	int	 	preference value used for preference based loadbalancing

active_passive:

	"yes"|"no"	"no"	set to yes if this will only be up on 1 of the clusters - only affects monitoring

clusternodes:

	int	*calculated*	the ammount of cluster nodes serving this backend - only affects monitoring (used for backend that are only available on 1 of multiple loadbalancers)
}	 	 	 
Loadbalancing Methods

Loadbalancing Methods are applied in reverse order, meaning that the last entry is the first type of loadbalancing method beeing applied. the mechanism only orders the nodes, so the last method beeing applied (first entry) matters the most.

(tick) a loadbalance method of `topology, leastconnected` will first check the lease connected node, for all clients, and then check if any clients match the topology. forcing the clients that match the topology to this host.

(error) a loadbalance method of `leastconnected,topology` will first check the client to see if it matches a topology, and then check the least connected node, ignoring previously applied topology based loadbalancing
 eastconnected	balance based on current clients connected
leasttraffic	balance based on traffic generated
preference	balance based on preference set in node of backend (see preference attribute)
random	up to the rng gods
roundrobin	try to switch them a bit
sticky

balance based on sticky cookie

(warning) Important!: to apply sticky based loadbalancing you Must apply the `Stickyness Loadbalancing ACL` mentioned in the ACL Attribute section
topology

balance based on topology based networks

(info) Note that this topology will match the server making the dns request, which is your DNS Server, not the client. Ensure that your cliens use the DNS server of their topology for this to work
responsetime

Loadbalance based on server response time, in theory a less busy server responds quicker, or if you have servers with difference service offerings.

NOTE that this is a BETA Feature, and currently not suitable for production!
firstavailable	This limits the DNS records returned to 1

By default when balancing the available DNS records, all are returned. They are however ordered based on the loadbalancing methods above.

The following methods are an exception: `sticky`, `topology` and `firstavailable`. These methods will only return 1 record to ensure the client does not mistakenly connect to the second DNS record


HealthCheck attributes

Health checks will be fired on backend nodes to ensure they can server requests. It is highly recommended to have a functional test here.
{	 	 	 

   type:

	string	"tcpconnect"	See HealthCheck types for all available healthchecks

   tcprequest:

	string	 	the data to send to a tcp socket for testing

   tcpreply:

	string	 	the reply expected to a tcp socket for testing

   httprequest:

	string	 	the request sent to a webserver (e.g. "http://www.example.com/")

   httppostdata:

	string/special	 	post data sent to the host, see `Specials Keys` below for special parameters in the post string

   httpheaders:

	array of strings	 	headers sent with the http request (e.g. [ 'Accept: application/json' ])

   httpstatus:

	int	200	http status code expected from backend

   httpreply:

	string/regex	 	string/regex expected in http reply from backend

   pingpackets:

	int	4	number of ping packets to send (only when 100% packetloss this will be reported as down)

   pingtimeout:

	int	1	timeout in seconds for each ping request

   ip:

	string	backend ip	alternative IP to send request to

   port:

	int	backend port	alternative Port to send request to

   sourceip:

	string	listener ip	alternative source IP to use when sending request

   interval:

	int	10	how often to check this backend

   timeout:

	int	10	how long to wait for backend to finish its reply before reporting it in error state

   tls:

	see TLS attributes	 	

TLS settings for connecting to the backend. the only attribute that applies here is the `insecureskipverify` for when connecting to a node with a self-signed certificate

e.g. `{ insecureskipverify: true }`
}	 	 	 


HealthCheck types

The following types are available:
tcpconnect	does a simple tcp connect
tcpdata

connect to the host.

sends `tcprequest`

and expects `tcpreply` string to match the answer
httpget

performs a GET request on the backend using the `http*` attributes.

If `httpreply` is not provided only the `httpstatus` will be matched
httppost	same as `httpget`, only performs a POST instead of a GET
icmpping	does a icmpping for the amount of `pingpackets` and will report down if there is 100% packetloss
tcpping	does a tcpping for the amount of `pingpackets` and will report down if there is 100% packetloss
udpping	does a udpping for the amount of `pingpackets` and will report down if there is 100% packetloss
Special Keys

The following special keys are translated in the `httppostdata` to a value

All values are placed between 3 hashes(#) on both sides. for example: ###DATE###
DATE	returns todays date in system timezone
DATEUTC	returns todays date in UTC timezone
DATE+(number)[s|m|h]FORMAT	returns todays date + number (seconds/minutes/hours) in FORMAT - see https://golang.org/pkg/time/ for FORMAT options
DATE-(number)[s|m|h]FORMATUTC	returns todays date - number (seconds/minutes/hours) in UTC

example:

`httppostdata: '<date>###DATE+5m2006-01-02T15:04:05.000Z|UTC###</date>'` will  post this xml date with the value of todays date in the format 2006-01-02T15:04:05.000Z in UTC time and add +5 minutes.
Creating a Loadbalance Pool

A Loadbalance pool consists of a attributes defining a pool, and should contain a backend pool to work


Adding a Pool

node['sbp_mercury']['loadbalancer']['pool'][poolname]

	hash	 	poolname must be a string that defines the name of the loadbalancer pool

   ...['listener']

	hash	 	describes to where the pool should listen on, and how it should handle requests

   ...['listener']['ip']

	string	 	IP address where the Pool should listen on when using the internal loadbalancer

   ...['listener']['port']

	int	80	Port the pool should listen on for requests

   ...['listener']['mode']

	string	"http"	The protocol this listener should support. Available: "http", "https", "tcp"

   ...['listener']['tls']

	see TLS attributes	 	TLS settings for use with this listener (required for https)

   ...['listener']['httpproto']

	int	2	Set to 1 to enforce HTTP/1.1 instead of HTTP/2 http requests (required for websockets)

   ...['inboundacls']

	array of ACLs -> see ACL attributes	 	

Inbound ACLs are applied on incomming traffic from a client, before beeing sent to a backend server

ACLs on the listener are applied to all backends

   ...['outboundacls']

	array of ACLs -> see ACL attributes	 	

Outbound ACLs are applied on outgoing traffic from a webserver, before beeing sent to the customer

ACLs on the listener are applied to all backends

   ...['errorpage']

	see ErrorPage attributes	 	

Specifies a custom error page, to show if errors do occur.

When adding an error page to a pool, it applies to all backends

   ...['backends']

	see Backend attributes	 	Specifies the backends for a pool

   ...['healthchecks']



array of HealthChecks

see HealthChecks attributes
	 	a healtcheck put on a pool, will affect ALL backends of this vip (e.g. usefull for testing your internet connectivity)


Adding a Backend

A Pool can have multiple Backend only if the listening mode of the pool is `http` or `https`. for `tcp` there can be only 1 backend.

node['sbp_mercury']['loadbalancer']['pool'][poolname]['backend'][backendname]

	hash	 	

backend name must be a string that defines the name of the backend

    a http/https listener can serve multiple backends
    a tcp listener can only serve a single backend.

   ...['inboundacls']

	array of ACLs -> see ACL attributes	 	

Inbound ACLs are applied on incomming traffic from a client, before beeing sent to a backend server

ACLs on the backend are only applied to that backend

   ...['outboundacls']

	array of ACLs -> see ACL attributes	 	

Outbound ACLs are applied on outgoing traffic from a webserver, before beeing sent to the customer

ACLs on the backend are only applied to that backend

   ...['hostnames']

	array of strings	 	

List of hostnames this backend serves. the client is redirected to this backend base on the client request header.

This applies to http(s) only

   ...['dnsentry']

	see BackendDNS attributes	 	

Specifies which DNS entry to balance across this backend.

The DNS entry will point to the loadbalance that can serve requests to this backend

   ...['healthcheck']

	see HealthCheck attributes	 	

Applies to mercury <= 0.9.x

Healthcheck specifies what to check in order to determain if the backend is serving requests.

   ...['healthcheckmode']

	string "(all|any)"	"all"

Applies to mercury >= 0.10.x

Specifies wether all or only 1 check should succeed before the backend is marked as down

   ...['healthchecks']

	array of 1 or mutliple HealthChecks	 	

Applies to mercury >= 0.10.x

Healthcheck specifies what to check in order to determain if the backend is serving requests

see HealthCheck attributes

   ...['balance']

	see Balance attributes	 	Balance defines the balance modes for this backend.

   ...['nodes']

	array of nodes	 	

use search to automaticly add all nodes, or specify them using `host:` instead of `search:`

example: `[ { search: "recipe:yourwebapp", port: 443 } ]`

   ...['connectmode']

	string	"http"

how do we connect to the backend

available options are:

    http - for serving http requests to the backend node
    https - for serving https requests to the backend node
    tcp - for serving tcp requests to the backend node
    internal - for not sending a request to a backend but handle this internaly (see example on Http to Https redirect)



   ...['errorpage']

	see ErrorPage attributes	 	

Specifies a custom error page, to show if errors do occur.
When adding an error page to a backend, it applies to the specified backend only




Adding Static DNS Records

You can add static DNS entries to Mercury. You might want this if you want to loadbalance a your TLD domain. (example.org) instead balancing sub domains (www.example.org)

(warning)Note that if your using Mercury as DNS server, that we do not yet support DNSSEC (see gitlab: FXT/mercury/issues/12 (sbp.gitlab.schubergphilis.com))

the records contains a array of hashes with dns records

node['sbp_mercury']['dns']['domains'][domainname]['records'] = [{

	array of hashes	 	`domainname` is the domain you are serving the dns requests for

   name:

	string	 	host name of the dns record for the domain (e.g. "www")

   type:

	string	 	type of dns record (e.g. "A")

   target:

	string	 	target of the record (e.g. "1.2.3.4")

}]




Static DNS Records examples:

Below are some examples of common DNS records

{ name: "dns1", type: "A", target: "1.2.3.4" }

	 a simple A record for dns1 (in the previously defeined domain)

{ name: "dns1", type: "AAAA", target: "::1" }

	a simple ipv6 record pointing dns1 to a ipv6 ip

{ name: "", type: "NS", target: "dns1.example.com." }

	 a NS record for the domain to point to dns1.example.com

{ name: "", type: "SOA", target: "dns1.example.com. hostmaster.example.com. ###SERIAL### 3600 10 10" }

	 a SOA record for the domain

{ name: "", type: "MX", target: "20 mx1.example.com." }

	 a MX record for the domain to mx1.example.com with a preference of 20
