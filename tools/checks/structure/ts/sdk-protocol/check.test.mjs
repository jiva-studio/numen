import assert from "node:assert/strict";
import fs from "node:fs";
import os from "node:os";
import path from "node:path";
import { test } from "node:test";
import ts from "typescript";
import { findProtocolExposures, verifySdkTypesEncapsulateProtocol } from "./check.mjs";

const PROTOCOL_FILES = {
  "protocol/package.json": JSON.stringify({ name: "proto", main: "./index.ts" }),
  "protocol/index.ts": [
    "export interface Message { text: string }",
    "export enum Level { Low, High }",
    "export class Client { send(): void {} }",
    "",
  ].join("\n"),
};

const SDK_PACKAGE = JSON.stringify({ name: "sdk", exports: { ".": "./index.ts" } });

/** Writes the files into a new directory and returns the findings for the SDK entries in it. */
function readExposures(sdkSource, sdkFiles = { "sdk/package.json": SDK_PACKAGE }) {
  // TypeScript reports real paths, and the temporary directory of macOS is a symbolic link.
  const root = fs.realpathSync(fs.mkdtempSync(path.join(os.tmpdir(), "sdk-protocol-")));
  try {
    const files = { ...PROTOCOL_FILES, ...sdkFiles, "sdk/index.ts": sdkSource };
    for (const [name, text] of Object.entries(files)) {
      fs.mkdirSync(path.dirname(path.join(root, name)), { recursive: true });
      fs.writeFileSync(path.join(root, name), text);
    }
    const options = {
      target: ts.ScriptTarget.ES2022,
      module: ts.ModuleKind.ESNext,
      moduleResolution: ts.ModuleResolutionKind.Bundler,
      strict: true,
      noEmit: true,
      allowImportingTsExtensions: true,
      baseUrl: root,
      paths: { proto: ["protocol/index.ts"] },
    };
    return findProtocolExposures({
      packageDir: path.join(root, "sdk"),
      protocolDir: path.join(root, "protocol"),
      options,
      rootDir: root,
    });
  } finally {
    fs.rmSync(root, { recursive: true, force: true });
  }
}

const getSubjects = (result) => [...new Set(result.findings.map((one) => one.symbol.split(" mentions ")[0]))].sort();

test("every exported position of a protocol declaration is reported", () => {
  const result = readExposures(
    [
      "import type { Message, Level } from 'proto'",
      "import { Client } from 'proto'",
      "export type { Message } from 'proto'",
      "export function send(message: Message): void {}",
      "export function level(): Level { return Level.Low }",
      "export interface Envelope { body: Message }",
      "export const client = new Client()",
      "export function many(): Promise<Message[]> { return Promise.resolve([]) }",
      "export type Table = Record<string, Message>",
      "export class Holder { value!: Message }",
      "interface Hidden { inner: Message }",
      "export function reach(): Hidden { return { inner: { text: '' } } }",
      "",
    ].join("\n"),
  );
  assert.deepEqual(getSubjects(result), ["Envelope", "Hidden", "Holder", "Message", "Table", "client", "level", "many", "send"]);
  assert.ok(result.findings.every((one) => one.file === "sdk/index.ts" && one.line >= 1));
  assert.ok(result.findings.some((one) => /declared in index\.ts$/.test(one.symbol)));
  assert.ok(result.protocolFiles >= 1);
});

test("declarations that name no protocol declaration in their type are accepted", () => {
  const result = readExposures(
    [
      "import type { Message } from 'proto'",
      "function internal(message: Message): string { return message.text }",
      "export function text(value: string): Promise<string[]> { return Promise.resolve([value]) }",
      "export interface Own { name: string; run(count: number): Own }",
      "export const render = (message: string) => internal({ text: message })",
      "",
    ].join("\n"),
  );
  assert.deepEqual(result.findings, []);
  assert.equal(result.exported, 3);
  assert.ok(result.protocolFiles >= 1);
});

test("a protocol declaration exposed by an entry besides the main one is reported", () => {
  const result = readExposures("export const name = 'sdk'\n", {
    "sdk/package.json": JSON.stringify({ name: "sdk", exports: { ".": "./index.ts", "./testing": { types: "./testing.ts" }, "./styles.css": "./styles.css" } }),
    "sdk/testing.ts": "import type { Message } from 'proto'\nexport function fake(): Message { return { text: '' } }\n",
  });
  assert.equal(result.entries, 2);
  assert.deepEqual(getSubjects(result), ["fake"]);
});

test("the plugin SDK of this repository reads enough declarations and exposes no protocol declaration", () => {
  assert.doesNotThrow(verifySdkTypesEncapsulateProtocol);
});
