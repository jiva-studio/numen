/**
 * What the plugin commands are read as, over a plugins folder made up here,
 * and that the walk over the tree finds the commands it must.
 */
import assert from 'node:assert/strict'
import { mkdtemp, mkdir, rm, writeFile } from 'node:fs/promises'
import { tmpdir } from 'node:os'
import { join } from 'node:path'
import { pathToFileURL } from 'node:url'
import test from 'node:test'
import { ReadError, getPluginCommands } from './commands.mjs'

const writePlugin = async (root, name, manifest, frontend) => {
  await mkdir(join(root, name, 'frontend'), { recursive: true })
  await writeFile(join(root, name, 'manifest.json'), JSON.stringify(manifest))
  for (const [file, text] of Object.entries(frontend ?? {})) {
    await writeFile(join(root, name, 'frontend', file), text)
  }
}

const withPlugins = async (fill, check) => {
  const root = await mkdtemp(join(tmpdir(), 'ci-commands-'))
  try {
    await fill(root)
    await check(pathToFileURL(`${root}/`))
  } finally {
    await rm(root, { recursive: true, force: true })
  }
}

test('a plugin is read with its settings, declared and registered commands', () =>
  withPlugins(
    async (root) => {
      await writePlugin(
        root,
        'tools-lantern',
        {
          id: 'tools.lantern',
          name: 'Lantern',
          version: '1.0.0',
          targets: ['desktop.editor'],
          frontend: { entrypoint: 'dist/index.js' },
          settings: [{ title: 'Glow', palette: true }, { title: 'Hidden' }],
          contributes: {
            commands: [
              { id: 'file.light', title: 'Light a file', contentTypes: ['text/plain'] },
              { id: 'room.light', title: 'Light the room' },
              {
                id: 'text.dim',
                title: 'Dim a text',
                when: [{ key: 'version.plugin', is: ['md.numen.tools.lantern'] }],
              },
              { id: 'lamp.create', title: 'New lamp', shortcut: 'mod+shift+l' },
              { id: 'wick.trim', title: 'Trim the wick', scope: 'note' },
            ],
          },
        },
        {
          'index.ts': [
            'context.registerCommand({',
            "  id: 'lamp.create',",
            '  handler: () => undefined,',
            '})',
            'context.registerCommand({',
            "  id: 'wick.trim',",
            '  handler: () => undefined,',
            '})',
          ].join('\n'),
        },
      )
      await writePlugin(root, 'tools-elsewhere', {
        id: 'tools.elsewhere',
        name: 'Elsewhere',
        version: '1.0.0',
        targets: ['mobile'],
      })
    },
    async (at) => {
      assert.deepEqual(await getPluginCommands('desktop.editor', at), [
        {
          id: 'tools.lantern',
          settings: [{ text: 'Lantern: Glow', group: 'settings' }],
          declared: [
            { text: 'Light a file', group: 'file' },
            { text: 'Light the room', group: 'window' },
          ],
          commands: [
            {
              id: 'lamp.create',
              text: 'New lamp',
              group: 'window',
              chord: { letter: 'l', shift: true },
            },
            { id: 'wick.trim', text: 'Trim the wick', group: 'note' },
          ],
        },
      ])
    },
  ))

test('a shortcut the window cannot draw is refused', () =>
  withPlugins(
    (root) =>
      writePlugin(
        root,
        'tools-bell',
        {
          id: 'tools.bell',
          name: 'Bell',
          version: '1.0.0',
          targets: ['desktop.editor'],
          frontend: { entrypoint: 'dist/index.js' },
          contributes: { commands: [{ id: 'ring.create', title: 'Ring', shortcut: 'alt+f4' }] },
        },
        {
          'index.ts':
            "context.registerCommand({\n  id: 'ring.create',\n  handler: () => undefined,\n})\n",
        },
      ),
    (at) =>
      assert.rejects(getPluginCommands('desktop.editor', at), (error) => {
        assert.ok(error instanceof ReadError)
        assert.equal(
          error.message,
          "tools-bell declares the shortcut 'alt+f4', which the window draws as nothing",
        )
        return true
      }),
  ))

test('a command the frontend registers and the manifest does not declare is refused', () =>
  withPlugins(
    (root) =>
      writePlugin(
        root,
        'tools-drum',
        {
          id: 'tools.drum',
          name: 'Drum',
          version: '1.0.0',
          targets: ['desktop.editor'],
          frontend: { entrypoint: 'dist/index.js' },
        },
        {
          'index.ts':
            "context.registerCommand({\n  id: 'beat.create',\n  handler: () => undefined,\n})\n",
        },
      ),
    (at) =>
      assert.rejects(getPluginCommands('desktop.editor', at), {
        name: 'Error',
        message: 'tools-drum registers beat.create, which its manifest does not declare',
      }),
  ))

test('a registration this cannot read an id out of is refused', () =>
  withPlugins(
    (root) =>
      writePlugin(
        root,
        'tools-flute',
        {
          id: 'tools.flute',
          name: 'Flute',
          version: '1.0.0',
          targets: ['desktop.editor'],
          frontend: { entrypoint: 'dist/index.js' },
        },
        { 'index.ts': 'context.registerCommand({\n  handler: () => undefined,\n})\n' },
      ),
    (at) =>
      assert.rejects(getPluginCommands('desktop.editor', at), {
        message: 'tools-flute registers a command this cannot read',
      }),
  ))

test('a manifest that does not fit the schema is refused', () =>
  withPlugins(
    (root) =>
      writePlugin(root, 'tools-gong', {
        id: 'tools.gong',
        name: 'Gong',
        version: '1.0.0',
        targets: 'desktop.editor',
      }),
    (at) =>
      assert.rejects(getPluginCommands('desktop.editor', at), (error) => {
        assert.ok(error instanceof ReadError)
        assert.match(error.message, /does not fit the manifest schema/)
        return true
      }),
  ))

/**
 * A walk that read the wrong folder finds nothing, so commands it must have
 * found in the tree are named.
 */
test('the plugins of the tree are read with their commands', async () => {
  const plugins = await getPluginCommands('desktop.editor')
  const agent = plugins.find((one) => one.id === 'md.numen.views.agent.chat')
  assert.ok(agent, 'the agent chat plugin is read')
  assert.ok(agent.commands.some((one) => one.id === 'chat.open' && one.chord))
})
