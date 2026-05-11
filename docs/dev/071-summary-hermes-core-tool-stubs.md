# 071 总结：Hermes core 工具缺口的 stub 对齐

## 背景

Hermes `_HERMES_CORE_TOOLS` 默认包含 browser/vision/image_gen/tts 等工具域。
agent-daemon 目前不实现这些能力，但为了减少与 Hermes toolsets/脚本的“名称不匹配”问题，可以先补齐接口级 stub。

## 本次变更

新增以下工具名的 stub（调用会返回 `not implemented in agent-daemon`）：

- `vision_analyze`
- `image_generate`
- `mixture_of_agents`
- `text_to_speech`
- `browser_*`（navigate/snapshot/click/type/scroll/back/press/get_images/vision/console/cdp/dialog）

并在 `toolsets` 中补齐 `vision/image_gen/browser/tts` 分组，且将它们纳入 `core` includes（接口对齐优先）。

## 边界

这只是接口对齐，不代表能力对齐；真正实现需要引入浏览器后端、视觉模型接入与音频管线等。
