# 校园Memo（Campus Memos）——基于 memos 的校园场景改造版

> 本仓库 fork 自 [usememos/memos](https://github.com/usememos/memos)（MIT License）。
> 原项目版权与许可见 [LICENSE](./LICENSE)，本文件说明**哪些属于原项目、哪些是我们新增的部分**。

## 一键安装（非技术同学也能跑）

```bash
# Linux / macOS
curl -fsSL https://raw.githubusercontent.com/<你的用户名>/campus-memos/main/scripts/install-campus.sh | bash

# Windows（PowerShell）
irm https://raw.githubusercontent.com/<你的用户名>/campus-memos/main/scripts/install-campus.ps1 | iex
```

装完打开 `http://localhost:5230` 即可。单二进制 + SQLite，无需安装任何数据库或服务。

也可以从源码构建（与原项目一致）：

```bash
cd web && npm install && npm run release && cd .. && go build -o campus-memos ./cmd/memos && ./campus-memos
```

## 功能总览

顶部导航七大分区：**好友 / 群 / 圈 / 话题区 / 推荐 / 主题分类 / 我的**。

| 功能 | 说明 |
| --- | --- |
| 关键词提取与主题分类 | 发布时自动提取关键词（中文 n-gram + #标签 + 主题词库加成），按相关性归入 10 个校园主题；管理员可一键重建索引 |
| 好友 | 用户名搜索添加、申请/通过、好友列表、好友动态（参考微信） |
| 群 | 复用原项目 Space：建群、群内动态（参考微信） |
| 圈 | 全校动态流，支持匿名发布、收藏、地点、日程（参考朋友圈/QQ 动态/Ins/推特） |
| 话题区 | #话题 热榜 + 话题内帖子（参考小红书/贴吧） |
| 推荐 | 明示同意后，按点赞/收藏/浏览记录做关键词相关性推送（参考抖音/B站） |
| 主题分类 | 分类书架式浏览（参考番茄免费小说） |
| 我的 | 日期表（日历）、我的发布、浏览记录、收藏、设置 |
| 匿名发布 | memo 级匿名：浏览者不可见作者任何信息；作者本人可管理 |
| 四档可见性 | 仅自己 / 仅好友（新增）/ 登录可见 / 完全公开，服务端强制鉴权 |
| 注册/找回密码 | 用户名（唯一）+ 密码 + 手机号 + 短信验证码；短信支持开发模式与 Webhook 对接 |
| 隐私设置 | 日期表/我的 memo/浏览记录/收藏 四类内容可独立设置公开与否 |
| 界面个性化 | 默认简洁；每个分区可单独设置静态壁纸或动态画面（仿手机动态壁纸） |
| 下载到本地 | PWA 安装到手机/电脑桌面；注册成功后仅提示一次；搜索「下载」置顶入口 |
| 日程导入 | memo 可附日程时间+提前提醒；ICS 订阅一键导入系统日历（含 VALARM 到时提醒） |
| 数据导出 | JSON / CSV（带统计汇总），方便向学校汇报与成果展示 |

## 代码归属说明（MIT 合规）

### 属于原项目（usememos/memos，MIT License）
- `cmd/`、`core/`、`filter/`、`internal/`、`proto/`、`provider/`、`server/api/v1/`、`server/auth/`、`server/fileserver/`、`server/frontend/`、`server/mcp/`、`store/`、`markdown/` 等全部原有目录与文件；
- `web/src/` 下除 `web/src/campus/` 以外的全部原有代码；
- 我们仅对原项目做了以下**最小侵入式挂载**（每处均有注释标注「本项目新增」）：
  - `server/server.go`：注册 campus 模块路由（+6 行）；
  - `web/src/router/index.tsx`：注册校园页面路由（+21 行）；
  - `web/src/layouts/MainLayout.tsx`：挂载顶部导航（+3 行）；
  - `web/src/pages/SignIn.tsx`：登录页加校园注册入口（+8 行）；
  - `web/src/main.tsx`：注册 PWA Service Worker（+10 行）；
  - `web/src/index.css`：引入校园样式（+1 行）；
  - `web/public/site.webmanifest`：更新应用名称/描述。

### 我们新增的部分（同样以 MIT 发布）
- `server/api/campus/`：校园模块后端（关键词提取与主题分类、匿名/仅好友可见、好友、浏览/收藏、推荐、设置、导出、ICS、短信注册）；
- `web/src/campus/`：校园模块前端（顶部导航、七个功能页、发布器、卡片、搜索、校园注册页、个性化设置）；
- `web/public/sw.js`：PWA Service Worker；
- `scripts/install-campus.sh` / `install-campus.ps1`：一键安装脚本；
- `docs/campus-requirements.md`：需求规格说明书；
- `CAMPUS.md`：本文件。

### 数据隔离设计
校园模块全部使用 `campus_` 前缀的独立数据表，通过 `store.Driver.GetDB()` 访问，
**不修改原项目任何表结构与 API 行为**；原项目升级后可低冲突合并。

## 校园 API 一览（/api/campus）

| 方法 | 路径 | 说明 |
| --- | --- | --- |
| POST | /auth/sms | 发送短信验证码（register/reset） |
| POST | /auth/register | 校园注册（用户名唯一+手机号+验证码） |
| POST | /auth/reset-password | 找回密码 |
| POST | /memos | 发布 memo（四档可见性/匿名/地点/日程） |
| GET | /feed?box=circle\|friends\|topics\|theme\|recommend | 各分区信息流 |
| GET | /themes | 主题分类索引（含分类词条） |
| POST | /reindex | 重建关键词索引（管理员） |
| POST | /memos/:uid/browse | 记录浏览 |
| POST | /memos/:uid/favorite | 收藏/取消收藏 |
| GET | /favorites、/history、/mine | 收藏/浏览记录/我的发布 |
| GET/POST | /friends、/friends/request、/friends/accept | 好友系统 |
| GET/PUT | /settings | 推荐授权/隐私/界面个性化 |
| GET | /export.json、/export.csv、/stats | 数据导出与统计 |
| GET | /calendar.ics | 日程 ICS 订阅（含提醒） |

## 短信通道说明
默认「开发模式」：验证码在接口响应中回显（`devCode`）并写入服务日志，方便演示与试点。
配置环境变量 `CAMPUS_SMS_WEBHOOK` 后，系统会向该地址 POST `{"phone","code","purpose"}`，
可对接阿里云/腾讯云短信或学校统一消息平台。
