import express from "express";
import http from "node:http";
import { config } from "./config.js";
import { sessionsRouter } from "./routes/sessions.js";
import { fsRouter } from "./routes/fs.js";
import { authRouter } from "./routes/auth.js";
import { attachChatSocket } from "./ws/chatSocket.js";
import { loadTokenIntoEnv } from "./auth/tokenStore.js";

loadTokenIntoEnv();

const app = express();
app.use(express.json());

app.get("/healthz", (_req, res) => res.json({ ok: true }));
app.use(sessionsRouter);
app.use(fsRouter);
app.use(authRouter);

const server = http.createServer(app);
attachChatSocket(server);

server.listen(config.port, () => {
  console.log(`claude-chat-service listening on :${config.port}`);
});
