/**
 * The reader over the markup a book really arrives as.
 *
 * The corpus is one spine document as the application writes it, and a test
 * beside the writer holds it to what that writer emits. So the attribute a run
 * carries, the elements a book is drawn from, the escaping and the offsets are
 * agreed on here rather than in a fixture each side writes for itself.
 */
import { mount } from '@vue/test-utils'
import { describe, expect, it } from 'vitest'
import spine from '../../../../protocol/testdata/spine.html?raw'
import Book from './Book.vue'
import { bytesIn } from './spread'

/** The document the corpus is taken from, as the archive names it. */
const MIDDLE = 'OEBPS/middle.xhtml'

/** One run of the corpus: where it says it begins, and what it says. */
interface Run {
  readonly at: number
  readonly said: string
}

/** The runs the corpus carries, in the order it writes them. */
const runs = (): readonly Run[] => {
  const held = new DOMParser().parseFromString(spine, 'text/html')
  return [...held.body.querySelectorAll('[data-offset]')].map((one) => ({
    at: Number(one.getAttribute('data-offset')),
    said: one.textContent ?? '',
  }))
}

/** Where the document stands in the book's text, read off its own runs. */
const span = () => {
  const all = runs()
  const first = all[0]!
  const last = all[all.length - 1]!
  return { begins: first.at, ends: last.at + bytesIn(last.said) }
}

/** A reader drawing the corpus, in a room jsdom has laid nothing out in. */
const reading = async () => {
  const held = mount(Book, {
    props: { markup: spine, path: MIDDLE, span: span(), book: span(), at: span().begins },
    attachTo: document.body,
  })
  ;(held.find('.book__area').element as HTMLElement).scrollTo = () => {}
  await held.vm.$nextTick()
  return held
}

/** One link of the document pressed, and the press as the page left it. */
const press = async (held: Awaited<ReturnType<typeof reading>>, says: string) => {
  const link = held.findAll('a').find((one) => one.text() === says)!
  link.element.dispatchEvent(new MouseEvent('click', { bubbles: true, cancelable: true }))
  await held.vm.$nextTick()
}

/** The offsets the reader has asked to be sent to, in the order it asked. */
const asked = (held: Awaited<ReturnType<typeof reading>>) =>
  (held.emitted('go') ?? []).map((one) => (one as [number])[0])

describe('the markup a spine document arrives as', () => {
  it('says where every run of it begins', () => {
    expect(runs().length).toBeGreaterThan(0)
    for (const run of runs()) expect(Number.isFinite(run.at)).toBe(true)
  })

  it('runs on from one offset to the next, by the bytes each run says', () => {
    // Every byte of the document stands in one run, so a run reaches exactly to
    // where the next begins. An escaping the two sides read differently parts
    // these numbers on the first paragraph that carries an ampersand.
    const all = runs()
    for (let i = 1; i < all.length; i++) {
      const before = all[i - 1]!
      expect(all[i]!.at).toBe(before.at + bytesIn(before.said))
    }
  })

  it('begins where the document does, and not at nothing', () => {
    expect(span().begins).toBeGreaterThan(0)
  })
})

describe('a reader handed that markup', () => {
  it('reads the offsets off the runs, and asks for the one a place inside names', async () => {
    const held = await reading()
    const note = held.find('#note').find('[data-offset]').attributes('data-offset')

    await press(held, 'the note')

    expect(asked(held)).toEqual([Number(note)])
    expect(asked(held)).not.toEqual([span().begins])
  })

  it('asks for the document a link names, as the archive names it', async () => {
    const held = await reading()

    await press(held, 'the opening')

    expect(held.emitted('follow')).toEqual([['OEBPS/opening.xhtml']])
  })

  it('asks for nothing of the book where the link leads out of it', async () => {
    const held = await reading()

    await press(held, 'somewhere else entirely')

    expect(asked(held)).toHaveLength(0)
    expect(held.emitted('follow')).toBeUndefined()
  })
})
