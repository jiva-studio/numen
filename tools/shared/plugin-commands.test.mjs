/**
 * The commands of the plugins in the tree: each is declared under `contributes.commands`, and the
 * frontend registers its id, its handler and, where it has a menu, its items. A shared command the
 * manifest handles under `contributes.handlers` is registered under its id the same way. Every
 * command and shared handler in a manifest has a handler in the frontend or backend, and every
 * handler registered in the frontend or backend is declared in the manifest.
 */
import assert from 'node:assert/strict'
import { existsSync, mkdirSync, mkdtempSync, rmSync, writeFileSync } from 'node:fs'
import { tmpdir } from 'node:os'
import { join } from 'node:path'
import test from 'node:test'
import { PLUGINS_DIR, readPluginManifests } from './manifests.mjs'
import {
  UNREADABLE_FIELD,
  findPluginCommandMismatches,
  listRegistrations,
} from './registrations.mjs'

const listPluginRegistrations = (name) => {
  const frontend = join(PLUGINS_DIR, name, 'frontend')
  return existsSync(frontend) ? listRegistrations(frontend) : []
}

/** The shared commands a manifest handles under `contributes.handlers`. */
const getHandledIds = (document) =>
  new Set((document.contributes?.handlers ?? []).map((one) => one.command))

const withPluginsFixture = (setup, verify) => {
  const root = mkdtempSync(join(tmpdir(), 'plugin-commands-'))
  try {
    setup(root)
    verify(root)
  } finally {
    rmSync(root, { recursive: true, force: true })
  }
}

const writePluginFixture = (root, name, { manifest, frontend, backend }) => {
  const dir = join(root, name)
  mkdirSync(dir, { recursive: true })
  writeFileSync(
    join(dir, 'manifest.json'),
    JSON.stringify({
      id: `md.numen.${name.replaceAll('-', '.')}`,
      name,
      version: '2026.11.1',
      targets: ['desktop.editor'],
      ...manifest,
    }),
  )
  if (frontend) {
    mkdirSync(join(dir, 'frontend'), { recursive: true })
    for (const [file, text] of Object.entries(frontend)) {
      writeFileSync(join(dir, 'frontend', file), text)
    }
  }
  if (backend) {
    mkdirSync(join(dir, 'backend'), { recursive: true })
    for (const [file, text] of Object.entries(backend)) {
      writeFileSync(join(dir, 'backend', file), text)
    }
  }
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

test('a frontend registers a command with an id, a handler and its items or none', () => {
  for (const { name } of readPluginManifests()) {
    for (const { file, fields } of listPluginRegistrations(name)) {
      assert.ok(!fields.includes(UNREADABLE_FIELD), `${file} registers options this cannot read`)
      const allowed = fields.includes('listItems')
        ? ['handler', 'id', 'listItems']
        : ['handler', 'id']
      assert.deepEqual([...fields].sort(), allowed, `${file} registers ${fields.join(', ')}`)
    }
  }
})

test('every command a frontend registers is declared or handled by its manifest', () => {
  for (const { name, document } of readPluginManifests()) {
    const declared = new Set((document.contributes?.commands ?? []).map((one) => one.id))
    const handled = getHandledIds(document)
    for (const { file, id } of listPluginRegistrations(name)) {
      assert.ok(
        declared.has(id) || handled.has(id),
        `${file} registers ${id}, which ${name} does not declare`,
      )
    }
  }
})

test('every declared command and handler of a plugin has a handler, and every handler is declared', () => {
  const mismatches = findPluginCommandMismatches()
  assert.deepEqual(mismatches, [], mismatches.join('\n'))
})

test('findPluginCommandMismatches reports a declared command with no handler', () =>
  withPluginsFixture(
    (root) => {
      writePluginFixture(root, 'views-sample', {
        manifest: {
          frontend: { entrypoint: 'dist/index.js' },
          contributes: {
            commands: [
              { id: 'sample.open', title: 'Open sample', icon: 'file' },
              { id: 'sample.close', title: 'Close sample', icon: 'x' },
            ],
          },
        },
        frontend: {
          'index.ts': "context.register.command({ id: 'sample.open', handler: () => undefined })\n",
        },
      })
    },
    (root) => {
      assert.deepEqual(findPluginCommandMismatches(root), [
        'views-sample: command "sample.close" is declared in manifest.contributes.commands, missing handler in code',
      ])
    },
  ))

test('findPluginCommandMismatches reports a declared shared handler with no handler in code', () =>
  withPluginsFixture(
    (root) => {
      writePluginFixture(root, 'engines-sample', {
        manifest: {
          backend: { entrypoint: 'bin/engines-sample' },
          contributes: {
            handlers: [{ command: 'app:file.createText', title: 'Extract text' }],
          },
        },
        backend: {
          'server.go': 'package service\nfunc newCommands() {}\n',
        },
      })
    },
    (root) => {
      assert.deepEqual(findPluginCommandMismatches(root), [
        'engines-sample: handler "app:file.createText" is declared in manifest.contributes.handlers, missing handler in code',
      ])
    },
  ))

test('findPluginCommandMismatches reports a frontend command registration missing from the manifest', () =>
  withPluginsFixture(
    (root) => {
      writePluginFixture(root, 'views-sample', {
        manifest: {
          frontend: { entrypoint: 'dist/index.js' },
          contributes: {
            commands: [{ id: 'sample.open', title: 'Open sample', icon: 'file' }],
          },
        },
        frontend: {
          'index.ts': [
            "context.register.command({ id: 'sample.open', handler: () => undefined })",
            "context.register.command({ id: 'sample.typo', handler: () => undefined })",
          ].join('\n'),
        },
      })
    },
    (root) => {
      const mismatches = findPluginCommandMismatches(root)
      assert.equal(mismatches.length, 1)
      assert.match(
        mismatches[0],
        /^views-sample: command "sample\.typo" is registered in frontend \(.+\), missing from manifest$/,
      )
    },
  ))

test('findPluginCommandMismatches reports a backend command registration missing from the manifest', () =>
  withPluginsFixture(
    (root) => {
      writePluginFixture(root, 'tools-sample', {
        manifest: {
          backend: { entrypoint: 'bin/tools-sample' },
          contributes: {
            commands: [{ id: 'sample.run', title: 'Run sample', icon: 'play' }],
          },
        },
        backend: {
          'commands.go': [
            'package service',
            'import "github.com/jiva-studio/numen/libs/plugin/sdk-go/command"',
            'func newCommands() *command.Registry {',
            '\tcommands := command.New()',
            '\t_ = commands.Add("sample.run", nil)',
            '\t_ = commands.Add("sample.extra", nil)',
            '\treturn commands',
            '}',
          ].join('\n'),
        },
      })
    },
    (root) => {
      const mismatches = findPluginCommandMismatches(root)
      assert.equal(mismatches.length, 1)
      assert.match(
        mismatches[0],
        /^tools-sample: command "sample\.extra" is registered in backend \(.+\), missing from manifest$/,
      )
    },
  ))
