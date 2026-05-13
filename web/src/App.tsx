import { useEffect, useState } from "react";
import { cancelChat, getUIConfig, getUIGatewayStatus, getUISessions, getUITools, sendChat } from "./lib/api";

type Tab = "chat" | "sessions" | "tools" | "gateway" | "config";

const tabs: Tab[] = ["chat", "sessions", "tools", "gateway", "config"];

export function App() {
  const [tab, setTab] = useState<Tab>("chat");
  const [sessionID, setSessionID] = useState("web-default");
  const [input, setInput] = useState("");
  const [output, setOutput] = useState("");
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState("");
  const [sessions, setSessions] = useState<Array<{ session_id: string; last_seen?: string }>>([]);
  const [tools, setTools] = useState<string[]>([]);
  const [gatewayStatus, setGatewayStatus] = useState<Record<string, unknown> | null>(null);
  const [config, setConfig] = useState<Record<string, unknown> | null>(null);

  async function onSend() {
    if (!input.trim()) return;
    setBusy(true);
    setError("");
    try {
      const res = await sendChat(input, sessionID);
      setOutput(res.final_response || "(empty)");
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
      getUISessions(30).then((r) => setSessions(r.sessions)).catch((e) => setError(String(e)));
    }
    if (tab === "tools") {
      getUITools().then((r) => setTools(r.tools)).catch((e) => setError(String(e)));
    }
    if (tab === "gateway") {
      getUIGatewayStatus().then(setGatewayStatus).catch((e) => setError(String(e)));
    }
    if (tab === "config") {
      getUIConfig().then(setConfig).catch((e) => setError(String(e)));
    }
  }, [tab]);

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
            <div className="row">
              <button onClick={onSend} disabled={busy}>发送</button>
              <button onClick={onCancel} disabled={busy}>取消会话</button>
            </div>
            {error && <pre className="error">{error}</pre>}
            <pre>{output || "等待响应..."}</pre>
          </section>
        )}
        {tab === "sessions" && (
          <section>
            <h2>sessions</h2>
            <pre>{JSON.stringify(sessions, null, 2)}</pre>
          </section>
        )}
        {tab === "tools" && (
          <section>
            <h2>tools ({tools.length})</h2>
            <pre>{JSON.stringify(tools, null, 2)}</pre>
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
