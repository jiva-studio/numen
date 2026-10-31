import assert from "node:assert/strict";
import fs from "node:fs";
import os from "node:os";
import path from "node:path";
import { describe, it } from "node:test";
import { parseManifest } from "../../shared/manifests.mjs";
import { loadManifestWords } from "./dictionary.mjs";
import { findPluginViewsInUi, loadPluginViews } from "./frontend.mjs";
import { findForeignTokensInManifest, listContributedTokens, loadOwnedTokens } from "./plugins.mjs";
import { findPluginIdLiterals, getIdNamespace } from "./vocabulary.mjs";

/** Manifests of three imaginary plugins; two of them share a category and a tab kind word. */
const SHARED_MANIFESTS = {
  "plugins/views-aaa/manifest.json": JSON.stringify({
    id: "md.example.views.aaa",
    name: "Aaa",
    version: "1.0.0",
    contributes: { category: "views", tabKinds: [{ kind: "gadget", title: "Gadget" }] },
    settings: [{ key: "mode", type: "enum", options: ["sprocket"] }],
  }),
  "plugins/views-bbb/manifest.json": JSON.stringify({
    id: "md.example.views.bbb",
    name: "Bbb",
    version: "1.0.0",
    contributes: {
      category: "views",
      tabKinds: [
        { kind: "widget", title: "Widget" },
        { kind: "sprocket", title: "Sprocket" },
      ],
    },
  }),
  "plugins/editors-ccc/manifest.json": JSON.stringify({
    id: "md.example.editors.ccc",
    name: "Ccc",
    version: "1.0.0",
    contributes: { category: "scribbling" },
  }),
};

const readShared = (file) => SHARED_MANIFESTS[file];
const sharedWords = () => loadManifestWords(Object.keys(SHARED_MANIFESTS), readShared);

describe("a word the manifests of two plugins share", () => {
  it("identifies no plugin and is left out of the words", () => {
    const words = sharedWords();
    assert.equal(words.has("views"), false, "a category of two plugins");
    assert.equal(words.has("sprocket"), false, "a value of one plugin and a tab kind of another");
    assert.equal(words.has("gadget"), true, "a tab kind of one plugin");
  });

  it("is kept for a category only one plugin states", () => {
    assert.deepEqual([...sharedWords().get("scribbling")], ["editors-ccc"]);
  });
});

describe("the tokens a plugin owns", () => {
  const manifests = Object.entries(SHARED_MANIFESTS).map(([file, text]) => ({ name: file.split("/")[1], manifest: parseManifest(text, file).manifest }));

  it("are the contributed tab kinds that no other manifest states", () => {
    const owned = loadOwnedTokens(manifests, sharedWords());
    assert.deepEqual(
      owned.map(({ token }) => token).sort(),
      ["gadget", "widget"],
    );
    assert.deepEqual(owned.find(({ token }) => token === "widget").owners, ["views-bbb"]);
  });

  it("include a tab kind with and without its reader prefix, a note type, a command and a condition value", () => {
    const tokens = listContributedTokens({
      contributes: { tabKinds: [{ kind: "reader:Epub" }], noteTypes: [{ type: "book" }] },
      commands: [{ id: "make-text", when: [{ key: "version.kind", is: ["transcript"] }] }],
    });
    assert.deepEqual(tokens, ["reader:epub", "epub", "book", "make-text", "transcript"]);
  });

  it("are found in the manifest of another plugin and not in the manifest of the owner", () => {
    const owned = [{ token: "widget", owners: ["views-bbb"] }];
    const text = '{\n  "kinds": ["widget"]\n}';
    assert.equal(findForeignTokensInManifest("plugins/views-aaa/manifest.json", "views-aaa", text, owned).length, 1);
    assert.deepEqual(findForeignTokensInManifest("plugins/views-bbb/manifest.json", "views-bbb", text, owned), []);
  });

  it("are read from the live manifests, enough to examine", () => {
    assert.ok(loadOwnedTokens().length >= 10);
  });
});

describe("the views a plugin owns", () => {
  const withPlugins = (check) => {
    const root = fs.mkdtempSync(path.join(os.tmpdir(), "views-"));
    try {
      const write = (plugin, files) => {
        fs.mkdirSync(path.join(root, "plugins", plugin), { recursive: true });
        fs.writeFileSync(
          path.join(root, "plugins", plugin, "manifest.json"),
          JSON.stringify({ id: `md.example.${plugin}`, name: plugin, version: "1.0.0" }),
        );
        for (const file of files) {
          fs.mkdirSync(path.join(root, "plugins", plugin, "frontend", path.dirname(file)), { recursive: true });
          fs.writeFileSync(path.join(root, "plugins", plugin, "frontend", file), "");
        }
      };
      write("alpha", ["thread/Thread.vue", "ui/Common.vue", "thread/ui/Inner.vue"]);
      write("beta", ["ui/Common.vue", "chat/Chat.vue"]);
      fs.mkdirSync(path.join(root, "ui", "features", "thread"), { recursive: true });
      check(root);
    } finally {
      fs.rmSync(root, { recursive: true, force: true });
    }
  };

  it("are the folders and components of exactly one plugin", () => {
    withPlugins((root) => {
      const views = loadPluginViews(path.join(root, "plugins"));
      assert.deepEqual([...views.dirs].sort(), [
        ["chat", "beta"],
        ["thread", "alpha"],
      ]);
      assert.deepEqual([...views.names.keys()].sort(), ["Chat", "Inner", "Thread"]);
    });
  });

  it("are found in the UI library as a folder or as an export", () => {
    withPlugins((root) => {
      const views = loadPluginViews(path.join(root, "plugins"));
      const found = findPluginViewsInUi(views, new Set(["Chat", "Button"]), path.join(root, "ui"));
      assert.equal(found.length, 2);
      assert.ok(found.some((one) => one.includes("thread") && one.includes("alpha")));
      assert.ok(found.some((one) => one.includes("Chat") && one.includes("beta")));
    });
  });

  it("are read from the live plugins, enough to examine", () => {
    const views = loadPluginViews();
    assert.ok(views.dirs.size >= 5);
    assert.ok(views.names.size >= 20);
  });
});

describe("a plugin id in production code", () => {
  const ids = ["md.example.views.alpha", "md.example.views.beta", "md.example.notes"];
  const found = (...values) => findPluginIdLiterals([{ file: "fixture.ts", literals: values.map((value) => ({ line: 1, value })) }], ids);

  it("takes its namespace from the ids", () => {
    assert.equal(getIdNamespace(ids), "md.example.");
  });

  it("is found as a whole id, as the start of one and inside a longer literal", () => {
    assert.equal(found("md.example.views.alpha").length, 1);
    assert.equal(found("md.example.views.").length, 1);
    assert.equal(found("md.example.").length, 1);
    assert.equal(found("plugins.md.example.views.beta.setting").length, 1);
  });

  it("is not found in a literal of the namespace that names no plugin", () => {
    assert.deepEqual(found("md.example.Application", "md.example.spike", "md.example.notesfoo", "other.example.views.alpha"), []);
  });
});
