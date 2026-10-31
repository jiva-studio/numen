import fs from "node:fs";
import path from "node:path";
import { REPO_ROOT, failOnFindings, listShippedFiles, readSource, requireMatches } from "./helpers.mjs";

/** The libraries and applications that serve, call or wrap the services of `numen.v1`. */
const HOST_ROOTS = [
  "libs/core",
  "libs/core-client",
  "libs/plugin-sdk-go",
  "libs/plugin-sdk-ts",
  "libs/plugin-host-ts",
  "libs/plugin-manager",
  "libs/plugin-catalog",
  "libs/ui",
  "apps",
];

/**
 * The top-level services, messages and enums of proto sources. Each source is `{ file, code }`, the
 * code having its comments blanked. A definition is `{ kind, name, file, line, body }`.
 */
export function listDefinitions(sources) {
  const definitions = new Map();
  for (const { file, code } of sources) {
    let current = null;
    let depth = 0;
    code.split("\n").forEach((text, index) => {
      const opening = depth === 0 ? /^\s*(message|enum|service)\s+(\w+)\s*\{/.exec(text) : null;
      if (opening) current = { kind: opening[1], name: opening[2], file, line: index + 1, body: "" };
      if (current) current.body += text + "\n";
      for (const char of text) {
        if (char === "{") depth++;
        if (char === "}") depth--;
      }
      if (current && depth === 0) {
        definitions.set(current.name, current);
        current = null;
      }
    });
  }
  return definitions;
}

const escapePattern = (text) => text.replace(/[.*+?^${}()|[\]\\]/g, "\\$&");

/** Whether `text` states the name of a definition: as an identifier, a generated type, a client or a handler constructor. */
export function isNamedIn(name, text) {
  return new RegExp(`(?<![A-Za-z0-9])(?:New)?${escapePattern(name)}(?![a-z0-9])`).test(text);
}

/**
 * The definitions that neither a host source nor another reachable definition refers to. A definition is
 * a root when the host corpus names it; every definition that a reachable one refers to is reachable.
 */
export function findUnreachableDefinitions(definitions, corpus) {
  const reachable = new Set([...definitions.keys()].filter((name) => isNamedIn(name, corpus)));
  const queue = [...reachable];
  while (queue.length > 0) {
    const { body } = definitions.get(queue.pop());
    for (const other of definitions.keys()) {
      if (reachable.has(other)) continue;
      if (new RegExp(`(?<![A-Za-z0-9.])${escapePattern(other)}(?![A-Za-z0-9])`).test(body)) {
        reachable.add(other);
        queue.push(other);
      }
    }
  }
  return [...definitions.values()]
    .filter(({ name }) => !reachable.has(name))
    .map(({ file, line, kind, name }) => ({ file, line, symbol: `${kind} ${name} is used by no host, SDK, client or application source` }));
}

/**
 * `numen.v1` holds the contracts of the core and its extension points. A service, message or enum that
 * no source of the core, the clients, the SDKs, the plugin host, the UI libraries or the applications
 * uses, directly or through a definition they use, is a contract between plugins.
 */
export function verifyProtocolHoldsOnlyHostContracts() {
  const sources = listShippedFiles(["libs/protocol/proto/numen/v1/*.proto"]).map((file) => ({ file, code: readSource(file).code }));
  const definitions = listDefinitions(sources);
  requireMatches("definitions of numen.v1", definitions.size, 200);
  const files = listShippedFiles(HOST_ROOTS).filter((file) => /\.(go|ts|vue)$/.test(file));
  requireMatches("host sources read for protocol use", files.length, 500);
  const corpus = files.map((file) => fs.readFileSync(path.join(REPO_ROOT, file), "utf8")).join("\n");
  failOnFindings("numen.v1 holds a definition that only plugins use", findUnreachableDefinitions(definitions, corpus));
}

export const PROTOCOL_VERIFIERS = [
  {
    id: "protocol-holds-only-host-contracts",
    scope: "libs",
    module: "protocol",
    name: "numen.v1 Holds Only Host Contracts (every service, message and enum is used by the core, a client, an SDK, the plugin host, the UI or an application)",
    run: verifyProtocolHoldsOnlyHostContracts,
  },
];
