import { describe, expect, it } from "vitest";
import { extractToken } from "./setupTokenBridge.js";

describe("extractToken", () => {
  it("finds a prefixed sk-ant-oat token anywhere in the output", () => {
    const output = "Login successful!\nYour token: sk-ant-oat01-abcDEF_123-xyz\nDone.\n";
    expect(extractToken(output)).toBe("sk-ant-oat01-abcDEF_123-xyz");
  });

  it("prefers the prefixed token over the last-line heuristic", () => {
    const output = "sk-ant-oat01-realtoken1234567890\nsome trailing noise line here\n";
    expect(extractToken(output)).toBe("sk-ant-oat01-realtoken1234567890");
  });

  it("falls back to the last non-empty line when it looks token-shaped", () => {
    const output = "Authenticating...\r\nDone.\r\n\r\nabcDEF123456789012345_-\r\n";
    expect(extractToken(output)).toBe("abcDEF123456789012345_-");
  });

  it("returns undefined when the last line is plain prose (not token-shaped)", () => {
    const output = "Login successful! You can close this window.\n";
    expect(extractToken(output)).toBeUndefined();
  });

  it("returns undefined for empty output", () => {
    expect(extractToken("")).toBeUndefined();
  });
});
