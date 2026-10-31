import assert from "node:assert/strict";
import fs from "node:fs";
import os from "node:os";
import path from "node:path";
import { describe, it } from "node:test";
import { loadManifestWords } from "./dictionary.mjs";
import { findPluginViewsInUi, loadPluginViews } from "./frontend.mjs";

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
