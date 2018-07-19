# pingd

pingd is a Prometheus exporter that keep tracks of RTT and packet lost
rate for multiple hosts. It is the modern replacement for Smokeping.

## Docker image

To build the Docker image from source:

```
sudo docker build -t ericyan/pingd .
```

To use the container:

```
sudo docker run -d --restart=always --name=pingd \
  --net=host --volume=/path/to/dst.list:/dst.list \
  ericyan/pingd /pingd -v
```
