# Deskbook 共享工位预约服务

Deskbook 是一个从零实现的 Go 内存预约服务示例，支持创建、确认、取消、转让、改期、删除和多维查询预约。

## 行为约束

- 同一工位的预约时间不能重叠，相邻时间段允许首尾衔接。
- 存储层不向调用方泄露内部可变对象。
- 已取消的 `context.Context` 不得继续执行状态变更。
- 缺失记录返回明确错误，不发生 panic。
- 所有共享状态访问都受读写锁保护。
- 重复确认已确认预约保持幂等。
- 转让、改期和删除在产生副作用前尊重已取消的 context。
- 按工位和预约人查询均返回稳定排序的对象副本。

## 验证

```bash
go test ./... -count=1
go test -race ./... -count=1
go vet ./...
go build ./...
```

Docker 构建：

```bash
DOCKER_PLATFORM=linux/amd64 ./build_docker.sh deskbook:amd64
DOCKER_PLATFORM=linux/arm64 ./build_docker.sh deskbook:arm64
```
