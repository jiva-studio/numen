import { spawnSync } from "node:child_process";
import fs from "node:fs";
import path from "node:path";
import { REPO_ROOT, failOnFindings, requireMatches } from "./helpers.mjs";

/** The error severity of a resolved ESLint rule. */
const ERROR = 2;

/** The directories under `plugins/` that are templates, not plugins. */
const isTemplateDirectory = (name) => name.startsWith(".");

/**
 * The rule set every package enforces on every TypeScript file, tests included. Each entry names
 * the rule, the option that must hold, and the reason the default is not enough.
 */
export const STRICT_RULES = [
  { rule: "no-void", isOptionValid: (options) => options?.allowAsStatement !== true },
  { rule: "no-empty", isOptionValid: (options) => options?.allowEmptyCatch === false },
  { rule: "@typescript-eslint/no-empty-function" },
  { rule: "@typescript-eslint/no-floating-promises", isOptionValid: (options) => options?.ignoreVoid === false },
  { rule: "@typescript-eslint/no-misused-promises" },
  { rule: "@typescript-eslint/await-thenable" },
  { rule: "@typescript-eslint/switch-exhaustiveness-check" },
];

/** The rule set every package of plain scripts (`.mjs`) enforces; the typed rules need a TypeScript parser. */
export const SCRIPT_STRICT_RULES = [
  { rule: "no-void", isOptionValid: (options) => options?.allowAsStatement !== true },
  { rule: "no-empty", isOptionValid: (options) => options?.allowEmptyCatch === false },
  { rule: "no-empty-function" },
];

/** The severity and first option of a resolved rule entry (`[2, {...}]`, `2` or `"error"`). */
function readRuleEntry(entry) {
  const [level, options] = Array.isArray(entry) ? entry : [entry];
  const isError = level === ERROR || level === "error";
  return { isError, options };
}

/**
 * The gaps between the resolved rules of one file (`eslint --print-config`) and `STRICT_RULES`.
 *
 * @param {Record<string, unknown> | undefined} rules The `rules` object of the resolved config
 * @param {typeof STRICT_RULES} required The rule set the package must resolve
 * @returns {string[]} One description per missing or loosened rule
 */
export function findStrictRuleGaps(rules, required = STRICT_RULES) {
  if (!rules || typeof rules !== "object") return ["no resolved rules"];
  const gaps = [];
  for (const { rule, isOptionValid } of required) {
    const { isError, options } = readRuleEntry(rules[rule]);
    if (!isError) {
      gaps.push(`${rule} is not an error`);
    } else if (isOptionValid && !isOptionValid(options)) {
      gaps.push(`${rule} has options ${JSON.stringify(options ?? null)} that loosen it`);
    }
  }
  return gaps;
}

/**
 * The packages that lint TypeScript: every directory holding an `eslint.config.js` or `eslint.config.mjs`, plus every
 * `plugins/<name>/` that has a `package.json` (a plugin without a config is a gap).
 *
 * @param {string[]} trackedFiles Repository-relative paths of the tracked files
 * @param {(dir: string) => boolean} hasPackageJson Whether a repository-relative directory holds a package.json
 * @returns {Array<{ dir: string, hasConfig: boolean }>}
 */
export function listLintedPackages(trackedFiles, hasPackageJson) {
  const configured = new Set(
    trackedFiles.filter((file) => /^eslint\.config\.m?js$/.test(path.posix.basename(file))).map((file) => path.posix.dirname(file)),
  );
  const plugins = new Set(
    trackedFiles
      .map((file) => file.match(/^plugins\/([^/]+)\//)?.[1])
      .filter((name) => name && !isTemplateDirectory(name))
      .map((name) => `plugins/${name}`)
      .filter(hasPackageJson),
  );
  return Array.from(new Set([...configured, ...plugins]))
    .sort()
    .map((dir) => ({ dir, hasConfig: configured.has(dir) }));
}

/**
 * The files of a package the resolved config is read for: the first source file and the first
 * test file, so the rules are proven on both. A package with no such file (a template) is probed at
 * `probe.ts` and `probe.test.ts` in its root.
 *
 * @param {string[]} trackedFiles Repository-relative paths of the tracked files
 * @param {string} dir Repository-relative package directory
 * @returns {string[]}
 */
export function pickProbeFiles(trackedFiles, dir) {
  const own = trackedFiles.filter((file) => file.startsWith(`${dir}/`) && /\.ts$/.test(file) && !/\.d\.ts$/.test(file));
  const isTest = (file) => /\.(test|spec)\.ts$/.test(file);
  const isStory = (file) => /\.stories\.ts$/.test(file);
  const sources = own.filter((file) => !isTest(file) && !isStory(file) && !/(^|\/)[^/]*\.config\.ts$/.test(file));
  const tests = own.filter(isTest);
  return [sources[0] ?? `${dir}/probe.ts`, tests[0] ?? `${dir}/probe.test.ts`];
}

/**
 * The script files of a package the resolved config is read for: the first source and the first
 * test, as `pickProbeFiles` does for TypeScript.
 *
 * @param {string[]} trackedFiles Repository-relative paths of the tracked files
 * @param {string} dir Repository-relative package directory
 * @returns {string[]}
 */
export function pickScriptProbeFiles(trackedFiles, dir) {
  const own = trackedFiles.filter((file) => file.startsWith(`${dir}/`) && /\.mjs$/.test(file) && !/(^|\/)[^/]*\.config\.mjs$/.test(file));
  const isTest = (file) => /\.test\.mjs$/.test(file);
  return [own.find((file) => !isTest(file)), own.find(isTest)].filter(Boolean);
}

/** Whether a package holds TypeScript, and so resolves the typed rule set. */
function hasTypeScript(trackedFiles, dir) {
  return trackedFiles.some((file) => file.startsWith(`${dir}/`) && /\.(ts|vue)$/.test(file));
}

function readTrackedFiles() {
  const listed = spawnSync("git", ["ls-files"], { cwd: REPO_ROOT, encoding: "utf8", maxBuffer: 64 * 1024 * 1024 });
  if (listed.status !== 0) throw new Error(`git ls-files failed: ${listed.stderr}`);
  return listed.stdout.split("\n").filter((file) => file && fs.existsSync(path.join(REPO_ROOT, file)));
}

/** The resolved `rules` of a file, read by `eslint --print-config` run in its package. */
function readResolvedRules(dir, file) {
  const eslint = path.join(REPO_ROOT, "node_modules/.bin/eslint");
  const run = spawnSync(eslint, ["--print-config", path.relative(dir, file)], {
    cwd: path.join(REPO_ROOT, dir),
    encoding: "utf8",
    maxBuffer: 64 * 1024 * 1024,
  });
  if (run.status !== 0) throw new Error(`eslint --print-config ${file} failed: ${(run.stderr || run.stdout).trim()}`);
  return JSON.parse(run.stdout).rules;
}

/**
 * The packages the check leaves out. apps/landing: the website is being rewritten by its owner and keeps the
 * lint configuration it had. Remove this entry when that is done.
 */
const LEFT_OUT_PACKAGES = ["apps/landing"];

/** The smallest number of linted packages the repository has; fewer means discovery broke. */
const MINIMUM_PACKAGES = 16;

/** Every package with an ESLint config resolves the strict rule set on a source file and a test file. */
export function verifyStrictLintRules() {
  const tracked = readTrackedFiles();
  const packages = listLintedPackages(tracked, (dir) => fs.existsSync(path.join(REPO_ROOT, dir, "package.json"))).filter(
    ({ dir }) => !LEFT_OUT_PACKAGES.includes(dir),
  );
  requireMatches("Linted packages", packages.length, MINIMUM_PACKAGES);

  const findings = [];
  let probed = 0;
  for (const { dir, hasConfig } of packages) {
    if (!hasConfig) {
      findings.push({ file: dir, line: 1, symbol: "package has no eslint.config.js" });
      continue;
    }
    const isScriptPackage = !hasTypeScript(tracked, dir) && !dir.startsWith("plugins/");
    const probes = isScriptPackage ? pickScriptProbeFiles(tracked, dir) : pickProbeFiles(tracked, dir);
    const required = isScriptPackage ? SCRIPT_STRICT_RULES : STRICT_RULES;
    for (const probe of probes) {
      probed += 1;
      for (const gap of findStrictRuleGaps(readResolvedRules(dir, probe), required)) {
        findings.push({ file: probe, line: 1, symbol: gap });
      }
    }
  }
  requireMatches("Probed files", probed, MINIMUM_PACKAGES);
  failOnFindings("Package lint configuration lacks the strict rule set", findings);
}

export const LINT_CONFIG_VERIFIERS = [
  {
    id: "strict-lint-rules",
    scope: "libs",
    module: "lint",
    name: "Every Package Resolves the Strict Lint Rule Set (no-void, no-empty, no-empty-function, no-floating-promises, no-misused-promises)",
    run: verifyStrictLintRules,
  },
];
