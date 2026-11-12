import assert from "node:assert/strict";
import { describe, it } from "node:test";
import {
  GOLANGCI_LINT_VERSION,
  compareGoVersions,
  getLinterBuildGap,
  getLinterVersionGap,
  parseBuiltWithGo,
  parseGoDirective,
  parseLinterVersion,
} from "./golangci.mjs";
describe("the version of the linter", () => {
  const report = (go) => `golangci-lint has version 2.15.0 built with go${go} from (unknown) on (unknown)`;
  it("reads the Go version it was built with and the one the workspace declares", () => {
    assert.equal(parseBuiltWithGo(report("1.26.10")), "1.26.10");
    assert.equal(parseGoDirective("go 1.26.9\n\nuse (\n)\n"), "1.26.9");
  });
  it("compares versions by number", () => {
    assert.ok(compareGoVersions("1.26.10", "1.26.9") > 0);
    assert.ok(compareGoVersions("1.9", "1.26") < 0);
    assert.equal(compareGoVersions("1.26", "1.26.0"), 0);
  });
  it("fails a linter built with a Go older than the workspace", () => {
    assert.match(getLinterBuildGap(report("1.25.4"), "1.26.9"), /older than go1\.26\.9/);
    assert.match(getLinterBuildGap("golangci-lint has version 2.15.0", "1.26.9"), /does not report/);
  });
  it("accepts a linter built with the same or a newer Go", () => {
    assert.equal(getLinterBuildGap(report("1.26.9"), "1.26.9"), "");
    assert.equal(getLinterBuildGap(report("1.27.0"), "1.26.9"), "");
  });
});

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
