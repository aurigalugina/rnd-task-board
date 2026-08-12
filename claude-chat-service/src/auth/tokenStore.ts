import fs from "node:fs";
import path from "node:path";

/**
 * Penyimpanan lokal buat token hasil `claude setup-token` (lihat routes/auth.ts).
 * `claude setup-token` TIDAK menyimpan token itu sendiri (cuma dicetak sekali ke
 * stdout) — kita yang wajib nyimpen supaya restart service gak butuh setup ulang.
 *
 * CATATAN KEAMANAN: disimpan plaintext di disk lokal untuk sekarang (keputusan
 * sadar — auth/security service ini sengaja ditunda dulu, lihat
 * docs/decision-log/decision-log-claude-chat-service-20260811.md). WAJIB
 * dipindah ke secret store beneran sebelum dipakai di server yang genuinely
 * production/multi-tenant.
 */

const STORE_PATH = process.env.TOKEN_STORE_PATH ?? path.join(process.cwd(), "data", "claude-oauth-token.json");

type TokenRecord = { token: string; capturedAt: string };

export function saveToken(token: string): TokenRecord {
  const record: TokenRecord = { token, capturedAt: new Date().toISOString() };
  fs.mkdirSync(path.dirname(STORE_PATH), { recursive: true });
  fs.writeFileSync(STORE_PATH, JSON.stringify(record, null, 2), { mode: 0o600 });
  process.env.CLAUDE_CODE_OAUTH_TOKEN = token;
  return record;
}

export function loadTokenIntoEnv(): { present: boolean; capturedAt?: string } {
  if (!fs.existsSync(STORE_PATH)) return { present: false };
  const record = JSON.parse(fs.readFileSync(STORE_PATH, "utf8")) as TokenRecord;
  process.env.CLAUDE_CODE_OAUTH_TOKEN = record.token;
  return { present: true, capturedAt: record.capturedAt };
}

export function getTokenMeta(): { present: boolean; capturedAt?: string } {
  if (!fs.existsSync(STORE_PATH)) return { present: false };
  const record = JSON.parse(fs.readFileSync(STORE_PATH, "utf8")) as TokenRecord;
  return { present: true, capturedAt: record.capturedAt };
}
