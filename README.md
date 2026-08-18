# DomainManager

域名管理系统 - 集域名管理、WHOIS查询、ICP备案查询、价格比对、证书管理、到期通知于一体的综合平台。

## 功能特性

- **域名管理** - 域名的增删改查、批量操作、CSV导入导出
- **WHOIS查询** - 域名注册信息查询（依赖 [next-whois](https://github.com/zmh-program/next-whois)；`*.pp.ua` 免费二级域名走 UANIC 官方 whois 服务器，即 [dig.ua](https://dig.ua) 的数据源）
- **ICP备案查询** - 域名备案信息查询（依赖 [ICP_Query](https://github.com/HG-ha/ICP_Query)）
- **价格比对** - 多注册商域名续费价格对比（当前为估算参考价，非实时注册商报价）
- **注册商管理** - 注册商信息维护、域名批量导入
- **证书管理** - SSL证书管理，支持与 Certimate 同步
- **到期通知** - 多渠道域名到期提醒（邮件、Webhook等）
- **用户系统** - 多用户支持、角色权限管理
- **第三方登录** - 接入 OauthGo 或 彩虹聚合登录，支持 QQ / 微信 / Gitee 等渠道一键登录
- **系统设置** - 系统参数配置

## 技术栈

**后端**
- Go 1.26 + Gin
- GORM + SQLite
- JWT 认证

**前端**
- React 18 + Vite
- Ant Design 5
- React Router
- axios

## 快速开始

### Docker 部署（推荐）

```bash
docker compose up -d
```

包含三个服务：
- **domain-manager** - 主应用 (`xiaoman1221/domain-manager:latest`)
- **whois** - WHOIS查询服务 (`programzmh/next-whois-ui`)
- **icp** - ICP备案查询服务 (`yiminger/ymicp`)

访问 http://localhost:8080

### 手动部署

**环境要求**

- Go 1.26+
- Node.js 18+
- npm

**配置**

复制 `.env.example` 为 `.env` 并修改配置：

```bash
cp .env.example .env
```

| 变量 | 说明 | 默认值 |
|------|------|--------|
| `PORT` | 服务端口 | 8080 |
| `GIN_MODE` | Gin模式（debug/release） | debug |
| `DB_PATH` | 数据库文件路径 | domain_manager.db |
| `JWT_KEY` | JWT密钥（建议 `openssl rand -hex 32` 生成；不设置时每次启动随机生成，重启后所有登录失效） | - |

**运行时配置（迁移至数据库，管理员可在 个人设置 → 系统管理 → 系统设置 中修改，保存后立即生效）**

系统设置按类别分组（个人设置 → 系统设置，仅管理员）：

| 分组 | 配置项 | 说明 |
|------|--------|------|
| 基础 | `WHOIS_API_URL` / `ICP_API_URL` / `DIGITALPLAT_RDAP_URL` | 查询服务地址 |
| 第三方登录 | `OAUTH_PROVIDER`（`oauthgo`/`rainbow`） | 选择登录服务：OauthGo 或 彩虹聚合登录 |
| 第三方登录 | `OAUTHGO_BASE_URL` / `APP_ID` / `APP_KEY` / `REDIRECT_URI` | OauthGo 配置（留空禁用） |
| 第三方登录 | `RAINBOW_BASE_URL` / `APP_ID` / `APP_KEY` / `REDIRECT_URI` | 彩虹聚合登录配置（留空禁用） |
| SMTP | `SMTP_HOST` / `PORT` / `USERNAME` / `PASSWORD` / `FROM` / `FROM_NAME` / `ENCRYPTION` / `ENABLED` | 全局 SMTP，作为邮件通知渠道的默认服务器 |
| 支付系统 | `PAYMENT_PROVIDER` / `MERCHANT_ID` / `APP_ID` / `APP_KEY` / `NOTIFY_URL` / `ENABLED` | 支付渠道配置（预留） |
| SNS | `SNS_CONFIG` | 社交平台链接（展示在登录页页脚） |
| 页脚 | `FOOTER_DESCRIPTION` / `COPYRIGHT` / `ICP` / `POLICE` / `LINKS` | 登录页页脚文案与链接 |

> 上述运行时配置在首次启动时以 `.env` 中的值为默认值写入数据库，之后以数据库为准。

**构建运行**

> 后端通过 `//go:embed web/dist/*` 内嵌前端构建产物，因此 `go build` 前必须先构建前端。

```bash
# 一键构建（前端+后端）
make
# 或
bash build.sh

# 运行
./domain-manager
```

### 默认账号

首次运行会自动创建管理员账号：

- 用户名：`admin`
- 密码：`123456`

> 请登录后立即修改默认密码。


### 第三方登录

本系统支持通过 [OauthGo](https://o.1v.fit/docs) 或 彩虹聚合登录（兼容 `u.cccyun.cc` 的 `connect.php` 协议）接入第三方登录（QQ、微信、Gitee、Google、GitHub 等渠道）。

1. 在 系统设置 → 第三方登录 中通过 `OAUTH_PROVIDER` 选择登录服务（`oauthgo` 或 `rainbow`）。
2. 在对应平台注册应用，获得 `APP_ID` / `APP_KEY`，并将本系统地址加入回调域名白名单（默认回调路径为 `/api/auth/oauth/callback`）。
3. 配置所选服务（两种方式任选，数据库保存后立即生效）：
   - 首次启动前：在 `.env` 中填写（作为初始默认值写入数据库），或
   - 启动后：在 个人设置 → 系统管理 → 系统设置 中填写对应服务的地址与应用凭证。

4. 登录弹窗中会出现「或使用第三方账号登录」入口。

首次通过第三方渠道登录会自动创建本地账号（默认普通用户角色组），之后同一渠道账号会直接登录并自动更新昵称/头像。

可在 系统设置 → 第三方登录 中勾选「启用登录方式」，控制登录弹窗中展示的第三方渠道；不勾选时展示所选服务已开启的全部渠道。

登录后在 个人设置 → 第三方登录 中可绑定/解绑多个第三方账号，绑定后可直接用该账号登录。


## 角色组与用户组

- **角色组（role_group）**：决定访问权限，取值 `admin`（管理员）或 `user`（普通用户）。系统设置、用户管理等后端管理功能**仅角色组为管理员的用户可访问**。
- **用户组（user_group）**：用于组织标记（如「运维组」「运营组」），仅作标识，不影响权限。可在 个人设置 → 系统管理 → 用户管理 中为每个用户设置。

首次部署时自动创建的管理员账号属于 `admin` 角色组；通过第三方登录创建的账号默认属于 `user` 角色组。

用户管理（`/users`）页面支持查看、编辑（昵称/邮箱/角色组/用户组）与删除用户，仅管理员角色组可用；不能删除当前登录账号或最后一个管理员。

## 权限说明

- 用户管理、系统设置、系统信息、Certimate 配置/同步等接口仅管理员可用。
- 注册商数据按用户隔离，互不可见。
- 登录/注册接口有基于 IP 的限流保护。

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
├── web/             # React 前端
│   ├── src/
│   │   ├── api/      # 接口封装
│   │   ├── pages/    # 页面
│   │   ├── layouts/  # 主布局
│   │   └── utils/    # 头像/格式化等
│   └── dist/        # 前端构建产物
├── .env.example     # 环境变量模板
├── go.mod
└── main.go
```

## License

MIT
