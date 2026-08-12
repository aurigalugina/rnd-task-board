import type { Server as HttpServer } from "node:http";
import { WebSocketServer, type WebSocket } from "ws";
import { getSession } from "../sessionManager.js";
import type { ClientToServerMessage, ServerToClientMessage } from "../types.js";

const SESSION_WS_PATH = /^\/ws\/sessions\/([^/]+)$/;

function send(socket: WebSocket, message: ServerToClientMessage): void {
  if (socket.readyState === socket.OPEN) socket.send(JSON.stringify(message));
}

export function attachChatSocket(server: HttpServer): void {
  const wss = new WebSocketServer({ noServer: true });

  server.on("upgrade", (req, socket, head) => {
    const url = new URL(req.url ?? "", "http://internal");
    const match = SESSION_WS_PATH.exec(url.pathname);
    if (!match) {
      socket.destroy();
      return;
    }

    const session = getSession(match[1]);
    if (!session || session.status !== "active") {
      socket.write("HTTP/1.1 404 Not Found\r\n\r\n");
      socket.destroy();
      return;
    }

    wss.handleUpgrade(req, socket, head, (ws) => {
      wss.emit("connection", ws, session);
    });
  });

  wss.on("connection", (ws: WebSocket, session: NonNullable<ReturnType<typeof getSession>>) => {
    const unsubscribe = session.subscribe((message) => {
      send(ws, { type: "sdk_message", message });
    });

    ws.on("message", (raw) => {
      let parsed: ClientToServerMessage;
      try {
        parsed = JSON.parse(raw.toString());
      } catch {
        send(ws, { type: "error", message: "payload bukan JSON valid" });
        return;
      }

      switch (parsed.type) {
        case "prompt":
          session.sendPrompt(parsed.text, parsed.images);
          break;
        case "set_mode":
          session
            .setPermissionMode(parsed.mode)
            .then(() => send(ws, { type: "session_updated", session: session.toSummary() }))
            .catch((err) => send(ws, { type: "error", message: String(err) }));
          break;
        case "set_model":
          session
            .setModel(parsed.model)
            .then(() => send(ws, { type: "session_updated", session: session.toSummary() }))
            .catch((err) => send(ws, { type: "error", message: String(err) }));
          break;
        case "interrupt":
          session.interrupt().catch((err) => send(ws, { type: "error", message: String(err) }));
          break;
        default:
          send(ws, { type: "error", message: `tipe pesan tidak dikenal: ${JSON.stringify(parsed)}` });
      }
    });

    ws.on("close", unsubscribe);
  });
}
