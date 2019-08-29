[![Codacy Badge](https://api.codacy.com/project/badge/Grade/d2c4dff7ca2b4279a57e245d1059b6ff)](https://www.codacy.com/app/schubergphilis/mercury?utm_source=github.com&utm_medium=referral&utm_content=schubergphilis/mercury&utm_campaign=badger)
[![CircleCI](https://circleci.com/gh/schubergphilis/mercury/tree/master.svg?style=shield&circle-token=86c89af895bb11c86e53256b9c1cca7c93d47c46)](https://circleci.com/gh/schubergphilis/mercury/tree/master)
[![ReadTheDocs](https://readthedocs.org/projects/mercury-global-loadbalancer/badge/?version=latest)](http://mercury-global-loadbalancer.readthedocs.io/en/latest/)
[![Apache 2.0](https://img.shields.io/badge/license-Apache--2.0-success?style=flat)](https://github.com/schubergphilis/mercury/blob/master/LICENSE)
[![Github All Releases](https://img.shields.io/github/downloads/schubergphilis/mercury/total.svg)()

![Mercury](https://github.com/schubergphilis/mercury/blob/master/web/images/mercury_logo_background_color_header.png "Mercury: global loadbalancer")
---
Mercury is an intelligent software loadbalancer which can distribution of traffic across server resources located in multiple geographies.
The servers can be on premises in a company’s own data centers, or hosted in a private cloud or the public cloud.

## Loadbalancing vs Global Loadbalancing
Traditional Loadbalancers such as HA Proxy or Nginx are great for balancing traffic at a single endpoint, to forward or redirect this traffic to the nodes behind this endpoint in the same region.
Globabl Loadbalancers such as F5 BigIP or Nginx Plus balance traffic using DNS techniques, which allow you to direct traffic to multiple regions. However these products are expensive and in some cases require specific hardware.
With Mercury you have the power of a Global Loadbalancer, and its totally free and opensource!

![loadbalancing example](https://github.com/schubergphilis/mercury/blob/master/docs/images/mercury_lb_readme.png "Global Loadbalancing")
Besides a Global loadbalancer, you can also user Mercury as normal Loadbalancer within your domain

## Resources

* Binaries: https://github.com/schubergphilis/mercury/releases
* Documentation: http://mercury-global-loadbalancer.readthedocs.io/en/latest/
* Chef Cookbook: https://github.com/sbp-cookbooks/mercury

## Requirements
* OSX
* Linux (with iproute 3+)

## Features
* Global Load balancing across multiple datacenter or Cloud infrastructures using DNS based load balancing
* L4 and L7 Loadbalancing support
* Web GUI for viewing/managing your host state
* Seamless configuration updates without interrupting connected clients (e.g. reload your configuration without your clients noticing)
* Does HealthChecks on local backends, and propagates their availability across other GLB instances
  * HTTP health checks (POST or GET)
  * TCP Connect checks (connects only)
  * TCP Data check (sends and/or expects data)
  * ICMP/UDP/TCP ping checks
  * None (always online)
* Is a functional DNS server which provides GLB based replies with
  * Topology based load balancing, with predefined networks
  * Preference based load balancing, for active/passive setup
  * Round robin based load balancing for the most balanced setup
  * LeastConnected based load balancing for the host with the least connections
  * LeastTraffic based load balancing for the host with the least traffic
  * Response time based load balancing for the host with the quickest response time
  * Random based load balancing for when you can't choose
  * Sticky based load balancing for client sticky cookies
* Is a full load balancer using the supported balancing methods
* Supports automated Error / Maintenance pages
* Includes checks for Nagios/Sensu to be used
* Internal DNS server supports most record types
* HTTP/2 support
* Web-socket support
* AD web login integration

## Installing
### Linux
For Linux we can make a RHEL/Centos RPM package. To do so run the following:

    $ make linux-package
    $ rpm -i builds/packages/mercury-${version}.rpm

For other distributions:

    $ make linux

### OSX
OSX has no package, but you can run the following to create the binary:

    $ make osx

## Configuration and Documentation
To configure Mercury please look at the example configurations and the documentation below:

Documentation is are available at [here](https://github.com/schubergphilis/mercury/tree/master/docs)

Examples configuration files are available at [here](https://github.com/schubergphilis/mercury/tree/master/examples)

## TLS & HTTP/2

a Full list of supported TLS cyphers in the golang tls.Config package is [here](https://golang.org/pkg/crypto/tls/#pkg-constants)

The recommended cyphers are:
* `TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256` (tls 1.2 + HTTP/2)
* `TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384` (tls 1.2)
* `TLS_AES_256_GCM_SHA384` (tls 1.3)
* `TLS_AES_128_GCM_SHA256` (tls 1.3)
* `TLS_CHACHA20_POLY1305_SHA256` (tls 1.3)

This combined with the mercury default settings, will make the SSLLabs checks give you an A+ in regards to security on SSL enabled web sites

Required for HTTP/2 is `TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256` see [RFC](https://tools.ietf.org/html/rfc7540#section-9.2.2)
```Notice
Note that this has to be the first cipher in the list!
```

HTTP/2 also requires CurveP256 to be supported

## Web
You can browse to the web server within Mercury on the default url `http://localhost:9001`
In the web interface you can view all the global (cluster wide) and local vips, health checks, status and dns entries served.

Screenshot:
![mercury web ui](https://github.com/schubergphilis/mercury/blob/master/docs/images/mercury_web_screenshot.png "Mercury Web UI")

Alternatively you can use curl to read the status

    $ curl http://localhost:9001/backend
    $ curl http://localhost:9001/glb

Ready to use for monitoring with nagios/sensu.

for json output pass the following option:

    $ curl http://localhost:9001/backend -H 'Content-Type: application/json'

## Checks
There are 2 checks which you can execute, and implement them in your monitoring system

Checking the Global Load balancing

```
    $ mercury -config-file /etc/mercury/mercury.toml -check-glb
    OK: All checks are fine!
```

Checking the Backend nodes

    $ mercury -config-file /etc/mercury/mercury.toml -check-backend

Exitcodes are nagios/sensu compatible:
0. All is fine
1. Warning
2. Critical

## Contributing

1. Clone this repository from GitHub:

        $ git clone git@github.com:schubergphilis/mercury.git

2. Create a git branch

        $ git checkout -b my_bug_fix

3. Install dependencies:

        $ make deps

4. Make your changes/patches/fixes, committing appropriately
5. **Write tests**
6. Run tests

        $ make test

# License & Authors
        - Author: Ronald Doorn (<rdoorn@schubergphilis.com>)

```text
Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
```
