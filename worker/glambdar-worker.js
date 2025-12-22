const fs = require("fs");
const path = require("path");
const net = require("net");

function error(msg) {
    console.error(JSON.stringify({ error: msg }));
    process.exit(1);
}

const funcDir = process.argv[2];
if (!funcDir) error("missing function directory as argument");

const indexJs = path.join(funcDir, "index.js");
if (!fs.existsSync(indexJs))
    error(`index.js not found in the function directory '${funcDir}'`);

let handler;
try {
    const mod = require(indexJs);
    handler = mod.handler;
    if (typeof handler !== "function") {
        error("exports.handler must be a function");
    }
} catch (e) {
    error("failed to load handler: " + e.message);
}

const client = net.createConnection({
    path: "/glambdar/glambdar.sock"
});

client.on("data", (data) => {
    (async () => {
        const req = JSON.parse(data.toString());
        req.json = async () => {
            try {
                return JSON.parse(req.body || null);
            } catch {
                throw new Error("invalid JSON body");
            }
        }

        const res = await handler(req);
        client.write(JSON.stringify(res))
    })();
});
