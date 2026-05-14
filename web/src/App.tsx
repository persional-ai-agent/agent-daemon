import { useEffect, useState } from "react";
import {
  cancelChat,
  getUIConfig,
  getUIAgentDetail,
  getUIAgents,
  getUIAgentsActive,
  getUIAgentsHistory,
  getUICronJob,
  getUICronJobs,
  getUIModel,
  getUIModelProviders,
  getUIGatewayStatus,
  getUIGatewayDiagnostics,
  getUIPluginDashboards,
  getUISkillDetail,
  getUISkills,
  getUISessionDetail,
  getUISessions,
  postUIGatewayAction,
  postUIAgentInterrupt,
  postUICronJob,
  postUICronJobAction,
  postUISkillManage,
  postUISkillsReload,
  postUISkillsSearch,
  postUISkillsSync,
  getUIVoiceStatus,
  postUIVoiceRecord,
  postUIVoiceToggle,
  postUIVoiceTTS,
  setUIConfig,
  setUIModel,
  getUIToolSchema,
  getUITools,
  sendChat,
  streamChat,
  streamEventDedupeKey,
  normalizeAPIError,
  type StreamReconnectStatus,
  type StreamTransport,
  type StreamTimeoutAction,
  type StreamEvent,
  type UICronAction
} from "./lib/api";
import { buildDiagnosticsBundle } from "./lib/diagnostics";

type Tab = "chat" | "sessions" | "tools" | "skills" | "agents" | "cron" | "models" | "plugins" | "gateway" | "voice" | "config";

const tabs: Tab[] = ["chat", "sessions", "tools", "skills", "agents", "cron", "models", "plugins", "gateway", "voice", "config"];

export function App() {
  const [tab, setTab] = useState<Tab>("chat");
  const [sessionID, setSessionID] = useState("web-default");
  const [input, setInput] = useState("");
  const [output, setOutput] = useState("");
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState("");
  const [streamMode, setStreamMode] = useState(true);
  const [timeline, setTimeline] = useState<Array<{ ts: string; event: string; data: unknown }>>([]);
  const [streamStatus, setStreamStatus] = useState<StreamReconnectStatus>("connecting");
  const [streamTransport, setStreamTransport] = useState<StreamTransport>("ws");
  const [activeTransport, setActiveTransport] = useState<StreamTransport>("ws");
  const [lastTurnID, setLastTurnID] = useState("");
  const [reconnectCount, setReconnectCount] = useState(0);
  const [lastErrorCode, setLastErrorCode] = useState("ok");
  const [fallbackHint, setFallbackHint] = useState("");
  const [reconnectEnabled, setReconnectEnabled] = useState(true);
  const [reconnectMax, setReconnectMax] = useState(2);
  const [timeoutAction, setTimeoutAction] = useState<StreamTimeoutAction>("wait");
  const [readTimeoutSec, setReadTimeoutSec] = useState(15);
  const [turnTimeoutSec, setTurnTimeoutSec] = useState(120);
  const [sessions, setSessions] = useState<Array<{ session_id: string; last_seen?: string }>>([]);
  const [sessionDetail, setSessionDetail] = useState<Record<string, unknown> | null>(null);
  const [sessionOffset, setSessionOffset] = useState(0);
  const [sessionPageSize, setSessionPageSize] = useState(50);
  const [selectedSession, setSelectedSession] = useState("");
  const [tools, setTools] = useState<string[]>([]);
  const [toolSchema, setToolSchema] = useState<Record<string, unknown> | null>(null);
  const [toolFilter, setToolFilter] = useState("");
  const [selectedTool, setSelectedTool] = useState("");
  const [gatewayStatus, setGatewayStatus] = useState<Record<string, unknown> | null>(null);
  const [gatewayDiagnostics, setGatewayDiagnostics] = useState<Record<string, unknown> | null>(null);
  const [config, setConfig] = useState<Record<string, unknown> | null>(null);
  const [configKey, setConfigKey] = useState("api.type");
  const [configValue, setConfigValue] = useState("openai");
  const [modelInfo, setModelInfo] = useState<Record<string, unknown> | null>(null);
  const [modelProviders, setModelProviders] = useState<string[]>([]);
  const [modelProvider, setModelProvider] = useState("openai");
  const [modelName, setModelName] = useState("");
  const [modelBaseURL, setModelBaseURL] = useState("");
  const [modelSetResult, setModelSetResult] = useState<Record<string, unknown> | null>(null);
  const [skills, setSkills] = useState<Array<Record<string, unknown>>>([]);
  const [selectedSkill, setSelectedSkill] = useState("");
  const [skillContent, setSkillContent] = useState("");
  const [newSkillName, setNewSkillName] = useState("");
  const [skillSearchQuery, setSkillSearchQuery] = useState("");
  const [skillSearchRepo, setSkillSearchRepo] = useState("");
  const [skillSearchResult, setSkillSearchResult] = useState<Record<string, unknown> | null>(null);
  const [skillSyncSource, setSkillSyncSource] = useState<"url" | "github">("github");
  const [skillSyncURL, setSkillSyncURL] = useState("");
  const [skillSyncRepo, setSkillSyncRepo] = useState("");
  const [skillSyncPath, setSkillSyncPath] = useState("");
  const [agents, setAgents] = useState<Record<string, unknown> | null>(null);
  const [activeAgents, setActiveAgents] = useState<Record<string, unknown> | null>(null);
  const [agentHistory, setAgentHistory] = useState<Record<string, unknown> | null>(null);
  const [agentSessionID, setAgentSessionID] = useState("");
  const [agentDetail, setAgentDetail] = useState<Record<string, unknown> | null>(null);
  const [cronJobs, setCronJobs] = useState<Record<string, unknown> | null>(null);
  const [cronJobName, setCronJobName] = useState("");
  const [cronPrompt, setCronPrompt] = useState("");
  const [cronSchedule, setCronSchedule] = useState("every 30m");
  const [cronRepeat, setCronRepeat] = useState("");
  const [cronDeliveryTarget, setCronDeliveryTarget] = useState("");
  const [cronDeliverOn, setCronDeliverOn] = useState("always");
  const [cronContextMode, setCronContextMode] = useState("isolated");
  const [cronRunMode, setCronRunMode] = useState("agent");
  const [cronScriptCommand, setCronScriptCommand] = useState("");
  const [cronScriptCWD, setCronScriptCWD] = useState("");
  const [cronScriptTimeout, setCronScriptTimeout] = useState("");
  const [cronJobID, setCronJobID] = useState("");
  const [cronDetail, setCronDetail] = useState<Record<string, unknown> | null>(null);
  const [cronRuns, setCronRuns] = useState<Record<string, unknown> | null>(null);
  const [cronResult, setCronResult] = useState<Record<string, unknown> | null>(null);
  const [pluginDashboards, setPluginDashboards] = useState<Array<Record<string, unknown>>>([]);
  const [voiceStatus, setVoiceStatus] = useState<Record<string, unknown> | null>(null);
  const [voiceText, setVoiceText] = useState("");
  const [voiceResult, setVoiceResult] = useState<Record<string, unknown> | null>(null);

  async function onSend() {
    if (!input.trim()) return;
    setBusy(true);
    setError("");
    setTimeline([]);
    setStreamStatus("connecting");
    setActiveTransport(streamTransport);
    setFallbackHint("");
    setReconnectCount(0);
    const thisTurnID = `web-${Math.random().toString(36).slice(2)}`;
    setLastTurnID(thisTurnID);
    try {
      if (streamMode) {
        const seen = new Set<string>();
        await streamChat(input, sessionID, (evt: StreamEvent) => {
          const dedupeKey = streamEventDedupeKey(evt);
          if (seen.has(dedupeKey)) return;
          seen.add(dedupeKey);
          setTimeline((prev) => prev.concat([{ ts: new Date().toISOString(), event: evt.event, data: evt.data }]));
          if (evt.event === "transport_fallback" && evt.data && typeof evt.data === "object") {
            const d = evt.data as any;
            setFallbackHint(`fallback ${d.from}->${d.to} at ${d.at} (${d.reason})`);
          }
          if (evt.event === "result" && typeof evt.data === "object" && evt.data !== null) {
            const finalResponse = (evt.data as any).final_response;
            if (typeof finalResponse === "string") {
              setOutput(finalResponse || "(empty)");
            }
          }
        }, {
          transport: streamTransport,
          fallbackToSSE: true,
          reconnectEnabled,
          maxReconnect: reconnectMax,
          readTimeoutMs: Math.max(1, readTimeoutSec) * 1000,
          turnTimeoutMs: Math.max(1, turnTimeoutSec) * 1000,
          timeoutAction,
          onStatus: (s) => {
            setStreamStatus(s);
            if (s === "resumed" || s === "degraded") {
              setReconnectCount((n) => n + 1);
            }
          },
          onTransport: setActiveTransport
        });
      } else {
        const res = await sendChat(input, sessionID);
        setOutput(res.final_response || "(empty)");
      }
    } catch (e) {
      const n = normalizeAPIError(e instanceof Error ? e.message : String(e));
      setError(`[${n.code}] ${n.message}`);
      setLastErrorCode(n.code);
      setStreamStatus("failed");
    } finally {
      setBusy(false);
    }
  }

  async function onCancel() {
    setError("");
    try {
      await cancelChat(sessionID);
      setOutput("已发送取消请求。");
    } catch (e) {
      const n = normalizeAPIError(e instanceof Error ? e.message : String(e));
      setError(`[${n.code}] ${n.message}`);
    }
  }

  async function reconnectNow() {
    if (!streamMode || busy || !input.trim()) return;
    await onSend();
  }

  function exportDiagnostics() {
    const payload = buildDiagnosticsBundle({
      session_id: sessionID,
      turn_id: lastTurnID,
      stream_mode: streamMode,
      configured_transport: streamTransport,
      active_transport: activeTransport,
      reconnect_status: streamStatus,
      reconnect_count: reconnectCount,
      timeout_action: timeoutAction,
      read_timeout_sec: readTimeoutSec,
      turn_timeout_sec: turnTimeoutSec,
      fallback_hint: fallbackHint,
      last_error_code: lastErrorCode,
      error_text: error,
      events: timeline.map((t) => ({ ts: t.ts, event: t.event, data: t.data }))
    });
    const blob = new Blob([JSON.stringify(payload, null, 2)], { type: "application/json" });
    const href = URL.createObjectURL(blob);
    const a = document.createElement("a");
    a.href = href;
    a.download = `diagnostics-${sessionID}-${Date.now()}.json`;
    a.click();
    URL.revokeObjectURL(href);
  }

  useEffect(() => {
    if (tab === "sessions") {
      getUISessions(30)
        .then((r) => {
          setSessions(r.sessions);
          if (r.sessions.length > 0) {
            setSelectedSession(r.sessions[0].session_id);
            setSessionOffset(0);
            return getUISessionDetail(r.sessions[0].session_id, 0, sessionPageSize);
          }
          return null;
        })
        .then((detail) => setSessionDetail(detail))
        .catch((e) => setError(String(e)));
    }
    if (tab === "tools") {
      getUITools()
        .then((r) => {
          setTools(r.tools);
          if (r.tools.length > 0) {
            setSelectedTool(r.tools[0]);
            return getUIToolSchema(r.tools[0]);
          }
          return null;
        })
        .then((schema) => setToolSchema(schema))
        .catch((e) => setError(String(e)));
    }
    if (tab === "skills") {
      loadSkills().catch((e) => setError(String(e)));
    }
    if (tab === "agents") {
      loadAgents().catch((e) => setError(String(e)));
    }
    if (tab === "cron") {
      loadCronJobs().catch((e) => setError(String(e)));
    }
    if (tab === "models") {
      loadModelPage().catch((e) => setError(String(e)));
    }
    if (tab === "plugins") {
      getUIPluginDashboards()
        .then((r) => setPluginDashboards(r.dashboards))
        .catch((e) => setError(String(e)));
    }
    if (tab === "gateway") {
      Promise.all([getUIGatewayStatus(), getUIGatewayDiagnostics()])
        .then(([status, diagnostics]) => {
          setGatewayStatus(status);
          setGatewayDiagnostics(diagnostics);
        })
        .catch((e) => setError(String(e)));
    }
    if (tab === "voice") {
      refreshVoice().catch((e) => setError(String(e)));
    }
    if (tab === "config") {
      getUIConfig().then(setConfig).catch((e) => setError(String(e)));
    }
  }, [tab]);

  function pickSession(id: string) {
    setSelectedSession(id);
    setSessionOffset(0);
    getUISessionDetail(id, 0, sessionPageSize).then(setSessionDetail).catch((e) => setError(String(e)));
  }

  function changeSessionPage(nextOffset: number) {
    if (!selectedSession) return;
    const offset = Math.max(0, nextOffset);
    setSessionOffset(offset);
    getUISessionDetail(selectedSession, offset, sessionPageSize).then(setSessionDetail).catch((e) => setError(String(e)));
  }

  function pickTool(name: string) {
    setSelectedTool(name);
    getUIToolSchema(name).then(setToolSchema).catch((e) => setError(String(e)));
  }

  async function applyConfig() {
    setError("");
    try {
      await setUIConfig(configKey, configValue);
      const next = await getUIConfig();
      setConfig(next);
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e));
    }
  }

  async function loadModelPage() {
    const [info, providers] = await Promise.all([getUIModel(), getUIModelProviders()]);
    setModelInfo(info);
    const model = (info.model as Record<string, unknown> | undefined) || {};
    const providerList = Array.isArray(providers.providers) ? providers.providers.map(String) : [];
    setModelProviders(providerList);
    setModelProvider(String(model.provider || providerList[0] || "openai"));
    setModelName(String(model.model || ""));
    setModelBaseURL(String(model.base_url || ""));
  }

  async function applyModel() {
    setError("");
    try {
      const res = await setUIModel(modelProvider, modelName, modelBaseURL);
      setModelSetResult(res);
      await loadModelPage();
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e));
    }
  }

  async function gatewayAction(action: "enable" | "disable") {
    setError("");
    try {
      const res = await postUIGatewayAction(action);
      setGatewayStatus((res.status as Record<string, unknown>) || res);
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e));
    }
  }

  async function loadSkills() {
    const res = await getUISkills();
    setSkills(res.skills);
    if (!selectedSkill && res.skills.length > 0) {
      const name = String(res.skills[0].name || "");
      if (name) await pickSkill(name);
    }
  }

  async function pickSkill(name: string) {
    setSelectedSkill(name);
    const detail = await getUISkillDetail(name);
    const skill = detail.skill as Record<string, unknown> | undefined;
    setSkillContent(String(skill?.content || ""));
  }

  async function saveSkill() {
    setError("");
    const name = selectedSkill || newSkillName;
    if (!name.trim()) return;
    try {
      await postUISkillManage({
        action: selectedSkill ? "edit" : "create",
        name,
        content: skillContent
      });
      setSelectedSkill(name);
      setNewSkillName("");
      await loadSkills();
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e));
    }
  }

  async function createSkillDraft() {
    const name = newSkillName.trim();
    if (!name) return;
    setSelectedSkill("");
    setSkillContent(`# ${name}\n\n## 触发条件\n\n\n## 工作流\n\n`);
  }

  async function deleteSkill() {
    if (!selectedSkill || !window.confirm(`Delete skill ${selectedSkill}?`)) return;
    setError("");
    try {
      await postUISkillManage({ action: "delete", name: selectedSkill });
      setSelectedSkill("");
      setSkillContent("");
      await loadSkills();
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e));
    }
  }

  async function searchSkills() {
    if (!skillSearchQuery.trim()) return;
    setError("");
    try {
      setSkillSearchResult(await postUISkillsSearch(skillSearchQuery, skillSearchRepo || undefined));
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e));
    }
  }

  async function syncSkill() {
    const name = (selectedSkill || newSkillName).trim();
    if (!name) return;
    setError("");
    try {
      const payload = skillSyncSource === "url"
        ? { name, source: "url" as const, url: skillSyncURL }
        : { name, source: "github" as const, repo: skillSyncRepo, path: skillSyncPath };
      setSkillSearchResult(await postUISkillsSync(payload));
      await loadSkills();
      await pickSkill(name);
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e));
    }
  }

  async function loadAgents() {
    const [all, active, history] = await Promise.all([
      getUIAgents(30),
      getUIAgentsActive(),
      getUIAgentsHistory(30)
    ]);
    setAgents(all as Record<string, unknown>);
    setActiveAgents(active as Record<string, unknown>);
    setAgentHistory(history as Record<string, unknown>);
  }

  async function loadAgentDetail() {
    if (!agentSessionID.trim()) return;
    setError("");
    try {
      setAgentDetail(await getUIAgentDetail(agentSessionID));
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e));
    }
  }

  async function interruptAgent() {
    if (!agentSessionID.trim()) return;
    setError("");
    try {
      setAgentDetail(await postUIAgentInterrupt(agentSessionID));
      await loadAgents();
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e));
    }
  }

  async function loadCronJobs() {
    setCronJobs(await getUICronJobs());
  }

  async function createCronJob() {
    setError("");
    try {
      const payload: {
        name: string;
        prompt: string;
        schedule: string;
        repeat?: number;
        delivery_target?: string;
        deliver_on?: string;
        context_mode?: string;
        run_mode?: string;
        script_command?: string;
        script_cwd?: string;
        script_timeout?: number;
      } = {
        name: cronJobName,
        prompt: cronPrompt,
        schedule: cronSchedule,
        context_mode: cronContextMode,
        run_mode: cronRunMode
      };
      const repeat = Number(cronRepeat);
      if (cronRepeat.trim() && Number.isFinite(repeat) && repeat > 0) {
        payload.repeat = repeat;
      }
      if (cronDeliveryTarget.trim()) {
        payload.delivery_target = cronDeliveryTarget.trim();
        payload.deliver_on = cronDeliverOn;
      }
      if (cronRunMode === "script") {
        payload.script_command = cronScriptCommand.trim();
        if (cronScriptCWD.trim()) payload.script_cwd = cronScriptCWD.trim();
        const timeout = Number(cronScriptTimeout);
        if (cronScriptTimeout.trim() && Number.isFinite(timeout) && timeout > 0) {
          payload.script_timeout = timeout;
        }
      }
      const res = await postUICronJob(payload);
      setCronResult(res);
      const job = (res.result as any)?.job;
      if (job?.id) setCronJobID(String(job.id));
      await loadCronJobs();
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e));
    }
  }

  async function loadCronDetail() {
    if (!cronJobID.trim()) return;
    setError("");
    try {
      setCronDetail(await getUICronJob(cronJobID));
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e));
    }
  }

  async function cronAction(action: UICronAction) {
    if (!cronJobID.trim() && action !== "run_get") return;
    setError("");
    try {
      const res = await postUICronJobAction({
        action,
        job_id: cronJobID,
        limit: 20
      });
      if (action === "runs") {
        setCronRuns(res);
      } else {
        setCronResult(res);
      }
      await loadCronJobs();
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e));
    }
  }

  async function refreshVoice() {
    setVoiceStatus(await getUIVoiceStatus());
  }

  async function voiceToggle(action: "on" | "off" | "tts" | "status") {
    setError("");
    try {
      const res = await postUIVoiceToggle(action);
      setVoiceResult(res);
      await refreshVoice();
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e));
    }
  }

  async function voiceRecord(action: "start" | "stop") {
    setError("");
    try {
      const res = await postUIVoiceRecord(action);
      setVoiceResult(res);
      await refreshVoice();
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e));
    }
  }

  async function voiceSpeak() {
    setError("");
    try {
      const res = await postUIVoiceTTS(voiceText);
      setVoiceResult(res);
      await refreshVoice();
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e));
    }
  }

  const filteredTools = tools.filter((t) => t.toLowerCase().includes(toolFilter.toLowerCase()));

  return (
    <div className="page">
      <header className="header">
        <h1>Agent Daemon Dashboard</h1>
        <p>Phase 1 基座：Chat / Sessions / Tools / Gateway / Config</p>
      </header>
      <nav className="tabs">
        {tabs.map((t) => (
          <button key={t} className={tab === t ? "active" : ""} onClick={() => setTab(t)}>
            {t}
          </button>
        ))}
      </nav>
      <main className="panel">
        {tab === "chat" && (
          <section>
            <label>Session ID</label>
            <input value={sessionID} onChange={(e) => setSessionID(e.target.value)} />
            <label>Message</label>
            <textarea value={input} onChange={(e) => setInput(e.target.value)} rows={5} />
            <label className="checkbox">
              <input type="checkbox" checked={streamMode} onChange={(e) => setStreamMode(e.target.checked)} />
              流式模式（/v1/chat/stream）
            </label>
            {streamMode && (
              <div className="row">
                <label>
                  传输模式
                  <select value={streamTransport} onChange={(e) => setStreamTransport(e.target.value as StreamTransport)}>
                    <option value="ws">WS（主通道）</option>
                    <option value="sse">SSE（降级）</option>
                  </select>
                </label>
              </div>
            )}
            {streamMode && (
              <div className={`conn-state conn-${streamStatus}`}>
                连接状态：{streamStatus} / transport={activeTransport}
              </div>
            )}
            {streamMode && fallbackHint && <div className="fallback-note">{fallbackHint}</div>}
            {streamMode && (
              <div className="control-box">
                <h3>重连控制</h3>
                <label className="checkbox">
                  <input type="checkbox" checked={reconnectEnabled} onChange={(e) => setReconnectEnabled(e.target.checked)} />
                  启用自动重连
                </label>
                <div className="row">
                  <label>
                    最大重连
                    <input type="number" min={0} value={reconnectMax} onChange={(e) => setReconnectMax(Number(e.target.value || 0))} />
                  </label>
                  <label>
                    读超时(秒)
                    <input type="number" min={1} value={readTimeoutSec} onChange={(e) => setReadTimeoutSec(Number(e.target.value || 15))} />
                  </label>
                  <label>
                    轮次超时(秒)
                    <input type="number" min={1} value={turnTimeoutSec} onChange={(e) => setTurnTimeoutSec(Number(e.target.value || 120))} />
                  </label>
                </div>
                <div className="row">
                  <label>
                    超时策略
                    <select value={timeoutAction} onChange={(e) => setTimeoutAction(e.target.value as StreamTimeoutAction)}>
                      <option value="wait">wait</option>
                      <option value="reconnect">reconnect</option>
                      <option value="cancel">cancel</option>
                    </select>
                  </label>
                  <button onClick={reconnectNow} disabled={busy || !input.trim()}>手动重连</button>
                  <button onClick={exportDiagnostics}>导出诊断包</button>
                </div>
              </div>
            )}
            {streamMode && (
              <div className="control-box">
                <h3>实时诊断</h3>
                <pre>{JSON.stringify({
                  active_transport: activeTransport,
                  reconnect_status: streamStatus,
                  reconnect_count: reconnectCount,
                  last_error_code: lastErrorCode,
                  last_turn_id: lastTurnID,
                  fallback_hint: fallbackHint || null
                }, null, 2)}</pre>
              </div>
            )}
            <div className="row">
              <button onClick={onSend} disabled={busy}>发送</button>
              <button onClick={onCancel} disabled={busy}>取消会话</button>
            </div>
            {error && <pre className="error">{error}</pre>}
            <pre>{output || "等待响应..."}</pre>
            {streamMode && (
              <>
                <h3>Timeline</h3>
                <pre>{JSON.stringify(timeline, null, 2)}</pre>
              </>
            )}
          </section>
        )}
        {tab === "sessions" && (
          <section>
            <h2>sessions</h2>
            <div className="split">
              <div>
                {sessions.map((s) => (
                  <button key={s.session_id} onClick={() => pickSession(s.session_id)}>
                    {s.session_id}
                  </button>
                ))}
              </div>
              <div>
                <div className="row">
                  <button onClick={() => changeSessionPage(sessionOffset - sessionPageSize)}>上一页</button>
                  <button onClick={() => changeSessionPage(sessionOffset + sessionPageSize)}>下一页</button>
                  <button onClick={() => selectedSession && changeSessionPage(sessionOffset)}>刷新</button>
                </div>
                <pre>{JSON.stringify(sessionDetail, null, 2)}</pre>
              </div>
            </div>
          </section>
        )}
        {tab === "tools" && (
          <section>
            <h2>tools ({tools.length})</h2>
            <div className="split">
              <div>
                <input placeholder="筛选工具" value={toolFilter} onChange={(e) => setToolFilter(e.target.value)} />
                {filteredTools.map((t) => (
                  <button key={t} onClick={() => pickTool(t)}>
                    {selectedTool === t ? `* ${t}` : t}
                  </button>
                ))}
              </div>
              <pre>{JSON.stringify(toolSchema, null, 2)}</pre>
            </div>
          </section>
        )}
        {tab === "skills" && (
          <section>
            <h2>skills ({skills.length})</h2>
            <div className="split wide">
              <div>
                <input placeholder="新 skill 名称" value={newSkillName} onChange={(e) => setNewSkillName(e.target.value)} />
                <button onClick={createSkillDraft}>新建草稿</button>
                <button onClick={() => postUISkillsReload().then(loadSkills).catch((e) => setError(String(e)))}>重载列表</button>
                {skills.map((s) => {
                  const name = String(s.name || "");
                  return (
                    <button key={name} onClick={() => pickSkill(name).catch((e) => setError(String(e)))}>
                      {selectedSkill === name ? `* ${name}` : name}
                    </button>
                  );
                })}
              </div>
              <div>
                <div className="row">
                  <button onClick={saveSkill}>{selectedSkill ? "保存修改" : "创建 skill"}</button>
                  <button onClick={deleteSkill} disabled={!selectedSkill}>删除</button>
                </div>
                <textarea className="editor" value={skillContent} onChange={(e) => setSkillContent(e.target.value)} rows={18} />
                <div className="control-box">
                  <h3>搜索 / 同步</h3>
                  <div className="row">
                    <input placeholder="搜索关键词" value={skillSearchQuery} onChange={(e) => setSkillSearchQuery(e.target.value)} />
                    <input placeholder="repo，可选" value={skillSearchRepo} onChange={(e) => setSkillSearchRepo(e.target.value)} />
                    <button onClick={searchSkills}>搜索</button>
                  </div>
                  <div className="row">
                    <select value={skillSyncSource} onChange={(e) => setSkillSyncSource(e.target.value as "url" | "github")}>
                      <option value="github">github</option>
                      <option value="url">url</option>
                    </select>
                    {skillSyncSource === "url" ? (
                      <input placeholder="SKILL.md URL" value={skillSyncURL} onChange={(e) => setSkillSyncURL(e.target.value)} />
                    ) : (
                      <>
                        <input placeholder="owner/repo" value={skillSyncRepo} onChange={(e) => setSkillSyncRepo(e.target.value)} />
                        <input placeholder="path/to/skill" value={skillSyncPath} onChange={(e) => setSkillSyncPath(e.target.value)} />
                      </>
                    )}
                    <button onClick={syncSkill}>同步到当前名称</button>
                  </div>
                  <pre>{JSON.stringify(skillSearchResult, null, 2)}</pre>
                </div>
              </div>
            </div>
          </section>
        )}
        {tab === "agents" && (
          <section>
            <h2>agents</h2>
            <div className="row">
              <button onClick={() => loadAgents().catch((e) => setError(String(e)))}>刷新</button>
              <input placeholder="session_id" value={agentSessionID} onChange={(e) => setAgentSessionID(e.target.value)} />
              <button onClick={loadAgentDetail}>详情</button>
              <button onClick={interruptAgent}>中断</button>
            </div>
            <div className="grid3">
              <div>
                <h3>delegates</h3>
                <pre>{JSON.stringify(agents, null, 2)}</pre>
              </div>
              <div>
                <h3>active</h3>
                <pre>{JSON.stringify(activeAgents, null, 2)}</pre>
              </div>
              <div>
                <h3>history</h3>
                <pre>{JSON.stringify(agentHistory, null, 2)}</pre>
              </div>
            </div>
            <h3>detail</h3>
            <pre>{JSON.stringify(agentDetail, null, 2)}</pre>
          </section>
        )}
        {tab === "cron" && (
          <section>
            <h2>cron</h2>
            <div className="control-box">
              <h3>新建任务</h3>
              <input placeholder="名称" value={cronJobName} onChange={(e) => setCronJobName(e.target.value)} />
              <textarea placeholder="执行提示词" value={cronPrompt} onChange={(e) => setCronPrompt(e.target.value)} rows={4} />
              <div className="row">
                <select value={cronRunMode} onChange={(e) => setCronRunMode(e.target.value)}>
                  <option value="agent">agent</option>
                  <option value="script">script</option>
                </select>
                {cronRunMode === "script" && (
                  <>
                    <input placeholder="script_command，例如: ./scripts/report.sh" value={cronScriptCommand} onChange={(e) => setCronScriptCommand(e.target.value)} />
                    <input placeholder="script_cwd，可选" value={cronScriptCWD} onChange={(e) => setCronScriptCWD(e.target.value)} />
                    <input placeholder="script_timeout 秒，可选" value={cronScriptTimeout} onChange={(e) => setCronScriptTimeout(e.target.value)} />
                  </>
                )}
              </div>
              <div className="row">
                <input placeholder="every 30m / 30m / */15 9-17 * * 1-5 / 2026-05-14T10:00:00Z" value={cronSchedule} onChange={(e) => setCronSchedule(e.target.value)} />
                <input placeholder="repeat，可选" value={cronRepeat} onChange={(e) => setCronRepeat(e.target.value)} />
                <button onClick={createCronJob}>创建</button>
              </div>
              <div className="row">
                <input placeholder="投递目标，可选：telegram:123 / discord:channel / slack:channel" value={cronDeliveryTarget} onChange={(e) => setCronDeliveryTarget(e.target.value)} />
                <select value={cronDeliverOn} onChange={(e) => setCronDeliverOn(e.target.value)}>
                  <option value="always">always</option>
                  <option value="success">success</option>
                  <option value="failure">failure</option>
                </select>
                <select value={cronContextMode} onChange={(e) => setCronContextMode(e.target.value)}>
                  <option value="isolated">isolated</option>
                  <option value="chained">chained</option>
                </select>
              </div>
            </div>
            <div className="row">
              <button onClick={() => loadCronJobs().catch((e) => setError(String(e)))}>刷新列表</button>
              <input placeholder="job_id" value={cronJobID} onChange={(e) => setCronJobID(e.target.value)} />
              <button onClick={loadCronDetail}>详情</button>
              <button onClick={() => cronAction("pause")}>暂停</button>
              <button onClick={() => cronAction("resume")}>恢复</button>
              <button onClick={() => cronAction("trigger")}>立即触发</button>
              <button onClick={() => cronAction("runs")}>运行记录</button>
              <button onClick={() => cronAction("remove")}>删除</button>
            </div>
            <div className="grid3">
              <div>
                <h3>jobs</h3>
                <pre>{JSON.stringify(cronJobs, null, 2)}</pre>
              </div>
              <div>
                <h3>detail</h3>
                <pre>{JSON.stringify(cronDetail, null, 2)}</pre>
              </div>
              <div>
                <h3>runs</h3>
                <pre>{JSON.stringify(cronRuns, null, 2)}</pre>
              </div>
            </div>
            <h3>last result</h3>
            <pre>{JSON.stringify(cronResult, null, 2)}</pre>
          </section>
        )}
        {tab === "models" && (
          <section>
            <h2>models</h2>
            <div className="control-box">
              <h3>切换模型</h3>
              <div className="row">
                <label>
                  provider
                  <select value={modelProvider} onChange={(e) => setModelProvider(e.target.value)}>
                    {modelProviders.map((p) => (
                      <option key={p} value={p}>{p}</option>
                    ))}
                  </select>
                </label>
                <label>
                  model
                  <input value={modelName} onChange={(e) => setModelName(e.target.value)} placeholder="model name" />
                </label>
                <label>
                  base_url
                  <input value={modelBaseURL} onChange={(e) => setModelBaseURL(e.target.value)} placeholder="optional provider base URL" />
                </label>
                <button onClick={applyModel}>写入模型配置</button>
                <button onClick={() => loadModelPage().catch((e) => setError(String(e)))}>刷新</button>
              </div>
            </div>
            <div className="grid3">
              <div>
                <h3>current</h3>
                <pre>{JSON.stringify(modelInfo, null, 2)}</pre>
              </div>
              <div>
                <h3>providers</h3>
                <pre>{JSON.stringify(modelProviders, null, 2)}</pre>
              </div>
              <div>
                <h3>last result</h3>
                <pre>{JSON.stringify(modelSetResult, null, 2)}</pre>
              </div>
            </div>
          </section>
        )}
        {tab === "plugins" && (
          <section>
            <h2>plugin dashboards ({pluginDashboards.length})</h2>
            <div className="row">
              <button onClick={() => getUIPluginDashboards().then((r) => setPluginDashboards(r.dashboards)).catch((e) => setError(String(e)))}>
                刷新
              </button>
            </div>
            <pre>{JSON.stringify(pluginDashboards, null, 2)}</pre>
          </section>
        )}
        {tab === "gateway" && (
          <section>
            <h2>gateway status</h2>
            <div className="row">
              <button onClick={() => gatewayAction("enable")}>启用网关</button>
              <button onClick={() => gatewayAction("disable")}>禁用网关</button>
              <button onClick={() => Promise.all([getUIGatewayStatus(), getUIGatewayDiagnostics()]).then(([s, d]) => { setGatewayStatus(s); setGatewayDiagnostics(d); }).catch((e) => setError(String(e)))}>
                刷新状态
              </button>
            </div>
            <h3>status</h3>
            <pre>{JSON.stringify(gatewayStatus, null, 2)}</pre>
            <h3>diagnostics</h3>
            <pre>{JSON.stringify(gatewayDiagnostics, null, 2)}</pre>
          </section>
        )}
        {tab === "voice" && (
          <section>
            <h2>voice</h2>
            <div className="row">
              <button onClick={() => voiceToggle("on")}>开启</button>
              <button onClick={() => voiceToggle("off")}>关闭</button>
              <button onClick={() => voiceToggle("tts")}>切换 TTS</button>
              <button onClick={() => voiceRecord("start")}>开始录音</button>
              <button onClick={() => voiceRecord("stop")}>停止录音</button>
              <button onClick={() => refreshVoice().catch((e) => setError(String(e)))}>刷新</button>
            </div>
            <textarea value={voiceText} onChange={(e) => setVoiceText(e.target.value)} rows={4} placeholder="TTS 文本" />
            <button onClick={voiceSpeak}>播放/提交 TTS</button>
            <h3>status</h3>
            <pre>{JSON.stringify(voiceStatus, null, 2)}</pre>
            <h3>last result</h3>
            <pre>{JSON.stringify(voiceResult, null, 2)}</pre>
          </section>
        )}
        {tab === "config" && (
          <section>
            <h2>config snapshot</h2>
            <label>配置键（section.key）</label>
            <input value={configKey} onChange={(e) => setConfigKey(e.target.value)} />
            <label>配置值</label>
            <input value={configValue} onChange={(e) => setConfigValue(e.target.value)} />
            <div className="row">
              <button onClick={applyConfig}>写入配置</button>
              <button onClick={() => getUIConfig().then(setConfig).catch((e) => setError(String(e)))}>刷新快照</button>
            </div>
            <pre>{JSON.stringify(config, null, 2)}</pre>
          </section>
        )}
      </main>
    </div>
  );
}
