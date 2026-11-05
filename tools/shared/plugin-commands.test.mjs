/**
 * The commands of the plugins in the tree: each is declared under `contributes.commands`, and the
 * frontend registers only the id and the handler.
 */
import assert from 'node:assert/strict'
import { existsSync } from 'node:fs'
import { join } from 'node:path'
import test from 'node:test'
import { PLUGINS_DIR, readPluginManifests } from './manifests.mjs'
import { UNREADABLE_FIELD, listRegistrations } from './registrations.mjs'

const listPluginRegistrations = (name) => {
  const frontend = join(PLUGINS_DIR, name, 'frontend')
  return existsSync(frontend) ? listRegistrations(frontend) : []
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
    for (const { file, fields } of listPluginRegistrations(name)) {
      assert.ok(!fields.includes(UNREADABLE_FIELD), `${file} registers options this cannot read`)
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
    for (const { file, id } of listPluginRegistrations(name)) {
      assert.ok(declared.has(id), `${file} registers ${id}, which ${name} does not declare`)
    }
  }
})

test('a plugin with a frontend and no backend that declares commands has a registration found', () => {
  for (const { name, document } of readPluginManifests()) {
    if (!document.frontend || document.backend) continue
    if ((document.contributes?.commands ?? []).length === 0) continue
    assert.ok(
      listPluginRegistrations(name).length > 0,
      `${name} declares commands and no registration was read from its frontend`,
    )
  }
})
