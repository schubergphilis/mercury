[![CircleCI](https://circleci.com/gh/schubergphilis/mercury/tree/master.svg?style=svg&circle-token=86c89af895bb11c86e53256b9c1cca7c93d47c46)](https://circleci.com/gh/schubergphilis/mercury/tree/master)

# Mercury
Mercury is a loadbalancer with Global Loadbalance capababilities across multiple Datacentre or Cloud infrastructures.

## What is Mercury ?
Mercury is a Global loadbalancer, designed to add a dns based loadbalancing layer on top of its internal loadbalancer or 3rd pary loadbalancers such as cloud services.
This makes mercury able to loadbalance across multiple cloud environments using dns, while keeping existing cloud loadbancer sollutions in place.

## Resources

* Binaries: https://github.com/schubergphilis/mercury/releases
* Chef Cookbook: https://github.com/sbp-cookbooks/mercury
* Additional Documentation: http://mercury-global-loadbalancer.readthedocs.io/en/latest/

# Requirements
* OSX
* Linux (with iproute 3+)

# Features
* Global Loadbalacing across multiple datacenters or Cloud infrastructures using DNS based loadbalancing
* Does HealthChecks on local backends, and propegates their availability across other GLB instances
  * HTTP health checks (POST or GET)
  * TCP Connect checks (connects only)
  * TCP Data check (sends and/or expects data)
  * ICMP/UDP/TCP ping checks
  * None (always online)
* Is a functional DNS host to give GLB based replies with
  * Topology based loadbalancing, with predefined networks
  * Preference based loadbalancing, for active/passive setup
  * Roundrobin based loadbalancing for the most balanced setup
  * LeastConnected based loadbalancing for the host with the least connections
  * LeastTraffic based loadbalancing for the host with the least traffic
  * Responsetime based loadbalancing for the host with the quickest responsetime
  * Random based loadbalancing for when you can't choose
  * Sticky based loadbalancing for client sticky cookies
* Is a full loadbalancer using the supported balancing methods
* Includes checks for Nagios/Sensu to be used
* Internal DNS server supports most record types
* HTTP/2 support
* Websocket support

## Installing
### Linux
For Linux we can make a RHEL/Centos RPM package. to do so run the following:

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
