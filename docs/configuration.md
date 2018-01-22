# Mercury Configuration

Mercury uses a [toml](https://github.com/toml-lang/toml) configuration file.
this documentation will go through each section to explain the configurable items of each section

## Global Settings
Settings are defined in the `[settings]` block.
options are:

Option | Default | Values | Description
--- | --- | --- | ---
manage_network_interfaces | "yes" | "yes"/"no" | when enabled this will add virtual ip's to your network interfaces if needed. this is the IP address defined on your pool listener.
enable_proxy              | "yes" | "yes"/"no" | when enabled this will act as a reverse proxy. Mercury will start listening on the pool listener to accept connections and forward these to the specified backend.

##
```
[cluster]
  binding = "localhost"
  name = "MY_GLB_POOL"
  nodes = ["localhost", "remotehost"]
  secretkey = "yourclusterkey"
```
