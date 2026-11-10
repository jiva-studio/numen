/**
 * The demo window answers every command the real menus and palette send. The ids the plugins send
 * are read out of the plugins' sources, and each must be one the demo declares and one its
 * command runner handles.
 */
import assert from 'node:assert/strict'
import { readFileSync } from 'node:fs'
import test from 'node:test'

const root = new URL('../../../', import.meta.url)
const read = (path) => readFileSync(new URL(path, root), 'utf8')

const demo = read('apps/landing/src/components/demo/lib/commands.ts')
const runner = read('apps/landing/src/components/demo/model/commands.ts') + read('apps/landing/src/components/demo/lib/palette.ts')
const declared = new Map([...demo.matchAll(/^\s+([A-Za-z]+): '([^']+)',$/gm)].map((m) => [m[2], m[1]]))

/** The full id of every command a plugin registers: its plugin id and the id it registers. */
const readRegistered = (plugin) => {
  const manifest = JSON.parse(read(`plugins/${plugin}/manifest.json`))
  const source = read(`plugins/${plugin}/frontend/index.ts`)
  const ids = [...source.matchAll(/registerCommand\(\{\s+id: '([^']+)'/g)].map((m) => m[1])
  return ids.map((id) => `${manifest.id}:${id}`)
}

const readAppIds = (path) => [...read(path).matchAll(/'(app:[a-z.-]+)'/g)].map((m) => m[1])

test('the demo declares every command the plex menu sends to the window', () => {
  for (const id of readAppIds('plugins/views-plex-graph/frontend/lib/commands.ts')) {
    assert.ok(declared.has(id), `the demo does not declare ${id}`)
  }
})

test('the demo declares the command the file tree sends to remove files', () => {
  for (const id of readAppIds('plugins/views-file-tree/frontend/lib/menu.ts')) {
    assert.ok(declared.has(id), `the demo does not declare ${id}`)
  }
})

test('the demo declares every command the plugins it draws register', () => {
  for (const plugin of ['views-file-tree', 'views-plex-graph', 'views-agent-chat']) {
    for (const id of readRegistered(plugin)) {
      assert.ok(declared.has(id), `the demo does not declare ${id}`)
    }
  }
})

test('the demo runs each command it declares', () => {
  for (const name of declared.values()) {
    assert.ok(runner.includes(`DEMO_COMMAND.${name}`), `the command runner does not handle ${name}`)
  }
})
