import fs from "node:fs";
import path from "node:path";
import { readPluginManifests } from "../../shared/manifests.mjs";
import { findWordsInGo, findWordsInScript, getIdPrefix, listManifestFiles, loadImplementationDictionary, loadPluginDictionary, matchIdentifier } from "./dictionary.mjs";
import { REPO_ROOT, SCRIPT_EXTENSIONS, failOnFindings, isBoundaryRule, listShippedFiles, readSource, requireMatches } from "./helpers.mjs";

/** The libraries and applications whose frontend sources know no plugin. */
const FRONTEND_NAME_FREE_ROOTS = ["apps", "libs/ui/shell", "libs/core-client", "libs/plugin-host-ts", "libs/plugin-manager", "libs/ui/core"];

/** The Go library whose sources know no plugin. */
const GO_NAME_FREE_ROOTS = ["libs/notes"];

/** The sources of a frontend package: TypeScript and Vue under a `src` directory. */
const FRONTEND_SOURCE = /(^|\/)src\/.*\.(ts|tsx|vue)$/;

/** Every tracked, shipped frontend source under the roots. */
export function listFrontendSources(roots = FRONTEND_NAME_FREE_ROOTS) {
  return listShippedFiles(roots).filter((file) => FRONTEND_SOURCE.test(file));
}

/** Every tracked, shipped Go source under the roots. */
export function listGoSources(roots = GO_NAME_FREE_ROOTS) {
  return listShippedFiles(roots).filter((file) => file.endsWith(".go"));
}

/**
 * The plugin words the sources state. Each source is `{ file, text }`; `kind` is `"script"` or `"go"`.
 * A word is a plugin word when a manifest states it and neither an SDK nor the protocol defines it.
 */
export function findPluginWords(sources, kind, dictionary) {
  const find = kind === "go" ? findWordsInGo : findWordsInScript;
  return sources.flatMap(({ file, text }) => find(file, dictionary, text));
}

function readSources(files) {
  return files.map((file) => ({ file, text: fs.readFileSync(path.join(REPO_ROOT, file), "utf8") }));
}

/**
 * The applications, the shell, the plugin host and the UI library name no plugin: no literal, object key,
 * enum member, `case` label or identifier equals a word a plugin manifest states.
 */
export function verifyWindowNamesNoPluginKind() {
  const dictionary = loadPluginDictionary(FRONTEND_NAME_FREE_ROOTS);
  requireMatches("plugin manifests read", listManifestFiles().length, 2);
  requireMatches("plugin words in the dictionary", dictionary.size, 10);
  const files = listFrontendSources();
  requireMatches("frontend sources read", files.length, 100);
  failOnFindings("The frontend names plugin words", findPluginWords(readSources(files), "script", dictionary));
}

/** The notes library names no plugin: no literal or identifier equals a word a plugin manifest states. */
export function verifyNotesNamesNoProducer() {
  const dictionary = loadPluginDictionary(GO_NAME_FREE_ROOTS);
  requireMatches("plugin words in the dictionary", dictionary.size, 10);
  const files = listGoSources();
  requireMatches("notes sources read", files.length, 10);
  failOnFindings("The notes library names plugin words", findPluginWords(readSources(files), "go", dictionary));
}

/** The plugin SDKs, whose sources, comments and documents know no implementation. */
const SDK_ROOTS = ["libs/plugin-sdk-go", "libs/plugin-sdk-ts"];

/** The tracked files of the SDKs that hold text a person writes: code, configuration and documents. */
const SDK_TEXT_FILE = /\.(go|ts|mts|vue|js|mjs|json|ya?ml|md)$/;

/** Files that hold no prose or code of the SDK: dependency locks and module sums. */
const SDK_GENERATED_FILE = /(^|\/)(package-lock\.json|go\.sum|go\.mod)$/;

/** The path of a module of the repository, which names a directory and states no word of its own. */
const MODULE_PATH = /github\.com\/jiva-studio\/numen\/[\w./-]*/g;

/**
 * The implementation words the lines state. Each source is `{ file, text }`. A word is read as a whole
 * identifier or as consecutive parts of one, so `OnnxRuntime` and `onnxruntime_purego` state `onnxruntime`.
 */
export function findImplementationWords(sources, dictionary) {
  const found = [];
  for (const { file, text } of sources) {
    text.split("\n").forEach((line, index) => {
      for (const token of line.replace(MODULE_PATH, "").matchAll(/[A-Za-z][A-Za-z0-9_]*/g)) {
        for (const word of new Set(matchIdentifier(token[0], dictionary))) {
          found.push({ file, line: index + 1, symbol: `implementation word ${JSON.stringify(word)}` });
        }
      }
    });
  }
  return found;
}

/**
 * The Go and TypeScript plugin SDKs, with their comments and documents, state no word a plugin manifest
 * uses to declare an implementation: a runtime, a dependency, a model format or the identity of an
 * engine or an agent program. The protocol and the notes library define the words that remain.
 */
export function verifySdkNamesNoImplementation() {
  const dictionary = loadImplementationDictionary();
  requireMatches("implementation words in the dictionary", dictionary.size, 10);
  const files = listShippedFiles(SDK_ROOTS).filter((file) => SDK_TEXT_FILE.test(file) && !SDK_GENERATED_FILE.test(file) && !isBoundaryRule(file));
  requireMatches("SDK files read", files.length, 60);
  failOnFindings("The plugin SDKs state implementation words", findImplementationWords(readSources(files), dictionary));
}

/** The libraries and applications whose production code names no plugin id. */
const PLUGIN_ID_FREE_ROOTS = ["libs/core", "apps", "libs/ui"];

/** The reverse-DNS namespace every plugin id shares, with its trailing dot, such as `md.numen.`. */
export function getIdNamespace(ids) {
  return ids[0].split(".").slice(0, getIdPrefix(ids)).join(".") + ".";
}

/**
 * The literals that state a plugin id, or the start of one: a token that begins at the namespace of
 * the ids and equals an id, continues an id with a segment, or is the start of an id. Each source is
 * `{ file, literals }`, a literal being `{ line, value }`.
 */
export function findPluginIdLiterals(sources, ids) {
  const namespace = getIdNamespace(ids);
  const found = [];
  for (const { file, literals } of sources) {
    for (const { line, value } of literals) {
      for (let at = value.indexOf(namespace); at >= 0; at = value.indexOf(namespace, at + 1)) {
        const token = /^[\w.-]*/.exec(value.slice(at))[0];
        if (ids.some((id) => id.startsWith(token) || token === id || token.startsWith(id + "."))) {
          found.push({ file, line, symbol: `literal ${JSON.stringify(value)} states the plugin id ${JSON.stringify(token)}` });
          break;
        }
      }
    }
  }
  return found;
}

function readLiterals(files) {
  return files.flatMap((file) => {
    const source = readSource(file);
    return source ? [{ file, literals: source.literals }] : [];
  });
}

/** The core, the applications and the UI library name no plugin id in their production code. */
export function verifyHostNamesNoPluginId() {
  const ids = readPluginManifests(path.join(REPO_ROOT, "plugins")).map(({ manifest }) => manifest.id);
  requireMatches("plugin ids read", ids.length, 10);
  const files = listShippedFiles(PLUGIN_ID_FREE_ROOTS).filter((file) => file.endsWith(".go") || SCRIPT_EXTENSIONS.has(path.extname(file)));
  requireMatches("production sources read for plugin ids", files.length, 500);
  failOnFindings("Production code names a plugin id", findPluginIdLiterals(readLiterals(files), ids));
}

export const VOCABULARY_VERIFIERS = [
  {
    id: "window-names-no-plugin-kind",
    scope: "apps",
    module: "editor",
    name: "Applications, Shell, Plugin Host and UI Library Name No Plugin Word (words of the plugin manifests)",
    run: verifyWindowNamesNoPluginKind,
  },
  {
    id: "notes-names-no-producer",
    scope: "libs",
    module: "notes",
    name: "Notes Library Names No Plugin Word (words of the plugin manifests)",
    run: verifyNotesNamesNoProducer,
  },
  {
    id: "sdk-names-no-implementation",
    scope: "libs",
    module: "sdk",
    name: "Plugin SDKs Name No Implementation (runtimes, dependencies, model formats and engine identities of the plugin manifests)",
    run: verifySdkNamesNoImplementation,
  },
  {
    id: "host-names-no-plugin-id",
    scope: "libs",
    module: "core",
    name: "Core, Applications and UI Library Name No Plugin Id (no literal states an id of a plugin manifest)",
    run: verifyHostNamesNoPluginId,
  },
];
