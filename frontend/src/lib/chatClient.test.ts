import { describe, expect, it } from 'vitest';
import { buildChatWsUrl } from './chatClient';

describe('buildChatWsUrl', () => {
  it('http origin -> ws, menempel access_token', () => {
    expect(buildChatWsUrl('sess-1', 'http://localhost:5173', 'tok123')).toBe(
      'ws://localhost:5173/api/v1/chat/ws/sessions/sess-1?access_token=tok123'
    );
  });

  it('https origin -> wss', () => {
    expect(buildChatWsUrl('s', 'https://rndops.local', 'abc')).toBe(
      'wss://rndops.local/api/v1/chat/ws/sessions/s?access_token=abc'
    );
  });

  it('token di-encode (karakter khusus JWT aman)', () => {
    const url = buildChatWsUrl('s', 'http://x', 'a.b+c/d=');
    expect(url).toContain('access_token=a.b%2Bc%2Fd%3D');
  });

  it('tanpa token: tidak ada query access_token', () => {
    expect(buildChatWsUrl('s', 'http://x', null)).toBe('ws://x/api/v1/chat/ws/sessions/s');
  });
});
