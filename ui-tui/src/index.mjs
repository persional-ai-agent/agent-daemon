import readline from "node:readline";
import { randomUUID } from "node:crypto";
import WebSocket from "ws";

let apiBase = process.env.AGENT_API_BASE || "ws://127.0.0.1:8080/v1/chat/ws";
let sessionID = process.env.AGENT_SESSION_ID || randomUUID();

const rl = readline.createInterface({
  input: process.stdin,
  output: process.stdout,
  prompt: "tui> "
});

function printEvent(evt) {
  const type = evt?.type || evt?.Type || "event";
  if (type === "assistant_message") {
    process.stdout.write(`\n[assistant] ${evt.content || ""}\n`);
    return;
  }
  if (type === "tool_started" || type === "tool_finished") {
    process.stdout.write(`\n[${type}] ${evt.tool_name || evt.ToolName || ""}\n`);
    return;
  }
  if (type === "result") {
    process.stdout.write(`\n[result] ${evt.final_response || ""}\n`);
    return;
  }
  if (type === "error") {
    process.stdout.write(`\n[error] ${evt.error || JSON.stringify(evt)}\n`);
    return;
  }
  process.stdout.write(`\n[${type}] ${JSON.stringify(evt)}\n`);
}

function printHelp() {
  process.stdout.write(
    "\ncommands:\n" +
      "/help                 show help\n" +
      "/session              show current session id\n" +
      "/session <id>         switch session id\n" +
      "/api                  show websocket endpoint\n" +
      "/api <ws-url>         switch websocket endpoint\n" +
      "/quit                 exit\n\n"
  );
}

function sendTurn(message) {
  return new Promise((resolve, reject) => {
    const ws = new WebSocket(apiBase);
    let resolved = false;

    ws.on("open", () => {
      ws.send(JSON.stringify({ session_id: sessionID, message }));
    });
    ws.on("message", (raw) => {
      try {
        const evt = JSON.parse(String(raw));
        printEvent(evt);
        const type = evt?.type || evt?.Type;
        if (type === "result" || type === "error" || type === "cancelled") {
          resolved = true;
          ws.close();
          resolve();
        }
      } catch (err) {
        process.stdout.write(`\n[decode-error] ${String(err)}\n`);
      }
    });
    ws.on("error", (err) => {
      if (!resolved) reject(err);
    });
    ws.on("close", () => {
      if (!resolved) resolve();
    });
  });
}

process.stdout.write(`session: ${sessionID}\n`);
process.stdout.write(`ws: ${apiBase}\n`);
process.stdout.write("输入 /help 查看命令\n");
rl.prompt();

rl.on("line", async (line) => {
  const text = line.trim();
  if (!text) {
    rl.prompt();
    return;
  }
  if (text === "/quit" || text === "/exit") {
    rl.close();
    return;
  }
  if (text === "/help") {
    printHelp();
    rl.prompt();
    return;
  }
  if (text === "/session") {
    process.stdout.write(`session: ${sessionID}\n`);
    rl.prompt();
    return;
  }
  if (text.startsWith("/session ")) {
    const next = text.slice("/session ".length).trim();
    if (!next) {
      process.stdout.write("session id required\n");
      rl.prompt();
      return;
    }
    sessionID = next;
    process.stdout.write(`session switched: ${sessionID}\n`);
    rl.prompt();
    return;
  }
  if (text === "/api") {
    process.stdout.write(`ws: ${apiBase}\n`);
    rl.prompt();
    return;
  }
  if (text.startsWith("/api ")) {
    const next = text.slice("/api ".length).trim();
    if (!next.startsWith("ws://") && !next.startsWith("wss://")) {
      process.stdout.write("api must start with ws:// or wss://\n");
      rl.prompt();
      return;
    }
    apiBase = next;
    process.stdout.write(`ws switched: ${apiBase}\n`);
    rl.prompt();
    return;
  }
  try {
    await sendTurn(text);
  } catch (err) {
    process.stdout.write(`\n[ws-error] ${String(err)}\n`);
  }
  rl.prompt();
});

rl.on("close", () => {
  process.stdout.write("bye\n");
  process.exit(0);
});
