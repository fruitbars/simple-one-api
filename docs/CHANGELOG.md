# simple-one-api更新日志

## Unreleased

1. Release workflow 同步发布 `linux/amd64`、`linux/arm64` 多架构 GHCR 镜像，包含语义化标签、provenance 和 SBOM。
2. Docker 镜像增加非 root 运行、`/healthz` 健康检查，并统一 Compose、本地构建脚本和跨架构参数。
3. 更新中英文 README 和构建发布文档，增加 GitHub Release/GHCR 入口并移除失效的外部统计图。

## v0.10.1 - 2026-08-17

1. 增加内嵌 React Web 聊天界面和 Wails v2 桌面端，共用同一套 API 与资源。
2. 增加 SQLite 配置版本库、发布、激活、回滚、文件配置导入和运行时原子快照。
3. Admin 默认改为可视化编辑，保留配置源码作为复杂配置和未知字段的无损入口，并支持 JSON/YAML 导入导出。
4. 增加 Provider 稳定 ID、配置规范化、端口/代理/限流校验，以及扩展字段透传。
5. 增加远程首次初始化 bootstrap token；设置正式 `api_key` 后临时 token 失效。
6. 统一服务端、Web、桌面端和 Docker 构建说明，修复交叉编译脚本未使用目标平台的问题。
7. 重写中英文 README，移除 2024 年额度表和失效链接，并将历史 Provider 文档明确标记为非权威资料。
8. 删除已由 React/Web 单文件方案替代的 `static/` 页面和 jQuery 资源，清理手工客户端与忽略规则。
9. 增加 GitHub Actions：持续验证 Go/Web/Docker/Wails，并在 `v*` 标签上构建服务端与桌面端发布包。
10. 增加 OpenAI Responses 与 Anthropic Messages 客户端协议，支持 Codex、Claude Code、工具调用、图片输入和实时 SSE 转换。
11. 增加可开关的后台实时日志，支持级别筛选、自动跟随、敏感信息脱敏和固定容量环形缓冲。
12. 改进桌面聊天体验：Markdown 渲染、打字机增量、Token 用量与速度、输出自动滚动、本地对话历史，以及更清晰的 Provider 配置流程。
13. 下线 Coze v2/v3 与 Baidu AgentBuilder 等失效旧接入，增加请求体限制、取消传播、协议错误映射和并发安全测试。
14. 增加 Provider/模型粒度熔断、半开自动恢复、未知请求字段透传，以及聊天窗口思考开关和流式思考展示。

## v0.3 - 2024-06-04
1. 程序调整默认gin为release模型
2. 支持了星火的function call
3. 修复了abab6-chat的默认maxtokens太小的问题（自动调整为最大）
4. 千帆的模型maxtokens超出时，自动调整为区间范围内
