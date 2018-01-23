# CHANGE LOG for Mercury

 This file is used to list changes made in each major version of Mercury.

## unreleased
Changes:
  * Adding Circle-Ci

## 0.10.1:
Changes:
  * You can now specify a topology per backend Node allowing you to do topology based loadbalancing on proxy level too (#60)

Bug:
  * Default TTL on all outgoing dns requests is now set to 10 seconds

## 0.10.0:
Feature:
  * Added Support for multiple healthchecks (#38)
  * Added Support for healtchecks on VIP - these would affect all backends of the vip (#58)
  * Added Support for ICMP/UDP/TCP pings (#59)

Changes:
  * Now a random time before first health check (max 5000ms) to spread the load on servers with many checks

Bugs:
  * Reload could cause errors/state to be incorrectly displayed on multiple nodes in the same backend (gui only)
  * Fix network dependency in mercury service for systemd

## 0.9.4:
Bugs:
   * Fix issues with OCSP stapling when using SNI certificates (#54)

## 0.9.3:
Changes:
  * Cluster config has changed to increase stability within the cluster - see readme for config changes (#41)
  * Graphing to collectd has been removed, splunk is the prefered way to go. code is still in place should we change our mind (#52)

Bugs:
  * Fix incorrect listener exit on update causing crash
  * Fix certificate loading order, since map is random - causing issues on reload
  * Add correct no-caching headers to sorry and mercury custom errors
  * Fix 0x20 case insensitive requests beeing handled according to https://tools.ietf.org/html/draft-vixie-dnsext-dns0x20-00 (#53)

## 0.9.2:
Bugs:
  * Fix incorrect listener exit on update causing crash
  * Fix certificate loading order, since map is random - causing issues on reload

## 0.9.2:
Feature:
  * Your now allowed to specify the amount of cluster nodes that serve a dns record (#48)

Bugs:
  * Fix loglevel not affected by reload (#51)
  * Fix monitoring message output to correctly show only the failing nodes on glb errors

## 0.9.1:
Feature:
  * Per Backend Error/Sorry page can now be specified

Bugs:
  * No longer send SRVFAIL on non-existing AAAA records, if a A record does exists.
  * Fixed possible index out of range issue in healtcheck on reload
  * Fixed crash when requesting a dns without a domain

## 0.9.0:
Changes:
  * IMPORTANT! Removed cross-connects - instead add multiple nodes to both backends for stickyness that supports proper failover
  * UUID's are now hash based, so they won't change up on restarts

Bugs:
  * Fix locking issue that could occur on dns updates (#42)
  * Fix possible dns pointer overwrite before cluster updates were sent

## 0.8.9:
Feature:
 * Add option to specify at which level to trigger sorry page, will always trigger on internal errors, but you can specify to trigger on 500+ or other result codes (#39)
 * Add OCSP Stapling support for SSL certificate verification (enabled by default for all https sites)
 * Add option to deny requests based on header match
 * Add option to allow/deny request based on CIDRS (#3)

## 0.8.8:
Feature:
 * Added firstavailable loadbalancing type. this returns only 1 host if multiple are availble. usable for compatibility reasons if needed.
 * Added option to use vip in active/passive setup - this is used by monitoring only: will alert if 0 or >1 nodes/pools are online

Bugs:
 * Correct alerting on offline GLB entries

## 0.8.7:
Changes:
 * Only setup a proxy if there is a listener IP, otherwise treat it as a dns balancer only

## 0.8.6:
Feature:
 * Now supports DNS forwarding for specified cidr's (#36)

Changes:
 * Allow resolving and serving of domain-only A and CNAME records

Bugs:
 * fix dead channels for configurations where the proxy function is disabled
 * fix dns authoritive and recursive answers in replies
 * fix locking issue on race condition during startup
 * fix crash that could occur on invalid dns request
 * correct dns reply return codes
 * fix content length issue on error page

## 0.8.5:

Changes:
 * Offline GLB pools now return all IP's instead of none, directing client to proper error instead of dns not found (#33)

Bugs:
 * fix incorrect domain name in dns result
 * fix backend duplication on reload with cross-connects with more than one node in a single backend (#32)

## 0.8.4:
Features:
 * Websocket support (note that you must force httpproto on the listener to 1, as websocket is not supported by http/2 which is enabled by default) (#30)
 * Better support for SOA records and serial updating

Bugs:
 * Stale node in proxy config if removed by reload, should no longer occur (#31)
 * Properly handle main and sub certificates and check them all during config loading
 * Fix web interface for local dns entries
 * Fix additional replies for dns entries

## 0.8.3:
Features:
 * Origin traffic is now the listener ip for both proxy and healthcheck
 * Better deal with timeouts in healthcheck

Bugs:
 * Fix healtcheck json unmarshal for duration for backend check
 * Fix healtcheck json unsupported type: chan bool
 * Fix healtcheck don't report internal vip's as down, they are always up
 * Fix sticky session if pointer no longer exists
 * Remove deadline timeout on tcp proxy

## 0.8.2:
Features:
 * DNS server now uses proxy statistics for loadbalancing algorithm when using internal loadbalancer (uses its own counter if not)

Bugs:
 * Fix Roundrobin statistics (#25)

## 0.8.1:
Features:
 * Change health check parameters to be more clear on check type
    * config file changes:
    * reply -> httpreply / tcpreply depending on check
    * request -> httprequest / tcprequest depending on check
    * postdata -> httppostdata
 * No longer are session ID's automaticly added (#26)
    * Requires config to add:
    * example: { action: 'add', cookie_key: 'mercid', cookie_value: '###UUID###', cookie_expire: '24h', cookie_secure: true, cookie_httponly: true }
 * Sticky session cookies are only parsed if we use sticky based loadbalancing (#27)
 * adding of cookies will now only add if cookie is not set yet

Bugs:
 * fix ACLs on self-generated responses (#24)

## 0.8.0:
Features:
 * HTTP/2 Support added for both client and backend (#19)
 * ResponseTime based Load-balancing added (#23)
 * Failed http requests to backend now return 500 Internal Server Error

Bugs:
 * DNS responses now are case insensitive (#18)
 * Fix client connected count (#10)

## 0.7.0:
Features:
 * Add sorry page ability (#4)
 * Added client session tracking (#13)

Bugs:
 * Reload now works for DNS Listener (#5)
 * Fix concurrency issues (#20, #21)

## 0.6.0:
 * Start of Change log
