import fs from "node:fs";
import path from "node:path";
import { listPluginNames, parseManifest, MANIFEST_FILE } from "../../shared/manifests.mjs";
import { REPO_ROOT, listShippedFiles, readSource } from "./helpers.mjs";

const MIN_WORD_LENGTH = 3;
const TOKEN = /^[A-Za-z][\w.:*-]*$/;
const HASH = /^[0-9a-f]{32,}$/;

/** The manifest of each plugin that holds one. */
export function listManifestFiles() {
  return listPluginNames(path.join(REPO_ROOT, "plugins")).map((name) => `plugins/${name}/${MANIFEST_FILE}`);
}

/** The manifest fields that hold presentation or configuration values, which name no plugin. */
const PRESENTATION_KEYS = new Set(["icon", "key", "title", "description", "default", "label", "unit", "placeholder", "name"]);

/**
 * Whether a manifest array lists files: its objects hold a `file` path. The entries of such an array
 * (the themes of a plugin are one) are assets with their own ids and titles, which name no plugin.
 */
export function isFileList(value) {
  return Array.isArray(value) && value.some((item) => item && typeof item === "object" && typeof item.file === "string");
}

/** The manifest fields that name the windows a plugin runs in; the host defines them. */
const HOST_KEYS = new Set(["targets"]);

/** Every string value of a parsed JSON document outside the fields in `skipKeys`; with `onlyKeys`, only the values under those fields. */
function collectStrings(value, into, { skipKeys = PRESENTATION_KEYS, onlyKeys = null } = {}, key = "", active = onlyKeys === null) {
  if (typeof value === "string") {
    if (active) into.push(value);
  } else if (Array.isArray(value)) {
    if (onlyKeys === null && isFileList(value)) return;
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

/** The number of leading segments every plugin id shares, such as the two of `md.numen.`. */
export function getIdPrefix(ids) {
  const segments = ids.map((id) => id.split("."));
  const prefix = [];
  for (let i = 0; segments.every((one) => one.length > i && one[i] === segments[0][i]); i++) prefix.push(segments[0][i]);
  return prefix.length;
}

/**
 * The words the manifests of all plugins state, each with the plugin directory that states it: every
 * single-token value, and the segments of each plugin id that no other plugin id shares. A word that
 * the manifests of two or more plugins state identifies no plugin and is left out.
 */
export function loadManifestWords(manifestFiles = listManifestFiles(), readFile = (file) => fs.readFileSync(path.join(REPO_ROOT, file), "utf8")) {
  const words = new Map();
  const add = (word, plugin) => {
    if (word.length < MIN_WORD_LENGTH) return;
    if (!words.has(word)) words.set(word, new Set());
    words.get(word).add(plugin);
  };
  const hostWords = [];
  const manifests = manifestFiles.map((file) => ({ plugin: file.split("/")[1], json: parseManifest(readFile(file), file).document }));
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
  for (const [word, plugins] of words) if (plugins.size > 1) words.delete(word);
  return words;
}

/** The words the files under `roots` define, as literals or as names. */
export function loadWordsOf(roots) {
  const words = new Set();
  for (const file of listShippedFiles(roots)) {
    const source = readSource(file);
    if (!source) continue;
    for (const { value } of source.literals) splitWords(value).forEach((word) => words.add(word));
    for (const match of source.code.matchAll(/[A-Za-z_]\w*/g)) splitSegments(match[0]).forEach((word) => words.add(word));
  }
  return words;
}

/** The libraries that define the contracts between the host and a plugin; their words are not implementation names. */
const CONTRACT_SOURCES = ["libs/protocol", "libs/notes"];

/** The compact form of an identifier: lower case, without separators, so that `pp-doclayout` and `PPDocLayout` agree. */
const compact = (text) => text.toLowerCase().replace(/[^a-z0-9]/g, "");

/** The file extension of a file name, without the dot. */
const getExtension = (fileName) => (fileName.includes(".") ? fileName.slice(fileName.lastIndexOf(".") + 1) : "");

/**
 * The words that name how a plugin is implemented, from its manifests: the id and the model file format
 * of each dependency, runtimes included, and, for a plugin that runs a backend and declares capabilities
 * or dependencies, each segment of its id that no other plugin id has. A dependency or a model format
 * that several plugins declare is kept, because the plugins that use it share it.
 */
export function loadImplementationWords(manifestFiles = listManifestFiles(), readFile = (file) => fs.readFileSync(path.join(REPO_ROOT, file), "utf8")) {
  const words = new Map();
  const add = (word, plugin) => {
    if (word.length < MIN_WORD_LENGTH) return;
    if (!words.has(word)) words.set(word, new Set());
    words.get(word).add(plugin);
  };
  const manifests = manifestFiles.map((file) => ({ plugin: file.split("/")[1], json: parseManifest(readFile(file), file).document }));
  const prefix = getIdPrefix(manifests.map(({ json }) => String(json.id)));
  const segmentOwners = new Map();
  for (const { plugin, json } of manifests) {
    for (const segment of new Set(String(json.id).split(".").slice(prefix))) {
      if (!segmentOwners.has(segment)) segmentOwners.set(segment, new Set());
      segmentOwners.get(segment).add(plugin);
    }
  }
  for (const { plugin, json } of manifests) {
    const hasEngine = Boolean(json.backend) && ((json.capabilities ?? []).length > 0 || (json.dependencies ?? []).length > 0);
    for (const segment of hasEngine ? String(json.id).split(".").slice(prefix) : []) {
      if (segmentOwners.get(segment).size === 1) add(segment.toLowerCase(), plugin);
    }
    for (const dependency of json.dependencies ?? []) {
      add(compact(String(dependency.id ?? "")), plugin);
      if (dependency.kind === "model") add(compact(getExtension(String(dependency.metadata?.fileName ?? ""))), plugin);
    }
  }
  return words;
}

/** The implementation words that the protocol and the notes library do not define. */
export function loadImplementationDictionary() {
  return subtractDataWords(loadImplementationWords(), loadWordsOf(CONTRACT_SOURCES));
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
