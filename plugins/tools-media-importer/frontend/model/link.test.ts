import { describe, expect, it } from 'vitest'
import { flushPromises } from '@vue/test-utils'
import { useLink } from './link'
import type { LinkFacts, Links } from '../types'

/** A link whose facts and text a test sets, and whose work it starts and ends. */
function createLinks(facts: LinkFacts) {
  const link = {
    facts,
    text: '',
    calls: 0,
    fetches: [] as boolean[],
    started: [] as string[],
    onChange: (_: boolean) => {},
    isStopped: false,
  }
  const links: Links = {
    getLink: async () => {
      link.calls++
      return { ok: true, value: link.facts }
    },
    readArticle: async () => link.text,
    getSource: async (path) => ({
      mediaUrl: `http://127.0.0.1:1/t/v/${path}`,
      mediaType: 'video/mp4',
    }),
    getPageSource: async (page) => `http://127.0.0.1:1/t/plugin/importer/${page}`,
    fetchText: async (_, isUserStarted) => {
      link.fetches.push(isUserStarted)
      return { ok: true, value: null }
    },
    copyVideo: async (path) => {
      link.started.push(`copy ${path}`)
      return { ok: true, value: null }
    },
    deleteCopy: async (path) => {
      link.started.push(`delete ${path}`)
      link.facts = { ...link.facts, copySize: 0 }
      return { ok: true, value: null }
    },
    watchWork: (_, onChange) => {
      link.onChange = onChange
      return () => {
        link.isStopped = true
      }
    },
  }
  return { link, links }
}

const ARTICLE = 'http://tidal-mills.numen.test/history/rance'
const FACTS: LinkFacts = {
  url: ARTICLE,
  isCopyable: false,
  copySize: 0,
  isTextDeleted: false,
  embedPage: '',
}

describe('useLink', () => {
  it('reads the address and the text, and reads them again once the work on the link ends', async () => {
    const { link, links } = createLinks(FACTS)
    const tab = useLink(links, 'links/rance.md')
    await flushPromises()
    expect(tab.url.value).toBe(ARTICLE)
    expect(tab.paragraphs.value).toEqual([])
    expect(tab.status.isLoaded.value).toBe(true)

    link.onChange(true)
    expect(tab.status.isWorking.value).toBe(true)
    link.text = 'A tidal mill stored seawater.\n\nThe pond filled twice a day.'
    link.onChange(false)
    await flushPromises()
    expect(tab.paragraphs.value).toEqual([
      'A tidal mill stored seawater.',
      'The pond filled twice a day.',
    ])
    expect(link.calls).toBe(2)

    tab.close()
    expect(link.isStopped).toBe(true)
  })

  it('requests the text of a link that has none once, not as a user-started fetch', async () => {
    const { link, links } = createLinks(FACTS)
    useLink(links, 'links/rance.md')
    await flushPromises()
    link.onChange(false)
    await flushPromises()

    expect(link.fetches).toEqual([false])
  })

  it('leaves a text the person deleted deleted, and says so', async () => {
    const { link, links } = createLinks({ ...FACTS, isTextDeleted: true })
    const tab = useLink(links, 'links/rance.md')
    await flushPromises()

    expect(link.fetches).toEqual([])
    expect(tab.status.isTextDeleted.value).toBe(true)
  })

  it('fetches the text again when the person asks', async () => {
    const { link, links } = createLinks({ ...FACTS, isTextDeleted: true })
    const tab = useLink(links, 'links/rance.md')
    await flushPromises()

    await tab.actions.fetchText()
    expect(link.fetches).toEqual([true])
  })

  it('plays the copy once one is kept, and stops showing it once it is deleted', async () => {
    const { link, links } = createLinks({
      url: 'https://videos.example/clip',
      isCopyable: true,
      copySize: 0,
      isTextDeleted: false,
    })
    link.text = 'The tide comes in.'
    const tab = useLink(links, 'links/clip.md')
    await flushPromises()
    expect(tab.source.value).toBeNull()
    expect(tab.status.isCopyable.value).toBe(true)

    await tab.actions.copyVideo()
    link.facts = { ...link.facts, copySize: 6000 }
    link.onChange(false)
    await flushPromises()
    expect(tab.source.value).toEqual({
      mediaUrl: 'http://127.0.0.1:1/t/v/links/clip.md',
      mediaType: 'video/mp4',
    })

    await tab.actions.deleteCopy()
    await flushPromises()
    expect(tab.source.value).toBeNull()
    expect(link.started).toEqual(['copy links/clip.md', 'delete links/clip.md'])
  })

  it('says why a file could not be read as a link', async () => {
    const links: Links = {
      ...createLinks(FACTS).links,
      getLink: async () => ({ ok: false, error: 'This note has no web address' }),
    }
    const tab = useLink(links, 'notes/plain.md')
    await flushPromises()
    expect(tab.error.value).toBe('This note has no web address')
  })
})

describe('useLink of a video', () => {
  it('frames the player page while no copy is kept, and plays the copy once one is', async () => {
    const { link, links } = createLinks({
      ...FACTS,
      isCopyable: true,
      embedPage: 'embed?address=x',
    })
    const tab = useLink(links, 'links/clip.md')
    await flushPromises()
    expect(tab.embedSource.value).toBe('http://127.0.0.1:1/t/plugin/importer/embed?address=x')
    expect(tab.source.value).toBeNull()

    link.facts = { ...link.facts, copySize: 6000 }
    link.onChange(false)
    await flushPromises()
    expect(tab.embedSource.value).toBe('')
    expect(tab.source.value?.mediaType).toBe('video/mp4')
  })
})
