import { describe, expect, it } from 'vitest'
import { mount } from '@vue/test-utils'
import { h, type VNode } from 'vue'
import { indexLinkPieces, type LinkPieces } from '@/shared/lib/text/marks'
import { WORD, render } from './render'

/** What the marks come out as, drawn. */
const mountProse = (text: string) => mount({ render: () => h('div', render(text)) })

/** The same, with the pieces the caller draws as links. */
const mountWith = (text: string, pieces: LinkPieces) =>
  mount({ render: () => h('div', render(text, undefined, pieces)) })

describe('what is drawn at the top', () => {
  it('is nothing for no text', () => {
    expect(render('')).toEqual([])
  })

  it('is an element with nothing in it for a heading with no words', () => {
    const [heading] = render('#')
    expect(heading?.type).toBe('h1')
    expect(heading?.children).toEqual([])
  })
})

describe('prose', () => {
  it('makes a paragraph of a line', () => {
    expect(mountProse('Two notes are about entropy.').findAll('p')).toHaveLength(1)
  })

  it('makes a word a node of its own, so one that arrives can be shown arriving', () => {
    const words = mountProse('two notes here').findAll(`.${WORD}`)
    expect(words.map((word) => word.text())).toEqual(['two', 'notes', 'here'])
  })

  it('keeps the space between words', () => {
    expect(mountProse('two notes').find('p').text()).toBe('two notes')
  })

  it('reads the marks', () => {
    const prose = mountProse('**bold**, *leaning*, and `code`.')
    expect(prose.find('strong').text()).toBe('bold')
    expect(prose.find('em').text()).toBe('leaning')
    expect(prose.find('code').text()).toBe('code')
  })

  it('makes a list of a list', () => {
    expect(mountProse('- one\n- two\n- three').findAll('li')).toHaveLength(3)
  })

  it('keeps a fenced block whole', () => {
    const prose = mountProse('```\n# Title\n\npart of: [[Harmonic oscillator]]\n```')
    expect(prose.find('pre code').text()).toContain('part of: [[Harmonic oscillator]]')
  })

  it('draws a heading as a heading', () => {
    expect(mountProse('## What is here').find('h2').text()).toBe('What is here')
  })

  it('follows a link', () => {
    expect(mountProse('[a note](https://example.com)').find('a').attributes('href')).toBe(
      'https://example.com',
    )
  })

  it('shows prose that stops mid-sentence', () => {
    expect(mountProse('The style is clear: a `# Title` line, one evocative sen').text()).toContain(
      'evocative sen',
    )
  })

  it('shows a mark that has only been half written', () => {
    expect(mountProse('the answer is **bol').text()).toContain('bol')
  })

  it('leaves what somebody else wrote as words', () => {
    const prose = mountProse('A note can say <script>alert(1)</script> and it is prose.')
    expect(prose.find('script').exists()).toBe(false)
    expect(prose.text()).toContain('alert(1)')
  })

  it('draws a piece the caller names as a link', () => {
    const pieces = indexLinkPieces([{ raw: '<<Entropy>>', href: 'x://Entropy', text: 'Entropy' }])
    const link = mountWith('Under <<Entropy>> it sits.', pieces).find('a')
    expect(link.attributes('href')).toBe('x://Entropy')
    expect(link.text()).toBe('Entropy')
  })

  it('finds a piece that begins inside a word', () => {
    const pieces = indexLinkPieces([{ raw: 'ropy', href: 'x://r', text: 'ROPY' }])
    expect(mountWith('Entropy', pieces).text()).toBe('EntROPY')
  })

  it('leaves a piece inside code as code', () => {
    const pieces = indexLinkPieces([{ raw: '[[A]]', href: 'x://A', text: 'A' }])
    const prose = mountWith('`[[A]]` and [[A]]', pieces)
    expect(prose.find('code').text()).toBe('[[A]]')
    expect(prose.findAll('a')).toHaveLength(1)
  })

  it('draws a piece that is not named as the text it is', () => {
    expect(mountProse('Under [[Entropy]] it sits.').find('a').exists()).toBe(false)
  })

  it('holds a piece to what every other link is retained to', () => {
    for (const href of [
      'javascript://%0Aalert(1)',
      'JavaScript://%0Aalert(1)',
      'vbscript://alert(1)',
      'file://etc/passwd',
      'data://text/html,<script>alert(1)</script>',
    ]) {
      const pieces = indexLinkPieces([{ raw: '[[x]]', href, text: 'Click' }])
      const prose = mountWith('[[x]]', pieces)
      expect(prose.find('a').exists()).toBe(false)
      expect(prose.text()).toContain('[[x]]')
    }
  })

  it('has nothing to draw for nothing', () => {
    expect(render('')).toEqual([])
  })

  it('draws the paragraph of a tight list item as the item itself', () => {
    const prose = mountProse('- one\n- two')
    expect(prose.find('li').text()).toBe('one')
    expect(prose.find('li p').exists()).toBe(false)
  })

  it('draws the blocks of a text side by side', () => {
    const blocks = render('# Title\n\nfirst\n\nsecond')
    expect(blocks.map((block) => block.type)).toEqual(['h1', 'p', 'p'])
  })

  it('writes no attribute on an element the marks gave none', () => {
    expect(mountProse('plain words').find('p').attributes()).toEqual({})
  })

  it('keys each word and each inline code by where it falls in the answer', () => {
    const [paragraph] = render('one `two` three')
    const nodes = (paragraph!.children as (VNode | string)[]).filter(
      (child): child is VNode => typeof child !== 'string',
    )
    expect(nodes.map((node) => node.key)).toEqual([0, 1, 2])
  })

  it('makes no word of the space around a mark', () => {
    const words = mountProse('**bold** after').findAll(`.${WORD}`)
    expect(words.map((word) => word.text())).toEqual(['bold', 'after'])
  })

  it('keeps a run of spaces as one piece of space between two words', () => {
    const [paragraph] = render('one   two')
    const pieces = paragraph!.children as (VNode | string)[]
    expect(pieces).toHaveLength(3)
    expect(pieces[1]).toBe('   ')
  })

  it('marks a link to an address that reaches nothing', () => {
    const text = '[gone](https://example.com/gone) and [here](https://example.com/here)'
    const dead = new Set(['https://example.com/gone'])
    const prose = mount({ render: () => h('div', render(text, dead)) })
    const [removed, here] = prose.findAll('a')
    expect(removed!.attributes('data-reaches')).toBe('nothing')
    expect(here!.attributes('data-reaches')).toBeUndefined()
  })

  it('draws a rule', () => {
    expect(mountProse('above\n\n---\n\nbelow').find('hr').exists()).toBe(true)
  })

  it('keeps an indented block whole', () => {
    expect(mountProse('    let a = 1\n    let b = 2').find('pre code').text()).toBe(
      'let a = 1\nlet b = 2',
    )
  })

  it('joins the lines of a paragraph with a space', () => {
    expect(mountProse('first line\nsecond line').find('p').text()).toBe('first line second line')
  })

  it('breaks the line where two spaces end it', () => {
    expect(mountProse('first  \nsecond').find('p br').exists()).toBe(true)
  })

  it('draws an image from its address', () => {
    const image = mountProse('![a chart](https://example.com/chart.png)').find('img')
    expect(image.attributes('src')).toBe('https://example.com/chart.png')
  })

  it('keeps the alt text of an image', () => {
    expect(mountProse('![a cat](cat.png)').find('img').attributes('alt')).toBe('a cat')
  })

  it('writes the alt text of an image without its marks', () => {
    expect(mountProse('![a *big* cat](x.png)').find('img').attributes('alt')).toBe('a big cat')
  })
})
