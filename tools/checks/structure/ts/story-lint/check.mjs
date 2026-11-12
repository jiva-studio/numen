import { spawnSync } from "node:child_process";
import { createRequire } from "node:module";
import path from "node:path";
import { pathToFileURL } from "node:url";
import { runTool } from "../../../../shared/tools.mjs";
import { REPO_ROOT, failOnFindings, requireMatches } from "../../../lib/graph.mjs";

/** The rule the story rules leave off: a story's component is found through the import graph. */
const RULE_LEFT_OFF = "storybook/csf-component";

/** The libraries whose story files are linted with the rules of the Storybook plugin. */
const STORY_LIBRARIES = ["libs/ui/components", "libs/ui/workbench"];

/** The severity of a rule entry as `eslint --print-config` prints it (`[2, ...]`), or as a config writes it (`"error"`, `2`). */
export function readSeverity(entry) {
  const level = Array.isArray(entry) ? entry[0] : entry;
  if (level === "error" || level === 2) return "error";
  if (level === "warn" || level === 1) return "warn";
  return "off";
}

/**
 * The gaps between the rules a story file resolves and the recommended rules of the Storybook plugin: a rule of
 * the plugin that is weaker or missing, and the rule that stays off but is on.
 *
 * @param {Record<string, unknown> | undefined} resolved The `rules` of `eslint --print-config` for a story file
 * @param {Record<string, unknown>} recommended The `rules` the plugin's recommended config sets for story files
 * @returns {string[]}
 */
export function findStoryRuleGaps(resolved, recommended) {
  if (!resolved || typeof resolved !== "object") return ["no resolved rules"];
  const expected = Object.entries(recommended).filter(([rule]) => rule.startsWith("storybook/") && rule !== RULE_LEFT_OFF);
  const gaps = expected.flatMap(([rule, entry]) => {
    const want = readSeverity(entry);
    const have = readSeverity(resolved[rule]);
    return want === have ? [] : [`${rule} is ${have}, the plugin recommends ${want}`];
  });
  if (readSeverity(resolved[RULE_LEFT_OFF]) !== "off") gaps.push(`${RULE_LEFT_OFF} is on`);
  return gaps;
}

/** The rules the recommended config of the installed plugin sets for story files. */
async function readRecommendedRules(dir) {
  const require = createRequire(path.join(REPO_ROOT, dir, "package.json"));
  const plugin = (await import(pathToFileURL(require.resolve("eslint-plugin-storybook")).href)).default;
  return plugin.configs["flat/recommended"].find((config) => config.name === "storybook:recommended:stories-rules")?.rules ?? {};
}

/** The first tracked story file under a library's `src`. */
function findStoryFile(dir) {
  const listed = runTool("git", ["ls-files", "--", `${dir}/src/*.stories.ts`], { cwd: REPO_ROOT, encoding: "utf8" });
  return (listed.stdout || "").split("\n").find(Boolean);
}

/** The resolved `rules` of a file, read by `eslint --print-config` run in its package. */
function readResolvedRules(dir, file) {
  const eslint = path.join(REPO_ROOT, "node_modules/.bin/eslint");
  const run = spawnSync(eslint, ["--print-config", path.relative(dir, file)], { cwd: path.join(REPO_ROOT, dir), encoding: "utf8", maxBuffer: 64 * 1024 * 1024 });
  if (run.status !== 0) throw new Error(`eslint --print-config ${file} failed: ${(run.stderr || run.stdout).trim()}`);
  return JSON.parse(run.stdout).rules;
}

export async function verifyStoryFilesResolveStorybookRules() {
  const findings = [];
  for (const dir of STORY_LIBRARIES) {
    const file = findStoryFile(dir);
    if (!file) {
      findings.push({ file: dir, line: 1, symbol: "no story file to read the lint configuration for" });
      continue;
    }
    const recommended = await readRecommendedRules(dir);
    requireMatches(`Storybook rules recommended for ${dir}`, Object.keys(recommended).filter((rule) => rule.startsWith("storybook/")).length);
    for (const gap of findStoryRuleGaps(readResolvedRules(dir, file), recommended)) findings.push({ file, line: 1, symbol: gap });
  }
  failOnFindings("A story file does not resolve the recommended Storybook rules", findings);
}

export const STORY_LINT_VERIFIERS = [
  {
    id: "story-files-lint-storybook-recommended",
    scope: "libs",
    module: "ui",
    name: "Story Files Resolve the Recommended Rules of eslint-plugin-storybook With csf-component Off",
    run: verifyStoryFilesResolveStorybookRules,
  },
];
