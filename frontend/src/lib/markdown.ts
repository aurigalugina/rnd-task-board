// Render Markdown → HTML aman untuk bubble chat assistant (Vision §6). Balasan
// Claude berformat markdown; di-parse pakai `marked` lalu DISANITASI pakai
// DOMPurify (marked meneruskan raw HTML apa adanya, jadi sanitasi wajib —
// konsisten dgn kultur anti-XSS repo, lihat lib/comments.ts). Lihat
// docs/decision-log/decision-log-chat-markdown-rendering-20260811.md.
import { marked } from 'marked';
import DOMPurify from 'dompurify';

marked.setOptions({ gfm: true, breaks: true });

// Link buka di tab baru + rel aman, biar klik link tidak menendang user keluar
// SPA. Hook cuma didaftarkan di browser (satu-satunya tempat sanitasi jalan).
if (typeof window !== 'undefined') {
  DOMPurify.addHook('afterSanitizeAttributes', (node) => {
    if (node.tagName === 'A') {
      node.setAttribute('target', '_blank');
      node.setAttribute('rel', 'noopener noreferrer');
    }
  });
}

export function renderMarkdown(md: string): string {
  const html = marked.parse(md ?? '', { async: false }) as string;
  // Sanitasi butuh DOM — hanya jalan di browser. Di server/prerender tidak
  // pernah ada pesan chat, jadi cabang ini aman (tidak butuh jsdom di bundle).
  if (typeof window === 'undefined') return html;
  return DOMPurify.sanitize(html, { USE_PROFILES: { html: true } });
}
