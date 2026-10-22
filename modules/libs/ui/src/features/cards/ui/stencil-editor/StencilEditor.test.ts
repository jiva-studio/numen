/**
 * What the stencil editor draws from the fields and faces it was handed, and
 * what it emits.
 *
 * What one face draws is that face's own, and is in `Face.test.ts`. Here are
 * the two orders, the two drags, and what each face is handed.
 *
 * The negatives are here: the first field has no handle and no way to go, a
 * name that objects does not rename, a name abandoned does not rename, and a
 * field let go where it was does not move.
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

const getRow = (wrapper: Editor, field: string) => wrapper.get(`[data-field="${field}"]`)
const getBox = (wrapper: Editor, field: string) => getRow(wrapper, field).get('input')

const getDrawnFields = (wrapper: Editor) =>
  wrapper.findAll('[data-field]').map((row) => row.attributes('data-field'))

/** A name typed into a box and not yet committed. */
const type = async (wrapper: Editor, field: string, name: string): Promise<void> => {
  const box = getBox(wrapper, field)
  box.element.value = name
  await box.trigger('input')
}

/** A field picked up by its grip and let go over another, or over the tail. */
const dragTo = async (wrapper: Editor, field: string, onto: string | null): Promise<void> => {
  await getRow(wrapper, field).get('[data-grip]').trigger('dragstart')
  if (onto === null) {
    await wrapper.get('.stencil__fields').trigger('dragover')
    await wrapper.get('.stencil__fields').trigger('drop')
    return
  }
  await getRow(wrapper, onto).trigger('dragover')
  await getRow(wrapper, onto).trigger('drop')
}

/** A key pressed on something, as the event it was pressed with. */
const pressKey = (on: Element, key: string): KeyboardEvent => {
  const press = new KeyboardEvent('keydown', { key, bubbles: true, cancelable: true })
  on.dispatchEvent(press)
  return press
}

/** A field picked up and the drag ended without it being let go anywhere. */
const dragOff = async (wrapper: Editor, field: string, over: string | null): Promise<void> => {
  const grip = getRow(wrapper, field).get('[data-grip]')
  await grip.trigger('dragstart')
  if (over !== null) await getRow(wrapper, over).trigger('dragover')
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
    const target = mountStencil({ fields: [] })
    expect(target.findAll('[data-field]')).toHaveLength(0)
    expect(target.get('.stencil__silence').text()).toBe('No fields yet')
  })

  it('marks the first field as the one naming the cards', () => {
    const target = mountStencil()
    expect(getRow(target, 'Name').attributes('data-names')).toBe('true')
    expect(getRow(target, 'Height').attributes('data-names')).toBeUndefined()
  })

  it('gives every row the same handle and the same way to remove it', () => {
    const target = mountStencil()
    for (const field of FIELDS) {
      expect(getRow(target, field).findAll('[data-grip]')).toHaveLength(1)
      expect(getRow(target, field).findAll('button')).toHaveLength(1)
    }
  })

  it('turns off the first field’s handle, and leaves every other one on', () => {
    const target = mountStencil()
    const first = getRow(target, 'Name').get('[data-grip]')
    expect(first.attributes('draggable')).toBe('false')
    expect(first.attributes('data-disabled')).toBe('true')

    const other = getRow(target, 'Height').get('[data-grip]')
    expect(other.attributes('draggable')).toBe('true')
    expect(other.attributes('data-disabled')).toBeUndefined()
  })

  it('turns off the first field’s way to be removed, and leaves every other one on', () => {
    const target = mountStencil()
    expect(getRow(target, 'Name').get('button').attributes('disabled')).toBeDefined()
    expect(getRow(target, 'Height').get('button').attributes('disabled')).toBeUndefined()
  })

  it('redoes not move when the first field’s button is pressed', async () => {
    const target = mountStencil()
    await getRow(target, 'Name').get('button').trigger('click')
    expect(target.emitted('remove-field')).toBeUndefined()
  })

  it('does not move where a field is let go above the first', async () => {
    const target = mountStencil()
    await dragTo(target, 'Weight', 'Name')
    expect(target.emitted('move-field')).toBeUndefined()
  })

  it('renames the first field like any other', async () => {
    const target = mountStencil()
    await type(target, 'Name', 'Question')
    await getBox(target, 'Name').trigger('change')
    expect(target.emitted('rename-field')).toEqual([['Name', 'Question']])
  })

  it('remains the way to add a field on a divider that is announced as nothing', () => {
    const target = mountStencil()
    const divider = target.get('.stencil__part .divider')
    expect(divider.attributes('role')).toBe('presentation')
    expect(divider.get('button').text()).toBe('Add a field')
    expect(target.findAll('hr')).toHaveLength(0)
  })

  it('asks for a field under a name nothing has taken', async () => {
    const target = mountStencil({ fields: ['Field', 'Field 2'] })
    await target
      .findAll('button')
      .filter((each) => each.text() === 'Add a field')[0]
      ?.trigger('click')
    expect(target.emitted('add-field')).toEqual([['Field 1']])
  })

  it('emits a name typed and committed', async () => {
    const target = mountStencil()
    await type(target, 'Height', 'Tallness')
    await getBox(target, 'Height').trigger('change')
    expect(target.emitted('rename-field')).toEqual([['Height', 'Tallness']])
  })

  it('does not rename where the name typed is another field’s', async () => {
    const target = mountStencil()
    await type(target, 'Height', 'Weight')
    await getBox(target, 'Height').trigger('change')
    expect(target.emitted('rename-field')).toBeUndefined()
  })

  it('says why a name typed cannot be used, while it is being typed', async () => {
    const target = mountStencil()
    await type(target, 'Height', 'Weight')
    expect(getRow(target, 'Height').get('.stencil__objections').text()).toBe('That name is taken')

    // The line belongs to the row, and the box says what is wrong with it to a
    // reader.
    expect(getRow(target, 'Height').get('.stencil__row').attributes('data-objections')).toBe('taken')
    expect(getBox(target, 'Height').attributes('aria-invalid')).toBe('true')
  })

  it('draws a row as one block: the handle, the box and the way to remove it stand inside it', () => {
    const target = mountStencil()
    for (const field of FIELDS) {
      const row = getRow(target, field).get('.stencil__row')
      expect(row.find('[data-grip]').exists()).toBe(true)
      expect(row.find('input').exists()).toBe(true)
      expect(row.find('button').exists()).toBe(true)
    }
  })

  it('gives the turned-off row the same block, with nothing missing from it', () => {
    const target = mountStencil()
    const first = getRow(target, 'Name').get('.stencil__row')
    const other = getRow(target, 'Height').get('.stencil__row')

    const shape = (row: typeof first): readonly string[] =>
      [...row.element.children].map((each) => each.tagName)
    expect(shape(first)).toEqual(shape(other))
  })

  it('says nothing about a name nothing is being typed over', () => {
    const target = mountStencil()
    expect(target.find('.stencil__field .stencil__objections').exists()).toBe(false)
  })

  it('does not rename where the name typed is the one it already carries', async () => {
    const target = mountStencil()
    await type(target, 'Height', 'Height')
    await getBox(target, 'Height').trigger('change')
    expect(target.emitted('rename-field')).toBeUndefined()
  })

  it('does not rename where a name is abandoned', async () => {
    const target = mountStencil()
    await type(target, 'Height', 'Tallness')
    await getBox(target, 'Height').trigger('keydown', { key: 'Escape' })
    await getBox(target, 'Height').trigger('change')
    expect(target.emitted('rename-field')).toBeUndefined()
    expect(getBox(target, 'Height').element.value).toBe('Height')
  })

  it('emits the field requested to go', async () => {
    const target = mountStencil()
    await getRow(target, 'Weight').findAll('button')[0]?.trigger('click')
    expect(target.emitted('remove-field')).toEqual([['Weight']])
  })

  it('emits a field let go before another below the first', async () => {
    const target = mountStencil()
    await dragTo(target, 'Weight', 'Height')
    expect(target.emitted('move-field')).toEqual([['Weight', 'Height']])
  })

  it('emits a field let go past the last of them', async () => {
    const target = mountStencil()
    await dragTo(target, 'Height', null)
    expect(target.emitted('move-field')).toEqual([['Height', null]])
  })

  it('emits a field let go on the rule the list is added by', async () => {
    const target = mountStencil()
    await getRow(target, 'Height').get('[data-grip]').trigger('dragstart')
    const rule = target.findAll('section')[0]?.get('.divider')
    await rule?.trigger('dragover')
    await rule?.trigger('drop')
    expect(target.emitted('move-field')).toEqual([['Height', null]])
  })

  it('emits a field let go anywhere else in the part the fields stand in', async () => {
    const target = mountStencil()
    await getRow(target, 'Height').get('[data-grip]').trigger('dragstart')
    const heading = target.get('.stencil__heading')
    await heading.trigger('dragover')
    await heading.trigger('drop')
    expect(target.emitted('move-field')).toEqual([['Height', null]])
  })

  it('does not move where a field is let go at its current position', async () => {
    const target = mountStencil()
    await dragTo(target, 'Height', 'Height')
    expect(target.emitted('move-field')).toBeUndefined()
  })

  it('does not move where a drag ends with the field let go nowhere', async () => {
    const target = mountStencil()
    await dragOff(target, 'Weight', null)
    expect(target.emitted('move-field')).toBeUndefined()
    expect(getRow(target, 'Weight').attributes('data-dragged')).toBeUndefined()
  })

  it('does not move where a drag over another field ends with it let go nowhere', async () => {
    const target = mountStencil()
    await dragOff(target, 'Weight', 'Height')
    expect(target.emitted('move-field')).toBeUndefined()
  })

  it('marks the field on its way, and no other', async () => {
    const target = mountStencil()
    await getRow(target, 'Weight').get('[data-grip]').trigger('dragstart')
    expect(getRow(target, 'Weight').attributes('data-dragged')).toBe('true')
    expect(getRow(target, 'Height').attributes('data-dragged')).toBeUndefined()
  })

  it('draws one row for a name the stencil declares twice', () => {
    const target = mountStencil({ fields: ['Name', 'Height', 'Height'] })
    expect(getDrawnFields(target)).toEqual(['Name', 'Height'])
    expect(target.findAll('.stencil__field input')).toHaveLength(2)
  })

  it('says what is wrong with a name to the box it is wrong about', async () => {
    const target = mountStencil()
    await type(target, 'Height', 'Weight')

    const response = getRow(target, 'Height').get('.stencil__objections')
    expect(getBox(target, 'Height').attributes('aria-describedby')).toBe(response.attributes('id'))
    expect(response.attributes('id')).toBeTruthy()
  })

  it('describes a box by nothing while what is in it can be used', () => {
    expect(getBox(mountStencil(), 'Height').attributes('aria-describedby')).toBeUndefined()
  })

  describe('dragging a field by the keyboard', () => {
    const getGrip = (wrapper: Editor, field: string) => getRow(wrapper, field).get('[data-grip]')

    it('names the handle, and gives it a place in the order', () => {
      const target = mountStencil()
      const grip = getGrip(target, 'Height')
      expect(grip.attributes('aria-label')).toBe('Reorder: Height')
      expect(grip.attributes('tabindex')).toBe('0')
      expect(grip.attributes('role')).toBe('button')
      expect(grip.attributes('aria-keyshortcuts')).toBe('ArrowUp ArrowDown')
    })

    it('names the first field’s handle by what it is, and takes it out of the order', () => {
      const grip = getGrip(mountStencil(), 'Name')
      expect(grip.attributes('aria-label')).toBe(
        'The first field names every card, and stays first',
      )
      expect(grip.attributes('tabindex')).toBe('-1')
      expect(grip.attributes('aria-disabled')).toBe('true')
      // A handle that goes nowhere along the order says no keys it answers.
      expect(grip.attributes('aria-keyshortcuts')).toBeUndefined()
    })

    it('emits a field dragged one place down the order', () => {
      const target = mountStencil()
      const press = pressKey(getGrip(target, 'Height').element, 'ArrowDown')
      expect(press.defaultPrevented).toBe(true)
      expect(target.emitted('move-field')).toEqual([['Height', null]])
    })

    it('emits a field dragged one place up the order', () => {
      const target = mountStencil()
      pressKey(getGrip(target, 'Weight').element, 'ArrowUp')
      expect(target.emitted('move-field')).toEqual([['Weight', 'Height']])
    })

    it('drags nothing above the first field, which names every card', () => {
      const target = mountStencil()
      pressKey(getGrip(target, 'Height').element, 'ArrowUp')
      expect(target.emitted('move-field')).toBeUndefined()
    })

    it('drags the first field nowhere', () => {
      const target = mountStencil()
      pressKey(getGrip(target, 'Name').element, 'ArrowDown')
      expect(target.emitted('move-field')).toBeUndefined()
    })
  })
})

describe('Stencil, what the caller found wrong', () => {
  const WRONG = {
    at: new Map([['recognise', ['this face has no back']]]),
    fields: new Map([['Height', ['declared twice']]]),
  }

  it('says what is wrong with a field under that field’s row', () => {
    const target = mountStencil({ wrong: WRONG })
    const response = getRow(target, 'Height').get('[data-wrong]')
    expect(response.text()).toBe('declared twice')
  })

  it('says it on no other row', () => {
    const target = mountStencil({ wrong: WRONG })
    expect(getRow(target, 'Weight').find('[data-wrong]').exists()).toBe(false)
  })

  it('says what is wrong with a face under that face’s name', () => {
    const target = mountStencil({ wrong: WRONG })
    const response = target.get('[data-face="recognise"] header [data-wrong]')
    expect(response.text()).toBe('this face has no back')
  })

  it('says a line for each of them', () => {
    const wrong = {
      at: new Map(),
      fields: new Map([['Height', ['declared twice', 'and nothing fills it']]]),
    }
    const target = mountStencil({ wrong })
    expect(getRow(target, 'Height').get('[data-wrong]').findAll('li')).toHaveLength(2)
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
    const target = mountStencil({ faces: [] })
    expect(target.findAll('[data-face]')).toHaveLength(0)
    expect(target.text()).toContain('No faces yet')
  })

  it('passes every face the fields the stencil declares', () => {
    const target = mountStencil({ faces: TWO })
    for (const id of ['one', 'two']) {
      const response = target
        .findAll(`[data-face="${id}"] [data-insert]`)
        .map((each) => each.attributes('data-insert'))
      expect(response).toEqual(FIELDS)
    }
  })

  it('passes a face the names the other faces carry', async () => {
    const target = mountStencil({ faces: TWO })
    const box = target.get<HTMLInputElement>('[data-face="one"] header input')
    box.element.value = 'Two'
    await box.trigger('input')
    expect(target.get('[data-face="one"] header [role="alert"]').text()).toBe('That name is taken')
  })

  it('emits the face and the half a box was typed into', async () => {
    const target = mountStencil()
    await target.get('[data-face="recognise"] [data-half="back"]').setValue('nothing but words')
    expect(target.emitted('write')).toEqual([['recognise', 'back', 'nothing but words']])
  })

  it('remains the way to add a face on a rule of the same make', () => {
    const target = mountStencil()
    const rules = target.findAll('.divider')
    expect(rules.map((rule) => rule.get('button').text())).toEqual(['Add a field', 'Add a face'])
    for (const rule of rules) expect(rule.attributes('role')).toBe('presentation')
  })

  it('asks for a face under a name nothing has taken', async () => {
    const target = mountStencil({ faces: [{ id: 'a', name: 'Face', front: '', back: '' }] })
    await target
      .findAll('button')
      .filter((each) => each.text() === 'Add a face')[0]
      ?.trigger('click')
    expect(target.emitted('add-face')).toEqual([['Face 1']])
  })

  it('emits the face requested to go', async () => {
    const target = mountStencil()
    await target.get('[data-face="recognise"] .card-header__actions button').trigger('click')
    expect(target.emitted('remove-face')).toEqual([['recognise']])
  })

  it('emits a face renamed', async () => {
    const target = mountStencil()
    const box = target.get<HTMLInputElement>('[data-face="recognise"] header input')
    box.element.value = 'Name it'
    await box.trigger('input')
    await box.trigger('change')
    expect(target.emitted('rename-face')).toEqual([['recognise', 'Name it']])
  })

  describe('dragging a face by the keyboard', () => {
    const THREE: readonly StencilFace[] = [
      { id: 'one', name: 'One', front: '', back: '' },
      { id: 'two', name: 'Two', front: '', back: '' },
      { id: 'three', name: 'Three', front: '', back: '' },
    ]

    const getStrip = (wrapper: Editor, id: string) =>
      wrapper.get(`[data-face="${id}"] .card-header__grip`)

    it('names the handle a face is dragged by, and gives it a place in the order', () => {
      const strip = getStrip(mountStencil({ faces: THREE }), 'two')
      expect(strip.attributes('aria-label')).toBe('Reorder: Two')
      expect(strip.attributes('tabindex')).toBe('0')
      expect(strip.attributes('role')).toBe('button')
    })

    it('emits a face dragged one place down the order', () => {
      const target = mountStencil({ faces: THREE })
      const press = pressKey(getStrip(target, 'one').element, 'ArrowDown')
      expect(press.defaultPrevented).toBe(true)
      expect(target.emitted('move-face')).toEqual([['one', 'three']])
    })

    it('emits a face dragged one place up the order: nothing among them is fixed', () => {
      const target = mountStencil({ faces: THREE })
      pressKey(getStrip(target, 'two').element, 'ArrowUp')
      expect(target.emitted('move-face')).toEqual([['two', 'one']])
    })

    it('does not move where there is no place that way', () => {
      const target = mountStencil({ faces: THREE })
      pressKey(getStrip(target, 'one').element, 'ArrowUp')
      expect(target.emitted('move-face')).toBeUndefined()
    })
  })

  describe('reordering the faces', () => {
    const THREE: readonly StencilFace[] = [
      { id: 'one', name: 'One', front: '', back: '' },
      { id: 'two', name: 'Two', front: '', back: '' },
      { id: 'three', name: 'Three', front: '', back: '' },
    ]

    const getFace = (editor: Editor, id: string) => editor.get(`[data-face="${id}"]`)

    /** A face picked up by its strip and let go over another, or over the tail. */
    const drag = async (editor: Editor, id: string, onto: string | null): Promise<void> => {
      await getFace(editor, id).get('.card-header').trigger('dragstart')
      const overElement = onto === null ? editor.findAll('section')[1] : getFace(editor, onto)
      await overElement?.trigger('dragover')
      await overElement?.trigger('drop')
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

    it('does not move where a face is let go at its current position', async () => {
      const editor = mountStencil({ faces: THREE })
      await drag(editor, 'two', 'two')
      expect(editor.emitted('move-face')).toBeUndefined()
    })

    /** A face picked up and the drag ended without it being let go anywhere. */
    const dragOff = async (editor: Editor, id: string, onto: string | null): Promise<void> => {
      const header = getFace(editor, id).get('.card-header')
      await header.trigger('dragstart')
      if (onto !== null) await getFace(editor, onto).trigger('dragover')
      await header.trigger('dragend')
    }

    it('does not move where a drag ends with the face let go nowhere', async () => {
      const editor = mountStencil({ faces: THREE })
      await dragOff(editor, 'two', null)
      expect(editor.emitted('move-face')).toBeUndefined()
      expect(getFace(editor, 'two').attributes('data-dragged')).toBeUndefined()
    })

    it('does not move where a drag over another face ends with it let go nowhere', async () => {
      const editor = mountStencil({ faces: THREE })
      await dragOff(editor, 'two', 'one')
      expect(editor.emitted('move-face')).toBeUndefined()
    })

    it('marks the face on its way, and no other', async () => {
      const editor = mountStencil({ faces: THREE })
      await getFace(editor, 'two').get('.card-header').trigger('dragstart')
      expect(getFace(editor, 'two').attributes('data-dragged')).toBe('true')
      expect(getFace(editor, 'one').attributes('data-dragged')).toBeUndefined()
    })

    it('moves no field when a face is dragged', async () => {
      const editor = mountStencil({ faces: THREE })
      await drag(editor, 'three', 'one')
      expect(editor.emitted('move-field')).toBeUndefined()
    })
  })
})
