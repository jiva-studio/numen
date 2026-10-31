import assert from "node:assert/strict";
import { describe, it } from "node:test";
import { parseManifest } from "../../shared/manifests.mjs";
import { findIdentifierLiterals, getIdNamespace, listPluginIdentifiers, verifyPluginIdentifiersOnlyInManifests } from "./identifiers.mjs";

const MANIFESTS = {
  alpha: {
    id: "md.example.views.alpha",
    name: "Alpha",
    version: "1.0.0",
    services: ["example.alpha"],
    commands: [{ id: "make-gadget", title: "Make" }, { id: "alpha.review", title: "Review" }],
    contributes: { tabKinds: [{ kind: "gadget" }, { kind: "reader:alpha-reader" }], noteTypes: [{ type: "gadget" }], readers: [{ id: "shared.command" }] },
  },
  beta: {
    id: "md.example.views.beta",
    name: "Beta",
    version: "1.0.0",
    commands: [{ id: "shared.command", title: "Shared" }],
    contributes: { tabKinds: [{ kind: "gadget" }] },
  },
  notes: { id: "md.example.notes", name: "Notes", version: "1.0.0", contributes: { tabKinds: [{ kind: "note" }] } },
};

const manifests = Object.entries(MANIFESTS).map(([name, manifest]) => ({
  name,
  manifest: parseManifest(JSON.stringify(manifest), `plugins/${name}/manifest.json`).document,
}));
const identifiers = listPluginIdentifiers(manifests);
const tokens = identifiers.map(({ token }) => token).sort();

describe("the identifiers a manifest declares", () => {
  it("are the plugin ids and the namespaced identifiers", () => {
    assert.deepEqual(tokens, [
      "alpha.review",
      "example.alpha",
      "md.example.notes",
      "md.example.views.alpha",
      "md.example.views.beta",
      "reader:alpha-reader",
    ]);
  });

  it("leave out a bare word, which is also a word of the host", () => {
    for (const word of ["gadget", "note", "make-gadget"]) assert.equal(tokens.includes(word), false, word);
  });

  it("leave out an identifier that two plugins declare", () => {
    assert.equal(tokens.includes("shared.command"), false);
  });

  it("name the plugin that declares them", () => {
    assert.equal(identifiers.find(({ token }) => token === "example.alpha").owner, "alpha");
    assert.equal(identifiers.find(({ token }) => token === "md.example.notes").isPluginId, true);
  });

  it("give the namespace the plugin ids share", () => {
    assert.equal(getIdNamespace(identifiers.filter(({ isPluginId }) => isPluginId).map(({ token }) => token)), "md.example.");
  });
});

describe("a source that states an identifier", () => {
  const host = (...values) => [{ file: "libs/core/a.go", plugin: null, literals: values.map((value) => ({ line: 7, value })) }];
  const plugin = (name, ...values) => [{ file: `plugins/${name}/a.ts`, plugin: name, literals: values.map((value) => ({ line: 3, value })) }];

  it("is found in the host as a whole plugin id, as the start of one and inside a longer literal", () => {
    for (const value of ["md.example.views.alpha", "md.example.views.", "md.example.", "plugins.md.example.views.beta.setting"]) {
      assert.equal(findIdentifierLiterals(host(value), identifiers).length, 1, value);
    }
  });

  it("is found in the host as a namespaced identifier of a plugin", () => {
    const found = findIdentifierLiterals(host("reader:alpha-reader", "example.alpha"), identifiers);
    assert.equal(found.length, 2);
    assert.equal(found[0].line, 7);
  });

  it("is not found in a literal that names no plugin", () => {
    assert.deepEqual(findIdentifierLiterals(host("md.example.Application", "md.example.spike", "gadget", "note", "other.example.views.alpha"), identifiers), []);
  });

  it("is found in another plugin and not in the plugin that declares it", () => {
    assert.equal(findIdentifierLiterals(plugin("beta", "md.example.views.alpha"), identifiers).length, 1);
    assert.equal(findIdentifierLiterals(plugin("beta", "alpha.review"), identifiers).length, 1);
    assert.deepEqual(findIdentifierLiterals(plugin("alpha", "md.example.views.alpha", "alpha.review"), identifiers), []);
  });

  it("is not found in another plugin as a prefix of an id", () => {
    assert.deepEqual(findIdentifierLiterals(plugin("beta", "md.example.views."), identifiers), []);
  });
});

describe("the live repository", () => {
  it("states no plugin identifier outside the manifests", () => {
    assert.doesNotThrow(verifyPluginIdentifiersOnlyInManifests);
  });
});
