package tools

import "fmt"

const (
	CommandSessionUsage          = "/session [session_id]"
	CommandShowUsage             = "/show [session_id] [offset] [limit]"
	CommandNextPrevUsage         = "/next or /prev"
	CommandHistoryUsage          = "/history [n]"
	CommandSessionsUsage         = "/sessions [n]"
	CommandPickUsage             = "/pick <index>"
	CommandStatsUsage            = "/stats [session_id]"
	CommandNewResetUsage         = "/new [session_id] or /reset"
	CommandResumeUsage           = "/resume <session_id>"
	CommandRecoverUsage          = "/recover context"
	CommandClearUsage            = "/clear"
	CommandReloadUsage           = "/reload"
	CommandSaveUsage             = "/save [path]"
	CommandTargetsUsage          = "/targets [platform]"
	CommandToolsUsage            = "/tools [list|show <name>|schemas]"
	CommandToolsShowUsage        = "/tools show <name>"
	CommandToolsSchemasUsage     = "/tools schemas"
	CommandPersonalityUsage      = "/personality [show|reset|<text>]"
	CommandSkillsUsage           = "/skills [name]"
	CommandUsageUsage            = "/usage [session_id]"
	CommandCompressUsage         = "/compress [tail_messages]"
	CommandToolsetsShowUsage     = "/toolsets show <name>"
	CommandToolsetsResolveUsage  = "/toolsets resolve <name[,name]>"
	CommandToolsetsUsage         = "/toolsets [list|show <name>|resolve <name[,name]>]"
	CommandResetUsage            = "/reset [session_id]"
	CommandSaveFileUsage         = "/save <file>"
	CommandSessionsPickUsage     = "/sessions [limit] [pick <index>]"
	CommandShowPickUsage         = "/show [session] [offset>=0] [limit>0] [pick <index>]"
	CommandStatsSessionUsage     = "/stats [session]"
	CommandPendingUsage          = "/pending [limit] [approve|deny|a|d <index>]"
	CommandOpenUsage             = "/open <index> [a|d|approve|deny]"
	CommandPanelAutoUsage        = "/panel auto on|off"
	CommandPanelIntervalUsage    = "/panel interval <sec>"
	CommandPanelUsage            = "/panel [overview|dashboard|sessions|tools|approvals|gateway|diag|next|prev]"
	CommandViewUsage             = "/view human|json"
	CommandFullscreenUsage       = "/fullscreen [on|off]"
	CommandDiagExportUsage       = "/diag export <file>"
	CommandReconnectTimeoutUsage = "/reconnect timeout wait|reconnect|cancel"
	CommandReconnectUsage        = "/reconnect status|on|off|now|timeout ..."
	CommandRerunUsage            = "/rerun <index>"
	CommandBookmarkUsage         = "/bookmark add <name> | /bookmark list | /bookmark use <name>"
	CommandWorkbenchUsage        = "/workbench save|list|load|delete ..."
	CommandWorkbenchSaveUsage    = "/workbench save <name>"
	CommandWorkbenchLoadUsage    = "/workbench load <name>"
	CommandWorkbenchDeleteUsage  = "/workbench delete <name>"
	CommandWorkflowUsage         = "/workflow save|list|run|delete ..."
	CommandWorkflowSaveUsage     = "/workflow save <name> <cmd1;cmd2;...>"
	CommandWorkflowRunUsage      = "/workflow run <name> [dry]"
	CommandWorkflowDeleteUsage   = "/workflow delete <name>"
	CommandGatewayUsage          = "/gateway status|enable|disable|resolve <platform> <chat_type> <chat_id> <user_id> [user_name]"
	CommandGatewayResolveUsage   = "/gateway resolve <platform> <chat_type> <chat_id> <user_id> [user_name]"
	CommandConfigUsage           = "/config get|set <section.key> <value>|tui"
	CommandConfigSetUsage        = "/config set <section.key> <value>"
	CommandApproveUsage          = "/approve <approval_id>"
	CommandDenyUsage             = "/deny <approval_id>"
	CommandPrettyUsage           = "/pretty on|off"
	CommandToolUsage             = "/tool <name>"
)

func UsageEN(usage string) string { return "Usage: " + usage }
func UsageZH(usage string) string { return "用法: " + usage }

func UsageENEither(left, right string) string {
	return UsageEN(left + " or " + right)
}

func NotSupportedBySessionStoreEN(feature string) string {
	return "_" + feature + " not supported by session store._"
}

func CLIWelcomeHintZH() string {
	return "输入 /help 查看可用命令，/quit 退出。"
}

func UnknownCommandHintZH() string {
	return "输入 /help 查看命令"
}

func UnknownCommandMessageZH(cmd string) string {
	return "未知命令: " + cmd + "（" + UnknownCommandHintZH() + "）"
}

func SessionStoreUnavailableEN() string {
	return "session store unavailable"
}

func SessionStoreNotSupportedZH(feature string) string {
	return "当前会话存储不支持" + feature + "。"
}

func CLICancelNotSupportedZH() string {
	return "当前 CLI 模式不支持 /cancel；请使用 Ctrl+C 中断当前轮。"
}

func NotFoundEN(kind, name string) string {
	return kind + " not found: " + name
}

func PendingApprovalNotFoundZH() string {
	return "未找到待处理审批"
}

func UsageZHOptionalN(prefix string) string {
	return UsageZH(prefix + " [n]")
}

func UsageZHOptionalNPositive(prefix string) string {
	return UsageZH(prefix+" [n]") + "（n 必须是正整数）"
}

func UsageZHRequiredIndex(prefix string) string {
	return UsageZH(prefix + " <index>")
}

func UsageZHRequiredIndexPositive(prefix string) string {
	return UsageZH(prefix+" <index>") + "（index 必须是正整数）"
}

func UsageZHActionIndexRange(max int) string {
	return fmt.Sprintf("%s (1..%d)", UsageZH("/actions <index>"), max)
}
