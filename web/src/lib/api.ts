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
  if (transport === "ws") {
    try {
      await streamChatWS(message, sessionID, onEvent, options);
      return;
    } catch (e) {
      if (!options?.fallbackToSSE) {
        throw e;
      }
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
