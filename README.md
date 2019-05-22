rdns-server
========

The rdns-server implements the API interface of Dynamic DNS, its goal is to use a variety of DNS servers such as Route53, CoreDNS etc...

| Default | Backend | Description |
| ------- | ------- | ----------- |
|    *    | Route53 | Store the records in the aws route53 service and copy them to the database |

## Building

`make`

## Running

#### Running Database
```shell
MYSQL_ROOT_PASSWORD="[password]" docker-compose -f deploy/mysql-compose.yaml up -d
MYSQL_ROOT_PASSWORD="[password]" database/migrate-up.sh
```

#### Running RDNS
```shell
AWS_HOSTED_ZONE_ID="[aws hosted zone ID]" AWS_ACCESS_KEY_ID="[aws access key ID]" AWS_SECRET_ACCESS_KEY="[aws secret access key]" DSN="[dsn]" docker-compose -f deploy/rdns-compose.yaml up -d
```

> DSN="[username[:password]@][tcp[(address)]]/rdns?parseTime=true"

## Usage
### Global Usage
```
NAME:
   rdns-server - control and configure RDNS('2019-06-06T06:47:02Z')

USAGE:
   rdns-server [global options] command [command options] [arguments...]

VERSION:
   v0.5.0

AUTHOR:
   Rancher Labs, Inc.

COMMANDS:
     route53, r53  use aws route53 backend

GLOBAL OPTIONS:
   --debug, -d     used to set debug mode. [$DEBUG]
   --listen value  used to set listen port. (default: ":9333") [$LISTEN]
   --frozen value  used to set the duration when the domain name can be used again. (default: "2160h") [$FROZEN]
   --version, -v   print the version
```

### Route53 Usage

```
NAME:
   rdns-server route53 - use aws route53 backend

USAGE:
   rdns-server route53 [command options] [arguments...]

OPTIONS:
   --aws_hosted_zone_id value     used to set aws hosted zone ID. [$AWS_HOSTED_ZONE_ID]
   --aws_access_key_id value      used to set aws access key ID. [$AWS_ACCESS_KEY_ID]
   --aws_secret_access_key value  used to set aws secret access key. [$AWS_SECRET_ACCESS_KEY]
   --database value               used to set database. (default: "mysql") [$DATABASE]
   --dsn value                    used to set database dsn. [$DSN]
   --ttl value                    used to set record ttl. (default: "240h") [$TTL]
```

## API References

| API | Method | Header | Payload | Description |
| --- | ------ | ------ | ------- | ----------- |
| /v1/domain | POST | **Content-Type:** application/json <br/><br/> **Accept:** application/json | {"hosts": ["4.4.4.4", "2.2.2.2"], "subdomain": {"sub1": ["9.9.9.9","4.4.4.4"], "sub2": ["5.5.5.5","6.6.6.6"]}} | Create A Records |
| /v1/domain/&lt;FQDN&gt; | GET | **Content-Type:** application/json <br/><br/> **Accept:** application/json <br/><br/> **Authorization:** Bearer &lt;Token&gt; | - | Get A Records |
| /v1/domain/&lt;FQDN&gt; | PUT | **Content-Type:** application/json <br/><br/> **Accept:** application/json <br/><br/> **Authorization:** Bearer &lt;Token&gt; | {"hosts": ["4.4.4.4", "3.3.3.3"], "subdomain": {"sub1": ["9.9.9.9","4.4.4.4"], "sub3": ["5.5.5.5","6.6.6.6"]}} | Update A Records |
| /v1/domain/&lt;FQDN&gt; | DELETE | **Content-Type:** application/json <br/><br/> **Accept:** application/json <br/><br/> **Authorization:** Bearer &lt;Token&gt; | - | Delete A Records |
| /v1/domain/&lt;FQDN&gt;/txt | POST | **Content-Type:** application/json <br/><br/> **Accept:** application/json <br/><br/> **Authorization:** Bearer &lt;Token&gt; | {"text": "xxxxxx"} | Create TXT Record |
| /v1/domain/&lt;FQDN&gt;/txt | GET | **Content-Type:** application/json <br/><br/> **Accept:** application/json <br/><br/> **Authorization:** Bearer &lt;Token&gt; | - | Get TXT Record |
| /v1/domain/&lt;FQDN&gt;/txt | PUT | **Content-Type:** application/json <br/><br/> **Accept:** application/json <br/><br/> **Authorization:** Bearer &lt;Token&gt; | {"text": "xxxxxxxxx"} | Update TXT Record |
| /v1/domain/&lt;FQDN&gt;/txt | DELETE | **Content-Type:** application/json <br/><br/> **Accept:** application/json <br/><br/> **Authorization:** Bearer &lt;Token&gt; | - | Delete TXT Record |
| /v1/domain/&lt;FQDN&gt;/renew | PUT | **Content-Type:** application/json <br/><br/> **Accept:** application/json <br/><br/> **Authorization:** Bearer &lt;Token&gt; | - | Renew Records |
| /metrics | GET | - | - | Prometheus metrics |

## License
Copyright (c) 2014-2017 [Rancher Labs, Inc.](http://rancher.com)

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

[http://www.apache.org/licenses/LICENSE-2.0](http://www.apache.org/licenses/LICENSE-2.0)

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
