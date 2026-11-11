import { runTool } from "../../shared/tools.mjs";
import path from "node:path";
import stylelint from "stylelint";
import { REPO_ROOT, failOnFindings, requireMatches } from "./helpers.mjs";

/** The stylelint configuration of the repository. */
const STYLE_CONFIG = path.join(REPO_ROOT, "stylelint.config.mjs");

/** The findings of a stylelint report: every warning, and every option the configuration gets wrong. */
function readStyleFindings(report, root = REPO_ROOT) {
  return report.results.flatMap((result) => {
    const file = path.relative(root, result.source ?? "");
    return [
      ...result.warnings.map(({ line, text }) => ({ file, line, symbol: text })),
      ...(result.invalidOptionWarnings ?? []).map(({ text }) => ({ file, line: 1, symbol: text })),
      ...(result.parseErrors ?? []).map(({ line, text }) => ({ file, line, symbol: text })),
    ];
  });
}

/** The tracked stylesheets and components of the repository. */
function listStyleFiles() {
  const listed = runTool("git", ["ls-files", "-z", "--", "*.css", "*.vue"], { cwd: REPO_ROOT, encoding: "utf8", maxBuffer: 1 << 28 });
  return (listed.stdout || "").split("\0").filter(Boolean);
}

export async function verifyStyles() {
  const files = listStyleFiles();
  requireMatches("stylesheets and components read", files.length);
  const report = await stylelint.lint({ files: files.map((file) => path.join(REPO_ROOT, file)), configFile: STYLE_CONFIG, cwd: REPO_ROOT });
  failOnFindings("stylelint rejects a style", readStyleFindings(report));
}

export const STYLE_VERIFIERS = [
  {
    id: "styles-use-tokens",
    scope: "libs",
    module: "ui",
    name: "Styles Use Declared Tokens (stylelint: no color literal or --numen- declaration outside token and theme files, no var() fallback, no undeclared --numen- token, no length on a property)",
    run: verifyStyles,
  },
];
