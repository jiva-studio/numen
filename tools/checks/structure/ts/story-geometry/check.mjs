import fs from "node:fs";
import path from "node:path";
import ts from "typescript";
import { runTool } from "../../../../shared/tools.mjs";
import { REPO_ROOT, failOnFindings, requireMatches } from "../../../lib/graph.mjs";

/** The reads of a drawn box or of a declared style, which a story does not make: a story proves what a person sees or does. */
const BANNED_READS = new Set(["getBoundingClientRect", "measureRect", "measureElementRect", "getComputedStyle", "toHaveStyle"]);

/** The banned reads of a story file with their lines. */
export function findGeometryReads(file, text) {
  const tree = ts.createSourceFile(file, text, ts.ScriptTarget.Latest, true);
  const reads = [];
  const visit = (node) => {
    const isKey = ts.isStringLiteralLike(node) && ts.isElementAccessExpression(node.parent) && node.parent.argumentExpression === node;
    if ((ts.isIdentifier(node) || isKey) && BANNED_READS.has(node.text)) {
      reads.push({ name: node.text, line: tree.getLineAndCharacterOfPosition(node.getStart(tree)).line + 1 });
    }
    ts.forEachChild(node, visit);
  };
  visit(tree);
  return reads;
}

export function verifyStoriesReadNoGeometry() {
  const listed = runTool("git", ["ls-files", "-z", "--", "apps", "libs", "plugins"], { cwd: REPO_ROOT, encoding: "utf8", maxBuffer: 1 << 28 });
  const files = (listed.stdout || "").split("\0").filter((file) => file.endsWith(".stories.ts") && fs.existsSync(path.join(REPO_ROOT, file)));
  requireMatches("Story files", files.length);
  const findings = files.flatMap((file) =>
    findGeometryReads(file, fs.readFileSync(path.join(REPO_ROOT, file), "utf8")).map(({ line, name }) => ({
      file,
      line,
      symbol: `a story reads ${name}; assert observable behavior`,
    })),
  );
  failOnFindings("A story reads a box or a style", findings);
}

export const STORY_GEOMETRY_VERIFIERS = [
  {
    id: "story-reads-no-geometry",
    scope: "libs",
    module: "ui",
    name: "Stories Prove Browser Behavior Without Reading Element Boxes or Styles",
    run: verifyStoriesReadNoGeometry,
  },
];
