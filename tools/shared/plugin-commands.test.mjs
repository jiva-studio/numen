/**
 * The commands of the plugins in the tree: each is declared under `contributes.commands`, and the
 * frontend registers only the id and the handler.
 */
import assert from 'node:assert/strict'
import { existsSync, readFileSync, readdirSync } from 'node:fs'
import { join } from 'node:path'
import test from 'node:test'
import { PLUGINS_DIR, readPluginManifests } from './manifests.mjs'

const REGISTRATION = /registerCommand\(\{(.*?)\n\s{2}\}\)/gs
const FIELD = /^\s{4}(\w+):/gm

const listSources = (dir) =>
  readdirSync(dir, { withFileTypes: true }).flatMap((entry) => {
    const path = join(dir, entry.name)
    if (entry.isDirectory()) return entry.name === 'node_modules' ? [] : listSources(path)
    return /\.ts$/.test(entry.name) && !/\.(test|stories)\.ts$/.test(entry.name) ? [path] : []
  })

const listRegistrations = (name) => {
  const frontend = join(PLUGINS_DIR, name, 'frontend')
  if (!existsSync(frontend)) return []
  return listSources(frontend).flatMap((file) =>
    [...readFileSync(file, 'utf8').matchAll(REGISTRATION)].map((match) => ({
      file,
      fields: [...match[1].matchAll(FIELD)].map((one) => one[1]),
      id: /id:\s*'([^']+)'/.exec(match[1])?.[1],
    })),
  )
}

test('no manifest of the tree declares commands at the top level', () => {
  for (const { name, document } of readPluginManifests()) {
    assert.equal(document.commands, undefined, `${name} declares commands at the top level`)
  }
})

test('every command a manifest declares has a title and an icon name', () => {
  for (const { name, document } of readPluginManifests()) {
    for (const command of document.contributes?.commands ?? []) {
      assert.ok(command.title, `${name}: ${command.id} has no title`)
      assert.match(
        command.icon ?? '',
        /^[a-z][a-z0-9]*(-[a-z0-9]+)*$/,
        `${name}: ${command.id} has no icon name`,
      )
    }
  }
})

test('a frontend registers a command with an id and a handler only', () => {
  for (const { name } of readPluginManifests()) {
    for (const { file, fields } of listRegistrations(name)) {
      assert.deepEqual(
        [...fields].sort(),
        ['handler', 'id'],
        `${file} registers ${fields.join(', ')}`,
      )
    }
  }
})

test('every command a frontend registers is declared by its manifest', () => {
  for (const { name, document } of readPluginManifests()) {
    const declared = new Set((document.contributes?.commands ?? []).map((one) => one.id))
    for (const { file, id } of listRegistrations(name)) {
      assert.ok(declared.has(id), `${file} registers ${id}, which ${name} does not declare`)
    }
  }
})
