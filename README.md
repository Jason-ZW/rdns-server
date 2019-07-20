rdns-server
===========

[![Build Status](https://drone-publish.rancher.io/api/badges/rancher/rdns-server/status.svg)](https://drone-publish.rancher.io/rancher/rdns-server)
[![Go Report Card](https://goreportcard.com/badge/github.com/rancher/rdns-server)](https://goreportcard.com/report/github.com/rancher/rdns-server)
[![GoDoc](https://godoc.org/github.com/rancher/rdns-server?status.svg)](http://godoc.org/github.com/rancher/rdns-server)

rdns-server is designed to work with a variety of currently popular DNS providers (e.g. AWS Route 53 or CoreDNS).
rdns-server allows you to control DNS records via [API](documents/apis.md) in a DNS provider-agnostic way.

rdns-server is integrated by:
- [rancher/rio](https://github.com/rancher/rio)
- [kubernetes-incubator/external-dns](https://github.com/kubernetes-incubator/external-dns)

## Providers
- AWS Route53
- CoreDNS

## Build
Install [docker](https://docs.docker.com/install/linux/docker-ce/ubuntu/) on your build machine.

```
make
```

## License
Copyright (c) 2014-2019 [Rancher Labs, Inc.](http://rancher.com)

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

[http://www.apache.org/licenses/LICENSE-2.0](http://www.apache.org/licenses/LICENSE-2.0)

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
