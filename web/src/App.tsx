import { useEffect, useState } from "react";
import {
  cancelChat,
  getUIConfig,
  getUIGatewayStatus,
  getUISessionDetail,
  getUISessions,
  postUIGatewayAction,
  setUIConfig,
  getUIToolSchema,
  getUITools,
  sendChat,
  streamChat,
  streamEventDedupeKey,
  normalizeAPIError,
  type StreamReconnectStatus,
  type StreamTransport,
  type StreamTimeoutAction,
  type StreamEvent
} from "./lib/api";

type Tab = "chat" | "sessions" | "tools" | "gateway" | "config";

const tabs: Tab[] = ["chat", "sessions", "tools", "gateway", "config"];

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
  const [config, setConfig] = useState<Record<string, unknown> | null>(null);
  const [configKey, setConfigKey] = useState("api.type");
  const [configValue, setConfigValue] = useState("openai");

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
    const payload = {
      exported_at: new Date().toISOString(),
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
      timeline
    };
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
    if (tab === "gateway") {
      getUIGatewayStatus().then(setGatewayStatus).catch((e) => setError(String(e)));
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

  async function gatewayAction(action: "enable" | "disable") {
    setError("");
    try {
      const res = await postUIGatewayAction(action);
      setGatewayStatus((res.status as Record<string, unknown>) || res);
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
        {tab === "gateway" && (
          <section>
            <h2>gateway status</h2>
            <div className="row">
              <button onClick={() => gatewayAction("enable")}>启用网关</button>
              <button onClick={() => gatewayAction("disable")}>禁用网关</button>
              <button onClick={() => getUIGatewayStatus().then(setGatewayStatus).catch((e) => setError(String(e)))}>刷新状态</button>
            </div>
            <pre>{JSON.stringify(gatewayStatus, null, 2)}</pre>
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
