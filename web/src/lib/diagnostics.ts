export type DiagnosticsEvent = Record<string, unknown>;

export type DiagnosticsBundle = {
  schema_version: "diag.v1";
  source: "web" | "ui-tui";
  exported_at: string;
  session_id: string;
  turn_id: string;
  stream_mode: boolean;
  configured_transport: "ws" | "sse";
  active_transport: "ws" | "sse";
  reconnect_status: string;
  reconnect_count: number;
  timeout_action: "wait" | "reconnect" | "cancel";
  read_timeout_sec: number;
  turn_timeout_sec: number;
  fallback_hint: string;
  last_error_code: string;
  error_text: string;
  events: DiagnosticsEvent[];
};

export function buildDiagnosticsBundle(input: Omit<DiagnosticsBundle, "schema_version" | "source" | "exported_at">): DiagnosticsBundle {
  return {
    schema_version: "diag.v1",
    source: "web",
    exported_at: new Date().toISOString(),
    ...input
  };
}
