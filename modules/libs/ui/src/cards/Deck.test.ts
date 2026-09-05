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
import { endOf, HEAD, type Banded, type Drawn } from './deck'
import type { Stencil } from './stencil'

const CUTS: readonly Stencil[] = [
  { name: 'Animal', fields: ['Name', 'Height', 'Life span'] },
  { name: 'Word', fields: ['Word', 'Meaning'] },
]

/* Every field a stencil declares stands under its own heading, the first
   included, and a card is addressed by the identity the caller drew it under. */
const CARDS: readonly Drawn[] = [
  {
    id: 'llama',
    section: null,
    stencil: 'Animal',
    filled: [
      { field: 'Name', text: 'Llama' },
      { field: 'Height', text: 'about 45"' },
    ],
  },
  { id: 'yak', section: null, stencil: 'Animal', filled: [] },
]

const mountDeck = (props: Record<string, unknown> = {}) =>
  mount(Deck, { attachTo: document.body, props: { cards: CARDS, stencils: CUTS, ...props } })

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

/** Every box standing under one field of one card, in the order they are drawn. */
const boxesFor = (held: Grid, id: string, field: string): readonly HTMLTextAreaElement[] =>
  tileFor(held, id)
    .findAll<HTMLTextAreaElement>(`[data-value="${field}"]`)
    .map((box) => box.element)

/**
 * A card picked up by its grip and let go over another card, the head of the
 * deck, a section, or the grid.
 */
const dragTo = async (held: Grid, id: string, onto: string | null): Promise<void> => {
  await tileFor(held, id).get('[data-grip]').trigger('dragstart')
  const over =
    onto === null
      ? held.findAll('[data-plus]').at(-1)!
      : onto === HEAD
        ? held.get('[data-head]')
        : held.find(`[data-card="${onto}"]`).exists()
          ? tileFor(held, onto)
          : held.get(`[data-band="${onto}"]`)
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

  it('draws every field of the stencil in a box, the first among them', () => {
    expect(fieldsOf(mountDeck(), 'llama')).toEqual(['Name', 'Height', 'Life span'])
  })

  it('draws the first field as it draws every other', () => {
    const held = mountDeck()
    const values = tileFor(held, 'llama').findAll('.card__value')
    expect(values.map((each) => each.get('label').text())).toEqual([
      'Name',
      'Height',
      'Life span',
    ])
    expect(values[0]?.attributes('data-names')).toBeUndefined()
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

  it('stands what the card wrote under the first field in that field’s box', () => {
    expect(boxFor(mountDeck(), 'llama', 'Name').element.value).toBe('Llama')
  })

  it('holds what was typed, breaks and all, in every box, the first field’s among them', async () => {
    const held = mountDeck()
    const box = boxFor(held, 'llama', 'Name')
    expect(pressing(box.element, 'Enter').defaultPrevented).toBe(false)

    await box.setValue('Llama\nand alpaca')
    expect(held.emitted('write')).toEqual([['llama', 'Name', 1, 'Llama\nand alpaca']])
  })

  it('emits one value of one card as it now reads', async () => {
    const held = mountDeck()
    await boxFor(held, 'llama', 'Life span').setValue('about 20 years')
    expect(held.emitted('write')).toEqual([['llama', 'Life span', 1, 'about 20 years']])
  })

  it('sets the ground behind each box to the box’s own text, which is what sizes it', () => {
    const cards: readonly Drawn[] = [
      {
        id: 'x',
        section: null,
        stencil: 'Animal',
        filled: [{ field: 'Height', text: 'a\nb\nc' }],
      },
    ]
    const held = mountDeck({ cards })
    const boxes = tileFor(held, 'x').findAll('[data-autosize]')
    expect(boxes.map((each) => each.attributes('data-autosize'))).toEqual(['', 'a\nb\nc', ''])
  })

  it('says on each tile what cut the card it draws', () => {
    // Two stencils, named nothing that any field is named, so what is looked
    // for here can only be the stencil's own name.
    const stencils: readonly Stencil[] = [
      { name: 'Beast', fields: ['Name', 'Height'] },
      { name: 'Vocabulary', fields: ['Word', 'Meaning'] },
    ]
    const mixed: readonly Drawn[] = [
      { id: 'llama', section: null, stencil: 'Beast', filled: [] },
      { id: 'llano', section: null, stencil: 'Vocabulary', filled: [] },
    ]
    const held = mountDeck({ cards: mixed, stencils })

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

  it('announces a card by the place it stands in the deck, and by no name of its own', () => {
    const held = mountDeck()
    expect(tileFor(held, 'yak').attributes('aria-label')).toBe('Card 2')
  })

  describe('carrying a tile by the keyboard', () => {
    const stripOf = (held: Grid, id: string) => tileFor(held, id).get('[data-grip]')

    it('names the strip a tile is carried by, and gives it a place in the order', () => {
      const strip = stripOf(mountDeck(), 'llama')
      expect(strip.attributes('aria-label')).toBe('Reorder: Card 1')
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

  it('asks for a card cut by the stencil chosen, before the first section', async () => {
    const held = mountDeck()
    await held.get('[data-plus] button').trigger('click')
    await held.get('[data-cut="Animal"]').trigger('click')
    expect(held.emitted('add')).toEqual([['Animal', null]])
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

  it('emits a card let go past the last of them, which is where the plus stands', async () => {
    const held = mountDeck()
    await dragTo(held, 'llama', null)
    expect(held.emitted('move')).toEqual([['llama', endOf(HEAD)]])
  })

  it('moves nothing where the card let go past the last of them is the last of them', async () => {
    const held = mountDeck()
    await dragTo(held, 'yak', null)
    expect(held.emitted('move')).toBeUndefined()
  })

  // A card is put somewhere, and the caret says where. Let go where the caret
  // says nothing — the ground between the tiles — it stays where it was.
  it('moves nothing where a card is let go on no place at all', async () => {
    const held = mountDeck()
    await tileFor(held, 'llama').get('[data-grip]').trigger('dragstart')
    await held.get('.deck').trigger('dragover')
    await held.get('.deck').trigger('drop')
    expect(held.emitted('move')).toBeUndefined()
  })

  it('forgets where a card would land once it is carried off every place', async () => {
    const held = mountDeck()
    await tileFor(held, 'llama').get('[data-grip]').trigger('dragstart')
    await tileFor(held, 'yak').trigger('dragover')
    await held.get('.deck').trigger('dragover')
    await held.get('.deck').trigger('drop')
    expect(held.emitted('move')).toBeUndefined()
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

  it('draws no value of a card whose stencil was not handed in, and says which it wants', () => {
    const orphan: readonly Drawn[] = [
      { id: 'gone', section: null, stencil: 'Missing', filled: [{ field: 'A', text: 'kept' }] },
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
        section: null,
        stencil: 'Animal',
        filled: [
          { field: 'Height', text: 'about 45"' },
          { field: 'Height', text: 'about 6 feet' },
        ],
      },
    ]
    const held = mountDeck({ cards: twice })
    expect(boxesFor(held, 'llama', 'Height').map((box) => box.value)).toEqual([
      'about 45"',
      'about 6 feet',
    ])
  })

  it('tells the two boxes of a field written twice apart in what each emits', async () => {
    const twice: readonly Drawn[] = [
      {
        id: 'twice',
        section: null,
        stencil: 'Animal',
        filled: [
          { field: 'Name', text: 'Llama' },
          { field: 'Name', text: 'Alpaca' },
        ],
      },
    ]
    const held = mountDeck({ cards: twice })
    const boxes = boxesFor(held, 'twice', 'Name')
    typed(boxes[0], 'Vicuña')
    typed(boxes[1], 'Guanaco')

    expect(held.emitted('write')).toEqual([
      ['twice', 'Name', 1, 'Vicuña'],
      ['twice', 'Name', 2, 'Guanaco'],
    ])
  })

  it('leaves the caret in no box but the one it was put in when a field goes', async () => {
    const held = mountDeck()
    const box = boxFor(held, 'llama', 'Height').element
    box.focus()
    expect(document.activeElement).toBe(box)

    await held.setProps({ stencils: [{ name: 'Animal', fields: ['Name', 'Life span'] }] })

    // The stencil no longer names the field, so its box goes and its value
    // stays in the file. The caret goes with the box and lands in no other.
    expect(fieldsOf(held, 'llama')).toEqual(['Name', 'Life span'])
    expect(tileFor(held, 'llama').find('[data-value="Height"]').exists()).toBe(false)
    expect(document.activeElement).not.toBe(boxFor(held, 'llama', 'Life span').element)
  })

  describe('the sections of a deck', () => {
    const SECTIONS: readonly Banded[] = [
      { id: 'roots', name: 'Roots' },
      { id: 'leaves', name: 'Leaves' },
    ]
    const UNDER: readonly Drawn[] = [
      { id: 'loose', section: null, stencil: 'Animal', filled: [] },
      { id: 'llama', section: 'roots', stencil: 'Animal', filled: [] },
    ]

    const mountSectioned = (props: Record<string, unknown> = {}) =>
      mountDeck({ cards: UNDER, sections: SECTIONS, ...props })

    it('draws a heading per section, in the order they were handed in', () => {
      const held = mountSectioned()
      expect(held.findAll('[data-band]').map((band) => band.attributes('data-band'))).toEqual([
        'roots',
        'leaves',
      ])
    })

    it('draws a heading for a section holding no card', () => {
      const held = mountSectioned()
      expect(held.get('[data-band="leaves"]').get('input').element.value).toBe('Leaves')
    })

    it('stands each card under the section it was handed in under', () => {
      const held = mountSectioned()
      expect(tileFor(held, 'llama').attributes('data-section')).toBe('roots')
      expect(tileFor(held, 'loose').attributes('data-section')).toBeUndefined()
    })

    it('draws no heading for a deck the caller handed no section', () => {
      expect(mountDeck().findAll('[data-band]')).toHaveLength(0)
    })

    it('emits a section asked for, named by something nothing has taken', async () => {
      const held = mountSectioned()
      await held.get('[data-add-section]').trigger('click')
      expect(held.emitted('add-section')).toEqual([['Section 1']])
    })

    it('emits a section renamed', async () => {
      const held = mountSectioned()
      const box = held.get<HTMLInputElement>('[data-band="roots"] input')
      box.element.value = 'Roots and shoots'
      await box.trigger('input')
      await box.trigger('change')
      expect(held.emitted('rename-section')).toEqual([['roots', 'Roots and shoots']])
    })

    it('emits a section asked to go', async () => {
      const held = mountSectioned()
      await held.get('[data-band="roots"] .remove').trigger('click')
      expect(held.emitted('remove-section')).toEqual([['roots']])
    })

    it('asks for a card under the section whose plus was pressed', async () => {
      const held = mountSectioned()
      await held.get('[data-plus-of="roots"] button').trigger('click')
      await held.get('[data-plus-of="roots"] [data-cut="Animal"]').trigger('click')
      expect(held.emitted('add')?.map((call) => call[1])).toEqual(['roots'])
    })

    it('asks from one plus at a time, the others standing shut', async () => {
      const held = mountSectioned()
      await held.get('[data-plus-of="roots"] button').trigger('click')
      expect(held.findAll('[data-cut]').length).toBeGreaterThan(0)
      expect(held.find('[data-plus-of="leaves"] [data-cut]').exists()).toBe(false)

      await held.get('[data-plus-of="leaves"] button').trigger('click')
      expect(held.find('[data-plus-of="roots"] [data-cut]').exists()).toBe(false)
      expect(held.find('[data-plus-of="leaves"] [data-cut]').exists()).toBe(true)
    })

    it('moves nothing where the card let go past a section is the last under it', async () => {
      const held = mountSectioned()
      const plus = held.get('[data-plus-of="roots"]')
      await tileFor(held, 'llama').get('[data-grip]').trigger('dragstart')
      await plus.trigger('dragover')
      await plus.trigger('drop')
      expect(held.emitted('move')).toBeUndefined()
    })

    it('stands a plus in every section, and one where the loose cards stand', () => {
      const held = mountSectioned()
      expect(
        held.findAll('[data-plus]').map((plus) => plus.attributes('data-plus-of') ?? ''),
      ).toEqual(['', 'roots', 'leaves'])
    })

    it('stands none before the first section where no card stands there', () => {
      const held = mountSectioned({ cards: [UNDER[1]] })
      expect(
        held.findAll('[data-plus]').map((plus) => plus.attributes('data-plus-of') ?? ''),
      ).toEqual(['roots', 'leaves'])
    })

    it('emits a card let go past the last card of a section', async () => {
      const held = mountSectioned()
      const plus = held.get('[data-plus-of="roots"]')
      await tileFor(held, 'loose').get('[data-grip]').trigger('dragstart')
      await plus.trigger('dragover')
      await plus.trigger('drop')
      expect(held.emitted('move')).toEqual([['loose', endOf('roots')]])
    })

    it('emits a card let go past the last of those standing under no section', async () => {
      const held = mountSectioned()
      const plus = held.get('[data-plus]')
      await tileFor(held, 'llama').get('[data-grip]').trigger('dragstart')
      await plus.trigger('dragover')
      await plus.trigger('drop')
      expect(held.emitted('move')).toEqual([['llama', endOf(HEAD)]])
    })

    it('emits a card let go at the head of a section', async () => {
      const held = mountSectioned()
      await dragTo(held, 'loose', 'roots')
      expect(held.emitted('move')).toEqual([['loose', 'roots']])
    })

    it('emits a card let go at the head of a section holding none', async () => {
      const held = mountSectioned()
      await dragTo(held, 'llama', 'leaves')
      expect(held.emitted('move')).toEqual([['llama', 'leaves']])
    })

    describe('the cards before the first section', () => {
      /* A deck whose every card stands in a section, so the place before the
         first of them holds none. */
      const INSIDE: readonly Drawn[] = [
        { id: 'llama', section: 'roots', stencil: 'Animal', filled: [] },
        { id: 'yak', section: 'leaves', stencil: 'Animal', filled: [] },
      ]

      it('take a landing of their own where no card stands there', () => {
        expect(mountSectioned({ cards: INSIDE }).find('[data-head]').exists()).toBe(true)
      })

      it('take none where a card already stands there to land in front of', () => {
        expect(mountSectioned().find('[data-head]').exists()).toBe(false)
      })

      it('take none in a deck holding no section, the end of it being that place', () => {
        expect(mountDeck().find('[data-head]').exists()).toBe(false)
      })

      it('emit a card let go before the first section', async () => {
        const held = mountSectioned({ cards: INSIDE })
        await dragTo(held, 'yak', HEAD)
        expect(held.emitted('move')).toEqual([['yak', HEAD]])
      })

      it('emit the first card of the deck carried out of its section', () => {
        const held = mountSectioned({ cards: INSIDE })
        const press = pressing(tileFor(held, 'llama').get('[data-grip]').element, 'ArrowUp')
        expect(press.defaultPrevented).toBe(true)
        expect(held.emitted('move')).toEqual([['llama', HEAD]])
      })

      it('move nothing where the card carried up already stands there', () => {
        const held = mountSectioned()
        const press = pressing(tileFor(held, 'loose').get('[data-grip]').element, 'ArrowUp')
        expect(press.defaultPrevented).toBe(false)
        expect(held.emitted('move')).toBeUndefined()
      })

      it('hold a card standing under a section the deck was not handed', () => {
        const lost: readonly Drawn[] = [
          { id: 'lost', section: 'gone', stencil: 'Animal', filled: [] },
          { id: 'llama', section: 'roots', stencil: 'Animal', filled: [] },
        ]
        const held = mountSectioned({ cards: lost })

        expect(drawnCards(held)).toEqual(['lost', 'llama'])
        expect(tileFor(held, 'lost').attributes('data-section')).toBeUndefined()
        // Every card is drawn, so what each tile is announced by counts them
        // all and the plus each run carries.
        expect(tileFor(held, 'lost').attributes('aria-setsize')).toBe('5')
      })
    })
  })
})
