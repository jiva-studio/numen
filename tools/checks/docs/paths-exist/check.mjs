import fs from "node:fs";
import path from "node:path";
import { REPO_ROOT, failOnFindings, requireMatches, listTrackedFiles } from "../../lib/graph.mjs";

/** The roots a backticked path starts with to be read as a path of the repository. */
const PATH_ROOTS = ["libs", "apps", "plugins", "tools", ".agents"];

/** The documents that name folders and files of the repository, as patterns of tracked paths. */
const DOCUMENT_PATTERNS = [
  /^docs\/architecture\/target-layout\.md$/,
  /^\.agents\/skills\/nm-place-[^/]+\/SKILL\.md$/,
  /^\.agents\/rules\/architecture\.md$/,
  /(^|\/)AGENTS\.md$/,
];

const escapeRegExp = (text) => text.replace(/[.+?^${}()|[\]\\]/g, "\\$&");

/** The text of a backticked path with its line suffix, anchor and trailing punctuation removed. */
function cleanPath(token) {
  return token
    .replace(/[#:]\S*$/, "")
    .replace(/[.,;)]+$/, "")
    .replace(/\/+$/, "");
}

/** The regular expression of a path with `*`, `<name>`, `(X)`, `{a,b}` and `...`, or undefined for a plain path. */
function getPattern(token) {
  if (!/[*<{(]|\.\.\./.test(token)) return undefined;
  const source = token
    .split(/(\*\*|\*|<[^>]*>|\([^)]*\)|\{[^}]*\}|\.\.\.)/)
    .map((part) => {
      if (part === "**" || part === "...") return ".+";
      if (part === "*") return "[^/]*";
      if (part.startsWith("<") || part.startsWith("(")) return "[^/]+";
      if (part.startsWith("{")) return `(?:${part.slice(1, -1).split(",").map(escapeRegExp).join("|")})`;
      return escapeRegExp(part);
    })
    .join("");
  return new RegExp(`^${source}$`);
}

/**
 * The backticked paths in documents that name no file or folder of the tree. A path is a backticked text without
 * spaces that starts with `libs/`, `apps/`, `plugins/`, `tools/` or `.agents/`; a `*`, `<name>`, `{a,b}` or `...`
 * stands for part of a path and needs one match.
 *
 * @param {Array<{ file: string, text: string }>} documents The documents, with their text
 * @param {Iterable<string>} tracked The tracked files of the tree
 * @returns {{ findings: Array<{ file: string, line: number, symbol: string }>, read: number }}
 */
export function findMissingPaths(documents, tracked) {
  const existing = new Set();
  for (const file of tracked) {
    for (let dir = file; dir && dir !== "."; dir = path.posix.dirname(dir)) existing.add(dir);
  }
  const findings = [];
  let read = 0;
  for (const { file, text } of documents) {
    text.split("\n").forEach((line, index) => {
      for (const match of line.matchAll(/`([^`\s]+)`/g)) {
        const token = cleanPath(match[1]);
        if (!PATH_ROOTS.some((root) => token.startsWith(`${root}/`))) continue;
        read += 1;
        const pattern = getPattern(token);
        const found = pattern ? [...existing].some((one) => pattern.test(one)) : existing.has(token);
        if (!found) findings.push({ file, line: index + 1, symbol: `${match[1]} is in no file or folder of the tree` });
      }
    });
  }
  return { findings, read };
}

export function verifyDocumentsNameExistingPaths() {
  const tracked = listTrackedFiles();
  const documents = tracked
    .filter((file) => DOCUMENT_PATTERNS.some((pattern) => pattern.test(file)))
    .map((file) => ({ file, text: fs.readFileSync(path.join(REPO_ROOT, file), "utf8") }));
  const one = findMissingPaths(documents, tracked);
  requireMatches("Paths in documents", one.read);
  failOnFindings("A document names a path that does not exist in the tree", one.findings);
}

export const DOCUMENT_VERIFIERS = [
  {
    id: "documents-name-existing-paths",
    scope: "libs",
    module: "documents",
    name: "Paths in Backticks in the Architecture Documents and AGENTS.md Files Exist in the Tree",
    run: verifyDocumentsNameExistingPaths,
  },
];
