/**
 * The text of a recording as the shared command `app:file.createText` creates it.
 */
import { describe, expect, it } from 'vitest'
import { createMockCommandsHost } from '@numen/plugin-sdk/testing'
import { CREATE_TEXT, createTextCommand } from './text'

const TARGET = { path: 'talks/Ants.mp3', title: 'Ants', isFolder: false, others: [], tab: '' }

describe('createTextCommand', () => {
  it('can create the text of a file a plugin handles the command for', async () => {
    const commands = createMockCommandsHost()
    commands.setItems(CREATE_TEXT, [{ id: 'md.numen.transcription', title: 'Transcribe' }])

    expect(await createTextCommand(() => commands).canCreateText('talks/Ants.mp3')).toBe(true)
  })

  it('cannot create the text where no plugin handles the command', async () => {
    const commands = createMockCommandsHost()

    expect(await createTextCommand(() => commands).canCreateText('talks/Ants.mp3')).toBe(false)
    expect(await createTextCommand(() => undefined).canCreateText('talks/Ants.mp3')).toBe(false)
  })

  it('runs the command over the file', async () => {
    const commands = createMockCommandsHost()

    await createTextCommand(() => commands).createText('talks/Ants.mp3')

    expect(commands.getRuns()).toStrictEqual([
      { command: 'app:file.createText', target: TARGET, argument: undefined },
    ])
  })
})
