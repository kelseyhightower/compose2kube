# compose2kube

Convert docker-compose service files to Kubernetes objects.

## Status

compose2kube is in functional beta stage and supports mapping container images, varables, ports, labels, volumes, and restart policies to Kubernetes [replication controllers](https://github.com/kubernetes/kubernetes/blob/release-1.0/docs/user-guide/replication-controller.md) and [services](https://github.com/kubernetes/kubernetes/blob/release-1.0/docs/user-guide/services.md).
Thanks to the [docker/libcompose](https://github.com/docker/libcompose) library, compose2kube will support the complete docker-compose specification in the near future.

*Rancher support:* (optionally) compose2kube also reads `rancher-compose.yml` in order to get the information about scale and healthchecks of the containers.

## Set your GOPATH environment
For example:
```
export GOPATH=`pwd`/gopath
```

## Install dependencies
```
go get -v ./...
```

## Build
```
go build
```

## Usage

Create a `docker-compose.yml` file

```
web:
  image: nginx
  ports:
    - "80"
    - "443"
database:
  image: postgres
  ports:
    - "5432"
cache:
  image: memcached
  ports:
    - "11211"
```

Test the service using the docker-compose command:
```
docker-compose up -d
```

List the running services:
```
docker-compose ps
```

Stop the services:
```
docker-compose stop
```

Remove the services:
```
docker-compose rm
```

At this point the docker-compose.yml file is ready for conversion.

## docker-compose to Kubernetes

Use the compose2kube command to convert compose files to native Kubernetes objects.
By default, compose2kube will search for `docker-compose.yml` and `rancher-compose.yml` in the current directory. You can change that with `compose-file-path` option

```
$ compose2kube -output-dir output
```

```
output/cache-rc.yaml
output/cache-srv.yaml
output/database-rc.yaml
output/database-srv.yaml
output/web-rc.yaml
output/web-srv.yaml
```

### Launch the Kubernetes replication controllers

```
$ kubectl create -f output/
```

```
replicationcontrollers/cache
services/cache
replicationcontrollers/database
services/database
replicationcontrollers/web
services/web
```

List the replication controllers:

```
$ kubectl get rc
```

```
CONTROLLER   CONTAINER(S)   IMAGE(S)    SELECTOR           REPLICAS
cache        cache          memcached   service=cache      1
database     database       postgres    service=database   1
web          web            nginx       service=web        1
```

List the services:

```
kubectl get services
```

```
NAME       CLUSTER_IP     EXTERNAL_IP   PORT(S)          SELECTOR           AGE
cache      10.43.32.169   <none>        11211/TCP        service=cache      5m
database   10.43.32.170   <none>        5432/TCP         service=database   5m
web        10.43.32.171   <none>        80/TCP,443/TCP   service=web        5m
```

View the service pods:

```
$ kubectl get pods
```
```
NAME                             READY     STATUS    RESTARTS   AGE
cache-i3az8                      1/1       Running   0          6h
database-jq2lr                   1/1       Running   0          6h
kube-controller-172.16.238.141   4/4       Running   0          6h
web-1vj6h                        1/1       Running   0          6h
```

### Advanced Features

#### Environment Variables

Environment variables may be injected into the container.

```yaml
web:
  image: nginx
  ports:
    - "80"
    - "443"
  environment:
    - NGINX_HOST=example.com
```

#### Modifying the default command

The default command may be overwritten with the "command" option.

```yaml
web:
  image: nginx
  ports:
    - "80"
    - "443"
  command:
    - apt update
```

#### Host Volumes

For volumes, we currently only support mounting a host volume to a container.

The host volume is by default writable. The `:ro` option may be appended to
bind the volume as read only.

```yaml
web:
  image: nginx
  ports:
    - "80"
    - "443"
  volumes:
    - /srv/nginx/uploads:/usr/share/nginx/uploads # Writable
    - /srv/nginx/html:/usr/share/nginx/html:ro    # Read Only
```
