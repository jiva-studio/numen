/**
 * The menus of the plugins in the tree: each placement names a command the manifest declares or
 * the editor provides, in a place the window draws, and the window and the host name no plugin.
 */
import assert from 'node:assert/strict'
import { readFileSync, readdirSync, statSync } from 'node:fs'
import { join, relative } from 'node:path'
import { fileURLToPath } from 'node:url'
import test from 'node:test'
import { readPluginManifests } from './manifests.mjs'

const ROOT = fileURLToPath(new URL('../../', import.meta.url))
const EDITOR_SRC = join(ROOT, 'apps/desktop/editor/src')
const LOCATIONS = ['titleBar.commands', 'searchResult']
const APP_PREFIX = 'app:'

const isTestFile = (name) => /\.(test|stories|spec)\.[cm]?[jt]s$/.test(name)

/** Every source file of the folder that is neither a test nor a story, as a path. */
const listSources = (folder, extensions) =>
  readdirSync(folder).flatMap((name) => {
    const path = join(folder, name)
    if (name === 'node_modules' || name === 'dist') return []
    if (statSync(path).isDirectory()) return listSources(path, extensions)
    return extensions.some((one) => name.endsWith(one)) && !isTestFile(name) ? [path] : []
  })

const readEditorSources = () =>
  listSources(EDITOR_SRC, ['.ts', '.vue']).filter(
    (file) => !relative(EDITOR_SRC, file).startsWith('testing'),
  )

/** The commands the editor provides, by full id, with whether the table gives each an icon name. */
const readEditorCommands = () => {
  const ids = readFileSync(join(EDITOR_SRC, 'shared/commandIds.ts'), 'utf8')
  const byKey = new Map(
    [...ids.matchAll(/^\s+(\w+): formatAppCommandId\('([^']+)'\)/gm)].map((one) => [
      one[1],
      `${APP_PREFIX}${one[2]}`,
    ]),
  )
  const table = listSources(join(EDITOR_SRC, 'features/command-palette/lib'), ['.ts'])
    .map((file) => readFileSync(file, 'utf8'))
    .join('\n')
  const commands = new Map()
  for (const [key, fullId] of byKey) {
    const starts = [...table.matchAll(new RegExp(`id: COMMAND\\.${key},`, 'g'))].map(
      (one) => one.index,
    )
    const hasIconName = starts.some((start) => {
      const block = table.slice(start, table.indexOf('isAvailable', start))
      return /\biconName:\s*'[a-z][a-z0-9-]*'/.test(block)
    })
    commands.set(fullId, { hasIconName })
  }
  return commands
}

test('neither the host module nor the window names a plugin', () => {
  const files = [join(ROOT, 'libs/plugin-host-ts/src/menus.ts'), ...readEditorSources()]
  for (const file of files) {
    assert.doesNotMatch(
      readFileSync(file, 'utf8'),
      /md\.numen\./,
      `${relative(ROOT, file)} names a plugin`,
    )
  }
})

test('no source records where a note found by its name is shown', () => {
  const folders = ['apps', 'libs', 'plugins'].map((one) => join(ROOT, one))
  for (const file of folders.flatMap((one) => listSources(one, ['.ts', '.vue', '.go', '.proto']))) {
    if (file.endsWith('_test.go') || file.includes(`${join('libs', 'protocol', 'gen')}`)) continue
    assert.doesNotMatch(
      readFileSync(file, 'utf8'),
      /isNoteDestination|noteDestination/,
      `${relative(ROOT, file)} carries the mark of a note destination`,
    )
  }
})

test('a command declaration says nothing of where it is shown', () => {
  const proto = readFileSync(join(ROOT, 'libs/protocol/proto/numen/v1/manifest.proto'), 'utf8')
  const command = proto.slice(proto.indexOf('message ManifestCommand {'))
  const body = command.slice(0, command.indexOf('\n}'))
  assert.doesNotMatch(
    body,
    /\b(menus?|location)\b\s*=/,
    'ManifestCommand has a field about a place',
  )
  const types = readFileSync(join(ROOT, 'libs/plugin-sdk-ts/src/types.ts'), 'utf8')
  const registration = types.slice(types.indexOf('export interface CommandRegistration'))
  const registrationBody = registration.slice(0, registration.indexOf('\n}'))
  assert.doesNotMatch(
    registrationBody,
    /\b(menus?|location)\??:/,
    'CommandRegistration has a field about a place',
  )
})

test('every placement of a manifest of the tree is in a known place and names a command', () => {
  const editorCommands = readEditorCommands()
  for (const { name, document } of readPluginManifests()) {
    const declared = new Map((document.contributes?.commands ?? []).map((one) => [one.id, one]))
    for (const menu of document.contributes?.menus ?? []) {
      const where = `${name}: ${menu.location} ${menu.command}`
      assert.ok(
        LOCATIONS.includes(menu.location),
        `${where} is in a place the window does not draw`,
      )
      if (menu.command.startsWith(APP_PREFIX)) {
        assert.ok(editorCommands.has(menu.command), `${where} is no command of the editor`)
        if (menu.location === 'titleBar.commands') {
          assert.ok(
            editorCommands.get(menu.command)?.hasIconName,
            `${where} has no icon name in the editor`,
          )
        }
      } else {
        const command = declared.get(menu.command)
        assert.ok(command, `${where} is not declared by the manifest`)
        if (menu.location === 'titleBar.commands') {
          assert.match(
            command.icon ?? '',
            /^[a-z][a-z0-9]*(-[a-z0-9]+)*$/,
            `${where} has no icon name`,
          )
        }
      }
    }
  }
})

test('the plugin that edits notes places nothing in the menu of a search result', () => {
  const notes = readPluginManifests().find(({ document }) => document.id === 'md.numen.notes')
  assert.ok(notes, 'the notes plugin is not in the tree')
  const placed = (notes.document.contributes?.menus ?? []).filter(
    (one) => one.location === 'searchResult',
  )
  assert.deepEqual(placed, [])
})

test('the plugin documentation describes menus', () => {
  const docs = readFileSync(join(ROOT, 'docs/plugins/README.md'), 'utf8')
  const start = docs.search(/^#{2,3} Menus\s*$/m)
  assert.ok(start >= 0, 'the documentation has no Menus section')
  const rest = docs.slice(start + 1)
  const end = rest.search(/\n#{2,3} /)
  const section = end < 0 ? rest : rest.slice(0, end)
  for (const word of [
    'contributes.menus',
    'titleBar.commands',
    'searchResult',
    'group',
    'order',
    'alphabetical',
  ]) {
    assert.ok(section.includes(word), `the Menus section does not mention ${word}`)
  }
  assert.match(
    docs,
    /\|\s*`contributes\.menus`/,
    'the table of contributions has no row for contributes.menus',
  )
})

test('the window frame of the shell names no plugin', () => {
  const folder = join(ROOT, 'libs/ui/shell/src/features/window-frame')
  for (const file of listSources(folder, ['.ts', '.vue'])) {
    assert.doesNotMatch(
      readFileSync(file, 'utf8'),
      /md\.numen\./,
      `${relative(ROOT, file)} names a plugin`,
    )
  }
})

test('the docs say how the title bar draws an entry', () => {
  const readme = readFileSync(join(ROOT, 'docs/plugins/README.md'), 'utf8')
  const menus = readme.slice(readme.indexOf('\n## Menus'))
  const section = menus.slice(0, menus.indexOf('\n## ', 4))
  const titleBar = section
    .split('\n')
    .filter((line) => line.includes('titleBar.commands'))
    .join('\n')
  assert.match(titleBar, /button/i, 'the section does not say an entry is a button')
  assert.match(titleBar, /vault is open/i, 'the section does not say when the buttons are shown')
  assert.match(titleBar, /shortcut/i, 'the section does not say the shortcut is in the hover text')
})
