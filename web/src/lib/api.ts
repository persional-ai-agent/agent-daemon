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

export function cancelChat(sessionID: string) {
  return request<{ ok: boolean; session_id: string }>("/v1/chat/cancel", {
    method: "POST",
    body: JSON.stringify({ session_id: sessionID })
  });
}

export function getUISessions(limit = 20) {
  return request<UISessionsResponse>(`/v1/ui/sessions?limit=${limit}`);
}

export function getUITools() {
  return request<UIToolsResponse>("/v1/ui/tools");
}

export function getUIConfig() {
  return request<Record<string, unknown>>("/v1/ui/config");
}

export function getUIGatewayStatus() {
  return request<Record<string, unknown>>("/v1/ui/gateway/status");
}
