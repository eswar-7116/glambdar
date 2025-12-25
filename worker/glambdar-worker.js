const fs = require("fs");
const path = require("path");
const net = require("net");

const funcDir = process.argv[2];
const SOCKET_PATH = "/glambdar/glambdar.sock";

if (!funcDir) {
    console.error("missing function directory argument");
    process.exit(1);
}

const client = net.createConnection({ path: SOCKET_PATH });

function sendError(message) {
    client.write(JSON.stringify({
        statusCode: 500,
        body: { error: message }
    }));
    process.exit(1);
}

let handler;

// Establish IPC
client.on("connect", () => {
    try {
        const indexJs = path.join(funcDir, "index.js");

        if (!fs.existsSync(indexJs)) {
            sendError("index.js not found in function directory");
        }

        const mod = require(indexJs);
        if (typeof mod.handler !== "function") {
            sendError("exports.handler must be a function");
        }

        handler = mod.handler;
    } catch (err) {
        sendError("failed to load handler: " + err.message);
    }
});

client.on("data", async (data) => {
    try {
        const req = JSON.parse(data.toString());

        req.json = async () => {
            try {
                return JSON.parse(req.body || null);
            } catch {
                throw new Error("invalid JSON body");
            }
        };

        const res = await handler(req);

        client.write(JSON.stringify(res));
    } catch (err) {
        sendError(err.message || "function execution failed");
    }
});

client.on("error", (err) => {
    console.error("UDS error:", err.message);
    process.exit(1);
});
