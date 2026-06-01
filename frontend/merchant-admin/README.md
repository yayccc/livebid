# LiveBid 商家后台

`merchant-admin` 是 LiveBid 的商家后台项目，面向桌面端管理场景。后续主要承载商家登录、商品管理、竞拍管理、订单管理、直播管理、数据列表和表单录入等功能。

## 技术栈

- React
- TypeScript
- Vite
- ESLint

本工程由 Vite 官方 `react-ts` 模板生成，目前只保留模板默认依赖，暂未手动引入路由、状态管理、请求、表单、测试或 UI 组件库。

## 常用命令

```bash
npm install
npm run dev
npm run build
npm run lint
npm run preview
```

## 目录说明

```text
src/
  assets/       静态资源
  App.tsx       应用入口组件
  main.tsx      React 挂载入口
  index.css     全局样式
```

后续编写业务代码前，请先阅读 `CODING_GUIDELINES.md`。
