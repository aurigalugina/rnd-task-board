// Render body komentar jadi HTML aman: escape dulu (body dari user, bisa berisi
// apa saja), baru bungkus token @nama yang cocok dengan salah satu nama
// assignable user jadi <span class="mention-tag">. Dipakai via {@html} di
// CommentSection.svelte — JANGAN pernah render body mentah tanpa escape.
export function escapeHtml(text: string): string {
  return text
    .replace(/&/g, '&amp;')
    .replace(/</g, '&lt;')
    .replace(/>/g, '&gt;')
    .replace(/"/g, '&quot;')
    .replace(/'/g, '&#39;');
}

function escapeRegExp(s: string): string {
  return s.replace(/[.*+?^${}()|[\]\\]/g, '\\$&');
}

export function renderCommentHtml(text: string, names: string[]): string {
  const escaped = escapeHtml(text);
  const uniqueNames = [...new Set(names.filter((n) => n.length > 0))];
  if (uniqueNames.length === 0) return escaped;

  // Nama terpanjang duluan supaya "Rani Putri" tidak keburu ke-match sebagian oleh "Rani".
  const sorted = uniqueNames.sort((a, b) => b.length - a.length);
  const pattern = sorted.map((n) => escapeRegExp(escapeHtml(n))).join('|');
  const regex = new RegExp(`@(${pattern})\\b`, 'g');
  return escaped.replace(regex, '<span class="mention-tag">@$1</span>');
}

// mentionQuery dipakai form komentar buat munculkan dropdown saran saat user
// mengetik "@" diikuti sebagian nama.
export function mentionQuery(text: string): string | null {
  const match = text.match(/@(\w*)$/);
  return match ? match[1] : null;
}

export function applyMention(text: string, name: string): string {
  return text.replace(/@(\w*)$/, `@${name} `);
}
