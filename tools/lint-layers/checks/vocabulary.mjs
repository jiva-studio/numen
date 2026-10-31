import fs from "node:fs";
import path from "node:path";
import { loadImplementationDictionary, matchIdentifier } from "./dictionary.mjs";
import { REPO_ROOT, failOnFindings, isBoundaryRule, listShippedFiles, requireMatches } from "./helpers.mjs";

function readSources(files) {
  return files.map((file) => ({ file, text: fs.readFileSync(path.join(REPO_ROOT, file), "utf8") }));
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

export const VOCABULARY_VERIFIERS = [
  {
    id: "sdk-names-no-implementation",
    scope: "libs",
    module: "sdk",
    name: "Plugin SDKs Name No Implementation (runtimes, dependencies, model formats and engine identities of the plugin manifests)",
    run: verifySdkNamesNoImplementation,
  },
];
