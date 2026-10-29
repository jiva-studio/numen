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
          targets: ['desktop.editor'],
          frontend: true,
          settings: [{ title: 'Glow', palette: true }, { title: 'Hidden' }],
          commands: [
            { title: 'Light a file', contentTypes: ['text/plain'] },
            { title: 'Light the room' },
            { title: 'Secret', hiddenInPalette: true },
            { title: 'Dim a text', when: { hasOwnText: true } },
          ],
        },
        {
          'index.ts': [
            "import { WORDS } from './words'",
            'context.registerCommand({',
            "  id: 'lamp',",
            '  title: WORDS.lamp,',
            "  shortcut: 'mod+shift+l',",
            '})',
            'context.registerCommand({',
            "  id: 'wick',",
            '  title: WORDS.wick,',
            "  scope: 'note',",
            '})',
          ].join('\n'),
          'words.ts':
            "export const WORDS = {\n  lamp: 'New lamp',\n  wick: (name) => `Trim ${name}`,\n}\n",
        },
      )
      await writePlugin(root, 'tools-elsewhere', { id: 'tools.elsewhere', targets: ['mobile'] })
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
            { id: 'lamp', text: 'New lamp', group: 'window', chord: { letter: 'l', shift: true } },
            { id: 'wick', text: 'Trim *name*', group: 'note' },
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
        { id: 'tools.bell', name: 'Bell', targets: ['desktop.editor'], frontend: true },
        {
          'index.ts':
            "import { WORDS } from './words'\ncontext.registerCommand({\n  id: 'ring',\n  title: WORDS.ring,\n  shortcut: 'alt+f4',\n})\n",
          'words.ts': "export const WORDS = {\n  ring: 'Ring',\n}\n",
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

test('a command with no title this can read is refused', () =>
  withPlugins(
    (root) =>
      writePlugin(
        root,
        'tools-drum',
        { id: 'tools.drum', name: 'Drum', targets: ['desktop.editor'], frontend: true },
        { 'index.ts': "context.registerCommand({\n  id: 'beat',\n  title: 'Beat',\n})\n" },
      ),
    (at) =>
      assert.rejects(getPluginCommands('desktop.editor', at), {
        name: 'Error',
        message: 'tools-drum registers a command this cannot read',
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
  assert.ok(agent.commands.some((one) => one.id === 'agent' && one.chord))
})
