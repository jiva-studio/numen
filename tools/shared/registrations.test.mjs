/**
 * What the registration reader finds in layouts a text pattern misses.
 */
import assert from 'node:assert/strict'
import { mkdtempSync, rmSync, writeFileSync } from 'node:fs'
import { tmpdir } from 'node:os'
import { join } from 'node:path'
import test from 'node:test'
import { UNREADABLE_FIELD, listBackendRegistrations, readRegistrations } from './registrations.mjs'

test('a call with the options on one line is read', () => {
  assert.deepEqual(readRegistrations("ctx.register.command({ id: 'a.b', handler })"), [
    { fields: ['id', 'handler'], id: 'a.b' },
  ])
})

test('a call with other indentation and a quoted key is read', () => {
  const source = "ctx.register.command({\n\t\t'id': 'a.b',\n\t\thandler: () => 1,\n\t})"
  assert.deepEqual(readRegistrations(source), [{ fields: ['id', 'handler'], id: 'a.b' }])
})

test('options held in a constant, with the id in another constant, are read', () => {
  const source = [
    "const ID = 'a.b'",
    'const options = { id: ID, handler: () => 1, title: "T" } as const',
    'register.command(options)',
  ].join('\n')
  assert.deepEqual(readRegistrations(source), [{ fields: ['id', 'handler', 'title'], id: 'a.b' }])
})

test('a call with options this cannot follow is listed as unreadable', () => {
  assert.deepEqual(readRegistrations('ctx.register.command(build())'), [
    { fields: [UNREADABLE_FIELD], id: undefined },
  ])
})

test('a mention in a comment or a string is not a call', () => {
  assert.deepEqual(
    readRegistrations("// register.command({ id: 'x' })\nconst s = 'register.command({})'"),
    [],
  )
})

test('backend registrations are read from string literals, constants, shared commands and map loops', () => {
  const root = mkdtempSync(join(tmpdir(), 'registrations-go-'))
  try {
    const file = join(root, 'commands.go')
    writeFileSync(
      file,
      [
        'package service',
        'import "github.com/jiva-studio/numen/libs/plugin/sdk-go/command"',
        'const (',
        '\tcommandOpen = "item.open"',
        '\tcommandImport = command.ImportLink',
        ')',
        'func newCommands() *command.Registry {',
        '\tcommands := command.New()',
        '\t_ = commands.Add("direct.run", nil)',
        '\t_ = commands.Add(commandOpen, nil)',
        '\t_ = commands.Add(command.CreateText, nil)',
        '\tfor id, run := range map[string]command.Run{',
        '\t\tcommandImport: nil,',
        '\t\t"mapped.run": nil,',
        '\t} {',
        '\t\t_ = commands.Add(id, run)',
        '\t}',
        '\treturn commands',
        '}',
      ].join('\n'),
    )
    assert.deepEqual(
      listBackendRegistrations(root).map((one) => one.id),
      ['direct.run', 'item.open', 'app:file.createText', 'app:link.import', 'mapped.run'],
    )
  } finally {
    rmSync(root, { recursive: true, force: true })
  }
})
