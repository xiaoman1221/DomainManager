# DomainManager

域名管理系统 - 集域名管理、WHOIS查询、ICP备案查询、价格比对、证书管理、到期通知于一体的综合平台。

## 功能特性

- **域名管理** - 域名的增删改查、批量操作、CSV导入导出
- **WHOIS查询** - 域名注册信息查询（依赖 [next-whois](https://github.com/zmh-program/next-whois)）
- **ICP备案查询** - 域名备案信息查询（依赖 [ICP_Query](https://github.com/HG-ha/ICP_Query)）
- **价格比对** - 多注册商域名续费价格对比
- **注册商管理** - 注册商信息维护、域名批量导入
- **证书管理** - SSL证书管理，支持与 Certimate 同步
- **到期通知** - 多渠道域名到期提醒（邮件、Webhook等）
- **用户系统** - 多用户支持、角色权限管理
- **系统设置** - 系统参数配置

## 技术栈

**后端**
- Go 1.26 + Gin
- GORM + SQLite
- JWT 认证

**前端**
- Vue 3 + Vite
- Element Plus
- Pinia 状态管理
- Vue Router

## 快速开始

### 环境要求

- Go 1.26+
- Node.js 18+
- npm

### 配置

复制 `.env.example` 为 `.env` 并修改配置：

```bash
cp .env.example .env
```

| 变量 | 说明 | 默认值 |
|------|------|--------|
| `PORT` | 服务端口 | 8080 |
| `GIN_MODE` | Gin模式（debug/release） | debug |
| `DB_PATH` | 数据库文件路径 | domain_manager.db |
| `JWT_KEY` | JWT密钥 | - |
| `WHOIS_API_URL` | WHOIS API地址 | https://who.zmh.me |
| `ICP_API_URL` | ICP备案API地址 | http://127.0.0.1:16181 |

### 构建运行

**构建前端**

```bash
cd web
npm install
npm run build
```

**构建并运行后端**

```bash
go build -o domain-manager.exe .
./domain-manager.exe
```

### 默认账号

首次运行会自动创建管理员账号：

- 用户名：`admin`
- 密码：`123456`

> 请登录后立即修改默认密码。

## API 接口

| 模块 | 路径前缀 | 说明 |
|------|----------|------|
| 认证 | `/api/auth` | 注册、登录、获取用户信息 |
| 域名 | `/api/domains` | 域名CRUD、批量操作、导入导出 |
| WHOIS | `/api/whois` | WHOIS查询 |
| ICP | `/api/icp` | ICP备案查询 |
| 价格 | `/api/price` | 域名价格查询与比对 |
| 注册商 | `/api/registrars` | 注册商管理 |
| 证书 | `/api/certificates` | 证书管理、Certimate同步 |
| 通知 | `/api/notifications` | 通知渠道与日志管理 |
| 设置 | `/api/settings` | 系统设置、用户管理 |

## 项目结构

```
DomainManager/
├── config/          # 配置加载
├── database/        # 数据库初始化与迁移
├── handlers/        # 请求处理器
├── middleware/       # 中间件（JWT认证）
├── models/          # 数据模型
├── router/          # 路由定义
├── services/        # 业务逻辑（WHOIS/ICP/价格等）
├── utils/           # 工具函数
├── web/             # Vue前端
│   ├── src/
│   └── dist/        # 前端构建产物
├── .env.example     # 环境变量模板
├── go.mod
└── main.go
```

## License

MIT
