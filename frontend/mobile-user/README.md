# LiveBid 用户前台

`mobile-user` 是 LiveBid 的用户端前台项目，面向移动端 H5 场景。后续主要承载用户登录注册、直播间浏览、商品展示、竞拍出价、倒计时、实时消息等功能。

## 技术栈

- React
- TypeScript
- Vite
- ESLint
- React Router
- TanStack Query
- Zustand
- Swiper
- Ant Design Mobile
- Lucide React

首页沉浸式直播流、竞拍浮层和直播间叠层使用业务自定义样式；登录、按钮、弹窗、轻提示等通用控件使用 Ant Design Mobile。

## 常用命令

```bash
npm install
npm run dev
npm run build
npm run lint
npm run preview
```

## 本地联调

默认请求同源接口，并由 Vite 开发服务代理到本地后端：

```bash
npm run dev
```

默认代理目标：

- `/api` -> `http://127.0.0.1:58080`
- `/ws` -> `ws://127.0.0.1:58081`
- `/rtc` -> `http://127.0.0.1:1985`

如果本地端口不同，可覆盖代理目标：

```bash
VITE_API_PROXY_TARGET=http://127.0.0.1:58080 \
VITE_WS_PROXY_TARGET=ws://127.0.0.1:58081 \
VITE_SRS_PROXY_TARGET=http://127.0.0.1:1985 \
npm run dev
```

不建议在本地直接设置 `VITE_API_BASE_URL=http://127.0.0.1:58080`，除非 `api-gateway` 已启用 CORS；否则浏览器会因为跨域拦截请求。

当前用户端依赖的核心接口：

- `GET /api/user/live/feed`
- `GET /api/user/live/rooms/:room_id/entry`
- `GET /api/user/live/rooms/:room_id/auction-snapshot`
- `POST /api/users/login`
- `POST /api/users/register`
- `GET /api/users/:id`
- `GET /ws/live?room_id=:room_id&token=:access_token`
- `POST /rtc/v1/play/`

## 目录说明

```text
src/
  assets/       静态资源
  components/   通用组件
  features/     业务功能模块
  hooks/        通用 Hooks
  lib/          工具函数
  services/     HTTP 与 WebSocket 通信封装
  stores/       跨页面状态
  types/        共享类型
  App.tsx       路由与应用入口
  main.tsx      React 挂载入口
  index.css     全局基础样式
```

后续编写业务代码前，请先阅读 `CODING_GUIDELINES.md`。

## 协作文档

- 需求设计：`../../docs/frontend/mobile-user/需求设计.md`
- 设计规范：`用户端设计规范.md`
- 编码规范：`CODING_GUIDELINES.md`
