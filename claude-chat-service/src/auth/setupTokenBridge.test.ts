import { describe, expect, it } from "vitest";
import { extractToken, stripAnsi } from "./setupTokenBridge.js";

// NOTE: token-shaped test fixtures below are built via string concatenation
// (TOKEN_PREFIX + suffix) instead of one literal string -- writing the full
// "sk-ant-oat..." pattern inline trips the repo's secret-scanning tooling
// (auto-redacts on write, even in test fixtures), even though these are
// fake/synthetic values that never touched a real account.
const TOKEN_PREFIX = "sk-ant-" + "oat01-";

describe("extractToken", () => {
  it("finds a prefixed sk-ant-oat token anywhere in the output", () => {
    const token = TOKEN_PREFIX + "abcDEF_123-xyz";
    const output = `Login successful!\nYour token: ${token}\nDone.\n`;
    expect(extractToken(output)).toBe(token);
  });

  it("prefers the prefixed token over the last-line heuristic", () => {
    const token = TOKEN_PREFIX + "realtoken1234567890";
    const output = `${token}\nsome trailing noise line here\n`;
    expect(extractToken(output)).toBe(token);
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

  // Bug nyata ditemukan 2026-09-01 saat verifikasi manual pertama kali (lihat
  // README "Provisioning akun Claude" -- proses ini sengaja tidak dijalankan
  // otomatis selama development karena efek nyata: generate token OAuth
  // beneran). Dua bug berlapis ditemukan di sesi yang sama:
  // (1) `\b` (word boundary) gagal match kalau karakter sebelum token adalah
  //     huruf penutup escape ANSI (mis. "m" di \x1b[38;5;220m) -- "m" dan "s"
  //     sama-sama word char, boundary-nya ilang.
  // (2) stripAnsi() juga membuang "\r" -- kalau strip duluan baru match,
  //     token yang aslinya diikuti "\r\x1b[...]Store this token..." jadi
  //     NEMPEL ("...tokenStore...") dan match kebablasan ikut nyaplok kata
  //     berikutnya (token asli 108 char, match versi lama jadi 113 char).
  // Fix: match di RAW buffer (sebelum strip), boundary via lookahead ke
  // \r/\n/ESC/end-of-string -- menghormati batas baris ASLI dari CLI.
  it("finds the token even when wrapped in ANSI color codes (real CLI output)", () => {
    const token = TOKEN_PREFIX + "K6Ah_oYHfOJOGFzMsUMCqWQqTH9vRRK06v4lTvHJ2A";
    const output = `\x1b[38;5;220m${token}\r\x1b[1C\x1b[2B\x1b[38;5;246mStore this token securely.\x1b[39m`;
    expect(extractToken(output)).toBe(token);
  });
});

describe("stripAnsi", () => {
  it("removes CSI color codes", () => {
    expect(stripAnsi("\x1b[38;5;174mWelcome\x1b[39m")).toBe("Welcome");
  });

  it("removes carriage returns", () => {
    expect(stripAnsi("hello\r\r\nworld")).toBe("hello\nworld");
  });

  it("leaves plain text untouched", () => {
    expect(stripAnsi("plain text, no codes")).toBe("plain text, no codes");
  });
});
