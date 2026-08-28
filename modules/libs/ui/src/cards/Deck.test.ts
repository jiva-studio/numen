/**
 * What the deck editor draws from the cards it was handed, and what it emits.
 *
 * The negatives are here: no tile says what stencil cuts it, the head holds
 * nothing but the two things a tile is reached for, the plus offers nothing
 * until it is pressed, a card let go where it stands moves nothing, and a card
 * whose stencil was not handed in keeps every value it has.
 */
import { mount } from '@vue/test-utils'
import { afterEach, describe, expect, it } from 'vitest'
import { nextTick } from 'vue'
import Deck from './Deck.vue'
import type { Drawn } from './deck'
import type { Cut } from './stencil'

const CUTS: readonly Cut[] = [
  { name: 'Animal', fields: ['Name', 'Height', 'Life span'] },
  { name: 'Word', fields: ['Word', 'Meaning'] },
]

/* A card is named by its heading, which reaches here as `name`. The field that
   names it stands in no value. */
const CARDS: readonly Drawn[] = [
  {
    id: 'llama',
    name: 'Llama',
    stencil: 'Animal',
    filled: [{ field: 'Height', text: 'about 45"' }],
  },
  { id: 'yak', name: 'Yak', stencil: 'Animal', filled: [] },
]

const mountDeck = (props: Record<string, unknown> = {}) =>
  mount(Deck, { attachTo: document.body, props: { cards: CARDS, cuts: CUTS, ...props } })

type Grid = ReturnType<typeof mountDeck>

const tileFor = (held: Grid, id: string) => held.get(`[data-card="${id}"]`)

const drawnCards = (held: Grid) =>
  held.findAll('[data-card]').map((tile) => tile.attributes('data-card'))

/** The fields a tile puts in boxes, in the order it draws them. */
const fieldsOf = (held: Grid, id: string): readonly (string | undefined)[] =>
  tileFor(held, id)
    .findAll('textarea')
    .map((box) => box.attributes('data-value'))

const boxFor = (held: Grid, id: string, field: string) =>
  tileFor(held, id).get<HTMLTextAreaElement>(`[data-value="${field}"]`)

/** Every box standing under the name of the field a card is named by. */
const namedBoxes = (held: Grid): readonly HTMLTextAreaElement[] =>
  held.findAll<HTMLTextAreaElement>('[data-value="Name"]').map((box) => box.element)

/** A card picked up by its grip and let go over another, or over the grid. */
const dragTo = async (held: Grid, id: string, onto: string | null): Promise<void> => {
  await tileFor(held, id).get('[data-grip]').trigger('dragstart')
  const over = onto === null ? held.get('.deck') : tileFor(held, onto)
  await over.trigger('dragover')
  await over.trigger('drop')
}

/** A box typed into, as a person types into it. */
const typed = (box: HTMLTextAreaElement | undefined, text: string): void => {
  if (!box) throw new Error('no box to type in')
  box.value = text
  box.dispatchEvent(new Event('input'))
}

/** A key pressed on something, as the event it was pressed with. */
const pressing = (on: Element, key: string): KeyboardEvent => {
  const press = new KeyboardEvent('keydown', { key, bubbles: true, cancelable: true })
  on.dispatchEvent(press)
  return press
}

/** A card picked up and the carry ended without it being let go anywhere. */
const dragOff = async (held: Grid, id: string, over: string | null): Promise<void> => {
  const grip = tileFor(held, id).get('[data-grip]')
  await grip.trigger('dragstart')
  if (over !== null) await tileFor(held, over).trigger('dragover')
  await grip.trigger('dragend')
}

afterEach(() => {
  document.body.innerHTML = ''
})

describe('Deck', () => {
  it('draws a tile per card, in the order they were handed in', () => {
    expect(drawnCards(mountDeck())).toEqual(['llama', 'yak'])
  })

  it('stands the plus last, and counts it in what a tile is announced as', () => {
    const held = mountDeck()
    expect(tileFor(held, 'yak').attributes('aria-setsize')).toBe('3')
    expect(held.get('[data-plus]').attributes('aria-posinset')).toBe('3')
  })

  it('draws the plus alone for a deck holding nothing', () => {
    const held = mountDeck({ cards: [] })
    expect(held.findAll('[data-card]')).toHaveLength(0)
    expect(held.get('[data-plus]').attributes('aria-posinset')).toBe('1')
  })

  it('draws every value in a box, the field naming the card first', () => {
    expect(fieldsOf(mountDeck(), 'llama')).toEqual(['Name', 'Height', 'Life span'])
  })

  it('draws the field naming the card as it draws every other', () => {
    const held = mountDeck()
    const values = tileFor(held, 'llama').findAll('.card__value')
    expect(values.map((each) => each.get('label').text())).toEqual([
      'Name',
      'Height',
      'Life span',
    ])
    expect(values[0]?.attributes('data-names')).toBe('true')
    expect(values[1]?.attributes('data-names')).toBeUndefined()
  })

  it('names each box by a label the box itself is named by', () => {
    const held = mountDeck()
    for (const value of tileFor(held, 'llama').findAll('.card__value')) {
      const label = value.get('label')
      const box = value.get('textarea')
      expect(label.attributes('for')).toBe(box.attributes('id'))
      expect(label.attributes('for')).toBeTruthy()
    }
  })

  it('names a value on the rule that divides it from the one above, and nowhere else', () => {
    const value = tileFor(mountDeck(), 'llama').findAll('.card__value')[0]

    // One name, and it stands on the rule. Nothing is drawn around the box
    // under it.
    expect(value?.findAll('label')).toHaveLength(1)
    expect(value?.get('.rule').get('label').text()).toBe('Name')
    expect(value?.find('fieldset').exists()).toBe(false)
  })

  it('stands the name the card was handed in the field that names it', () => {
    expect(boxFor(mountDeck(), 'llama', 'Name').element.value).toBe('Llama')
  })

  it('emits the naming field written, which is what names the card', async () => {
    const held = mountDeck()
    await boxFor(held, 'llama', 'Name').setValue('Alpaca')
    expect(held.emitted('write')).toEqual([['llama', 'Name', 0, true, 'Alpaca']])
  })

  it('holds one line in the box the card is named by', async () => {
    const held = mountDeck()
    const box = boxFor(held, 'llama', 'Name')

    // The name is a heading, so a break in it is refused where it is typed.
    expect(pressing(box.element, 'Enter').defaultPrevented).toBe(true)

    // And a break arriving another way is closed up before it is emitted.
    await box.setValue('Llama\n## Alpaca')
    expect(held.emitted('write')).toEqual([['llama', 'Name', 0, true, 'Llama ## Alpaca']])
  })

  it('holds what was typed, breaks and all, in every other box', async () => {
    const held = mountDeck()
    const box = boxFor(held, 'llama', 'Height')
    expect(pressing(box.element, 'Enter').defaultPrevented).toBe(false)

    await box.setValue('about 45"\nat the shoulder')
    expect(held.emitted('write')).toEqual([
      ['llama', 'Height', 1, false, 'about 45"\nat the shoulder'],
    ])
  })

  it('emits one other value of one card as it now reads', async () => {
    const held = mountDeck()
    await boxFor(held, 'llama', 'Life span').setValue('about 20 years')
    expect(held.emitted('write')).toEqual([['llama', 'Life span', 1, false, 'about 20 years']])
  })

  it('sets the ground behind each box to the box’s own text, which is what sizes it', () => {
    const cards: readonly Drawn[] = [
      { id: 'x', name: 'X', stencil: 'Animal', filled: [{ field: 'Height', text: 'a\nb\nc' }] },
    ]
    const held = mountDeck({ cards })
    const grown = tileFor(held, 'x').findAll('[data-grown]')
    expect(grown.map((each) => each.attributes('data-grown'))).toEqual(['X', 'a\nb\nc', ''])
  })

  it('says the name a card whose stencil was not handed in was given', () => {
    const orphan: readonly Drawn[] = [
      { id: 'gone', name: 'Gone', stencil: 'Missing', filled: [{ field: 'A', text: 'kept' }] },
    ]
    const held = mountDeck({ cards: orphan })
    expect(tileFor(held, 'gone').get('.card__said').text()).toBe('Gone')
  })

  it('says the name of no card whose stencil names a field to hold it', () => {
    expect(mountDeck().find('.card__said').exists()).toBe(false)
  })

  it('says on each tile what cut the card it draws', () => {
    // Two stencils, named nothing that any field is named, so what is looked
    // for here can only be the stencil's own name.
    const cuts: readonly Cut[] = [
      { name: 'Beast', fields: ['Name', 'Height'] },
      { name: 'Vocabulary', fields: ['Word', 'Meaning'] },
    ]
    const mixed: readonly Drawn[] = [
      { id: 'llama', name: 'Llama', stencil: 'Beast', filled: [] },
      { id: 'llano', name: 'llano', stencil: 'Vocabulary', filled: [] },
    ]
    const held = mountDeck({ cards: mixed, cuts })

    expect(held.findAll('[data-card]')).toHaveLength(2)
    expect(tileFor(held, 'llama').get('[data-cut-of]').text()).toBe('Beast')
    expect(tileFor(held, 'llano').get('[data-cut-of]').text()).toBe('Vocabulary')
  })

  it('holds nothing in a tile’s head but what cut it, the grip and the way to remove it', () => {
    const head = tileFor(mountDeck(), 'llama').get('.bar')
    expect(head.findAll('button')).toHaveLength(1)
    expect(head.attributes('data-grip')).toBeDefined()
    expect(head.attributes('draggable')).toBe('true')
    expect(head.text().trim()).toBe('Animal')
  })

  it('carries a tile by the strip itself, and not by a handle inside it', () => {
    const tile = tileFor(mountDeck(), 'llama')
    expect(tile.findAll('[data-grip]')).toHaveLength(1)
    expect(tile.get('[data-grip]').element.tagName).toBe('HEADER')
  })

  it('leaves the way to remove a card out of what carries it', () => {
    const away = tileFor(mountDeck(), 'llama').get('.bar button')
    expect(away.attributes('draggable')).toBe('false')
  })

  describe('a press landing on something the strip holds', () => {
    const stripOf = (held: Grid, id: string) => tileFor(held, id).get('[data-grip]')

    it('lets that thing have the press, so the strip is not carried by it', async () => {
      const held = mountDeck()
      await stripOf(held, 'llama').get('button').trigger('pointerdown')
      expect(stripOf(held, 'llama').attributes('draggable')).toBe('false')
    })

    it('takes the strip up again once the press is let go anywhere at all', async () => {
      const held = mountDeck()
      await stripOf(held, 'llama').get('button').trigger('pointerdown')

      // The pointer may be let go far outside the strip, and the strip is
      // carried again from there.
      document.body.dispatchEvent(new Event('pointerup', { bubbles: true }))
      await nextTick()

      expect(stripOf(held, 'llama').attributes('draggable')).toBe('true')
    })

    it('takes the strip up again where the press is called off', async () => {
      const held = mountDeck()
      await stripOf(held, 'llama').get('button').trigger('pointerdown')

      document.body.dispatchEvent(new Event('pointercancel', { bubbles: true }))
      await nextTick()

      expect(stripOf(held, 'llama').attributes('draggable')).toBe('true')
    })

    it('carries the strip from a press landing on the strip itself', async () => {
      const held = mountDeck()
      await stripOf(held, 'llama').trigger('pointerdown')
      expect(stripOf(held, 'llama').attributes('draggable')).toBe('true')
    })
  })

  it('draws no way to open or shut a tile: every value is open to typing', () => {
    const held = mountDeck()
    expect(tileFor(held, 'llama').find('[aria-expanded]').exists()).toBe(false)
  })

  it('says of nothing that it is open, the plus being gone once it has opened', async () => {
    const held = mountDeck()
    expect(held.find('[aria-expanded]').exists()).toBe(false)
    await held.get('[data-plus] button').trigger('click')
    expect(held.find('[aria-expanded]').exists()).toBe(false)
  })

  it('names the card a stencil names no field to hold to a reader as well', () => {
    const orphan: readonly Drawn[] = [
      { id: 'gone', name: 'Gone', stencil: 'Missing', filled: [{ field: 'A', text: 'kept' }] },
    ]
    const said = tileFor(mountDeck({ cards: orphan }), 'gone').get('.card__said')

    // A name is exposed by nothing standing on a paragraph, so the text takes
    // a role that carries one.
    expect(said.attributes('role')).toBe('group')
    expect(said.attributes('aria-label')).toBe('Name')
  })

  describe('carrying a tile by the keyboard', () => {
    const stripOf = (held: Grid, id: string) => tileFor(held, id).get('[data-grip]')

    it('names the strip a tile is carried by, and gives it a place in the order', () => {
      const strip = stripOf(mountDeck(), 'llama')
      expect(strip.attributes('aria-label')).toBe('Reorder: Llama')
      expect(strip.attributes('tabindex')).toBe('0')
      expect(strip.attributes('role')).toBe('group')
    })

    it('emits a tile carried one place down the order', () => {
      const held = mountDeck()
      const press = pressing(stripOf(held, 'llama').element, 'ArrowDown')
      expect(press.defaultPrevented).toBe(true)
      expect(held.emitted('move')).toEqual([['llama', null]])
    })

    it('emits a tile carried one place up the order', () => {
      const held = mountDeck()
      pressing(stripOf(held, 'yak').element, 'ArrowUp')
      expect(held.emitted('move')).toEqual([['yak', 'llama']])
    })

    it('moves nothing where there is no place that way', () => {
      const held = mountDeck()
      const press = pressing(stripOf(held, 'llama').element, 'ArrowUp')
      expect(press.defaultPrevented).toBe(false)
      expect(held.emitted('move')).toBeUndefined()
    })

    it('moves nothing on a key that is no way along the order', () => {
      const held = mountDeck()
      pressing(stripOf(held, 'llama').element, 'ArrowRight')
      expect(held.emitted('move')).toBeUndefined()
    })
  })

  it('emits the card asked to go', async () => {
    const held = mountDeck()
    await tileFor(held, 'yak').get('.bar button').trigger('click')
    expect(held.emitted('remove')).toEqual([['yak']])
  })

  it('offers no stencil until the plus is pressed', () => {
    expect(mountDeck().findAll('[data-cut]')).toHaveLength(0)
  })

  it('asks which stencil once the plus is pressed', async () => {
    const held = mountDeck()
    await held.get('[data-plus] button').trigger('click')
    expect(held.findAll('[data-cut]').map((each) => each.attributes('data-cut'))).toEqual([
      'Animal',
      'Word',
    ])
  })

  it('adds nothing while it is only asking', async () => {
    const held = mountDeck()
    await held.get('[data-plus] button').trigger('click')
    expect(held.emitted('add')).toBeUndefined()
  })

  it('asks for a card named by something nothing has taken', async () => {
    const held = mountDeck()
    await held.get('[data-plus] button').trigger('click')
    await held.get('[data-cut="Animal"]').trigger('click')
    expect(held.emitted('add')).toEqual([
      [
        'Card 1',
        'Animal',
        [
          { field: 'Height', text: '' },
          { field: 'Life span', text: '' },
        ],
      ],
    ])
  })

  it('asks for a card whose naming field stands in no value', async () => {
    const held = mountDeck()
    await held.get('[data-plus] button').trigger('click')
    await held.get('[data-cut="Animal"]').trigger('click')
    const filled = held.emitted('add')?.[0]?.[2] as readonly { field: string }[]
    expect(filled.map((each) => each.field)).not.toContain('Name')
  })

  it('stops asking once a stencil is chosen', async () => {
    const held = mountDeck()
    await held.get('[data-plus] button').trigger('click')
    await held.get('[data-cut="Word"]').trigger('click')
    expect(held.findAll('[data-cut]')).toHaveLength(0)
  })

  it('emits a card let go before another', async () => {
    const held = mountDeck()
    await dragTo(held, 'yak', 'llama')
    expect(held.emitted('move')).toEqual([['yak', 'llama']])
  })

  it('emits a card let go past the last of them', async () => {
    const held = mountDeck()
    await dragTo(held, 'llama', null)
    expect(held.emitted('move')).toEqual([['llama', null]])
  })

  it('moves nothing where a card is let go where it stands', async () => {
    const held = mountDeck()
    await dragTo(held, 'llama', 'llama')
    expect(held.emitted('move')).toBeUndefined()
  })

  it('moves nothing where a carry ends with the card let go nowhere', async () => {
    const held = mountDeck()
    await dragOff(held, 'yak', null)
    expect(held.emitted('move')).toBeUndefined()
    expect(tileFor(held, 'yak').attributes('data-carried')).toBeUndefined()
  })

  it('moves nothing where a carry over another card ends with it let go nowhere', async () => {
    const held = mountDeck()
    await dragOff(held, 'yak', 'llama')
    expect(held.emitted('move')).toBeUndefined()
  })

  it('draws no value of a card whose stencil was not handed in, and names it', () => {
    const orphan: readonly Drawn[] = [
      { id: 'gone', name: 'Gone', stencil: 'Missing', filled: [{ field: 'A', text: 'kept' }] },
    ]
    const held = mountDeck({ cards: orphan })
    const tile = tileFor(held, 'gone')

    // The stencil is named, so the person knows what the card is waiting for.
    expect(tile.get('.card__objects').text()).toBe('No stencil called Missing')
    // Nothing names the values, so nothing draws them. They stay in the file.
    expect(tile.findAll('.card__value')).toHaveLength(0)
    expect(tile.find('textarea').exists()).toBe(false)
  })

  it('says nothing about a card whose stencil was handed in', () => {
    expect(mountDeck().find('.card__objects').exists()).toBe(false)
  })

  it('draws every value a card writes under one field, and hides none of them', () => {
    const twice: readonly Drawn[] = [
      {
        id: 'llama',
        name: 'Llama',
        stencil: 'Animal',
        filled: [
          { field: 'Height', text: 'about 45"' },
          { field: 'Height', text: 'about 6 feet' },
        ],
      },
    ]
    const held = mountDeck({ cards: twice })
    const written = tileFor(held, 'llama')
      .findAll<HTMLTextAreaElement>('[data-value="Height"]')
      .map((box) => box.element.value)
    expect(written).toEqual(['about 45"', 'about 6 feet'])
  })

  it('leaves the caret in no box but the one it was put in when a field goes', async () => {
    const held = mountDeck()
    const box = boxFor(held, 'llama', 'Height').element
    box.focus()
    expect(document.activeElement).toBe(box)

    await held.setProps({ cuts: [{ name: 'Animal', fields: ['Name', 'Life span'] }] })

    // The stencil no longer names the field, so its box goes and its value
    // stays in the file. The caret goes with the box and lands in no other.
    expect(fieldsOf(held, 'llama')).toEqual(['Name', 'Life span'])
    expect(tileFor(held, 'llama').find('[data-value="Height"]').exists()).toBe(false)
    expect(document.activeElement).not.toBe(boxFor(held, 'llama', 'Life span').element)
  })

  describe('a card carrying the field that names it as a value as well', () => {
    const TWICE: readonly Drawn[] = [
      {
        id: 'twice',
        name: 'Llama',
        stencil: 'Animal',
        filled: [
          { field: 'Name', text: 'Alpaca' },
          { field: 'Height', text: 'about 45"' },
        ],
      },
    ]

    it('names the card by its heading, and not by the value', () => {
      const held = mountDeck({ cards: TWICE })
      expect(namedBoxes(held)[0]?.value).toBe('Llama')
    })

    it('keeps the value, and says the card is already named by that field', () => {
      const held = mountDeck({ cards: TWICE })
      const row = held.get('[data-twice]')
      expect(row.text()).toContain('Name')
      expect(row.get('.card__objects').text()).toBe('The card is named by this field')
      expect(namedBoxes(held)[1]?.value).toBe('Alpaca')
    })

    it('does not call it a field of another stencil, which it is not', () => {
      const held = mountDeck({ cards: TWICE })
      expect(held.get('[data-twice]').attributes('data-stray')).toBeUndefined()
      expect(held.text()).not.toContain('Not a field of this stencil')
    })

    it('tells the two boxes apart in what each of them emits', async () => {
      const held = mountDeck({ cards: TWICE })
      const boxes = namedBoxes(held)
      typed(boxes[0], 'Vicuña')
      typed(boxes[1], 'Guanaco')

      // The first is the heading, which is what names the card; the second is
      // the first value the card writes under that field.
      expect(held.emitted('write')).toEqual([
        ['twice', 'Name', 0, true, 'Vicuña'],
        ['twice', 'Name', 1, false, 'Guanaco'],
      ])
    })

    it('says nothing of the kind about a card leaving that field out', () => {
      expect(mountDeck().find('[data-twice]').exists()).toBe(false)
    })
  })
})
