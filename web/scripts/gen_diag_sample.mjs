import fs from "node:fs";

const out = process.argv[2];
if (!out) {
  console.error("usage: node web/scripts/gen_diag_sample.mjs <output>");
  process.exit(2);
}

const payload = {
  schema_version: "diag.v1",
  source: "web",
  exported_at: new Date().toISOString(),
  session_id: "web-smoke",
  turn_id: "web-turn-1",
  stream_mode: true,
  configured_transport: "ws",
  active_transport: "sse",
  reconnect_status: "degraded",
  reconnect_count: 1,
  timeout_action: "reconnect",
  read_timeout_sec: 15,
  turn_timeout_sec: 120,
  fallback_hint: "fallback ws->sse",
  last_error_code: "timeout",
  error_text: "read timeout",
  events: [
    { ts: new Date().toISOString(), event: "assistant_message", data: { content: "partial" } },
    { ts: new Date().toISOString(), event: "result", data: { final_response: "ok" } }
  ]
};

fs.writeFileSync(out, JSON.stringify(payload, null, 2));
