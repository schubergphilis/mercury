# Welcome to Mercury!

Mercury is a Global Load balancer application designed to add a dns based load balancing layer on top of its internal load balancer or 3rd party load balancers such as cloud services. This makes mercury able to load balance across multiple cloud environments using dns, while keeping existing cloud load balancing solutions in place

# What does it do?
Traditional load balancers work great if you have 1 location where all your servers run in. However for redundancy purposes you will want to have your servers spread across multiple locations. This includes your Load balancer. But how do you ensure that clients can connect to the best performing location, or keep their session sticky to a specific location? Global Load balancing is the answer. In short it acts as a DNS service, and based on the state of your service it will redirect you to the location best suited for your request. This can be an existing load balancer at your datacenter or even a cloud solution such as AWS or Azure. Allowing you to balance between multiple locations based on health checks of their local services.
Optionally you could replace your load balancer with the build-in load balancer provided by Mercury. Removing the need of additional Local load balancers.

# Sources
You can find the mercury Source code at Github:
[https://github.com/schubergphilis/mercury](https://github.com/schubergphilis/mercury)

Pre-compiled binaries are also available at Github:
[https://github.com/schubergphilis/mercury/releases](https://github.com/schubergphilis/mercury/releases)

Automated deployment and configuration using Chef using the Chef Cookbook:
[https://github.com/sbp-cookbooks/mercury](https://github.com/sbp-cookbooks/mercury)
