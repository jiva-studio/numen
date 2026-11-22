import fs from 'node:fs'
import path from 'node:path'
import { isNotShipped } from '../frontend-graph/check.mjs'
import { REPO_ROOT, failOnFindings, listShippedFiles, requireMatches } from '../../../lib/graph.mjs'

const EDITOR_SRC = 'apps/desktop/editor/src'
const MOBILE = 'apps/mobile'
const SOURCE = /\.(ts|vue)$/

/** The line of a character offset in a text, counted from 1. */
const lineAt = (text, offset) => text.slice(0, offset).split('\n').length

/** The matches of a pattern in the files, one finding each, with the line of the match. */
function findPattern(files, pattern, describe) {
  const findings = []
  for (const { file, text } of files) {
    for (const match of text.matchAll(new RegExp(pattern.source, 'g'))) {
      findings.push({ file, line: lineAt(text, match.index), symbol: describe(match[0]) })
    }
  }
  return findings
}

const isShippedSource = (file, root) =>
  file.startsWith(`${root}/`) && SOURCE.test(file) && !isNotShipped(file)

/** The files of the editor that ship, or of any other root, with their text. */
function readSources(root) {
  return listShippedFiles([root])
    .filter((file) => isShippedSource(file, root))
    .map((file) => ({ file, text: fs.readFileSync(path.join(REPO_ROOT, file), 'utf8') }))
}

const PLUGIN_ID = /md\.numen\.[a-zA-Z0-9_-]+/
const READER_KIND = /['"]reader:[a-zA-Z0-9_-]+['"]/

/** A plugin id or a reader tab kind spelled out in a source of the host. The host learns both from the plugins. */
export function findPluginNames(files) {
  const found = [PLUGIN_ID, READER_KIND].flatMap((pattern) =>
    findPattern(files, pattern, (hit) => `spells out ${hit}`),
  )
  return { findings: found, read: files.length }
}

/** The directory whose files alone may read the options of a setting: the core's answer is applied there once. */
const OPTIONS_READER_DIR = 'apps/desktop/editor/src/entities/settings/api/'

/** A read of `.optionsBy` outside the settings api: the rules of options and conditions are applied once, by the core. */
export function findOptionsReaders(files) {
  const readers = files.filter((one) => !one.file.startsWith(OPTIONS_READER_DIR))
  return {
    findings: findPattern(
      readers,
      /\.optionsBy\b/,
      () => 'reads optionsBy outside entities/settings/api',
    ),
    read: files.length,
  }
}

/** The names of the old paths of a chosen palette result: each one was a second way to open it. */
const OLD_PALETTE_NAMES =
  /\b(openDestination|resolveDestination|SearchDestination|createNoteInvocation|CREATE_NOTE_ID|MENU_ACTION_PREFIX|resolveMenuChoice)\b/
const PALETTE = 'apps/desktop/editor/src/features/command-palette/'
const PALETTE_VIEW = `${PALETTE}ui/CommandPalette.vue`
const PALETTE_FIELD = `${PALETTE}model/usePaletteInput.ts`

/**
 * Where a chosen palette result is turned into a command: no second destination, no create-note shortcut, no
 * prefix that makes an action id differ from its command id, no document id among the actions, and one call of
 * `runInvocation`, in the field and not in the view.
 */
export function findPaletteBypassingCommands(files) {
  const findings = findPattern(
    files,
    OLD_PALETTE_NAMES,
    (hit) => `uses ${hit}, a path of a result that is no command`,
  )
  const inPalette = files.filter((one) => one.file.startsWith(PALETTE))
  findings.push(
    ...findPattern(
      inPalette,
      /\bNOTE = 'note'/,
      () => 'gives a document an action id that is no command',
    ),
  )
  const view = files.find((one) => one.file === PALETTE_VIEW)
  const field = files.find((one) => one.file === PALETTE_FIELD)
  for (const one of [view, field].filter(Boolean)) {
    findings.push(
      ...findPattern(
        [one],
        /CREATE_NOTE|item\s*===/,
        (hit) => `decides on a result itself (${hit})`,
      ),
    )
  }
  if (view)
    findings.push(
      ...findPattern(
        [view],
        /runInvocation\(/,
        () => 'runs an invocation in the view; only the field does',
      ),
    )
  if (field) {
    const calls = field.text.match(/runInvocation\(/g) ?? []
    if (calls.length !== 1)
      findings.push({
        file: field.file,
        line: 1,
        symbol: `calls runInvocation ${calls.length} times, once is the choice it is handed`,
      })
  }
  return { findings, read: files.length }
}

/** A settings page section of its own for the shell: one renderer draws the settings of every owner. */
export function findShellSettingsSections(files) {
  const own = files.filter(
    (file) => file.endsWith('.vue') && /WindowSettingsSection|NamingSettings/.test(file),
  )
  return {
    findings: own.map((file) => ({ file, line: 1, symbol: 'is a settings section of the shell' })),
    read: files.length,
  }
}

/** The policy of the phone's page, exactly: what the WebView may load. */
export const MOBILE_POLICY = {
  'default-src': ["'self'"],
  'img-src': ["'self'", 'data:'],
  'style-src': ["'self'", "'unsafe-inline'"],
  'font-src': ["'self'"],
  'connect-src': ["'self'", 'http://127.0.0.1:*'],
  'object-src': ["'none'"],
  'base-uri': ["'none'"],
  'form-action': ["'none'"],
}

/** The directives of the policy a page carries, by name. */
function readPolicy(html) {
  const content = /http-equiv="Content-Security-Policy"\s+content="([^"]+)"/.exec(html)?.[1] ?? ''
  return Object.fromEntries(
    content
      .split(';')
      .map((one) => one.trim().split(/\s+/))
      .filter(([named]) => named)
      .map(([named, ...sources]) => [named, sources]),
  )
}

/**
 * The policy of the phone's page is the whole of what it may load, and its sources name no host but the core.
 * What a WebView makes of the policy is answered by a device.
 */
export function findMobilePageViolations(html, sources) {
  const findings = []
  const policy = readPolicy(html)
  for (const [named, allowed] of Object.entries(MOBILE_POLICY)) {
    if (JSON.stringify(policy[named]) !== JSON.stringify(allowed))
      findings.push({
        file: `${MOBILE}/index.html`,
        line: 1,
        symbol: `${named} is not ${allowed.join(' ')}`,
      })
  }
  for (const named of Object.keys(policy)) {
    if (!(named in MOBILE_POLICY))
      findings.push({
        file: `${MOBILE}/index.html`,
        line: 1,
        symbol: `writes ${named}, a directive the policy does not hold`,
      })
  }
  for (const { file, text } of sources) {
    for (const match of text.matchAll(/https?:\/\/([^:/'"`\s$]+)/g)) {
      if (match[1] !== '127.0.0.1')
        findings.push({
          file,
          line: lineAt(text, match.index),
          symbol: `names the host ${match[1]}, which is not the core`,
        })
    }
  }
  return { findings, read: sources.length }
}

export function verifyHostNamesNoPlugin() {
  const one = findPluginNames(readSources(EDITOR_SRC))
  requireMatches('Sources of the editor', one.read)
  failOnFindings('A source of the host spells out a plugin id or a reader tab kind', one.findings)
}

export function verifyOptionsAreAppliedByTheCore() {
  const one = findOptionsReaders(readSources(EDITOR_SRC))
  requireMatches('Sources of the editor', one.read)
  failOnFindings('A source of the window reads optionsBy outside the settings api', one.findings)
}

export function verifyPaletteRunsCommands() {
  const one = findPaletteBypassingCommands(readSources(EDITOR_SRC))
  requireMatches('Sources of the editor', one.read)
  failOnFindings('The palette opens a chosen result on a path of its own', one.findings)
}

export function verifySettingsPageHasNoShellSection() {
  const one = findShellSettingsSections(listShippedFiles([`${EDITOR_SRC}/pages/settings`]))
  requireMatches('Files of the settings page', one.read)
  failOnFindings('The settings page keeps a section of its own for the shell', one.findings)
}

export function verifyMobilePagePolicy() {
  const html = fs.readFileSync(path.join(REPO_ROOT, MOBILE, 'index.html'), 'utf8')
  const one = findMobilePageViolations(html, readSources(`${MOBILE}/src`))
  requireMatches('Sources of the phone window', one.read)
  failOnFindings('The page of the phone loads more than its policy allows', one.findings)
}

export const WINDOW_SOURCES_VERIFIERS = [
  {
    id: 'host-names-no-plugin',
    scope: 'apps',
    module: 'boundary',
    name: 'The Editor Names No Plugin Id and No Plugin Reader Tab Kind',
    run: verifyHostNamesNoPlugin,
  },
  {
    id: 'options-applied-by-the-core',
    scope: 'apps',
    module: 'boundary',
    name: 'Only the Settings Api of the Editor Reads optionsBy',
    run: verifyOptionsAreAppliedByTheCore,
  },
  {
    id: 'palette-runs-commands',
    scope: 'apps',
    module: 'boundary',
    name: 'A Palette Result Opens Only Through the Command It Is',
    run: verifyPaletteRunsCommands,
  },
  {
    id: 'settings-page-has-no-shell-section',
    scope: 'apps',
    module: 'boundary',
    name: 'The Settings Page Has No Section of Its Own for the Shell',
    run: verifySettingsPageHasNoShellSection,
  },
  {
    id: 'mobile-page-policy',
    scope: 'apps',
    module: 'boundary',
    name: 'The Page of the Phone Carries Its Exact Content Security Policy and Names No Host but the Core',
    run: verifyMobilePagePolicy,
  },
]
