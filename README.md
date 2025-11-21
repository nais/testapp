# testapp

## services

simple go binary that exposes the following services

- `/env` (prints all environment variables)
- `/ping(?delay=<duration>)` (returns "pong\n" and HTTP 200. Valid durations include 10s, 6m, 9h etc, and will delay the response accordingly)
- `/hostname` (prints hostname)
- `/connect` (performs a HTTP GET to the URL configured in `$CONNECT_URL` and prints the result. Ignores certs)
- `/loginfo` (logs "info log entry from testapp" with level `info`)
- `/logerror` (logs "error log entry from testapp" with level `error`)
- `/logdebug` (logs "debug log entry from testapp" with level `debug`)

## development

```bash
mise install        # Install dependencies
mise run local      # Run locally on 127.0.0.1:8080
mise run test       # Run tests with coverage (73.4%)
mise run check      # Run static checks and vulnerability scanning
mise run fmt        # Format code
mise run build      # Build binary
```

## binaries

the docker container has the following debugging binaries:

`nc`, `curl`, `dig`, `nmap`, `socat`, `bash`, `openssl`, `tcpdump`, `tcptraceroute`, `strace`, `iperf`, `telnet`

## options

```
      --bind-address string          ip:port where http requests are served (default ":8080")
      --connect-url string           URL to connect to with /connect (default "https://google.com")
      --ping-response string         what to respond when pinged (default "pong\n")
```
