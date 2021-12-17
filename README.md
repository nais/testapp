# testapp

## services

simple go binary that exposes the following services

* `/env`  (prints all environment variables)
* `/ping(?delay=<duration>)` (returns "pong\n" and HTTP 200. Valid durations include 10s, 6m, 9h etc, and will delay the response accordingly)
* `/hostname` (prints hostname)
* `/metrics` (prints Prometheus metrics)) 
* `/version` (prints running version of testapp binary) 
* `/connect` (performs a HTTP GET to the URL configured in `$CONNECT_URL` and prints the result. Ignores certs)
* `/log` (logs "this is a log statement" with level `info`)
* `/logerror` (logs "this is a error log statement" with level `error`)
* `/logevent` (logs "this is a event log statement" with `logtype` set to `event`. Query parameters will be added as fields)
* `/database/test` (test that write/read works for database)
* `/bucket/test` (test that write/read works for bucket)
* `/ceph/test` (test that write/read works for ceph)
* `/bigquery/test` (test that write/read works for bigquery)

## binaries
the docker container has the following binaries

`nc`, `curl`, `dig`, `nmap`, `socat`, [hey](https://github.com/rakyll/hey), `vim`, `tcpdump`, `traceroute`, `strace`, `iperf`, `telnet`

## options
```
      --app-name string              application name (used when having several instances of application running in same namespace) (default "testapp")
      --bind-address string          ip:port where http requests are served (default ":8080")
      --bucket-name string           name of bucket used with /{read,write}bucket
      --bucket-object-name string    name of bucket object used with /{read,write}bucket (default "test")
      --connect-url string           URL to connect to with /connect (default "https://google.com")
      --db-hostname string           database hostname (default "localhost")
      --db-name string               database name (default "testapp")
      --db-password string           database password
      --db-user string               database username (default "testapp")
      --bigqueryName                 bigquery dataset name (default "bigqueryname")
	  --bigqueryTableName            bigquery table name (default "bigquerytablename")
      --graceful-shutdown-wait int   when receiving interrupt signal, it will wait this amount of seconds before shutting down server
      --ping-response string         what to respond when pinged (default "pong\n")

```
