/**
 * The corner says what the window lost touch with, whichever part lost it.
 */
import { ref } from 'vue'
import { describe, expect, it } from 'vitest'
import { useWindowNotices } from './useWindowNotices'

const notices = (windowLost: string, settingsLost: string) =>
  useWindowNotices({
    log: { messages: ref([]) } as never,
    window: {
      unwatched: ref(''),
      error: ref(''),
      lost: ref(windowLost),
      isIndexing: ref(false),
      hasNote: ref(true),
    } as never,
    settings: { appearance: { lost: ref(settingsLost) } } as never,
    vaults: { coverage: () => ({ chunks: 0, embedded: 0, isEmbedding: false }) } as never,
  }).value.filter((one) => one.id === 'lost')

describe('the lost notice', () => {
  it('names what the window lost', () => {
    expect(notices('the vault', '').map((one) => one.text)).toEqual(['the vault'])
  })

  it('names what the settings lost when the window lost nothing', () => {
    expect(notices('', 'the theme').map((one) => one.text)).toEqual(['the theme'])
  })

  it('names the window loss first when both lost something', () => {
    expect(notices('the vault', 'the theme').map((one) => one.text)).toEqual(['the vault'])
  })

  it('is absent when nothing was lost', () => {
    expect(notices('', '')).toEqual([])
  })
})
