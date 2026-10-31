import { spawnSync } from "node:child_process";
import fs from "node:fs";
import path from "node:path";
import {
  MODULES_DIR,
  REPO_ROOT,
  WINDOW_IDS,
  exec,
  failOnFindings,
  getGoReferences,
  listShippedFiles,
  readSource,
} from "./helpers.mjs";

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

/** Every file in `libs/wire` and every reference to `@numen/wire` across the repository. */
function findWireLibraryReferences() {
  const found = [];
  if (fs.existsSync(path.join(REPO_ROOT, "libs/wire"))) {
    found.push({ file: "libs/wire", line: 1, symbol: "directory exists (must be removed)" });
  }
  const grep = spawnSync("git", ["grep", "-n", "@numen/wire", "--", ":!tools/lint-layers/*"], {
    cwd: REPO_ROOT,
    encoding: "utf8",
  });
  for (const line of (grep.stdout || "").split("\n").filter(Boolean)) {
    const [file, lineNum, ...rest] = line.split(":");
    found.push({ file, line: Number(lineNum) || 1, symbol: rest.join(":").trim() });
  }
  return found;
}

/** The `@numen/wire` library does not exist; generic utilities live in `@numen/plugin-sdk` and core protocol adapters live in the host. */
export function verifyNoWireLibrary() {
  failOnFindings("The @numen/wire library still exists or is referenced", findWireLibraryReferences());
}

/** Every exported declaration in `libs/plugin-sdk-go` or `libs/plugin-sdk-ts` that leaks protocol/notes types or window ids. */
function findSdkProtocolLeaks() {
  const found = [];
  for (const file of listShippedFiles(["libs/plugin-sdk-go/*.go"])) {
    const { code } = readSource(file);
    const lines = code.split("\n");
    lines.forEach((line, idx) => {
      const isExportedFuncOrType =
        /^func\s+(?:\([^)]*\)\s+)?[A-Z]\w*/.test(line) ||
        /^type\s+[A-Z]\w*\s+(?!=\s)/.test(line) ||
        /^\s+[A-Z]\w*\s+.*\b(?:numenv1|v1|numenv1connect|domain|edit|markdown)\.[A-Z]\w*/.test(line);
      if (isExportedFuncOrType) {
        for (const m of line.matchAll(/\b((?:numenv1|v1|numenv1connect|domain|edit|markdown)\.[A-Z]\w*)/g)) {
          found.push({ file, line: idx + 1, symbol: m[1] });
        }
      }
    });
  }
  for (const file of listShippedFiles(["libs/plugin-sdk-ts/src"])) {
    const text = fs.readFileSync(path.join(REPO_ROOT, file), "utf8");
    for (const match of text.matchAll(/^\s*export\s[^;]*?\bfrom\s*(['"])(@numen\/protocol(?:\/[^'"]*)?)\1/gm)) {
      const line = text.slice(0, match.index).split("\n").length;
      found.push({ file, line, symbol: `export from ${JSON.stringify(match[2])}` });
    }
    const source = readSource(file);
    if (!source) continue;
    for (const { line, value } of source.literals) {
      if (value === "@numen/wire" || value.startsWith("@numen/wire/") || WINDOW_IDS.includes(value)) {
        found.push({ file, line, symbol: JSON.stringify(value) });
      }
    }
  }
  return found;
}

/** The plugin SDK encapsulates `libs/protocol` and `libs/notes` and exposes only SDK types. */
export function verifySdkEncapsulatesProtocol() {
  failOnFindings("The plugin SDK leaks protocol/notes types or host window IDs in its public contract", findSdkProtocolLeaks());
}

/** Only plugins/*, apps/*, libs/plugin-host-ts, and the SDKs themselves may import libs/plugin-sdk-go or @numen/plugin-sdk. */
function findUnauthorizedSdkImports() {
  const found = [];
  for (const file of listShippedFiles(["libs/core", "libs/notes", "libs/protocol", "libs/ui"])) {
    if (file.endsWith(".go")) {
      const text = fs.readFileSync(path.join(REPO_ROOT, file), "utf8");
      const { imports } = getGoReferences(text);
      for (const { line, value } of imports) {
        if (value.startsWith("github.com/jiva-studio/numen/libs/plugin-sdk-go")) {
          found.push({ file, line, symbol: `import "${value}"` });
        }
      }
      continue;
    }
    const source = readSource(file);
    if (!source) continue;
    for (const { line, value } of source.literals) {
      if (value === "@numen/plugin-sdk" || value.startsWith("@numen/plugin-sdk/")) {
        found.push({ file, line, symbol: JSON.stringify(value) });
      }
    }
  }
  return found;
}

export function verifySdkImportedOnlyByPluginsAndApps() {
  failOnFindings("Only plugins/*, apps/*, and libs/plugin-host-ts may import the plugin SDK", findUnauthorizedSdkImports());
}

/** Package-level mutable singletons (sync.Mutex, sync.Once, atomic.*, mutable vars) and os.Exit outside cmd/. */
function findGoMutableGlobalsAndExits() {
  const found = [];
  for (const file of listShippedFiles(["libs/core", "libs/notes", "libs/plugin-sdk-go", "plugins", "apps/desktop"])) {
    if (!file.endsWith(".go") || file.endsWith(".pb.go") || file.endsWith(".connect.go")) continue;
    const { code } = readSource(file);
    const lines = code.split("\n");
    const isCmd = /\/cmd\//.test(file) || /\/main\.go$/.test(file);
    let inVarBlock = false;
    lines.forEach((line, idx) => {
      if (/^var\s*\(\s*$/.test(line)) {
        inVarBlock = true;
        return;
      }
      if (inVarBlock && /^\)/.test(line)) {
        inVarBlock = false;
        return;
      }
      if (
        /^var\s+\w+\s+(?:sync\.(?:Mutex|RWMutex|Once|Map)|atomic\.\w+)/.test(line) ||
        (inVarBlock && /^\t\w+\s+(?:sync\.(?:Mutex|RWMutex|Once|Map)|atomic\.\w+)/.test(line))
      ) {
        found.push({ file, line: idx + 1, symbol: `mutable global synchronization state: ${line.trim()}` });
      }
      if (/^var\s+newHub\s*=/.test(line.trim())) {
        found.push({ file, line: idx + 1, symbol: `mutable package-level hook: ${line.trim()}` });
      }
      if (!isCmd && /\b(os\.Exit|log\.Fatal(?:f|ln)?)\s*\(/.test(line)) {
        found.push({ file, line: idx + 1, symbol: `process termination outside cmd/: ${line.trim()}` });
      }
    });
  }
  return found;
}

export function verifyGoNoMutableGlobalsOrExit() {
  failOnFindings("Go package-level mutable state or os.Exit outside cmd/", findGoMutableGlobalsAndExits());
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
    id: "sdk-consumers",
    scope: "libs",
    module: "sdk",
    name: "Plugin SDK Imported Only By Plugins and Applications",
    run: verifySdkImportedOnlyByPluginsAndApps,
  },
  {
    id: "go-no-mutable-globals",
    scope: "libs",
    module: "core",
    name: "Go No Package-Level Mutable Singletons or os.Exit Outside cmd/",
    run: verifyGoNoMutableGlobalsOrExit,
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
  {
    id: "sdk-encapsulates-protocol",
    scope: "libs",
    module: "sdk",
    name: "Plugin SDK Encapsulates Protocol (no numenv1.* or window IDs in public SDK API)",
    run: verifySdkEncapsulatesProtocol,
  },
  {
    id: "no-wire-library",
    scope: "libs",
    module: "wire",
    name: "No @numen/wire Library (libs/wire removed; utilities in @numen/plugin-sdk, protocol adapters in host)",
    run: verifyNoWireLibrary,
  },
];
