import { describe, expect, it } from 'vitest'
import { mount } from '@vue/test-utils'
import { h } from 'vue'
import { WORD, render } from './render'

/** What the marks come out as, drawn. */
const mountProse = (text: string) => mount({ render: () => h('div', render(text)) })

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

  it('draws a wikilink as a link to the note it names', () => {
    expect(mountProse('Under [[Entropy]] it sits.').find('a').attributes('href')).toBe(
      'name://Entropy',
    )
  })

  it('holds a wikilink to what every other link is held to', () => {
    for (const address of [
      'javascript://%0Aalert(1)',
      'JavaScript://%0Aalert(1)',
      'vbscript://alert(1)',
      'file://etc/passwd',
      'data://text/html,<script>alert(1)</script>',
    ]) {
      const prose = mountProse(`[[${address}|Click]]`)
      expect(prose.find('a').exists()).toBe(false)
      expect(prose.text()).toContain('Click')
    }
  })

  it('reads a colon that begins no scheme as part of a name', () => {
    expect(mountProse('[[javascript:alert(1)|Click]]').find('a').attributes('href')).toBe(
      'name://javascript:alert(1)',
    )
  })

  it('has nothing to draw for nothing', () => {
    expect(render('')).toEqual([])
  })
})
