import assert from "node:assert/strict";
import fs from "node:fs";
import os from "node:os";
import path from "node:path";
import { after, before, describe, it } from "node:test";
import stylelint from "stylelint";
import config from "../../../../../stylelint.config.mjs";
import { isTokenFile, readDeclaredTokens } from "../../../../lint-layers/stylelint-plugin.mjs";
import { REPO_ROOT } from "../../../lib/graph.mjs";

describe("the style rules", () => {
  let tokens;
  before(() => {
    tokens = fs.mkdtempSync(path.join(os.tmpdir(), "style-tokens-"));
    fs.mkdirSync(path.join(tokens, "tokens"));
    fs.writeFileSync(path.join(tokens, "tokens", "tokens.css"), ":root { --numen-ink: light-dark(#000, #fff); }\n");
  });
  after(() => fs.rmSync(tokens, { recursive: true, force: true }));

  /** The rules stylelint reports for `code` written at `file`, a path below the repository root that need not exist. */
  const lint = async (file, code) => {
    const { results } = await stylelint.lint({
      code,
      codeFilename: path.join(REPO_ROOT, file),
      config: { ...config, rules: { ...config.rules, "numen/known-token": [true, { root: tokens }] } },
      configBasedir: REPO_ROOT,
    });
    return [...results[0].warnings.map((one) => one.rule), ...(results[0].invalidOptionWarnings ?? []).map((one) => one.text)];
  };

  it("rejects a color literal outside a token or theme file", async () => {
    assert.deepEqual(await lint("view/a.css", "a { color: #fff; }"), ["color-no-hex"]);
    assert.deepEqual(await lint("view/a.css", "a { color: rgb(0 0 0); }"), ["function-disallowed-list"]);
    assert.deepEqual(await lint("view/a.css", "a { color: red; }"), ["color-named"]);
  });

  it("rejects a color literal in the style block of a component", async () => {
    assert.deepEqual(await lint("view/View.vue", "<template><p /></template>\n<style scoped>\np { color: #fff; }\n</style>\n"), ["color-no-hex"]);
  });

  it("rejects a var() fallback, a token declared outside a token file and a token no token file declares", async () => {
    assert.deepEqual(await lint("view/a.css", "a { color: var(--numen-ink, inherit); }"), ["numen/no-var-fallback"]);
    assert.deepEqual(await lint("view/a.css", "a { --numen-ink: inherit; }"), ["numen/no-token-declaration"]);
    assert.deepEqual(await lint("view/a.css", "a { color: var(--numen-missing); }"), ["numen/known-token"]);
  });

  it("accepts a declared token, a local property, and colors in token and theme files", async () => {
    assert.deepEqual(await lint("view/a.css", "a { --gap: 0.25rem; margin: var(--gap); color: var(--numen-ink); }"), []);
    assert.deepEqual(await lint("view/tokens/tokens.css", ":root { --numen-ink: light-dark(#000, rgb(255 255 255)); }"), []);
    assert.deepEqual(await lint("view/themes/dark.css", ":root { --numen-ink: white; }"), []);
  });
});

describe("the rule on a border and a radius", () => {
  /** The rules stylelint reports for `code` written as a stylesheet. */
  const lint = async (code) => {
    const { results } = await stylelint.lint({ code, codeFilename: path.join(REPO_ROOT, "view/a.css"), config, configBasedir: REPO_ROOT });
    return results[0].warnings.map((one) => one.rule);
  };

  it("rejects a border on one side in a rule that rounds the box", async () => {
    const rejected = ["numen/no-rounded-side-border"];
    assert.deepEqual(await lint("a { border-block-end: 1px solid currentColor; border-radius: var(--round); }"), rejected);
    assert.deepEqual(await lint("a { border-bottom: 1px solid currentColor; border-top-left-radius: var(--round); }"), rejected);
    assert.deepEqual(await lint("a { border-inline-start: 1px solid currentColor; border-radius: var(--round); }"), rejected);
    assert.deepEqual(await lint("a { @apply border-b rounded-md; }"), rejected);
  });

  it("accepts a border on one side without a radius, a radius with a border on every side, and a side border that draws nothing", async () => {
    assert.deepEqual(await lint("a { border-block-end: 1px solid currentColor; }"), []);
    assert.deepEqual(await lint("a { border: 1px solid currentColor; border-radius: var(--round); }"), []);
    assert.deepEqual(await lint("a { border-block-end: none; border-radius: var(--round); }"), []);
    assert.deepEqual(await lint("a { border-block-end: 1px solid currentColor; border-radius: 0; }"), []);
    assert.deepEqual(await lint("a { @apply border-b rounded-none; }"), []);
  });
});

describe("the rule on lengths", () => {
  /** The rules stylelint reports for `code` written at `file`. */
  const lint = async (file, code) => {
    const { results } = await stylelint.lint({ code, codeFilename: path.join(REPO_ROOT, file), config, configBasedir: REPO_ROOT });
    return results[0].warnings.map((one) => one.rule);
  };
  const rejected = ["numen/no-bare-length"];

  it("rejects a planted component that writes a length straight on a property", async () => {
    const planted = "<template><p /></template>\n<style scoped>\np { padding: 0.3rem; }\n</style>\n";
    assert.deepEqual(await lint("apps/desktop/editor/src/Planted.vue", planted), rejected);
  });

  it("rejects a length on a property in rem, em, ch and inside calc()", async () => {
    assert.deepEqual(await lint("plugins/views-tasks/frontend/a.css", "a { gap: 0.5rem; }"), rejected);
    assert.deepEqual(await lint("plugins/views-tasks/frontend/a.css", "a { margin-inline-start: 1.5em; }"), rejected);
    assert.deepEqual(await lint("plugins/views-tasks/frontend/a.css", "a { max-inline-size: 60ch; }"), rejected);
    assert.deepEqual(await lint("plugins/views-tasks/frontend/a.css", "a { inline-size: calc(100% - 2rem); }"), rejected);
    assert.deepEqual(await lint("plugins/views-tasks/frontend/a.css", "a { padding: var(--pad) 0.5rem; }"), rejected);
  });

  it("rejects px on a property and in a custom property, and a border wider than a hairline", async () => {
    assert.deepEqual(await lint("plugins/views-tasks/frontend/a.css", "a { outline: 2px solid currentColor; }"), rejected);
    assert.deepEqual(await lint("plugins/views-tasks/frontend/a.css", "a { border-radius: 4px; }"), rejected);
    assert.deepEqual(await lint("plugins/views-tasks/frontend/a.css", "a { --gap: 4px; }"), rejected);
    assert.deepEqual(await lint("plugins/views-tasks/frontend/a.css", "a { border: 2px solid currentColor; }"), rejected);
    assert.deepEqual(await lint("plugins/views-tasks/frontend/a.css", "a { outline: 1px solid currentColor; }"), rejected);
  });

  it("accepts a local custom property in rem or em, read with var()", async () => {
    assert.deepEqual(await lint("plugins/views-tasks/frontend/a.css", "a { --pad: 0.3rem; --icon: 1.2em; padding: var(--pad); inline-size: calc(var(--icon) * 2); }"), []);
  });

  it("accepts a 1px hairline border, zero, proportions and a length inside url()", async () => {
    assert.deepEqual(await lint("plugins/views-tasks/frontend/a.css", "a { border: 1px solid currentColor; border-block-end: 1px dashed currentColor; border-width: 1px; }"), []);
    assert.deepEqual(await lint("plugins/views-tasks/frontend/a.css", "a { margin: 0; padding: 0px; inline-size: 50%; block-size: 100vh; flex: 1 1 0%; line-height: 1.4; }"), []);
    assert.deepEqual(await lint("plugins/views-tasks/frontend/a.css", "a { background: url(icon-16px.png); transition-duration: 150ms; }"), []);
  });

  it("leaves a package outside the converted ones alone", async () => {
    assert.deepEqual(await lint("apps/docs/src/a.css", "a { gap: 0.5rem; border-radius: 4px; }"), []);
  });

  it("leaves the token files alone", async () => {
    assert.deepEqual(await lint("libs/ui/components/src/shared/tokens/tokens.css", ":root { --numen-hairline: 1px; --numen-radius: 4px; }"), []);
  });
});

describe("the token files", () => {
  it("are the stylesheets in a tokens directory", () => {
    assert.ok(isTokenFile("libs/ui/components/src/shared/tokens/tokens.css"));
    assert.equal(isTokenFile("libs/ui/components/src/tokens.css"), false);
    assert.equal(isTokenFile("libs/ui/components/src/tokens/index.ts"), false);
  });
  it("declare the custom properties with the token prefix", () => {
    assert.deepEqual([...readDeclaredTokens(":root { --numen-ink: #000; --local: 1px; }\n.a { --numen-rule: #111; }")], ["--numen-ink", "--numen-rule"]);
  });
  it("of this repository declare the tokens the components use", () => {
    const declared = readDeclaredTokens(fs.readFileSync(path.join(REPO_ROOT, "libs/ui/components/src/shared/tokens/tokens.css"), "utf8"));
    assert.ok(declared.has("--numen-ink"));
  });
});
