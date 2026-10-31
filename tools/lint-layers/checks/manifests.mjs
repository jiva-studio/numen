import { spawnSync } from 'node:child_process'
import fs from 'node:fs'
import path from 'node:path'
import {
  REPO_ROOT,
  failOnFindings,
  listShippedFiles,
  readSource,
  requireMatches,
} from './helpers.mjs'

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

/** A Go call that decodes a value named for the manifest, which is the manifest text without its file name. */
const GO_MANIFEST_VALUE_DECODE = /\bUnmarshal\s*\(\s*(?:\w+\.)*[Mm]anifest\w*\s*[,)]/

/** A call that decodes JSON text in a script. */
const SCRIPT_DECODE = /\bJSON\.parse\s*\(|\.json\s*\(\s*\)/

/** A call that decodes JSON text in Go. */
const GO_DECODE = /\b(?:json\.(?:Unmarshal|NewDecoder)|protojson\.\w+|(?!yaml\.|toml\.)\w+\.Unmarshal)\s*\(/

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
      MANIFEST_TEMPLATE.test(text) ||
      (file.endsWith('.go') && GO_MANIFEST_VALUE_DECODE.test(source.code))
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

/** Generated code and fixture data, which no author writes. */
const NOT_AUTHORED = /(\.pb\.go|\.connect\.go|_pb\.ts|\.d\.ts)$|(^|\/)(gen|testdata|fixtures|node_modules)\//

/** The tracked files under `pathspecs` that an author wrote, tests included. */
function listAuthoredFiles(pathspecs) {
  const listed = spawnSync('git', ['ls-files', '--', ...pathspecs], { cwd: REPO_ROOT, encoding: 'utf8' })
  return (listed.stdout || '')
    .split('\n')
    .filter((file) => file && !NOT_AUTHORED.test(file) && fs.existsSync(path.join(REPO_ROOT, file)))
}

/**
 * Reads the Go files and scripts of the libraries, applications and tools, tests included, and the
 * shipped ones of the plugins. A plugin test reads the manifest its own module embeds.
 */
function readHostSources() {
  const files = [...listAuthoredFiles(['libs', 'apps', 'tools']), ...listShippedFiles(['plugins'])].filter(
    (file) => file.endsWith('.go') || SCRIPT_FILE.test(file),
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

export const MANIFEST_VERIFIERS = [
  {
    id: "manifest-parsed-in-one-place",
    scope: 'libs',
    module: 'quality',
    name: 'Manifest Parsed In One Place (no file that names manifest.json decodes JSON itself; readers decode through the generated schema)',
    run: verifyManifestParsedOnce,
  },
]
