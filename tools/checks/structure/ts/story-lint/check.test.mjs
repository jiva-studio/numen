import assert from "node:assert/strict";
import { describe, it } from "node:test";
import { findStoryRuleGaps, readSeverity } from "./check.mjs";

const RECOMMENDED = {
  "react-hooks/rules-of-hooks": "off",
  "storybook/await-interactions": "error",
  "storybook/default-exports": "error",
  "storybook/prefer-pascal-case": "warn",
};

/** The rules of a story file that follows the recommendation, in the shape `eslint --print-config` prints. */
const resolve = () => ({
  "storybook/await-interactions": [2],
  "storybook/default-exports": [2],
  "storybook/prefer-pascal-case": [1],
  "storybook/csf-component": [0],
});

describe("the severity of a rule", () => {
  it("reads the printed array, the number and the name", () => {
    assert.deepEqual([[2], 2, "error", [1], "warn", [0], 0, "off", undefined].map(readSeverity), ["error", "error", "error", "warn", "warn", "off", "off", "off", "off"]);
  });
});

describe("the recommended story rules", () => {
  it("resolve without a gap when every rule matches and csf-component is off or absent", () => {
    assert.deepEqual(findStoryRuleGaps(resolve(), RECOMMENDED), []);
    const absent = resolve();
    delete absent["storybook/csf-component"];
    assert.deepEqual(findStoryRuleGaps(absent, RECOMMENDED), []);
  });

  it("report a rule that is missing, off or weaker", () => {
    const rules = resolve();
    delete rules["storybook/await-interactions"];
    rules["storybook/default-exports"] = [1];
    assert.deepEqual(findStoryRuleGaps(rules, RECOMMENDED), [
      "storybook/await-interactions is off, the plugin recommends error",
      "storybook/default-exports is warn, the plugin recommends error",
    ]);
  });

  it("report csf-component when it is on", () => {
    assert.deepEqual(findStoryRuleGaps({ ...resolve(), "storybook/csf-component": [1] }, RECOMMENDED), ["storybook/csf-component is on"]);
  });

  it("report a configuration that resolved nothing", () => {
    assert.deepEqual(findStoryRuleGaps(undefined, RECOMMENDED), ["no resolved rules"]);
  });
});
