# ui-tui 体验增强总结（命令别名与容错输入）

## 本阶段完成

1. 新增命令别名与无斜杠输入容错：
   - `:q` / `quit` -> `/quit`
   - `ls` -> `/tools`
   - `show ...` -> `/show ...`
   - `gw` / `gw ...` -> `/gateway status` / `/gateway ...`
   - `cfg` / `cfg ...` -> `/config get` / `/config ...`
2. `help` 输出同步显示别名提示。

## 结果

在纯终端环境下减少输入负担，提升高频操作效率。

## 验证

- `go test ./...` 通过。
