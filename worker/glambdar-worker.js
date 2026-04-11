const fs = require("fs");
const path = require("path");
const net = require("net");

const funcDir = process.argv[2];
const SOCKET_PATH = "/glambdar-sock/glambdar.sock";

if (!funcDir) {
    console.error("missing function directory argument");
    process.exit(1);
}

// Load the handler once at startup (cold start)
let handler;
try {
    const indexJs = path.join(funcDir, "index.js");

    if (!fs.existsSync(indexJs)) {
        console.error("index.js not found in function directory");
        process.exit(1);
    }

    const mod = require(indexJs);
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
if (fs.existsSync(SOCKET_PATH)) {
    fs.unlinkSync(SOCKET_PATH);
}

// Create UDS server with allowHalfOpen enabled so we can respond after receiving EOF
const server = net.createServer({ allowHalfOpen: true }, (conn) => {
    let chunks = [];

    conn.on("data", (data) => {
        chunks.push(data);
    });

    conn.on("end", async () => {
        try {
            const req = JSON.parse(Buffer.concat(chunks).toString());

            req.json = async () => {
                try {
                    return JSON.parse(req.body || null);
                } catch {
                    throw new Error("invalid JSON body");
                }
            };

            const res = await handler(req);

            conn.end(JSON.stringify(res));
        } catch (err) {
            conn.end(JSON.stringify({
                statusCode: 500,
                body: { error: err.message || "function execution failed" }
            }));
        }
    });

    conn.on("error", (err) => {
        console.error("connection error:", err.message);
    });
});

server.listen(SOCKET_PATH, () => {
    console.log("glambdar worker listening on", SOCKET_PATH);
    try {
        // Ensure host has permissions to connect to the socket
        fs.chmodSync(SOCKET_PATH, 0o777);
    } catch (err) {
        console.error("failed to chmod socket:", err.message);
    }
});

// Graceful shutdown
process.on("SIGTERM", () => {
    server.close(() => {
        process.exit(0);
    });
});

server.on("error", (err) => {
    console.error("server error:", err.message);
    process.exit(1);
});
