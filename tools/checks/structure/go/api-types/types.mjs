#!/usr/bin/env node
/**
 * The TypeScript half of the check `go/api-types`: the JSON methods a plugin frontend calls through the
 * `Api` of `@numen/plugin-sdk`, with the request and response type of each call, read with the type
 * checker of the plugin's tsconfig and the script blocks of its Vue files. Prints JSON keyed by each
 * plugin directory as given: `files`, `calls` as {method, place, request, response} and `failures` as
 * {place, reason} for a call whose method or types the checker cannot name.
 *
 *   node types.mjs <plugin dir>...
 *
 * A call made inside a function of the plugin that passes its own parameter as the method, such as a
 * helper that retries, is read at each call of that function, with the type arguments given there.
 *
 * A type is printed as a shape: {kind: "string", values?}, {kind: "number"}, {kind: "boolean"},
 * {kind: "any"}, {kind: "array" | "record", element}, {kind: "object", fields: {name: {isOptional, shape}}}
 * or {kind: "other", text}. A type met again inside itself is printed as {kind: "any"}.
 */
import fs from "node:fs";
import path from "node:path";
import process from "node:process";
import { parse as parseVue } from "@vue/compiler-sfc";
import ts from "typescript";

const SDK_PACKAGE = "@numen/plugin-sdk";
const API_INTERFACE = "Api";
/** How many functions a method name may pass through before the call that names it. */
const MAX_DEPTH = 6;

const packageNames = new Map();

/** The name in the nearest package.json above a file. */
function getPackageName(file) {
  for (let dir = path.dirname(file); ; dir = path.dirname(dir)) {
    if (!packageNames.has(dir)) {
      const manifest = path.join(dir, "package.json");
      packageNames.set(dir, fs.existsSync(manifest) ? (JSON.parse(fs.readFileSync(manifest, "utf8")).name ?? null) : undefined);
    }
    const name = packageNames.get(dir);
    if (name !== undefined) return name;
    if (path.dirname(dir) === dir) return null;
  }
}

const isTestFile = (file) => [".d.ts", ".test.ts", ".spec.ts", ".stories.ts"].some((suffix) => file.endsWith(suffix));

function isPluginSource(root, file) {
  const rel = path.relative(root, file);
  return !rel.startsWith("..") && !path.isAbsolute(rel) && !rel.split(path.sep).includes("node_modules") && !isTestFile(file);
}

const SKIPPED_DIRS = new Set(["node_modules", "dist", "build"]);
/** A Vue file is read as `<file>.vue.ts`, which is also where TypeScript looks for an import of `<file>.vue`. */
const VIRTUAL_SUFFIX = ".vue.ts";

function listVueFiles(dir) {
  const vueFiles = [];
  for (const one of fs.readdirSync(dir, { withFileTypes: true })) {
    const file = path.join(dir, one.name);
    if (one.isDirectory() && !SKIPPED_DIRS.has(one.name) && !one.name.startsWith(".")) vueFiles.push(...listVueFiles(file));
    else if (one.isFile() && one.name.endsWith(".vue")) vueFiles.push(file);
  }
  return vueFiles;
}

/** The script blocks of a Vue file in place, the rest blanked, so that a line of the text is the line of the file. */
function readVueScript(file) {
  const source = fs.readFileSync(file, "utf8");
  const { descriptor } = parseVue(source, { filename: file });
  const chars = source.split("").map((char) => (char === "\n" ? "\n" : " "));
  for (const block of [descriptor.script, descriptor.scriptSetup]) {
    if (!block) continue;
    for (let i = block.loc.start.offset; i < block.loc.end.offset; i++) chars[i] = source[i];
  }
  return chars.join("");
}

function createProgram(root) {
  const configFile = path.join(root, "tsconfig.json");
  const config = ts.readConfigFile(configFile, ts.sys.readFile);
  if (config.error) throw new Error(`${configFile}: ${ts.flattenDiagnosticMessageText(config.error.messageText, "\n")}`);
  const parsed = ts.parseJsonConfigFileContent(config.config, ts.sys, root);
  const options = { ...parsed.options, noEmit: true };
  const virtual = new Map(listVueFiles(root).map((file) => [file + ".ts", readVueScript(file)]));
  const host = ts.createCompilerHost(options, true);
  const getSourceFile = host.getSourceFile.bind(host);
  const fileExists = host.fileExists.bind(host);
  const readFile = host.readFile.bind(host);
  host.getSourceFile = (name, language, onError, create) =>
    virtual.has(name) ? ts.createSourceFile(name, virtual.get(name), language, true, ts.ScriptKind.TS) : getSourceFile(name, language, onError, create);
  host.fileExists = (name) => virtual.has(name) || fileExists(name);
  host.readFile = (name) => (virtual.has(name) ? virtual.get(name) : readFile(name));
  return ts.createProgram({ rootNames: [...new Set([...parsed.fileNames, ...virtual.keys()])], options, host });
}

/** Where the type parameters of a method of the Api stand: the one the request parameter has, and the other. */
function getTypeSlots(declaration) {
  const names = (declaration.typeParameters ?? []).map((one) => one.name.text);
  const request = declaration.parameters[1]?.type;
  const index = request && ts.isTypeReferenceNode(request) && ts.isIdentifier(request.typeName) ? names.indexOf(request.typeName.text) : -1;
  return names.length === 2 && index >= 0 ? { request: index, response: 1 - index } : null;
}

function readPlugin(dir) {
  const root = fs.realpathSync(dir);
  if (!fs.existsSync(path.join(root, "tsconfig.json"))) return { files: 0, calls: [], failures: [] };
  const program = createProgram(root);
  const checker = program.getTypeChecker();
  const files = program.getSourceFiles().filter((file) => isPluginSource(root, file.fileName));

  const getPlace = (node) => {
    const file = node.getSourceFile();
    const { line } = file.getLineAndCharacterOfPosition(node.getStart());
    const name = file.fileName.endsWith(VIRTUAL_SUFFIX) ? file.fileName.slice(0, -3) : file.fileName;
    return `${path.relative(root, name).split(path.sep).join("/")}:${line + 1}`;
  };

  const isApiMethod = (declaration) =>
    declaration !== undefined &&
    ts.isMethodSignature(declaration) &&
    ts.isInterfaceDeclaration(declaration.parent) &&
    declaration.parent.name.text === API_INTERFACE &&
    getPackageName(declaration.getSourceFile().fileName) === SDK_PACKAGE;

  /** The calls of each function, by the declaration of the signature the checker resolves at the call. */
  const callers = new Map();
  for (const file of files) {
    const visit = (node) => {
      if (ts.isCallExpression(node)) {
        const declaration = checker.getResolvedSignature(node)?.declaration;
        if (declaration) callers.set(declaration, [...(callers.get(declaration) ?? []), node]);
      }
      ts.forEachChild(node, visit);
    };
    visit(file);
  }

  const getParameter = (expression) => {
    if (!ts.isIdentifier(expression)) return null;
    const declaration = checker.getSymbolAtLocation(expression)?.valueDeclaration;
    return declaration && ts.isParameter(declaration) ? declaration : null;
  };

  /** The type parameters of `fn` bound to the type arguments of one call of it, given or inferred. */
  const bindTypeParameters = (fn, site, bindings) => {
    const parameters = fn.typeParameters ?? [];
    if (parameters.length === 0) return bindings;
    const next = new Map(bindings);
    if (site.typeArguments) {
      parameters.forEach((one, k) => next.set(checker.getSymbolAtLocation(one.name), { node: site.typeArguments[k] }));
      return next;
    }
    const signature = checker.getResolvedSignature(site);
    const inferred = signature && checker.getTypeArgumentsForResolvedSignature(signature);
    if (!inferred) return null;
    parameters.forEach((one, k) => next.set(checker.getSymbolAtLocation(one.name), { type: inferred[k] }));
    return next;
  };

  /** The method names an argument stands for, each with the type parameters bound on its way. */
  const getMethods = (expression, bindings, depth) => {
    const parameter = getParameter(expression);
    if (parameter) {
      const fn = parameter.parent;
      const sites = callers.get(fn) ?? [];
      if (depth >= MAX_DEPTH) return [{ place: getPlace(expression), reason: `the method passes through more than ${MAX_DEPTH} functions` }];
      if (sites.length === 0) return [{ place: getPlace(expression), reason: "the method is a parameter of a function the plugin never calls" }];
      const index = fn.parameters.indexOf(parameter);
      return sites.flatMap((site) => {
        const next = bindTypeParameters(fn, site, bindings);
        const argument = site.arguments[index];
        if (!next || !argument) return [{ place: getPlace(site), reason: "the checker gives no type arguments or no method for this call" }];
        return getMethods(argument, next, depth + 1);
      });
    }
    const type = checker.getTypeAtLocation(expression);
    if (type.isStringLiteral()) return [{ method: type.value, bindings, place: getPlace(expression) }];
    return [{ place: getPlace(expression), reason: "the method is not a string the checker can name" }];
  };

  const hasBoundReference = (node, bindings) => {
    if (ts.isTypeReferenceNode(node) && bindings.has(checker.getSymbolAtLocation(node.typeName))) return true;
    return ts.forEachChild(node, (child) => hasBoundReference(child, bindings) || undefined) === true;
  };

  const resolveBinding = (bound, bindings, depth) => {
    if (depth > MAX_DEPTH) return null;
    if ("node" in bound) return evaluate(bound.node, bindings, depth + 1);
    if (!bound.type) return null;
    if (bound.type.flags & ts.TypeFlags.TypeParameter) {
      const next = bindings.get(bound.type.symbol);
      return next ? resolveBinding(next, bindings, depth + 1) : null;
    }
    return bound.type;
  };

  /** The type a type node names once the type parameters in it are replaced by their bindings. */
  const evaluate = (node, bindings, depth = 0) => {
    if (!node) return null;
    if (ts.isParenthesizedTypeNode(node)) return evaluate(node.type, bindings, depth);
    if (ts.isTypeReferenceNode(node) && !node.typeArguments) {
      const bound = bindings.get(checker.getSymbolAtLocation(node.typeName));
      if (bound) return resolveBinding(bound, bindings, depth);
    }
    if (ts.isIndexedAccessTypeNode(node)) {
      const object = evaluate(node.objectType, bindings, depth);
      const index = evaluate(node.indexType, bindings, depth);
      const property = object && index?.isStringLiteral() ? checker.getPropertyOfType(object, index.value) : undefined;
      return property ? checker.getTypeOfSymbol(property) : null;
    }
    return hasBoundReference(node, bindings) ? null : checker.getTypeFromTypeNode(node);
  };

  const getShape = (type, stack = []) => {
    const flags = type.flags;
    if (flags & (ts.TypeFlags.Any | ts.TypeFlags.Unknown) || stack.includes(type)) return { kind: "any" };
    if (flags & ts.TypeFlags.StringLiteral) return { kind: "string", values: [type.value] };
    if (flags & ts.TypeFlags.StringLike) return { kind: "string" };
    if (flags & ts.TypeFlags.NumberLike) return { kind: "number" };
    if (flags & ts.TypeFlags.BooleanLike) return { kind: "boolean" };
    if (type.isUnion()) {
      const members = type.types.filter((one) => !(one.flags & (ts.TypeFlags.Undefined | ts.TypeFlags.Null)));
      if (members.length === 1) return getShape(members[0], stack);
      if (members.length > 0 && members.every((one) => one.flags & ts.TypeFlags.BooleanLike)) return { kind: "boolean" };
      if (members.length > 0 && members.every((one) => one.flags & ts.TypeFlags.StringLiteral)) {
        return { kind: "string", values: members.map((one) => one.value) };
      }
      return { kind: "other", text: checker.typeToString(type) };
    }
    const inner = [...stack, type];
    if (checker.isArrayType(type)) return { kind: "array", element: getShape(checker.getTypeArguments(type)[0], inner) };
    if (flags & (ts.TypeFlags.Object | ts.TypeFlags.Intersection) && !checker.isTupleType(type) && type.getCallSignatures().length === 0) {
      const properties = checker.getPropertiesOfType(type);
      const index = checker.getIndexInfosOfType(type).find((one) => one.keyType.flags & ts.TypeFlags.String);
      if (properties.length === 0 && index && !(index.type.flags & ts.TypeFlags.Never)) {
        return { kind: "record", element: getShape(index.type, inner) };
      }
      const fields = {};
      for (const property of properties) {
        fields[property.name] = {
          isOptional: (property.flags & ts.SymbolFlags.Optional) !== 0,
          shape: getShape(checker.getTypeOfSymbol(property), inner),
        };
      }
      return { kind: "object", fields };
    }
    return { kind: "other", text: checker.typeToString(type) };
  };

  const calls = [];
  const failures = [];
  for (const [declaration, sites] of callers) {
    if (!isApiMethod(declaration)) continue;
    const slots = getTypeSlots(declaration);
    for (const site of sites) {
      if (!slots || site.typeArguments?.length !== 2) {
        failures.push({ place: getPlace(site), reason: "the call of the Api gives no type arguments for its request and response" });
        continue;
      }
      if (!site.arguments[0]) {
        failures.push({ place: getPlace(site), reason: "the call of the Api names no method" });
        continue;
      }
      for (const one of getMethods(site.arguments[0], new Map(), 0)) {
        if (one.reason) {
          failures.push({ place: one.place, reason: one.reason });
          continue;
        }
        const request = evaluate(site.typeArguments[slots.request], one.bindings);
        const response = evaluate(site.typeArguments[slots.response], one.bindings);
        if (!request || !response) {
          failures.push({ place: one.place, reason: "the checker cannot name the request or the response type of this call" });
          continue;
        }
        calls.push({ method: one.method, place: one.place, request: getShape(request), response: getShape(response) });
      }
    }
  }
  return { files: files.length, calls, failures };
}

const plugins = {};
for (const dir of process.argv.slice(2)) plugins[dir] = readPlugin(dir);
process.stdout.write(JSON.stringify(plugins));
