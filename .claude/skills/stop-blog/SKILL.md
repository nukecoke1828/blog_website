---
name: stop-blog
description: 停止博客项目并清空容器中的所有数据 —— 删除容器与数据卷（MySQL、Redis、Kafka、etcd 数据全部清除）。
---

# 停止博客项目并清空数据

停止并删除博客项目的全部容器，同时删除数据卷以清空 MySQL、Redis、Kafka、etcd 中的所有持久化数据。

## 需要执行的命令

```bash
docker-compose down -v
```

## 说明

- `down` 停止并删除该项目定义的所有容器与网络。
- `-v` 额外删除挂载的数据卷（`mysql-data`、`etcd-data`、`kafka-data`），即清空数据库、缓存、消息队列等全部持久化数据。
- ⚠️ 执行后所有数据将无法恢复，请谨慎操作。
