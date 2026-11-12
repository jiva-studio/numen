import fs from "node:fs";
import path from "node:path";
import { parse as parseVue } from "@vue/compiler-sfc";
import ts from "typescript";
import { ARRAY_CALLBACK_METHODS, isNotShipped, listAppFiles, listWindows } from "../frontend-graph/check.mjs";
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

/** Whether a node takes one way or another: `if`, `?:`, `try`, `&&`, `||`, `??` or an array method with a callback. */
function isDecision(node) {
  if (ts.isIfStatement(node) || ts.isConditionalExpression(node) || ts.isTryStatement(node)) return true;
  if (ts.isBinaryExpression(node)) {
    return [ts.SyntaxKind.AmpersandAmpersandToken, ts.SyntaxKind.BarBarToken, ts.SyntaxKind.QuestionQuestionToken].includes(node.operatorToken.kind);
  }
  return ts.isCallExpression(node) && ts.isPropertyAccessExpression(node.expression) && ARRAY_CALLBACK_METHODS.has(node.expression.name.text) && node.arguments.some((one) => ts.isArrowFunction(one) || ts.isFunctionExpression(one));
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

/** The most inputs a function of the app layer takes: its parameters, or the members of the one object it takes. */
export const APP_MAX_INPUTS = 8;

/** The type operators that keep the members of the type they wrap. */
const TYPE_WRAPPERS = new Set(["Pick", "Omit", "Partial", "Required", "Readonly"]);

const unwrapType = (node) => (ts.isParenthesizedTypeNode(node) || ts.isTypeOperatorNode(node) ? unwrapType(node.type) : node);

/** Whether a type node is the whole return value of a function of the app layer: `ReturnType<typeof createX>`, not a member of it. */
const isReturnTypeOfAppFunction = (node, appFunctions) =>
  ts.isTypeReferenceNode(node) &&
  ts.isIdentifier(node.typeName) &&
  node.typeName.text === "ReturnType" &&
  node.typeArguments?.length === 1 &&
  ts.isTypeQueryNode(node.typeArguments[0]) &&
  ts.isIdentifier(node.typeArguments[0].exprName) &&
  appFunctions.has(node.typeArguments[0].exprName.text);

/** The interfaces and type aliases declared in the sources, by name. */
function collectDeclaredTypes(sources) {
  const declared = new Map();
  for (const source of sources) {
    const visit = (node) => {
      if (ts.isInterfaceDeclaration(node)) declared.set(node.name.text, { members: node.members.filter(ts.isPropertySignature) });
      if (ts.isTypeAliasDeclaration(node)) declared.set(node.name.text, { alias: node.type });
      ts.forEachChild(node, visit);
    };
    visit(source);
  }
  return declared;
}

/** Builds the test for a type that narrows the result of a function of the app layer, written directly or by an alias. */
function createNarrowedResultTest(declared, appFunctions) {
  const resultNames = new Set(
    [...declared].filter(([, one]) => one.alias && isReturnTypeOfAppFunction(unwrapType(one.alias), appFunctions)).map(([name]) => name),
  );
  const isResult = (node) => isReturnTypeOfAppFunction(node, appFunctions) || (ts.isTypeReferenceNode(node) && ts.isIdentifier(node.typeName) && resultNames.has(node.typeName.text));
  return (type) => {
    const node = unwrapType(type);
    if (!ts.isTypeReferenceNode(node) || !ts.isIdentifier(node.typeName) || !TYPE_WRAPPERS.has(node.typeName.text)) return false;
    const [first] = node.typeArguments ?? [];
    return first !== undefined && isResult(unwrapType(first));
  };
}

/** The members of an object type written in place or declared in the app layer, or undefined for any other type. */
function getObjectMembers(type, declared) {
  const node = type && unwrapType(type);
  if (!node) return undefined;
  if (ts.isTypeLiteralNode(node)) return node.members.filter(ts.isPropertySignature);
  if (ts.isTypeReferenceNode(node) && ts.isIdentifier(node.typeName) && !TYPE_WRAPPERS.has(node.typeName.text)) {
    const one = declared.get(node.typeName.text);
    if (one?.members) return one.members;
    if (one?.alias) return getObjectMembers(one.alias, declared);
  }
  return undefined;
}

/** The functions of a source, declared or assigned to a name. */
function listFunctions(source) {
  const found = [];
  const visit = (node) => {
    if (ts.isFunctionDeclaration(node) && node.name) found.push({ name: node.name.text, fn: node });
    const initializer = ts.isVariableDeclaration(node) ? node.initializer : undefined;
    if (initializer && ts.isIdentifier(node.name) && (ts.isArrowFunction(initializer) || ts.isFunctionExpression(initializer))) {
      found.push({ name: node.name.text, fn: initializer });
    }
    ts.forEachChild(node, visit);
  };
  visit(source);
  return found;
}

/** The inputs of a function: the members of the object type it takes, or each of its parameters. */
function listInputs(fn, declared, source) {
  return fn.parameters.flatMap((param) => {
    const members = getObjectMembers(param.type, declared);
    if (members) return members.map((one) => ({ label: one.name.getText(source), type: one.type }));
    if (ts.isObjectBindingPattern(param.name)) return param.name.elements.map((one) => ({ label: one.name.getText(source), type: undefined }));
    return [{ label: param.name.getText(source), type: param.type }];
  });
}

/**
 * The functions of the `app` layer that take more than `maxInputs` inputs, or take an input that narrows the result
 * of another function with `Pick`, `Omit`, `Partial`, `Required` or `Readonly`. The inputs of a function are its
 * parameters, or the members of the object type it takes. A narrowed result hides in its type what the function uses of
 * it; a function that takes what it uses names each of them, and takes a result whole only where it uses most of it.
 *
 * @param {Array<{ file: string, text: string }>} files The files under `app/`, with their text
 * @param {number} maxInputs The most inputs a function may take
 * @returns {{ findings: Array<{ file: string, line: number, symbol: string }>, read: number }}
 */
export function findWideAppFunctions(files, maxInputs = APP_MAX_INPUTS) {
  const findings = [];
  const shipped = files.filter((one) => !isNotShipped(one.file));
  const parsed = shipped.flatMap(({ file, text }) =>
    readScripts(file, text).map((block) => ({ file, lineOffset: block.lineOffset, source: ts.createSourceFile(`${file}.ts`, block.text, ts.ScriptTarget.Latest, true, ts.ScriptKind.TS) })),
  );
  const declared = collectDeclaredTypes(parsed.map((one) => one.source));
  const appFunctions = new Set(parsed.flatMap((one) => listFunctions(one.source).map((fn) => fn.name)));
  const isNarrowedResult = createNarrowedResultTest(declared, appFunctions);
  for (const { file, lineOffset, source } of parsed) {
    for (const { name, fn } of listFunctions(source)) {
      const line = source.getLineAndCharacterOfPosition(fn.getStart(source)).line + 1 + lineOffset;
      const inputs = listInputs(fn, declared, source);
      if (inputs.length > maxInputs) findings.push({ file, line, symbol: `${name} takes ${inputs.length} inputs, at most ${maxInputs} are allowed` });
      for (const one of inputs.filter((input) => input.type && isNarrowedResult(input.type))) {
        findings.push({ file, line, symbol: `${name} takes ${one.label}, a narrowed result of a function` });
      }
    }
  }
  return { findings, read: shipped.length };
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

export function verifyAppFunctionsTakeConcreteInputs() {
  const findings = [];
  let read = 0;
  for (const root of listWindows(REPO_ROOT)) {
    const one = findWideAppFunctions(listAppFiles(root));
    findings.push(...one.findings);
    read += one.read;
  }
  requireMatches("Source files of the app layers", read);
  failOnFindings("A function of the app layer takes more than eight inputs, or the whole result of another function", findings);
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
    id: "app-takes-concrete-inputs",
    scope: "apps",
    module: "boundary",
    name: "A Function of the app Layer of a Window Takes at Most Eight Inputs, and None Is the Whole Result of Another Function",
    run: verifyAppFunctionsTakeConcreteInputs,
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
