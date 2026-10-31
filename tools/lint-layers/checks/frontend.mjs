import { spawnSync } from "node:child_process";
import fs from "node:fs";
import path from "node:path";
import { readPluginManifests } from "../../shared/manifests.mjs";
import {
  MODULES_DIR,
  REPO_ROOT,
  exec,
  failOnFindings,
  findCrossings,
  findInCode,
  getImportSpecifiers,
  listShippedFiles,
  readSource,
  requireMatches,
} from "./helpers.mjs";

/** The names `libs/ui/core/src/index.ts` exports in braces. */
export function getUiExports() {
  const index = fs.readFileSync(path.join(MODULES_DIR, "libs/ui/core/src/index.ts"), "utf8");
  return Array.from(index.matchAll(/^export\s+(?:type\s+)?\{([^}]*)\}/gm)).flatMap((match) =>
    match[1]
      .split(",")
      .map((one) => one.trim().replace(/^type\s+/, "").split(/\s+as\s+/).pop())
      .filter(Boolean),
  );
}

/** The folders of a UI library under `src` that hold views, besides `src` itself. */
const UI_VIEW_ROOTS = ["", "features", "screens"];

/**
 * The views a plugin owns, read from the frontend folders of the plugins: the folders directly under
 * `frontend` and the components (`.vue` files, stories and tests apart) of each plugin. A folder or a
 * component of two or more plugins identifies no plugin and is not listed.
 *
 * @param {string} pluginsDir - the folder that holds the plugins
 * @returns {{ dirs: Map<string, string>, names: Map<string, string> }} each owned folder name and
 *   component name with the plugin that owns it
 */
export function loadPluginViews(pluginsDir = path.join(MODULES_DIR, "plugins")) {
  const dirOwners = new Map();
  const nameOwners = new Map();
  const add = (owners, key, plugin) => {
    if (!owners.has(key)) owners.set(key, new Set());
    owners.get(key).add(plugin);
  };
  for (const { name: plugin } of readPluginManifests(pluginsDir)) {
    const frontend = path.join(pluginsDir, plugin, "frontend");
    if (!fs.existsSync(frontend)) continue;
    for (const entry of fs.readdirSync(frontend, { recursive: true, withFileTypes: true })) {
      if (entry.isDirectory() && path.relative(frontend, path.join(entry.parentPath, entry.name)).split(path.sep).length === 1) {
        add(dirOwners, entry.name, plugin);
      }
      if (entry.isFile() && /^[A-Z]\w*\.vue$/.test(entry.name)) add(nameOwners, entry.name.replace(/\.vue$/, ""), plugin);
    }
  }
  const owned = (owners) => new Map([...owners].filter(([, plugins]) => plugins.size === 1).map(([key, plugins]) => [key, [...plugins][0]]));
  return { dirs: owned(dirOwners), names: owned(nameOwners) };
}

/** The views of `views` that the UI library at `uiSrc` holds as a folder or exports by `exported`. */
export function findPluginViewsInUi(views, exported, uiSrc = path.join(MODULES_DIR, "libs/ui/core/src")) {
  const found = [];
  for (const [dir, plugin] of views.dirs) {
    for (const root of UI_VIEW_ROOTS) {
      const relative = path.join(root, dir);
      if (fs.existsSync(path.join(uiSrc, relative))) found.push(`${relative} of the UI library belongs to ${plugin}`);
    }
  }
  for (const [name, plugin] of views.names) {
    if (exported.has(name)) found.push(`@numen/ui exports ${name}, which belongs to ${plugin}`);
  }
  return found;
}

/** `@numen/ui` holds and exports none of the views a plugin owns. */
export function verifyUiHoldsNoPluginView() {
  const views = loadPluginViews();
  requireMatches("view folders owned by one plugin", views.dirs.size, 5);
  requireMatches("view components owned by one plugin", views.names.size, 20);
  const found = findPluginViewsInUi(views, new Set(getUiExports()));
  if (found.length > 0) {
    throw new Error("The UI library holds views plugins own:\n" + found.join("\n"));
  }
}

/** The window chrome `@numen/shell` holds: its directories under `src/` and the names it exports. */
const SHELL_PARTS = {
  dirs: ["features/workspace", "features/palette", "features/status-bar", "features/notices", "features/welcome", "screens"],
  names: ["WorkspaceLayout", "Palette", "StatusBar", "Notices", "createNoticeQueue", "WelcomePage", "getVaultForKey", "VaultRow", "WelcomeAction"],
};

/** `@numen/ui` holds and exports none of the window chrome `@numen/shell` owns. */
export function verifyUiHoldsNoShellPart() {
  const exported = new Set(getUiExports());
  const found = [
    ...SHELL_PARTS.dirs
      .filter((dir) => fs.existsSync(path.join(MODULES_DIR, "libs/ui/core/src", dir)))
      .map((dir) => `libs/ui/core/src/${dir} belongs to libs/ui/shell`),
    ...SHELL_PARTS.names.filter((name) => exported.has(name)).map((name) => `@numen/ui exports ${name}, which belongs to @numen/shell`),
  ];
  if (found.length > 0) {
    throw new Error("The UI library holds the window chrome:\n" + found.join("\n"));
  }
}

export function verifyUiDomainIsolation() {
  const leakCheck = spawnSync("git", ["grep", "-E", "(@numen/(protocol|wire|shell)|apps|libs/ui/shell)", "libs/ui/core/src/"], { cwd: REPO_ROOT });
  const leaks = (leakCheck.stdout ? leakCheck.stdout.toString() : "").trim();
  if (leaks) {
    throw new Error("UI library imports domain, wire, protocol, the shell or apps:\n" + leaks);
  }
  exec("npm run lint", path.join(MODULES_DIR, "libs/ui/core"));
}

export function verifyShellDomainIsolation() {
  const leakCheck = spawnSync("git", ["grep", "-E", "(@numen/(protocol|wire)|apps)", "libs/ui/shell/src/"], { cwd: REPO_ROOT });
  const leaks = (leakCheck.stdout ? leakCheck.stdout.toString() : "").trim();
  if (leaks) {
    throw new Error("Shell library imports domain, wire, protocol or apps:\n" + leaks);
  }
  exec("npm run lint", path.join(MODULES_DIR, "libs/ui/shell"));
}

/**
 * What `libs/plugin-host-ts/src` may import besides its own files.
 */
const PLUGIN_HOST_IMPORTS = [
  { files: /./, specifiers: /^(vue|@numen\/plugin-sdk|@numen\/core-client)(\/|$)/ },
  { files: /\.test\.ts$/, specifiers: /^vitest$/ },
  { files: /^vite\.ts$/, specifiers: /^vite$/ },
];

/** The imports of one file of `libs/plugin-host-ts/src` that are neither relative nor in PLUGIN_HOST_IMPORTS. */
export function findDisallowedHostImports(file, specifiers) {
  return specifiers
    .filter((specifier) => !specifier.startsWith("."))
    .filter((specifier) => !PLUGIN_HOST_IMPORTS.some((rule) => rule.files.test(path.basename(file)) && rule.specifiers.test(specifier)))
    .map((specifier) => `libs/plugin-host-ts/src/${file}: ${specifier}`);
}

/** Every import of `libs/plugin-host-ts/src` that is neither relative nor in PLUGIN_HOST_IMPORTS, and the number of imports read. */
function findPluginHostImports() {
  const src = path.join(MODULES_DIR, "libs/plugin-host-ts/src");
  const found = [];
  let read = 0;
  for (const file of fs.readdirSync(src, { recursive: true })) {
    if (!/\.(ts|vue)$/.test(file)) continue;
    const text = fs.readFileSync(path.join(src, file), "utf8");
    const specifiers = getImportSpecifiers(text);
    read += specifiers.length;
    found.push(...findDisallowedHostImports(file, specifiers));
  }
  return { found, read };
}

/** The plugin loader knows Vue, the plugin SDK and the core client, and nothing of the UI, the shell, the schema, the wire or an application. */
export function verifyPluginHostIsolation() {
  const { found, read } = findPluginHostImports();
  requireMatches("imports of the plugin host", read, 20);
  if (found.length > 0) {
    throw new Error("The plugin host imports beyond vue, @numen/plugin-sdk and @numen/core-client:\n" + found.join("\n"));
  }
  exec("npm run lint", path.join(MODULES_DIR, "libs/plugin-host-ts"));
}

/** The Markdown editor of `@numen/ui`, which the notes plugin owns, and the wikilink helpers the core replaces. */
const UI_EDITOR = {
  dirs: ["features/editor"],
  modules: ["./features/editor", "./shared/lib/text/address"],
  names: ["Editor", "createTimeGutter", "isNoteAddress", "getWikilinks"],
};

/** The names of the note link format: its address helpers and any name holding `wikilink`. */
const NOTE_LINK_NAME = /\b(isNoteAddress|getWikilinks|parseAddress|writeAddress|[A-Za-z_$]*[Ww]ikilink[A-Za-z0-9_$]*)\b/g;

/** The schemes a note address is written with. */
const NOTE_SCHEMES = ["name://", "note://"];

/** Every name of the note link format and every note address literal in the shipped code of `libs/ui/core/src`. */
function findUiNoteLinks() {
  const found = [];
  for (const file of listShippedFiles(["libs/ui/core/src"])) {
    const source = readSource(file);
    if (!source) continue;
    for (const hit of findInCode(source.code, NOTE_LINK_NAME)) found.push({ file, ...hit });
    for (const { line, value } of source.literals) {
      if (NOTE_SCHEMES.some((scheme) => value.startsWith(scheme))) found.push({ file, line, symbol: JSON.stringify(value) });
    }
  }
  return found;
}

/** Every editor directory `libs/ui/core/src` holds, every editor or wikilink name `src/index.ts` exports, and every note link name it defines or uses. */
function findUiEditor() {
  const found = UI_EDITOR.dirs
    .filter((dir) => fs.existsSync(path.join(REPO_ROOT, "libs/ui/core/src", dir)))
    .map((dir) => ({ file: `libs/ui/core/src/${dir}`, line: 0, symbol: "directory" }));
  found.push(...findUiNoteLinks());
  const file = "libs/ui/core/src/index.ts";
  const text = fs.readFileSync(path.join(REPO_ROOT, file), "utf8");
  for (const match of text.matchAll(/^export\s+(?:type\s+)?\{([^}]*)\}\s*from\s*(['"])([^'"]+)\2/gm)) {
    const line = text.slice(0, match.index).split("\n").length;
    const fromEditor = UI_EDITOR.modules.some((one) => match[3] === one || match[3].startsWith(one + "/"));
    for (const name of match[1].split(",").map((one) => one.trim().replace(/^type\s+/, "").split(/\s+as\s+/).pop())) {
      if (name && (fromEditor || UI_EDITOR.names.includes(name))) found.push({ file, line, symbol: `export ${name}` });
    }
  }
  return found;
}

/** `@numen/ui` holds and exports no editor and no wikilink helper. */
export function verifyUiExportsNoEditor() {
  failOnFindings("The UI library holds the editor or wikilink helpers", findUiEditor());
}

/** An application knows no plugin: it compiles, imports, aliases and reads nothing under `plugins/`. */
export function verifyAppsReachNoPlugin() {
  const crossings = findCrossings("apps", "plugins", [
    "@plugins\\b",
    "github\\.com/jiva-studio/numen/plugins\\b",
  ]);
  if (crossings.length > 0) {
    throw new Error("An application reaches into plugins/:\n" + crossings.join("\n"));
  }
}

/** Segment direction (ui -> model -> api -> lib) and no @x cross-entity imports. */
function findFrontendSegmentAndCrossEntityViolations() {
  const found = [];
  const segRank = { lib: 0, api: 1, model: 2, ui: 3 };
  for (const file of listShippedFiles(["libs/ui/core/src", "libs/ui/shell/src", "apps/desktop/editor/src", "apps/desktop/flashcards/src", "apps/mobile/src", "plugins"])) {
    if (!/\.(ts|vue)$/.test(file) || /\/frontend\/gen\//.test(file)) continue;
    const parts = file.split("/");
    const segIdx = parts.findLastIndex((p) => Object.prototype.hasOwnProperty.call(segRank, p));
    const source = readSource(file);
    if (!source) continue;
    for (const { line, value } of source.literals) {
      if (value.includes("/@x/")) {
        found.push({ file, line, symbol: `cross-entity @x import: ${JSON.stringify(value)}` });
      }
      if (segIdx !== -1 && value.startsWith(".")) {
        const fromSeg = parts[segIdx];
        const resolved = path.posix.normalize(path.posix.join(path.posix.dirname(file), value));
        const resParts = resolved.split("/");
        const toSeg = resParts[segIdx];
        if (
          resParts.slice(0, segIdx).join("/") === parts.slice(0, segIdx).join("/") &&
          Object.prototype.hasOwnProperty.call(segRank, toSeg) &&
          segRank[toSeg] > segRank[fromSeg]
        ) {
          found.push({ file, line, symbol: `upward segment import ${fromSeg} -> ${toSeg}: ${JSON.stringify(value)}` });
        }
      }
    }
  }
  return found;
}

export function verifyFrontendSegmentsAndEntities() {
  failOnFindings("Frontend upward segment imports (lib/api/model -> ui) or @x cross-entity imports", findFrontendSegmentAndCrossEntityViolations());
}

/** Vue <style> raw colors/fallbacks and <template> array transforms. */
function findVueStyleAndTemplateViolations() {
  const found = [];
  for (const file of listShippedFiles(["libs/ui/core/src", "libs/ui/shell/src", "apps/desktop/editor/src", "apps/desktop/flashcards/src", "apps/mobile/src", "plugins"])) {
    if (!file.endsWith(".vue") || /\/frontend\/gen\//.test(file)) continue;
    const raw = fs.readFileSync(path.join(REPO_ROOT, file), "utf8");
    const tplMatch = raw.match(/<template[^>]*>([\s\S]*?)<\/template>/);
    if (tplMatch) {
      const baseLine = raw.slice(0, tplMatch.index).split("\n").length;
      tplMatch[1].split("\n").forEach((line, idx) => {
        const m = line.match(/\.(filter|map|reduce|sort|slice)\s*\(|\bnew\s+Date\s*\(|\bJSON\.(?:stringify|parse)\s*\(/);
        if (m) found.push({ file, line: baseLine + idx, symbol: `<template> inline computation ${m[0].trim()}` });
      });
    }
    const styleMatch = raw.match(/<style[^>]*>([\s\S]*?)<\/style>/);
    if (styleMatch && !/theme/i.test(file)) {
      const baseLine = raw.slice(0, styleMatch.index).split("\n").length;
      styleMatch[1].split("\n").forEach((line, idx) => {
        if (/^\s*\/\//.test(line)) return;
        if (/#[0-9a-fA-F]{3,8}\b|\brgba?\s*\(|\bhsla?\s*\(/.test(line)) {
          found.push({ file, line: baseLine + idx, symbol: `<style> raw color literal: ${line.trim()}` });
        }
        if (/var\(\s*--[\w-]+\s*,/.test(line)) {
          found.push({ file, line: baseLine + idx, symbol: `<style> token fallback var(--token, fallback): ${line.trim()}` });
        }
        if (/^\s*--numen-[\w-]+\s*:/.test(line)) {
          found.push({ file, line: baseLine + idx, symbol: `<style> local --numen-* token override: ${line.trim()}` });
        }
      });
    }
  }
  return found;
}

export function verifyVueStyleAndTemplatePurity() {
  failOnFindings("Vue <style> token bypass or <template> inline transform", findVueStyleAndTemplateViolations());
}

/** Every TypeScript package with a tsconfig.json has an eslint.config.js. */
function findTsPackagesWithoutEslint() {
  const candidates = [
    "libs/ui/core",
    "libs/ui/shell",
    "libs/plugin-sdk-ts",
    "libs/core-client",
    "libs/plugin-host-ts",
    "apps/desktop/editor",
    "apps/desktop/flashcards",
    "apps/mobile",
  ];
  return candidates
    .filter((dir) => fs.existsSync(path.join(REPO_ROOT, dir, "tsconfig.json")) && !fs.existsSync(path.join(REPO_ROOT, dir, "eslint.config.js")))
    .map((dir) => ({ file: `${dir}/tsconfig.json`, line: 1, symbol: "missing eslint.config.js" }));
}

export function verifyAllTsPackagesHaveEslint() {
  failOnFindings("TypeScript package missing eslint.config.js", findTsPackagesWithoutEslint());
}

export const FRONTEND_VERIFIERS = [
  {
    id: "all-ts-packages-eslint",
    scope: "libs",
    module: "sdk",
    name: "Every TypeScript Package Has ESLint Configured",
    run: verifyAllTsPackagesHaveEslint,
  },
  {
    id: "ui-isolation",
    scope: "libs",
    module: "ui",
    name: "UI Library Domain & Wire Isolation (0 domain, 0 protocol, 0 wire, 0 shell, 0 apps)",
    run: verifyUiDomainIsolation,
  },
  {
    id: "ui-holds-no-plugin-view",
    scope: "libs",
    module: "ui",
    name: "UI Library Holds No Plugin View (no folder or export named after a folder or component of one plugin)",
    run: verifyUiHoldsNoPluginView,
  },
  {
    id: "ui-holds-no-shell-part",
    scope: "libs",
    module: "ui",
    name: "UI Library Holds No Window Chrome (workspace, palette, status bar, notices, welcome page)",
    run: verifyUiHoldsNoShellPart,
  },
  {
    id: "ui-exports-no-editor",
    scope: "libs",
    module: "ui",
    name: "UI Library Exports No Editor or Wikilink Helper (features/editor, isNoteAddress, getWikilinks)",
    run: verifyUiExportsNoEditor,
  },
  {
    id: "shell-isolation",
    scope: "libs",
    module: "shell",
    name: "Shell Library Domain & Wire Isolation (0 domain, 0 protocol, 0 wire, 0 apps)",
    run: verifyShellDomainIsolation,
  },
  {
    id: "plugin-host-isolation",
    scope: "libs",
    module: "host",
    name: "Plugin Host Isolation (imports only vue, @numen/plugin-sdk and @numen/core-client)",
    run: verifyPluginHostIsolation,
  },
  {
    id: "frontend-segments-and-entities",
    scope: "apps",
    module: "desktop",
    name: "Frontend Segment Direction (ui -> model -> api -> lib) & No @x Entity Cross-Imports",
    run: verifyFrontendSegmentsAndEntities,
  },
  {
    id: "vue-style-and-template",
    scope: "apps",
    module: "desktop",
    name: "Vue <style> Tokens & <template> Purity",
    run: verifyVueStyleAndTemplatePurity,
  },
  {
    id: "apps-reach-no-plugin",
    scope: "apps",
    module: "apps",
    name: "Applications Reach No Plugin (no path, alias, glob or import into plugins/)",
    run: verifyAppsReachNoPlugin,
  },
  {
    id: "desktop-editor-fsd",
    scope: "apps",
    module: "desktop",
    name: "Desktop Editor FSD Layer Hierarchy (shared -> entities -> features -> widgets -> pages -> app)",
    run: () => {
      exec("npm run lint", path.join(MODULES_DIR, "apps/desktop/editor"));
    },
  },
  {
    id: "desktop-flashcards-fsd",
    scope: "apps",
    module: "desktop",
    name: "Desktop Flashcards FSD Layer Hierarchy",
    run: () => {
      exec("npm run lint", path.join(MODULES_DIR, "apps/desktop/flashcards"));
    },
  },
];
