# DEPRECATED

[![Codacy Badge](https://api.codacy.com/project/badge/Grade/d2c4dff7ca2b4279a57e245d1059b6ff)](https://www.codacy.com/app/schubergphilis/mercury?utm_source=github.com&utm_medium=referral&utm_content=schubergphilis/mercury&utm_campaign=badger) [![CircleCI](https://circleci.com/gh/schubergphilis/mercury/tree/master.svg?style=shield&circle-token=86c89af895bb11c86e53256b9c1cca7c93d47c46)](https://circleci.com/gh/schubergphilis/mercury/tree/master) [![ReadTheDocs](https://readthedocs.org/projects/mercury-global-loadbalancer/badge/?version=latest)](http://mercury-global-loadbalancer.readthedocs.io/en/latest/) [![Apache-2.0](https://img.shields.io/badge/license-Apache--2.0-informational?style=flat)](https://github.com/schubergphilis/mercury/blob/master/LICENSE) [![GitHub-Downloads](https://img.shields.io/github/downloads/schubergphilis/mercury/total?color=informational)]()

# ![Mercury](https://github.com/schubergphilis/mercury/blob/master/web/images/mercury_logo_background_color_header.png "Mercury: global loadbalancer")

Mercury is an intelligent software load balancer which can distribution of traffic across server resources located in multiple geographies. The servers can be on premises in a company's own data centers, or hosted in a private cloud or the public cloud.

# Load balancing vs. Global Load balancing

Traditional load balancers such as HA Proxy or Nginx are great for balancing traffic at a single endpoint, to forward or redirect this traffic to the nodes behind this endpoint in the same region. Global load balancers such as F5 BigIP or Nginx Plus balance traffic using DNS techniques, which allow you to direct traffic to multiple regions. However these products are expensive and in some cases require specific hardware. With Mercury, you have the power of a global load balancer, and its totally free and open source!

![](https://github.com/schubergphilis/mercury/blob/master/docs/images/mercury_lb_readme.png)

Besides a global load balancer, you can also use Mercury as normal load balancer within your domain.

# Enhance your existing load balancers

Add GLB capabilities to your existing load balancers by letting Mercury take care of your DNS requests. Mercury will test the state of your existing load balancers, and redirect the clients to the best geographic region available. In this example Mercury serves the DNS requests, while the L4/L7 traffic is handled by another load balancer which does not support global load balancing.

![](https://github.com/schubergphilis/mercury/blob/master/docs/images/mercury_lb_readme_extlb.png)

This makes it easy to add GLB capabilities without changing any configuration in your existing load balancers.

# Resources

- Binaries: <https://github.com/schubergphilis/mercury/releases>
- Documentation: <http://mercury-global-loadbalancer.readthedocs.io/en/latest/>
- Chef Cookbook: <https://github.com/sbp-cookbooks/mercury>

# Requirements

- OSX
- Linux (with iproute 3+ when letting mercury manage your network interfaces, which is optional)

# Features

- Global Load balancing across multiple data center or Cloud infrastructures using DNS based load balancing
- L4 and L7 Load balancing support
- Web GUI for viewing/managing your host state
- Seamless configuration updates without interrupting connected clients (e.g. reload your configuration without your clients noticing)
- Does HealthChecks on local backends, and propagates their availability across other GLB instances

  - HTTP health checks (POST or GET)
  - TCP Connect checks (connects only)
  - TCP Data check (sends and/or expects data)
  - ICMP/UDP/TCP ping checks
  - None (always online)

- Is a functional DNS server which provides GLB based replies with

  - Topology based load balancing, with predefined networks
  - Preference based load balancing, for active/passive setup
  - Round robin based load balancing for the most balanced setup
  - LeastConnected based load balancing for the host with the least connections
  - LeastTraffic based load balancing for the host with the least traffic
  - Response time based load balancing for the host with the quickest response time (experimental)
  - Random based load balancing for when you can't choose
  - Sticky based load balancing for client sticky cookies

- Is a full load balancer using the supported balancing methods

- Script based rules on pre-inbound, inbound and outbound connection states (gorules)

- Supports automated Error / Maintenance pages

- Includes checks for Nagios/Sensu to be used

- Internal DNS server supports most record types

- HTTP/2 support
- Web-socket support
- AD web login integration

# Installing

## Linux

For RPM managed Linux ssytems, there is a release package available on GitHub release tab [here](https://github.com/schubergphilis/mercury/releases)

You can also compile the latest version your self with the following commands:

```
$ make linux-package
$ rpm -i builds/packages/mercury-${version}.rpm
```

For other distributions:

```
$ make linux

When make is finished, you can find the binary at: ./build/linux/mercury
```

## OSX

OSX has no package, but you can run the following to create the binary:

```
$ make osx
```

## Docker

Docker images are available, you can run a standalone mercury based on scratch The container does come with an example configuration you can start it with.

```
create a directory to place your mercury.toml and possible certificates (/home/user/mercury in the example below)

docker run -d -p 9001:9001 -p 80:80 -p 443:443 -v /home/user/mercury/:/etc/mercury/ rdoorn/mercury:latest
```

once its running you can connect to the web service of mercury on port 9001 with https (e.g. <https://localhost:9001>)

You can also create your own containers using the included Docker file or run:

- `make docker` for a scratch image of mercury (makes a mercury image based on scratch of approximately 19mb)
- `make docker-alpine` for an alpine image of mercury (makes a mercury-alpine image based on alpine of approximately 730mb)

you might want to use the docker-alpine image if you need to troubleshoot connections using curl or other tools

## Docker-composer

Included is a docker composer config which will start mercury with 2 web services running httpd

from the `docker` directory
```
docker-compose -f docker-compose.yml up
```

this will start 3 containers as an example, of mercury load balancing between the 2 apache webservers.

connect to the web service of mercury on port 9001 with https (e.g. <https://localhost:9001>)

# Configuration and Documentation

To configure Mercury please look at the documentation described at [ReadTheDocs:Mercury](https://mercury-global-loadbalancer.readthedocs.io/en/latest/configuration/)

[![](https://github.com/schubergphilis/mercury/blob/master/docs/images/mercury_read_the_docs.png)](https://mercury-global-loadbalancer.readthedocs.io/en/latest/configuration/)

You will find all possible configuration items documented here, including examples on a few specific use cases.

# TLS & HTTP/2

a Full list of supported TLS cyphers in the golang tls.Config package is [here](https://golang.org/pkg/crypto/tls/#pkg-constants)

The recommended cyphers are:

- `TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256` (tls 1.2 + HTTP/2)
- `TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384` (tls 1.2)
- `TLS_AES_256_GCM_SHA384` (tls 1.3)
- `TLS_AES_128_GCM_SHA256` (tls 1.3)
- `TLS_CHACHA20_POLY1305_SHA256` (tls 1.3)

This combined with the mercury default settings, will make the SSLLabs checks give you an A+ in regards to security on SSL enabled web sites

## HTTP/2.0

When enabling HTTP/2 you must include the following cypher as the _first_ cypher: `TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256` see [RFC](https://tools.ietf.org/html/rfc7540#section-9.2.2)

curve256 also has to be enabled: `CurveP256` (enabled by default)

# Web

You can browse to the web server within Mercury on the default url `http://localhost:9001` In the web interface you can view all the global (cluster wide) and local vips, health checks, status and dns entries served.

Screenshot: ![mercury web ui](https://github.com/schubergphilis/mercury/blob/master/docs/images/mercury_web_screenshot.png "Mercury Web UI")

Alternatively you can use curl to read the status

```
$ curl http://localhost:9001/backend
$ curl http://localhost:9001/glb
```

Ready to use for monitoring with nagios/sensu.

for json output pass the following option:

```
$ curl http://localhost:9001/backend -H 'Content-Type: application/json'
```

The web interface is mostly used for viewing the status of Mercury, however when you enable login, you can enable/disable backends if the correct credentials are provided.

:warning: Advice: please do _NOT_ expose the web interface to the public internet. The world wide web has no reason to view your load balancer status.

# Checks

There are a few checks which you can execute, and implement them in your monitoring system

Checking all of the Global Load balancing and cluster nodes in a single check

```
    $ mercury -config-file /etc/mercury/mercury.toml -check-glb
    OK: All checks are fine!
```

Checking all of the Backend nodes

```
    $ mercury -config-file /etc/mercury/mercury.toml -check-backend
```

You can seperate the checks using additional parameters

- Checking a specific pool/backend

  ```
    $ mercury -config-file /etc/mercury/mercury.toml -check-backend -pool-name example_https_443 -backend-name www_example_com
  ```

- Checking the Cluster nodes specificly

  ```
    $ mercury -config-file /etc/mercury/mercury.toml -check-glb -cluster-only
  ```

- Checking a specific glb entry

  ```
    $ mercury -config-file /etc/mercury/mercury.toml -check-glb -dns-name www.example.com
  ```

  Exitcodes are nagios/sensu compatible:

- All is fine

- Warning

- Critical

# Contributing

1. Clone this repository from GitHub:

  ```
  $ git clone git@github.com:schubergphilis/mercury.git
  ```

2. Create a git branch

  ```
  $ git checkout -b my_bug_fix
  ```

3. Install dependencies:

  ```
  $ make deps
  ```

4. Make your changes/patches/fixes, committing appropriately

5. **Write tests**

6. Run tests

  ```
  $ make test
  ```

# License & Authors

```
    - Author: Ronald Doorn (<rdoorn@schubergphilis.com>)
```

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
