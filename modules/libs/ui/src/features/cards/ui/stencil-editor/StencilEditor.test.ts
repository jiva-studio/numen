/**
 * What the stencil editor draws from the fields and faces it was handed, and
 * what it emits.
 *
 * What one face draws is that face's own, and is in `Face.test.ts`. Here are
 * the two orders, the two drags, and what each face is handed.
 *
 * The negatives are here: the first field has no handle and no way to go, a
 * name that objects renames nothing, a name abandoned renames nothing, and a
 * field let go where it was moves nothing.
 */
import { mount } from '@vue/test-utils'
import { afterEach, describe, expect, it } from 'vitest'
import Stencil from './StencilEditor.vue'
import type { StencilFace } from '../../lib/stencil'

const FIELDS = ['Name', 'Height', 'Weight']

const FACES: readonly StencilFace[] = [
  {
    id: 'recognise',
    name: 'Recognise',
    front: '{{Name}}',
    back: '**Height:** {{Height}}',
  },
]

const mountStencil = (props: Record<string, unknown> = {}) =>
  mount(Stencil, { attachTo: document.body, props: { fields: FIELDS, faces: FACES, ...props } })

type Editor = ReturnType<typeof mountStencil>

const rowFor = (held: Editor, field: string) => held.get(`[data-field="${field}"]`)
const boxIn = (held: Editor, field: string) => rowFor(held, field).get('input')

const getDrawnFields = (held: Editor) =>
  held.findAll('[data-field]').map((row) => row.attributes('data-field'))

/** A name typed into a box and not yet committed. */
const type = async (held: Editor, field: string, name: string): Promise<void> => {
  const box = boxIn(held, field)
  box.element.value = name
  await box.trigger('input')
}

/** A field picked up by its grip and let go over another, or over the tail. */
const dragTo = async (held: Editor, field: string, onto: string | null): Promise<void> => {
  await rowFor(held, field).get('[data-grip]').trigger('dragstart')
  if (onto === null) {
    await held.get('.stencil__fields').trigger('dragover')
    await held.get('.stencil__fields').trigger('drop')
    return
  }
  await rowFor(held, onto).trigger('dragover')
  await rowFor(held, onto).trigger('drop')
}

/** A key pressed on something, as the event it was pressed with. */
const pressKey = (on: Element, key: string): KeyboardEvent => {
  const press = new KeyboardEvent('keydown', { key, bubbles: true, cancelable: true })
  on.dispatchEvent(press)
  return press
}

/** A field picked up and the drag ended without it being let go anywhere. */
const dragOff = async (held: Editor, field: string, over: string | null): Promise<void> => {
  const grip = rowFor(held, field).get('[data-grip]')
  await grip.trigger('dragstart')
  if (over !== null) await rowFor(held, over).trigger('dragover')
  await grip.trigger('dragend')
}

afterEach(() => {
  document.body.innerHTML = ''
})

describe('Stencil, the fields', () => {
  it('draws a row per field, in the order they were handed in', () => {
    expect(getDrawnFields(mountStencil())).toEqual(FIELDS)
  })

  it('draws the silence, and no rows, for a stencil naming nothing', () => {
    const held = mountStencil({ fields: [] })
    expect(held.findAll('[data-field]')).toHaveLength(0)
    expect(held.get('.stencil__silence').text()).toBe('No fields yet')
  })

  it('marks the first field as the one naming the cards', () => {
    const held = mountStencil()
    expect(rowFor(held, 'Name').attributes('data-names')).toBe('true')
    expect(rowFor(held, 'Height').attributes('data-names')).toBeUndefined()
  })

  it('gives every row the same handle and the same way to remove it', () => {
    const held = mountStencil()
    for (const field of FIELDS) {
      expect(rowFor(held, field).findAll('[data-grip]')).toHaveLength(1)
      expect(rowFor(held, field).findAll('button')).toHaveLength(1)
    }
  })

  it('turns off the first field’s handle, and leaves every other one on', () => {
    const held = mountStencil()
    const first = rowFor(held, 'Name').get('[data-grip]')
    expect(first.attributes('draggable')).toBe('false')
    expect(first.attributes('data-disabled')).toBe('true')

    const other = rowFor(held, 'Height').get('[data-grip]')
    expect(other.attributes('draggable')).toBe('true')
    expect(other.attributes('data-disabled')).toBeUndefined()
  })

  it('turns off the first field’s way to be removed, and leaves every other one on', () => {
    const held = mountStencil()
    expect(rowFor(held, 'Name').get('button').attributes('disabled')).toBeDefined()
    expect(rowFor(held, 'Height').get('button').attributes('disabled')).toBeUndefined()
  })

  it('removes nothing when the first field’s button is pressed', async () => {
    const held = mountStencil()
    await rowFor(held, 'Name').get('button').trigger('click')
    expect(held.emitted('remove-field')).toBeUndefined()
  })

  it('moves nothing where a field is let go above the first', async () => {
    const held = mountStencil()
    await dragTo(held, 'Weight', 'Name')
    expect(held.emitted('move-field')).toBeUndefined()
  })

  it('renames the first field like any other', async () => {
    const held = mountStencil()
    await type(held, 'Name', 'Question')
    await boxIn(held, 'Name').trigger('change')
    expect(held.emitted('rename-field')).toEqual([['Name', 'Question']])
  })

  it('stands the way to add a field on a divider that is announced as nothing', () => {
    const held = mountStencil()
    const divider = held.get('.stencil__part .divider')
    expect(divider.attributes('role')).toBe('presentation')
    expect(divider.get('button').text()).toBe('Add a field')
    expect(held.findAll('hr')).toHaveLength(0)
  })

  it('asks for a field under a name nothing has taken', async () => {
    const held = mountStencil({ fields: ['Field', 'Field 2'] })
    await held.findAll('button').filter((each) => each.text() === 'Add a field')[0]?.trigger('click')
    expect(held.emitted('add-field')).toEqual([['Field 1']])
  })

  it('emits a name typed and committed', async () => {
    const held = mountStencil()
    await type(held, 'Height', 'Tallness')
    await boxIn(held, 'Height').trigger('change')
    expect(held.emitted('rename-field')).toEqual([['Height', 'Tallness']])
  })

  it('renames nothing where the name typed is another field’s', async () => {
    const held = mountStencil()
    await type(held, 'Height', 'Weight')
    await boxIn(held, 'Height').trigger('change')
    expect(held.emitted('rename-field')).toBeUndefined()
  })

  it('says why a name typed cannot be used, while it is being typed', async () => {
    const held = mountStencil()
    await type(held, 'Height', 'Weight')
    expect(rowFor(held, 'Height').get('.stencil__objects').text()).toBe('That name is taken')

    // The line belongs to the row, and the box says what is wrong with it to a
    // reader.
    expect(rowFor(held, 'Height').get('.stencil__row').attributes('data-objects')).toBe('taken')
    expect(boxIn(held, 'Height').attributes('aria-invalid')).toBe('true')
  })

  it('draws a row as one block: the handle, the box and the way to remove it stand inside it', () => {
    const held = mountStencil()
    for (const field of FIELDS) {
      const row = rowFor(held, field).get('.stencil__row')
      expect(row.find('[data-grip]').exists()).toBe(true)
      expect(row.find('input').exists()).toBe(true)
      expect(row.find('button').exists()).toBe(true)
    }
  })

  it('gives the turned-off row the same block, with nothing missing from it', () => {
    const held = mountStencil()
    const first = rowFor(held, 'Name').get('.stencil__row')
    const other = rowFor(held, 'Height').get('.stencil__row')

    const shape = (row: typeof first): readonly string[] =>
      [...row.element.children].map((each) => each.tagName)
    expect(shape(first)).toEqual(shape(other))
  })

  it('says nothing about a name nothing is being typed over', () => {
    const held = mountStencil()
    expect(held.find('.stencil__field .stencil__objects').exists()).toBe(false)
  })

  it('renames nothing where the name typed is the one it already carries', async () => {
    const held = mountStencil()
    await type(held, 'Height', 'Height')
    await boxIn(held, 'Height').trigger('change')
    expect(held.emitted('rename-field')).toBeUndefined()
  })

  it('renames nothing where a name is abandoned', async () => {
    const held = mountStencil()
    await type(held, 'Height', 'Tallness')
    await boxIn(held, 'Height').trigger('keydown', { key: 'Escape' })
    await boxIn(held, 'Height').trigger('change')
    expect(held.emitted('rename-field')).toBeUndefined()
    expect(boxIn(held, 'Height').element.value).toBe('Height')
  })

  it('emits the field asked to go', async () => {
    const held = mountStencil()
    await rowFor(held, 'Weight').findAll('button')[0]?.trigger('click')
    expect(held.emitted('remove-field')).toEqual([['Weight']])
  })

  it('emits a field let go before another below the first', async () => {
    const held = mountStencil()
    await dragTo(held, 'Weight', 'Height')
    expect(held.emitted('move-field')).toEqual([['Weight', 'Height']])
  })

  it('emits a field let go past the last of them', async () => {
    const held = mountStencil()
    await dragTo(held, 'Height', null)
    expect(held.emitted('move-field')).toEqual([['Height', null]])
  })

  it('emits a field let go on the rule the list is added by', async () => {
    const held = mountStencil()
    await rowFor(held, 'Height').get('[data-grip]').trigger('dragstart')
    const rule = held.findAll('section')[0]?.get('.divider')
    await rule?.trigger('dragover')
    await rule?.trigger('drop')
    expect(held.emitted('move-field')).toEqual([['Height', null]])
  })

  it('emits a field let go anywhere else in the part the fields stand in', async () => {
    const held = mountStencil()
    await rowFor(held, 'Height').get('[data-grip]').trigger('dragstart')
    const heading = held.get('.stencil__heading')
    await heading.trigger('dragover')
    await heading.trigger('drop')
    expect(held.emitted('move-field')).toEqual([['Height', null]])
  })

  it('moves nothing where a field is let go where it stands', async () => {
    const held = mountStencil()
    await dragTo(held, 'Height', 'Height')
    expect(held.emitted('move-field')).toBeUndefined()
  })

  it('moves nothing where a drag ends with the field let go nowhere', async () => {
    const held = mountStencil()
    await dragOff(held, 'Weight', null)
    expect(held.emitted('move-field')).toBeUndefined()
    expect(rowFor(held, 'Weight').attributes('data-dragged')).toBeUndefined()
  })

  it('moves nothing where a drag over another field ends with it let go nowhere', async () => {
    const held = mountStencil()
    await dragOff(held, 'Weight', 'Height')
    expect(held.emitted('move-field')).toBeUndefined()
  })

  it('marks the field on its way, and no other', async () => {
    const held = mountStencil()
    await rowFor(held, 'Weight').get('[data-grip]').trigger('dragstart')
    expect(rowFor(held, 'Weight').attributes('data-dragged')).toBe('true')
    expect(rowFor(held, 'Height').attributes('data-dragged')).toBeUndefined()
  })

  it('draws one row for a name the stencil declares twice', () => {
    const held = mountStencil({ fields: ['Name', 'Height', 'Height'] })
    expect(getDrawnFields(held)).toEqual(['Name', 'Height'])
    expect(held.findAll('.stencil__field input')).toHaveLength(2)
  })

  it('says what is wrong with a name to the box it is wrong about', async () => {
    const held = mountStencil()
    await type(held, 'Height', 'Weight')

    const said = rowFor(held, 'Height').get('.stencil__objects')
    expect(boxIn(held, 'Height').attributes('aria-describedby')).toBe(said.attributes('id'))
    expect(said.attributes('id')).toBeTruthy()
  })

  it('describes a box by nothing while what is in it can be used', () => {
    expect(boxIn(mountStencil(), 'Height').attributes('aria-describedby')).toBeUndefined()
  })

  describe('dragging a field by the keyboard', () => {
    const gripFor = (held: Editor, field: string) => rowFor(held, field).get('[data-grip]')

    it('names the handle, and gives it a place in the order', () => {
      const held = mountStencil()
      const grip = gripFor(held, 'Height')
      expect(grip.attributes('aria-label')).toBe('Reorder: Height')
      expect(grip.attributes('tabindex')).toBe('0')
      expect(grip.attributes('role')).toBe('button')
      expect(grip.attributes('aria-keyshortcuts')).toBe('ArrowUp ArrowDown')
    })

    it('names the first field’s handle by what it is, and takes it out of the order', () => {
      const grip = gripFor(mountStencil(), 'Name')
      expect(grip.attributes('aria-label')).toBe(
        'The first field names every card, and stays first',
      )
      expect(grip.attributes('tabindex')).toBe('-1')
      expect(grip.attributes('aria-disabled')).toBe('true')
      // A handle that goes nowhere along the order says no keys it answers.
      expect(grip.attributes('aria-keyshortcuts')).toBeUndefined()
    })

    it('emits a field dragged one place down the order', () => {
      const held = mountStencil()
      const press = pressKey(gripFor(held, 'Height').element, 'ArrowDown')
      expect(press.defaultPrevented).toBe(true)
      expect(held.emitted('move-field')).toEqual([['Height', null]])
    })

    it('emits a field dragged one place up the order', () => {
      const held = mountStencil()
      pressKey(gripFor(held, 'Weight').element, 'ArrowUp')
      expect(held.emitted('move-field')).toEqual([['Weight', 'Height']])
    })

    it('drags nothing above the first field, which names every card', () => {
      const held = mountStencil()
      pressKey(gripFor(held, 'Height').element, 'ArrowUp')
      expect(held.emitted('move-field')).toBeUndefined()
    })

    it('drags the first field nowhere', () => {
      const held = mountStencil()
      pressKey(gripFor(held, 'Name').element, 'ArrowDown')
      expect(held.emitted('move-field')).toBeUndefined()
    })
  })
})

describe('Stencil, what the caller found wrong', () => {
  const WRONG = {
    at: new Map([['recognise', ['this face has no back']]]),
    fields: new Map([['Height', ['declared twice']]]),
  }

  it('says what is wrong with a field under that field’s row', () => {
    const held = mountStencil({ wrong: WRONG })
    const said = rowFor(held, 'Height').get('[data-wrong]')
    expect(said.text()).toBe('declared twice')
  })

  it('says it on no other row', () => {
    const held = mountStencil({ wrong: WRONG })
    expect(rowFor(held, 'Weight').find('[data-wrong]').exists()).toBe(false)
  })

  it('says what is wrong with a face under that face’s name', () => {
    const held = mountStencil({ wrong: WRONG })
    const said = held.get('[data-face="recognise"] header [data-wrong]')
    expect(said.text()).toBe('this face has no back')
  })

  it('says a line for each of them', () => {
    const wrong = {
      at: new Map(),
      fields: new Map([['Height', ['declared twice', 'and nothing fills it']]]),
    }
    const held = mountStencil({ wrong })
    expect(rowFor(held, 'Height').get('[data-wrong]').findAll('li')).toHaveLength(2)
  })

  it('says nothing where the caller found nothing wrong', () => {
    expect(mountStencil().findAll('[data-wrong]')).toHaveLength(0)
  })
})

describe('Stencil, the faces', () => {
  const TWO: readonly StencilFace[] = [
    { id: 'one', name: 'One', front: '', back: '' },
    { id: 'two', name: 'Two', front: '', back: '' },
  ]

  it('draws one of them per face', () => {
    expect(mountStencil().findAll('[data-face]')).toHaveLength(1)
  })

  it('draws the silence, and no faces, for a stencil showing nothing', () => {
    const held = mountStencil({ faces: [] })
    expect(held.findAll('[data-face]')).toHaveLength(0)
    expect(held.text()).toContain('No faces yet')
  })

  it('hands every face the fields the stencil declares', () => {
    const held = mountStencil({ faces: TWO })
    for (const id of ['one', 'two']) {
      const said = held
        .findAll(`[data-face="${id}"] [data-insert]`)
        .map((each) => each.attributes('data-insert'))
      expect(said).toEqual(FIELDS)
    }
  })

  it('hands a face the names the other faces carry', async () => {
    const held = mountStencil({ faces: TWO })
    const box = held.get<HTMLInputElement>('[data-face="one"] header input')
    box.element.value = 'Two'
    await box.trigger('input')
    expect(held.get('[data-face="one"] header [role="alert"]').text()).toBe(
      'That name is taken',
    )
  })

  it('emits the face and the half a box was typed into', async () => {
    const held = mountStencil()
    await held.get('[data-face="recognise"] [data-half="back"]').setValue('nothing but words')
    expect(held.emitted('write')).toEqual([['recognise', 'back', 'nothing but words']])
  })

  it('stands the way to add a face on a rule of the same make', () => {
    const held = mountStencil()
    const rules = held.findAll('.divider')
    expect(rules.map((rule) => rule.get('button').text())).toEqual([
      'Add a field',
      'Add a face',
    ])
    for (const rule of rules) expect(rule.attributes('role')).toBe('presentation')
  })

  it('asks for a face under a name nothing has taken', async () => {
    const held = mountStencil({ faces: [{ id: 'a', name: 'Face', front: '', back: '' }] })
    await held.findAll('button').filter((each) => each.text() === 'Add a face')[0]?.trigger('click')
    expect(held.emitted('add-face')).toEqual([['Face 1']])
  })

  it('emits the face asked to go', async () => {
    const held = mountStencil()
    await held.get('[data-face="recognise"] .card-header__actions button').trigger('click')
    expect(held.emitted('remove-face')).toEqual([['recognise']])
  })

  it('emits a face renamed', async () => {
    const held = mountStencil()
    const box = held.get<HTMLInputElement>('[data-face="recognise"] header input')
    box.element.value = 'Name it'
    await box.trigger('input')
    await box.trigger('change')
    expect(held.emitted('rename-face')).toEqual([['recognise', 'Name it']])
  })

  describe('dragging a face by the keyboard', () => {
    const THREE: readonly StencilFace[] = [
      { id: 'one', name: 'One', front: '', back: '' },
      { id: 'two', name: 'Two', front: '', back: '' },
      { id: 'three', name: 'Three', front: '', back: '' },
    ]

    const stripOf = (held: Editor, id: string) =>
      held.get(`[data-face="${id}"] .card-header__grip`)

    it('names the handle a face is dragged by, and gives it a place in the order', () => {
      const strip = stripOf(mountStencil({ faces: THREE }), 'two')
      expect(strip.attributes('aria-label')).toBe('Reorder: Two')
      expect(strip.attributes('tabindex')).toBe('0')
      expect(strip.attributes('role')).toBe('button')
    })

    it('emits a face dragged one place down the order', () => {
      const held = mountStencil({ faces: THREE })
      const press = pressKey(stripOf(held, 'one').element, 'ArrowDown')
      expect(press.defaultPrevented).toBe(true)
      expect(held.emitted('move-face')).toEqual([['one', 'three']])
    })

    it('emits a face dragged one place up the order: nothing among them is fixed', () => {
      const held = mountStencil({ faces: THREE })
      pressKey(stripOf(held, 'two').element, 'ArrowUp')
      expect(held.emitted('move-face')).toEqual([['two', 'one']])
    })

    it('moves nothing where there is no place that way', () => {
      const held = mountStencil({ faces: THREE })
      pressKey(stripOf(held, 'one').element, 'ArrowUp')
      expect(held.emitted('move-face')).toBeUndefined()
    })
  })

  describe('reordering the faces', () => {
    const THREE: readonly StencilFace[] = [
      { id: 'one', name: 'One', front: '', back: '' },
      { id: 'two', name: 'Two', front: '', back: '' },
      { id: 'three', name: 'Three', front: '', back: '' },
    ]

    const faceFor = (editor: Editor, id: string) => editor.get(`[data-face="${id}"]`)

    /** A face picked up by its strip and let go over another, or over the tail. */
    const drag = async (editor: Editor, id: string, onto: string | null): Promise<void> => {
      await faceFor(editor, id).get('.card-header').trigger('dragstart')
      const over = onto === null ? editor.findAll('section')[1] : faceFor(editor, onto)
      await over?.trigger('dragover')
      await over?.trigger('drop')
    }

    it('emits a face let go before another', async () => {
      const editor = mountStencil({ faces: THREE })
      await drag(editor, 'three', 'one')
      expect(editor.emitted('move-face')).toEqual([['three', 'one']])
    })

    it('emits a face let go past the last of them', async () => {
      const editor = mountStencil({ faces: THREE })
      await drag(editor, 'one', null)
      expect(editor.emitted('move-face')).toEqual([['one', null]])
    })

    it('moves the first face like any other: nothing among the faces is fixed', async () => {
      const editor = mountStencil({ faces: THREE })
      await drag(editor, 'one', 'three')
      expect(editor.emitted('move-face')).toEqual([['one', 'three']])
    })

    it('lands a face above the first, which no rule forbids', async () => {
      const editor = mountStencil({ faces: THREE })
      await drag(editor, 'two', 'one')
      expect(editor.emitted('move-face')).toEqual([['two', 'one']])
    })

    it('moves nothing where a face is let go where it stands', async () => {
      const editor = mountStencil({ faces: THREE })
      await drag(editor, 'two', 'two')
      expect(editor.emitted('move-face')).toBeUndefined()
    })

    /** A face picked up and the drag ended without it being let go anywhere. */
    const dragOff = async (editor: Editor, id: string, onto: string | null): Promise<void> => {
      const header = faceFor(editor, id).get('.card-header')
      await header.trigger('dragstart')
      if (onto !== null) await faceFor(editor, onto).trigger('dragover')
      await header.trigger('dragend')
    }

    it('moves nothing where a drag ends with the face let go nowhere', async () => {
      const editor = mountStencil({ faces: THREE })
      await dragOff(editor, 'two', null)
      expect(editor.emitted('move-face')).toBeUndefined()
      expect(faceFor(editor, 'two').attributes('data-dragged')).toBeUndefined()
    })

    it('moves nothing where a drag over another face ends with it let go nowhere', async () => {
      const editor = mountStencil({ faces: THREE })
      await dragOff(editor, 'two', 'one')
      expect(editor.emitted('move-face')).toBeUndefined()
    })

    it('marks the face on its way, and no other', async () => {
      const editor = mountStencil({ faces: THREE })
      await faceFor(editor, 'two').get('.card-header').trigger('dragstart')
      expect(faceFor(editor, 'two').attributes('data-dragged')).toBe('true')
      expect(faceFor(editor, 'one').attributes('data-dragged')).toBeUndefined()
    })

    it('moves no field when a face is dragged', async () => {
      const editor = mountStencil({ faces: THREE })
      await drag(editor, 'three', 'one')
      expect(editor.emitted('move-field')).toBeUndefined()
    })
  })

})
