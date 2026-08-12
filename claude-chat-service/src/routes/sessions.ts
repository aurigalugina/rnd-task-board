import { Router } from "express";
import { z } from "zod";
import os from "node:os";
import fs from "node:fs";
import { closeSession, createSession, getSession, listSessions } from "../sessionManager.js";
import { validateCwd } from "../lib/cwdValidation.js";
import { config } from "../config.js";

const PERMISSION_MODES = ["default", "acceptEdits", "bypassPermissions", "plan", "dontAsk", "auto"] as const;

const createSessionSchema = z.object({
  cwd: z.string().min(1),
  model: z.string().optional(),
  permissionMode: z.enum(PERMISSION_MODES).optional(),
  resume: z.string().optional(),
});

export const sessionsRouter = Router();

sessionsRouter.post("/sessions", (req, res) => {
  const parsed = createSessionSchema.safeParse(req.body);
  if (!parsed.success) {
    return res.status(400).json({ error: parsed.error.flatten() });
  }

  const { cwd, model, permissionMode, resume } = parsed.data;

  const cwdCheck = validateCwd(cwd, os.homedir(), config.forbiddenRoots);
  if (!cwdCheck.ok) {
    return res.status(400).json({ error: cwdCheck.reason });
  }
  if (!fs.existsSync(cwd) || !fs.statSync(cwd).isDirectory()) {
    return res.status(400).json({ error: `"${cwd}" bukan direktori yang valid` });
  }

  const session = createSession({ cwd, model, permissionMode, resume });
  res.status(201).json(session.toSummary());
});

sessionsRouter.get("/sessions", (_req, res) => {
  res.json(listSessions().map((s) => s.toSummary()));
});

sessionsRouter.get("/sessions/:id", (req, res) => {
  const session = getSession(req.params.id);
  if (!session) return res.status(404).json({ error: "session tidak ditemukan" });
  res.json(session.toSummary());
});

sessionsRouter.delete("/sessions/:id", (req, res) => {
  const closed = closeSession(req.params.id);
  if (!closed) return res.status(404).json({ error: "session tidak ditemukan" });
  res.status(204).end();
});
