import { spawnSync } from "node:child_process";
import fs from "node:fs";
import path from "node:path";
import {
  MODULES_DIR,
  REPO_ROOT,
  exec,
  failOnFindings,
  findCrossings,
  getImportSpecifiers,
  listShippedFiles,
  readSource,
  requireMatches,
} from "./helpers.mjs";

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
