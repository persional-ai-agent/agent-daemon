export type ChatResponse = {
  session_id: string;
  final_response: string;
};

export type UISessionsResponse = {
  count: number;
  sessions: Array<{ session_id: string; last_seen?: string }>;
};

export type UIToolsResponse = {
  count: number;
  tools: string[];
};

export type UISkillsResponse = {
  count: number;
  skills: Array<Record<string, unknown>>;
};

export type UIAgentsResponse = {
  count: number;
  agents?: Array<Record<string, unknown>>;
  active?: Array<Record<string, unknown>>;
  history?: Array<Record<string, unknown>>;
};

export type UIPluginDashboardsResponse = {
  count: number;
  dashboards: Array<Record<string, unknown>>;
};

export type UICronAction =
  | "update"
  | "pause"
  | "resume"
  | "remove"
  | "trigger"
  | "runs"
  | "run_get";

const BASE = (globalThis as any).__AGENT_API_BASE__ || "http://127.0.0.1:8080";
const WS_BASE = BASE.replace(/^http:\/\//, "ws://").replace(/^https:\/\//, "wss://");

export type APIErrorNormalized = {
  code: string;
  message: string;
};

export function normalizeAPIError(input: unknown): APIErrorNormalized {
  if (typeof input === "string") {
    return { code: "unknown", message: input };
  }
  if (!input || typeof input !== "object") {
    return { code: "unknown", message: "unknown error" };
  }
  const obj = input as Record<string, unknown>;
  const detail = (obj.error_detail && typeof obj.error_detail === "object") ? (obj.error_detail as Record<string, unknown>) : null;
  const code =
    (typeof obj.error_code === "string" && obj.error_code) ||
    (typeof detail?.code === "string" && detail.code) ||
    "unknown";
  const message =
    (typeof obj.error === "string" && obj.error) ||
    (typeof detail?.message === "string" && detail.message) ||
    (typeof obj.reason === "string" && obj.reason) ||
    "unknown error";
  return { code, message };
}

function errorFromHTTP(status: number, rawBody: string): Error {
  try {
    const parsed = JSON.parse(rawBody);
    const e = normalizeAPIError(parsed);
    return new Error(`HTTP ${status} [${e.code}]: ${e.message}`);
  } catch {
    return new Error(`HTTP ${status}: ${rawBody}`);
  }
}

async function request<T>(path: string, init?: RequestInit): Promise<T> {
  const res = await fetch(`${BASE}${path}`, {
    headers: { "Content-Type": "application/json" },
    ...init
  });
  if (!res.ok) {
    const text = await res.text();
    throw errorFromHTTP(res.status, text);
  }
  return (await res.json()) as T;
}

export function sendChat(message: string, sessionID: string) {
  return request<ChatResponse>("/v1/chat", {
    method: "POST",
    body: JSON.stringify({ message, session_id: sessionID })
  });
}

export type StreamEvent = {
  event: string;
  data: unknown;
};

export function createTransportFallbackEvent(
  from: StreamTransport,
  to: StreamTransport,
  reason: string,
  at?: string
): StreamEvent {
  return {
    event: "transport_fallback",
    data: {
      from,
      to,
      reason,
      at: at ?? new Date().toISOString()
    }
  };
}

export function streamEventDedupeKey(evt: StreamEvent): string {
  return `${evt.event}:${JSON.stringify(evt.data)}`;
}

export type StreamReconnectStatus = "connecting" | "resumed" | "degraded" | "failed";
export type StreamTimeoutAction = "wait" | "reconnect" | "cancel";
export type StreamTransport = "ws" | "sse";

export type StreamChatOptions = {
  transport?: StreamTransport;
  fallbackToSSE?: boolean;
  reconnectEnabled?: boolean;
  maxReconnect?: number;
  readTimeoutMs?: number;
  turnTimeoutMs?: number;
  timeoutAction?: StreamTimeoutAction;
  onStatus?: (status: StreamReconnectStatus) => void;
  onTransport?: (transport: StreamTransport) => void;
};

async function streamOnce(
  message: string,
  sessionID: string,
  turnID: string,
  resume: boolean,
  onEvent: (evt: StreamEvent) => void,
  readTimeoutMs: number,
  turnDeadline: number
) {
  const res = await fetch(`${BASE}/v1/chat/stream`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ message, session_id: sessionID, turn_id: turnID, resume })
  });
  if (!res.ok || !res.body) {
    const text = await res.text();
    throw errorFromHTTP(res.status, text);
  }
  const reader = res.body.getReader();
  const decoder = new TextDecoder();
  let buffer = "";
  while (true) {
    const now = Date.now();
    if (now > turnDeadline) {
      throw new Error("TURN_TIMEOUT");
    }
    const timeoutPromise = new Promise<never>((_, reject) => {
      setTimeout(() => reject(new Error("READ_TIMEOUT")), readTimeoutMs);
    });
    const { done, value } = await Promise.race([reader.read(), timeoutPromise]) as ReadableStreamReadResult<Uint8Array>;
    if (done) break;
    buffer += decoder.decode(value, { stream: true });
    let split = buffer.indexOf("\n\n");
    while (split >= 0) {
      const chunk = buffer.slice(0, split);
      buffer = buffer.slice(split + 2);
      const eventLine = chunk.split("\n").find((l) => l.startsWith("event: "));
      const dataLine = chunk.split("\n").find((l) => l.startsWith("data: "));
      if (eventLine && dataLine) {
        const event = eventLine.slice(7).trim();
        const raw = dataLine.slice(6);
        let data: unknown = raw;
        try {
          data = JSON.parse(raw);
        } catch {
          // keep raw string
        }
        onEvent({ event, data });
        if (event === "result" || event === "error" || event === "cancelled") {
          return;
        }
      }
      split = buffer.indexOf("\n\n");
    }
  }
}

export async function streamChat(
  message: string,
  sessionID: string,
  onEvent: (evt: StreamEvent) => void,
  options?: StreamChatOptions
) {
  const transport = options?.transport ?? "ws";
  options?.onTransport?.(transport);
  if (transport === "ws") {
    try {
      await streamChatWS(message, sessionID, onEvent, options);
      return;
    } catch (e) {
      if (!options?.fallbackToSSE) {
        throw e;
      }
      options?.onTransport?.("sse");
      onEvent(createTransportFallbackEvent("ws", "sse", e instanceof Error ? e.message : String(e)));
    }
  }
  await streamChatSSE(message, sessionID, onEvent, options);
}

async function streamChatSSE(
  message: string,
  sessionID: string,
  onEvent: (evt: StreamEvent) => void,
  options?: StreamChatOptions
) {
  const reconnectEnabled = options?.reconnectEnabled ?? true;
  const maxReconnect = reconnectEnabled ? (options?.maxReconnect ?? 2) : 0;
  const readTimeoutMs = options?.readTimeoutMs ?? 15_000;
  const turnTimeoutMs = options?.turnTimeoutMs ?? 120_000;
  const timeoutAction = options?.timeoutAction ?? "wait";
  const onStatus = options?.onStatus;
  const turnID = `web-${Math.random().toString(36).slice(2)}`;
  const started = Date.now();
  onStatus?.("connecting");

  for (let attempt = 0; attempt <= maxReconnect; attempt++) {
    const resume = attempt > 0;
    if (resume) onStatus?.("degraded");
    try {
      await streamOnce(message, sessionID, turnID, resume, onEvent, readTimeoutMs, started + turnTimeoutMs);
      return;
    } catch (e) {
      const msg = e instanceof Error ? e.message : String(e);
      if (msg === "READ_TIMEOUT" || msg === "TURN_TIMEOUT") {
        if (timeoutAction === "wait") {
          continue;
        }
        if (timeoutAction === "cancel") {
          await cancelChat(sessionID);
          onStatus?.("failed");
          throw new Error("timeout cancelled");
        }
      }
      if (attempt >= maxReconnect) {
        onStatus?.("failed");
        throw e;
      }
      await new Promise((r) => setTimeout(r, 300));
      onStatus?.("resumed");
    }
  }
}

function streamChatWSOnce(
  wsURL: string,
  payload: Record<string, unknown>,
  onEvent: (evt: StreamEvent) => void,
  readTimeoutMs: number,
  turnTimeoutAt: number,
  timeoutAction: StreamTimeoutAction
): Promise<void> {
  return new Promise((resolve, reject) => {
    const ws = new WebSocket(wsURL);
    let settled = false;
    let lastEventAt = Date.now();

    const fail = (err: Error) => {
      if (settled) return;
      settled = true;
      try { ws.close(); } catch {}
      reject(err);
    };
    const done = () => {
      if (settled) return;
      settled = true;
      clearInterval(timer);
      resolve();
    };

    const timer = setInterval(() => {
      const now = Date.now();
      if (now >= turnTimeoutAt) {
        clearInterval(timer);
        fail(new Error("TURN_TIMEOUT"));
        return;
      }
      if (now-lastEventAt >= readTimeoutMs) {
        switch (timeoutAction) {
          case "wait":
            lastEventAt = now;
            break;
          case "reconnect":
            clearInterval(timer);
            fail(new Error("READ_TIMEOUT"));
            break;
          case "cancel":
            clearInterval(timer);
            fail(new Error("CANCEL_TIMEOUT"));
            break;
        }
      }
    }, 250);

    ws.onopen = () => {
      ws.send(JSON.stringify(payload));
    };
    ws.onerror = () => {
      clearInterval(timer);
      fail(new Error("WS_ERROR"));
    };
    ws.onmessage = (evt) => {
      lastEventAt = Date.now();
      let data: unknown = evt.data;
      try { data = JSON.parse(String(evt.data)); } catch {}
      if (data && typeof data === "object") {
        const obj = data as Record<string, unknown>;
        const type = typeof obj.type === "string" ? obj.type : "message";
        onEvent({ event: type, data });
        if (type === "result" || type === "error" || type === "cancelled") {
          clearInterval(timer);
          done();
        }
      }
    };
    ws.onclose = () => {
      if (!settled) {
        clearInterval(timer);
        fail(new Error("WS_CLOSED"));
      }
    };
  });
}

async function streamChatWS(
  message: string,
  sessionID: string,
  onEvent: (evt: StreamEvent) => void,
  options?: StreamChatOptions
) {
  const reconnectEnabled = options?.reconnectEnabled ?? true;
  const maxReconnect = reconnectEnabled ? (options?.maxReconnect ?? 2) : 0;
  const readTimeoutMs = options?.readTimeoutMs ?? 15_000;
  const turnTimeoutMs = options?.turnTimeoutMs ?? 120_000;
  const timeoutAction = options?.timeoutAction ?? "wait";
  const onStatus = options?.onStatus;
  const turnID = `web-${Math.random().toString(36).slice(2)}`;
  const started = Date.now();
  onStatus?.("connecting");
  for (let attempt = 0; attempt <= maxReconnect; attempt++) {
    const resume = attempt > 0;
    if (resume) onStatus?.("degraded");
    try {
      await streamChatWSOnce(`${WS_BASE}/v1/chat/ws`, {
        message,
        session_id: sessionID,
        turn_id: turnID,
        resume
      }, onEvent, readTimeoutMs, started + turnTimeoutMs, timeoutAction);
      return;
    } catch (e) {
      const msg = e instanceof Error ? e.message : String(e);
      if (msg === "CANCEL_TIMEOUT") {
        await cancelChat(sessionID);
        onStatus?.("failed");
        throw new Error("timeout cancelled");
      }
      if (attempt >= maxReconnect) {
        onStatus?.("failed");
        throw e;
      }
      await new Promise((r) => setTimeout(r, 300));
      onStatus?.("resumed");
    }
  }
}

export function cancelChat(sessionID: string) {
  return request<{ ok: boolean; session_id: string }>("/v1/chat/cancel", {
    method: "POST",
    body: JSON.stringify({ session_id: sessionID })
  });
}

export function getUISessions(limit = 20) {
  return request<UISessionsResponse>(`/v1/ui/sessions?limit=${limit}`);
}

export function getUISessionDetail(sessionID: string, offset = 0, limit = 100) {
  return request<Record<string, unknown>>(
    `/v1/ui/sessions/${encodeURIComponent(sessionID)}?offset=${offset}&limit=${limit}`
  );
}

export function getUITools() {
  return request<UIToolsResponse>("/v1/ui/tools");
}

export function getUIToolSchema(name: string) {
  return request<Record<string, unknown>>(`/v1/ui/tools/${encodeURIComponent(name)}/schema`);
}

export function getUIConfig() {
  return request<Record<string, unknown>>("/v1/ui/config");
}

export function getUIModel() {
  return request<Record<string, unknown>>("/v1/ui/model");
}

export function getUIModelProviders() {
  return request<Record<string, unknown>>("/v1/ui/model/providers");
}

export function setUIModel(provider: string, model: string, baseURL?: string) {
  return request<Record<string, unknown>>("/v1/ui/model/set", {
    method: "POST",
    body: JSON.stringify({ provider, model, base_url: baseURL || "" })
  });
}

export function getUIGatewayStatus() {
  return request<Record<string, unknown>>("/v1/ui/gateway/status");
}

export function setUIConfig(key: string, value: string) {
  return request<Record<string, unknown>>("/v1/ui/config/set", {
    method: "POST",
    body: JSON.stringify({ key, value })
  });
}

export function postUIGatewayAction(action: "enable" | "disable") {
  return request<Record<string, unknown>>("/v1/ui/gateway/action", {
    method: "POST",
    body: JSON.stringify({ action })
  });
}

export function getUIGatewayDiagnostics() {
  return request<Record<string, unknown>>("/v1/ui/gateway/diagnostics");
}

export function getUISkills() {
  return request<UISkillsResponse>("/v1/ui/skills");
}

export function getUISkillDetail(name: string) {
  return request<Record<string, unknown>>(`/v1/ui/skills/detail?name=${encodeURIComponent(name)}`);
}

export function postUISkillManage(payload: {
  action: "create" | "edit" | "patch" | "delete";
  name: string;
  content?: string;
  old_string?: string;
  new_string?: string;
  replace_all?: boolean;
}) {
  return request<Record<string, unknown>>("/v1/ui/skills/manage", {
    method: "POST",
    body: JSON.stringify(payload)
  });
}

export function postUISkillsReload() {
  return request<Record<string, unknown>>("/v1/ui/skills/reload", { method: "POST" });
}

export function postUISkillsSearch(query: string, repo?: string) {
  return request<Record<string, unknown>>("/v1/ui/skills/search", {
    method: "POST",
    body: JSON.stringify({ query, repo })
  });
}

export function postUISkillsSync(payload: {
  name: string;
  source: "url" | "github";
  url?: string;
  repo?: string;
  path?: string;
}) {
  return request<Record<string, unknown>>("/v1/ui/skills/sync", {
    method: "POST",
    body: JSON.stringify(payload)
  });
}

export function getUIAgents(limit = 20, sessionID?: string) {
  const qs = new URLSearchParams({ limit: String(limit) });
  if (sessionID) qs.set("session_id", sessionID);
  return request<UIAgentsResponse>(`/v1/ui/agents?${qs.toString()}`);
}

export function getUIAgentsActive(sessionID?: string) {
  const suffix = sessionID ? `?session_id=${encodeURIComponent(sessionID)}` : "";
  return request<UIAgentsResponse>(`/v1/ui/agents/active${suffix}`);
}

export function getUIAgentsHistory(limit = 20) {
  return request<UIAgentsResponse>(`/v1/ui/agents/history?limit=${limit}`);
}

export function getUIAgentDetail(sessionID: string) {
  return request<Record<string, unknown>>(`/v1/ui/agents/detail?session_id=${encodeURIComponent(sessionID)}`);
}

export function postUIAgentInterrupt(sessionID: string) {
  return request<Record<string, unknown>>("/v1/ui/agents/interrupt", {
    method: "POST",
    body: JSON.stringify({ session_id: sessionID })
  });
}

export function getUIPluginDashboards() {
  return request<UIPluginDashboardsResponse>("/v1/ui/plugins/dashboards");
}

export function getUIVoiceStatus() {
  return request<Record<string, unknown>>("/v1/ui/voice/status");
}

export function postUIVoiceToggle(action: "on" | "off" | "tts" | "status") {
  return request<Record<string, unknown>>("/v1/ui/voice/toggle", {
    method: "POST",
    body: JSON.stringify({ action })
  });
}

export function postUIVoiceRecord(action: "start" | "stop") {
  return request<Record<string, unknown>>("/v1/ui/voice/record", {
    method: "POST",
    body: JSON.stringify({ action })
  });
}

export function postUIVoiceTTS(text: string) {
  return request<Record<string, unknown>>("/v1/ui/voice/tts", {
    method: "POST",
    body: JSON.stringify({ text })
  });
}

export function getUICronJobs() {
  return request<Record<string, unknown>>("/v1/ui/cron/jobs");
}

export function postUICronJob(payload: {
  name: string;
  prompt: string;
  schedule: string;
  repeat?: number;
  delivery_target?: string;
  deliver_on?: string;
  context_mode?: string;
  chain_context?: boolean;
  run_mode?: string;
  script_command?: string;
  script_cwd?: string;
  script_timeout?: number;
}) {
  return request<Record<string, unknown>>("/v1/ui/cron/jobs", {
    method: "POST",
    body: JSON.stringify(payload)
  });
}

export function getUICronJob(jobID: string) {
  return request<Record<string, unknown>>(`/v1/ui/cron/jobs/${encodeURIComponent(jobID)}`);
}

export function postUICronJobAction(payload: {
  action: UICronAction;
  job_id?: string;
  run_id?: string;
  name?: string;
  prompt?: string;
  schedule?: string;
  repeat?: number;
  delivery_target?: string;
  deliver_on?: string;
  context_mode?: string;
  chain_context?: boolean;
  run_mode?: string;
  script_command?: string;
  script_cwd?: string;
  script_timeout?: number;
  paused?: boolean;
  limit?: number;
}) {
  return request<Record<string, unknown>>("/v1/ui/cron/jobs/action", {
    method: "POST",
    body: JSON.stringify(payload)
  });
}
