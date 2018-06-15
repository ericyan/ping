# ping

Send ICMP ECHO_REQUEST in pure Go. `cmd/ping` implements a command-line
utility similar to good old `ping`. `cmd/pingd` is a Proemtheus exporter
that keep tracks of RTT and packet lost rate for multiple hosts.
