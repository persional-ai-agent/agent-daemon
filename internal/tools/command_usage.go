package tools

const (
	CommandSessionUsage         = "/session [session_id]"
	CommandShowUsage            = "/show [session_id] [offset] [limit]"
	CommandNextPrevUsage        = "/next or /prev"
	CommandHistoryUsage         = "/history [n]"
	CommandSessionsUsage        = "/sessions [n]"
	CommandPickUsage            = "/pick <index>"
	CommandStatsUsage           = "/stats [session_id]"
	CommandNewResetUsage        = "/new [session_id] or /reset"
	CommandResumeUsage          = "/resume <session_id>"
	CommandRecoverUsage         = "/recover context"
	CommandClearUsage           = "/clear"
	CommandReloadUsage          = "/reload"
	CommandSaveUsage            = "/save [path]"
	CommandTargetsUsage         = "/targets [platform]"
	CommandToolsUsage           = "/tools [list|show <name>|schemas]"
	CommandToolsShowUsage       = "/tools show <name>"
	CommandToolsSchemasUsage    = "/tools schemas"
	CommandPersonalityUsage     = "/personality [show|reset|<text>]"
	CommandSkillsUsage          = "/skills [name]"
	CommandUsageUsage           = "/usage [session_id]"
	CommandCompressUsage        = "/compress [tail_messages]"
	CommandToolsetsShowUsage    = "/toolsets show <name>"
	CommandToolsetsResolveUsage = "/toolsets resolve <name[,name]>"
	CommandToolsetsUsage        = "/toolsets [list|show <name>|resolve <name[,name]>]"
	CommandResetUsage           = "/reset [session_id]"
	CommandSaveFileUsage        = "/save <file>"
	CommandSessionsPickUsage    = "/sessions [limit] [pick <index>]"
	CommandShowPickUsage        = "/show [session] [offset>=0] [limit>0] [pick <index>]"
	CommandStatsSessionUsage    = "/stats [session]"
	CommandPendingUsage         = "/pending [limit] [approve|deny|a|d <index>]"
	CommandOpenUsage            = "/open <index> [a|d|approve|deny]"
)

func UsageEN(usage string) string { return "Usage: " + usage }
func UsageZH(usage string) string { return "用法: " + usage }

func CLIWelcomeHintZH() string {
	return "输入 /help 查看可用命令，/quit 退出。"
}

func UnknownCommandHintZH() string {
	return "输入 /help 查看命令"
}

func UnknownCommandMessageZH(cmd string) string {
	return "未知命令: " + cmd + "（" + UnknownCommandHintZH() + "）"
}
