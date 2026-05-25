import { join } from "node:path";
import { existsSync, unlinkSync, chmodSync } from "node:fs";

const funcDir = process.argv[2];
const SOCKET_PATH = "/glambdar-sock/glambdar.sock";

if (!funcDir) {
    console.error("missing function directory argument");
    process.exit(1);
}

// Load the handler once at startup (cold start)
let handler;
try {
    const indexJs = join(funcDir, "index.js");
    if (!existsSync(indexJs)) {
        console.error("index.js not found in function directory");
        process.exit(1);
    }

    const mod = await import(indexJs);
    if (typeof mod.handler !== "function") {
        console.error("exports.handler must be a function");
        process.exit(1);
    }

    handler = mod.handler;
} catch (err) {
    console.error("failed to load handler:", err.message);
    process.exit(1);
}

// Clean up stale socket file if it exists
if (existsSync(SOCKET_PATH)) {
    unlinkSync(SOCKET_PATH);
}

// Create UDS server with allowHalfOpen enabled
const server = Bun.listen({
    unix: SOCKET_PATH,
    socket: {
        data(socket, data) {
            // Accumulate buffer chunks directly on the socket instance
            socket._payloadChunks = socket._payloadChunks 
                ? Buffer.concat([socket._payloadChunks, data]) 
                : data;
        },
        async end(socket) {
            // Triggered on client EOF. Socket remains open for writing.
            try {
                const payload = socket._payloadChunks ? socket._payloadChunks.toString() : "{}";
                const req = JSON.parse(payload);

                req.json = async () => {
                    try {
                        return JSON.parse(req.body || null);
                    } catch {
                        throw new Error("invalid JSON body");
                    }
                };

                const res = await handler(req);
                socket.write(JSON.stringify(res));
                socket.end(); // Fully close connection
                
            } catch (err) {
                socket.write(JSON.stringify({
                    statusCode: 500,
                    body: { error: err.message || "function execution failed" }
                }));
                socket.end();
            }
        },
        error(_, err) {
            console.error("connection error:", err.message);
        }
    }
});

console.log("glambdar worker listening on", SOCKET_PATH);

try {
    // Ensure host has permissions to connect to the socket
    chmodSync(SOCKET_PATH, 0o777);
} catch (err) {
    console.error("failed to chmod socket:", err.message);
}

// Graceful shutdown
process.on("SIGTERM", () => {
    // Drain and close all active connections gracefully
    server.stop(true); 
    process.exit(0);
});
