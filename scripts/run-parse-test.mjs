#!/usr/bin/env node

import { execSync } from "node:child_process";
import fs from "node:fs";
import { fileURLToPath } from "node:url";
import path from "node:path";

const __filename = fileURLToPath(import.meta.url);
const __dirname = path.dirname(__filename);
const repoRoot = path.resolve(__dirname, "..");

function fail(message, details) {
  console.error(`Error: ${message}`);
  if (details) {
    console.error(details);
  }
  process.exit(1);
}

function resolveTargetPath(inputPath) {
  const candidate = path.resolve(repoRoot, inputPath);

  if (!fs.existsSync(candidate)) {
    fail(`target does not exist: ${inputPath}`);
  }

  const stats = fs.statSync(candidate);
  if (stats.isDirectory()) {
    const indexPaths = findIndexMarkdownFiles(candidate);
    if (indexPaths.length === 0) {
      fail(`index.md not found under directory: ${inputPath}`);
    }
    if (indexPaths.length > 1) {
      const matches = indexPaths
        .map((indexPath) => path.relative(repoRoot, indexPath))
        .join("\n");
      fail(`multiple index.md files found under directory: ${inputPath}`, matches);
    }
    return path.relative(repoRoot, indexPaths[0]);
  }

  return path.relative(repoRoot, candidate);
}

function findIndexMarkdownFiles(directoryPath) {
  const results = [];
  const entries = fs.readdirSync(directoryPath, { withFileTypes: true });

  for (const entry of entries) {
    const entryPath = path.join(directoryPath, entry.name);
    if (entry.isDirectory()) {
      results.push(...findIndexMarkdownFiles(entryPath));
      continue;
    }
    if (entry.isFile() && entry.name === "index.md") {
      results.push(entryPath);
    }
  }

  return results.sort();
}

function readTestSpec(targetPath) {
  const parseCommand = `go run cmd/mdxs-parser/main.go parse ${shellQuote(targetPath)} --json`;

  let raw;
  try {
    raw = execSync(parseCommand, {
      cwd: repoRoot,
      encoding: "utf8",
      stdio: ["ignore", "pipe", "pipe"],
      shell: "/bin/bash",
    });
  } catch (error) {
    fail("failed to build test specification", formatCommandError(error));
  }

  let parsed;
  try {
    parsed = JSON.parse(raw);
  } catch (error) {
    fail(
      "failed to parse JSON output from test specification command",
      error instanceof Error ? error.message : String(error),
    );
  }

  const bashCommand = parsed?.body?.Test?.Command?.bash;
  const expectedJson = parsed?.body?.Test?.Expected?.json;

  if (typeof bashCommand !== "string" || bashCommand.length === 0) {
    fail("missing required field body.Test.Command.bash");
  }

  if (typeof expectedJson !== "string") {
    fail("missing required field body.Test.Expected.json");
  }

  return { bashCommand, expectedJson };
}

function shellQuote(value) {
  return `'${value.replace(/'/g, `'\\''`)}'`;
}

function runCommand(command) {
  try {
    const output = execSync(command, {
      cwd: repoRoot,
      encoding: "utf8",
      stdio: ["ignore", "pipe", "pipe"],
      shell: "/bin/bash",
    });
    return output.replace(/\r?\n$/, "");
  } catch (error) {
    fail(`test command failed: ${command}`, formatCommandError(error));
  }
}

function formatCommandError(error) {
  if (!(error instanceof Error)) {
    return String(error);
  }

  const lines = [];
  if ("status" in error && error.status !== null && error.status !== undefined) {
    lines.push(`exit status: ${error.status}`);
  }
  if ("stdout" in error && error.stdout) {
    lines.push(`stdout:\n${String(error.stdout)}`);
  }
  if ("stderr" in error && error.stderr) {
    lines.push(`stderr:\n${String(error.stderr)}`);
  }
  if (lines.length === 0) {
    lines.push(error.message);
  }
  return lines.join("\n");
}

const targetArg = process.argv[2];
if (!targetArg) {
  fail("missing required argument: provide a markdown file or a directory containing index.md");
}

const targetPath = resolveTargetPath(targetArg);
const { bashCommand, expectedJson } = readTestSpec(targetPath);
const actualJson = runCommand(bashCommand);

if (actualJson !== expectedJson) {
  console.error("Parse test failed: output did not match Expected.json");
  console.error("");
  console.error("Expected:");
  console.error(expectedJson);
  console.error("");
  console.error("Actual:");
  console.error(actualJson);
  process.exit(1);
}

console.log(
  `Parse test passed for ${targetPath}: output matched Expected.json`,
);
