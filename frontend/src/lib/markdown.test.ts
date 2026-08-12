// @vitest-environment jsdom
import { describe, expect, it } from 'vitest';
import { renderMarkdown } from './markdown';

describe('renderMarkdown', () => {
  it('bold → <strong>', () => {
    expect(renderMarkdown('**tebal**')).toContain('<strong>tebal</strong>');
  });

  it('heading → <h1>', () => {
    expect(renderMarkdown('# Judul')).toContain('<h1>Judul</h1>');
  });

  it('list → <ul><li>', () => {
    const h = renderMarkdown('- a\n- b');
    expect(h).toContain('<ul>');
    expect(h).toContain('<li>a</li>');
    expect(h).toContain('<li>b</li>');
  });

  it('code fence → <pre><code>', () => {
    const h = renderMarkdown('```\nkode\n```');
    expect(h).toContain('<pre>');
    expect(h).toContain('<code');
    expect(h).toContain('kode');
  });

  it('inline code → <code>', () => {
    expect(renderMarkdown('`x`')).toContain('<code>x</code>');
  });

  it('XSS: <script> di-strip', () => {
    const h = renderMarkdown('halo <script>alert(1)</script> dunia');
    expect(h).not.toContain('<script>');
    expect(h).toContain('halo');
  });

  it('XSS: handler onerror di-strip', () => {
    const h = renderMarkdown('<img src=x onerror="alert(1)">');
    expect(h.toLowerCase()).not.toContain('onerror');
  });

  it('link dapat target=_blank + rel aman', () => {
    const h = renderMarkdown('[situs](https://contoh.com)');
    expect(h).toContain('href="https://contoh.com"');
    expect(h).toContain('target="_blank"');
    expect(h).toContain('rel="noopener noreferrer"');
  });

  it('input kosong aman', () => {
    expect(renderMarkdown('')).toBe('');
  });
});
