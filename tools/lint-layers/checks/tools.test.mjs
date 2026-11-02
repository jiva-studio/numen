import assert from "node:assert/strict";
import { describe, it } from "node:test";

import { GOLANGCI_LINT_VERSION, getLinterVersionGap, parseLinterVersion } from "./tools.mjs";

describe("the golangci-lint release", () => {
  const report = (version) => `golangci-lint has version ${version} built with go1.26.9 from 114493f on 2026-09-24T10:00:00Z`;
  it("reads the release with or without a v", () => {
    assert.equal(parseLinterVersion(report("2.14.0")), "2.14.0");
    assert.equal(parseLinterVersion(report("v2.14.0")), "2.14.0");
  });
  it("accepts the pinned release", () => {
    assert.equal(getLinterVersionGap(report(GOLANGCI_LINT_VERSION)), "");
  });
  it("fails an older and a newer release", () => {
    assert.match(getLinterVersionGap(report("2.13.2")), /golangci-lint is 2\.13\.2, the repository is linted with 2\.14\.0/);
    assert.match(getLinterVersionGap(report("2.15.0")), /golangci-lint is 2\.15\.0/);
  });
  it("fails output without a version", () => {
    assert.match(getLinterVersionGap("golangci-lint: command output"), /does not report its version/);
  });
});
