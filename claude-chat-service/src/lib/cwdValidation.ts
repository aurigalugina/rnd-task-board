import path from "node:path";

export type CwdValidationResult = { ok: true } | { ok: false; reason: string };

/**
 * Validasi RINGAN buat direktori yang mau dijadiin cwd sesi Claude Code.
 * Keputusan sadar (lihat decision-log-claude-chat-service-20260811.md): BUKAN
 * allowlist path yang di-maintain manual — cukup tolak target yang jelas
 * "terlalu luas" (root filesystem, home directory itu sendiri, atau root
 * lain yang dikonfigurasi via CWD_FORBIDDEN_ROOTS). Folder project di DALAM
 * home (mis. `~/projects/foo`) tetap valid.
 *
 * Fungsi ini murni logic path string — TIDAK menyentuh filesystem (cek
 * exists/is-directory dilakukan terpisah di pemanggil, di boundary I/O).
 */
export function validateCwd(
  targetPath: string,
  homeDir: string,
  extraForbiddenRoots: string[] = [],
): CwdValidationResult {
  if (!targetPath || !path.isAbsolute(targetPath)) {
    return { ok: false, reason: "cwd harus berupa absolute path" };
  }

  const normalizedTarget = path.normalize(targetPath).replace(/[/\\]+$/, "") || path.sep;
  const normalizedHome = path.normalize(homeDir).replace(/[/\\]+$/, "") || path.sep;

  const forbidden = [path.parse(normalizedTarget).root, normalizedHome, ...extraForbiddenRoots].map(
    (p) => path.normalize(p).replace(/[/\\]+$/, "") || path.sep,
  );

  if (forbidden.includes(normalizedTarget)) {
    return {
      ok: false,
      reason: `"${targetPath}" terlalu luas (root filesystem atau home directory global) — pilih folder project di dalamnya`,
    };
  }

  return { ok: true };
}
