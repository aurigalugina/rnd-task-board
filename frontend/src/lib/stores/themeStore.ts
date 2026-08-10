// Tema disimpan di kolom users.theme_preference (backend) — store ini cuma
// state UI lokal, disinkronkan dari $auth.user.theme_preference oleh
// +layout.svelte, dan ditulis balik ke server oleh SettingsModal.
import { writable } from 'svelte/store';

export type ThemeName = 'retro-light' | 'retro-dark' | 'modern-light' | 'modern-dark';

export const THEMES: { key: ThemeName; label: string; titlebarA: string; face: string; winBlue: string }[] = [
  { key: 'retro-light', label: 'Retro light', titlebarA: '#34506E', face: '#DEE3E8', winBlue: '#34506E' },
  { key: 'retro-dark', label: 'Retro dark', titlebarA: '#0F2E44', face: '#232B33', winBlue: '#6FAEDE' },
  { key: 'modern-light', label: 'Modern light', titlebarA: '#FFFFFF', face: '#F3F3F3', winBlue: '#005FB8' },
  { key: 'modern-dark', label: 'Modern dark', titlebarA: '#202020', face: '#202020', winBlue: '#60CDFF' }
];

export const theme = writable<ThemeName>('retro-light');
