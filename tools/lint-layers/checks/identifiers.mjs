import path from "node:path";
import { spawnSync } from "node:child_process";
import { readPluginManifests } from "../../shared/manifests.mjs";
import { getIdPrefix } from "./dictionary.mjs";
import { REPO_ROOT, SCRIPT_EXTENSIONS, failOnFindings, listShippedFiles, readSource, requireMatches } from "./helpers.mjs";

/** The libraries and applications whose shipped sources state no identifier of a plugin. */
const HOST_ROOTS = ["libs", "apps"];

/** An identifier that carries a namespace separator: `numen.epub`, `reader:reader-epub`, `flashcards.review`. */
const NAMESPACED = /[.:]/;

/** The reverse-DNS namespace every plugin id shares, with its trailing dot, such as `md.numen.`. */
export function getIdNamespace(ids) {
  return ids[0].split(".").slice(0, getIdPrefix(ids)).join(".") + ".";
}

/**
 * The identifiers a manifest declares: the plugin id, and every service, tab kind, note type, reader,
 * content type, command, theme, status view and settings view whose identifier carries a namespace
 * separator. A bare word such as `note` is also a word of the host and identifies no plugin.
 *
 * @param {{ name: string, manifest: object }[]} manifests - each plugin folder with its parsed manifest
 * @returns {{ token: string, owner: string, isPluginId: boolean }[]} the identifiers only one plugin declares
 */
export function listPluginIdentifiers(manifests) {
  const declared = new Map();
  const add = (token, owner, isPluginId) => {
    if (typeof token !== "string" || (!isPluginId && !NAMESPACED.test(token))) return;
    if (!declared.has(token)) declared.set(token, { owners: new Set(), isPluginId });
    declared.get(token).owners.add(owner);
  };
  for (const { name, manifest } of manifests) {
    const contributes = manifest.contributes ?? {};
    add(manifest.id, name, true);
    (manifest.services ?? []).forEach((service) => add(service, name, false));
    (manifest.commands ?? []).forEach((command) => add(command.id, name, false));
    (manifest.settingsViews ?? []).forEach((view) => add(view.id, name, false));
    (contributes.tabKinds ?? []).forEach((tabKind) => add(tabKind.kind, name, false));
    (contributes.noteTypes ?? []).forEach((noteType) => add(noteType.type, name, false));
    (contributes.readers ?? []).forEach((reader) => add(reader.id, name, false));
    (contributes.contentTypes ?? []).forEach((contentType) => add(contentType.type, name, false));
    (contributes.themes ?? []).forEach((theme) => add(theme.id, name, false));
    (contributes.statusViews ?? []).forEach((view) => add(view.id, name, false));
  }
  return [...declared]
    .filter(([, { owners }]) => owners.size === 1)
    .map(([token, { owners, isPluginId }]) => ({ token, owner: [...owners][0], isPluginId }));
}

/**
 * The literals that state an identifier of a plugin. Each source is `{ file, plugin, literals }`: `plugin`
 * is the folder name for a file of a plugin and null for a file of the host, a literal being `{ line, value }`.
 * A host literal states a plugin id when it starts at the id namespace and equals an id, continues an id
 * with a segment, or is the start of an id. Every other literal states an identifier by being equal to it.
 * A plugin may state its own identifiers.
 */
export function findIdentifierLiterals(sources, identifiers) {
  const ids = identifiers.filter(({ isPluginId }) => isPluginId).map(({ token }) => token);
  const namespace = ids.length > 0 ? getIdNamespace(ids) : null;
  const found = [];
  for (const { file, plugin, literals } of sources) {
    for (const { line, value } of literals) {
      const exact = identifiers.find(({ token, owner }) => token === value && owner !== plugin);
      if (exact) {
        found.push({ file, line, symbol: `literal ${JSON.stringify(value)} states an identifier of ${exact.owner}` });
        continue;
      }
      if (plugin !== null || namespace === null) continue;
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

const GENERATED_PLUGIN_FILE = /\/frontend\/gen\/|\.pb\.go$|\.connect\.go$|(^|\/)eslint\.config\.js$/;

function readLiteralSources(files, getPlugin) {
  return files.flatMap((file) => {
    const source = readSource(file);
    return source ? [{ file, plugin: getPlugin(file), literals: source.literals }] : [];
  });
}

const isCodeFile = (file) => file.endsWith(".go") || SCRIPT_EXTENSIONS.has(path.extname(file));

/**
 * Identifiers appear only in the manifest that declares them: no shipped source of `libs` or `apps`, and no
 * source of another plugin, states a plugin id or a namespaced identifier a manifest declares.
 */
export function verifyPluginIdentifiersOnlyInManifests() {
  const identifiers = listPluginIdentifiers(readPluginManifests(path.join(REPO_ROOT, "plugins")));
  requireMatches("plugin ids read", identifiers.filter(({ isPluginId }) => isPluginId).length, 10);
  requireMatches("namespaced identifiers read", identifiers.filter(({ isPluginId }) => !isPluginId).length, 8);
  const hostFiles = listShippedFiles(HOST_ROOTS).filter(isCodeFile);
  requireMatches("host sources read for plugin identifiers", hostFiles.length, 1000);
  const listed = spawnSync("git", ["ls-files", "--", "plugins"], { cwd: REPO_ROOT, encoding: "utf8" });
  const pluginFiles = (listed.stdout || "")
    .split("\n")
    .filter((file) => file && isCodeFile(file) && !GENERATED_PLUGIN_FILE.test(file));
  requireMatches("plugin sources read for plugin identifiers", pluginFiles.length, 500);
  const sources = [...readLiteralSources(hostFiles, () => null), ...readLiteralSources(pluginFiles, (file) => file.split("/")[1])];
  failOnFindings("A source states an identifier that a plugin manifest declares", findIdentifierLiterals(sources, identifiers));
}

export const IDENTIFIER_VERIFIERS = [
  {
    id: "plugin-identifiers-only-in-manifests",
    scope: "libs",
    module: "boundary",
    name: "Plugin Identifiers Appear Only In Their Manifest (no source of libs, apps or another plugin states a plugin id, service, tab kind, command or reader identifier)",
    run: verifyPluginIdentifiersOnlyInManifests,
  },
];
