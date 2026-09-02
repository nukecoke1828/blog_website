# My Blog Website

一个基于 **Go 语言** 开发的个人博客网站，采用 **Gin + GORM + MySQL + Redis + Kafka + etcd** 技术栈，支持 Docker 一键部署。

## ✨ 功能特性

### 核心功能
- **用户认证系统** — 支持登录/注册（首次登录即自动注册），基于 JWT 的 AccessToken + RefreshToken 双令牌机制，自动续期与登出撤销
- **博客管理** — 文章列表（分页展示）、文章详情查看、文章创建（仅限管理员）
- **点赞功能** — 支持对博客文章和评论进行点赞/取消点赞
- **评论系统** — 支持对文章发表评论，以及多层级的嵌套回复；评论通过 **Kafka** 异步落库，提升响应速度
- **评论删除** — 支持作者或管理员删除评论（递归删除所有子回复）
- **个人主页** — 展示个人信息

### 高级特性
- **JWT 密钥热更新** — 通过 **etcd** 管理 JWT 签名密钥，配合定时任务自动轮换密钥；密钥变更时自动撤销所有旧 RefreshToken 并签发新令牌
- **Redis 缓存** — 对博客列表分页查询和总数统计做缓存加速，支持缓存失效与防穿透
- **CSRF 防护** — 使用 gorilla/csrf 中间件保护表单提交
- **静态资源嵌入** — 前端静态文件（CSS/JS）通过 Go embed 嵌入二进制，部署无需额外文件
- **优雅退出** — 支持 SIGINT/SIGTERM 信号捕获，HTTP 服务平滑关闭，Kafka 资源安全释放
- **Docker 一键部署** — 提供完整的 Dockerfile 与 docker-compose.yaml，包含 Go 应用、MySQL、etcd、Kafka、Redis 全套服务编排

## 🏗️ 技术栈

| 技术 | 用途 |
|------|------|
| **Go 1.26** | 主要编程语言 |
| **Gin** | HTTP Web 框架 |
| **GORM** | ORM 框架，操作 MySQL 数据库 |
| **MySQL 8.0** | 主数据库，存储用户、博客、评论等数据 |
| **Redis 7** | 缓存层，加速分页查询 |
| **Kafka** | 消息队列，异步处理评论入库 |
| **etcd** | 配置中心，管理 JWT 密钥并支持热更新 |
| **JWT (golang-jwt/v5)** | 用户认证与授权 |
| **bcrypt** | 密码加密存储 |
| **Docker & Docker Compose** | 容器化一键部署 |

## 📁 项目结构

```
my_blog_website/
├── cmd/                        # 应用入口
│   └── main.go                 # 主函数，路由注册，启动服务
├── config/                     # 配置管理
│   └── model.go                # etcd 配置管理器（热更新）
├── handlers/                   # HTTP 请求处理器
│   ├── blog.go                 # 博客相关（列表、详情、创建、点赞、评论、分页）
│   ├── home.go                 # 首页
│   ├── login.go                # 登录/登出
│   └── profile.go              # 个人主页
├── middleware/                  # 中间件
│   └── auth.go                 # JWT 认证中间件（普通用户 & 管理员）
├── models/                     # 数据模型
│   ├── db.go                   # 数据库初始化（MySQL 连接 + 自动建库/迁移）
│   ├── models.go               # 数据模型定义（User, Blog, Comment, Like 等）
│   └── redis.go                # Redis 缓存封装
├── templates/                  # HTML 模板
│   ├── index.html              # 首页
│   ├── blog_list.html          # 博客列表
│   ├── blog_detail.html        # 博客详情
│   ├── blog_new.html           # 新建博客
│   ├── login.html              # 登录页
│   ├── nav.html                # 导航栏
│   ├── profile.html            # 个人主页
│   └── no_permission.html      # 无权限提示
├── assets/                     # 静态资源（嵌入二进制）
│   ├── static/
│   │   ├── app.js
│   │   └── style.css
│   └── assets.go               # embed 声明
├── utils/                      # 工具函数
│   ├── encrypt.go              # bcrypt 密码加密与验证
│   ├── jwt.go                  # JWT 生成/验证/刷新/撤销
│   ├── kafka/                  # Kafka 生产者/消费者
│   │   ├── kafka.go            # Kafka 初始化与关闭
│   │   ├── producer/producer.go
│   │   └── consumer/consumer.go
│   ├── secret_key/
│   │   └── key.go              # 安全随机密钥生成
│   └── taskrunner/             # 定时任务调度器
│       ├── trmain.go           # 定时任务启动
│       ├── runner.go           # 调度器核心循环
│       ├── config_task.go      # JWT 密钥热更新任务
│       └── defs.go             # 类型定义
├── Dockerfile                  # 多阶段构建
├── docker-compose.yaml         # Docker Compose 编排
├── .env.example                # 环境变量示例
├── go.mod
└── go.sum
```

## 🚀 快速开始

### 前提条件

- 安装 [Docker](https://docs.docker.com/get-docker/) 和 [Docker Compose](https://docs.docker.com/compose/install/)

### 使用 Docker Compose 一键部署

1. **克隆项目**

```bash
git clone https://github.com/nukecoke1828/blog_website.git
cd blog_website
```

2. **配置环境变量**

```bash
cp .env.example .env
```

编辑 `.env` 文件，设置以下变量：

```env
MYSQL_ROOT_PASSWORD=your_root_password
MYSQL_PASSWORD=your_blog_user_password
DB_PASSWORD=your_db_password
JWT_ACCESS_SECRET=your_access_secret
JWT_REFRESH_SECRET=your_refresh_secret
ADMIN_PASSWORD=admin_password
```

其中 `JWT_ACCESS_SECRET`、`JWT_REFRESH_SECRET`、`ADMIN_PASSWORD` 可以使用提供的获取环境变量的方法设置默认值。

3. **启动全部服务**

```bash
docker-compose up -d
```

等待所有服务启动完成，访问 **http://localhost:8080** 即可看到博客首页。

### 使用 Claude Code 技能一键启动（推荐）

如果你正在使用 Claude Code，无需手动执行上面的命令，直接在项目目录下运行以下技能即可自动完成启动/停止：

- **`/start-blog`** — 自动检查并补全缺失的环境变量（`JWT_ACCESS_SECRET`、`JWT_REFRESH_SECRET`、`ADMIN_PASSWORD`），然后执行 `docker-compose up -d` 启动全部服务
- **`/stop-blog`** — 停止博客项目并清空容器中的所有数据（执行 `docker-compose down -v`）

### 本地开发（不使用 Docker）

如果你希望在本地直接运行 Go 应用进行开发，需要提前启动 MySQL、Redis、Kafka 和 etcd 服务，然后：

```bash
# 设置环境变量
export DB_HOST=localhost
export DB_PORT=3306
export DB_USER=root
export DB_PASSWORD=your_password
export DB_NAME=blogdb
export REDIS_HOST=localhost
export REDIS_PORT=6379
export KAFKA_BROKERS=localhost:9092
export ETCD_ENDPOINTS=localhost:2379
export JWT_ACCESS_SECRET=your_access_secret
export JWT_REFRESH_SECRET=your_refresh_secret
export ADMIN_PASSWORD=admin_password

# 运行应用
go run ./cmd
```

## 📖 API 路由

| 方法 | 路径 | 说明 | 权限 |
|------|------|------|------|
| GET | `/` | 首页 | 公开 |
| GET | `/login` | 登录页面 | 公开 |
| POST | `/login` | 提交登录 | 公开 |
| GET | `/logout` | 退出登录 | 登录用户 |
| GET | `/profile` | 个人主页 | 公开 |
| GET | `/blog` | 博客列表（分页） | 登录用户 |
| GET | `/blog/:id` | 博客详情 | 登录用户 |
| POST | `/blog/:id/like` | 点赞/取消点赞文章 | 登录用户 |
| POST | `/blog/:id/comment` | 发表评论（走 Kafka） | 登录用户 |
| POST | `/comment/:id/like` | 点赞/取消点赞评论 | 登录用户 |
| POST | `/comment/:id/reply` | 回复评论（嵌套） | 登录用户 |
| POST | `/comment/:id/delete` | 删除评论 | 作者/管理员 |
| GET | `/blog/create` | 创建博客页面 | 管理员 |
| POST | `/blog/create` | 提交创建博客 | 管理员 |

## 🔐 认证流程

```
用户登录
  │
  ├── 首次登录 → 自动创建用户
  │
  └── 密码验证通过
       │
       ├── 签发 AccessToken（5分钟有效期，存 Cookie）
       ├── 签发 RefreshToken（7天有效期，存 Cookie + 数据库）
       │
       └── 后续请求
            │
            ├── AccessToken 有效 → 直接放行
            │
            └── AccessToken 过期
                 │
                 ├── 用 RefreshToken 刷新 → 签发新的令牌对
                 │    └── 旧 RefreshToken 撤销，新 RefreshToken 继承原过期时间
                 │
                 └── RefreshToken 也过期 → 重新登录
```

JWT 签名密钥通过 etcd 管理，定时任务（每 30 天）自动生成新密钥并写入 etcd；所有应用实例通过 Watch 机制实时获取最新密钥，实现**零停机密钥轮换**。

## 🗄️ 评论异步处理

用户发表评论时，评论数据通过 **Kafka Producer** 发送到消息队列，由独立的 **Kafka Consumer** 异步批量写入数据库。这样的设计：

- 解耦了 HTTP 请求处理与数据库写入，提升用户体验
- 支持消息幂等（每条评论使用 UUID 作为 `MsgID` 唯一键，防止重复消费）
- 批量写入提高数据库吞吐量

## 🐳 Docker 服务架构

```
                    ┌──────────────────────┐
                    │      Nginx (可选)     │
                    │      Port: 80/443    │
                    └──────────┬───────────┘
                               │
                    ┌──────────▼───────────┐
                    │    Go App :8080      │
                    │  (Gin HTTP Server)   │
                    └──┬─────┬──────┬──┬───┘
                       │     │      │  │
          ┌────────────▼┐ ┌──▼──┐ ┌─▼──▼──┐
          │    MySQL     │ │Redis│ │ Kafka │
          │   :3306      │ │:6379│ │ :9092 │
          └──────────────┘ └─────┘ └───────┘
                              │
                       ┌──────▼──────┐
                       │    etcd     │
                       │   :2379     │
                       └─────────────┘
```

## 📝 待办事项

- [ ] 图片/视频上传功能
- [ ] 文章支持 Markdown 格式编写
- [ ] 性能优化（数据库查询优化、索引优化）
- [ ] 单元测试与集成测试
- [ ] CI/CD 流水线
- [ ] 文章搜索功能
- [ ] 文章分类与标签筛选
- [ ] RSS 订阅

## 🤝 贡献

如果你有好的建议或想法，欢迎在 [GitHub](https://github.com/nukecoke1828/blog_website) 上提交 Issue 或 Pull Request！

## 📄 许可证

MIT License

---

**作者：** NukeCoke  
**邮箱：** imdxhd@gmail.com  
**GitHub：** [@nukecoke1828](https://github.com/nukecoke1828)
