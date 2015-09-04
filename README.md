# compose2kube

Convert docker-compose service files to Kubernetes objects.

## Status

compose2kube is in the prototype stage and only supports mapping container images, ports, and restart policies to Kubernetes replication controllers. Thanks to the [docker/libcompose](https://github.com/docker/libcompose) library, compose2kube will support the complete docker-compose specification in the near future.

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
