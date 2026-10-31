import fs from "node:fs";
import path from "node:path";
import {
  REPO_ROOT,
  failOnFindings,
  getImportSpecifiers,
  getVueScript,
  parsePackageJson,
  requireMatches,
} from "./helpers.mjs";
import { spawnSync } from "node:child_process";

/** The package that holds the generated schema. */
const PROTOCOL = "@numen/protocol";

/** The first-party client of the core: the only package besides the plugin SDK that names the schema. */
const CORE_CLIENT = "@numen/core-client";

/** The packages that may import or declare the schema. */
export const PROTOCOL_USERS = ["@numen/plugin-sdk", CORE_CLIENT];

const DEPENDENCY_FIELDS = ["dependencies", "devDependencies", "peerDependencies", "optionalDependencies"];

const SOURCE_FILE = /\.(ts|tsx|mts|cts|js|mjs|cjs|vue)$/;

/** Generated code, installed packages and build output, which are not source of a package. */
const NOT_SOURCE = /(^|\/)(node_modules|dist|gen)\//;

/** True for a module specifier that names the schema package or a path inside it. */
function isProtocolSpecifier(specifier) {
  return specifier === PROTOCOL || specifier.startsWith(PROTOCOL + "/");
}

/**
 * Lists the findings for `references` given as `{ file, line, owner, symbol }`: every import or
 * declaration of the schema package by a package that is not in `allowed`. `owner` is the name of
 * the package the file belongs to.
 */
export function findProtocolViolations(references, allowed = PROTOCOL_USERS) {
  return references
    .filter((reference) => !allowed.includes(reference.owner))
    .map(({ file, line, owner, symbol }) => ({
      file,
      line,
      symbol: `${owner} names ${PROTOCOL} (${symbol}); only ${allowed.join(" and ")} may`,
    }));
}

/** The package.json files below `dirs`, each with the parsed manifest and its directory. */
function readManifests(dirs) {
  const listed = spawnSync("git", ["ls-files", "--", ...dirs], { cwd: REPO_ROOT, encoding: "utf8" });
  return (listed.stdout || "")
    .split("\n")
    .filter((file) => file && path.basename(file) === "package.json" && !NOT_SOURCE.test(file) && fs.existsSync(path.join(REPO_ROOT, file)))
    .map((file) => ({ file, dir: path.dirname(file), manifest: parsePackageJson(fs.readFileSync(path.join(REPO_ROOT, file), "utf8")) }));
}

/** The tracked source files below `dirs`, tests included. */
function listSourceFiles(dirs) {
  const listed = spawnSync("git", ["ls-files", "--", ...dirs], { cwd: REPO_ROOT, encoding: "utf8" });
  return (listed.stdout || "")
    .split("\n")
    .filter((file) => file && SOURCE_FILE.test(file) && !NOT_SOURCE.test(file) && fs.existsSync(path.join(REPO_ROOT, file)));
}

/** Reads every declaration and every import of the schema package outside `libs/protocol`. */
export function readProtocolReferences() {
  const dirs = ["apps", "libs", "plugins", "tools"];
  const manifests = readManifests(dirs).filter((one) => one.manifest.name !== PROTOCOL);
  const ownerOf = (file) =>
    manifests.filter((one) => file.startsWith(one.dir + "/")).sort((a, b) => b.dir.length - a.dir.length)[0];
  const references = [];
  for (const one of manifests) {
    for (const field of DEPENDENCY_FIELDS) {
      if (PROTOCOL in (one.manifest[field] ?? {})) {
        references.push({ file: one.file, line: 1, owner: one.manifest.name, symbol: `${field}: ${PROTOCOL}` });
      }
    }
  }
  let examined = 0;
  for (const file of listSourceFiles(dirs)) {
    const owner = ownerOf(file);
    if (!owner) continue;
    examined += 1;
    const text = fs.readFileSync(path.join(REPO_ROOT, file), "utf8");
    const script = file.endsWith(".vue") ? getVueScript(text) : text;
    for (const specifier of getImportSpecifiers(script).filter(isProtocolSpecifier)) {
      const line = script.slice(0, Math.max(0, script.indexOf(specifier))).split("\n").length;
      references.push({ file, line, owner: owner.manifest.name, symbol: `import ${JSON.stringify(specifier)}` });
    }
  }
  return { references, examined, packages: manifests.length };
}

/** Only the plugin SDK and the core client import or declare the schema; every other package reaches the core through one of them. */
export function verifyProtocolImportedOnlyByClients() {
  const { references, examined, packages } = readProtocolReferences();
  requireMatches("packages read for the schema boundary", packages, 15);
  requireMatches("source files read for the schema boundary", examined, 500);
  requireMatches(
    "imports of the schema by the SDK and the core client",
    references.filter((reference) => PROTOCOL_USERS.includes(reference.owner)).length,
    10,
  );
  failOnFindings(
    "A package other than the plugin SDK and the core client names @numen/protocol",
    findProtocolViolations(references),
  );
}

/** The directory of the core client package. */
const CORE_CLIENT_DIR = "libs/core-client";

/**
 * Lists the findings for `manifests` given as `{ file, manifest }`: the core client, the manifest in
 * `libs/core-client`, must be private, and no package that is not private may declare it as a dependency.
 */
export function findCoreClientPublishViolations(manifests) {
  const client = manifests.find((one) => one.file === `${CORE_CLIENT_DIR}/package.json`);
  if (!client) return [{ file: `${CORE_CLIENT_DIR}/package.json`, line: 1, symbol: "the core client has no package.json" }];
  const found = [];
  if (client.manifest.private !== true) {
    found.push({ file: client.file, line: 1, symbol: `${client.manifest.name} is not "private": true` });
  }
  for (const { file, manifest } of manifests) {
    if (manifest.private === true || file === client.file) continue;
    for (const field of DEPENDENCY_FIELDS) {
      if (client.manifest.name in (manifest[field] ?? {})) {
        found.push({ file, line: 1, symbol: `${manifest.name} is published and declares ${client.manifest.name} in ${field}` });
      }
    }
  }
  return found;
}

/** The core client is never published: it is private and no published package depends on it. */
export function verifyCoreClientIsPrivate() {
  const manifests = readManifests(["apps", "libs", "plugins", "tools", "tests"]);
  requireMatches("packages read for the publish boundary", manifests.length, 15);
  failOnFindings("The core client is public or a published package depends on it", findCoreClientPublishViolations(manifests));
}

export const CORE_CLIENT_VERIFIERS = [
  {
    id: "protocol-imported-only-by-clients",
    scope: "libs",
    module: "boundary",
    name: "Applications Reach the Core Only Through the Core Client (only @numen/plugin-sdk and @numen/core-client import or declare @numen/protocol)",
    run: verifyProtocolImportedOnlyByClients,
  },
  {
    id: "core-client-private",
    scope: "libs",
    module: "boundary",
    name: "Core Client Is Private (libs/core-client is \"private\": true and no package that is not private depends on it)",
    run: verifyCoreClientIsPrivate,
  },
];
