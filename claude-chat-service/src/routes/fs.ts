import { Router } from "express";
import fs from "node:fs";
import path from "node:path";
import { config } from "../config.js";

export const fsRouter = Router();

/**
 * Browse subdirektori — dipakai client buat "pointing dir" sebelum bikin
 * session. Cuma list folder (bukan file). SENGAJA tidak dibatasi
 * cwdValidation di sini — itu larangan buat "cwd yang dipakai sesi chat"
 * (mis. gak boleh home root), bukan buat sekadar melihat isi folder saat
 * navigasi (browsing home root buat cari subfolder project harus tetap
 * bisa). Batasan yang berlaku cuma: absolute path & memang direktori.
 */
fsRouter.get("/fs/browse", (req, res) => {
  const targetPath = typeof req.query.path === "string" ? req.query.path : config.browseDefaultRoot;

  if (!path.isAbsolute(targetPath)) {
    return res.status(400).json({ error: "path harus berupa absolute path" });
  }

  let entries: fs.Dirent[];
  try {
    entries = fs.readdirSync(targetPath, { withFileTypes: true });
  } catch (err) {
    const message = err instanceof Error ? err.message : String(err);
    return res.status(400).json({ error: `Gagal baca direktori: ${message}` });
  }

  const directories = entries
    .filter((entry) => entry.isDirectory() && !entry.name.startsWith("."))
    .map((entry) => ({ name: entry.name, path: path.join(targetPath, entry.name) }))
    .sort((a, b) => a.name.localeCompare(b.name));

  res.json({ path: targetPath, parent: path.dirname(targetPath), directories });
});
