/**
 * What the registration reader finds in layouts a text pattern misses.
 */
import assert from 'node:assert/strict'
import test from 'node:test'
import { UNREADABLE_FIELD, readRegistrations } from './registrations.mjs'

test('a call with the options on one line is read', () => {
  assert.deepEqual(readRegistrations("ctx.registerCommand({ id: 'a.b', handler })"), [
    { fields: ['id', 'handler'], id: 'a.b' },
  ])
})

test('a call with other indentation and a quoted key is read', () => {
  const source = "ctx.registerCommand({\n\t\t'id': 'a.b',\n\t\thandler: () => 1,\n\t})"
  assert.deepEqual(readRegistrations(source), [{ fields: ['id', 'handler'], id: 'a.b' }])
})

test('options held in a constant, with the id in another constant, are read', () => {
  const source = [
    "const ID = 'a.b'",
    'const options = { id: ID, handler: () => 1, title: "T" } as const',
    'registerCommand(options)',
  ].join('\n')
  assert.deepEqual(readRegistrations(source), [{ fields: ['id', 'handler', 'title'], id: 'a.b' }])
})

test('a call with options this cannot follow is listed as unreadable', () => {
  assert.deepEqual(readRegistrations('ctx.registerCommand(build())'), [
    { fields: [UNREADABLE_FIELD], id: undefined },
  ])
})

test('a mention in a comment or a string is not a call', () => {
  assert.deepEqual(
    readRegistrations("// registerCommand({ id: 'x' })\nconst s = 'registerCommand({})'"),
    [],
  )
})
