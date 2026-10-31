import assert from "node:assert/strict";
import { test } from "node:test";
import { readGo, readScript } from "./helpers.mjs";
import { findManifestParsers } from "./manifests.mjs";

const script = (file, text) => ({ file, source: readScript(text), text });
const go = (file, text) => ({ file, source: readGo(text), text });

test("a script that reads manifest.json and parses it itself is reported with the line", () => {
  const found = findManifestParsers([
    script("tools/a.mjs", "const text = read('plugins/x/manifest.json')\nconst data = JSON.parse(text)\n"),
  ]);
  assert.equal(found.length, 1);
  assert.equal(found[0].line, 2);
  assert.match(found[0].symbol, /JSON\.parse/);
});

test("a script that decodes a response in a file naming manifest.json is reported", () => {
  const found = findManifestParsers([
    script("libs/h.ts", "const res = await get(`${base}/manifest.json`)\nconst body = await res.json()\n"),
  ]);
  assert.equal(found.length, 1);
  assert.match(found[0].symbol, /\.json\(\)/);
});

test("a script that names the file through a constant and parses is reported", () => {
  const found = findManifestParsers([script("tools/b.mjs", "const file = join(dir, MANIFEST_FILE)\nJSON.parse(read(file))\n")]);
  assert.equal(found.length, 1);
});

test("a script that decodes through the generated schema is the reader and is clean", () => {
  const found = findManifestParsers([
    script("tools/shared/m.mjs", "const FILE = 'manifest.json'\nconst d = JSON.parse(t)\nfromJson(ManifestSchema, d)\n"),
  ]);
  assert.deepEqual(found, []);
});

test("a script that parses JSON without naming manifest.json is clean", () => {
  assert.deepEqual(findManifestParsers([script("tools/c.mjs", "JSON.parse(read('package.json'))\n")]), []);
});

test("a manifest.json named only in a comment is not read", () => {
  assert.deepEqual(findManifestParsers([script("tools/d.mjs", "// the manifest.json is read elsewhere\nJSON.parse(x)\n")]), []);
});

test("a Go file that reads manifest.json and decodes it with encoding/json is reported", () => {
  const found = findManifestParsers([
    go("libs/core/a.go", 'package a\n\nfunc f(raw []byte) {\n\tname := "manifest.json"\n\tjson.Unmarshal(raw, &m)\n}\n'),
  ]);
  assert.equal(found.length, 1);
  assert.match(found[0].symbol, /json\.Unmarshal/);
});

test("a Go file that decodes the generated message with protojson is the reader and is clean", () => {
  const found = findManifestParsers([
    go("libs/plugin-catalog/manifestjson/d.go", 'package m\n\nfunc f() {\n\tname := "manifest.json"\n\tvar wire v1.Manifest\n\tprotojson.Unmarshal(b, &wire)\n}\n'),
  ]);
  assert.deepEqual(found, []);
});

test("a Go file that decodes JSON without naming manifest.json is clean", () => {
  assert.deepEqual(findManifestParsers([go("libs/core/b.go", "package b\n\nfunc f() {\n\tjson.Unmarshal(raw, &m)\n}\n")]), []);
});
