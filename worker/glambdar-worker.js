const fs = require("fs");
const path = require("path");

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

// Read request from stdin
let input = "";
try {
    input = fs.readFileSync(0, { encoding: "utf-8" });
} catch (e) {
    error("failed to read stdin");
}

let event = {};
try {
    event = input ? JSON.parse(input) : {};
} catch (e) {
    error("invalid JSON input");
}

// Build request object
const request = {
    headers: event.headers || {},
    body: event.body || "",
    async json() {
        try {
            return JSON.parse(event.body || "{}");
        } catch {
            throw new Error("invalid JSON body");
        }
    },
};

(async () => {
    try {
        const result = await handler(request);
        if (!result || typeof result !== "object") {
            error("handler must return an object");
        }
        console.log(JSON.stringify(result));
    } catch (e) {
        error(e.message);
    }
})();
