import fs from "node:fs";
import path from "node:path";
import { listPackages, readModuleGraph } from "./dependencies.mjs";
import { REPO_ROOT, failOnFindings, requireMatches } from "./helpers.mjs";

/** The segments of a slice from the lowest. */
const SEGMENTS = ["lib", "api", "model", "ui"];

/** The layers of a window that are cut into slices; `app` and the entry are one unit each. */
const SLICED_LAYERS = ["pages", "features", "entities"];

/** The package every window reaches the core through. */
const CORE_CLIENT = "@numen/core-client";

/** Whether a file is a test or a story, which exercises the code from outside and is not shipped. */
export function isNotShipped(file) {
  return /\.(test|spec|stories)\.ts$/.test(file) || file.split("/").includes("testing") || file.split("/").includes("testdata");
}

/** The segment a file belongs to: the nearest folder above it named after a segment, or undefined. */
export function getSegment(file) {
  return path.posix.dirname(file).split("/").filter((name) => SEGMENTS.includes(name)).at(-1);
}

/** The local files a module's imports resolve to. */
function listLocalTargets(one) {
  return (one.dependencies ?? []).filter((dep) => !dep.couldNotResolve && dep.resolved && !dep.resolved.split("/").includes("node_modules"));
}

/**
 * The components (`.vue` files) of the cruised source roots whose segment is `lib`, `api` or `model`: a component
 * belongs in a `ui` segment.
 *
 * @param {Array<{ source: string }>} modules The resolved import graph
 * @returns {{ findings: Array<{ file: string, line: number, symbol: string }>, read: number }}
 */
export function findComponentsBelowUi(modules) {
  const components = [...new Set(modules.map((one) => one.source))].filter((file) => file.endsWith(".vue") && !file.split("/").includes("node_modules"));
  const findings = components
    .filter((file) => ["lib", "api", "model"].includes(getSegment(file)))
    .map((file) => ({ file, line: 1, symbol: `a component in a ${getSegment(file)} segment` }));
  return { findings, read: components.length };
}

/**
 * The shipped files under a window's `src` that import `@numen/core-client` and are neither in an `api` segment nor
 * the entry `src/main.ts`.
 */
export function findCoreClientOutsideApi(modules, sourceRoots) {
  const findings = [];
  for (const one of modules) {
    const root = sourceRoots.find((dir) => one.source.startsWith(`${dir}/`));
    if (!root || isNotShipped(one.source) || one.source === `${root}/main.ts` || getSegment(one.source) === "api") continue;
    for (const dep of one.dependencies ?? []) {
      if (dep.module === CORE_CLIENT || dep.module.startsWith(`${CORE_CLIENT}/`)) {
        findings.push({ file: one.source, line: 1, symbol: `imports ${dep.module} from the ${one.source.slice(root.length + 1).split("/")[0]} layer` });
      }
    }
  }
  return findings;
}

/** The slice a file of a window belongs to: `layer/slice` on a sliced layer, the layer otherwise. */
export function getSliceOf(file, sourceRoot) {
  const [layer, slice] = file.slice(sourceRoot.length + 1).split("/");
  return SLICED_LAYERS.includes(layer) && slice ? `${layer}/${slice}` : layer;
}

/**
 * The shipped files of a window's `shared` layer that serve one slice only. A shared file's consumers are the
 * slices of the files outside `shared` that import it, and the consumers of the shared files that import it. The `app`
 * layer composes the slices and is not a consumer that makes a file shared: a file used by one slice and `app` serves
 * that slice.
 *
 * @returns {{ findings: Array<{ file: string, line: number, symbol: string }>, read: number }}
 */
export function findSharedServingOneSlice(modules, sourceRoot) {
  const shared = `${sourceRoot}/shared/`;
  const importers = new Map();
  for (const one of modules) {
    if (!one.source.startsWith(`${sourceRoot}/`) || isNotShipped(one.source)) continue;
    for (const dep of listLocalTargets(one)) {
      if (!dep.resolved.startsWith(shared)) continue;
      if (!importers.has(dep.resolved)) importers.set(dep.resolved, new Set());
      importers.get(dep.resolved).add(one.source);
    }
  }
  const consumers = new Map();
  const getConsumers = (file, visiting) => {
    if (consumers.has(file)) return consumers.get(file);
    const found = new Set();
    for (const importer of importers.get(file) ?? []) {
      if (!importer.startsWith(shared)) found.add(getSliceOf(importer, sourceRoot));
      else if (!visiting.has(importer)) for (const one of getConsumers(importer, new Set([...visiting, importer]))) found.add(one);
    }
    if (visiting.size === 1) consumers.set(file, found);
    return found;
  };
  const findings = [];
  const files = modules.map((one) => one.source).filter((file) => file.startsWith(shared) && !isNotShipped(file));
  for (const file of files) {
    const consumers = getConsumers(file, new Set([file]));
    const slices = [...consumers].filter((one) => one !== "app");
    if (slices.length === 1) findings.push({ file, line: 1, symbol: `serves only ${slices[0]}; it belongs in that slice` });
    else if (slices.length === 0 && consumers.size === 1) findings.push({ file, line: 1, symbol: "serves only app; it belongs in that layer" });
  }
  return { findings, read: files.length };
}

/**
 * The source roots of the windows: the applications that declare `@numen/core-client` and are cut into layers, with an
 * entry `src/main.ts` and an `app` layer.
 */
function listWindows(root) {
  return listPackages(root)
    .filter((one) => one.kind === "application")
    .filter((one) => {
      const manifest = JSON.parse(fs.readFileSync(path.join(root, one.dir, "package.json"), "utf8"));
      return Object.keys({ ...manifest.dependencies, ...manifest.devDependencies }).includes(CORE_CLIENT);
    })
    .map((one) => path.posix.join(one.dir, "src"))
    .filter((dir) => fs.existsSync(path.join(root, dir, "main.ts")) && fs.existsSync(path.join(root, dir, "app")));
}

let graph;
function getGraph() {
  graph ??= readModuleGraph(REPO_ROOT);
  return graph;
}

export function verifyNoComponentBelowUi() {
  const { findings, read } = findComponentsBelowUi(getGraph());
  // 249 on the tree when the floor was set; the floor is that count less 5%.
  requireMatches("Components cruised", read, 236);
  failOnFindings("A component lies in a lib, api or model segment", findings);
}

export function verifyCoreClientOnlyInApi() {
  const roots = listWindows(REPO_ROOT);
  // 3 on the tree when the floor was set; the floor is that count less 5%.
  requireMatches("Windows", roots.length, 2);
  const importing = getGraph().filter((one) => (one.dependencies ?? []).some((dep) => dep.module === CORE_CLIENT || dep.module.startsWith(`${CORE_CLIENT}/`)));
  // 123 on the tree when the floor was set; the floor is that count less 5%.
  requireMatches("Files importing @numen/core-client", importing.length, 116);
  failOnFindings("A window imports @numen/core-client outside an api segment and its entry", findCoreClientOutsideApi(getGraph(), roots));
}

export function verifySharedServesSeveralSlices() {
  const findings = [];
  let read = 0;
  for (const root of listWindows(REPO_ROOT).filter((dir) => fs.existsSync(path.join(REPO_ROOT, dir, "shared")))) {
    const one = findSharedServingOneSlice(getGraph(), root);
    findings.push(...one.findings);
    read += one.read;
  }
  // 15 on the tree when the floor was set; the floor is that count less 5%.
  requireMatches("Shared files of the windows", read, 14);
  failOnFindings("A shared file serves one slice", findings);
}

/** The longest file the app layer of a window holds; a longer one holds more than composition. */
export const APP_MAX_LINES = 300;

/**
 * The files of a window's `app` layer that hold more than composition: a source file over `maxLines`, a file named
 * as a composable (`use<Name>.ts`) and a composable exported from any other file. State and behavior belong to a
 * slice; `app` creates, connects and orders slices.
 *
 * @param {Array<{ file: string, text: string }>} files The files under `app/`, with their text
 * @param {number} maxLines The longest file allowed
 * @returns {{ findings: Array<{ file: string, line: number, symbol: string }>, read: number }}
 */
export function findAppOverreach(files, maxLines = APP_MAX_LINES) {
  const findings = [];
  const shipped = files.filter((one) => !isNotShipped(one.file));
  for (const { file, text } of shipped) {
    const count = text.split("\n").length - (text.endsWith("\n") ? 1 : 0);
    if (count > maxLines) findings.push({ file, line: maxLines + 1, symbol: `${count} lines, at most ${maxLines} are allowed in app` });
    if (/^use[A-Z]/.test(path.posix.basename(file))) findings.push({ file, line: 1, symbol: "a composable file in app" });
    text.split("\n").forEach((line, at) => {
      const exported = /^export (?:async )?(?:function|const) (use[A-Z]\w*)/.exec(line);
      if (exported) findings.push({ file, line: at + 1, symbol: `composable ${exported[1]} is exported from app` });
    });
  }
  return { findings, read: shipped.length };
}

/** The forms of a part that is made empty and filled once the parts it needs exist. */
const LATE_BINDINGS = [
  { pattern: /\bcurrent\?\s*:/, symbol: "a holder with an optional current" },
  { pattern: /\.current\s*=(?!=)/, symbol: "a holder assigned after it was made" },
  { pattern: /\bObject\.assign\(\s*(?!\{)/, symbol: "Object.assign fills an object that was made before" },
];

/**
 * The lines of a window's `app` layer that bind a part late: a `{ current?: ... }` holder, a `.current = ...`
 * assignment and an `Object.assign` onto an object made before. `app` creates each part after the parts it needs;
 * a part two others need is made first and handed to both, as a state with its own methods.
 *
 * @param {Array<{ file: string, text: string }>} files The files under `app/`, with their text
 * @returns {{ findings: Array<{ file: string, line: number, symbol: string }>, read: number }}
 */
export function findLateBindings(files) {
  const findings = [];
  const shipped = files.filter((one) => !isNotShipped(one.file));
  for (const { file, text } of shipped) {
    text.split("\n").forEach((line, at) => {
      if (/^\s*(\/\/|\*|\/\*)/.test(line)) return;
      for (const { pattern, symbol } of LATE_BINDINGS) {
        if (pattern.test(line)) findings.push({ file, line: at + 1, symbol });
      }
    });
  }
  return { findings, read: shipped.length };
}

function listAppFiles(root) {
  const out = [];
  const walk = (dir) => {
    for (const entry of fs.readdirSync(path.join(REPO_ROOT, dir), { withFileTypes: true })) {
      const next = path.posix.join(dir, entry.name);
      if (entry.isDirectory()) walk(next);
      else if (/\.(ts|vue)$/.test(entry.name)) out.push({ file: next, text: fs.readFileSync(path.join(REPO_ROOT, next), "utf8") });
    }
  };
  walk(path.posix.join(root, "app"));
  return out;
}

export function verifyAppIsComposition() {
  const findings = [];
  let read = 0;
  for (const root of listWindows(REPO_ROOT)) {
    const one = findAppOverreach(listAppFiles(root));
    findings.push(...one.findings);
    read += one.read;
  }
  // 44 on the tree when the floor was set; the floor is that count less 5%.
  requireMatches("Source files of the app layers", read, 41);
  failOnFindings("The app layer of a window holds more than composition", findings);
}

export function verifyAppBindsInOrder() {
  const findings = [];
  let read = 0;
  for (const root of listWindows(REPO_ROOT)) {
    const one = findLateBindings(listAppFiles(root));
    findings.push(...one.findings);
    read += one.read;
  }
  requireMatches("Source files of the app layers", read, 10);
  failOnFindings("The app layer of a window binds a part late", findings);
}

export const FRONTEND_GRAPH_VERIFIERS = [
  {
    id: "no-component-below-ui",
    scope: "libs",
    module: "boundary",
    name: "No Component (.vue) Lies in a lib, api or model Segment of a Library, Window or Plugin Frontend",
    run: verifyNoComponentBelowUi,
  },
  {
    id: "core-client-only-in-api",
    scope: "apps",
    module: "boundary",
    name: "A Window Imports @numen/core-client Only From an api Segment and Its Entry",
    run: verifyCoreClientOnlyInApi,
  },
  {
    id: "shared-serves-several-slices",
    scope: "apps",
    module: "boundary",
    name: "A Shared File of a Window Is Imported From More Than One Slice",
    run: verifySharedServesSeveralSlices,
  },
  {
    id: "app-is-composition",
    scope: "apps",
    module: "boundary",
    name: "The app Layer of a Window Holds Short Files and No Composable",
    run: verifyAppIsComposition,
  },
  {
    id: "app-binds-in-order",
    scope: "apps",
    module: "boundary",
    name: "The app Layer of a Window Makes Each Part After the Parts It Needs, With No Holder Filled Later",
    run: verifyAppBindsInOrder,
  },
];
