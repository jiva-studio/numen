/**
 * The rules over design tokens that stylelint has no rule for: no `var()` takes a fallback, a stylesheet
 * declares no `--numen-` token, and every `var(--numen-…)` names a token a token file declares.
 */
import fs from "node:fs";
import path from "node:path";
import process from "node:process";
import postcss from "postcss";
import valueParser from "postcss-value-parser";
import stylelint from "stylelint";

const {
  createPlugin,
  utils: { report, ruleMessages, validateOptions },
} = stylelint;

/** The prefix of the design tokens. */
const TOKEN_PREFIX = "--numen-";

/** Directories the token walk does not enter: installed packages, build output, fixtures; hidden directories neither. */
const SKIPPED_DIRECTORIES = new Set(["node_modules", "dist", "testdata"]);

/** True for a stylesheet that lies in a `tokens` directory. */
export function isTokenFile(file) {
  return file.endsWith(".css") && file.split(/[\\/]/).slice(0, -1).includes("tokens");
}

/** The stylesheets below `dir` that are token files, as absolute paths. */
function findTokenFiles(dir) {
  const found = [];
  for (const entry of fs.readdirSync(dir, { withFileTypes: true })) {
    const abs = path.join(dir, entry.name);
    if (entry.isDirectory() && !SKIPPED_DIRECTORIES.has(entry.name) && !entry.name.startsWith(".")) found.push(...findTokenFiles(abs));
    else if (entry.isFile() && isTokenFile(abs)) found.push(abs);
  }
  return found;
}

/** The custom properties with the token prefix that the declarations of `css` declare. */
export function readDeclaredTokens(css) {
  const declared = new Set();
  postcss.parse(css).walkDecls((decl) => {
    if (decl.prop.startsWith(TOKEN_PREFIX)) declared.add(decl.prop);
  });
  return declared;
}

const declaredByRoot = new Map();

/** The tokens the token files below `root` declare, read once per root. */
function readRootTokens(root) {
  if (!declaredByRoot.has(root)) {
    const declared = new Set();
    for (const file of findTokenFiles(root)) {
      for (const token of readDeclaredTokens(fs.readFileSync(file, "utf8"))) declared.add(token);
    }
    declaredByRoot.set(root, declared);
  }
  return declaredByRoot.get(root);
}

/** The `var()` calls of a declaration value, each as `{ name, hasFallback, index }`. */
function findVarCalls(value) {
  const calls = [];
  valueParser(value).walk((node) => {
    if (node.type !== "function" || node.value.toLowerCase() !== "var") return;
    const name = node.nodes.find((one) => one.type === "word")?.value ?? "";
    const hasFallback = node.nodes.some((one) => one.type === "div" && one.value === ",");
    calls.push({ name, hasFallback, index: node.sourceIndex });
  });
  return calls;
}

const NO_FALLBACK = "numen/no-var-fallback";

const noFallbackMessages = ruleMessages(NO_FALLBACK, {
  rejected: (name) => `var(${name}, …) takes a fallback; the property is declared where it is read`,
});

const noFallback = (primary) => (root, result) => {
  if (!validateOptions(result, NO_FALLBACK, { actual: primary })) return;
  root.walkDecls((decl) => {
    for (const call of findVarCalls(decl.value)) {
      if (!call.hasFallback) continue;
      report({ ruleName: NO_FALLBACK, result, node: decl, message: noFallbackMessages.rejected(call.name), word: call.name });
    }
  });
};
noFallback.ruleName = NO_FALLBACK;
noFallback.messages = noFallbackMessages;

const NO_TOKEN_DECLARATION = "numen/no-token-declaration";

const noTokenDeclarationMessages = ruleMessages(NO_TOKEN_DECLARATION, {
  rejected: (name) => `${name} is a design token; tokens are declared in a token or theme file`,
});

const noTokenDeclaration = (primary) => (root, result) => {
  if (!validateOptions(result, NO_TOKEN_DECLARATION, { actual: primary })) return;
  root.walkDecls((decl) => {
    if (!decl.prop.startsWith(TOKEN_PREFIX)) return;
    report({ ruleName: NO_TOKEN_DECLARATION, result, node: decl, message: noTokenDeclarationMessages.rejected(decl.prop), word: decl.prop });
  });
};
noTokenDeclaration.ruleName = NO_TOKEN_DECLARATION;
noTokenDeclaration.messages = noTokenDeclarationMessages;

const KNOWN_TOKEN = "numen/known-token";

const knownTokenMessages = ruleMessages(KNOWN_TOKEN, {
  rejected: (name) => `${name} is declared by no token file`,
});

const knownToken = (primary, secondary) => (root, result) => {
  const isValid = validateOptions(
    result,
    KNOWN_TOKEN,
    { actual: primary },
    { actual: secondary, possible: { root: [(value) => typeof value === "string"] }, optional: true },
  );
  if (!isValid) return;
  const declared = readRootTokens(path.resolve(secondary?.root ?? process.cwd()));
  root.walkDecls((decl) => {
    for (const { name } of findVarCalls(decl.value)) {
      if (!name.startsWith(TOKEN_PREFIX) || declared.has(name)) continue;
      report({ ruleName: KNOWN_TOKEN, result, node: decl, message: knownTokenMessages.rejected(name), word: name });
    }
  });
};
knownToken.ruleName = KNOWN_TOKEN;
knownToken.messages = knownTokenMessages;

export default [
  createPlugin(NO_FALLBACK, noFallback),
  createPlugin(NO_TOKEN_DECLARATION, noTokenDeclaration),
  createPlugin(KNOWN_TOKEN, knownToken),
];
