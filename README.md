# compose2kube

Convert docker-compose service files to kubernetes objects.

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

```
docker-compose up -d
```

```
docker-compose stop
```

```
docker-compose rm
```

## Convert docker-compose services to Kubernetes pods

Use the compose2kube command to convert `docker-compose.yml` to native Kubernetes objects.

```
$ compose2kube -compose-file docker-compose.yml -output-dir output
```

```
tree output
```
```
output
├── cache-rc.yaml
├── database-rc.yaml
└── web-rc.yaml

0 directories, 3 files
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
