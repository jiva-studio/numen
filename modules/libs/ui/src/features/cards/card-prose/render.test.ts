/**
 * What a card comes to on the way to the screen.
 *
 * The card keeps the breaks a person typed, so a line they wrote with no tag
 * around it reads as a line. A break between two tags is the whitespace of
 * HTML and is no line of the card.
 */
import { describe, expect, it } from 'vitest'
import { rendered } from './render'

describe('rendered', () => {
  it('keeps the break between two lines a person wrote', () => {
    expect(rendered('about 45"\nabout 20 years')).toBe('about 45"\nabout 20 years')
  })

  it('keeps the break between two lines carrying a tag apiece', () => {
    expect(rendered('<b>Height:</b> 45"\n<b>Weight:</b> 130 kg')).toBe(
      '<b>Height:</b> 45"\n<b>Weight:</b> 130 kg',
    )
  })

  it('takes out the break standing between two tags', () => {
    expect(rendered('<p>one</p>\n<p>two</p>')).toBe('<p>one</p><p>two</p>')
  })

  it('takes out the space a list was laid out with', () => {
    expect(rendered('<ul>\n  <li>one</li>\n  <li>two</li>\n</ul>')).toBe(
      '<ul><li>one</li><li>two</li></ul>',
    )
  })

  it('takes out the breaks a face opens and closes with', () => {
    expect(rendered('\n<p>one</p>\n')).toBe('<p>one</p>')
  })

  it('keeps the break between a tag and a line of prose', () => {
    expect(rendered('<p>one</p>\nand a line')).toBe('<p>one</p>\nand a line')
  })

  it('keeps every character of preformatted text', () => {
    expect(rendered('<pre>  one\n\n  two\n</pre>')).toBe('<pre>  one\n\n  two\n</pre>')
  })

  it('draws nothing a card may not be drawn with', () => {
    expect(rendered('<p>a</p>\n<script>alert(1)</script>')).toBe('<p>a</p>')
  })
})
