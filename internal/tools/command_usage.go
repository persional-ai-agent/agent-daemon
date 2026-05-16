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
)

func UsageEN(usage string) string { return "Usage: " + usage }
func UsageZH(usage string) string { return "用法: " + usage }
