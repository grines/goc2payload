# goc2payload

Payloads for goc2

# Option 1
go get github.com/grines/goc2payload  ** go gettable  | c2 server can be configured using env variable | export goc2server=server.com

# Option 2
go build ** make sure to update the c2 server in main

# Option 3
sh build.sh c2.server.com ** replaces c2 string and issues build command. creates payload in /tmp/bin

# MacOs
- [X] Utilizing chrome dev tools to steal coookies over WSS
- [X] backdooring electron apps (appending node code to js app.on) * persistence
- [X] Opsec safe objective c in cgo
- [ ] Pre commit hook persistence
