import path from "node:path";
import { MODULES_DIR, exec } from "./helpers.mjs";

/** Runs every test of the package that holds the structural Go tests of the repository. */
export function verifyGoStructure() {
  exec("go test -count=1 ./container", path.join(MODULES_DIR, "libs/core"));
}

export const CORE_VERIFIERS = [
  {
    id: "go-structure",
    scope: "libs",
    module: "go",
    name: "Structural Go Tests of the Repository (every test of libs/core/container)",
    run: verifyGoStructure,
  },
  {
    id: "notes-hexagon",
    scope: "libs",
    module: "notes",
    name: "Notes Library Internal Hexagonal Layers (layers_test.go)",
    run: () => {
      exec("go test -count=1 .", path.join(MODULES_DIR, "libs/notes"));
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
