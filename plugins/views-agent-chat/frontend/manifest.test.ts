/**
 * The permissions the plugin's manifest declares, which the window holds every call to, and the
 * windows that load it.
 */
import { readFileSync } from 'node:fs'
import { resolve } from 'node:path'
import { describe, expect, it } from 'vitest'

const { permissions, targets } = JSON.parse(
  readFileSync(resolve(import.meta.dirname, '../manifest.json'), 'utf8'),
) as { permissions: readonly string[]; targets: readonly string[] }

describe('the manifest', () => {
  it('declares vault:read, which asking the agent needs', () => {
    expect(permissions).toStrictEqual(['vault:read'])
  })

  it('is loaded by the editor and by the flashcards window, whose review mounts its conversation', () => {
    expect(targets).toStrictEqual(['desktop.editor', 'desktop.flashcards'])
  })
})

describe('the settings', () => {
  it('declare the depth of reasoning the plugin preselects', () => {
    const { settings } = JSON.parse(
      readFileSync(resolve(import.meta.dirname, '../manifest.json'), 'utf8'),
    ) as { settings?: readonly { key: string; type: string; default: unknown }[] }
    expect(settings).toStrictEqual([
      expect.objectContaining({ key: 'reasoning', type: 'string', default: '' }),
    ])
  })
})

describe('the icons', () => {
  const { contributes } = JSON.parse(
    readFileSync(resolve(import.meta.dirname, '../manifest.json'), 'utf8'),
  ) as {
    contributes: {
      commands: readonly { id: string; icon: string }[]
      tabKinds: readonly { kind: string; icon: string }[]
    }
  }

  it('draw the command that opens the chat as a speech bubble', () => {
    expect(contributes.commands.find((one) => one.id === 'chat.open')?.icon).toBe('message-square')
  })

  it('keep the robot on the command that asks about a note', () => {
    expect(contributes.commands.find((one) => one.id === 'note.ask')?.icon).toBe('bot')
  })

  it('keep the robot on the agent tab', () => {
    expect(contributes.tabKinds.find((one) => one.kind === 'agent')?.icon).toBe('bot')
  })
})
