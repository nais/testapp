# testapp

## services

simple go binary that exposes the following services

* `/env`  (prints all environment variables)
* `/ping` (returns "pong\n" and HTTP 200)
* `/hostname` (prints hostname)
* `/metrics` (prints Prometheus metrics)) 
* `/version` (prints running version of testapp binary) 
* `/connect` (performs a HTTP GET to the URL configured in `$CONNECT_URL` and prints the result. Ignores certs)
* `/log` (logs "this is a log statement" with level `info`)
* `/logerror` (logs "this is a error log statement" with level `error`)
* `/writebucket` (writes request payload to configured bucket (HTTP POST))
* `/readbucket` (reads and outputs current bucket content)
* `/logerror` (logs "this is a error log statement" with level `error`)

## binaries
the docker container has the following binaries

`nc`, `curl`, `dig`, `nmap`, `socat`, [hey](https://github.com/rakyll/hey), `vim`, `tcpdump`, `traceroute`, `strace`, `iperf`, `telnet`

## options
```
Usage:
      --bind-address string                       ip:port where http requests are served (default ":8080")
      --bucket-name string                        name of bucket used with /{read,write}bucket
      --bucket-object-name string                 name of bucket object used with /{read,write}bucket (default "test")
      --connect-url string                        URL to connect to with /connect (default "https://google.com")
      --graceful-shutdown-wait int                when receiving interrupt signal, it will wait this amount of seconds before shutting down server
      --ping-response string                      what to respond when pinged (default "pong\n")
      --service-account-credentials-file string   path to service account credentials file (default "/var/run/secrets/testapp-serviceaccount.json")

```
