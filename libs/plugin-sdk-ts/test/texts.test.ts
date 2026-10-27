import { describe, expect, it } from 'vitest'
import { joinTextSegments, type TextVersion } from '../src/texts'
import { createMockVaultTexts } from '../src/testing/mockTexts'

function createVersion(id: string): TextVersion {
  const [pluginId = '', key = ''] = id.split('/')
  return {
    id,
    path: 'talk.mp3',
    pluginId,
    key,
    sourceVersionId: '',
    kind: 'transcript',
    file: `${key}.jsonl`,
    createTime: new Date(0),
  }
}

describe('the text versions of a vault', () => {
  it('joins the segments of a text', () => {
    expect(joinTextSegments([{ text: 'one ', heading: 'Intro' }, { text: 'two' }])).toBe('one two')
  })

  it('reads the live text and records an edit as the new live version', async () => {
    const texts = createMockVaultTexts('org.example.player')
    texts.setLive(createVersion('org.example.transcription/k'), [
      { text: 'Good mornin.\n', selector: { t: [0, 1.5] } },
    ])

    const edit = await texts.createEdit('talk.mp3', 'org.example.transcription/k', [
      { text: 'Good morning.\n', selector: { t: [0, 1.5] } },
    ])

    expect(edit.ok && edit.value.sourceVersionId).toBe('org.example.transcription/k')
    expect(edit.ok && edit.value.pluginId).toBe('org.example.player')
    const live = await texts.readLiveText('talk.mp3')
    expect(live?.segments[0]?.text).toBe('Good morning.\n')
  })

  it('refuses an edit of a version that is not live', async () => {
    const texts = createMockVaultTexts()

    expect(await texts.createEdit('talk.mp3', 'x/y', [])).toStrictEqual({
      ok: false,
      error: 'missing',
    })
    expect(await texts.readLiveText('talk.mp3')).toBeNull()
  })
})
