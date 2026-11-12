import fs from "node:fs";
import path from "node:path";
import { parse as parseVue } from "@vue/compiler-sfc";
import ts from "typescript";
import { GO_MODULE_ROOT, REPO_ROOT, failOnFindings, listTrackedFiles, readGo, requireMatches } from "../../../lib/graph.mjs";

/** The folders of the contracts between plugins, in the Go SDK and in the TypeScript SDK. */
export const CONTRACT_DIRS = { go: "libs/plugin/sdk-go/contracts", ts: "libs/plugin/sdk-ts/src/contracts" };

/** The Go import path and the TypeScript package entry of the contracts. */
const GO_CONTRACTS = GO_MODULE_ROOT + CONTRACT_DIRS.go;
const TS_CONTRACTS = "@numen/plugin-sdk/contracts";

const SCRIPT_FILE = /\.(ts|mts|cts|js|mjs|cjs|vue)$/;

const isWithin = (value, root) => value === root || value.startsWith(root + "/");

/** The module specifiers a script names, with their lines: imports, exports, `import()` and `import("x").T`. */
function readScriptSpecifiers(file, text, lineOffset = 0) {
  const source = ts.createSourceFile(file, text, ts.ScriptTarget.Latest, true);
  const found = [];
  const add = (literal) => {
    if (literal && ts.isStringLiteralLike(literal)) found.push({ line: source.getLineAndCharacterOfPosition(literal.getStart(source)).line + 1 + lineOffset, value: literal.text });
  };
  const visit = (node) => {
    if (ts.isImportDeclaration(node) || ts.isExportDeclaration(node)) add(node.moduleSpecifier);
    else if (ts.isCallExpression(node) && node.expression.kind === ts.SyntaxKind.ImportKeyword) add(node.arguments[0]);
    else if (ts.isImportTypeNode(node) && ts.isLiteralTypeNode(node.argument)) add(node.argument.literal);
    ts.forEachChild(node, visit);
  };
  visit(source);
  return found;
}

/** The module specifiers of a Go, TypeScript, JavaScript or Vue file, with their lines. */
function readSpecifiers(file, text) {
  if (file.endsWith(".go")) return readGo(text).imports;
  if (file.endsWith(".vue")) {
    const { descriptor } = parseVue(text, { filename: file });
    return [descriptor.script, descriptor.scriptSetup].filter(Boolean).flatMap((block) => readScriptSpecifiers(file.replace(/\.vue$/, ".ts"), block.content, block.loc.start.line - 1));
  }
  return readScriptSpecifiers(file, text);
}

/** True when the specifier of an import in `file` names a contracts package. */
function isContractImport(file, value) {
  if (file.endsWith(".go")) return isWithin(value, GO_CONTRACTS);
  if (isWithin(value, TS_CONTRACTS)) return true;
  if (!value.startsWith(".")) return false;
  return isWithin(path.posix.normalize(path.posix.join(path.posix.dirname(file), value)), CONTRACT_DIRS.ts);
}

/**
 * Lists every import of a contracts package by a file under `libs/` outside the contracts folders:
 * the kernel, the protocol, the core client, the plugin loader, the rest of the SDKs and every other
 * library. Only plugins and applications import the contracts. `readText` reads a file by its path
 * from the root. Returns the findings and the number of files read.
 */
export function findContractImports(files, readText) {
  const findings = [];
  let read = 0;
  for (const file of files) {
    if (!isWithin(file, "libs") || !(file.endsWith(".go") || SCRIPT_FILE.test(file))) continue;
    if (isWithin(file, CONTRACT_DIRS.go) || isWithin(file, CONTRACT_DIRS.ts)) continue;
    read++;
    for (const { line, value } of readSpecifiers(file, readText(file))) {
      if (isContractImport(file, value)) findings.push({ file, line, symbol: `imports ${value}; only plugins and applications import the contracts` });
    }
  }
  return { findings, read };
}

/** No library imports the contracts of the plugins; the kernel and the shared SDK stay without them. */
export function verifyContractsImportedOnlyByPluginsAndApps() {
  const { findings, read } = findContractImports(listTrackedFiles(), (file) => fs.readFileSync(path.join(REPO_ROOT, file), "utf8"));
  requireMatches("source files of the libraries", read);
  failOnFindings("A library imports the contracts of the plugins", findings);
}

export const CONTRACTS_VERIFIERS = [
  {
    id: "contracts-imported-by-plugins-and-apps",
    scope: "libs",
    module: "contracts",
    name: "Contracts Imported By Plugins And Apps (no file under libs/ outside the contracts folders of the SDKs imports libs/plugin/sdk-go/contracts or libs/plugin/sdk-ts/src/contracts)",
    run: verifyContractsImportedOnlyByPluginsAndApps,
  },
];
