import { describe, expect, it } from 'vitest';
import { extractErrorMessage, extractLoginUrl, parseSseMessage, stripAnsi } from './setupTokenClient';

describe('parseSseMessage', () => {
  it('parses a session event', () => {
    const raw = 'event: session\ndata: {"session_id":"abc-123"}';
    expect(parseSseMessage(raw)).toEqual({ type: 'session', sessionId: 'abc-123' });
  });

  it('parses an output event', () => {
    const raw = 'event: output\ndata: {"chunk":"Open this URL: https://..."}';
    expect(parseSseMessage(raw)).toEqual({ type: 'output', chunk: 'Open this URL: https://...' });
  });

  it('parses a successful done event', () => {
    const raw = 'event: done\ndata: {"success":true,"capturedAt":"2026-08-21T10:00:00Z"}';
    expect(parseSseMessage(raw)).toEqual({ type: 'done', success: true, capturedAt: '2026-08-21T10:00:00Z' });
  });

  it('parses a failed done event', () => {
    const raw = 'event: done\ndata: {"success":false,"reason":"Timeout menunggu login OAuth selesai"}';
    expect(parseSseMessage(raw)).toEqual({
      type: 'done',
      success: false,
      reason: 'Timeout menunggu login OAuth selesai'
    });
  });

  it('returns null when there is no data line', () => {
    expect(parseSseMessage('event: output')).toBeNull();
  });

  it('returns null on malformed JSON', () => {
    expect(parseSseMessage('event: output\ndata: not-json')).toBeNull();
  });

  it('returns null for an unknown event type', () => {
    expect(parseSseMessage('event: ping\ndata: {}')).toBeNull();
  });
});

describe('extractErrorMessage', () => {
  it('extracts the flat "error" field from a JSON body', () => {
    expect(extractErrorMessage(502, '{"error":"claude-chat-service tidak dapat dihubungi"}')).toBe(
      'claude-chat-service tidak dapat dihubungi'
    );
  });

  it('falls back to the raw text when the body is not JSON', () => {
    expect(extractErrorMessage(500, 'Internal Server Error')).toBe('Internal Server Error');
  });

  it('falls back to a generic message when the body is empty', () => {
    expect(extractErrorMessage(502, '')).toBe('Gagal memulai setup-token: 502');
  });

  it('falls back to the raw text when JSON has no string "error" field', () => {
    expect(extractErrorMessage(500, '{"other":"field"}')).toBe('{"other":"field"}');
  });
});

describe('stripAnsi', () => {
  it('removes CSI color codes', () => {
    expect(stripAnsi('\x1b[38;5;174mWelcome\x1b[39m')).toBe('Welcome');
  });

  it('removes cursor-positioning CSI codes', () => {
    expect(stripAnsi('\x1b[6G*\x1b[46G\u2588\u2588\u2588')).toBe('*\u2588\u2588\u2588');
  });

  it('removes two-character escape sequences and carriage returns', () => {
    expect(stripAnsi('\x1b7\x1b8\x1b[?25hhello\r\r\nworld')).toBe('hello\nworld');
  });

  it('leaves plain text untouched', () => {
    expect(stripAnsi('plain text, no codes')).toBe('plain text, no codes');
  });
});

describe('extractLoginUrl', () => {
  it('joins a URL wrapped across multiple lines by the terminal', () => {
    const output = [
      "Browser didn't open? Use the url below to sign in (c to copy)",
      '',
      'https://claude.com/cai/oauth/authorize?code=true&client_id=abc&redir',
      'ect_uri=https%3A%2F%2Fplatform.claude.com%2Foauth%2Fcode%2Fcallback&state=xyz',
      '',
      '',
      'Paste code here if prompted >'
    ].join('\n');
    expect(extractLoginUrl(output)).toBe(
      'https://claude.com/cai/oauth/authorize?code=true&client_id=abc&redirect_uri=https%3A%2F%2Fplatform.claude.com%2Foauth%2Fcode%2Fcallback&state=xyz'
    );
  });

  it('returns null when there is no URL in the output', () => {
    expect(extractLoginUrl('just some CLI banner text, no link yet')).toBeNull();
  });

  it('stops at "Paste code" even without a blank line separator', () => {
    const output = 'https://claude.com/cai/oauth/authorize?code=truePaste code here if prompted >';
    expect(extractLoginUrl(output)).toBe('https://claude.com/cai/oauth/authorize?code=true');
  });
});
