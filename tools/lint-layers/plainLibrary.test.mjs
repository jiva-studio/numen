import assert from "node:assert/strict";
import fs from "node:fs";
import os from "node:os";
import path from "node:path";
import { after, describe, it } from "node:test";
import { Linter } from "eslint";
import { createPlainLibraryConfig } from "./base-eslint.mjs";

const project = fs.mkdtempSync(path.join(os.tmpdir(), "lint-layers-plain-library-"));
fs.writeFileSync(
  path.join(project, "tsconfig.json"),
  JSON.stringify({ compilerOptions: { lib: ["ES2022", "DOM"], strict: true, target: "ES2022", module: "ESNext", skipLibCheck: true }, include: ["**/*.ts"] }),
);
after(() => fs.rmSync(project, { recursive: true, force: true }));

/** The lines the factory size rule reports in a file planted at `name` in a plain library. */
function findFactoryReports(name, source) {
  const file = path.join(project, name);
  fs.mkdirSync(path.dirname(file), { recursive: true });
  fs.writeFileSync(file, source);
  return new Linter({ cwd: project })
    .verify(source, createPlainLibraryConfig(project), { filename: file })
    .filter((message) => message.ruleId === "numen/max-composable-return-properties")
    .map((message) => `${message.line} ${message.messageId}`);
}

const NINE_FUNCTIONS = [
  "const run = (): number => 1",
  "export function createRunner() {",
  "  return { a: run, b: run, c: run, d: run, e: run, f: run, g: run, h: run, i: run }",
  "}",
].join("\n");

describe("the factory size rule in a plain library", () => {
  it("fails a factory that returns nine functions", () => {
    assert.deepEqual(findFactoryReports("src/runner.ts", NINE_FUNCTIONS), ["3 tooManyReturnedProperties"]);
  });

  it("passes the same factory under testing/ and fixtures/", () => {
    assert.deepEqual(findFactoryReports("src/testing/runner.ts", NINE_FUNCTIONS), []);
    assert.deepEqual(findFactoryReports("src/fixtures/runner.ts", NINE_FUNCTIONS), []);
  });
});
