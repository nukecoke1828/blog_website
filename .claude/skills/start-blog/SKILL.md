---
name: start-blog
description: 启动博客项目 —— 自动检查并补全缺失的环境变量，然后使用 Docker Compose 一键启动 Go 应用、MySQL、Redis、Kafka、etcd 全部服务。
---

# 启动博客项目

使用 Docker Compose 启动博客项目及其全部依赖服务（Go 应用、MySQL、Redis、Kafka、etcd）。启动前会自动检查并补全缺失的关键环境变量。

## 前置条件

- 已安装 Docker 与 Docker Compose
- 在项目根目录执行以下命令

## 需要执行的步骤与命令

### 1. 确保 `.env` 文件存在

若项目根目录缺少 `.env`，先复制示例文件：

```bash
[ -f .env ] || cp .env.example .env
```

### 2. 检查并补全关键环境变量

检查 `JWT_ACCESS_SECRET`、`JWT_REFRESH_SECRET`、`ADMIN_PASSWORD` 三个环境变量是否已存在于 `.env` 中，若不存在则自动生成/设置：

```bash
grep -q '^JWT_ACCESS_SECRET=' .env || echo "JWT_ACCESS_SECRET=$(openssl rand -base64 32)" >> .env
grep -q '^JWT_REFRESH_SECRET=' .env || echo "JWT_REFRESH_SECRET=$(openssl rand -base64 32)" >> .env
grep -q '^ADMIN_PASSWORD=' .env || echo "ADMIN_PASSWORD=admin123" >> .env
```

> 说明：
> - `JWT_ACCESS_SECRET` / `JWT_REFRESH_SECRET` 为 JWT 签名密钥，使用安全随机值即可。
> - `ADMIN_PASSWORD` 为管理员账号（admin）的登录密码，默认设为 `admin123`，如需自定义请先修改 `.env` 中的对应值再启动。

### 3. 启动全部服务

```bash
docker-compose up -d
```

## 完成后

- 等待服务启动完成，访问 http://localhost:8080 查看博客首页。
- 首次启动会自动构建 Go 应用镜像，并自动创建 Kafka topic `comment_topic`。
- 如需强制重新构建镜像，使用 `docker-compose up -d --build`。
