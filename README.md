# README

[![Build Status](https://github.com/ecri-org/go-redirector/workflows/branch/badge.svg)](https://github.com/ecri-org/go-redirector/actions?workflow=branch)
[![Coverage Status](https://coveralls.io/repos/github/ecri-org/go-redirector/badge.svg?branch=main)](https://coveralls.io/github/ecri-org/go-redirector?branch=main)
[![Go Report Card](https://goreportcard.com/badge/github.com/ecri-org/go-redirector)](https://goreportcard.com/report/github.com/ecri-org/go-redirector)


go-redirector aka "PlanetVegeta"

A reasonably fast (see perf data below) server that redirects users. It does this by offering a descriptive rendered html page with enough javascript which waits 15 seconds before redirecting the user to the correct URI.

All aspects of the html can be edited.
The server can contain multiple mapped entries of host:path -> destination.

Can be run as a docker container, and comes in at ~ 13MB in size.


## Versions

Versions:
  - `0.2.0`:
    - added new structure for each path entry, specifying `friendly` (bool, optional, default=true) which when false sends a direct 302, instead of a friendly page. See section _Mapping File_ below in docs.
    - swapped out logrus with zerolog 
  - `0.1.3`:
    - general improvements found through tests

## Mapping File

The mapping example below creates an entry for host `testhost`.
This host named `testhost` has two path entries.
  1. `/my-path` - a specific path
  2. `/` - presence of a root path `/` is the equivalent of specifying a wildcard. If you wish to exclude this path, then only matching paths (in this case `my-path`) will redirect, all others will return `404`.

Each mapping entry has two values which _MUST_ be set.
  1. `friendly`: (bool) true shows a friendly html page with a javascript redirect, false will have the client receive a 302 (proper for direct GET requests and where you don't want SEO resource link updates).
  2. `redirect`: (string) path starting with `/`. Can be explicitly `/` or `*` to denote being a wildcard. The author personally prefers `/`.

Preferred:
```yaml
---
mapping:
  testhost:
    "/my-path":
      friendly: true
      redirect: https://localhost:8081
    "/":
      friendly: true
      redirect: https://localhost:8082
```

Alternative Equivalent:
```yaml
---
mapping:
  testhost:
    "/my-path":
      friendly: true
      redirect: https://localhost:8081
    "*":
      friendly: true
      redirect: https://localhost:8082
```

## Devs

```shell
go mod tidy
go fmt ./...
golint ./...
golangci-lint run ./...
```

## Example Usage

1. Edit or supply (via bind mount) the map file: `redirect-map.yml` file.
1. Use the supplied template or provide your own: `/views/html.tpl`
1. Run the image

By default, this server starts in TLS mode, and listens on port 8443. You can change how the server operates with various flags. See the examples below on how.


### Examples

TLS Is enabled by default but no certs are provided.
Other flags which may be useful are:
  - `--cert <file>` defaults to `./certs/server.pem`
  - `--key <key>` defaults to `./certs/key.pem`
  - `--port <port-number>` defaults to `8443` unless specified
Below we bind the local directory containing `server.pem` and `server.key`.
This will allow the server to run.
```shell
docker run -it --rm \
  -p 8443:8443 \
  -v ./certs:/certs \
  go-redirector:0.1.0 \
  /go-redirector run
```

To run in HTTP mode you must supply the `-http` flag.
Other flags which may be useful are:
- `-port <port-number>` defaults to `8080` in `http` mode unless specified
```shell
docker run -it --rm \
  -p 8080:8080 \
  -v ./certs:/certs \
  go-redirector:0.1.0 \
  /go-redirector run --http
```

Version
```shell
docker run -it --rm -p 8080:8080 go-redirector:0.1.0 /entrypoint --version
```

Help
```shell
docker run -it --rm -p 8080:8080 go-redirector:0.1.0 /entrypoint --help
```

Run Help
```shell
docker run -it --rm -p 8080:8080 go-redirector:0.1.0 /entrypoint run --help
```


## Performance Data


### Fiber Implementation

TLDR: 32583.95 [#/sec] (mean)

The Fiber implementation based on fasthttp is slightly faster.
More importantly it is easier to implement.

Perf on a 2.9 GHz 6-Core Intel Core i9
```
│ Handlers ............. 9  Processes ........... 1 │ 
│ Prefork ....... Disabled  PID .............  xx   │ 
└───────────────────────────────────────────────────┘ 
```

```shell
$ ab -n 200000 -c 20 -k http://testhost:8080/

This is ApacheBench, Version 2.3 <$Revision: 1843412 $>
Copyright 1996 Adam Twiss, Zeus Technology Ltd, http://www.zeustech.net/
Licensed to The Apache Software Foundation, http://www.apache.org/

Benchmarking testhost (be patient)
Completed 20000 requests
Completed 40000 requests
Completed 60000 requests
Completed 80000 requests
Completed 100000 requests
Completed 120000 requests
Completed 140000 requests
Completed 160000 requests
Completed 180000 requests
Completed 200000 requests
Finished 200000 requests


Server Software:        PlanetVegeta
Server Hostname:        testhost
Server Port:            8080

Document Path:          /
Document Length:        826 bytes

Concurrency Level:      20
Time taken for tests:   6.138 seconds
Complete requests:      200000
Failed requests:        0
Keep-Alive requests:    200000
Total transferred:      197800000 bytes
HTML transferred:       165200000 bytes
Requests per second:    32583.95 [#/sec] (mean)
Time per request:       0.614 [ms] (mean)
Time per request:       0.031 [ms] (mean, across all concurrent requests)
Transfer rate:          31470.24 [Kbytes/sec] received

Connection Times (ms)
              min  mean[+/-sd] median   max
Connect:        0    0   0.0      0       1
Processing:     0    1   0.4      1      18
Waiting:        0    1   0.4      0      18
Total:          0    1   0.4      1      18
ERROR: The median and mean for the waiting time are more than twice the standard
       deviation apart. These results are NOT reliable.

Percentage of the requests served within a certain time (ms)
  50%      1
  66%      1
  75%      1
  80%      1
  90%      1
  95%      1
  98%      2
  99%      2
 100%     18 (longest request)
```


### Net/HTTP


#### Performance Mode

TLDR: 31187.52 [#/sec] (mean)

```shell
$ ab -n 200000 -c 20 -k http://testhost:8080/

This is ApacheBench, Version 2.3 <$Revision: 1843412 $>
Copyright 1996 Adam Twiss, Zeus Technology Ltd, http://www.zeustech.net/
Licensed to The Apache Software Foundation, http://www.apache.org/

Benchmarking testhost (be patient)
Completed 20000 requests
Completed 40000 requests
Completed 60000 requests
Completed 80000 requests
Completed 100000 requests
Completed 120000 requests
Completed 140000 requests
Completed 160000 requests
Completed 180000 requests
Completed 200000 requests
Finished 200000 requests


Server Software:        PlanetVegeta
Server Hostname:        testhost
Server Port:            8080

Document Path:          /
Document Length:        813 bytes

Concurrency Level:      20
Time taken for tests:   6.413 seconds
Complete requests:      200000
Failed requests:        0
Keep-Alive requests:    200000
Total transferred:      187800000 bytes
HTML transferred:       162600000 bytes
Requests per second:    31187.52 [#/sec] (mean)
Time per request:       0.641 [ms] (mean)
Time per request:       0.032 [ms] (mean, across all concurrent requests)
Transfer rate:          28598.71 [Kbytes/sec] received

Connection Times (ms)
              min  mean[+/-sd] median   max
Connect:        0    0   0.0      0       1
Processing:     0    1   0.8      0      16
Waiting:        0    1   0.8      0      16
Total:          0    1   0.8      0      16
WARNING: The median and mean for the processing time are not within a normal deviation
        These results are probably not that reliable.
WARNING: The median and mean for the waiting time are not within a normal deviation
        These results are probably not that reliable.
WARNING: The median and mean for the total time are not within a normal deviation
        These results are probably not that reliable.

Percentage of the requests served within a certain time (ms)
  50%      0
  66%      1
  75%      1
  80%      1
  90%      1
  95%      2
  98%      3
  99%      5
 100%     16 (longest request)
 ```


#### Safe HTML Mode

TLDR: 19407.44 [#/sec] (mean)

```shell
$ ab -n 200000 -c 20 -k http://testhost:8080/

This is ApacheBench, Version 2.3 <$Revision: 1843412 $>
Copyright 1996 Adam Twiss, Zeus Technology Ltd, http://www.zeustech.net/
Licensed to The Apache Software Foundation, http://www.apache.org/

Benchmarking testhost (be patient)
Completed 20000 requests
Completed 40000 requests
Completed 60000 requests
Completed 80000 requests
Completed 100000 requests
Completed 120000 requests
Completed 140000 requests
Completed 160000 requests
Completed 180000 requests
Completed 200000 requests
Finished 200000 requests


Server Software:        PlanetVegeta
Server Hostname:        localhost
Server Port:            8080

Document Path:          /
Document Length:        815 bytes

Concurrency Level:      20
Time taken for tests:   10.305 seconds
Complete requests:      200000
Failed requests:        0
Keep-Alive requests:    200000
Total transferred:      191200000 bytes
HTML transferred:       163000000 bytes
Requests per second:    19407.44 [#/sec] (mean)
Time per request:       1.031 [ms] (mean)
Time per request:       0.052 [ms] (mean, across all concurrent requests)
Transfer rate:          18118.67 [Kbytes/sec] received

Connection Times (ms)
              min  mean[+/-sd] median   max
Connect:        0    0   0.0      0       1
Processing:     0    1   1.3      1      21
Waiting:        0    1   1.3      1      21
Total:          0    1   1.3      1      21

Percentage of the requests served within a certain time (ms)
  50%      1
  66%      1
  75%      1
  80%      1
  90%      2
  95%      3
  98%      5
  99%      7
 100%     21 (longest request)
```
