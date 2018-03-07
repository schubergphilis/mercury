[![Codacy Badge](https://api.codacy.com/project/badge/Grade/d2c4dff7ca2b4279a57e245d1059b6ff)](https://www.codacy.com/app/schubergphilis/mercury?utm_source=github.com&utm_medium=referral&utm_content=schubergphilis/mercury&utm_campaign=badger)
[![CircleCI](https://circleci.com/gh/schubergphilis/mercury/tree/master.svg?style=shield&circle-token=86c89af895bb11c86e53256b9c1cca7c93d47c46)](https://circleci.com/gh/schubergphilis/mercury/tree/master)
[![ReadTheDocs](https://readthedocs.org/projects/mercury-global-loadbalancer/badge/?version=latest)](http://mercury-global-loadbalancer.readthedocs.io/en/latest/)

# Mercury
Mercury is a load balancer with Global Load balance capabilities across multiple Datacenter or Cloud infrastructures.

## What is Mercury ?
Mercury is a Global load balancer, designed to add a dns based load balancing layer on top of its internal load balancer or 3rd party load balancers such as cloud services.
This makes mercury able to load balance across multiple cloud environments using dns, while keeping existing cloud load balancer solutions in place.

## Resources

* Binaries: https://github.com/schubergphilis/mercury/releases
* Chef Cookbook: https://github.com/sbp-cookbooks/mercury
* Additional Documentation: http://mercury-global-loadbalancer.readthedocs.io/en/latest/

# Requirements
* OSX
* Linux (with iproute 3+)

# Features
* Global Load balancing across multiple datacenter or Cloud infrastructures using DNS based load balancing
* Web GUI for viewing/managing your host state
* Seamless configuration updates without interrupting connected clients
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
* Includes checks for Nagios / Sensu to be used
* Internal DNS server supports most record types
* HTTP/2 support
* Web-socket support

## Installing
### Linux
For Linux we can make a RHEL / Centos RPM package. to do so run the following:

    $ make linux-package
    $ rpm -i builds/packages/mercury-${version}.rpm

For other distributions:

    $ make linux

### OSX
OSX has no package, but you can run the following to create the binary:

    $ make osx

## Documentation

Documentation is are available at [here](https://github.com/schubergphilis/mercury/tree/master/docs)

Examples configuration files are available at [here](https://github.com/schubergphilis/mercury/tree/master/examples)

## TLS & HTTP/2

a Full list of supported TLS cyphers in the golang tls.Config package is [here](https://golang.org/pkg/crypto/tls/#pkg-constants)

The recommended cypers are:

Required for HTTP/2 is `TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256` see [RFC](https://tools.ietf.org/html/rfc7540#section-9.2.2)
```Notice
Note that this has to be the first cipher in the list!
```

HTTP/2 also requires CurveP256 to be suported

## Web
You can browse to the webserver within Mercury on the default url `http://localhost:9001`

Alternatively you can use curl to read the status

    $ curl http://localhost:9001/backend
    $ curl http://localhost:9001/glb

for json output pass the following option:

    $ curl http://localhost:9001/backend -H 'Content-Type: application/json'

## Checks
There are 2 checks which you can execute, and implement them in your monitoring system

Checking the Global Loadbalancing

    $ mercury -config-file /etc/mercury/mercury.toml -check-glb

Checking the Backend nodes

    $ mercury -config-file /etc/mercury/mercury.toml -check-backend

## Contributing

1. Clone this repository from GitHub:

        $ git clone git@github.com:schubergphilis/mercury.git

2. Create a git branch

        $ git checkout -b my_bug_fix

3. Install dependencies:

        $ make get

4. Make your changes/patches/fixes, committing appropiately
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
