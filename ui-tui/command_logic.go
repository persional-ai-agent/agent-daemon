package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	appconfig "github.com/dingjingmaster/agent-daemon/internal/config"
	"github.com/gorilla/websocket"
)

func handleTUICommand(s *appState, text string, onEvent func(map[string]any), onLine func(string)) (lines []string, err error, quit bool) {
	queue := []string{text}
	for len(queue) > 0 {
		current := strings.TrimSpace(queue[0])
		queue = queue[1:]
		if current == "" {
			continue
		}
		current = canonicalInput(current)
		s.appendHistory(current)

		emit := func(msg string) {
			lines = append(lines, msg)
			if onLine != nil {
				onLine(msg)
			}
		}
		emitData := func(v any) {
			s.lastJSON = v
			switch data := v.(type) {
			case []map[string]any:
				if s.viewMode == "human" {
					for i, row := range data {
						emit(fmt.Sprintf("%d. %v", i+1, row))
					}
					return
				}
			case map[string]any:
				_ = data
			}
			bs, mErr := json.MarshalIndent(v, "", "  ")
			if !s.pretty || mErr != nil {
				bs, _ = json.Marshal(v)
			}
			for _, ln := range strings.Split(string(bs), "\n") {
				emit(ln)
			}
		}

		switch {
		case current == "/quit" || current == "/exit":
			return lines, nil, true
		case current == "/help":
			for _, line := range helpLines() {
				emit(line)
			}
			s.setStatus(true, "ok", "help shown")
		case strings.HasPrefix(current, "/actions"):
			parts := strings.Fields(current)
			items := actionMenuItems(s)
			if len(parts) == 1 {
				for i, cmd := range items {
					emit(fmt.Sprintf("%d. %s", i+1, cmd))
				}
				emit("use: /actions <index>")
				s.setStatus(true, "ok", "actions listed")
				continue
			}
			idx, pErr := strconv.Atoi(parts[1])
			if pErr != nil || idx <= 0 || idx > len(items) {
				return lines, fmt.Errorf("用法: /actions <index> (1..%d)", len(items)), false
			}
			next, ok := actionCommandByIndex(s, idx)
			if !ok {
				return lines, fmt.Errorf("invalid action index"), false
			}
			emit("run action: " + next)
			queue = append([]string{next}, queue...)
			s.setStatus(true, "ok", "action selected")
		case current == "/clear":
			s.setStatus(true, "ok", "cleared")
		case current == "/refresh":
			if rErr := s.refreshCurrentPanel(); rErr != nil {
				s.setErrStatus(rErr)
				return lines, rErr, false
			}
			emit("panel refreshed: " + s.fullscreenPanel)
			s.setStatus(true, "ok", "panel refreshed")
		case strings.HasPrefix(current, "/panel"):
			parts := strings.Fields(current)
			if len(parts) == 1 {
				emit("panel: " + s.fullscreenPanel)
				s.setStatus(true, "ok", "panel shown")
				continue
			}
			target := strings.ToLower(strings.TrimSpace(parts[1]))
			if target == "status" {
				emitData(map[string]any{"panel": s.fullscreenPanel, "auto_refresh": s.panelAutoRefresh, "refresh_interval_sec": s.panelRefreshSec, "last_refresh_at": s.lastPanelRefresh.Format(time.RFC3339), "available_panel_names": panelNames()})
				s.setStatus(true, "ok", "panel status shown")
				continue
			}
			if target == "auto" {
				if len(parts) < 3 {
					return lines, fmt.Errorf("用法: /panel auto on|off"), false
				}
				switch strings.ToLower(strings.TrimSpace(parts[2])) {
				case "on":
					s.panelAutoRefresh = true
				case "off":
					s.panelAutoRefresh = false
				default:
					return lines, fmt.Errorf("用法: /panel auto on|off"), false
				}
				_ = s.saveRuntimeState()
				emit(fmt.Sprintf("panel auto refresh: %v", s.panelAutoRefresh))
				s.setStatus(true, "ok", "panel auto updated")
				continue
			}
			if target == "interval" {
				if len(parts) < 3 {
					return lines, fmt.Errorf("用法: /panel interval <sec>"), false
				}
				sec, cErr := strconv.Atoi(parts[2])
				if cErr != nil || sec < 1 || sec > 300 {
					return lines, fmt.Errorf("panel interval must be 1..300 seconds"), false
				}
				s.panelRefreshSec = sec
				_ = s.saveRuntimeState()
				emit(fmt.Sprintf("panel refresh interval: %ds", s.panelRefreshSec))
				s.setStatus(true, "ok", "panel interval updated")
				continue
			}
			if target == "list" {
				emit("panels: " + strings.Join(panelNames(), ", "))
				s.setStatus(true, "ok", "panel list shown")
				continue
			}
			switch target {
			case "next":
				s.fullscreenPanel = nextPanel(s.fullscreenPanel)
			case "prev":
				s.fullscreenPanel = prevPanel(s.fullscreenPanel)
			default:
				valid := false
				for _, n := range panelNames() {
					if n == target {
						valid = true
						break
					}
				}
				if !valid {
					return lines, fmt.Errorf("用法: /panel [overview|dashboard|sessions|tools|approvals|gateway|diag|next|prev]"), false
				}
				s.fullscreenPanel = target
			}
			_ = s.refreshCurrentPanel()
			_ = s.saveRuntimeState()
			emit("panel switched: " + s.fullscreenPanel)
			s.setStatus(true, "ok", "panel switched")
		case current == "/version":
			emitData(map[string]any{"version": BuildVersion, "commit": BuildCommit, "build_time": BuildTime})
			s.setStatus(true, "ok", "version shown")
		case current == "/doctor":
			items, allOK := s.runDoctor()
			emitData(map[string]any{"checks": items, "ok": allOK})
			if allOK {
				s.setStatus(true, "ok", "doctor checks passed")
			} else {
				s.setStatus(false, "doctor_failed", "doctor checks found failures")
			}
		case strings.HasPrefix(current, "/view "):
			mode := strings.ToLower(strings.TrimSpace(strings.TrimPrefix(current, "/view ")))
			if mode != "human" && mode != "json" {
				return lines, fmt.Errorf("用法: /view human|json"), false
			}
			s.viewMode = mode
			emit("view mode: " + mode)
			s.setStatus(true, "ok", "view mode switched")
		case current == "/reload-config":
			cfg := appconfig.Load()
			s.applyConfig(cfg)
			_ = s.saveRuntimeState()
			emitData(map[string]any{
				"ws_base":                 s.wsBase,
				"http_base":               s.httpBase,
				"ws_read_timeout_seconds": int(s.wsReadTimeout / time.Second),
				"ws_turn_timeout_seconds": int(s.wsTurnTimeout / time.Second),
				"ws_reconnect_max":        s.wsMaxReconnect,
				"history_max_lines":       s.historyMaxLines,
				"event_max_items":         s.eventMaxItems,
			})
			s.setStatus(true, "ok", "config reloaded")
		case current == "/status":
			emit(fmt.Sprintf("status=%s code=%s detail=%s", s.lastStatus, s.lastCode, s.lastDetail))
			emit(fmt.Sprintf("reconnect enabled=%v state=%s max=%d timeout_action=%s", s.reconnectEnabled, s.reconnectState, s.wsMaxReconnect, s.timeoutAction))
		case current == "/fullscreen":
			emit(fmt.Sprintf("fullscreen: %v", s.fullscreen))
			s.setStatus(true, "ok", "fullscreen status shown")
		case current == "/fullscreen on":
			s.fullscreen = true
			_ = s.saveRuntimeState()
			emit("fullscreen: on")
			s.setStatus(true, "ok", "fullscreen enabled")
		case current == "/fullscreen off":
			s.fullscreen = false
			_ = s.saveRuntimeState()
			emit("fullscreen: off")
			s.setStatus(true, "ok", "fullscreen disabled")
		case current == "/reconnect status":
			emitData(map[string]any{
				"enabled":         s.reconnectEnabled,
				"state":           s.reconnectState,
				"max_reconnect":   s.wsMaxReconnect,
				"timeout_action":  s.timeoutAction,
				"reconnect_count": s.reconnectCount,
				"fallback_hint":   s.fallbackHint,
				"last_error_code": s.lastErrorCode,
			})
			s.setStatus(true, "ok", "reconnect status shown")
		case current == "/diag":
			emitData(s.diagnosticsSnapshot())
			s.setStatus(true, "ok", "diagnostics shown")
		case current == "/diag export" || strings.HasPrefix(current, "/diag export "):
			path := strings.TrimSpace(strings.TrimPrefix(current, "/diag export"))
			if path == "" {
				return lines, fmt.Errorf("用法: /diag export <file>"), false
			}
			if dErr := s.exportDiagnostics(path); dErr != nil {
				s.setErrStatus(dErr)
				return lines, dErr, false
			}
			emit("diagnostics exported: " + path)
			s.setStatus(true, "ok", "diagnostics exported")
		case current == "/reconnect on":
			s.reconnectEnabled = true
			emit("reconnect: on")
			s.setStatus(true, "ok", "reconnect enabled")
		case current == "/reconnect off":
			s.reconnectEnabled = false
			emit("reconnect: off")
			s.setStatus(true, "ok", "reconnect disabled")
		case current == "/reconnect now":
			conn, _, dErr := websocket.DefaultDialer.Dial(s.wsBase, nil)
			if dErr != nil {
				s.reconnectState = "failed"
				s.setErrStatus(dErr)
				return lines, dErr, false
			}
			_ = conn.Close()
			s.reconnectState = "connecting"
			emit("reconnect probe ok")
			s.setStatus(true, "ok", "reconnect probe ok")
		case strings.HasPrefix(current, "/reconnect timeout "):
			mode := strings.TrimSpace(strings.TrimPrefix(current, "/reconnect timeout "))
			if mode != "wait" && mode != "reconnect" && mode != "cancel" {
				return lines, fmt.Errorf("用法: /reconnect timeout wait|reconnect|cancel"), false
			}
			s.timeoutAction = mode
			emit("reconnect timeout action: " + mode)
			s.setStatus(true, "ok", "reconnect timeout action updated")
		case current == "/health":
			out, hErr := httpJSON(http.MethodGet, s.httpBase+"/health", nil)
			if hErr != nil {
				s.setErrStatus(hErr)
				return lines, hErr, false
			}
			emitData(uiPayload(out, "status", "result"))
			s.setStatus(true, "ok", "health checked")
		case current == "/cancel":
			out, cErr := httpJSON(http.MethodPost, s.httpBase+"/v1/chat/cancel", map[string]any{"session_id": s.session})
			if cErr != nil {
				s.setErrStatus(cErr)
				return lines, cErr, false
			}
			s.audit("cancel", "requested")
			emitData(uiPayload(out, "result"))
			s.setStatus(true, "ok", "cancel requested")
		case strings.HasPrefix(current, "/history"):
			limit, pErr := parseOptionalPositiveIntArg(current, "/history", 20)
			if pErr != nil {
				return lines, pErr, false
			}
			if s.historyMaxLines > 0 && limit > s.historyMaxLines {
				limit = s.historyMaxLines
			}
			items, rErr := s.readHistory(limit)
			if rErr != nil {
				s.setErrStatus(rErr)
				return lines, rErr, false
			}
			for i, it := range items {
				emit(fmt.Sprintf("%d. %s", i+1, it))
			}
			s.setStatus(true, "ok", "history loaded")
		case strings.HasPrefix(current, "/timeline"):
			limit, pErr := parseOptionalPositiveIntArg(current, "/timeline", 20)
			if pErr != nil {
				return lines, pErr, false
			}
			items := s.timelineSlice(limit)
			if len(items) == 0 {
				emit("timeline empty")
			} else {
				for i, it := range items {
					emit(fmt.Sprintf("%d. %s", i+1, it))
				}
			}
			s.setStatus(true, "ok", "timeline listed")
		case current == "/rerun":
			return lines, fmt.Errorf("用法: /rerun <index>"), false
		case strings.HasPrefix(current, "/rerun "):
			idx, pErr := parseRequiredPositiveIntArg(current, "/rerun")
			if pErr != nil {
				return lines, pErr, false
			}
			items, rErr := s.readHistory(500)
			if rErr != nil {
				s.setErrStatus(rErr)
				return lines, rErr, false
			}
			for n := len(items); n > 0 && strings.TrimSpace(items[n-1]) == current; n = len(items) {
				items = items[:n-1]
			}
			if len(items) == 0 {
				return lines, fmt.Errorf("no history available"), false
			}
			if idx > len(items) {
				return lines, fmt.Errorf("index out of range, max=%d", len(items)), false
			}
			queue = append([]string{items[idx-1]}, queue...)
			emit("rerun: " + items[idx-1])
			s.setStatus(true, "ok", "rerun selected")
		case strings.HasPrefix(current, "/events"):
			if strings.HasPrefix(current, "/events save ") {
				path, format, since, until, pErr := parseEventSaveArgs(current)
				if pErr != nil {
					return lines, pErr, false
				}
				filtered := filterEventsByTime(s.eventLog, since, until)
				if svErr := saveEvents(path, format, filtered); svErr != nil {
					s.setErrStatus(svErr)
					return lines, svErr, false
				}
				emit(fmt.Sprintf("saved events: %s (format=%s count=%d)", path, format, len(filtered)))
				s.setStatus(true, "ok", "events saved")
				continue
			}
			limit, pErr := parseOptionalPositiveIntArg(current, "/events", 20)
			if pErr != nil {
				return lines, pErr, false
			}
			if limit > s.eventMaxItems {
				limit = s.eventMaxItems
			}
			start := len(s.eventLog) - limit
			if start < 0 {
				start = 0
			}
			emitData(s.eventLog[start:])
			s.setStatus(true, "ok", "events listed")
		case current == "/approve" || strings.HasPrefix(current, "/approve "):
			approve := true
			id := strings.TrimSpace(strings.TrimPrefix(current, "/approve"))
			if id == "" {
				out, hErr := httpJSON(http.MethodGet, fmt.Sprintf("%s/v1/ui/sessions/%s?offset=0&limit=200", s.httpBase, url.PathEscape(s.session)), nil)
				if hErr != nil {
					s.setErrStatus(hErr)
					return lines, hErr, false
				}
				msgs, _ := out["messages"].([]any)
				lastID, _ := findLatestPendingApproval(msgs)
				if lastID == "" {
					return lines, fmt.Errorf("未找到待处理审批；用法: /approve <approval_id>"), false
				}
				id = lastID
			}
			out, aErr := s.confirmApproval(id, approve)
			if aErr != nil {
				s.setErrStatus(aErr)
				return lines, aErr, false
			}
			s.audit("approve", "approval_id="+id)
			s.setStatus(true, "ok", "approval confirmed")
			emitData(uiPayload(out, "result"))
		case current == "/deny" || strings.HasPrefix(current, "/deny "):
			approve := false
			id := strings.TrimSpace(strings.TrimPrefix(current, "/deny"))
			if id == "" {
				out, hErr := httpJSON(http.MethodGet, fmt.Sprintf("%s/v1/ui/sessions/%s?offset=0&limit=200", s.httpBase, url.PathEscape(s.session)), nil)
				if hErr != nil {
					s.setErrStatus(hErr)
					return lines, hErr, false
				}
				msgs, _ := out["messages"].([]any)
				lastID, _ := findLatestPendingApproval(msgs)
				if lastID == "" {
					return lines, fmt.Errorf("未找到待处理审批；用法: /deny <approval_id>"), false
				}
				id = lastID
			}
			out, aErr := s.confirmApproval(id, approve)
			if aErr != nil {
				s.setErrStatus(aErr)
				return lines, aErr, false
			}
			s.audit("deny", "approval_id="+id)
			s.setStatus(true, "ok", "approval denied")
			emitData(uiPayload(out, "result"))
		case strings.HasPrefix(current, "/pending"):
			limit, action, actionIndex, pErr := parsePendingArgs(current)
			if pErr != nil {
				return lines, pErr, false
			}
			out, hErr := httpJSON(http.MethodGet, fmt.Sprintf("%s/v1/ui/sessions/%s?offset=0&limit=200", s.httpBase, url.PathEscape(s.session)), nil)
			if hErr != nil {
				s.setErrStatus(hErr)
				return lines, hErr, false
			}
			msgs, _ := out["messages"].([]any)
			items := findPendingApprovals(msgs, limit)
			if len(items) == 0 {
				return lines, fmt.Errorf("未找到待处理审批"), false
			}
			s.pendingCache = items
			if limit <= 1 {
				emit(fmt.Sprintf("pending approval id: %v", items[0]["approval_id"]))
				emitData(items[0])
			} else {
				emitData(items)
			}
			if action != "" {
				if action != "approve" && action != "deny" && action != "a" && action != "d" {
					return lines, fmt.Errorf("用法: /pending [limit] [approve|deny|a|d <index>]"), false
				}
				if actionIndex <= 0 || actionIndex > len(items) {
					return lines, fmt.Errorf("待处理审批索引越界，最大值=%d", len(items)), false
				}
				chosen := items[actionIndex-1]
				id, _ := chosen["approval_id"].(string)
				if strings.TrimSpace(id) == "" {
					return lines, fmt.Errorf("invalid approval id at index %d", actionIndex), false
				}
				approve := action == "approve" || action == "a"
				actionCmd := "/deny "
				if approve {
					actionCmd = "/approve "
				}
				queue = append([]string{actionCmd + id}, queue...)
			}
			s.setStatus(true, "ok", "pending approval found")
		case current == "/bookmark" || strings.HasPrefix(current, "/bookmark "):
			parts := strings.Fields(current)
			if len(parts) >= 2 && parts[1] == "list" {
				list, bErr := s.loadBookmarks()
				if bErr != nil {
					s.setErrStatus(bErr)
					return lines, bErr, false
				}
				emitData(list)
				s.setStatus(true, "ok", "bookmarks listed")
				continue
			}
			if len(parts) >= 3 && parts[1] == "add" {
				if bErr := s.addBookmark(parts[2]); bErr != nil {
					s.setErrStatus(bErr)
					return lines, bErr, false
				}
				emit("bookmark saved: " + parts[2])
				s.setStatus(true, "ok", "bookmark saved")
				continue
			}
			if len(parts) >= 3 && parts[1] == "use" {
				if bErr := s.useBookmark(parts[2]); bErr != nil {
					s.setErrStatus(bErr)
					return lines, bErr, false
				}
				_ = s.saveRuntimeState()
				emit(fmt.Sprintf("bookmark loaded: %s (session=%s)", parts[2], s.session))
				s.setStatus(true, "ok", "bookmark loaded")
				continue
			}
			return lines, fmt.Errorf("用法: /bookmark add <name> | /bookmark list | /bookmark use <name>"), false
		case current == "/workbench" || strings.HasPrefix(current, "/workbench "):
			parts := strings.Fields(current)
			if len(parts) < 2 {
				return lines, fmt.Errorf("用法: /workbench save|list|load|delete ..."), false
			}
			sub := parts[1]
			name := ""
			if len(parts) > 2 {
				name = strings.TrimSpace(parts[2])
			}
			switch sub {
			case "list":
				list, wErr := s.loadWorkbenchProfiles()
				if wErr != nil {
					s.setErrStatus(wErr)
					return lines, wErr, false
				}
				emitData(map[string]any{"count": len(list), "workbenches": list})
				s.setStatus(true, "ok", "workbench listed")
			case "save":
				if name == "" {
					return lines, fmt.Errorf("用法: /workbench save <name>"), false
				}
				if wErr := s.saveWorkbench(name); wErr != nil {
					s.setErrStatus(wErr)
					return lines, wErr, false
				}
				emit("workbench saved: " + name)
				s.audit("workbench_save", "name="+name)
				s.setStatus(true, "ok", "workbench saved")
			case "load":
				if name == "" {
					return lines, fmt.Errorf("用法: /workbench load <name>"), false
				}
				if wErr := s.loadWorkbench(name); wErr != nil {
					s.setErrStatus(wErr)
					return lines, wErr, false
				}
				_ = s.saveRuntimeState()
				_ = s.refreshCurrentPanel()
				emit("workbench loaded: " + name)
				s.audit("workbench_load", "name="+name)
				s.setStatus(true, "ok", "workbench loaded")
			case "delete":
				if name == "" {
					return lines, fmt.Errorf("用法: /workbench delete <name>"), false
				}
				if wErr := s.deleteWorkbench(name); wErr != nil {
					s.setErrStatus(wErr)
					return lines, wErr, false
				}
				emit("workbench deleted: " + name)
				s.audit("workbench_delete", "name="+name)
				s.setStatus(true, "ok", "workbench deleted")
			default:
				return lines, fmt.Errorf("用法: /workbench save|list|load|delete ..."), false
			}
		case current == "/workflow" || strings.HasPrefix(current, "/workflow "):
			parts := strings.Fields(current)
			if len(parts) < 2 {
				return lines, fmt.Errorf("用法: /workflow save|list|run|delete ..."), false
			}
			sub := parts[1]
			switch sub {
			case "list":
				list, wErr := s.loadWorkflows()
				if wErr != nil {
					s.setErrStatus(wErr)
					return lines, wErr, false
				}
				emitData(map[string]any{"count": len(list), "workflows": list})
				s.setStatus(true, "ok", "workflow listed")
			case "save":
				if len(parts) < 4 {
					return lines, fmt.Errorf("用法: /workflow save <name> <cmd1;cmd2;...>"), false
				}
				name := strings.TrimSpace(parts[2])
				raw := strings.TrimSpace(strings.TrimPrefix(current, "/workflow save "+name))
				cmds := parseWorkflowCommands(raw)
				if len(cmds) == 0 {
					return lines, fmt.Errorf("workflow commands empty"), false
				}
				if wErr := s.saveWorkflow(name, cmds); wErr != nil {
					s.setErrStatus(wErr)
					return lines, wErr, false
				}
				emit(fmt.Sprintf("workflow saved: %s (%d commands)", name, len(cmds)))
				s.audit("workflow_save", fmt.Sprintf("name=%s count=%d", name, len(cmds)))
				s.setStatus(true, "ok", "workflow saved")
			case "run":
				if len(parts) < 3 {
					return lines, fmt.Errorf("用法: /workflow run <name> [dry]"), false
				}
				name := strings.TrimSpace(parts[2])
				wf, ok, wErr := s.getWorkflow(name)
				if wErr != nil {
					s.setErrStatus(wErr)
					return lines, wErr, false
				}
				if !ok {
					return lines, fmt.Errorf("workflow not found: %s", name), false
				}
				dry := len(parts) > 3 && strings.EqualFold(strings.TrimSpace(parts[3]), "dry")
				if dry {
					emitData(map[string]any{"name": wf.Name, "commands": wf.Commands, "dry_run": true})
					s.setStatus(true, "ok", "workflow dry-run shown")
					continue
				}
				queue = append(queue, wf.Commands...)
				emit(fmt.Sprintf("workflow queued: %s (%d commands)", wf.Name, len(wf.Commands)))
				s.audit("workflow_run", fmt.Sprintf("name=%s count=%d", wf.Name, len(wf.Commands)))
				s.setStatus(true, "ok", "workflow queued")
			case "delete":
				if len(parts) < 3 {
					return lines, fmt.Errorf("用法: /workflow delete <name>"), false
				}
				name := strings.TrimSpace(parts[2])
				if wErr := s.deleteWorkflow(name); wErr != nil {
					s.setErrStatus(wErr)
					return lines, wErr, false
				}
				emit("workflow deleted: " + name)
				s.audit("workflow_delete", "name="+name)
				s.setStatus(true, "ok", "workflow deleted")
			default:
				return lines, fmt.Errorf("用法: /workflow save|list|run|delete ..."), false
			}
		case current == "/session":
			emit("session: " + s.session)
			s.setStatus(true, "ok", "session shown")
		case strings.HasPrefix(current, "/session "):
			next := strings.TrimSpace(strings.TrimPrefix(current, "/session "))
			if next == "" {
				return lines, fmt.Errorf("session id required"), false
			}
			s.session = next
			_ = s.saveRuntimeState()
			emit("session switched: " + s.session)
			s.setStatus(true, "ok", "session switched")
		case current == "/api":
			emit("ws: " + s.wsBase)
			s.setStatus(true, "ok", "ws shown")
		case strings.HasPrefix(current, "/api "):
			next := strings.TrimSpace(strings.TrimPrefix(current, "/api "))
			if !strings.HasPrefix(next, "ws://") && !strings.HasPrefix(next, "wss://") {
				return lines, fmt.Errorf("API 地址必须以 ws:// 或 wss:// 开头"), false
			}
			s.wsBase = next
			if strings.TrimSpace(os.Getenv("AGENT_HTTP_BASE")) == "" {
				s.httpBase = deriveHTTPBase(s.wsBase)
			}
			_ = s.saveRuntimeState()
			emit("ws switched: " + s.wsBase)
			s.setStatus(true, "ok", "ws switched")
		case current == "/http":
			emit("http: " + s.httpBase)
			s.setStatus(true, "ok", "http shown")
		case strings.HasPrefix(current, "/http "):
			next := strings.TrimSpace(strings.TrimPrefix(current, "/http "))
			if !strings.HasPrefix(next, "http://") && !strings.HasPrefix(next, "https://") {
				return lines, fmt.Errorf("HTTP API 地址必须以 http:// 或 https:// 开头"), false
			}
			s.httpBase = strings.TrimRight(next, "/")
			_ = s.saveRuntimeState()
			emit("http switched: " + s.httpBase)
			s.setStatus(true, "ok", "http switched")
		case current == "/last":
			if s.lastJSON == nil {
				return lines, fmt.Errorf("no last json payload"), false
			}
			emitData(s.lastJSON)
			s.setStatus(true, "ok", "last json shown")
		case current == "/save" || strings.HasPrefix(current, "/save "):
			path := strings.TrimSpace(strings.TrimPrefix(current, "/save"))
			if path == "" {
				return lines, fmt.Errorf("用法: /save <file>"), false
			}
			if s.lastJSON == nil {
				return lines, fmt.Errorf("no last json payload"), false
			}
			bs, mErr := json.MarshalIndent(s.lastJSON, "", "  ")
			if !s.pretty || mErr != nil {
				bs, _ = json.Marshal(s.lastJSON)
			}
			if wErr := os.WriteFile(path, bs, 0o644); wErr != nil {
				s.setErrStatus(wErr)
				return lines, wErr, false
			}
			emit("saved: " + path)
			s.setStatus(true, "ok", "json saved")
		case current == "/pretty" || strings.HasPrefix(current, "/pretty "):
			mode := strings.TrimSpace(strings.TrimPrefix(current, "/pretty"))
			if mode == "on" {
				s.pretty = true
				emit("pretty json: on")
				s.setStatus(true, "ok", "pretty on")
			} else if mode == "off" {
				s.pretty = false
				emit("pretty json: off")
				s.setStatus(true, "ok", "pretty off")
			} else {
				return lines, fmt.Errorf("用法: /pretty on|off"), false
			}
		case current == "/tools":
			out, tErr := httpJSON(http.MethodGet, s.httpBase+"/v1/ui/tools", nil)
			if tErr != nil {
				s.setErrStatus(tErr)
				return lines, tErr, false
			}
			emitData(uiPayload(out, "tools", "result"))
			s.setStatus(true, "ok", "tools listed")
		case current == "/tool" || strings.HasPrefix(current, "/tool "):
			name := strings.TrimSpace(strings.TrimPrefix(current, "/tool"))
			if name == "" {
				return lines, fmt.Errorf("用法: /tool <name>"), false
			}
			out, tErr := httpJSON(http.MethodGet, s.httpBase+"/v1/ui/tools/"+url.PathEscape(name)+"/schema", nil)
			if tErr != nil {
				s.setErrStatus(tErr)
				return lines, tErr, false
			}
			emitData(uiPayload(out, "schema", "result"))
			s.setStatus(true, "ok", "tool schema loaded")
		case strings.HasPrefix(current, "/sessions"):
			limit, pick, pErr := parseSessionsArgs(current)
			if pErr != nil {
				return lines, pErr, false
			}
			out, hErr := httpJSON(http.MethodGet, fmt.Sprintf("%s/v1/ui/sessions?limit=%d", s.httpBase, limit), nil)
			if hErr != nil {
				s.setErrStatus(hErr)
				return lines, hErr, false
			}
			payload := uiPayload(out, "sessions", "result")
			s.lastSessions = s.lastSessions[:0]
			if rows, ok := payload.([]any); ok {
				for _, row := range rows {
					if m, ok := row.(map[string]any); ok {
						if sid, ok := m["session_id"].(string); ok && strings.TrimSpace(sid) != "" {
							s.lastSessions = append(s.lastSessions, sid)
						}
					}
				}
			}
			emitData(payload)
			s.setStatus(true, "ok", "sessions listed")
			if pick > 0 {
				if pick > len(s.lastSessions) {
					return lines, fmt.Errorf("index out of range, max=%d", len(s.lastSessions)), false
				}
				s.session = s.lastSessions[pick-1]
				s.lastShowSession = s.session
				s.lastShowOffset = 0
				_ = s.saveRuntimeState()
				emit("session switched: " + s.session)
			}
		case current == "/pick":
			return lines, fmt.Errorf("用法: /pick <index>"), false
		case strings.HasPrefix(current, "/pick "):
			idx, pErr := parseRequiredPositiveIntArg(current, "/pick")
			if pErr != nil {
				return lines, pErr, false
			}
			if idx > len(s.lastSessions) {
				return lines, fmt.Errorf("index out of range, max=%d", len(s.lastSessions)), false
			}
			s.session = s.lastSessions[idx-1]
			s.lastShowSession = s.session
			s.lastShowOffset = 0
			_ = s.saveRuntimeState()
			emit("session switched: " + s.session)
			s.setStatus(true, "ok", "session switched")
		case strings.HasPrefix(current, "/open "):
			idx, action, pErr := parseOpenArgs(current)
			if pErr != nil {
				return lines, pErr, false
			}
			payload := s.panelData[s.fullscreenPanel]
			switch s.fullscreenPanel {
			case "sessions":
				sid, ok := selectSessionIDFromPanelData(payload, idx)
				if !ok {
					return lines, fmt.Errorf("open failed: invalid session index"), false
				}
				s.session = sid
				s.lastShowSession = sid
				s.lastShowOffset = 0
				_ = s.saveRuntimeState()
				emit("session switched: " + sid)
				s.setStatus(true, "ok", "session opened")
			case "tools":
				name, ok := selectToolNameFromPanelData(payload, idx)
				if !ok {
					return lines, fmt.Errorf("open failed: invalid tool index"), false
				}
				out, hErr := httpJSON(http.MethodGet, s.httpBase+"/v1/ui/tools/"+url.PathEscape(name)+"/schema", nil)
				if hErr != nil {
					s.setErrStatus(hErr)
					return lines, hErr, false
				}
				emitData(uiPayload(out, "schema", "result"))
				s.setStatus(true, "ok", "tool opened")
			case "approvals":
				id, ok := selectApprovalIDFromPanelData(payload, idx)
				if !ok {
					return lines, fmt.Errorf("open failed: invalid approval index"), false
				}
				emit("selected pending approval: " + id)
				emit("use /approve " + id + " or /deny " + id)
				if action != "" {
					if action == "a" || action == "approve" {
						queue = append([]string{"/approve " + id}, queue...)
					} else if action == "d" || action == "deny" {
						queue = append([]string{"/deny " + id}, queue...)
					}
				}
				s.setStatus(true, "ok", "approval selected")
			default:
				return lines, fmt.Errorf("open is available for panels: sessions/tools/approvals"), false
			}
		case strings.HasPrefix(current, "/show"):
			sid, offset, limit, pick, pErr := parseShowArgs(current, s.session)
			if pErr != nil {
				return lines, pErr, false
			}
			s.lastShowSession = sid
			s.lastShowOffset = offset
			s.lastShowLimit = limit
			out, hErr := httpJSON(http.MethodGet, fmt.Sprintf("%s/v1/ui/sessions/%s?offset=%d&limit=%d", s.httpBase, url.PathEscape(sid), offset, limit), nil)
			if hErr != nil {
				s.setErrStatus(hErr)
				return lines, hErr, false
			}
			emitData(out)
			if msgs, ok := out["messages"].([]any); ok && len(msgs) > 0 {
				if pick > 0 {
					if pick > len(msgs) {
						return lines, fmt.Errorf("message index out of range, max=%d", len(msgs)), false
					}
					emit(fmt.Sprintf("selected message index: %d", pick))
				} else {
					emit("hint: /show <session> <offset> <limit> pick <index>")
				}
			}
			s.setStatus(true, "ok", "show loaded")
		case current == "/next":
			if strings.TrimSpace(s.lastShowSession) == "" || s.lastShowLimit <= 0 {
				return lines, fmt.Errorf("run /show first before /next"), false
			}
			s.lastShowOffset += s.lastShowLimit
			out, hErr := httpJSON(http.MethodGet, fmt.Sprintf("%s/v1/ui/sessions/%s?offset=%d&limit=%d", s.httpBase, url.PathEscape(s.lastShowSession), s.lastShowOffset, s.lastShowLimit), nil)
			if hErr != nil {
				s.lastShowOffset -= s.lastShowLimit
				s.setErrStatus(hErr)
				return lines, hErr, false
			}
			emitData(out)
			s.setStatus(true, "ok", "next page loaded")
		case current == "/prev":
			if strings.TrimSpace(s.lastShowSession) == "" || s.lastShowLimit <= 0 {
				return lines, fmt.Errorf("run /show first before /prev"), false
			}
			s.lastShowOffset -= s.lastShowLimit
			if s.lastShowOffset < 0 {
				s.lastShowOffset = 0
			}
			out, hErr := httpJSON(http.MethodGet, fmt.Sprintf("%s/v1/ui/sessions/%s?offset=%d&limit=%d", s.httpBase, url.PathEscape(s.lastShowSession), s.lastShowOffset, s.lastShowLimit), nil)
			if hErr != nil {
				s.setErrStatus(hErr)
				return lines, hErr, false
			}
			emitData(out)
			s.setStatus(true, "ok", "prev page loaded")
		case strings.HasPrefix(current, "/stats"):
			sid, pErr := parseStatsArgs(current, s.session)
			if pErr != nil {
				return lines, pErr, false
			}
			out, hErr := httpJSON(http.MethodGet, fmt.Sprintf("%s/v1/ui/sessions/%s?offset=0&limit=1", s.httpBase, url.PathEscape(sid)), nil)
			if hErr != nil {
				s.setErrStatus(hErr)
				return lines, hErr, false
			}
			emitData(uiPayload(out, "stats", "result"))
			s.setStatus(true, "ok", "stats loaded")
		case current == "/gateway" || strings.HasPrefix(current, "/gateway "):
			parts := strings.Fields(current)
			if len(parts) != 2 {
				return lines, fmt.Errorf("用法: /gateway status|enable|disable"), false
			}
			if parts[1] == "status" {
				out, hErr := httpJSON(http.MethodGet, s.httpBase+"/v1/ui/gateway/status", nil)
				if hErr != nil {
					s.setErrStatus(hErr)
					return lines, hErr, false
				}
				emitData(uiPayload(out, "status", "result"))
				s.setStatus(true, "ok", "gateway status loaded")
			} else if parts[1] == "enable" || parts[1] == "disable" {
				out, hErr := httpJSON(http.MethodPost, s.httpBase+"/v1/ui/gateway/action", map[string]any{"action": parts[1]})
				if hErr != nil {
					s.setErrStatus(hErr)
					return lines, hErr, false
				}
				emitData(uiPayload(out, "result"))
				s.setStatus(true, "ok", "gateway action applied")
			} else {
				return lines, fmt.Errorf("用法: /gateway status|enable|disable"), false
			}
		case current == "/config get":
			out, hErr := httpJSON(http.MethodGet, s.httpBase+"/v1/ui/config", nil)
			if hErr != nil {
				s.setErrStatus(hErr)
				return lines, hErr, false
			}
			emitData(uiPayload(out, "snapshot", "result"))
			s.setStatus(true, "ok", "config loaded")
		case current == "/config tui":
			cfg := appconfig.Load()
			src := map[string]string{"ws_base": "config.ini", "http_base": "config.ini", "view_mode": "config.ini"}
			if strings.TrimSpace(os.Getenv("AGENT_API_BASE")) != "" {
				src["ws_base"] = "env"
			}
			if strings.TrimSpace(os.Getenv("AGENT_HTTP_BASE")) != "" {
				src["http_base"] = "env"
			}
			if strings.TrimSpace(os.Getenv("AGENT_UI_TUI_VIEW_MODE")) != "" {
				src["view_mode"] = "env"
			}
			emitData(uiPayload(map[string]any{"effective": map[string]any{"ws_base": s.wsBase, "http_base": s.httpBase, "view_mode": s.viewMode, "ws_read_timeout_seconds": int(s.wsReadTimeout / time.Second), "ws_turn_timeout_seconds": int(s.wsTurnTimeout / time.Second), "ws_reconnect_max": s.wsMaxReconnect, "history_max_lines": s.historyMaxLines, "event_max_items": s.eventMaxItems, "auto_doctor": s.autoDoctor}, "configured": map[string]any{"ws_base": cfg.UITUIWSBase, "http_base": cfg.UITUIHTTPBase, "view_mode": cfg.UITUIViewMode}, "source": src}, "result"))
			s.setStatus(true, "ok", "ui-tui config shown")
		case current == "/config":
			return lines, fmt.Errorf("用法: /config get|set <section.key> <value>|tui"), false
		case strings.HasPrefix(current, "/config set "):
			parts := strings.SplitN(strings.TrimPrefix(current, "/config set "), " ", 2)
			if len(parts) != 2 || strings.TrimSpace(parts[0]) == "" {
				return lines, fmt.Errorf("用法: /config set <section.key> <value>"), false
			}
			key := strings.TrimSpace(parts[0])
			value := parts[1]
			out, hErr := httpJSON(http.MethodPost, s.httpBase+"/v1/ui/config/set", map[string]any{"key": key, "value": value})
			if hErr != nil {
				s.setErrStatus(hErr)
				return lines, hErr, false
			}
			emitData(out)
			s.audit("config_set", "key="+key)
			s.setStatus(true, "ok", "config updated")
		default:
			s.addChatLine("user: " + current)
			if onEvent == nil {
				onEvent = func(map[string]any) {}
			}
			wrapped := func(evt map[string]any) {
				onEvent(evt)
				line := printEvent(evt, false)
				if strings.TrimSpace(line) != "" {
					emit(line)
				}
			}
			if runErr := s.sendTurn(current, wrapped); runErr != nil {
				s.setErrStatus(runErr)
				return lines, runErr, false
			}
			s.setStatus(true, "ok", "chat turn finished")
		}
	}
	return lines, nil, false
}

func parseOptionalPositiveIntArg(input, prefix string, def int) (int, error) {
	parts := strings.Fields(strings.TrimSpace(input))
	if len(parts) <= 1 {
		return def, nil
	}
	if len(parts) > 2 {
		return 0, fmt.Errorf("用法: %s [n]", prefix)
	}
	v, err := strconv.Atoi(parts[1])
	if err != nil || v <= 0 {
		return 0, fmt.Errorf("用法: %s [n]（n 必须是正整数）", prefix)
	}
	return v, nil
}

func parseRequiredPositiveIntArg(input, prefix string) (int, error) {
	parts := strings.Fields(strings.TrimSpace(input))
	if len(parts) != 2 {
		return 0, fmt.Errorf("用法: %s <index>", prefix)
	}
	v, err := strconv.Atoi(parts[1])
	if err != nil || v <= 0 {
		return 0, fmt.Errorf("用法: %s <index>（index 必须是正整数）", prefix)
	}
	return v, nil
}

func parsePendingArgs(input string) (limit int, action string, actionIndex int, err error) {
	parts := strings.Fields(strings.TrimSpace(input))
	limit = 1
	if len(parts) > 4 {
		return 0, "", 0, fmt.Errorf("用法: /pending [limit] [approve|deny|a|d <index>]")
	}
	if len(parts) == 1 {
		return limit, "", 0, nil
	}
	pos := 1
	if v, convErr := strconv.Atoi(parts[pos]); convErr == nil {
		if v <= 0 {
			return 0, "", 0, fmt.Errorf("用法: /pending [limit] [approve|deny|a|d <index>]")
		}
		limit = v
		pos++
	}
	if pos >= len(parts) {
		return limit, "", 0, nil
	}
	action = strings.ToLower(strings.TrimSpace(parts[pos]))
	if action != "approve" && action != "deny" && action != "a" && action != "d" {
		return 0, "", 0, fmt.Errorf("用法: /pending [limit] [approve|deny|a|d <index>]")
	}
	pos++
	if pos >= len(parts) {
		return 0, "", 0, fmt.Errorf("用法: /pending [limit] [approve|deny|a|d <index>]")
	}
	actionIndex, err = strconv.Atoi(parts[pos])
	if err != nil || actionIndex <= 0 {
		return 0, "", 0, fmt.Errorf("用法: /pending [limit] [approve|deny|a|d <index>]")
	}
	return limit, action, actionIndex, nil
}

func parseSessionsArgs(input string) (limit int, pick int, err error) {
	parts := strings.Fields(strings.TrimSpace(input))
	limit = 20
	pick = 0
	if len(parts) == 1 {
		return limit, pick, nil
	}
	if len(parts) == 2 {
		if strings.EqualFold(parts[1], "pick") {
			return 0, 0, fmt.Errorf("用法: /sessions [limit] [pick <index>]")
		}
		v, convErr := strconv.Atoi(parts[1])
		if convErr != nil || v <= 0 {
			return 0, 0, fmt.Errorf("用法: /sessions [limit] [pick <index>]")
		}
		return v, 0, nil
	}
	if len(parts) == 3 {
		if !strings.EqualFold(parts[1], "pick") {
			return 0, 0, fmt.Errorf("用法: /sessions [limit] [pick <index>]")
		}
		idx, convErr := strconv.Atoi(parts[2])
		if convErr != nil || idx <= 0 {
			return 0, 0, fmt.Errorf("用法: /sessions [limit] [pick <index>]")
		}
		return limit, idx, nil
	}
	if len(parts) == 4 {
		v, convErr := strconv.Atoi(parts[1])
		if convErr != nil || v <= 0 || !strings.EqualFold(parts[2], "pick") {
			return 0, 0, fmt.Errorf("用法: /sessions [limit] [pick <index>]")
		}
		idx, idxErr := strconv.Atoi(parts[3])
		if idxErr != nil || idx <= 0 {
			return 0, 0, fmt.Errorf("用法: /sessions [limit] [pick <index>]")
		}
		return v, idx, nil
	}
	return 0, 0, fmt.Errorf("用法: /sessions [limit] [pick <index>]")
}

func parseShowArgs(input, defaultSession string) (sid string, offset, limit, pick int, err error) {
	parts := strings.Fields(strings.TrimSpace(input))
	sid = strings.TrimSpace(defaultSession)
	offset = 0
	limit = 20
	pick = 0
	usageErr := fmt.Errorf("用法: /show [session] [offset>=0] [limit>0] [pick <index>]")
	if len(parts) == 1 {
		return sid, offset, limit, pick, nil
	}
	if len(parts) >= 2 {
		sid = strings.TrimSpace(parts[1])
		if sid == "" {
			return "", 0, 0, 0, usageErr
		}
	}
	pos := 2
	if len(parts) > pos {
		v, convErr := strconv.Atoi(parts[pos])
		if convErr != nil || v < 0 {
			return "", 0, 0, 0, usageErr
		}
		offset = v
		pos++
	}
	if len(parts) > pos {
		v, convErr := strconv.Atoi(parts[pos])
		if convErr != nil || v <= 0 {
			return "", 0, 0, 0, usageErr
		}
		limit = v
		pos++
	}
	if len(parts) > pos {
		if len(parts) != pos+2 || !strings.EqualFold(parts[pos], "pick") {
			return "", 0, 0, 0, usageErr
		}
		idx, idxErr := strconv.Atoi(parts[pos+1])
		if idxErr != nil || idx <= 0 {
			return "", 0, 0, 0, usageErr
		}
		pick = idx
	}
	return sid, offset, limit, pick, nil
}

func parseStatsArgs(input, defaultSession string) (string, error) {
	parts := strings.Fields(strings.TrimSpace(input))
	if len(parts) == 1 {
		return strings.TrimSpace(defaultSession), nil
	}
	if len(parts) == 2 {
		sid := strings.TrimSpace(parts[1])
		if sid == "" {
			return "", fmt.Errorf("用法: /stats [session]")
		}
		return sid, nil
	}
	return "", fmt.Errorf("用法: /stats [session]")
}

func parseOpenArgs(input string) (index int, action string, err error) {
	parts := strings.Fields(strings.TrimSpace(input))
	if len(parts) < 2 || len(parts) > 3 {
		return 0, "", fmt.Errorf("用法: /open <index> [a|d|approve|deny]")
	}
	idx, convErr := strconv.Atoi(parts[1])
	if convErr != nil || idx <= 0 {
		return 0, "", fmt.Errorf("用法: /open <index> [a|d|approve|deny]")
	}
	if len(parts) == 2 {
		return idx, "", nil
	}
	action = strings.ToLower(strings.TrimSpace(parts[2]))
	if action != "a" && action != "d" && action != "approve" && action != "deny" {
		return 0, "", fmt.Errorf("用法: /open <index> [a|d|approve|deny]")
	}
	return idx, action, nil
}
