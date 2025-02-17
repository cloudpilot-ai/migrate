# Migrate

## How to run

```shell
go run migrate --kubeconfig ~/.kube/config    
```

## How to build

```
go build  migrate

docker run --rm -v $(pwd):/app -w /app golang:latest \
   bash -c 'GOOS=linux GOARCH=amd64 go build -o migrate_linux_amd64'
```
