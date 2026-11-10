import fs from "node:fs";
import path from "node:path";

/**
 * Reads the structs of a Go package: struct name to its JSON fields, and the string constants of
 * each named string type.
 *
 * @param {string} source
 * @returns {{ structs: Map<string, Map<string, boolean>>, enums: Map<string, Set<string>> }}
 */
export function parseGoApi(source) {
  const structs = new Map();
  for (const block of source.matchAll(/^type (\w+) struct \{\n([\s\S]*?)^\}/gm)) {
    const fields = new Map();
    for (const line of block[2].split("\n")) {
      const field = line.match(/^\s+\w+\s+(\S+)\s+`json:"([^",]+)((?:,[^"]*)?)"`/);
      if (!field) continue;
      const isOptional = field[1].startsWith("*") || field[3].includes("omitempty");
      fields.set(field[2], isOptional);
    }
    if (fields.size > 0) structs.set(block[1], fields);
  }
  const stringTypes = new Set([...source.matchAll(/^type (\w+) string$/gm)].map((one) => one[1]));
  const enums = new Map([...stringTypes].map((name) => [name, new Set()]));
  for (const constant of source.matchAll(/^\s+\w+\s+(\w+)\s+=\s+"([^"]*)"$/gm)) {
    if (constant[2] !== "" && enums.has(constant[1])) enums.get(constant[1]).add(constant[2]);
  }
  return { structs, enums };
}

/**
 * Reads the TypeScript declarations of the API types of a plugin: interface name to its properties
 * and whether each is optional, and the values of each `const X = {...} as const` enum.
 *
 * @param {string} source
 * @returns {{ interfaces: Map<string, Map<string, boolean>>, enums: Map<string, Set<string>> }}
 */
export function parseTsApi(source) {
  const interfaces = new Map();
  for (const block of source.matchAll(/^export interface (\w+) \{\n([\s\S]*?)^\}/gm)) {
    const props = new Map();
    for (const line of block[2].split("\n")) {
      const prop = line.match(/^ {2}(?:readonly )?(\w+)(\?)?:/);
      if (prop) props.set(prop[1], prop[2] === "?");
    }
    interfaces.set(block[1], props);
  }
  const enums = new Map();
  for (const block of source.matchAll(/^export const (\w+) = \{\n([\s\S]*?)^\} as const/gm)) {
    const values = new Set([...block[2].matchAll(/:\s*'([^']*)'/g)].map((one) => one[1]));
    values.delete("");
    enums.set(block[1], values);
  }
  return { interfaces, enums };
}

const differences = (label, expected, actual, describe) => {
  const found = [];
  for (const name of expected) if (!actual.has(name)) found.push(`${label}: ${describe} ${name} is missing in TypeScript`);
  for (const name of actual) if (!expected.has(name)) found.push(`${label}: ${describe} ${name} is not in Go`);
  return found;
};

/**
 * Compares what a plugin's Go API declares with the TypeScript types of its frontend.
 *
 * @param {{ go: string, ts: string, label: string }} sources
 * @returns {string[]} one line for each difference
 */
export function compareApi({ go, ts, label }) {
  const goApi = parseGoApi(go);
  const tsApi = parseTsApi(ts);
  const found = [
    ...differences(label, new Set(goApi.structs.keys()), new Set(tsApi.interfaces.keys()), "type"),
    ...differences(label, new Set(goApi.enums.keys()), new Set(tsApi.enums.keys()), "enum"),
  ];
  for (const [name, fields] of goApi.structs) {
    const props = tsApi.interfaces.get(name);
    if (!props) continue;
    found.push(...differences(`${label} ${name}`, new Set(fields.keys()), new Set(props.keys()), "field"));
    for (const [field, isOptional] of fields) {
      // A request field left out is the zero value, so TypeScript makes every one optional.
      if (!name.endsWith("Request") && props.has(field) && props.get(field) !== isOptional) {
        found.push(`${label} ${name}.${field}: Go ${isOptional ? "omits it when empty" : "always sends it"}, TypeScript says the opposite`);
      }
    }
  }
  for (const [name, values] of goApi.enums) {
    const declared = tsApi.enums.get(name);
    if (declared) found.push(...differences(`${label} ${name}`, values, declared, "value"));
  }
  return found;
}

/** Where the TypeScript types of the Go API of a plugin are declared, relative to the plugin. */
const TYPES_FILES = [
  ["frontend", "api", "types.ts"],
  ["frontend", "lib", "wire", "types.ts"],
];

/** The source of the types file and of the files of the plugin it imports the enumerations from. */
function readTypesWithImports(tsFile) {
  const source = fs.readFileSync(tsFile, "utf8");
  const imported = [...source.matchAll(/from '(\.\.?\/[^']+)'/g)]
    .map((one) => path.resolve(path.dirname(tsFile), one[1] + ".ts"))
    .filter((file) => fs.existsSync(file));
  return [source, ...imported.map((file) => fs.readFileSync(file, "utf8"))].join("\n");
}

/**
* The plugins whose frontend mirrors the Go structs of `backend/api`: the TypeScript in
 * `frontend/api/types.ts` or `frontend/lib/wire/types.ts` must declare the same types, fields, optionality and enum values.
 *
 * @param {{ repoRoot: string, warnOnly?: boolean }} options
 * @returns {Promise<{ id: string, name: string, pass: boolean, summary: string, details?: string, durationMs: number }>}
 */
export async function runApiTypesCheck({ repoRoot, warnOnly = false }) {
  const start = Date.now();
  const pluginsDir = path.join(repoRoot, "plugins");
  const found = [];
  let compared = 0;
  for (const plugin of fs.readdirSync(pluginsDir)) {
    const apiDir = path.join(pluginsDir, plugin, "backend", "api");
    const tsFile = TYPES_FILES.map((one) => path.join(pluginsDir, plugin, ...one)).find((one) => fs.existsSync(one));
    if (!fs.existsSync(apiDir) || !tsFile) continue;
    const go = fs
      .readdirSync(apiDir)
      .filter((name) => name.endsWith(".go") && !name.endsWith("_test.go"))
      .map((name) => fs.readFileSync(path.join(apiDir, name), "utf8"))
      .join("\n");
    found.push(...compareApi({ go, ts: readTypesWithImports(tsFile), label: plugin }));
    compared++;
  }
  return {
    id: "api-types",
    name: "Plugin API types match their Go structs",
    pass: found.length === 0 || warnOnly,
    summary:
      found.length === 0
        ? `${compared} plugin(s): TypeScript API types match the Go structs`
        : `${found.length} difference(s) between TypeScript API types and Go structs`,
    ...(found.length > 0 ? { details: found.map((one) => `✖ ${one}`).join("\n") } : {}),
    durationMs: Date.now() - start,
  };
}
