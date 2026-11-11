import fs from "node:fs";
import path from "node:path";
import ts from "typescript";
import { REPO_ROOT, failOnFindings, requireMatches } from "../../../lib/graph.mjs";

/** The directory of the TypeScript plugin SDK package. */
const SDK_DIR = "libs/plugin/sdk-ts";

/** The directory of the package that holds the generated schema. */
const PROTOCOL_DIR = "libs/core/protocol";

/** True when `file` is `dir` or lies below it, outside any `node_modules` directory. */
function isInPackage(file, dir) {
  const relative = path.relative(dir, file);
  return !relative.startsWith("..") && !path.isAbsolute(relative) && !relative.split(path.sep).includes("node_modules");
}

/** The compiler options of the package in `dir`, read from its `tsconfig.json`, for a program that emits nothing. */
function readCompilerOptions(dir) {
  const configFile = path.join(dir, "tsconfig.json");
  const read = ts.readConfigFile(configFile, ts.sys.readFile);
  if (read.error) throw new Error(ts.flattenDiagnosticMessageText(read.error.messageText, "\n"));
  const parsed = ts.parseJsonConfigFileContent(read.config, ts.sys, dir);
  return { ...parsed.options, noEmit: true, composite: false, incremental: false, tsBuildInfoFile: undefined };
}

/** A script a module can be built from; a stylesheet or an asset in `exports` is not one. */
const SCRIPT_FILE = /\.(ts|tsx|mts|cts|js|mjs|cjs)$/;

/** The conditions of an `exports` target, in the order a type checker reads them. */
const CONDITIONS = ["types", "import", "default"];

/** The path an `exports` target names: a string, or the first condition of an object that names one. */
function getTargetPath(target) {
  if (typeof target === "string") return target;
  if (!target || typeof target !== "object") return undefined;
  for (const condition of CONDITIONS) {
    const one = getTargetPath(target[condition]);
    if (one) return one;
  }
  return undefined;
}

/** The files a target with one `*` matches in `dir`, or the target itself. */
function expandTarget(dir, target) {
  const star = target.indexOf("*");
  if (star < 0) return [path.resolve(dir, target)];
  const prefix = path.resolve(dir, target.slice(0, star));
  const suffix = target.slice(star + 1);
  const parent = path.dirname(prefix);
  if (!fs.existsSync(parent)) return [];
  return fs
    .readdirSync(parent, { recursive: true })
    .map((name) => path.join(parent, name))
    .filter((file) => file.startsWith(prefix) && file.endsWith(suffix));
}

/**
 * The script files every entry of the `exports` of the package in `dir` points to, as `{ entry, file }`.
 * A package without `exports` has the one entry its `types` or `main` names.
 */
function readEntries(dir) {
  const manifest = JSON.parse(fs.readFileSync(path.join(dir, "package.json"), "utf8"));
  const exported = typeof manifest.exports === "string" ? { ".": manifest.exports } : manifest.exports;
  const targets = exported ? Object.entries(exported) : [[".", manifest.types ?? manifest.main]];
  const entries = targets.flatMap(([entry, target]) => {
    const one = getTargetPath(target);
    return one ? expandTarget(dir, one).filter((file) => SCRIPT_FILE.test(file)).map((file) => ({ entry, file })) : [];
  });
  if (entries.length === 0) throw new Error(`${dir}/package.json declares no script entry`);
  return entries;
}

/** Collects the declarations of a protocol package that the exported declarations of a package mention. */
class ExposureWalk {
  constructor(program, packageDir, protocolDir, rootDir) {
    this.checker = program.getTypeChecker();
    this.packageDir = packageDir;
    this.protocolDir = protocolDir;
    this.rootDir = rootDir;
    this.visited = new Set();
    this.seenBySubject = new Map();
    this.found = new Map();
    this.subject = null;
  }

  /** The real path of the file a node is declared in. */
  getFile(node) {
    return fs.realpathSync(node.getSourceFile().fileName);
  }

  /** The position of a declaration as `{ file, line }`, relative to the root directory. */
  getPosition(node) {
    const source = node.getSourceFile();
    const { line } = source.getLineAndCharacterOfPosition(node.getStart());
    return { file: path.relative(this.rootDir, this.getFile(node)).split(path.sep).join("/"), line: line + 1 };
  }

  /** Records that the subject mentions the protocol declaration `symbol`. */
  record(symbol, declaration) {
    const subject = this.subject;
    const name = this.checker.getFullyQualifiedName(symbol);
    const key = `${subject.file}:${subject.line}:${subject.name}:${name}`;
    if (this.found.has(key)) return;
    const origin = path.relative(this.protocolDir, this.getFile(declaration)).split(path.sep).join("/");
    this.found.set(key, { file: subject.file, line: subject.line, symbol: `${subject.name} mentions ${name} declared in ${origin}` });
  }

  /** Walks the exported symbol `symbol` of the module of the package entry `entry`. */
  walkExport(symbol, entry) {
    const resolved = symbol.flags & ts.SymbolFlags.Alias ? this.checker.getAliasedSymbol(symbol) : symbol;
    const declarations = resolved.declarations ?? [];
    if (declarations.length === 0) return;
    const name = entry === "." ? symbol.name : `${symbol.name} of ${entry}`;
    this.subject = { name, ...this.getPosition(symbol.declarations?.[0] ?? declarations[0]) };
    const direct = declarations.find((one) => isInPackage(this.getFile(one), this.protocolDir));
    if (direct) {
      this.record(resolved, direct);
      return;
    }
    if (resolved.flags & ts.SymbolFlags.Type) {
      this.walk(this.checker.getDeclaredTypeOfSymbol(resolved));
    }
    if (resolved.flags & ts.SymbolFlags.Value) {
      this.walk(this.checker.getTypeOfSymbolAtLocation(resolved, declarations[0]));
    }
  }

  /** True the first time the current subject reaches `type`; a recursive type is walked once per subject. */
  isFirstVisit(type) {
    const key = `${this.subject.file}:${this.subject.line}:${this.subject.name}`;
    if (!this.seenBySubject.has(key)) this.seenBySubject.set(key, new Set());
    const seen = this.seenBySubject.get(key);
    if (seen.has(type)) return false;
    seen.add(type);
    return true;
  }

  /** Walks the members, signatures and type arguments of `type`. A protocol declaration is recorded for every subject that reaches it. */
  walk(type) {
    if (!this.isFirstVisit(type)) return;
    if (type.isUnionOrIntersection()) {
      type.types.forEach((one) => this.walk(one));
      return;
    }
    (type.aliasTypeArguments ?? []).forEach((one) => this.walk(one));
    if (type.flags & ts.TypeFlags.Object && type.objectFlags & ts.ObjectFlags.Reference) {
      this.checker.getTypeArguments(type).forEach((one) => this.walk(one));
    }
    const symbol = type.aliasSymbol ?? type.getSymbol();
    const declarations = symbol?.declarations ?? [];
    const protocol = declarations.find((one) => isInPackage(this.getFile(one), this.protocolDir));
    if (protocol) {
      this.record(symbol, protocol);
      return;
    }
    if (this.visited.has(type)) return;
    this.visited.add(type);
    if (type.flags & ts.TypeFlags.TypeParameter) {
      const constraint = this.checker.getBaseConstraintOfType(type);
      if (constraint) this.walk(constraint);
    }
    const own = declarations.find((one) => isInPackage(this.getFile(one), this.packageDir));
    if (!own) return;
    const outer = this.subject;
    if (!symbol.name.startsWith("__")) this.subject = { name: symbol.name, ...this.getPosition(own) };
    this.walkMembers(type);
    this.subject = outer;
  }

  /** Walks the properties, call and construct signatures, index signatures and bases of an object type. */
  walkMembers(type) {
    for (const property of this.checker.getPropertiesOfType(type)) {
      this.walk(this.checker.getTypeOfSymbol(property));
    }
    for (const signature of [...type.getCallSignatures(), ...type.getConstructSignatures()]) {
      signature.getParameters().forEach((parameter) => this.walk(this.checker.getTypeOfSymbol(parameter)));
      this.walk(signature.getReturnType());
      (signature.getTypeParameters() ?? []).forEach((parameter) => this.walk(parameter));
    }
    for (const info of this.checker.getIndexInfosOfType(type)) {
      this.walk(info.keyType);
      this.walk(info.type);
    }
    if (type.isClassOrInterface()) {
      this.checker.getBaseTypes(type).forEach((one) => this.walk(one));
    }
  }
}

/**
 * Lists the findings for every entry of the package in `packageDir`: every exported declaration whose
 * type reaches a declaration of the package in `protocolDir`. `options` are the compiler options of the
 * program; `rootDir` is what the reported files are relative to. Returns the findings and the number of
 * exported declarations and source files of each package that were read.
 */
export function findProtocolExposures({ packageDir, protocolDir, options, rootDir }) {
  const entries = readEntries(packageDir);
  const program = ts.createProgram({ rootNames: entries.map(({ file }) => file), options });
  const checker = program.getTypeChecker();
  const walk = new ExposureWalk(program, fs.realpathSync(packageDir), fs.realpathSync(protocolDir), rootDir);
  let exported = 0;
  for (const { entry, file } of entries) {
    const entrySource = program.getSourceFile(file);
    if (!entrySource) throw new Error(`${file} is not part of the program`);
    const entrySymbol = checker.getSymbolAtLocation(entrySource);
    const exports = entrySymbol ? checker.getExportsOfModule(entrySymbol) : [];
    exported += exports.length;
    for (const symbol of exports) walk.walkExport(symbol, entry);
  }
  const files = program.getSourceFiles().map((one) => fs.realpathSync(one.fileName));
  return {
    findings: [...walk.found.values()],
    entries: entries.length,
    exported,
    ownFiles: files.filter((one) => isInPackage(one, walk.packageDir)).length,
    protocolFiles: files.filter((one) => isInPackage(one, walk.protocolDir)).length,
  };
}

/** The exported declarations of every entry of `@numen/plugin-sdk` mention no declaration of `@numen/protocol`. */
export function verifySdkTypesEncapsulateProtocol() {
  const packageDir = path.join(REPO_ROOT, SDK_DIR);
  const protocolDir = path.join(REPO_ROOT, PROTOCOL_DIR);
  const { findings, entries, exported, ownFiles, protocolFiles } = findProtocolExposures({
    packageDir,
    protocolDir,
    options: readCompilerOptions(packageDir),
    rootDir: REPO_ROOT,
  });
  requireMatches("entries of the SDK exports", entries);
  requireMatches("exported declarations of the SDK entries", exported);
  requireMatches("source files of the SDK program", ownFiles);
  requireMatches("source files of the schema reached from the SDK program", protocolFiles);
  failOnFindings("An entry of the plugin SDK exposes declarations of the schema package", findings);
}

export const SDK_VERIFIERS = [
  {
    id: "sdk-ts-encapsulates-protocol",
    scope: "libs",
    module: "sdk",
    name: "Plugin SDK Encapsulates Protocol (no exported declaration of an entry in the exports of the TypeScript SDK reaches a declaration of libs/core/protocol)",
    run: verifySdkTypesEncapsulateProtocol,
  },
];
