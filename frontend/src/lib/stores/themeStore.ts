// Tema disimpan di kolom users.theme_preference (backend) — store ini cuma
// state UI lokal, disinkronkan dari $auth.user.theme_preference oleh
// +layout.svelte, dan ditulis balik ke server oleh SettingsModal.
import { writable } from 'svelte/store';

export type ThemeName = 'retro-light' | 'retro-dark' | 'modern-light' | 'modern-dark' | 'aurora-glass' | 'ocean-breeze';

export const THEMES: { key: ThemeName; label: string; titlebarA: string; face: string; winBlue: string }[] = [
  { key: 'retro-light',  label: 'Retro light',   titlebarA: '#34506E', face: '#DEE3E8', winBlue: '#34506E' },
  { key: 'retro-dark',   label: 'Retro dark',    titlebarA: '#00358a', face: '#1c2840', winBlue: '#5ab0e0' },
  { key: 'modern-light', label: 'Modern light',  titlebarA: '#6366f1', face: '#f1f5f9', winBlue: '#6366f1' },
  { key: 'modern-dark',  label: 'Modern dark',   titlebarA: '#1e293b', face: '#1e293b', winBlue: '#38bdf8' },
  { key: 'aurora-glass', label: 'Aurora Glass',  titlebarA: '#1e1b4b', face: '#16213e', winBlue: '#818cf8' },
  { key: 'ocean-breeze', label: 'Ocean Breeze',  titlebarA: '#0891b2', face: '#daeef5', winBlue: '#0891b2' }
];

export const theme = writable<ThemeName>('retro-light');
