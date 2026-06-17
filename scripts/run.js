#!/usr/bin/env node

const fs = require("fs");
const path = require("path");
const { execFileSync } = require("child_process");

const ext = process.platform === "win32" ? ".exe" : "";
const binaryPath = path.join(__dirname, "..", "bin", `findo${ext}`);

if (!fs.existsSync(binaryPath)) {
  console.error(
    "findo binary is missing. Reinstall with `npm install -g @geekjourneyx/findo`."
  );
  process.exit(1);
}

try {
  execFileSync(binaryPath, process.argv.slice(2), { stdio: "inherit" });
} catch (error) {
  if (typeof error.status === "number") {
    process.exit(error.status);
  }

  console.error(`Failed to launch findo: ${error.message}`);
  process.exit(1);
}
