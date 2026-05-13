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

async function request<T>(path: string, init?: RequestInit): Promise<T> {
  const res = await fetch(`${BASE}${path}`, {
    headers: { "Content-Type": "application/json" },
    ...init
  });
  if (!res.ok) {
    const text = await res.text();
    throw new Error(`HTTP ${res.status}: ${text}`);
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

export async function streamChat(
  message: string,
  sessionID: string,
  onEvent: (evt: StreamEvent) => void
) {
  const res = await fetch(`${BASE}/v1/chat/stream`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ message, session_id: sessionID })
  });
  if (!res.ok || !res.body) {
    const text = await res.text();
    throw new Error(`HTTP ${res.status}: ${text}`);
  }
  const reader = res.body.getReader();
  const decoder = new TextDecoder();
  let buffer = "";
  while (true) {
    const { done, value } = await reader.read();
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
      }
      split = buffer.indexOf("\n\n");
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
