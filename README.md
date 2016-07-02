# compose2kube

Convert docker-compose service files to Kubernetes objects.

## Status

compose2kube is in the prototype stage and only supports mapping container images, ports, and restart policies to Kubernetes [replication controllers](https://github.com/kubernetes/kubernetes/blob/release-1.0/docs/user-guide/replication-controller.md). Thanks to the [docker/libcompose](https://github.com/docker/libcompose) library, compose2kube will support the complete docker-compose specification in the near future.

## Build

```
go build .
```

## Usage

Create a docker-compose.yml file

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

Use the compose2kube command to convert `docker-compose.yml` to native Kubernetes objects.

```
$ compose2kube -compose-file docker-compose.yml -output-dir output
```

```
output/cache-rc.yaml
output/database-rc.yaml
output/web-rc.yaml
```

### Launch the Kubernetes replication controllers

```
$ kubectl create -f output/
```

```
replicationcontrollers/cache
replicationcontrollers/database
replicationcontrollers/web
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
