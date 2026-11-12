import { runTool } from "../../../../shared/tools.mjs";
import fs from "node:fs";
import path from "node:path";
import { REPO_ROOT, failOnFindings, parsePackageJson, requireMatches } from "../../../lib/graph.mjs";

/** The package that holds the generated schema. */
const PROTOCOL = "@numen/protocol";

/** The first-party client of the core: the only package besides the plugin SDK that names the schema. */
const CORE_CLIENT = "@numen/core-client";

/** The packages that may declare the schema. */
export const PROTOCOL_USERS = ["@numen/plugin-sdk", CORE_CLIENT];

const DEPENDENCY_FIELDS = ["dependencies", "devDependencies", "peerDependencies", "optionalDependencies"];

/** The fields of the dependencies a package brings to whoever installs it. */
const INSTALLED_FIELDS = ["dependencies", "peerDependencies", "optionalDependencies"];

/** Installed packages, build output, generated code and fixtures, which are not packages of the repository. */
const NOT_SOURCE = /(^|\/)(node_modules|dist|gen|testdata)\//;

/** The package.json files below `dirs`, each with the parsed manifest and its directory. */
function readManifests(dirs) {
  const listed = runTool("git", ["ls-files", "--", ...dirs], { cwd: REPO_ROOT, encoding: "utf8" });
  return (listed.stdout || "")
    .split("\n")
    .filter((file) => file && path.basename(file) === "package.json" && !NOT_SOURCE.test(file) && fs.existsSync(path.join(REPO_ROOT, file)))
    .map((file) => ({ file, dir: path.dirname(file), manifest: parsePackageJson(fs.readFileSync(path.join(REPO_ROOT, file), "utf8")) }));
}

/**
 * Lists the findings for `manifests` given as `{ file, manifest }`: every declaration of the schema
 * package by a package that is not in `allowed`. The imports are checked by `dependency-graph`.
 */
export function findProtocolDeclarations(manifests, allowed = PROTOCOL_USERS) {
  const found = [];
  for (const { file, manifest } of manifests) {
    if (manifest.name === PROTOCOL || allowed.includes(manifest.name)) continue;
    for (const field of DEPENDENCY_FIELDS) {
      if (PROTOCOL in (manifest[field] ?? {})) {
        found.push({ file, line: 1, symbol: `${manifest.name} declares ${PROTOCOL} in ${field}; only ${allowed.join(" and ")} may` });
      }
    }
  }
  return found;
}

/** Only the plugin SDK and the core client declare the schema; every other package reaches the core through one of them. */
export function verifyProtocolDeclaredOnlyByClients() {
  const manifests = readManifests(["apps", "libs", "plugins", "tools"]);
  requireMatches("packages read for the schema boundary", manifests.length);
  requireMatches(
    "declarations of the schema by the SDK and the core client",
    manifests.filter(({ manifest }) => PROTOCOL_USERS.includes(manifest.name) && DEPENDENCY_FIELDS.some((field) => PROTOCOL in (manifest[field] ?? {}))).length,
    PROTOCOL_USERS.length,
  );
  failOnFindings("A package other than the plugin SDK and the core client declares @numen/protocol", findProtocolDeclarations(manifests));
}

/**
 * The first-party packages plugins rely on: those a plugin's `package.json` declares, and those they bring in
 * turn through their installed dependencies. `manifests` are `{ file, manifest }`; a plugin is a manifest
 * below `plugins/`.
 */
export function findPluginReliedPackages(manifests) {
  const byName = new Map(manifests.map((one) => [one.manifest.name, one]));
  const relied = new Set();
  const queue = [];
  const take = (manifest, fields) => {
    for (const field of fields) {
      for (const name of Object.keys(manifest[field] ?? {})) {
        if (byName.has(name) && !relied.has(name)) {
          relied.add(name);
          queue.push(name);
        }
      }
    }
  };
  for (const { file, manifest } of manifests) if (file.startsWith("plugins/")) take(manifest, DEPENDENCY_FIELDS);
  while (queue.length > 0) take(byName.get(queue.pop()).manifest, INSTALLED_FIELDS);
  return relied;
}

/**
 * Lists the findings for `manifests`: the core client must be private, and no package plugins rely on may be
 * the core client. Fails when plugins rely on no package, since the rule would then hold over nothing.
 */
export function findCoreClientPublishViolations(manifests) {
  const client = manifests.find((one) => one.manifest.name === CORE_CLIENT);
  if (!client) return [{ file: "package.json", line: 1, symbol: `no package is named ${CORE_CLIENT}` }];
  const relied = findPluginReliedPackages(manifests);
  if (relied.size === 0) return [{ file: "plugins", line: 1, symbol: "plugins rely on no first-party package; the published set is empty" }];
  const found = [];
  if (client.manifest.private !== true) found.push({ file: client.file, line: 1, symbol: `${CORE_CLIENT} is not "private": true` });
  if (relied.has(CORE_CLIENT)) found.push({ file: client.file, line: 1, symbol: `plugins rely on ${CORE_CLIENT}` });
  return found;
}

/** The core client is never published: it is private, and nothing plugins rely on is or brings it. */
export function verifyCoreClientIsPrivate() {
  const manifests = readManifests(["apps", "libs", "plugins", "tools", "tests"]);
  requireMatches("packages read for the publish boundary", manifests.length);
  failOnFindings("The core client is public or plugins rely on it", findCoreClientPublishViolations(manifests));
}

export const CORE_CLIENT_VERIFIERS = [
  {
    id: "protocol-declared-only-by-clients",
    scope: "libs",
    module: "boundary",
    name: "Only @numen/plugin-sdk and @numen/core-client Declare @numen/protocol (package.json; the imports are dependency-graph's)",
    run: verifyProtocolDeclaredOnlyByClients,
  },
  {
    id: "core-client-private",
    scope: "libs",
    module: "boundary",
    name: "Core Client Is Private (libs/core/client is \"private\": true and is not among the packages plugins rely on)",
    run: verifyCoreClientIsPrivate,
  },
];
