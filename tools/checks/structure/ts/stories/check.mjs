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

/** The expression under `as`, `satisfies`, `await` and parentheses. */
function unwrap(expression) {
  let one = expression;
  while (one && (ts.isAsExpression(one) || ts.isSatisfiesExpression(one) || ts.isParenthesizedExpression(one) || ts.isAwaitExpression(one))) one = one.expression;
  return one;
}

/** The name of a property written as an identifier or a string, or undefined. */
function getPropertyName(member) {
  return member.name && (ts.isIdentifier(member.name) || ts.isStringLiteral(member.name)) ? member.name.text : undefined;
}

/** The member of an object literal called `name`, or undefined. */
function findMember(object, name) {
  return object.properties.find((one) => getPropertyName(one) === name);
}

/** The object literal an expression is, under `as`, `satisfies` and parentheses, or undefined. */
function asObject(expression) {
  const object = unwrap(expression);
  return object && ts.isObjectLiteralExpression(object) ? object : undefined;
}

/** The PascalCase name written as a tag: `TreeRow` is `tree-row`. */
const toKebab = (name) => name.replace(/([a-z0-9])([A-Z])/g, "$1-$2").toLowerCase();

/** Whether a template text opens a tag called `name`, in the written or the kebab-case form. */
const opensTag = (template, name) => new RegExp(`<(?:${name}|${toKebab(name)})[\\s/>]`).test(template);

/**
 * The local names a piece of a story file uses as the component it draws: the value of a `component:` property, a
 * name registered in `components: { ... }` next to a `template` that opens its tag, and the first argument of `h(...)`.
 * A name that is only imported, mentioned in a type (`typeof Button`) or listed somewhere else is not a target. A helper
 * of the file that the piece calls or names (`row(args)`) is read too.
 */
function listTargetNames(node, helpers, found = new Set(), visiting = new Set()) {
  const visit = (one) => {
    if (ts.isPropertyAssignment(one) && getPropertyName(one) === "component" && ts.isIdentifier(unwrap(one.initializer))) found.add(unwrap(one.initializer).text);
    if (ts.isObjectLiteralExpression(one)) {
      const registry = findMember(one, "components");
      const template = findMember(one, "template");
      const text = template && ts.isPropertyAssignment(template) ? template.initializer.getText() : "";
      const members = registry && ts.isPropertyAssignment(registry) ? asObject(registry.initializer)?.properties ?? [] : [];
      for (const member of members) {
        const key = getPropertyName(member) ?? (ts.isShorthandPropertyAssignment(member) ? member.name.text : undefined);
        const value = ts.isShorthandPropertyAssignment(member) ? member.name.text : ts.isPropertyAssignment(member) && ts.isIdentifier(member.initializer) ? member.initializer.text : undefined;
        if (key && value && (opensTag(text, key) || opensTag(text, value))) found.add(value);
      }
    }
    if (ts.isCallExpression(one) && ts.isIdentifier(one.expression) && ["h", "createVNode"].includes(one.expression.text) && one.arguments[0] && ts.isIdentifier(one.arguments[0])) {
      found.add(one.arguments[0].text);
    }
    if (ts.isIdentifier(one) && helpers.has(one.text) && !visiting.has(one.text) && !(ts.isPropertyAssignment(one.parent) && one.parent.name === one)) {
      visiting.add(one.text);
      visit(helpers.get(one.text));
    }
    ts.forEachChild(one, visit);
  };
  visit(node);
  return found;
}

/** The identifier a chain of accesses, calls and awaits starts at, or undefined. */
function getRoot(expression) {
  let one = unwrap(expression);
  while (one && (ts.isPropertyAccessExpression(one) || ts.isElementAccessExpression(one) || ts.isCallExpression(one) || ts.isNonNullExpression(one))) one = unwrap(one.expression);
  return one && ts.isIdentifier(one) ? one.text : undefined;
}

/** The names of a story file that stand for a spy or for what a component emitted: `args`, `fn()` results and what is read from them. */
function listSpyNames(tree, play) {
  const names = new Set(["args"]);
  const bind = (name) => {
    if (ts.isIdentifier(name)) names.add(name.text);
    else name.elements.forEach((one) => ts.isBindingElement(one) && bind(one.name));
  };
  const visit = (node) => {
    if (ts.isVariableDeclaration(node) && node.initializer) {
      const init = unwrap(node.initializer);
      if ((ts.isCallExpression(init) && getRoot(init.expression) === "fn") || (getRoot(init) && names.has(getRoot(init)))) bind(node.name);
    }
    ts.forEachChild(node, visit);
  };
  visit(tree);
  if (play) visit(play);
  return names;
}

/** The objects whose calls act as a person does: user-event, the testing library's `fireEvent` and a `user` made by `userEvent.setup()`. */
const USER_EVENT_ROOTS = new Set(["userEvent", "fireEvent", "user"]);

/** The methods of an element that act as a person does when the play calls them directly. */
const USER_ACTIONS = new Set(["click", "dblclick", "dispatchEvent"]);

/** Whether a call acts as a person does: a call on user-event or on `fireEvent`, or a click on an element. */
function isUserAction(call) {
  const callee = unwrap(call.expression);
  if (!ts.isPropertyAccessExpression(callee)) return false;
  return USER_ACTIONS.has(callee.name.text) || USER_EVENT_ROOTS.has(getRoot(callee.expression) ?? "");
}

/**
 * The `expect(...)` calls in a play function: all of them, and those after a user action whose subject is a spy or
 * what a component emitted (a call argument or a payload, rooted at `args`, at a `fn()` result or at a name that says
 * `emitted`). A spy assertion before any user action sees a render, not a reaction. A helper function of the file that
 * the play calls is read too.
 */
function readAssertions(play, spies, helpers) {
  let all = 0;
  let emission = 0;
  let hasActed = false;
  const visited = new Set();
  const visit = (node) => {
    if (ts.isCallExpression(node) && isUserAction(node)) hasActed = true;
    if (ts.isCallExpression(node) && ts.isIdentifier(node.expression) && helpers.has(node.expression.text) && !visited.has(node.expression.text)) {
      visited.add(node.expression.text);
      visit(helpers.get(node.expression.text));
    }
    if (ts.isCallExpression(node) && ts.isIdentifier(node.expression) && node.expression.text === "expect") {
      all += 1;
      const subject = node.arguments[0];
      const root = subject && getRoot(subject);
      if (hasActed && root && (spies.has(root) || /emitted/i.test(root))) emission += 1;
    }
    ts.forEachChild(node, visit);
  };
  visit(play);
  return { all, emission };
}

/** The `play` function of an object literal, or undefined. */
function findPlay(object) {
  const member = object && findMember(object, "play");
  if (!member) return undefined;
  if (ts.isMethodDeclaration(member)) return member;
  return ts.isPropertyAssignment(member) ? unwrap(member.initializer) : undefined;
}

/**
 * What a story file shows: the `.vue` files of the components its stories draw, and each exported story with the
 * components it draws and its play function. A story draws a component when the component is its `component:`
 * (or the `component:` of the default export every story inherits) or the target it renders (see `listTargetNames`).
 */
export function readStoryFile(sources, file, srcRoot) {
  const tree = sources.parse(file);
  const imports = new Map();
  for (const statement of tree.statements.filter(ts.isImportDeclaration)) {
    const target = resolveSpecifier(sources, file, statement.moduleSpecifier.text, srcRoot);
    for (const { local, imported } of listImportBindings(statement)) {
      const vue = !target ? undefined : target.endsWith(".vue") ? (imported === "default" ? target : undefined) : listComponentExports(sources, target, srcRoot).get(imported);
      imports.set(local, vue ?? null);
    }
  }
  const declared = new Map();
  const helpers = new Map();
  for (const statement of tree.statements.filter(ts.isVariableStatement)) {
    const isExported = statement.modifiers?.some((m) => m.kind === ts.SyntaxKind.ExportKeyword) ?? false;
    for (const one of statement.declarationList.declarations) {
      if (!ts.isIdentifier(one.name) || !one.initializer) continue;
      declared.set(one.name.text, { initializer: one.initializer, isExported, line: tree.getLineAndCharacterOfPosition(one.getStart(tree)).line + 1 });
      helpers.set(one.name.text, one.initializer);
    }
  }
  for (const statement of tree.statements.filter(ts.isFunctionDeclaration)) if (statement.name) helpers.set(statement.name.text, statement);
  const defaultExport = tree.statements.find(ts.isExportAssignment);
  const metaExpression = defaultExport && ts.isIdentifier(unwrap(defaultExport.expression)) ? declared.get(unwrap(defaultExport.expression).text)?.initializer : defaultExport?.expression;
  const metaObject = metaExpression && asObject(metaExpression);
  const metaName = defaultExport && ts.isIdentifier(unwrap(defaultExport.expression)) ? unwrap(defaultExport.expression).text : undefined;
  if (metaName) helpers.delete(metaName);
  const toVue = (names) => new Set([...names].map((name) => imports.get(name)).filter(Boolean));
  const usesImport = (names) => [...names].some((name) => imports.has(name));
  const metaNames = metaObject ? listTargetNames(metaObject, helpers) : new Set();
  const metaPlay = findPlay(metaObject);
  const rendered = toVue(metaNames);
  const stories = [];
  for (const [name, one] of declared) {
    if (!one.isExported || name === metaName) continue;
    const names = listTargetNames(one.initializer, helpers);
    const own = asObject(one.initializer);
    const play = findPlay(own) ?? metaPlay;
    const all = new Set([...names, ...metaNames]);
    for (const vue of toVue(names)) rendered.add(vue);
    const assertions = play ? readAssertions(play, listSpyNames(tree, play), helpers) : { all: 0, emission: 0 };
    stories.push({
      name,
      line: one.line,
      targets: toVue(all),
      drawsComponent: usesImport(all),
      hasPlay: Boolean(play),
      assertions: assertions.all,
      emissionAssertions: assertions.emission,
    });
  }
  return { rendered, stories, hasPlay: stories.some((one) => one.hasPlay) };
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
 * The public components of a library entry that no story draws, those that declare events and have no story whose play
 * function asserts an emitted event or a spy call, the plays that assert nothing, and the stories that draw no component.
 *
 * @returns {{ unstoried: Array, unplayed: Array, emptyPlays: Array, bareStories: Array, components: number, stories: number, emitting: number }}
 */
export function findComponentsWithoutStories(sources, entry) {
  const srcRoot = path.posix.dirname(entry);
  const components = listComponentExports(sources, entry, srcRoot);
  const files = sources.list().filter((file) => file.startsWith(`${srcRoot}/`) && file.endsWith(".stories.ts")).map((file) => ({ file, ...readStoryFile(sources, file, srcRoot) }));
  const unstoried = [];
  const unplayed = [];
  let emitting = 0;
  for (const [name, vue] of components) {
    const drawing = files.filter((story) => story.rendered.has(vue));
    if (drawing.length === 0) unstoried.push({ file: vue, line: 1, symbol: `public component ${name} is rendered by no story file` });
    if (isEmitting(sources.read(vue), vue)) {
      emitting += 1;
      const stories = files.flatMap((story) => story.stories).filter((story) => story.targets.has(vue));
      if (!stories.some((story) => story.emissionAssertions > 0)) {
        unplayed.push({ file: vue, line: 1, symbol: `public component ${name} declares events and the play function of no story that draws it asserts an emitted event or a spy call after a user action` });
      }
    }
  }
  const emptyPlays = files.flatMap((story) =>
    story.stories.filter((one) => one.hasPlay && one.assertions === 0).map((one) => ({ file: story.file, line: one.line, symbol: `story ${one.name} has a play function with no expect` })),
  );
  const bareStories = files.flatMap((story) =>
    story.stories.filter((one) => !one.drawsComponent).map((one) => ({ file: story.file, line: one.line, symbol: `story ${one.name} draws no component: none is its component: or its render target` })),
  );
  return { unstoried, unplayed, emptyPlays, bareStories, components: components.size, stories: files.length, emitting };
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
  failOnFindings("A public component that declares events has no story whose play function asserts an emitted event or a spy call after a user action", all.flatMap((one) => one.unplayed));
}

export function verifyEveryPlayAsserts() {
  const all = getReport();
  requireMatches("Story files", all.reduce((sum, one) => sum + one.stories, 0));
  failOnFindings("A play function of a story asserts nothing", all.flatMap((one) => one.emptyPlays));
}

export function verifyEveryStoryDrawsAComponent() {
  const all = getReport();
  requireMatches("Story files", all.reduce((sum, one) => sum + one.stories, 0));
  failOnFindings("A story draws no component", all.flatMap((one) => one.bareStories));
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
    name: "Every Public Component That Declares Events or a Model Has a Story Whose play Function Acts as a Person Does and Then Asserts an Emitted Event or a Spy Call",
    run: verifyEveryEmittingComponentHasPlayStory,
  },
  {
    id: "story-play-asserts",
    scope: "libs",
    module: "ui",
    name: "Every play Function of a Story Holds an expect",
    run: verifyEveryPlayAsserts,
  },
  {
    id: "story-draws-a-component",
    scope: "libs",
    module: "ui",
    name: "Every Exported Story Draws a Component as Its component: or Its Render Target",
    run: verifyEveryStoryDrawsAComponent,
  },
];
