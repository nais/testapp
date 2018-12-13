## testapp

simple go binary that exposes the following services

/env  (prints all environment variables)
/ping (returns "pong\n" and HTTP 200)
/hostname (prints hostname)
/metrics (prints Prometheus metrics)) 
/connect (performs a HTTP GET to the URL configured in `$CONNECT_URL` and prints the result. Ignores certs)

the docker container has the following binaries

`nc`, `curl`, `dig`, `nmap`, `socat`, `[hey](https://github.com/rakyll/hey)`, `vim`
