import fs from "node:fs";
import { parse as parseVue } from "@vue/compiler-sfc";
import ts from "typescript";
import path from "node:path";
import { listPackages, readModuleGraph } from "../dependency-graph/check.mjs";
import { REPO_ROOT, failOnFindings, requireMatches } from "../../../lib/graph.mjs";

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
export function listWindows(root) {
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
  requireMatches("Components cruised", read);
  failOnFindings("A component lies in a lib, api or model segment", findings);
}

export function verifyCoreClientOnlyInApi() {
  const roots = listWindows(REPO_ROOT);
  requireMatches("Windows", roots.length);
  const importing = getGraph().filter((one) => (one.dependencies ?? []).some((dep) => dep.module === CORE_CLIENT || dep.module.startsWith(`${CORE_CLIENT}/`)));
  requireMatches("Files importing @numen/core-client", importing.length);
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
  requireMatches("Shared files of the windows", read);
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

/** The most decisions one function of the app layer takes: an `if`, a `?:`, a loop or a `catch`. */
export const APP_MAX_DECISIONS_PER_FUNCTION = 1;

/** The most decisions the functions of one file of the app layer take together. */
export const APP_MAX_DECISIONS_PER_FILE = 3;

/** The calls that start a watcher, a listener or a timer, which belong to the slice that owns what they watch. */
const APP_FORBIDDEN_CALLS = new Set([
  "watch",
  "watchEffect",
  "watchPostEffect",
  "watchSyncEffect",
  "addEventListener",
  "removeEventListener",
  "setTimeout",
  "setInterval",
  "clearTimeout",
  "clearInterval",
  "requestAnimationFrame",
]);

const isFunctionNode = (node) =>
  ts.isFunctionDeclaration(node) || ts.isFunctionExpression(node) || ts.isArrowFunction(node) || ts.isMethodDeclaration(node);

/** Whether a node is a point where the code takes one way or another over a value. */
function isDecision(node) {
  if (ts.isIfStatement(node) || ts.isConditionalExpression(node) || ts.isCatchClause(node)) return true;
  return ts.isForStatement(node) || ts.isForOfStatement(node) || ts.isForInStatement(node) || ts.isWhileStatement(node) || ts.isDoStatement(node);
}

/** The name a call is made by: `watch` for `watch(...)` and `globalThis.addEventListener(...)`. */
function getCalledName(call) {
  const callee = call.expression;
  if (ts.isIdentifier(callee)) return callee.text;
  if (ts.isPropertyAccessExpression(callee)) return callee.name.text;
  return undefined;
}

/**
 * What the `app` layer of a window must not hold: a watcher, a listener or a timer, which a slice
 * owns, and code that decides over values (more than one `if`, `?:`, loop or `catch` in a function,
 * or more than a few in a file) or switches over them. `app` creates, connects and orders slices; a
 * rule over data belongs to the slice that owns the data.
 *
 * @param {Array<{ file: string, text: string }>} files The files under `app/`, with their text
 * @returns {{ findings: Array<{ file: string, line: number, symbol: string }>, read: number }}
 */
export function findAppLogic(files) {
  const findings = [];
  const shipped = files.filter((one) => !isNotShipped(one.file));
  for (const { file, text } of shipped) {
    const source = ts.createSourceFile(file, readScriptText(file, text), ts.ScriptTarget.Latest, true, ts.ScriptKind.TS);
    const lineOf = (node) => source.getLineAndCharacterOfPosition(node.getStart(source)).line + 1;
    let total = 0;
    const visit = (node, owner) => {
      if (ts.isCallExpression(node)) {
        const name = getCalledName(node);
        if (name && APP_FORBIDDEN_CALLS.has(name)) findings.push({ file, line: lineOf(node), symbol: `calls ${name}` });
      }
      if (ts.isSwitchStatement(node)) findings.push({ file, line: lineOf(node), symbol: "a switch" });
      if (isDecision(node)) {
        total += 1;
        owner.decisions += 1;
        if (owner.decisions === APP_MAX_DECISIONS_PER_FUNCTION + 1) {
          findings.push({ file, line: lineOf(node), symbol: `more than ${APP_MAX_DECISIONS_PER_FUNCTION} decision in a function` });
        }
      }
      const next = isFunctionNode(node) ? { decisions: 0 } : owner;
      ts.forEachChild(node, (child) => visit(child, next));
    };
    visit(source, { decisions: 0 });
    if (total > APP_MAX_DECISIONS_PER_FILE) {
      findings.push({ file, line: 1, symbol: `${total} decisions, at most ${APP_MAX_DECISIONS_PER_FILE} are allowed in a file of app` });
    }
  }
  return { findings, read: shipped.length };
}

/** The text of the script of a source file: the file itself, or the script blocks of a component. */
function readScriptText(file, text) {
  if (!file.endsWith(".vue")) return text;
  const { descriptor } = parseVue(text, { filename: file });
  return [descriptor.script?.content, descriptor.scriptSetup?.content].filter(Boolean).join("\n");
}

/** Whether an expression is an empty value: `{}`, `[]`, `undefined`, `null`, or a function that returns one. */
function isEmptyValue(node) {
  if (ts.isParenthesizedExpression(node)) return isEmptyValue(node.expression);
  if (ts.isObjectLiteralExpression(node)) return node.properties.length === 0;
  if (ts.isArrayLiteralExpression(node)) return node.elements.length === 0;
  if (node.kind === ts.SyntaxKind.NullKeyword) return true;
  if (ts.isIdentifier(node)) return node.text === "undefined";
  if (ts.isArrowFunction(node) || ts.isFunctionExpression(node) || ts.isMethodDeclaration(node)) {
    if (node.body && ts.isBlock(node.body)) return node.body.statements.length === 0 || (node.body.statements.length === 1 && ts.isReturnStatement(node.body.statements[0]) && (node.body.statements[0].expression === undefined || isEmptyValue(node.body.statements[0].expression)));
    return node.body === undefined || isEmptyValue(node.body);
  }
  return false;
}

/** Whether an object literal is empty or holds only empty values: a stub for a part made later. */
function isEmptyStub(node) {
  if (!ts.isObjectLiteralExpression(node)) return false;
  return node.properties.every((one) => (ts.isPropertyAssignment(one) && isEmptyValue(one.initializer)) || (ts.isMethodDeclaration(one) && isEmptyValue(one)));
}

const STATE_FACTORIES = new Set(["ref", "shallowRef", "reactive", "shallowReactive"]);

/** Whether a node names `current` as a property. */
const namesCurrent = (name) => name !== undefined && (ts.isIdentifier(name) || ts.isStringLiteral(name)) && name.text === "current";

/** The forms of a part that is made empty and filled once the parts it needs exist, found in one source text. */
function findLateBindingsIn(file, text) {
  const found = [];
  const source = ts.createSourceFile(file, readScriptText(file, text), ts.ScriptTarget.Latest, true, ts.ScriptKind.TS);
  const add = (node, symbol) => found.push({ file, line: source.getLineAndCharacterOfPosition(node.getStart(source)).line + 1, symbol });
  const visit = (node) => {
    if (ts.isPropertySignature(node) && node.questionToken && namesCurrent(node.name)) add(node, "a holder with an optional current");
    if (ts.isBinaryExpression(node) && node.operatorToken.kind === ts.SyntaxKind.EqualsToken && ts.isPropertyAccessExpression(node.left) && node.left.name.text === "current") {
      add(node, "a holder assigned after it was made");
    }
    if (ts.isCallExpression(node)) {
      const callee = node.expression;
      if (ts.isPropertyAccessExpression(callee) && ts.isIdentifier(callee.expression) && callee.expression.text === "Object" && callee.name.text === "assign" && node.arguments[0] && !ts.isObjectLiteralExpression(node.arguments[0])) {
        add(node, "Object.assign fills an object that was made before");
      }
      if (ts.isPropertyAccessExpression(callee) && callee.name.text === "provide") add(node, "provide fills a state that was made before");
      if (ts.isIdentifier(callee) && STATE_FACTORIES.has(callee.text) && node.arguments[0] && isEmptyStub(node.arguments[0])) {
        add(node, `${callee.text} starts as an empty stub that is filled later`);
      }
    }
    if (ts.isInterfaceDeclaration(node) && !node.heritageClauses) {
      const fields = node.members.filter((one) => ts.isPropertySignature(one));
      if (fields.length >= 2 && fields.every((one) => one.questionToken)) add(node, `interface ${node.name.text} has only optional fields`);
    }
    ts.forEachChild(node, visit);
  };
  visit(source);
  return found;
}

/**
 * The places in a window's `app` layer that bind a part late: a `{ current?: ... }` holder, a `.current = ...`
 * assignment, an `Object.assign` onto an object made before, a `.provide(...)` call on a state, a `ref`, `shallowRef`
 * or `reactive` that starts as `{}` or as a stub of empty functions, and an interface whose fields are all optional.
 * `app` creates each part after the parts it needs; a part two others need is made first and handed to both. Read
 * from the syntax tree, so a form is found wherever it is written.
 *
 * @param {Array<{ file: string, text: string }>} files The files under `app/`, with their text
 * @returns {{ findings: Array<{ file: string, line: number, symbol: string }>, read: number }}
 */
export function findLateBindings(files) {
  const shipped = files.filter((one) => !isNotShipped(one.file));
  return { findings: shipped.flatMap(({ file, text }) => findLateBindingsIn(file, text)), read: shipped.length };
}

/** The names a window's `app` layer may take from a page: a page that is made (`create<Name>Page`). A component may also be drawn by a component of `app`. */
const PAGE_NAME = /^create[A-Z]\w*Page$/;
const COMPONENT_NAME = /^[A-Z]\w*$/;

/** The page specifiers: `@/pages/...` as the windows write it. */
const isPageSpecifier = (specifier) => /^@\/pages(?:\/|$)/.test(specifier);

/**
 * The imports of `pages` that a layer below it or `app` must not make. `entities` and `features` import no page. `app`
 * composes pages, so it takes a page's `create<Name>Page`, and a component of `app` draws a page's component, never a state a page owns
 * for the window (a `use<Name>` composable, a store, a list).
 *
 * @param {Array<{ file: string, text: string }>} files Shipped files under `src/`, with their text
 * @returns {{ findings: Array<{ file: string, line: number, symbol: string }>, read: number }}
 */
export function findPageImports(files) {
  const findings = [];
  const shipped = files.filter((one) => !isNotShipped(one.file));
  for (const { file, text } of shipped) {
    const layer = /\/src\/(\w+)\//.exec(`/${file}`)?.[1];
    if (!layer || !["entities", "features", "app", "shared"].includes(layer)) continue;
    const source = ts.createSourceFile(file, readScriptText(file, text), ts.ScriptTarget.Latest, true, ts.ScriptKind.TS);
    for (const statement of source.statements) {
      const isImport = ts.isImportDeclaration(statement);
      const isReExport = ts.isExportDeclaration(statement) && statement.moduleSpecifier;
      if (!isImport && !isReExport) continue;
      const specifier = statement.moduleSpecifier.text;
      if (!isPageSpecifier(specifier)) continue;
      const line = source.getLineAndCharacterOfPosition(statement.getStart(source)).line + 1;
      if (layer !== "app") {
        findings.push({ file, line, symbol: `${layer} imports ${specifier}` });
        continue;
      }
      const clause = isImport ? statement.importClause : statement.exportClause;
      const named = clause?.namedBindings ?? clause;
      const names = named && ts.isNamedImports(named) ? named.elements.map((one) => (one.propertyName ?? one.name).text) : named && ts.isNamedExports(named) ? named.elements.map((one) => (one.propertyName ?? one.name).text) : [];
      if (clause === undefined || (isImport && clause.name === undefined && names.length === 0)) {
        findings.push({ file, line, symbol: `app imports ${specifier} as a whole` });
        continue;
      }
      const isComponent = file.endsWith(".vue");
      for (const name of names.filter((one) => !PAGE_NAME.test(one) && !(isComponent && COMPONENT_NAME.test(one)))) findings.push({ file, line, symbol: `app takes ${name} from ${specifier}` });
    }
  }
  return { findings, read: shipped.length };
}

/**
 * The deep slice imports of the test support (`src/testing`): `@/<layer>/<slice>/<more>`. Test support reaches a slice
 * through its index, as the code it supports does; a re-export of a deep path from `testing` would launder it.
 *
 * @param {Array<{ file: string, text: string }>} files The files under `src/testing/`, with their text
 * @returns {{ findings: Array<{ file: string, line: number, symbol: string }>, read: number }}
 */
export function findDeepSliceImports(files) {
  const findings = [];
  for (const { file, text } of files) {
    const source = ts.createSourceFile(file, text, ts.ScriptTarget.Latest, true, ts.ScriptKind.TS);
    for (const statement of source.statements) {
      if (!(ts.isImportDeclaration(statement) || ts.isExportDeclaration(statement)) || !statement.moduleSpecifier) continue;
      const specifier = statement.moduleSpecifier.text;
      if (/^@\/(?:entities|features|pages)\/[^/]+\/.+/.test(specifier)) {
        findings.push({ file, line: source.getLineAndCharacterOfPosition(statement.getStart(source)).line + 1, symbol: `${specifier} is not the index of a slice` });
      }
    }
  }
  return { findings, read: files.length };
}

function listLayerFiles(root, layer) {
  const out = [];
  const walk = (dir) => {
    if (!fs.existsSync(path.join(REPO_ROOT, dir))) return;
    for (const entry of fs.readdirSync(path.join(REPO_ROOT, dir), { withFileTypes: true })) {
      const next = path.posix.join(dir, entry.name);
      if (entry.isDirectory()) walk(next);
      else if (/\.(ts|vue)$/.test(entry.name)) out.push({ file: next, text: fs.readFileSync(path.join(REPO_ROOT, next), "utf8") });
    }
  };
  walk(path.posix.join(root, layer));
  return out;
}

/** The calls a component in a window's `ui` segment must not make: a window listener or a request guard belongs in `model`. */
const UI_COMPONENT_FORBIDDEN_CALLS = new Set([
  "addEventListener",
  "removeEventListener",
  "createRequestGuard",
  "setTimeout",
  "setInterval",
  "clearTimeout",
  "clearInterval",
  "requestAnimationFrame",
]);

/** The hooks and watchers inside which a component in `ui` must not call a service on its `props`. */
const UI_LIFECYCLE_OR_WATCH_CALLS = new Set([
  "onMounted",
  "onBeforeMount",
  "onUnmounted",
  "onBeforeUnmount",
  "watch",
  "watchEffect",
  "watchPostEffect",
  "watchSyncEffect",
]);

/** Whether an expression is rooted at `props` (such as `props.secrets.loadStates` or `props.manager.files.getReadme`). */
function isPropsChain(node) {
  let cur = node;
  while (ts.isPropertyAccessExpression(cur)) cur = cur.expression;
  return ts.isIdentifier(cur) && cur.text === "props";
}

/**
 * The places in a window's `ui` segment where a component (`.vue`) holds behavior or service calls that belong in
 * `model`: registering a listener or timer, guarding in-flight requests with `createRequestGuard`, or calling a method
 * on `props` inside a lifecycle hook or watcher (other than `props.mount(...)` to mount an external view into a DOM
 * element ref).
 *
 * @param {Array<{ file: string, text: string }>} files The `.vue` files in `ui` segments of windows, with their text
 * @returns {{ findings: Array<{ file: string, line: number, symbol: string }>, read: number }}
 */
export function findUiComponentSideEffects(files) {
  const findings = [];
  const shipped = files.filter((one) => one.file.endsWith(".vue") && !isNotShipped(one.file) && getSegment(one.file) === "ui");
  for (const { file, text } of shipped) {
    const source = ts.createSourceFile(file, readScriptText(file, text), ts.ScriptTarget.Latest, true, ts.ScriptKind.TS);
    const lineOf = (node) => source.getLineAndCharacterOfPosition(node.getStart(source)).line + 1;
    const visit = (node, inHook) => {
      if (ts.isCallExpression(node)) {
        const name = getCalledName(node);
        if (name && UI_COMPONENT_FORBIDDEN_CALLS.has(name)) {
          findings.push({ file, line: lineOf(node), symbol: `calls ${name} in a ui component` });
        }
        if (inHook && ts.isPropertyAccessExpression(node.expression) && isPropsChain(node.expression) && node.expression.name.text !== "mount") {
          findings.push({ file, line: lineOf(node), symbol: `calls props.${node.expression.name.text} inside ${inHook}` });
        }
        const nextHook = name && UI_LIFECYCLE_OR_WATCH_CALLS.has(name) ? name : inHook;
        ts.forEachChild(node, (child) => visit(child, nextHook));
        return;
      }
      ts.forEachChild(node, (child) => visit(child, inHook));
    };
    visit(source, undefined);
  }
  return { findings, read: shipped.length };
}

export function verifyUiComponentsAreHumble() {
  const findings = [];
  let read = 0;
  for (const root of listWindows(REPO_ROOT)) {
    const files = ["pages", "features", "entities", "shared"].flatMap((layer) => listLayerFiles(root, layer));
    const one = findUiComponentSideEffects(files);
    findings.push(...one.findings);
    read += one.read;
  }
  requireMatches("Components in ui segments of the windows", read);
  failOnFindings("A component in a ui segment of a window registers a listener, guards requests or calls a service on props in a hook or watcher", findings);
}

export function verifyPagesAreComposedNotConsumed() {
  const findings = [];
  let read = 0;
  for (const root of listWindows(REPO_ROOT)) {
    const files = ["app", "features", "entities", "shared"].flatMap((layer) => listLayerFiles(root, layer));
    const one = findPageImports(files);
    findings.push(...one.findings);
    read += one.read;
  }
  requireMatches("Source files below and in the app layers", read);
  failOnFindings("A layer below pages imports a page, or app takes more than a page it composes", findings);
}

export function verifyTestingReachesSlicesByIndex() {
  const findings = [];
  let read = 0;
  for (const root of listWindows(REPO_ROOT)) {
    const one = findDeepSliceImports(listLayerFiles(root, "testing"));
    findings.push(...one.findings);
    read += one.read;
  }
  requireMatches("Files of the test support", read);
  failOnFindings("The test support imports a slice by a deep path", findings);
}

export function listAppFiles(root) {
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
    const files = listAppFiles(root);
    const one = findAppOverreach(files);
    findings.push(...one.findings, ...findAppLogic(files).findings);
    read += one.read;
  }
  requireMatches("Source files of the app layers", read);
  failOnFindings("The app layer of a window holds more than composition: a long file, a composable, a watcher, a listener, a timer or code that decides over values", findings);
}

export function verifyAppBindsInOrder() {
  const findings = [];
  let read = 0;
  for (const root of listWindows(REPO_ROOT)) {
    const one = findLateBindings(listAppFiles(root));
    findings.push(...one.findings);
    read += one.read;
  }
  requireMatches("Source files of the app layers", read);
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
    name: "The app Layer of a Window Holds Short Files, No Composable, No Watcher, Listener or Timer and Little Branching",
    run: verifyAppIsComposition,
  },
  {
    id: "app-binds-in-order",
    scope: "apps",
    module: "boundary",
    name: "The app Layer of a Window Makes Each Part After the Parts It Needs, With No Holder Filled Later",
    run: verifyAppBindsInOrder,
  },
  {
    id: "pages-are-composed-not-consumed",
    scope: "apps",
    module: "boundary",
    name: "No Entity or Feature Imports a Page, and app Takes From a Page Only the Page It Composes",
    run: verifyPagesAreComposedNotConsumed,
  },
  {
    id: "testing-reaches-slices-by-index",
    scope: "apps",
    module: "boundary",
    name: "The Test Support of a Window Imports a Slice Only Through Its Index",
    run: verifyTestingReachesSlicesByIndex,
  },
  {
    id: "ui-components-are-humble",
    scope: "apps",
    module: "boundary",
    name: "A Component in a ui Segment of a Window Holds No Listener, Request Guard or Service Call in a Hook or Watcher",
    run: verifyUiComponentsAreHumble,
  },
];
