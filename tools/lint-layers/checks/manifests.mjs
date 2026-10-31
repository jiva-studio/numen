import fs from 'node:fs'
import path from 'node:path'
import { listManifestSections } from '../../shared/manifests.mjs'
import { loadManifestWords, splitSegments } from './dictionary.mjs'
import {
  REPO_ROOT,
  failOnFindings,
  listShippedFiles,
  readSource,
  requireMatches,
} from './helpers.mjs'

/** The schema of a manifest. */
const MANIFEST_PROTO = 'libs/protocol/proto/numen/v1/manifest.proto'

/**
 * The manifest of a plugin is read against the schema in one place per language: Go reads it in the
 * package that decodes `numen.v1.Manifest` from JSON, and the scripts read it in a module that decodes
 * it through the generated `ManifestSchema`. A file that names manifest.json decodes no JSON of its own.
 */

/** A string that names the file. */
const MANIFEST_LITERAL = /manifest\.json/

/** A template literal that names the file after a substitution, which the literal reader does not return. */
const MANIFEST_TEMPLATE = /`[^`\n]*[/}]manifest\.json[^`\n]*`/

/** A name that stands for the file in code. */
const MANIFEST_NAME = /\b(?:MANIFEST_FILE|manifest\.FileName)\b/

/** A call that decodes JSON text in a script. */
const SCRIPT_DECODE = /\bJSON\.parse\s*\(|\.json\s*\(\s*\)/

/** A call that decodes JSON text in Go. */
const GO_DECODE = /\b(?:json\.(?:Unmarshal|NewDecoder)|protojson\.\w+|\w+\.Unmarshal)\s*\(/

/** A call that decodes a manifest through the generated TypeScript schema. */
const SCRIPT_SCHEMA_DECODE = /\bfromJson\s*\(\s*ManifestSchema\b/

/** Whether Go code decodes the generated message with protojson. */
const isGoSchemaReader = (code) => /\bprotojson\./.test(code) && /\bv1\.Manifest\b/.test(code)

const SCRIPT_FILE = /\.(ts|mts|js|mjs|cjs|vue)$/

const isSchemaReader = (file, code) =>
  file.endsWith('.go') ? isGoSchemaReader(code) : SCRIPT_SCHEMA_DECODE.test(code)

/**
 * Lists the files that name manifest.json and decode JSON themselves, given as `{ file, source, text }`,
 * where source is what `readSource` returns and text is the file. A file that decodes through the
 * generated schema is the reader of its language and is not listed.
 */
export function findManifestParsers(sources) {
  const found = []
  for (const { file, source, text = '' } of sources) {
    const namesFile =
      source.literals.some(({ value }) => MANIFEST_LITERAL.test(value)) ||
      MANIFEST_NAME.test(source.code) ||
      MANIFEST_TEMPLATE.test(text)
    if (!namesFile || isSchemaReader(file, source.code)) continue
    const decode = file.endsWith('.go') ? GO_DECODE : SCRIPT_DECODE
    source.code.split('\n').forEach((line, index) => {
      const match = decode.exec(line)
      if (match) {
        found.push({
          file,
          line: index + 1,
          symbol: `${match[0].trim()} in a file that names manifest.json`,
        })
      }
    })
  }
  return found
}

/** Reads the shipped Go files and scripts of the libraries, applications and tools. */
function readHostSources() {
  const files = listShippedFiles(['libs', 'apps', 'tools']).filter(
    (file) => (file.endsWith('.go') || SCRIPT_FILE.test(file)) && !file.endsWith('.d.ts'),
  )
  return files
    .map((file) => ({
      file,
      source: readSource(file),
      text: fs.readFileSync(path.join(REPO_ROOT, file), 'utf8'),
    }))
    .filter(({ source }) => source !== null)
}

export function verifyManifestParsedOnce() {
  const sources = readHostSources()
  requireMatches('host and tool sources read for manifest parsing', sources.length, 500)
  const readers = sources.filter(({ file, source }) => isSchemaReader(file, source.code))
  requireMatches('schema readers of manifest.json', readers.length, 2)
  failOnFindings(
    'A file decodes manifest.json without the generated schema',
    findManifestParsers(sources),
  )
}

/**
 * The top-level sections of the manifest schema that are named after a function a plugin owns. A
 * section is named so when its name, or one of the words of its name, is a word that the manifest of
 * exactly one plugin states: an id segment, a tab kind, a note type, a command or a kind. The words
 * come from the manifests, and no list of names is kept here.
 *
 * @param {{ name: string, jsonName: string }[]} sections - the fields of the top-level message
 * @param {Map<string, Set<string>>} words - the words of the manifests, from `loadManifestWords`
 * @param {string} proto - the text of the schema, to locate each field
 */
export function findPluginNamedSections(sections, words, proto = '') {
  const lines = proto.split('\n')
  const found = []
  for (const { name, jsonName } of sections) {
    const candidates = new Set([jsonName.toLowerCase(), ...splitSegments(jsonName)])
    const line = lines.findIndex((one) => new RegExp(`\\b${name}\\s*=\\s*\\d+`).test(one)) + 1
    for (const word of candidates) {
      const owners = words.get(word)
      if (owners?.size !== 1) continue
      found.push({
        file: MANIFEST_PROTO,
        line,
        symbol: `top-level field "${jsonName}" is named after "${word}", which plugin ${[...owners][0]} owns`,
      })
    }
  }
  return found
}

export function verifySchemaNamesNoPluginFunction() {
  const sections = listManifestSections()
  requireMatches('top-level fields of the manifest schema', sections.length, 15)
  const words = loadManifestWords()
  requireMatches('words of the plugin manifests', words.size, 100)
  const proto = fs.readFileSync(path.join(REPO_ROOT, MANIFEST_PROTO), 'utf8')
  failOnFindings(
    'A top-level section of the manifest schema is named after a function a plugin owns',
    findPluginNamedSections(sections, words, proto),
  )
}

export const MANIFEST_VERIFIERS = [
  {
    id: "manifest-parsed-in-one-place",
    scope: 'libs',
    module: 'quality',
    name: 'Manifest Parsed In One Place (no file that names manifest.json decodes JSON itself; readers decode through the generated schema)',
    run: verifyManifestParsedOnce,
  },
  {
    id: "manifest-schema-names-no-plugin-function",
    scope: 'libs',
    module: 'quality',
    name: 'Manifest Schema Names No Plugin Function (no top-level section of the manifest is named after a word that the manifest of exactly one plugin owns)',
    run: verifySchemaNamesNoPluginFunction,
  },
]
