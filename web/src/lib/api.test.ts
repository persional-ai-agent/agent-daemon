import { describe, expect, it } from "vitest";
import { createTransportFallbackEvent, normalizeAPIError, streamEventDedupeKey } from "./api";
import { buildDiagnosticsBundle } from "./diagnostics";

describe("normalizeAPIError", () => {
  it("prefers structured error fields", () => {
    const out = normalizeAPIError({
      error_code: "timeout",
      error: "turn timeout",
      error_detail: { code: "timeout", message: "turn timeout detail" }
    });
    expect(out.code).toBe("timeout");
    expect(out.message).toBe("turn timeout");
  });

  it("falls back to reason/detail", () => {
    const out = normalizeAPIError({
      reason: "request cancelled",
      error_detail: { code: "cancelled", message: "request cancelled" }
    });
    expect(out.code).toBe("cancelled");
    expect(out.message).toBe("request cancelled");
  });
});

describe("streamEventDedupeKey", () => {
  it("returns stable key for identical event payload", () => {
    const a = { event: "assistant_message", data: { content: "partial" } };
    const b = { event: "assistant_message", data: { content: "partial" } };
    expect(streamEventDedupeKey(a)).toBe(streamEventDedupeKey(b));
  });
});

describe("ws payload shape", () => {
  it("dedupe key differs for resumed vs non-resumed event", () => {
    const a = { event: "resumed", data: { resumed: true, turn_id: "t1" } };
    const b = { event: "session", data: { session_id: "s1" } };
    expect(streamEventDedupeKey(a)).not.toBe(streamEventDedupeKey(b));
  });
});

describe("createTransportFallbackEvent", () => {
  it("creates schema-stable fallback event", () => {
    const out = createTransportFallbackEvent("ws", "sse", "WS_CLOSED", "2026-05-13T10:00:00.000Z");
    expect(out.event).toBe("transport_fallback");
    expect(out.data).toEqual({
      from: "ws",
      to: "sse",
      reason: "WS_CLOSED",
      at: "2026-05-13T10:00:00.000Z"
    });
  });
});

describe("buildDiagnosticsBundle", () => {
  it("returns stable v1 bundle shape", () => {
    const out = buildDiagnosticsBundle({
      session_id: "s1",
      turn_id: "t1",
      stream_mode: true,
      configured_transport: "ws",
      active_transport: "sse",
      reconnect_status: "degraded",
      reconnect_count: 1,
      timeout_action: "reconnect",
      read_timeout_sec: 15,
      turn_timeout_sec: 120,
      fallback_hint: "ws->sse",
      last_error_code: "timeout",
      error_text: "read timeout",
      events: []
    });
    expect(out.schema_version).toBe("diag.v1");
    expect(out.source).toBe("web");
    expect(out.session_id).toBe("s1");
    expect(out.active_transport).toBe("sse");
  });
});
