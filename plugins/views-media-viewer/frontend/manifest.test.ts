/**
 * The command the manifest declares to record audio, which the palette lists under its title.
 */
import { readFileSync } from 'node:fs'
import { resolve } from 'node:path'
import { describe, expect, it } from 'vitest'

const { contributes } = JSON.parse(
  readFileSync(resolve(import.meta.dirname, '../manifest.json'), 'utf8'),
) as { contributes: { commands: readonly { id: string; title: string; icon: string }[] } }

describe('the manifest', () => {
  it('declares the command that opens the recorder under the title Record audio', () => {
    expect(contributes.commands.find((one) => one.id === 'recording.create')).toMatchObject({
      title: 'Record audio',
      icon: 'mic',
    })
  })
})
