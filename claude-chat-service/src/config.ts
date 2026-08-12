export const config = {
  port: Number(process.env.PORT ?? 8090),
  /**
   * Sesi Claude Code TIDAK boleh pointing ke path ini atau turunannya —
   * validasi ringan (bukan allowlist ketat), lihat lib/cwdValidation.ts.
   * Default: home directory user yang menjalankan proses ini.
   */
  forbiddenRoots: (process.env.CWD_FORBIDDEN_ROOTS ?? "").split(",").filter(Boolean),
  /** Root default untuk GET /fs/browse ketika query ?path= tidak diisi. */
  browseDefaultRoot: process.env.BROWSE_DEFAULT_ROOT ?? process.env.HOME ?? "/",
};
