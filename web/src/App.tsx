import { useEffect, useState } from "react";
import {
  cancelChat,
  getUIConfig,
  getUIGatewayStatus,
  getUISessionDetail,
  getUISessions,
  getUIToolSchema,
  getUITools,
  sendChat,
  streamChat,
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
  const [sessions, setSessions] = useState<Array<{ session_id: string; last_seen?: string }>>([]);
  const [sessionDetail, setSessionDetail] = useState<Record<string, unknown> | null>(null);
  const [tools, setTools] = useState<string[]>([]);
  const [toolSchema, setToolSchema] = useState<Record<string, unknown> | null>(null);
  const [gatewayStatus, setGatewayStatus] = useState<Record<string, unknown> | null>(null);
  const [config, setConfig] = useState<Record<string, unknown> | null>(null);

  async function onSend() {
    if (!input.trim()) return;
    setBusy(true);
    setError("");
    setTimeline([]);
    try {
      if (streamMode) {
        await streamChat(input, sessionID, (evt: StreamEvent) => {
          setTimeline((prev) => prev.concat([{ ts: new Date().toISOString(), event: evt.event, data: evt.data }]));
          if (evt.event === "result" && typeof evt.data === "object" && evt.data !== null) {
            const finalResponse = (evt.data as any).final_response;
            if (typeof finalResponse === "string") {
              setOutput(finalResponse || "(empty)");
            }
          }
        });
      } else {
        const res = await sendChat(input, sessionID);
        setOutput(res.final_response || "(empty)");
      }
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e));
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
      setError(e instanceof Error ? e.message : String(e));
    }
  }

  useEffect(() => {
    if (tab === "sessions") {
      getUISessions(30)
        .then((r) => {
          setSessions(r.sessions);
          if (r.sessions.length > 0) {
            return getUISessionDetail(r.sessions[0].session_id, 0, 50);
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
    getUISessionDetail(id, 0, 50).then(setSessionDetail).catch((e) => setError(String(e)));
  }

  function pickTool(name: string) {
    getUIToolSchema(name).then(setToolSchema).catch((e) => setError(String(e)));
  }

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
              <pre>{JSON.stringify(sessionDetail, null, 2)}</pre>
            </div>
          </section>
        )}
        {tab === "tools" && (
          <section>
            <h2>tools ({tools.length})</h2>
            <div className="split">
              <div>
                {tools.map((t) => (
                  <button key={t} onClick={() => pickTool(t)}>
                    {t}
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
            <pre>{JSON.stringify(gatewayStatus, null, 2)}</pre>
          </section>
        )}
        {tab === "config" && (
          <section>
            <h2>config snapshot</h2>
            <pre>{JSON.stringify(config, null, 2)}</pre>
          </section>
        )}
      </main>
    </div>
  );
}
