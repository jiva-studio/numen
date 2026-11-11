import fs from "node:fs";
import path from "node:path";
import { parse as parseVue } from "@vue/compiler-sfc";
import ts from "typescript";
import { isNotShipped, listAppFiles, listWindows } from "../frontend-graph/check.mjs";
import { failOnFindings, listShippedFiles, requireMatches, REPO_ROOT } from "../../../lib/graph.mjs";

/** The script blocks of a source file, each with the number of lines above it in the file. */
function readScripts(file, text) {
  if (!file.endsWith(".vue")) return [{ text, lineOffset: 0, isSetup: false }];
  const { descriptor } = parseVue(text, { filename: file });
  return [descriptor.script, descriptor.scriptSetup]
    .filter(Boolean)
    .map((block) => ({ text: block.content, lineOffset: block.loc.start.line - 1, isSetup: block === descriptor.scriptSetup }));
}

/** Visits the syntax tree of every script block of a file with a function that gives the line of a node. */
function walkScripts(file, text, visit, { setupOnly = false } = {}) {
  for (const block of readScripts(file, text)) {
    if (setupOnly && !block.isSetup) continue;
    const source = ts.createSourceFile(`${file}.ts`, block.text, ts.ScriptTarget.Latest, true, ts.ScriptKind.TS);
    const lineOf = (node) => source.getLineAndCharacterOfPosition(node.getStart(source)).line + 1 + block.lineOffset;
    visit(source, lineOf);
  }
}

const isFunctionLike = (node) =>
  ts.isFunctionDeclaration(node) || ts.isFunctionExpression(node) || ts.isArrowFunction(node) || ts.isMethodDeclaration(node);

const isStringPart = (node) => ts.isStringLiteralLike(node) || ts.isTemplateExpression(node);

/** Whether a node joins text with `+` where one side is a string literal or a template. */
const isTextConcatenation = (node) =>
  ts.isBinaryExpression(node) && node.operatorToken.kind === ts.SyntaxKind.PlusToken && (isStringPart(node.left) || isStringPart(node.right) || isTextConcatenation(node.left));

const ERROR_CLASSES = /^(?:\w*Error)$/;

/** Whether some node inside `root`, `root` included, satisfies `test`. */
function contains(root, test) {
  let found = false;
  const visit = (node) => {
    if (found) return;
    if (test(node)) {
      found = true;
      return;
    }
    ts.forEachChild(node, visit);
  };
  visit(root);
  return found;
}

const hasOptionalChain = (root) =>
  contains(root, (node) => (ts.isPropertyAccessExpression(node) || ts.isElementAccessExpression(node) || ts.isCallExpression(node)) && node.questionDotToken !== undefined);

/** Whether a node takes one way or another: `if`, `?:`, `try`, or a fallback (`??`, `||`) over an optional chain. */
function isDecision(node) {
  if (ts.isIfStatement(node) || ts.isConditionalExpression(node) || ts.isTryStatement(node)) return true;
  const isFallback =
    ts.isBinaryExpression(node) && (node.operatorToken.kind === ts.SyntaxKind.QuestionQuestionToken || node.operatorToken.kind === ts.SyntaxKind.BarBarToken);
  return isFallback && hasOptionalChain(node.left);
}

/**
 * What the `app` layer of a window must not hold as logic: a text built with a template literal or `+`, an error
 * text, and a function given as a property of an object (an adapter) whose body makes a call and decides
 * over what it got. A text belongs in a words file, a decision in the slice that owns the data.
 *
 * @param {Array<{ file: string, text: string }>} files The files under `app/`, with their text
 * @returns {{ findings: Array<{ file: string, line: number, symbol: string }>, read: number }}
 */
export function findLogicInApp(files) {
  const findings = [];
  const shipped = files.filter((one) => !isNotShipped(one.file));
  for (const { file, text } of shipped) {
    walkScripts(file, text, (source, lineOf) => {
      const visit = (node, parent) => {
        if (ts.isImportDeclaration(node) || ts.isExportDeclaration(node)) return;
        if (ts.isTemplateExpression(node)) findings.push({ file, line: lineOf(node), symbol: "text built with a template literal" });
        if (isTextConcatenation(node) && !(parent && isTextConcatenation(parent))) {
          findings.push({ file, line: lineOf(node), symbol: "text built by concatenation" });
        }
        if (ts.isNewExpression(node) && ts.isIdentifier(node.expression) && ERROR_CLASSES.test(node.expression.text) && node.arguments?.[0] && isStringPart(node.arguments[0])) {
          findings.push({ file, line: lineOf(node), symbol: `error text in ${node.expression.text}` });
        }
        const closure = ts.isPropertyAssignment(node) && (ts.isArrowFunction(node.initializer) || ts.isFunctionExpression(node.initializer)) ? node.initializer : ts.isMethodDeclaration(node) && ts.isObjectLiteralExpression(node.parent) ? node : undefined;
        if (closure && closure.body && contains(closure.body, ts.isCallExpression) && contains(closure.body, isDecision)) {
          findings.push({ file, line: lineOf(node), symbol: `adapter function ${node.name.getText(source)} makes a call and decides` });
        }
        ts.forEachChild(node, (child) => visit(child, node));
      };
      visit(source, undefined);
    });
  }
  return { findings, read: shipped.length };
}

/** The DOM calls named `create...`, which are no factories of the window. */
const DOM_CREATORS = new Set([
  "createElement",
  "createElementNS",
  "createTextNode",
  "createComment",
  "createRange",
  "createDocumentFragment",
  "createEvent",
  "createNodeIterator",
  "createTreeWalker",
  "createObjectURL",
  "createImageBitmap",
]);

const FACTORY_NAME = /^create[A-Z]/;

const hasAsyncModifier = (node) => ts.getModifiers?.(node)?.some((one) => one.kind === ts.SyntaxKind.AsyncKeyword) ?? false;

/**
 * The calls of a `create*` function and the `async` functions in the `<script setup>` of components. A component
 * draws state that a `model` segment made; making a part and waiting on a promise belong there.
 *
 * @param {Array<{ file: string, text: string }>} files The files to read; only `.vue` files are examined
 * @returns {{ findings: Array<{ file: string, line: number, symbol: string }>, read: number }}
 */
export function findFactoriesInScriptSetup(files) {
  const findings = [];
  const components = files.filter((one) => one.file.endsWith(".vue"));
  let read = 0;
  for (const { file, text } of components) {
    const { descriptor } = parseVue(text, { filename: file });
    if (!descriptor.scriptSetup) continue;
    read += 1;
    walkScripts(
      file,
      text,
      (source, lineOf) => {
        const visit = (node) => {
          if (ts.isCallExpression(node)) {
            const callee = node.expression;
            const name = ts.isIdentifier(callee) ? callee.text : ts.isPropertyAccessExpression(callee) ? callee.name.text : undefined;
            if (name && FACTORY_NAME.test(name) && !DOM_CREATORS.has(name)) {
              findings.push({ file, line: lineOf(node), symbol: `<script setup> calls ${name}` });
            }
          }
          if (isFunctionLike(node) && hasAsyncModifier(node)) {
            findings.push({ file, line: lineOf(node), symbol: "<script setup> holds an async function" });
          }
          ts.forEachChild(node, visit);
        };
        visit(source);
      },
      { setupOnly: true },
    );
  }
  return { findings, read };
}

/** The names a parameter list binds, from identifiers and destructuring patterns. */
function getBoundNames(params) {
  const names = new Set();
  const collect = (name) => {
    if (ts.isIdentifier(name)) names.add(name.text);
    else for (const element of name.elements) if (ts.isBindingElement(element)) collect(element.name);
  };
  for (const param of params) collect(param.name);
  return names;
}

/** Whether an expression is a bound name or a member chain that starts at one. */
function isParameterField(expression, names) {
  let current = expression;
  while (ts.isPropertyAccessExpression(current) || ts.isNonNullExpression(current)) current = current.expression;
  return ts.isIdentifier(current) && names.has(current.text);
}

/** The expression a function returns when that is all it does, or undefined. */
function getOnlyReturned(fn) {
  const body = fn.body;
  if (!body) return undefined;
  if (!ts.isBlock(body)) return body;
  return body.statements.length === 1 && ts.isReturnStatement(body.statements[0]) ? body.statements[0].expression : undefined;
}

const unwrap = (node) => (node && ts.isParenthesizedExpression(node) ? unwrap(node.expression) : node);

/** The `create*` functions of a source with the node that names each. */
function listFactories(source) {
  const found = [];
  const visit = (node) => {
    if (ts.isFunctionDeclaration(node) && node.name && FACTORY_NAME.test(node.name.text)) found.push({ name: node.name.text, fn: node, at: node.name });
    if (ts.isVariableDeclaration(node) && ts.isIdentifier(node.name) && FACTORY_NAME.test(node.name.text) && node.initializer && (ts.isArrowFunction(node.initializer) || ts.isFunctionExpression(node.initializer))) {
      found.push({ name: node.name.text, fn: node.initializer, at: node.name });
    }
    ts.forEachChild(node, visit);
  };
  visit(source);
  return found;
}

/**
 * The `create*` functions that only copy fields of their parameters into the object they return, one by one, and
 * do nothing else. Such a factory adds a name and a hop, and the caller could pass the parts itself.
 *
 * @param {Array<{ file: string, text: string }>} files The files to read
 * @returns {{ findings: Array<{ file: string, line: number, symbol: string }>, read: number }}
 */
export function findPassThroughFactories(files) {
  const findings = [];
  let read = 0;
  for (const { file, text } of files) {
    walkScripts(file, text, (source, lineOf) => {
      for (const { name, fn, at } of listFactories(source)) {
        read += 1;
        const returned = unwrap(getOnlyReturned(fn));
        if (!returned || !ts.isObjectLiteralExpression(returned) || returned.properties.length < 2) continue;
        const names = getBoundNames(fn.parameters);
        const copies = returned.properties.every(
          (one) => ts.isShorthandPropertyAssignment(one) ? names.has(one.name.text) : ts.isPropertyAssignment(one) && isParameterField(one.initializer, names),
        );
        if (copies) findings.push({ file, line: lineOf(at), symbol: `${name} only copies fields of its parameter` });
      }
    });
  }
  return { findings, read };
}

const SOURCE_FILE = /\.(ts|vue)$/;

/** Tracked source files of the windows, libraries and plugin frontends that ship, with their text. */
function listSourceFiles() {
  return listShippedFiles(["apps", "libs", "plugins"])
    .filter((file) => SOURCE_FILE.test(file) && !file.endsWith(".d.ts") && !file.split("/").includes("node_modules") && /(^|\/)(src|frontend)\//.test(file))
    .map((file) => ({ file, text: fs.readFileSync(path.join(REPO_ROOT, file), "utf8") }));
}

export function verifyAppHoldsNoLogic() {
  const findings = [];
  let read = 0;
  for (const root of listWindows(REPO_ROOT).filter((one) => one.startsWith("apps/desktop/"))) {
    const one = findLogicInApp(listAppFiles(root));
    findings.push(...one.findings);
    read += one.read;
  }
  requireMatches("Source files of the app layers of the desktop windows", read);
  failOnFindings("The app layer of a window builds text or holds an adapter function that calls and decides", findings);
}

export function verifyScriptSetupHoldsNoFactories() {
  const one = findFactoriesInScriptSetup(listSourceFiles());
  requireMatches("Components with a <script setup>", one.read);
  failOnFindings("A <script setup> calls a create function or holds an async function", one.findings);
}

export function verifyNoPassThroughFactory() {
  const one = findPassThroughFactories(listSourceFiles());
  requireMatches("create functions", one.read);
  failOnFindings("A create function only copies fields of its parameter", one.findings);
}

export const APP_LOGIC_VERIFIERS = [
  {
    id: "app-holds-no-logic",
    scope: "apps",
    module: "boundary",
    name: "The app Layer of a Desktop Window Builds No Text and Gives No Adapter Function That Calls and Decides",
    run: verifyAppHoldsNoLogic,
  },
  {
    id: "script-setup-holds-no-factory",
    scope: "libs",
    module: "boundary",
    name: "A <script setup> Calls No create Function and Holds No async Function",
    run: verifyScriptSetupHoldsNoFactories,
  },
  {
    id: "no-pass-through-factory",
    scope: "libs",
    module: "boundary",
    name: "A create Function Does More Than Copy Fields of Its Parameter Into the Object It Returns",
    run: verifyNoPassThroughFactory,
  },
];
