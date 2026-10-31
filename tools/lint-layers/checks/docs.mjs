import fs from "node:fs";
import path from "node:path";
import { REPO_ROOT, failOnFindings } from "./helpers.mjs";

/** The ids of the verifiers, read from the check files. */
function listVerifierIds() {
  const checksDir = path.join(REPO_ROOT, "tools/lint-layers/checks");
  return fs
    .readdirSync(checksDir)
    .filter((file) => file.endsWith(".mjs"))
    .flatMap((file) => [...fs.readFileSync(path.join(checksDir, file), "utf8").matchAll(/^\s+id: "([a-z0-9-]+)",$/gm)])
    .map((match) => match[1]);
}

/** The verifiers and the README of the layer validator list each other. */
function findUnlistedVerifiers() {
  const readme = "tools/lint-layers/README.md";
  const lines = fs.readFileSync(path.join(REPO_ROOT, readme), "utf8").split("\n");
  const listed = new Map();
  lines.forEach((text, index) => {
    const match = text.match(/^\s*[-*]\s+`([a-z0-9-]+)`/);
    if (match) listed.set(match[1], index + 1);
  });
  const registered = new Set(listVerifierIds());
  const found = [];
  for (const id of registered) {
    if (!listed.has(id)) found.push({ file: readme, line: 1, symbol: `${id} is registered and not described` });
  }
  for (const [id, line] of listed) {
    if (/-/.test(id) && !registered.has(id) && !/^(?:libs|apps|plugins)/.test(id)) {
      found.push({ file: readme, line, symbol: `${id} is described and not registered` });
    }
  }
  return found;
}

/** Every verifier is described in the README of the layer validator, and every described one exists. */
export function verifyReadmeListsVerifiers() {
  failOnFindings("Layer validator README and verifiers disagree", findUnlistedVerifiers());
}

export const DOCS_VERIFIERS = [
  {
    id: "docs-list-verifiers",
    scope: "libs",
    module: "docs",
    name: "Layer Validator README Lists Every Registered Verifier",
    run: verifyReadmeListsVerifiers,
  },
];
