import fs from "node:fs";
import path from "node:path";
import { createRequire } from "node:module";
import { REPO_ROOT, listShippedFiles, readGo, readSource } from "./helpers.mjs";

const require = createRequire(import.meta.url);

/** Where the shared vocabulary of plugins is defined; words found as literals there are data, not plugin names. */
const DATA_SOURCES = ["libs/plugin-sdk-ts/src", "libs/plugin-sdk-go", "libs/protocol/proto"];

/** The libraries whose own words are common vocabulary, unless the check reads that library. */
const HOST_VOCABULARY = ["libs/core", "libs/notes"];

const MIN_WORD_LENGTH = 3;
const TOKEN = /^[A-Za-z][\w.:*-]*$/;
const SEPARATORS = /[.:*\-_/\s]+/;
const HASH = /^[0-9a-f]{32,}$/;

/** The plugin directories that hold a manifest. */
export function listManifestFiles() {
  const root = path.join(REPO_ROOT, "plugins");
  return fs
    .readdirSync(root, { withFileTypes: true })
    .filter((entry) => entry.isDirectory() && fs.existsSync(path.join(root, entry.name, "manifest.json")))
    .map((entry) => `plugins/${entry.name}/manifest.json`);
}

/** The manifest fields that hold presentation or configuration values, which name no plugin. */
const PRESENTATION_KEYS = new Set(["icon", "key", "title", "description", "default", "label", "unit", "placeholder", "name"]);

/** The manifest fields that name the windows a plugin runs in; the host defines them. */
const HOST_KEYS = new Set(["targets"]);

/** Every string value of a parsed JSON document outside the fields in `skipKeys`; with `onlyKeys`, only the values under those fields. */
function collectStrings(value, into, { skipKeys = PRESENTATION_KEYS, onlyKeys = null } = {}, key = "", active = onlyKeys === null) {
  if (typeof value === "string") {
    if (active) into.push(value);
  } else if (Array.isArray(value)) {
    value.forEach((item) => collectStrings(item, into, { skipKeys, onlyKeys }, key, active));
  } else if (value && typeof value === "object") {
    for (const [name, item] of Object.entries(value)) {
      if (skipKeys.has(name) && onlyKeys === null) continue;
      collectStrings(item, into, { skipKeys, onlyKeys }, name, active || (onlyKeys?.has(name) ?? false));
    }
  }
}

/** The lower-case words a token is made of when split at its separators and at case changes. */
export function splitSegments(text) {
  return text
    .replace(/([a-z0-9])([A-Z])/g, "$1 $2")
    .replace(/([A-Z]+)([A-Z][a-z])/g, "$1 $2")
    .split(/[^A-Za-z0-9]+/)
    .filter(Boolean)
    .map((part) => part.toLowerCase());
}

/** The words of one literal: its separated segments, and the whole value when it is one token. */
export function splitWords(text) {
  if (!TOKEN.test(text) || HASH.test(text)) return [];
  const words = new Set(splitSegments(text));
  words.add(text.toLowerCase());
  return [...words].filter((word) => word.length >= MIN_WORD_LENGTH);
}

/** The reverse-DNS prefix every plugin id shares, such as `md.numen.`. */
function getIdPrefix(ids) {
  const segments = ids.map((id) => id.split("."));
  const prefix = [];
  for (let i = 0; segments.every((one) => one.length > i && one[i] === segments[0][i]); i++) prefix.push(segments[0][i]);
  return prefix.length;
}

/**
 * The words the manifests of all plugins state, each with the plugin directories that state it: every
 * single-token value, and the segments of each plugin id that no other plugin id shares.
 */
export function loadManifestWords(manifestFiles = listManifestFiles(), readFile = (file) => fs.readFileSync(path.join(REPO_ROOT, file), "utf8")) {
  const words = new Map();
  const add = (word, plugin) => {
    if (word.length < MIN_WORD_LENGTH) return;
    if (!words.has(word)) words.set(word, new Set());
    words.get(word).add(plugin);
  };
  const hostWords = [];
  const manifests = manifestFiles.map((file) => ({ plugin: file.split("/")[1], json: JSON.parse(readFile(file)) }));
  const prefix = getIdPrefix(manifests.map(({ json }) => String(json.id)));
  const segmentOwners = new Map();
  for (const { plugin, json } of manifests) {
    for (const segment of new Set(String(json.id).split(".").slice(prefix))) {
      if (!segmentOwners.has(segment)) segmentOwners.set(segment, new Set());
      segmentOwners.get(segment).add(plugin);
    }
  }
  for (const { plugin, json } of manifests) {
    const strings = [];
    collectStrings(json, strings);
    const hostStrings = [];
    collectStrings(json, hostStrings, { onlyKeys: HOST_KEYS });
    hostWords.push(...hostStrings.flatMap(splitWords));
    for (const value of strings) {
      if (TOKEN.test(value) && !HASH.test(value)) add(value.toLowerCase(), plugin);
    }
    for (const segment of String(json.id).split(".").slice(prefix)) {
      if (segmentOwners.get(segment).size === 1) add(segment.toLowerCase(), plugin);
    }
  }
  for (const word of hostWords) words.delete(word);
  return words;
}

/** The words the SDKs and the protocol define, as literals or as names: shared data, not plugin names. */
export function loadDataWords(scanned = []) {
  const words = new Set();
  const isScanned = (root) => scanned.some((one) => one === root || one.startsWith(root + "/") || root.startsWith(one + "/"));
  const sources = [...DATA_SOURCES, ...HOST_VOCABULARY.filter((root) => !isScanned(root))];
  for (const file of listShippedFiles(sources)) {
    const source = readSource(file);
    if (!source) continue;
    for (const { value } of source.literals) splitWords(value).forEach((word) => words.add(word));
    for (const match of source.code.matchAll(/[A-Za-z_]\w*/g)) splitSegments(match[0]).forEach((word) => words.add(word));
  }
  return words;
}

/** The words of `manifestWords` that no data word defines: the names of plugins. */
export function subtractDataWords(manifestWords, data) {
  const dictionary = new Map();
  for (const [word, plugins] of manifestWords) {
    const isData = data.has(word) || splitSegments(word).every((part) => data.has(part));
    if (!isData) dictionary.set(word, plugins);
  }
  return dictionary;
}

/** The dictionary of plugin names: manifest words minus the shared data words. */
export function loadPluginDictionary(scanned = []) {
  return subtractDataWords(loadManifestWords(), loadDataWords(scanned));
}

const splitIdentifier = splitSegments;

/** The dictionary words an identifier is made of; a word of a plugin id is matched whole, never inside a longer word. */
export function matchIdentifier(name, dictionary) {
  const parts = splitIdentifier(name);
  const hits = [];
  for (const part of parts) if (dictionary.has(part)) hits.push(part);
  for (let size = 2; size <= parts.length; size++) {
    for (let from = 0; from + size <= parts.length; from++) {
      const joined = parts.slice(from, from + size).join("");
      if (dictionary.has(joined)) hits.push(joined);
    }
  }
  return hits;
}

/** The dictionary words a literal states: the whole value or one of its separated segments. */
export function matchLiteral(value, dictionary) {
  return splitWords(value).filter((word) => dictionary.has(word));
}

let tsModule;
function getTypeScript() {
  tsModule ??= require("typescript");
  return tsModule;
}

/** The script text of a single-file component and the line its script starts at. */
function readVueScripts(text) {
  const { parse } = require("@vue/compiler-sfc");
  const { descriptor } = parse(text);
  return [descriptor.script, descriptor.scriptSetup]
    .filter(Boolean)
    .map((block) => ({ text: block.content, lineOffset: block.loc.start.line - 1 }));
}

/**
 * Every dictionary word a TypeScript or Vue source states: string literals, template literal heads,
 * property keys, enum members, `case` labels and identifiers. Each hit has its line and kind.
 */
export function findWordsInScript(file, dictionary, text = fs.readFileSync(path.join(REPO_ROOT, file), "utf8")) {
  const ts = getTypeScript();
  const scripts = file.endsWith(".vue") ? readVueScripts(text) : [{ text, lineOffset: 0 }];
  const found = [];
  for (const script of scripts) {
    const kind = file.endsWith(".tsx") ? ts.ScriptKind.TSX : ts.ScriptKind.TS;
    const sourceFile = ts.createSourceFile(file, script.text, ts.ScriptTarget.Latest, true, kind);
    const report = (node, symbol, how) => {
      const line = sourceFile.getLineAndCharacterOfPosition(node.getStart(sourceFile)).line + 1 + script.lineOffset;
      found.push({ file, line, symbol: `${how} ${JSON.stringify(symbol)}` });
    };
    const visit = (node) => {
      if (ts.isStringLiteralLike(node) || ts.isTemplateHead(node)) {
        const insideImport = node.parent && (ts.isImportDeclaration(node.parent) || ts.isExportDeclaration(node.parent) || ts.isExternalModuleReference(node.parent) || ts.isImportTypeNode?.(node.parent));
        const isCall = node.parent && ts.isCallExpression(node.parent) && node.parent.expression.kind === ts.SyntaxKind.ImportKeyword;
        if (!insideImport && !isCall) {
          const how = node.parent && ts.isCaseClause(node.parent) ? "case" : "literal";
          for (const word of matchLiteral(node.text, dictionary)) report(node, word, how);
        }
      } else if (ts.isIdentifier(node) || ts.isPrivateIdentifier(node)) {
        const parent = node.parent;
        const isPropertyKey = parent && (ts.isPropertyAssignment(parent) || ts.isPropertySignature(parent)) && parent.name === node;
        const isEnumMember = parent && ts.isEnumMember(parent) && parent.name === node;
        const how = isEnumMember ? "enum member" : isPropertyKey ? "key" : "identifier";
        for (const word of matchIdentifier(node.text, dictionary)) report(node, word, how);
      }
      ts.forEachChild(node, visit);
    };
    visit(sourceFile);
  }
  return found;
}

/** Every dictionary word a Go source states: string literals and identifiers. */
export function findWordsInGo(file, dictionary, text = fs.readFileSync(path.join(REPO_ROOT, file), "utf8")) {
  const source = readGo(text);
  const found = [];
  for (const { line, value } of source.literals) {
    for (const word of matchLiteral(value, dictionary)) found.push({ file, line, symbol: `literal ${JSON.stringify(word)}` });
  }
  source.code.split("\n").forEach((text, index) => {
    for (const match of text.matchAll(/[A-Za-z_]\w*/g)) {
      for (const word of matchIdentifier(match[0], dictionary)) {
        found.push({ file, line: index + 1, symbol: `identifier ${JSON.stringify(word)}` });
      }
    }
  });
  return found;
}
