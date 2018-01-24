# Welcome to Mercury!

Mercury is a Global Loadbalancer application designed to add a dns based loadbalancing layer on top of its internal loadbalancer or 3rd pary loadbalancers such as cloud services. This makes mercury able to loadbalance across multiple cloud environments using dns, while keeping existing cloud loadbancer solutions in place

# What does it do?
Traditional loadbalancers work great if you have 1 location where all your servers run in. However for redundancy purposes you will want to have your servers spread across multiple locations. So also your Loadbalancer. But how do you ensure that clients can connect to the best performing location, or keep their session sticky to a specific location? Global Loadbalancing is the answer. In short it acts as a DNS service, and based on the state of your service it will redirect you to the location best suited for your request. This can be an existing loadbalancer at your data-center or even a cloud solution such as AWS or Azure. Allowing you to balance between multiple locations based on health checks of their local services.
Optionally you could replace your loadbalancer with the build-in loadbalancer provided by Mercury. Removing the need of additional Local loadbalancers.

# Sources
You can find the mercury Source code at Github:
[https://github.com/schubergphilis/mercury](https://github.com/schubergphilis/mercury)

Pre-compiled binaries are also available at Github:
[https://github.com/schubergphilis/mercury/releases](https://github.com/schubergphilis/mercury/releases)

Automated deployment and configuration using Chef using the Chef Cookbook:
[https://github.com/sbp-cookbooks/mercury](https://github.com/sbp-cookbooks/mercury)
