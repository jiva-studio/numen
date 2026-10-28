import { spawnSync } from "node:child_process";
import fs from "node:fs";
import path from "node:path";
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

/**
 * The views plugins own, which `@numen/ui` neither holds nor exports.
 */
const PLUGIN_VIEWS = [
  {
    plugin: "views-agent-chat",
    dirs: ["screens/agent", "features/thread", "features/message-composer"],
    names: ["Agent", "Thread", "MessageComposer", "useConversation", "Conversation", "Turn", "AgentModelOption", "AgentPort", "AgentStep"],
  },
];

/** `@numen/ui` holds and exports none of the views a plugin owns. */
export function verifyUiHoldsNoPluginView() {
  const exported = new Set(getUiExports());
  const found = [];
  for (const { plugin, dirs, names } of PLUGIN_VIEWS) {
    for (const dir of dirs) {
      if (fs.existsSync(path.join(MODULES_DIR, "libs/ui/core/src", dir))) {
        found.push(`libs/ui/core/src/${dir} belongs to ${plugin}`);
      }
    }
    for (const name of names) {
      if (exported.has(name)) found.push(`@numen/ui exports ${name}, which belongs to ${plugin}`);
    }
  }
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
  { files: /./, specifiers: /^(vue|@numen\/plugin-sdk)(\/|$)/ },
  { files: /\.test\.ts$/, specifiers: /^vitest$/ },
  { files: /^vite\.ts$/, specifiers: /^vite$/ },
];

/** Every import of `libs/plugin-host-ts/src` that is neither relative nor in PLUGIN_HOST_IMPORTS. */
function findPluginHostImports() {
  const src = path.join(MODULES_DIR, "libs/plugin-host-ts/src");
  const found = [];
  for (const file of fs.readdirSync(src, { recursive: true })) {
    if (!/\.(ts|vue)$/.test(file)) continue;
    const text = fs.readFileSync(path.join(src, file), "utf8");
    for (const specifier of getImportSpecifiers(text)) {
      if (specifier.startsWith(".")) continue;
      const allowed = PLUGIN_HOST_IMPORTS.some(
        (rule) => rule.files.test(path.basename(file)) && rule.specifiers.test(specifier),
      );
      if (!allowed) found.push(`libs/plugin-host-ts/src/${file}: ${specifier}`);
    }
  }
  return found;
}

/** The plugin loader knows Vue and the plugin SDK, and nothing of the UI, the shell, the wire or an application. */
export function verifyPluginHostIsolation() {
  const found = findPluginHostImports();
  if (found.length > 0) {
    throw new Error("The plugin host imports beyond vue and @numen/plugin-sdk:\n" + found.join("\n"));
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

/** The tab kinds, note types and commands plugins own. */
const PLUGIN_KINDS = [
  "deck",
  "card-model",
  "preset",
  "plex",
  "agent",
  "files",
  "travel",
  "ocr",
  "recognise",
  "proofread",
  "transcribe",
  "reads_hooks_and_skills",
];

/** Files of the window where one of PLUGIN_KINDS names a host service on the plugin context. */
const CORE_NAME_FILES = ["apps/desktop/editor/src/entities/plugin/model/context.ts"];

/** The window names no tab kind, note type or command of a plugin as a literal. */
export function verifyWindowNamesNoPluginKind() {
  const literal = `['"\`](${PLUGIN_KINDS.join("|")})['"\`]`;
  const grep = spawnSync(
    "git",
    [
      "grep", "-nE", literal, "--",
      "apps/desktop/editor/src/*.ts", "apps/desktop/editor/src/*.vue",
      ":!*.test.ts", ":!*.spec.ts", ":!*.stories.ts",
    ],
    { cwd: REPO_ROOT, encoding: "utf8" },
  );
  const found = (grep.stdout || "")
    .split("\n")
    .filter(Boolean)
    .filter((line) => !CORE_NAME_FILES.some((prefix) => line.split(":")[0].startsWith(prefix)));
  if (found.length > 0) {
    throw new Error(`The window names plugin kinds (${found.length}):\n` + found.join("\n"));
  }
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

/** Every direct `@numen/protocol` import in a frontend client application (`apps/*`). */
function findAppDirectProtocolImports() {
  const found = [];
  for (const file of listShippedFiles(["apps/desktop/editor/src", "apps/desktop/flashcards/src", "apps/mobile/src"])) {
    const source = readSource(file);
    if (!source) continue;
    for (const { line, value } of source.literals) {
      if (value === "@numen/protocol" || value.startsWith("@numen/protocol/") || value.includes("libs/protocol")) {
        found.push({ file, line, symbol: JSON.stringify(value) });
      }
    }
  }
  return found;
}

/** A frontend client application talks to the core through `@numen/plugin-sdk` and imports no `@numen/protocol`. */
export function verifyAppsNoDirectProtocol() {
  failOnFindings("An application frontend imports @numen/protocol directly instead of using @numen/plugin-sdk", findAppDirectProtocolImports());
}

/** F1 & F10: Segment direction (ui -> model -> api -> lib) and no @x cross-entity imports. */
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
        found.push({ file, line, symbol: `cross-entity @x import (F10): ${JSON.stringify(value)}` });
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
          found.push({ file, line, symbol: `upward segment import ${fromSeg} -> ${toSeg} (F1): ${JSON.stringify(value)}` });
        }
      }
    }
  }
  return found;
}

export function verifyFrontendSegmentsAndEntities() {
  failOnFindings("Frontend upward segment imports (lib/api/model -> ui) or @x cross-entity imports (F1, F10)", findFrontendSegmentAndCrossEntityViolations());
}

/** F5 & F6: Vue <style> raw colors/fallbacks, <template> array transforms, and TS catch returning null/undefined/false. */
function findVueStyleTemplateAndResultViolations() {
  const found = [];
  for (const file of listShippedFiles(["libs/ui/core/src", "libs/ui/shell/src", "apps/desktop/editor/src", "apps/desktop/flashcards/src", "apps/mobile/src", "plugins"])) {
    if (!/\.(ts|vue)$/.test(file) || /\/frontend\/gen\//.test(file)) continue;
    const raw = fs.readFileSync(path.join(REPO_ROOT, file), "utf8");
    if (file.endsWith(".vue")) {
      const tplMatch = raw.match(/<template[^>]*>([\s\S]*?)<\/template>/);
      if (tplMatch) {
        const baseLine = raw.slice(0, tplMatch.index).split("\n").length;
        tplMatch[1].split("\n").forEach((line, idx) => {
          const m = line.match(/\.(filter|map|reduce|sort|slice)\s*\(|\bnew\s+Date\s*\(|\bJSON\.(?:stringify|parse)\s*\(/);
          if (m) found.push({ file, line: baseLine + idx, symbol: `<template> inline computation ${m[0].trim()} (F5)` });
        });
      }
      const styleMatch = raw.match(/<style[^>]*>([\s\S]*?)<\/style>/);
      if (styleMatch && !/theme/i.test(file)) {
        const baseLine = raw.slice(0, styleMatch.index).split("\n").length;
        styleMatch[1].split("\n").forEach((line, idx) => {
          if (/^\s*\/\//.test(line)) return;
          if (/#[0-9a-fA-F]{3,8}\b|\brgba?\s*\(|\bhsla?\s*\(/.test(line)) {
            found.push({ file, line: baseLine + idx, symbol: `<style> raw color literal (F5): ${line.trim()}` });
          }
          if (/var\(\s*--[\w-]+\s*,/.test(line)) {
            found.push({ file, line: baseLine + idx, symbol: `<style> token fallback var(--token, fallback) (F5): ${line.trim()}` });
          }
          if (/^\s*--[\w-]+\s*:/.test(line)) {
            found.push({ file, line: baseLine + idx, symbol: `<style> local custom property declaration (F5): ${line.trim()}` });
          }
        });
      }
    }
    if (/\/(model|api|features|entities)\//.test(file)) {
      for (const m of raw.matchAll(/catch\s*(?:\([^)]*\))?\s*\{([^{}]*)\}/g)) {
        if (/\breturn\s+(?:null|undefined|false|""|''|\[\])\b/.test(m[1])) {
          const line = raw.slice(0, m.index).split("\n").length;
          found.push({ file, line, symbol: `catch block returns bare primitive instead of Result<T, E> (F6)` });
        }
      }
    }
  }
  return found;
}

export function verifyVueStyleTemplateAndResultPattern() {
  failOnFindings("Vue <style> token bypass, <template> inline transform, or catch returning bare null/undefined/false (F5, F6)", findVueStyleTemplateAndResultViolations());
}

/** F9: Every TypeScript package with a tsconfig.json has an eslint.config.js. */
function findTsPackagesWithoutEslint() {
  const candidates = [
    "libs/ui/core",
    "libs/ui/shell",
    "libs/plugin-sdk-ts",
    "libs/plugin-host-ts",
    "apps/desktop/editor",
    "apps/desktop/flashcards",
    "apps/mobile",
  ];
  return candidates
    .filter((dir) => fs.existsSync(path.join(REPO_ROOT, dir, "tsconfig.json")) && !fs.existsSync(path.join(REPO_ROOT, dir, "eslint.config.js")))
    .map((dir) => ({ file: `${dir}/tsconfig.json`, line: 1, symbol: "missing eslint.config.js (F9)" }));
}

export function verifyAllTsPackagesHaveEslint() {
  failOnFindings("TypeScript package missing eslint.config.js (F9)", findTsPackagesWithoutEslint());
}

export const FRONTEND_VERIFIERS = [
  {
    id: "all-ts-packages-eslint",
    scope: "libs",
    module: "sdk",
    name: "Every TypeScript Package Has ESLint Configured (F9)",
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
    name: "UI Library Holds No Plugin View (agent chat: Agent, thread, message composer)",
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
    name: "Plugin Host Isolation (imports only vue and @numen/plugin-sdk)",
    run: verifyPluginHostIsolation,
  },
  {
    id: "frontend-segments-and-entities",
    scope: "apps",
    module: "desktop",
    name: "Frontend Segment Direction (ui -> model -> api -> lib) & No @x Entity Cross-Imports (F1, F10)",
    run: verifyFrontendSegmentsAndEntities,
  },
  {
    id: "vue-style-template-and-result",
    scope: "apps",
    module: "desktop",
    name: "Vue <style> Tokens, <template> Purity & TS Result<T, E> in Catch (F5, F6)",
    run: verifyVueStyleTemplateAndResultPattern,
  },
  {
    id: "apps-reach-no-plugin",
    scope: "apps",
    module: "apps",
    name: "Applications Reach No Plugin (no path, alias, glob or import into plugins/)",
    run: verifyAppsReachNoPlugin,
  },
  {
    id: "apps-no-direct-protocol",
    scope: "apps",
    module: "apps",
    name: "Applications Talk to Core Through SDK (no direct @numen/protocol in apps/*/src)",
    run: verifyAppsNoDirectProtocol,
  },
  {
    id: "window-names-no-plugin-kind",
    scope: "apps",
    module: "desktop",
    name: "Desktop Window Names No Plugin Kind (deck, card model, preset, plex, agent, files)",
    run: verifyWindowNamesNoPluginKind,
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
