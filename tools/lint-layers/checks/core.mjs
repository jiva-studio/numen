import path from "node:path";
import { MODULES_DIR, exec } from "./helpers.mjs";

/** Writes plugins/plugins-index.json, which a structural test compares with the manifests; the file is not tracked. */
export function writePluginIndex() {
  exec("node tools/build-plugin-index/build.mjs");
}

/** Runs every test of the package that holds the structural Go tests of the repository. */
export function verifyGoStructure() {
  writePluginIndex();
  exec("go test -count=1 ./container", path.join(MODULES_DIR, "libs/core"));
}

/** A verifier that runs golangci-lint in the Go module at `dir`. */
const createDepguardVerifier = ({ id, module, name, dir }) => ({
  id,
  isLint: true,
  tools: ["golangci-lint"],
  scope: "libs",
  module,
  name,
  run: () => {
    exec("golangci-lint run --issues-exit-code 1 ./...", path.join(MODULES_DIR, dir));
  },
});

export const CORE_VERIFIERS = [
  {
    id: "go-structure",
    isGoTest: true,
    scope: "libs",
    module: "go",
    name: "Structural Go Tests of the Repository (every test of libs/core/container)",
    run: verifyGoStructure,
  },
  {
    id: "notes-hexagon",
    isGoTest: true,
    scope: "libs",
    module: "notes",
    name: "Notes Library Internal Hexagonal Layers (layers_test.go)",
    run: () => {
      exec("go test -count=1 .", path.join(MODULES_DIR, "libs/notes"));
    },
  },
  createDepguardVerifier({ id: "core-depguard", module: "core", dir: "libs/core",
    name: "Core Domain & UseCase Depguard Boundaries (.golangci.yml)" }),
  createDepguardVerifier({ id: "notes-depguard", module: "notes", dir: "libs/notes",
    name: "Notes Library Depguard Boundaries" }),
  createDepguardVerifier({ id: "sdk-isolation", module: "sdk", dir: "libs/plugin-sdk-go",
    name: "Plugin SDK Go Isolation (no core or app imports)" }),
];
