import { spawnSync } from "node:child_process";
import fs from "node:fs";
import path from "node:path";
import { REPO_ROOT, failOnFindings } from "./helpers.mjs";

/** The documents that state where code lives and which rule or test enforces it. */
const DOCUMENT_PATTERNS = [
  /^AGENTS\.md$/,
  /^\.agents\/(rules|skills)\/.+\.md$/,
  /^(libs|apps)\/[^/]+(\/[^/]+)?\/AGENTS\.md$/,
  /^tools\/lint-layers\/README\.md$/,
];

/** The top-level folders a repository path in a document starts with. */
const PATH_ROOTS = ["modules", "libs", "apps", "plugins", "tools", "tests", "docs", ".agents", ".github"];

/** A backticked token or link target; the capture is the text between the delimiters. */
const REFERENCE = /`([^`\n]+)`|\]\(([^)\s#]+)(?:#[^)]*)?\)/g;

function listTracked(pathspecs) {
  const listed = spawnSync("git", ["ls-files", "--", ...pathspecs], { cwd: REPO_ROOT, encoding: "utf8" });
  return (listed.stdout || "").split("\n").filter(Boolean);
}

function listDocuments() {
  return listTracked(["*.md"]).filter((file) => DOCUMENT_PATTERNS.some((pattern) => pattern.test(file)));
}

/** The repository path a token names, and null where the token is not one. */
function getRepositoryPath(token) {
  const cleaned = token.replace(/^\.\//, "").replace(/[.,:;]+$/, "");
  const first = cleaned.split("/")[0];
  if (!PATH_ROOTS.includes(first) || !cleaned.includes("/")) return null;
  if (/[\s<>{}$=()]/.test(cleaned)) return null;
  return cleaned;
}

function pathExists(candidate, trackedFiles) {
  if (!candidate.includes("*")) {
    const bare = candidate.replace(/\/$/, "");
    return fs.existsSync(path.join(REPO_ROOT, bare));
  }
  const literal = candidate.split("*")[0];
  return trackedFiles.some((file) => file.startsWith(literal));
}

/** Resolves a link target against the document that holds it, and a backticked path against the root. */
function findMissingPaths() {
  const trackedFiles = listTracked(["."]);
  const found = [];
  for (const document of listDocuments()) {
    const lines = fs.readFileSync(path.join(REPO_ROOT, document), "utf8").split("\n");
    lines.forEach((text, index) => {
      for (const match of text.matchAll(REFERENCE)) {
        const isLink = match[2] !== undefined;
        const token = match[1] ?? match[2];
        if (/^[a-z]+:/.test(token)) continue;
        const resolved = isLink
          ? path.posix.normalize(path.posix.join(path.posix.dirname(document), token))
          : getRepositoryPath(token);
        if (!resolved || resolved.startsWith("..")) continue;
        if (!isLink || getRepositoryPath(resolved)) {
          if (!pathExists(resolved, trackedFiles)) found.push({ file: document, line: index + 1, symbol: resolved });
        }
      }
    });
  }
  return found;
}

/** Every path a rule document names exists in the working tree. */
export function verifyDocumentsNameExistingPaths() {
  failOnFindings("Documents name paths that do not exist", findMissingPaths());
}

/** The Go tests of `libs/core` and `apps/desktop` a document names by identifier. */
function findMissingTests() {
  const sources = listTracked(["libs/**/*_test.go", "apps/**/*_test.go"])
    .map((file) => fs.readFileSync(path.join(REPO_ROOT, file), "utf8"))
    .join("\n");
  const declared = new Set([...sources.matchAll(/^func (Test\w+)\(/gm)].map((match) => match[1]));
  const found = [];
  for (const document of listDocuments()) {
    const lines = fs.readFileSync(path.join(REPO_ROOT, document), "utf8").split("\n");
    lines.forEach((text, index) => {
      for (const match of text.matchAll(/`(Test[A-Z]\w*)`/g)) {
        if (!declared.has(match[1])) found.push({ file: document, line: index + 1, symbol: match[1] });
      }
    });
  }
  return found;
}

/** Every Go test a rule document names is declared. */
export function verifyDocumentsNameExistingTests() {
  failOnFindings("Documents name tests that do not exist", findMissingTests());
}

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
    id: "docs-name-existing-paths",
    scope: "libs",
    module: "docs",
    name: "Rule Documents Name Existing Paths (AGENTS.md, .agents/rules, lint-layers README)",
    run: verifyDocumentsNameExistingPaths,
  },
  {
    id: "docs-name-existing-tests",
    scope: "libs",
    module: "docs",
    name: "Rule Documents Name Declared Go Tests",
    run: verifyDocumentsNameExistingTests,
  },
  {
    id: "docs-list-verifiers",
    scope: "libs",
    module: "docs",
    name: "Layer Validator README Lists Every Registered Verifier",
    run: verifyReadmeListsVerifiers,
  },
];
