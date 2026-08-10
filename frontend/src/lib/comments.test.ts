import { describe, expect, it } from 'vitest';
import { applyMention, escapeHtml, mentionQuery, renderCommentHtml } from './comments';

describe('escapeHtml', () => {
  it('escapes HTML-significant characters', () => {
    expect(escapeHtml('<script>alert(1)</script>')).toBe('&lt;script&gt;alert(1)&lt;/script&gt;');
  });

  it('escapes quotes', () => {
    expect(escapeHtml(`He said "hi" & left`)).toBe('He said &quot;hi&quot; &amp; left');
  });
});

describe('renderCommentHtml', () => {
  it('wraps a mention that matches an assignable user name', () => {
    expect(renderCommentHtml('cc @Mul please check', ['Mul', 'Rani'])).toBe(
      'cc <span class="mention-tag">@Mul</span> please check'
    );
  });

  it('does not wrap text that is not a known name', () => {
    expect(renderCommentHtml('email me at @random', ['Mul'])).toBe('email me at @random');
  });

  it('escapes HTML in the body even when a mention is present', () => {
    expect(renderCommentHtml('<b>@Mul</b>', ['Mul'])).toBe(
      '&lt;b&gt;<span class="mention-tag">@Mul</span>&lt;/b&gt;'
    );
  });

  it('prefers the longest matching name to avoid partial-match collisions', () => {
    expect(renderCommentHtml('@Rani Putri ok', ['Rani', 'Rani Putri'])).toBe(
      '<span class="mention-tag">@Rani Putri</span> ok'
    );
  });

  it('returns escaped text unchanged when there are no names to match', () => {
    expect(renderCommentHtml('no mentions here', [])).toBe('no mentions here');
  });
});

describe('mentionQuery', () => {
  it('detects an in-progress @mention at the end of the text', () => {
    expect(mentionQuery('halo @mu')).toBe('mu');
  });

  it('detects a bare @ with empty query', () => {
    expect(mentionQuery('halo @')).toBe('');
  });

  it('returns null when not composing a mention', () => {
    expect(mentionQuery('halo semua')).toBeNull();
  });

  it('ignores an @ that is not at the end of the text', () => {
    expect(mentionQuery('halo @mul, apa kabar')).toBeNull();
  });
});

describe('applyMention', () => {
  it('replaces the in-progress mention with the full name plus a trailing space', () => {
    expect(applyMention('halo @mu', 'Mul')).toBe('halo @Mul ');
  });
});
