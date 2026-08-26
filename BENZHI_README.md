# iw2547_2426

基于 Go 实现的命令行项目，一款业务数据管理工具，围绕 SensorReading、AlertSummary、SnapshotBatch 等业务对象完成创建、校验、更新、查询与结果记录。

## Standard commands

```bash
go build ./...
go test -count=1 ./...
```

## Run

```bash
go run ./cmd/snapshotd
```

## Frontend

```bash
(cd web && npm run build)
```

## Docker validation

```bash
chmod +x build_benzhi_docker.sh
./build_benzhi_docker.sh my-go-task linux/arm64
./build_benzhi_docker.sh my-go-task linux/amd64
docker run -it my-go-task:latest
```

## Known initial failures

Initial validation failures are retained in the package command output and run logs; they are not copied into the project repository.
