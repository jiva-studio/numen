#!/usr/bin/env node
/**
 * The TypeScript half of the check `go/single-consumer`: the exports of the TypeScript SDK, and every
 * use of them and of the protocol services in the TypeScript and Vue sources of apps, libs and plugins,
 * read with the type checker of each package. Prints JSON: `items`, `uses` as [target, file, user], and
 * the number of packages and files read.
 *
 *   node uses.mjs <root> <protocol dir> <sdk dir> <library dir>...
 *
 * A target is `rpc:<Service>.<Method>` for a method of a service of the protocol, or
 * `ts:<file>|<Name>` (`<Name>.<member>` for a member) for a declaration of a library. The user is the
 * declaration of the file that holds the use, in the same form; the check decides which files count.
 */
import fs from "node:fs";
import path from "node:path";
import process from "node:process";
import { parse as parseVue } from "@vue/compiler-sfc";
import ts from "typescript";

const [rootArg, protocolDir, sdkDir, ...libraryDirs] = process.argv.slice(2);
const ROOT = fs.realpathSync(rootArg);
const SKIPPED_DIRS = new Set(["node_modules", "dist", "build", ".git"]);
const VIRTUAL_SUFFIX = ".vue.ts";

const toRelative = (file) => path.relative(ROOT, file).split(path.sep).join("/");
const isWithin = (rel, dir) => rel === dir || rel.startsWith(dir + "/");

/** The real path of a file of a program; a Vue script read as `<file>.vue.ts` is its `.vue` file. */
function getRealFile(fileName) {
  const name = fileName.endsWith(VIRTUAL_SUFFIX) ? fileName.slice(0, -3) : fileName;
  try {
    return fs.realpathSync(name);
  } catch {
    return path.resolve(name);
  }
}

/** The directories under apps, libs and plugins that hold a tsconfig.json and a package.json. */
function listPackageDirs() {
  const found = [];
  const walk = (dir) => {
    const names = fs.readdirSync(dir, { withFileTypes: true });
    if (names.some((one) => one.name === "tsconfig.json") && names.some((one) => one.name === "package.json")) found.push(dir);
    for (const one of names) {
      if (one.isDirectory() && !SKIPPED_DIRS.has(one.name) && !one.name.startsWith(".")) walk(path.join(dir, one.name));
    }
  };
  for (const top of ["apps", "libs", "plugins"]) {
    if (fs.existsSync(path.join(ROOT, top))) walk(path.join(ROOT, top));
  }
  return found.sort();
}

function listFiles(dir, pattern) {
  const found = [];
  const walk = (current) => {
    for (const one of fs.readdirSync(current, { withFileTypes: true })) {
      if (one.isDirectory() && !SKIPPED_DIRS.has(one.name) && !one.name.startsWith(".")) walk(path.join(current, one.name));
      else if (one.isFile() && pattern.test(one.name)) found.push(path.join(current, one.name));
    }
  };
  walk(dir);
  return found;
}

function readConfig(dir) {
  const configFile = path.join(dir, "tsconfig.json");
  const read = ts.readConfigFile(configFile, ts.sys.readFile);
  if (read.error) throw new Error(`${configFile}: ${ts.flattenDiagnosticMessageText(read.error.messageText, "\n")}`);
  const parsed = ts.parseJsonConfigFileContent(read.config, ts.sys, dir);
  const options = { ...parsed.options, noEmit: true, composite: false, incremental: false, tsBuildInfoFile: undefined };
  return { options, fileNames: parsed.fileNames };
}

/** The script blocks of a Vue file as one TypeScript text. */
function readVueScript(file) {
  const { descriptor } = parseVue(fs.readFileSync(file, "utf8"), { filename: file });
  return [descriptor.script, descriptor.scriptSetup].filter(Boolean).map((block) => block.content).join("\n");
}

/** A program over the package in `dir`: its tsconfig files, the configuration scripts at its root and its Vue scripts. */
function createPackageProgram(dir) {
  const { options, fileNames } = readConfig(dir);
  const virtual = new Map();
  for (const file of listFiles(dir, /\.vue$/)) virtual.set(file + ".ts", readVueScript(file));
  const configs = fs.readdirSync(dir).filter((name) => /\.config\.(ts|mts)$/.test(name)).map((name) => path.join(dir, name));
  const host = ts.createCompilerHost(options, true);
  const getSourceFile = host.getSourceFile.bind(host);
  const fileExists = host.fileExists.bind(host);
  const readFile = host.readFile.bind(host);
  host.getSourceFile = (name, language, onError, create) =>
    virtual.has(name) ? ts.createSourceFile(name, virtual.get(name), language, true, ts.ScriptKind.TS) : getSourceFile(name, language, onError, create);
  host.fileExists = (name) => virtual.has(name) || fileExists(name);
  host.readFile = (name) => (virtual.has(name) ? virtual.get(name) : readFile(name));
  const rootNames = [...new Set([...fileNames, ...configs, ...virtual.keys()])];
  return ts.createProgram({ rootNames, options, host });
}

const getText = (name) => (name && (ts.isIdentifier(name) || ts.isStringLiteral(name) || ts.isPrivateIdentifier(name)) ? name.text : null);

function isMemberDeclaration(node) {
  return (
    ts.isPropertySignature(node) ||
    ts.isMethodSignature(node) ||
    ts.isPropertyDeclaration(node) ||
    ts.isMethodDeclaration(node) ||
    ts.isGetAccessorDeclaration(node) ||
    ts.isSetAccessorDeclaration(node) ||
    ts.isPropertyAssignment(node) ||
    ts.isShorthandPropertyAssignment(node) ||
    ts.isEnumMember(node)
  );
}

/**
 * The node key of the top-level declaration that holds `node`, with the outermost member below it:
 * `ts:<file>|<Name>` or `ts:<file>|<Name>.<member>`; null outside a named declaration.
 */
function getDeclarationKey(node, rel) {
  const chain = [];
  for (let current = node; current && !ts.isSourceFile(current); current = current.parent) chain.push(current);
  const top = chain.at(-1);
  if (!top) return null;
  let name = null;
  let depth = chain.length - 1;
  if (ts.isVariableStatement(top)) {
    depth = chain.findIndex((one) => ts.isVariableDeclaration(one) && one.parent?.parent === top);
    if (depth < 0) return null;
    name = getText(chain[depth].name);
  } else if (ts.isExportAssignment(top)) name = "default";
  else if (ts.isFunctionDeclaration(top) || ts.isClassDeclaration(top) || ts.isInterfaceDeclaration(top) || ts.isTypeAliasDeclaration(top) || ts.isEnumDeclaration(top)) {
    name = getText(top.name) ?? (top.modifiers?.some((one) => one.kind === ts.SyntaxKind.DefaultKeyword) ? "default" : null);
  }
  if (!name) return null;
  let member = null;
  for (let i = depth - 1; i >= 0; i--) {
    if (isMemberDeclaration(chain[i])) {
      member = getText(chain[i].name);
      break;
    }
  }
  return `ts:${rel}|${name}${member ? "." + member : ""}`;
}

const upperFirst = (text) => text.charAt(0).toUpperCase() + text.slice(1);

/** The RPC a member of the method table of a service of the protocol names, or null. */
function getRpcKey(declaration) {
  if (!ts.isPropertySignature(declaration) || !ts.isTypeLiteralNode(declaration.parent)) return null;
  const service = getServiceName(declaration.parent);
  const method = getText(declaration.name);
  return service && method ? `rpc:${service}.${upperFirst(method)}` : null;
}

/** The service whose method table is the type literal `literal`: `export const <Service>: GenService<{ ... }>`. */
function getServiceName(literal) {
  const reference = literal.parent;
  if (!reference || !ts.isTypeReferenceNode(reference)) return null;
  const variable = reference.parent;
  if (!variable || !ts.isVariableDeclaration(variable)) return null;
  return getText(variable.name);
}

function isProtocolSchemaFile(rel) {
  return isWithin(rel, protocolDir) && rel.endsWith("_pb.ts");
}

function isLibraryFile(rel) {
  return libraryDirs.some((dir) => isWithin(rel, dir)) && !rel.split("/").includes("node_modules");
}

/** The node a declaration stands for: an RPC of the protocol or a declaration of a library. */
function getDeclarationTarget(declaration) {
  const rel = toRelative(getRealFile(declaration.getSourceFile().fileName));
  if (isProtocolSchemaFile(rel)) return getRpcKey(declaration);
  if (isLibraryFile(rel)) return getDeclarationKey(declaration, rel);
  return null;
}

/** Looks through the type alias arguments of `type`, two levels down, for the method table of a service. */
function findServiceTable(type, depth = 0) {
  if (!type || depth > 2) return null;
  for (const declaration of type.getSymbol()?.declarations ?? []) {
    if (ts.isTypeLiteralNode(declaration) && isProtocolSchemaFile(toRelative(getRealFile(declaration.getSourceFile().fileName))) && getServiceName(declaration)) return declaration;
  }
  for (const argument of type.aliasTypeArguments ?? []) {
    const found = findServiceTable(argument, depth + 1);
    if (found) return found;
  }
  return null;
}

function getTargets(checker, node) {
  let symbol = ts.isShorthandPropertyAssignment(node.parent) && node.parent.name === node ? checker.getShorthandAssignmentValueSymbol(node.parent) : checker.getSymbolAtLocation(node);
  if (symbol && symbol.flags & ts.SymbolFlags.Alias) {
    try {
      symbol = checker.getAliasedSymbol(symbol);
    } catch {
      return [];
    }
  }
  const targets = new Set();
  for (const declaration of symbol?.declarations ?? []) {
    const target = getDeclarationTarget(declaration);
    if (target) targets.add(target);
  }
  const isMapped = !symbol || !symbol.declarations?.length || symbol.flags & ts.SymbolFlags.Transient;
  if (targets.size === 0 && isMapped && ts.isPropertyAccessExpression(node.parent) && node.parent.name === node) {
    // A client made from a service names its methods through mapped types, which may keep no declaration.
    const table = findServiceTable(checker.getTypeAtLocation(node.parent.expression));
    const member = table?.members.find((one) => getText(one.name) === node.text);
    const target = member ? getRpcKey(member) : null;
    if (target) targets.add(target);
  }
  return [...targets];
}

/** The library declarations of the property `name` of `type`, as node keys. */
function getPropertyTargets(checker, type, name) {
  const targets = new Set();
  for (const one of type.isUnion() ? type.types : [type]) {
    const property = checker.getPropertyOfType(checker.getApparentType(one), name);
    for (const declaration of property?.declarations ?? []) {
      const target = getDeclarationTarget(declaration);
      if (target) targets.add(target);
    }
  }
  return targets;
}

/**
 * The members of a library that answer a member of an interface of a library: a property of an object
 * literal typed by the interface, or a member of a class that implements or extends it. A consumer that
 * calls the interface member calls whichever member implements it, so each pair is a use of the
 * answering member by the interface member, as [answer, interface member].
 */
function listAnswers(checker, node, rel) {
  const pairs = [];
  const add = (element, type) => {
    const name = getText(element.name);
    const answer = getDeclarationKey(element, rel);
    if (!name || !answer || !type) return;
    for (const target of getPropertyTargets(checker, type, name)) if (target !== answer) pairs.push([answer, target]);
  };
  if (ts.isObjectLiteralExpression(node)) {
    const type = checker.getContextualType(node);
    for (const element of node.properties) if (isMemberDeclaration(element)) add(element, type);
  } else if (ts.isClassLike(node)) {
    for (const clause of node.heritageClauses ?? []) {
      for (const parent of clause.types) {
        const type = checker.getTypeAtLocation(parent);
        for (const element of node.members) if (isMemberDeclaration(element)) add(element, type);
      }
    }
  }
  return pairs;
}

/** Whether `node` is a value handed to a typed place: an argument, an initializer, a property, a return. */
function isHandedValue(node) {
  const parent = node.parent;
  if (!parent) return false;
  if ((ts.isCallExpression(parent) || ts.isNewExpression(parent)) && parent.arguments?.includes(node)) return true;
  if (ts.isPropertyAssignment(parent)) return parent.initializer === node;
  if (ts.isVariableDeclaration(parent)) return parent.initializer === node && parent.type !== undefined;
  if (ts.isReturnStatement(parent)) return true;
  if (ts.isArrowFunction(parent)) return parent.body === node;
  if (ts.isBinaryExpression(parent)) return parent.right === node && parent.operatorToken.kind === ts.SyntaxKind.EqualsToken;
  return false;
}

/** The type the place `node` is handed to expects, or null. */
function getExpectedType(checker, node) {
  if (ts.isShorthandPropertyAssignment(node)) {
    const literal = checker.getContextualType(node.parent);
    const property = literal && checker.getPropertyOfType(literal, node.name.text);
    return property ? checker.getTypeOfSymbolAtLocation(property, node) : null;
  }
  return isHandedValue(node) ? (checker.getContextualType(node) ?? null) : null;
}

/**
 * The members of a library or RPCs that a value passes to a type of another declaration, such as a
 * client of the core client passed where a package declares the calls it makes: the place that passes the
 * value uses each member the expected type names.
 */
function listHandedTargets(checker, node) {
  const expected = getExpectedType(checker, node);
  if (!expected || expected.flags & ts.TypeFlags.Any) return [];
  const value = checker.getTypeAtLocation(ts.isShorthandPropertyAssignment(node) ? node.name : node);
  if (!value || value === expected || !(value.flags & ts.TypeFlags.Object)) return [];
  const targets = new Set();
  for (const property of checker.getPropertiesOfType(expected)) {
    const own = getPropertyTargets(checker, expected, property.name);
    for (const target of getPropertyTargets(checker, value, property.name)) if (!own.has(target)) targets.add(target);
  }
  return [...targets];
}

function readUses(program, file, rel, uses) {
  const checker = program.getTypeChecker();
  const isLibrary = isLibraryFile(rel);
  const visit = (node) => {
    if (ts.isImportDeclaration(node) || ts.isExportDeclaration(node) || ts.isImportEqualsDeclaration(node)) return;
    if (isLibrary) for (const [answer, target] of listAnswers(checker, node, rel)) uses.add(JSON.stringify([answer, rel, target]));
    if (ts.isExpression(node) || ts.isShorthandPropertyAssignment(node)) {
      const handed = listHandedTargets(checker, node);
      if (handed.length > 0) {
        const user = getDeclarationKey(node, rel) ?? "";
        for (const target of handed) uses.add(JSON.stringify([target, rel, user]));
      }
    }
    if (ts.isIdentifier(node)) {
      const targets = getTargets(checker, node);
      if (targets.length > 0) {
        const user = getDeclarationKey(node, rel) ?? "";
        for (const target of targets) uses.add(JSON.stringify([target, rel, user]));
      }
    }
    ts.forEachChild(node, visit);
  };
  visit(file);
}

/** The script files the entries of the `exports` of the package in `dir` name. */
function readEntries(dir) {
  const manifest = JSON.parse(fs.readFileSync(path.join(dir, "package.json"), "utf8"));
  const exported = typeof manifest.exports === "string" ? { ".": manifest.exports } : (manifest.exports ?? { ".": manifest.types ?? manifest.main });
  const pick = (target) => (typeof target === "string" ? target : target && (pick(target.types) ?? pick(target.import) ?? pick(target.default)));
  return Object.entries(exported)
    .map(([entry, target]) => ({ entry, file: pick(target) }))
    .filter(({ file }) => file && !file.includes("*") && /\.(ts|mts)$/.test(file))
    .map(({ entry, file }) => ({ entry, file: path.resolve(dir, file) }));
}

function readSdkItems(program, dir) {
  const checker = program.getTypeChecker();
  const items = new Map();
  for (const { entry, file } of readEntries(dir)) {
    const source = program.getSourceFile(file);
    const module = source && checker.getSymbolAtLocation(source);
    if (!module) throw new Error(`${file} is not part of the program of ${dir}`);
    for (const symbol of checker.getExportsOfModule(module)) {
      const resolved = symbol.flags & ts.SymbolFlags.Alias ? checker.getAliasedSymbol(symbol) : symbol;
      const declaration = resolved.declarations?.[0];
      if (!declaration) continue;
      const rel = toRelative(getRealFile(declaration.getSourceFile().fileName));
      if (!isWithin(rel, sdkDir)) continue;
      const key = getDeclarationKey(declaration, rel);
      if (!key || items.has(key)) continue;
      const line = declaration.getSourceFile().getLineAndCharacterOfPosition(declaration.getStart()).line + 1;
      const name = `${rel}:${symbol.name}${entry === "." ? "" : ` (${entry})`}`;
      items.set(key, { key, name, position: `${rel}:${line}` });
    }
  }
  return [...items.values()];
}

function main() {
  const dirs = listPackageDirs();
  const uses = new Set();
  const read = new Set();
  let items = [];
  for (const dir of dirs) {
    if (isWithin(toRelative(dir), protocolDir)) continue;
    const program = createPackageProgram(dir);
    const others = dirs.filter((one) => one !== dir && one.startsWith(dir + path.sep));
    if (toRelative(dir) === sdkDir) items = readSdkItems(program, dir);
    for (const source of program.getSourceFiles()) {
      if (source.isDeclarationFile) continue;
      const real = getRealFile(source.fileName);
      if (!real.startsWith(dir + path.sep) || others.some((one) => real.startsWith(one + path.sep)) || read.has(real)) continue;
      const rel = toRelative(real);
      if (rel.split("/").includes("node_modules")) continue;
      read.add(real);
      readUses(program, source, rel, uses);
    }
  }
  const output = { items, uses: [...uses].sort().map((one) => JSON.parse(one)), packages: dirs.length, files: read.size };
  process.stdout.write(JSON.stringify(output));
}

main();
