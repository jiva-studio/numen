import fs from "node:fs";
import path from "node:path";
import { MODULES_DIR, exec } from "./helpers.mjs";

/** Scans AST/parser architectural test files in `libs/core/container/*_test.go` for `func Test...(t *testing.T)`. */
export function discoverCoreHexagonTests() {
  const containerDir = path.join(MODULES_DIR, "libs/core/container");
  const archFiles = fs
    .readdirSync(containerDir)
    .filter((f) => f.endsWith("_test.go") && !f.startsWith("plugin") && f !== "chunking_test.go" && f !== "clock_port_test.go")
    .filter((f) => {
      const text = fs.readFileSync(path.join(containerDir, f), "utf8");
      return text.includes('"go/ast"') || text.includes('"go/parser"');
    })
    .sort();

  const names = [];
  for (const file of archFiles) {
    const text = fs.readFileSync(path.join(containerDir, file), "utf8");
    for (const match of text.matchAll(/^func (Test\w+)\(t \*testing\.T\)/gm)) {
      names.push(match[1]);
    }
  }
  return names;
}

export function verifyCoreHexagon() {
  const tests = discoverCoreHexagonTests();
  if (tests.length === 0) {
    throw new Error("No core hexagonal architecture tests discovered in libs/core/container/*_test.go");
  }
  exec(`go test -v ./container -run "^(${tests.join("|")})$"`, path.join(MODULES_DIR, "libs/core"));
}

export const CORE_VERIFIERS = [
  {
    id: "core-hexagon",
    scope: "libs",
    module: "core",
    name: "Core Hexagonal Layers & Inward Dependencies (layers_test.go)",
    run: verifyCoreHexagon,
  },
  {
    id: "machine-clock",
    scope: "libs",
    module: "machine-clock",
    name: "Machine Clock Read Only By A Clock Port Implementation (clock_port_test.go, all Go modules)",
    run: () => {
      exec(
        'go test -v ./container -count=1 -run "^(TestTheMachineClockIsReadOnlyByAClockPortImplementation|TestClockPortRule_RejectsReadsOutsideAClockImplementation|TestClockPortRule_RejectsAClockMethodWithoutAPort)$"',
        path.join(MODULES_DIR, "libs/core"),
      );
    },
  },
  {
    id: "notes-hexagon",
    scope: "libs",
    module: "notes",
    name: "Notes Library Internal Hexagonal Layers (layers_test.go)",
    run: () => {
      exec("go test -v . -run TestNotesInternalHexagonalLayers", path.join(MODULES_DIR, "libs/notes"));
    },
  },
  {
    id: "sdk-encapsulates-protocol",
    scope: "libs",
    module: "sdk",
    name: "Go Plugin SDK Encapsulates Protocol (no exported identifier of a public package mentions a type of libs/protocol/gen; sdk_protocol_encapsulation_test.go)",
    run: () => {
      exec("go test -v ./container -count=1 -run ^TestSDKProtocolRule_", path.join(MODULES_DIR, "libs/core"));
    },
  },
  {
    id: "core-depguard",
    tools: ["golangci-lint"],
    scope: "libs",
    module: "core",
    name: "Core Domain & UseCase Depguard Boundaries (.golangci.yml)",
    run: () => {
      exec("golangci-lint run --issues-exit-code 1 ./...", path.join(MODULES_DIR, "libs/core"));
    },
  },
  {
    id: "notes-depguard",
    tools: ["golangci-lint"],
    scope: "libs",
    module: "notes",
    name: "Notes Library Depguard Boundaries",
    run: () => {
      exec("golangci-lint run --issues-exit-code 1 ./...", path.join(MODULES_DIR, "libs/notes"));
    },
  },
  {
    id: "sdk-isolation",
    tools: ["golangci-lint"],
    scope: "libs",
    module: "sdk",
    name: "Plugin SDK Go Isolation (no core or app imports)",
    run: () => {
      exec("golangci-lint run --issues-exit-code 1 ./...", path.join(MODULES_DIR, "libs/plugin-sdk-go"));
    },
  },
];
