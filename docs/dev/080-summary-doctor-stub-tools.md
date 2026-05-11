# 080 总结：doctor 增加 stub_tools 检查项

`agentd doctor` 新增 `stub_tools` 检查项：

- 当启用 `tools.enabled_toolsets` 时，若解析结果包含 browser/vision/tts 等 stub 工具，会给出 `warn` 提示（能力未实现）。
- 当未启用 toolsets 时，也会提示这些 stub 工具“仅接口对齐”。

