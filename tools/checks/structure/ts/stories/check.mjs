import { parse as parseVue } from "@vue/compiler-sfc";
import fs from "node:fs";
import path from "node:path";
import ts from "typescript";
import { runTool } from "../../../../shared/tools.mjs";
import { REPO_ROOT, failOnFindings, requireMatches } from "../../../lib/graph.mjs";

/** The libraries whose public components are drawn by stories: the packages under `libs/ui`. */
const LIBRARY_ENTRY = /^libs\/ui\/[^/]+\/src\/index\.ts$/;

/** The call that declares the events or the model of a component. */
const EVENT_DECLARATIONS = new Set(["defineEmits", "defineModel"]);

/**
 * A set of source files by repository-relative path. The checks read through it, so a test hands them
 * a few strings and the verifier hands them the tracked files.
 */
export function createSources(entries) {
  const texts = new Map(entries);
  const trees = new Map();
  return {
    has: (file) => texts.has(file),
    list: () => [...texts.keys()],
    read: (file) => texts.get(file),
    /** The syntax tree of a TypeScript file, kept for the next read. */
    parse(file) {
      if (!trees.has(file)) trees.set(file, ts.createSourceFile(file, texts.get(file), ts.ScriptTarget.Latest, true));
      return trees.get(file);
    },
  };
}

/** The file a module specifier written in `from` names, or undefined when it names none of the sources. */
function resolveSpecifier(sources, from, specifier, srcRoot) {
  let base;
  if (specifier.startsWith(".")) base = path.posix.join(path.posix.dirname(from), specifier);
  else if (specifier.startsWith("@/")) base = path.posix.join(srcRoot, specifier.slice(2));
  else return undefined;
  return [base, `${base}.ts`, `${base}.vue`, `${base}/index.ts`].find((candidate) => sources.has(candidate));
}

/** The local names an import declaration binds, each with the name it imports (`default` for a default import). */
function listImportBindings(declaration) {
  const clause = declaration.importClause;
  if (!clause || clause.isTypeOnly) return [];
  const named = clause.namedBindings && ts.isNamedImports(clause.namedBindings) ? clause.namedBindings.elements.filter((one) => !one.isTypeOnly) : [];
  return [
    ...(clause.name ? [{ local: clause.name.text, imported: "default" }] : []),
    ...named.map((one) => ({ local: one.name.text, imported: (one.propertyName ?? one.name).text })),
  ];
}

/**
 * The components a module exports by name, as a map from the exported name to the `.vue` file. A component
 * is a default export of a `.vue` file; a re-export, a re-export under another name, `export *` and an
 * import that is exported again are followed. A type export is not a component.
 */
export function listComponentExports(sources, file, srcRoot, visiting = new Set()) {
  const found = new Map();
  if (visiting.has(file) || !file.endsWith(".ts")) return found;
  const seen = new Set([...visiting, file]);
  const tree = sources.parse(file);
  const imported = new Map();
  for (const statement of tree.statements.filter(ts.isImportDeclaration)) {
    const target = resolveSpecifier(sources, file, statement.moduleSpecifier.text, srcRoot);
    for (const { local, imported: name } of listImportBindings(statement)) imported.set(local, { target, name });
  }
  const take = (target, name) => (target?.endsWith(".vue") ? (name === "default" ? target : undefined) : listComponentExports(sources, target, srcRoot, seen).get(name));
  for (const statement of tree.statements.filter(ts.isExportDeclaration)) {
    if (statement.isTypeOnly) continue;
    const target = statement.moduleSpecifier ? resolveSpecifier(sources, file, statement.moduleSpecifier.text, srcRoot) : undefined;
    if (statement.moduleSpecifier && !target) continue;
    const clause = statement.exportClause;
    if (!clause) {
      for (const [name, vue] of listComponentExports(sources, target, srcRoot, seen)) found.set(name, vue);
    } else if (ts.isNamedExports(clause)) {
      for (const one of clause.elements.filter((element) => !element.isTypeOnly)) {
        const name = (one.propertyName ?? one.name).text;
        const vue = target ? take(target, name) : imported.has(name) ? take(imported.get(name).target, imported.get(name).name) : undefined;
        if (vue) found.set(one.name.text, vue);
      }
    }
  }
  return found;
}

/** Whether a node is an identifier of the file other than the one in an import declaration. */
function isUseOf(node, name) {
  return ts.isIdentifier(node) && node.text === name && !ts.findAncestor(node, ts.isImportDeclaration);
}

/** Whether any node under `node` is an identifier called `name` outside the imports. */
function mentions(node, name) {
  let found = false;
  const visit = (one) => {
    if (found) return;
    if (isUseOf(one, name)) found = true;
    else ts.forEachChild(one, visit);
  };
  visit(node);
  return found;
}

/** The expression under `as`, `satisfies` and parentheses. */
function unwrap(expression) {
  let one = expression;
  while (one && (ts.isAsExpression(one) || ts.isSatisfiesExpression(one) || ts.isParenthesizedExpression(one))) one = one.expression;
  return one;
}

/** Whether an object literal has a member called `play`. */
function hasPlayMember(expression) {
  const object = unwrap(expression);
  return Boolean(object && ts.isObjectLiteralExpression(object) && object.properties.some((one) => one.name && ts.isIdentifier(one.name) && one.name.text === "play"));
}

/**
 * What a story file shows: the `.vue` files of the components it renders (an import of the component that the
 * file uses) and whether an exported story, or the default export every story inherits, has a `play` function.
 */
export function readStoryFile(sources, file, srcRoot) {
  const tree = sources.parse(file);
  const rendered = new Set();
  for (const statement of tree.statements.filter(ts.isImportDeclaration)) {
    const target = resolveSpecifier(sources, file, statement.moduleSpecifier.text, srcRoot);
    if (!target) continue;
    for (const { local, imported } of listImportBindings(statement)) {
      const vue = target.endsWith(".vue") ? (imported === "default" ? target : undefined) : listComponentExports(sources, target, srcRoot).get(imported);
      if (vue && mentions(tree, local)) rendered.add(vue);
    }
  }
  const declared = new Map();
  for (const statement of tree.statements.filter(ts.isVariableStatement)) {
    for (const one of statement.declarationList.declarations) if (ts.isIdentifier(one.name)) declared.set(one.name.text, { initializer: one.initializer, isExported: statement.modifiers?.some((m) => m.kind === ts.SyntaxKind.ExportKeyword) });
  }
  const defaultExport = tree.statements.find(ts.isExportAssignment);
  const metaName = defaultExport && ts.isIdentifier(unwrap(defaultExport.expression)) ? unwrap(defaultExport.expression).text : undefined;
  const hasPlay =
    (defaultExport && hasPlayMember(defaultExport.expression)) ||
    (metaName && declared.has(metaName) && hasPlayMember(declared.get(metaName).initializer)) ||
    [...declared.values()].some((one) => one.isExported && hasPlayMember(one.initializer));
  return { rendered, hasPlay: Boolean(hasPlay) };
}

/** Whether the script of a component declares events or a model. */
export function isEmitting(vueSource, file = "Component.vue") {
  const { descriptor } = parseVue(vueSource);
  return [descriptor.script, descriptor.scriptSetup].filter(Boolean).some((block) => {
    const tree = ts.createSourceFile(`${file}.ts`, block.content, ts.ScriptTarget.Latest, true);
    let found = false;
    const visit = (node) => {
      if (ts.isCallExpression(node) && ts.isIdentifier(node.expression) && EVENT_DECLARATIONS.has(node.expression.text)) found = true;
      else ts.forEachChild(node, visit);
    };
    visit(tree);
    return found;
  });
}

/**
 * The public components of a library entry that no story file of the library renders, and those that declare
 * events and are rendered by no story file with a `play` function.
 *
 * @returns {{ unstoried: Array, unplayed: Array, components: number, stories: number, emitting: number }}
 */
export function findComponentsWithoutStories(sources, entry) {
  const srcRoot = path.posix.dirname(entry);
  const components = listComponentExports(sources, entry, srcRoot);
  const stories = sources.list().filter((file) => file.startsWith(`${srcRoot}/`) && file.endsWith(".stories.ts")).map((file) => ({ file, ...readStoryFile(sources, file, srcRoot) }));
  const unstoried = [];
  const unplayed = [];
  let emitting = 0;
  for (const [name, vue] of components) {
    const drawing = stories.filter((story) => story.rendered.has(vue));
    if (drawing.length === 0) unstoried.push({ file: vue, line: 1, symbol: `public component ${name} is rendered by no story file` });
    if (isEmitting(sources.read(vue), vue)) {
      emitting += 1;
      if (!drawing.some((story) => story.hasPlay)) unplayed.push({ file: vue, line: 1, symbol: `public component ${name} declares events and no story file that renders it has a play function` });
    }
  }
  return { unstoried, unplayed, components: components.size, stories: stories.length, emitting };
}

let report;
/** The result of the two story checks over every library entry, read once. */
function getReport() {
  if (report) return report;
  const listed = runTool("git", ["ls-files", "-z", "--", "libs/ui"], { cwd: REPO_ROOT, encoding: "utf8", maxBuffer: 1 << 28 });
  const files = (listed.stdout || "").split("\0").filter((file) => /\.(ts|vue)$/.test(file) && fs.existsSync(path.join(REPO_ROOT, file)));
  const sources = createSources(files.map((file) => [file, fs.readFileSync(path.join(REPO_ROOT, file), "utf8")]));
  const entries = files.filter((file) => LIBRARY_ENTRY.test(file));
  requireMatches("Library entries", entries.length);
  report = entries.map((entry) => findComponentsWithoutStories(sources, entry));
  return report;
}

export function verifyEveryPublicComponentHasStory() {
  const all = getReport();
  requireMatches("Public components", all.reduce((sum, one) => sum + one.components, 0));
  requireMatches("Story files", all.reduce((sum, one) => sum + one.stories, 0));
  failOnFindings("A public component is rendered by no story", all.flatMap((one) => one.unstoried));
}

export function verifyEveryEmittingComponentHasPlayStory() {
  const all = getReport();
  requireMatches("Public components that declare events", all.reduce((sum, one) => sum + one.emitting, 0));
  failOnFindings("A public component that declares events has no story with a play function", all.flatMap((one) => one.unplayed));
}

export const STORY_VERIFIERS = [
  {
    id: "public-component-has-story",
    scope: "libs",
    module: "ui",
    name: "Every Public Component of @numen/ui and @numen/workbench Is Rendered by a Story File",
    run: verifyEveryPublicComponentHasStory,
  },
  {
    id: "emitting-component-has-play-story",
    scope: "libs",
    module: "ui",
    name: "Every Public Component That Declares Events or a Model Is Rendered by a Story With a play Function",
    run: verifyEveryEmittingComponentHasPlayStory,
  },
];
