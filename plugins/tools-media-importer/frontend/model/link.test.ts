import { describe, expect, it } from 'vitest'
import { flushPromises } from '@vue/test-utils'
import { useLink } from './link'
import type { LinkFacts, Links } from '../types'

/** A link whose facts and text a test sets, and whose work it starts and ends. */
function createLinks(facts: LinkFacts) {
  const link = { facts, text: '', calls: 0, onChange: (_: boolean) => {}, isStopped: false }
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
    createLink: async () => ({ ok: false, error: 'not here' }),
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

describe('useLink', () => {
  it('reads the address and the text, and reads them again once the work on the link ends', async () => {
    const { link, links } = createLinks({ url: ARTICLE, isCopyable: false, copySize: 0 })
    const tab = useLink(links, 'links/rance.url')
    await flushPromises()
    expect(tab.url.value).toBe(ARTICLE)
    expect(tab.paragraphs.value).toEqual([])
    expect(tab.isLoaded.value).toBe(true)

    link.onChange(true)
    expect(tab.isWorking.value).toBe(true)
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

  it('plays the copy once one is kept', async () => {
    const { link, links } = createLinks({
      url: 'https://videos.example/clip',
      isCopyable: true,
      copySize: 0,
    })
    const tab = useLink(links, 'links/clip.url')
    await flushPromises()
    expect(tab.source.value).toBeNull()
    expect(tab.isCopyable.value).toBe(true)

    link.facts = { ...link.facts, copySize: 6000 }
    link.onChange(false)
    await flushPromises()
    expect(tab.source.value).toEqual({
      mediaUrl: 'http://127.0.0.1:1/t/v/links/clip.url',
      mediaType: 'video/mp4',
    })
  })

  it('says why a file could not be read as a link', async () => {
    const links: Links = {
      ...createLinks({ url: '', isCopyable: false, copySize: 0 }).links,
      getLink: async () => ({ ok: false, error: 'This file holds no web address' }),
    }
    const tab = useLink(links, 'notes/plain.md')
    await flushPromises()
    expect(tab.error.value).toBe('This file holds no web address')
  })
})
