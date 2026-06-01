# React 编码规范与 AI 协作指南

本规范参考 React 官方文档、TypeScript 官方手册、Vite 官方模板实践，以及 Airbnb JavaScript / React JSX Style Guide，并结合本项目的 AI 协作开发场景整理。不要照搬外部规范中的过时类组件写法，本项目默认使用函数组件和 Hooks。

参考来源：

- React 官方 Hooks 规则：https://react.dev/reference/eslint-plugin-react-hooks/lints/rules-of-hooks
- TypeScript Handbook：https://www.typescriptlang.org/docs/handbook/intro.html
- Airbnb JavaScript Style Guide：https://github.com/airbnb/javascript
- Airbnb React/JSX Style Guide：https://javascript.airbnb.tech/react/
- Vite 官方文档：https://vite.dev/

## 基本原则

- 优先使用函数组件、Hooks 和 TypeScript。
- 组件保持小而清晰，一个组件只负责一个明确的 UI 或业务职责。
- 业务逻辑、数据请求、状态管理和 UI 展示分层组织，避免全部写进页面组件。
- 不为了“未来可能需要”提前抽象，先让代码直接、可读、可测试。
- AI 生成代码后必须能被人类快速理解，因此命名要准确，文件职责要稳定。

## 推荐目录结构

```text
src/
  assets/              静态资源
  components/          通用 UI 组件
  features/            按业务功能组织代码
  hooks/               通用 Hooks
  lib/                 无框架依赖的工具函数
  pages/               页面级组件
  services/            API、WebSocket 等外部通信封装
  styles/              全局样式、变量、主题
  types/               跨模块共享类型
```

功能代码优先放在 `features/<feature-name>/` 中。只有多个功能都会复用的代码，才提升到 `components/`、`hooks/`、`lib/` 或 `types/`。

## TypeScript 规范

- 禁止随意使用 `any`，外部未知数据优先用 `unknown`，再做类型收窄。
- 组件 props 类型命名为 `ComponentNameProps`。
- 普通对象类型优先使用 `type`；需要声明合并时再使用 `interface`。
- 复杂状态优先使用联合类型表达，例如 `idle | loading | success | error`。
- API 返回值、表单值、权限信息都必须有明确类型。

## React 规范

- Hooks 只能在组件或自定义 Hook 的顶层调用，不能写在条件、循环或普通函数中。
- 能从 props 或已有 state 推导出的值，不要重复存入 state。
- `useEffect` 只用于同步外部系统，例如请求、订阅、定时器、浏览器存储和 DOM API。
- `useMemo`、`useCallback` 只在依赖稳定性或性能确有需要时使用。
- 事件处理函数命名使用 `handleXxx`，传给子组件的回调使用 `onXxx`。

## 后台页面规范

- 页面以数据管理效率为核心，优先保证列表、筛选、表单、详情、弹窗流程清晰。
- 表格页必须明确处理 loading、empty、error、pagination 和筛选状态。
- 表单页必须明确处理初始值、校验、提交中、提交失败和提交成功状态。
- 权限判断应集中封装，不要在多个页面中散落重复条件。

## 样式规范

- 全局样式只放 reset、主题变量和基础排版。
- 组件样式应靠近组件或归属于功能模块。
- class 命名表达结构和语义，不使用颜色、字号、间距作为主要命名。
- 后台页面避免过度装饰，优先保证信息密度、对齐、可扫描性和操作反馈。

## AI 协作规则

- 修改前先确认当前任务所属模块，只改必要文件。
- 不主动重构无关代码，不删除用户已有代码，除非任务明确要求。
- 新增依赖前必须说明用途、替代方案和为什么当前内置能力不够。
- 生成代码后检查类型、边界状态、空状态、错误状态和桌面端布局。
- 每次新增公共组件、Hook 或工具函数，都要保证命名能让 AI 从名称理解用途。
- 若需求不明确，优先在文档或注释中记录假设，不把猜测写死成复杂实现。
