import fs from "node:fs";
import path from "node:path";
import ts from "typescript";
import { runTool } from "../../../../shared/tools.mjs";
import { REPO_ROOT, failOnFindings, requireMatches } from "../../../lib/graph.mjs";

/** The reads of a drawn box or of a declared style, which a story does not make: a story proves what a person sees or does. */
const BANNED_READS = new Set(["getBoundingClientRect", "getComputedStyle", "toHaveStyle"]);

const ALLOWLIST = path.join(import.meta.dirname, "allowlist.json");

/** The banned reads of a story file with their lines. */
export function findGeometryReads(file, text) {
  const tree = ts.createSourceFile(file, text, ts.ScriptTarget.Latest, true);
  const reads = [];
  const visit = (node) => {
    const name = ts.isPropertyAccessExpression(node) ? node.name.text : ts.isIdentifier(node) && !ts.isPropertyAccessExpression(node.parent) ? node.text : undefined;
    if (name && BANNED_READS.has(name)) reads.push({ name, line: tree.getLineAndCharacterOfPosition(node.getStart(tree)).line + 1 });
    ts.forEachChild(node, visit);
  };
  visit(tree);
  return reads;
}

/**
 * The story files that read a box or a style and are not allowed to, and the allowed files that read none or give no
 * reason. An allowed file proves that something is whole, inside or ordered, which jsdom cannot, and says so.
 *
 * @param {Array<[string, string]>} stories The story files with their text
 * @param {Array<{ file: string, reason: string }>} allowed
 */
export function findGeometryReadsOutsideAllowlist(stories, allowed) {
  const findings = [];
  const reads = new Map(stories.map(([file, text]) => [file, findGeometryReads(file, text)]));
  const allowedFiles = new Set(allowed.map((one) => one.file));
  for (const [file, found] of reads) {
    if (allowedFiles.has(file)) continue;
    for (const one of found) findings.push({ file, line: one.line, symbol: `a story reads ${one.name}; a story proves what a person sees or does, and a file that proves geometry as behavior is listed with its reason in story-geometry/allowlist.json` });
  }
  for (const one of allowed) {
    if (one.reason.trim() === "") findings.push({ file: one.file, line: 1, symbol: "an allowed story file has no reason" });
    if (!reads.has(one.file)) findings.push({ file: one.file, line: 1, symbol: "an allowed story file is not a tracked story file" });
    else if (reads.get(one.file).length === 0) findings.push({ file: one.file, line: 1, symbol: "an allowed story file reads no box and no style; remove it from the allow-list" });
  }
  return findings;
}

export function verifyStoriesReadNoGeometryOutsideAllowlist() {
  const listed = runTool("git", ["ls-files", "-z", "--", "apps", "libs", "plugins"], { cwd: REPO_ROOT, encoding: "utf8", maxBuffer: 1 << 28 });
  const files = (listed.stdout || "").split("\0").filter((file) => file.endsWith(".stories.ts") && fs.existsSync(path.join(REPO_ROOT, file)));
  requireMatches("Story files", files.length);
  const stories = files.map((file) => [file, fs.readFileSync(path.join(REPO_ROOT, file), "utf8")]);
  const allowed = JSON.parse(fs.readFileSync(ALLOWLIST, "utf8"));
  failOnFindings("A story reads a box or a style outside the allow-list", findGeometryReadsOutsideAllowlist(stories, allowed));
}

export const STORY_GEOMETRY_VERIFIERS = [
  {
    id: "story-reads-no-geometry",
    scope: "libs",
    module: "ui",
    name: "No Story Reads getBoundingClientRect, getComputedStyle or toHaveStyle Unless Its File Is Allowed With a Reason",
    run: verifyStoriesReadNoGeometryOutsideAllowlist,
  },
];
