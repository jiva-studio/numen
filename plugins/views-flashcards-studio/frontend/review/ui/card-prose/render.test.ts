/**
 * What a card renders to on the way to the screen.
 *
 * The card keeps the line breaks the user typed, so a line written with no tag
 * around it renders as a line. A break between two tags is HTML whitespace and
 * is not a line of the card.
 */
import { describe, expect, it } from 'vitest'
import { renderCardHtml } from './render'

describe('renderCardHtml', () => {
  it('keeps the break between two lines a person wrote', () => {
    expect(renderCardHtml('about 45"\nabout 20 years')).toBe('about 45"\nabout 20 years')
  })

  it('keeps the break between two lines carrying a tag apiece', () => {
    expect(renderCardHtml('<b>Height:</b> 45"\n<b>Weight:</b> 130 kg')).toBe(
      '<b>Height:</b> 45"\n<b>Weight:</b> 130 kg',
    )
  })

  it('takes out the break standing between two tags', () => {
    expect(renderCardHtml('<p>one</p>\n<p>two</p>')).toBe('<p>one</p><p>two</p>')
  })

  it('takes out the space a list was laid out with', () => {
    expect(renderCardHtml('<ul>\n  <li>one</li>\n  <li>two</li>\n</ul>')).toBe(
      '<ul><li>one</li><li>two</li></ul>',
    )
  })

  it('takes out the breaks a template opens and closes with', () => {
    expect(renderCardHtml('\n<p>one</p>\n')).toBe('<p>one</p>')
  })

  it('keeps the break between a tag and a line of prose', () => {
    expect(renderCardHtml('<p>one</p>\nand a line')).toBe('<p>one</p>\nand a line')
  })

  it('keeps every character of preformatted text', () => {
    expect(renderCardHtml('<pre>  one\n\n  two\n</pre>')).toBe('<pre>  one\n\n  two\n</pre>')
  })

  it('keeps the break between two inline tags', () => {
    expect(renderCardHtml('<b>one</b>\n<b>two</b>')).toBe('<b>one</b>\n<b>two</b>')
  })

  it('keeps a tag with no text between two blocks', () => {
    expect(renderCardHtml('<p>one</p><hr><p>two</p>')).toBe('<p>one</p><hr><p>two</p>')
    expect(renderCardHtml('<p>one</p><p></p><p>two</p>')).toBe('<p>one</p><p></p><p>two</p>')
  })

  it('keeps the break between two blocks inside preformatted text', () => {
    expect(renderCardHtml('<pre><div>one</div>\n<div>two</div></pre>')).toBe(
      '<pre><div>one</div>\n<div>two</div></pre>',
    )
  })

  it('draws nothing a card may not be drawn with', () => {
    expect(renderCardHtml('<p>a</p>\n<script>alert(1)</script>')).toBe('<p>a</p>')
  })
})
