import { describe, expect, it } from 'vitest'
import {
  joinTextParts,
  readWholeText,
  type TextPart,
  type TextVersion,
  type VaultTexts,
} from '../src/texts'
import { createMockVaultTexts } from '../src/testing/mockTexts'

const place = { page: 0, rects: [], startMs: 0, endMs: 0 }

function createVersion(id: string): TextVersion {
  const [pluginId, key] = id.split('/')
  return {
    id,
    path: 'a.pdf',
    pluginId,
    key,
    inputVersionId: '',
    format: 'positioned',
    inputHash: 'f',
    hash: 'h',
    createTime: new Date(0),
  }
}

describe('the text versions of a vault', () => {
  it('reads every page of a version and joins them', async () => {
    const texts = createMockVaultTexts()
    const parts: TextPart[] = [
      { text: 'one ', place, section: 'Intro' },
      { text: 'two', place: { ...place, page: 1 }, section: '' },
    ]
    texts.putVersion(createVersion('org.example.ocr/k'), parts)

    const read = await readWholeText(texts, 'a.pdf', 'org.example.ocr/k')

    expect(read.ok && joinTextParts(read.value)).toBe('one two')
  })

  it('answers missing for a version that is not there', async () => {
    const read = await readWholeText(createMockVaultTexts(), 'a.pdf', 'x/y')

    expect(read).toStrictEqual({ ok: false, error: 'missing' })
  })

  it('stops on a provider that answers the same page twice', async () => {
    const looping: VaultTexts = {
      listVersions: async () => ({ versions: [], liveVersionId: '' }),
      readText: async () => ({ ok: true, value: { parts: [], nextCursor: 'again' } }),
      onVersionsChange: () => ({ dispose: () => {} }),
    }

    expect(await readWholeText(looping, 'a.pdf', 'x/y')).toStrictEqual({
      ok: false,
      error: 'repeatedCursor',
    })
  })

  it('tells a handler of every file whose versions change, until it is disposed', async () => {
    const texts = createMockVaultTexts()
    const changed: string[] = []
    const subscription = texts.onVersionsChange((path) => changed.push(path))

    texts.putVersion(createVersion('org.example.ocr/k'), [])
    texts.setLive('a.pdf', 'org.example.ocr/k')
    subscription.dispose()
    texts.setLive('a.pdf', '')

    expect(changed).toStrictEqual(['a.pdf', 'a.pdf'])
    expect((await texts.listVersions('a.pdf')).liveVersionId).toBe('')
  })
})
