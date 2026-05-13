import { describe, expect, it } from "vitest";
import { normalizeAPIError, streamEventDedupeKey } from "./api";

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
