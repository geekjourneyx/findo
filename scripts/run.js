#!/usr/bin/env node

const fs = require("fs");
const path = require("path");
const { execFileSync } = require("child_process");

const ext = process.platform === "win32" ? ".exe" : "";
const binaryPath = path.join(__dirname, "..", "bin", `findo${ext}`);
const skillsPath = path.join(__dirname, "..", "skills");

if (!fs.existsSync(binaryPath)) {
  console.error(
    "findo binary is missing. Reinstall with `npm install -g @geekjourneyx/findo`."
  );
  process.exit(1);
}

try {
  const env = { ...process.env };
  if (!env.FINDO_SKILLS_DIR && fs.existsSync(skillsPath)) {
    env.FINDO_SKILLS_DIR = skillsPath;
  }
  execFileSync(binaryPath, process.argv.slice(2), { stdio: "inherit", env });
} catch (error) {
  if (typeof error.status === "number") {
    process.exit(error.status);
  }

  console.error(`Failed to launch findo: ${error.message}`);
  process.exit(1);
}
