import { spawnSync } from "node:child_process";
import fs from "node:fs";
import path from "node:path";
import ts from "typescript";
import { PACKAGE_DEPENDENCIES } from "./libraries.mjs";
import { REPO_ROOT, failOnFindings, requireMatches } from "./helpers.mjs";

/** The top-level directories that hold packages, and the kind of package each holds. */
const PACKAGE_KINDS = { libs: "library", apps: "application", plugins: "plugin" };

/** The first-party packages a plugin depends on. */
const PLUGIN_PACKAGES = ["@numen/plugin-sdk", "@numen/ui", "@numen/shell"];

/** The packages an application reaches the core without: it goes through `@numen/core-client`. */
const APPLICATION_EXCLUDED = ["@numen/plugin-sdk", "@numen/protocol"];

/** The third-party scope of the transport, which a plugin reaches only through the SDK. */
const TRANSPORT_SCOPE = "@connectrpc";

/** Directories a package walk does not enter: installed packages, build output, generated code and fixtures. */
const SKIPPED_DIRECTORIES = new Set(["node_modules", "dist", "gen", "testdata", ".git"]);

/** The configuration depcruise loads; it states the rules for the packages of the directory it runs in. */
const CRUISE_CONFIG = path.join(REPO_ROOT, "tools/lint-layers/dependency-cruiser.mjs");

/** The depcruise executable installed for the repository. */
const DEPCRUISE = path.join(REPO_ROOT, "node_modules/.bin/depcruise");

/** Every `package.json` below `dir`, outside the skipped directories, as paths relative to `root`. */
function findPackageFiles(root, dir) {
  const found = [];
  const abs = path.join(root, dir);
  if (!fs.existsSync(abs)) return found;
  for (const entry of fs.readdirSync(abs, { withFileTypes: true })) {
    const relative = path.posix.join(dir, entry.name);
    if (entry.isDirectory() && !SKIPPED_DIRECTORIES.has(entry.name)) found.push(...findPackageFiles(root, relative));
    else if (entry.isFile() && entry.name === "package.json") found.push(relative);
  }
  return found;
}

/** The packages under `libs`, `apps` and `plugins` of `root`, as `{ dir, name, kind }`. */
export function listPackages(root) {
  return Object.entries(PACKAGE_KINDS).flatMap(([top, kind]) =>
    findPackageFiles(root, top).map((file) => ({
      dir: path.posix.dirname(file),
      name: JSON.parse(fs.readFileSync(path.join(root, file), "utf8")).name,
      kind,
    })),
  );
}

const escapeRegExp = (text) => text.replace(/[.*+?^${}()|[\]\\]/g, "\\$&");

/**
 * The paths a dependency on a package resolves to: its directory, the installed copy, and the bare name
 * when the module is not resolved.
 */
function getPackagePattern({ dir, name }) {
  const named = escapeRegExp(name);
  return `^${escapeRegExp(dir)}/|(^|/)node_modules/${named}(/|$)|^${named}(/|$)`;
}

const getScopePattern = (scope) => `(^|/)node_modules/${escapeRegExp(scope)}/|^${escapeRegExp(scope)}/`;

/** The first-party packages `one` may depend on, or null for a library the package matrix has no row for. */
export function getAllowedPackages(one, packages, matrix = PACKAGE_DEPENDENCIES) {
  if (one.kind === "plugin") return PLUGIN_PACKAGES;
  if (one.kind === "application") {
    return packages.filter((other) => other.kind === "library" && !APPLICATION_EXCLUDED.includes(other.name)).map((other) => other.name);
  }
  return matrix[one.name] ?? null;
}

const getSlug = (dir) => dir.replace(/[^A-Za-z0-9]+/g, "-");

/** The rules of one package: the first-party packages it does not reach, and no relative path out of its directory. */
function createPackageRules(one, packages, matrix) {
  const allowed = getAllowedPackages(one, packages, matrix);
  const rules = [
    {
      name: `${getSlug(one.dir)}-stays-in-its-directory`,
      comment: `${one.name} reaches no file outside ${one.dir} by a relative path`,
      severity: "error",
      from: { path: `^${escapeRegExp(one.dir)}/` },
      to: { dependencyTypes: ["local"], dependencyTypesNot: ["aliased"], pathNot: `^${escapeRegExp(one.dir)}/` },
    },
  ];
  if (allowed === null) return rules;
  const forbidden = packages.filter((other) => other.name !== one.name && !allowed.includes(other.name)).map(getPackagePattern);
  if (one.kind === "plugin") forbidden.push(getScopePattern(TRANSPORT_SCOPE));
  if (forbidden.length > 0) {
    rules.push({
      name: `${getSlug(one.dir)}-reaches-only-its-packages`,
      comment: `${one.name} reaches only ${allowed.join(", ") || "no first-party package"}`,
      severity: "error",
      from: { path: `^${escapeRegExp(one.dir)}/` },
      to: { path: forbidden.join("|") },
    });
  }
  return rules;
}

/** The segments of a slice from the lowest: a segment imports only the segments below it. */
const SEGMENTS = ["lib", "api", "model", "ui"];

/** Tests and stories, which exercise a segment from outside and are not shipped. */
const NOT_SHIPPED = "\\.(test|spec|stories)\\.ts$";

function createSegmentRules() {
  return SEGMENTS.slice(0, -1).map((segment, index) => ({
    name: `segment-${segment}-imports-no-higher-segment`,
    comment: `a ${segment} segment imports nothing from ${SEGMENTS.slice(index + 1).join(", ")} of its slice`,
    severity: "error",
    from: { path: `^(.+)/${segment}/`, pathNot: NOT_SHIPPED },
    to: { path: `^$1/(${SEGMENTS.slice(index + 1).join("|")})/` },
  }));
}

/** The dependency-cruiser configuration for the packages under `root`. */
export function createCruiseConfig(root, matrix = PACKAGE_DEPENDENCIES) {
  const packages = listPackages(root);
  return {
    forbidden: [
      ...packages.flatMap((one) => createPackageRules(one, packages, matrix)),
      ...createSegmentRules(),
      {
        name: "plugin-host-reaches-only-vue",
        comment: "of third-party code the plugin host knows Vue, and its build helper Vite; its tests are not shipped",
        severity: "error",
        from: { path: "^libs/plugin-host-ts/src/", pathNot: NOT_SHIPPED },
        to: { path: "(^|/)node_modules/", pathNot: "(^|/)node_modules/(vue|@vue/[^/]+|vite)/" },
      },
      {
        name: "no-cross-slice-entry",
        comment: "there is no @x cross-import between slices",
        severity: "error",
        from: {},
        to: { path: "/@x/" },
      },
      {
        name: "no-circular",
        comment: "files form no cycle of runtime imports; a cycle only `import type` closes is erased",
        severity: "error",
        from: {},
        to: { circular: true, viaOnly: { dependencyTypesNot: ["type-only"] } },
      },
    ],
    options: {
      tsPreCompilationDeps: true,
      doNotFollow: { path: "node_modules" },
      moduleSystems: ["es6", "cjs"],
      enhancedResolveOptions: {
        exportsFields: ["exports"],
        conditionNames: ["import", "require", "node", "default", "types"],
        mainFields: ["module", "main", "types", "typings"],
        extensions: [".ts", ".tsx", ".d.ts", ".mts", ".js", ".mjs", ".cjs", ".vue", ".json"],
      },
    },
  };
}

/** The directory of a package that holds its sources: `src`, or `frontend` for a plugin. */
function getSourceRoot(root, one) {
  return ["src", "frontend"].map((name) => path.posix.join(one.dir, name)).find((dir) => fs.existsSync(path.join(root, dir)));
}

/** The `paths` a package's `tsconfig.json` resolves to, or an empty object. */
function readTsConfigPaths(file) {
  const read = ts.readConfigFile(file, ts.sys.readFile);
  if (read.error) return {};
  return ts.parseJsonConfigFileContent(read.config, ts.sys, path.dirname(file)).options.paths ?? {};
}

/**
 * The runs that cruise the source roots of `root`: a package whose `tsconfig.json` declares `paths` runs
 * alone with that file, so its aliases resolve; the others run together.
 */
export function listCruiseRuns(root) {
  const shared = { roots: [], tsConfig: undefined };
  const runs = [shared];
  for (const one of listPackages(root)) {
    const source = getSourceRoot(root, one);
    if (!source) continue;
    const tsConfig = path.posix.join(one.dir, "tsconfig.json");
    const hasPaths = fs.existsSync(path.join(root, tsConfig)) && Object.keys(readTsConfigPaths(path.join(root, tsConfig))).length > 0;
    if (hasPaths) runs.push({ roots: [source], tsConfig });
    else shared.roots.push(source);
  }
  return runs.filter((run) => run.roots.length > 0);
}

/** The violations of one run as `{ file, line, symbol }`, and the number of modules it read. */
function readCruiseRun(root, { roots, tsConfig }, config) {
  const notFollowed = `^(?!(?:${roots.map(escapeRegExp).join("|")})/)|node_modules`;
  const args = ["--config", config, "--output-type", "json", "--do-not-follow", notFollowed];
  if (tsConfig) args.push("--ts-config", path.join(root, tsConfig));
  const run = spawnSync(DEPCRUISE, [...args, ...roots], { cwd: root, encoding: "utf8", maxBuffer: 1 << 30 });
  if (run.error) throw run.error;
  let result;
  try {
    result = JSON.parse(run.stdout);
  } catch {
    throw new Error(`depcruise ${roots.join(" ")} gave no report:\n${run.stderr}`);
  }
  const findings = result.summary.violations.map((violation) => ({
    file: violation.from,
    line: 1,
    symbol: violation.cycle
      ? `${violation.rule.name}: ${[violation.from, ...violation.cycle.map((one) => one.name ?? one)].join(" -> ")}`
      : `${violation.rule.name}: ${violation.to}`,
  }));
  return { findings, cruised: result.summary.totalCruised };
}

/** Cruises every source root of `root` and returns the violations and the number of modules read. */
export function findDependencyViolations(root, config = CRUISE_CONFIG) {
  const findings = [];
  let cruised = 0;
  for (const run of listCruiseRuns(root)) {
    const one = readCruiseRun(root, run, config);
    findings.push(...one.findings);
    cruised += one.cruised;
  }
  return { findings, cruised };
}

export function verifyDependencyGraph() {
  const { findings, cruised } = findDependencyViolations(REPO_ROOT);
  requireMatches("modules cruised", cruised, 1000);
  failOnFindings("dependency-cruiser rejects an import", findings);
}

export const DEPENDENCY_VERIFIERS = [
  {
    id: "dependency-graph",
    scope: "libs",
    module: "boundary",
    name: "Import Graph (dependency-cruiser: package boundaries, segment direction, no runtime import cycle)",
    run: verifyDependencyGraph,
  },
];
