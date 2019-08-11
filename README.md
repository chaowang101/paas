# Passwd as a Service

Package `chaowang_brain/paas` implements a minimal HTTP service that exposes the user and group information on a UNIX-like system that is usually locked away in the UNIX /etc/passwd and /etc/groups files.

This service is read-only but responses will reflect changes made to the underlying passwd and groups files while the service is running.

---

* [Install](#Install)
* [Usage](#Usage)
* [Unit Test](#unit-test)
* [Documentation](#Documentation)
* [REST API](#REST-API)

---

## Install

With a [correctly configured](https://golang.org/doc/install#testing) Go toolchain:

```sh
go get -u github.com/chaowang_brain/paas
```
To Compile `chaowang_brain/paas`:
```sh
go build
```
To Install `chaowang_brain/paas` under `$GOPATH/bin`:
```sh
go install
```


## Usage

To start the `paas` service:
```sh
./paas [-Config $configFile]
```
`paas` takes in an optional JSON configuration file in the following format:
```sh
{
  "ListenHost": "127.0.0.1", # Default value is empty, i.e. listening on all NICs.
  "Port": "4321", #Default value is 8080
  "WriteTimeoutInSec":4321, # timeout value for responding user's request. Default value is 30.
  "ReadTimeoutInSec":4321, # timeout value for reading user's request. Default value is 30.
  "IdleTimeoutInSec": 4321, # timeout value for closing idle connection. Default value is 60.
  "RestDomain": "127.0.0.1", # domain name for the REST serivce. Default valud is empty, i.e. respond to any domain name
  "LogFilePath": "./testData/log", # Default value is 30 stdout
  "PasswdFilePath": "./testData/passwd", # Default value is /etc/passwd
  "GroupFilePath": "./testData/group"# Default value is /etc/group
}
```
Without specify a configuration file, `paas` will start using default configuration.

## Unit Test
To run unit tests of `paas`:
```sh
go test ./...
```
Or run unit tests with coverage report:
```sh
go test ./... -cover
```
To run unit tests for each module, go into the module directory and run `go test`, for exmaple:
```sh
cd ./handler
go test
```

## Documentation
To see the docs of `paas`, run:
```sh
godoc -http=:6060
```
then see http://localhost:6060/pkg/github.com/chaowang101/paas

## REST API
`paas` provides the following REST APIs:
1. `GET /users`
Return a list of all users in the specified passwd file. Return 204 if no users are found.
Example response:
```sh
[
{“name”: “root”, “uid”: 0, “gid”: 0, “comment”: “root”, “home”: “/root”,“shell”: “/bin/bash”},
{“name”: “dwoodlins”, “uid”: 1001, “gid”: 1001, “comment”: “”, “home”:“/home/dwoodlins”, “shell”: “/bin/false”}
]
```

2. `GET /users/query[?name=<nq>][&uid=<uq>][&gid=<gq>][&comment=<cq>][&home=<hq>][&shell=<sq>]`
Return a list of users matching all of the specified query fields. Only exact matches need to be supported. Return 204 if no users are found.
Example response:
```sh
[
{“name”: “dwoodlins”, “uid”: 1001, “gid”: 1001, “comment”: “”, “home”:“/home/dwoodlins”, “shell”: “/bin/false”}
]
```

3. `GET /users/<uid>`
Return a single user with <uid>. Return 404 if <uid> is not found.
Example response:
```sh
{“name”: “dwoodlins”, “uid”: 1001, “gid”: 1001, “comment”: “”, “home”:“/home/dwoodlins”, “shell”: “/bin/false”}
```

4. `GET /users/<uid>/groups`
Return all the groups for a given user. Return 204 if no groups are found.
Example response:
```sh
[
{“name”: “docker”, “gid”: 1002, “members”: [“dwoodlins”]}
]
```

5. `GET /groups`
Return a list of all groups in the specified group file. Return 204 if no groups are found.
Example response:
```sh
[
{“name”: “_analyticsusers”, “gid”: 250, “members”:
[“_analyticsd’,”_networkd”,”_timed”]},
{“name”: “docker”, “gid”: 1002, “members”: []}
]
```

6. `GET /groups/query[?name=<nq>][&gid=<gq>][&member=<mq1>[&member=<mq2>][&...]]`
Return a list of groups matching all of the specified query fields. Any group containing all the specified members should be returned, i.e. when query members are a subset of group members. Return 204 if no groups are found.
Example response:
```sh
[
{“name”: “_analyticsusers”, “gid”: 250, “members”:[“_analyticsd’,”_networkd”,”_timed”]}
]
```

7. `GET /groups/<gid>`
Return a single group with <gid>. Return 404 if <gid> is not found.
Example response:
```sh
{“name”: “docker”, “gid”: 1002, “members”: [“dwoodlins”]}
```
