import { spawnSync } from "node:child_process";
import fs from "node:fs";
import path from "node:path";
import {
  MODULES_DIR,
  NOT_SHIPPED,
  REPO_ROOT,
  WINDOW_IDS,
  exec,
  failOnFindings,
  findInCode,
  getGoReferences,
  getLiteralValues,
  listFiles,
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

/** The packages of the core that belong to a plugin, as directories, and as files of `container`. */
const CORE_MOVED_DIRS = [
  "usecase/document",
  "usecase/recognition",
  "usecase/transcription",
  "usecase/embedding",
  "internal/embedding",
  "internal/ocrartifact",
  "internal/transcript",
  "port/transcriptionport",
  "internal/adapter/plugin/transcriberbridge",
  "internal/adapter/plugin/proofreaderbridge",
  "internal/urlfile",
  "internal/adapter/download",
  "internal/adapter/plugin/downloaderbridge",
  "port/downloadport",
  "internal/correction",
  "adapter/window/flashcards",
  "internal/adapter/plugin/toolsbridge",
  "usecase/plugintool",
];
const CORE_MOVED_CONTAINER_FILES = ["recogniser", "transcriber", "proofreader", "embedder", "downloader"];

/** The core holds none of the packages a domain plugin owns. */
export function verifyCoreHoldsNoPluginPackage() {
  const found = [];
  for (const dir of CORE_MOVED_DIRS) {
    const files = listFiles(path.join("libs/core", dir));
    if (files.length > 0) found.push(`libs/core/${dir}/ (${files.length} files)`);
  }
  const container = listFiles("libs/core/container").filter((f) => path.dirname(f) === "libs/core/container");
  for (const name of CORE_MOVED_CONTAINER_FILES) {
    const files = container.filter((f) => new RegExp(`/${name}(_\\w+)?\\.go$`).test(f));
    if (files.length > 0) found.push(`libs/core/container/${name}*.go: ${files.map((f) => path.basename(f)).join(", ")}`);
  }
  if (found.length > 0) {
    throw new Error("The core holds packages a plugin owns:\n" + found.join("\n"));
  }
}

/**
 * The protos of the domains plugins own. Every one generates into the shared `numen/v1` package.
 * `recording.proto` and `embedding.proto` are the core's own contracts.
 */
const DOMAIN_PROTOS = [
  "transcript",
  "ocr",
  "article",
  "artifact",
  "cards",
  "presets",
  "flashcards",
  "transcription",
  "download",
  "proofreading",
];
const PROTOCOL_GEN = "github.com/jiva-studio/numen/libs/protocol/gen/numen/v1";

/** The exported top-level names a generated Go file declares. */
function readGoExports(file) {
  if (!fs.existsSync(file)) return [];
  const names = [];
  let block = "";
  for (const line of fs.readFileSync(file, "utf8").split("\n")) {
    const open = line.match(/^(const|var) \($/);
    if (open) {
      block = open[1];
      continue;
    }
    if (block && line === ")") {
      block = "";
      continue;
    }
    const member = block && line.match(/^\t([A-Z]\w*)\b/);
    if (member) names.push(member[1]);
    const top = line.match(/^(?:type|func|const|var) ([A-Z]\w*)/);
    if (top) names.push(top[1]);
  }
  return names;
}

/** Which domain proto each generated name comes from, by package: `v1` and `numenv1connect`. */
function readDomainSymbols() {
  const gen = path.join(REPO_ROOT, "libs/protocol/gen/numen/v1");
  const symbols = { [PROTOCOL_GEN]: new Map(), [PROTOCOL_GEN + "/numenv1connect"]: new Map() };
  for (const proto of DOMAIN_PROTOS) {
    for (const name of readGoExports(path.join(gen, proto + ".pb.go"))) {
      symbols[PROTOCOL_GEN].set(name, proto);
    }
    for (const name of readGoExports(path.join(gen, "numenv1connect", proto + ".connect.go"))) {
      symbols[PROTOCOL_GEN + "/numenv1connect"].set(name, proto);
    }
  }
  return symbols;
}

/** The core names no message, enum or service of a domain proto. */
export function verifyCoreImportsNoDomainProto() {
  const symbols = readDomainSymbols();
  const files = spawnSync("git", ["ls-files", "--", "libs/core/*.go"], { cwd: REPO_ROOT, encoding: "utf8" })
    .stdout.split("\n")
    .filter((f) => f && fs.existsSync(path.join(REPO_ROOT, f)));
  const found = [];
  const byProto = new Map();
  for (const file of files) {
    const text = fs.readFileSync(path.join(REPO_ROOT, file), "utf8");
    const aliases = new Map();
    for (const [, alias, pkg] of text.matchAll(/^\s*(?:import\s+)?(\w+\s+)?"([^"]+)"\s*$/gm)) {
      if (!symbols[pkg]) continue;
      aliases.set(alias ? alias.trim() : path.basename(pkg) === "v1" ? "numenv1" : path.basename(pkg), symbols[pkg]);
    }
    if (aliases.size === 0) continue;
    text.split("\n").forEach((line, i) => {
      const hits = new Set();
      for (const [, alias, name] of line.matchAll(/\b(\w+)\.([A-Z]\w*)\b/g)) {
        const proto = aliases.get(alias)?.get(name);
        if (!proto) continue;
        hits.add(`${name} (${proto}.proto)`);
        byProto.set(proto, (byProto.get(proto) ?? 0) + 1);
      }
      if (hits.size > 0) found.push(`${file}:${i + 1}: ${[...hits].join(", ")}`);
    });
  }
  if (found.length > 0) {
    const summary = [...byProto].sort().map(([proto, count]) => `  ${proto}.proto: ${count}`);
    throw new Error("The core names domain protos:\n" + summary.join("\n") + "\n" + found.join("\n"));
  }
}

/** The texts made from a file and the files a plugin keeps, named by the plugin that makes them. */
/**
 * The services, enums, values, messages and calls of `numen.v1` that a single plugin owns.
 */
const PLUGIN_PROTO_SYMBOLS = [
  "TranscriptService",
  "ArticleService",
  "BookService",
  "DocumentService",
  "RecognitionService",
  "OcrEngineService",
  "TranscriptionEngineService",
  "ProofreadingService",
  "DownloadService",
  "CardsService",
  "PresetsService",
  "FlashcardsService",
  "SOURCE_KIND_BOOK",
  "SOURCE_KIND_RECORDING",
  "SOURCE_KIND_URL",
  "BookFormat",
  "FILE_MENU_ACTION_RECOGNISE",
  "FILE_MENU_ACTION_PROOFREAD",
  "BookProgress",
  "RecordingProgress",
  "SetVaultInUse",
  "SetVaultInUseRequest",
  "SetVaultInUseResponse",
  "NoteType",
  "NOTE_TYPE_CARD_MODEL",
  "NOTE_TYPE_PRESET",
  "NOTE_TYPE_DECK",
];

/** The fixtures in `libs/protocol/testdata` that belong to a plugin. */
const PLUGIN_PROTOCOL_FIXTURES = ["presets.json", "templates.json", "spine.html"];

/** The fields of `agent.proto` that name a card of the flashcards plugin. */
const PLUGIN_AGENT_FIELDS = ["mark", "face"];

/** Every name of PLUGIN_PROTO_SYMBOLS and field of PLUGIN_AGENT_FIELDS in the code of `numen.v1`. */
function findPluginProtoSymbols() {
  const symbols = new RegExp(`\\b(${PLUGIN_PROTO_SYMBOLS.join("|")})\\b`, "g");
  const fields = new RegExp(`^\\s*(?:repeated\\s+|optional\\s+)?[\\w.]+\\s+(${PLUGIN_AGENT_FIELDS.join("|")})\\s*=\\s*\\d+`, "g");
  const found = [];
  for (const file of listShippedFiles(["libs/protocol/proto/numen/v1/*.proto"])) {
    const { code } = readSource(file);
    for (const hit of findInCode(code, symbols)) found.push({ file, ...hit });
    if (path.basename(file) !== "agent.proto") continue;
    for (const hit of findInCode(code, fields)) found.push({ file, line: hit.line, symbol: `field ${hit.symbol}` });
  }
  for (const fixture of PLUGIN_PROTOCOL_FIXTURES) {
    const rel = `libs/protocol/testdata/${fixture}`;
    if (fs.existsSync(path.join(REPO_ROOT, rel))) {
      found.push({ file: rel, line: 1, symbol: `plugin fixture ${fixture}` });
    }
  }
  return found;
}

/** `numen.v1` holds the contracts of the core and its extension points, and nothing one plugin owns. */
export function verifyProtocolHoldsNoPluginContract() {
  failOnFindings("numen.v1 holds what a plugin owns", findPluginProtoSymbols());
}

/** The shared contracts: what every plugin and the core are built against. */
const CONTRACT_PATHS = [
  "libs/protocol/proto",
  "libs/protocol/capabilities.go",
  "libs/protocol/capabilities.ts",
  "libs/plugin-sdk-go",
  "libs/plugin-sdk-ts/src",
  "libs/plugin-host-ts/src",
  "libs/notes",
];

/** The ids, products and formats of plugins, and the keys only one plugin writes. */
const PLUGIN_TERMS = [
  "md\\.numen\\.(?:views|readers|agents|engines|tools|notes)",
  "flashcards?",
  "epubs?",
  "pdfs?",
  "ocr",
  "cfi",
  "highlightbox",
  "antigravity",
  "codex",
  "claude",
  "gemini",
  "sonnet",
  "gpt-?[0-9][a-z0-9.-]*",
  "proofread[a-z]*",
  "transcri[a-z]+",
  "recognition",
].map((term) => new RegExp(`(?<![\\w-])(?:${term})(?![\\w-])`, "i"));

/** Every line of a shared contract, comments and documents included, that names a plugin. */
function findPluginNamesInContracts() {
  const listed = spawnSync("git", ["ls-files", "--", ...CONTRACT_PATHS], { cwd: REPO_ROOT, encoding: "utf8" });
  const found = [];
  for (const file of (listed.stdout || "").split("\n").filter(Boolean)) {
    if (NOT_SHIPPED.test(file) || /_pb\.ts$|(^|\/)go\.(mod|sum)$/.test(file)) continue;
    if (!fs.existsSync(path.join(REPO_ROOT, file))) continue;
    fs.readFileSync(path.join(REPO_ROOT, file), "utf8")
      .split("\n")
      .forEach((text, index) => {
        const hit = PLUGIN_TERMS.map((term) => text.match(term)).find(Boolean);
        if (hit) found.push({ file, line: index + 1, symbol: hit[0] });
      });
  }
  return found;
}

/** The schema, the SDKs, the plugin host and the notes library name no plugin, and hold no model a plugin owns. Comments are read. */
export function verifyContractsNameNoPlugin() {
  failOnFindings("Shared contracts name a plugin or its model", findPluginNamesInContracts());
}

/** The code that knows no file format and no source kind. */
const FILE_KIND_FREE_PATHS = [
  "libs/core",
  "libs/notes",
  "libs/plugin-sdk-go",
  "libs/plugin-sdk-ts",
  "libs/ui/core",
  "libs/ui/shell",
  "apps/desktop/editor/src",
];

/** The source kinds plugins own. */
const PLUGIN_SOURCE_KINDS = ["book", "recording", "url", "document"];

/** A literal that is a source kind: `"book"`. */
const SOURCE_KIND_LITERAL = new RegExp(`^(?:${PLUGIN_SOURCE_KINDS.join("|")})$`);

/** The names a source kind takes in code. */
const SOURCE_KIND_NAMES = [
  "\\b(\\w*(?:Kind|Tab|Capability)(?:Book|Recording|URL|Url|Document))\\b",
  "\\b(\\w*SOURCE_KIND_(?:BOOK|RECORDING|URL|DOCUMENT)|\\w*BOOK_FORMAT_\\w+)\\b",
  "\\b(\\w*Kinds?\\.(?:BOOK|RECORDING|URL|DOCUMENT))\\b",
  "\\b(\\w*(?:Book|Document)Formats?\\w*)\\b",
  "\\b(\\w*(?:Book|Recording)(?:Extensions|Progress)\\w*)\\b",
  "\\b(\\w*Books\\w*)\\b",
  "\\b(\\w*(?:Book|Document|Transcript|Article)Service\\w*)\\b",
].map((pattern) => new RegExp(pattern, "g"));

/** The constants a script names a source kind by. */
const SCRIPT_SOURCE_KIND_NAMES = /(?<![.\w])(BOOK|RECORDING|DOCUMENT)\b(?![\w(])/g;

/** Every source kind literal and source kind name in FILE_KIND_FREE_PATHS. */
function findFileKinds() {
  const found = [];
  for (const file of listShippedFiles(FILE_KIND_FREE_PATHS)) {
    const source = readSource(file);
    if (!source) continue;
    for (const { line, value } of getLiteralValues(source.literals)) {
      if (SOURCE_KIND_LITERAL.test(value)) {
        found.push({ file, line, symbol: JSON.stringify(value) });
      }
    }
    const patterns = /\.go$/.test(file) ? SOURCE_KIND_NAMES : [...SOURCE_KIND_NAMES, SCRIPT_SOURCE_KIND_NAMES];
    for (const pattern of patterns) {
      for (const hit of findInCode(source.code, pattern)) found.push({ file, ...hit });
    }
  }
  return found;
}

/** The core, the notes library, the SDKs, the UI libraries and the window name no file format and no source kind. */
export function verifyNoFileKindOutsidePlugins() {
  failOnFindings("File formats and source kinds outside plugins", findFileKinds());
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
    id: "core-holds-no-plugin-package",
    scope: "libs",
    module: "core",
    name: "Core Holds No Plugin Package (document, recognition, transcription, links, proofreading, embedding)",
    run: verifyCoreHoldsNoPluginPackage,
  },
  {
    id: "core-imports-no-domain-proto",
    scope: "libs",
    module: "core",
    name: "Core Imports No Domain Proto (transcript, ocr, article, artifact, cards, ...)",
    run: verifyCoreImportsNoDomainProto,
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
  {
    id: "protocol-holds-no-plugin-contract",
    scope: "libs",
    module: "protocol",
    name: "numen.v1 Holds No Plugin Contract (transcript, article, book, document services, book and recording kinds, card fields)",
    run: verifyProtocolHoldsNoPluginContract,
  },
  {
    id: "contracts-name-no-plugin",
    scope: "libs",
    module: "protocol",
    name: "Shared Contracts Name No Plugin (schema, SDKs, plugin host, notes: comments and documents included)",
    run: verifyContractsNameNoPlugin,
  },
  {
    id: "no-file-kind-outside-plugins",
    scope: "libs",
    module: "kinds",
    name: "Core, Notes, SDKs, UI and Window Name No File Format or Source Kind (.pdf, .mp3, book, recording, url, document)",
    run: verifyNoFileKindOutsidePlugins,
  },
];
