import { describe, expect, it } from 'vitest';
import { extractErrorMessage, parseSseMessage } from './setupTokenClient';

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
