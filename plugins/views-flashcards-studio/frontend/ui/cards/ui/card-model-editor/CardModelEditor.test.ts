/**
 * What the card model editor draws from the fields and templates it was handed, and
 * what it emits.
 *
 * What one template draws is that template's own, and is in `Template.test.ts`. Here are
 * the two orders, the two drags, and what each template is handed.
 *
 * The negatives are here: the first field has no handle and no way to go, a
 * name that objects does not rename, a name abandoned does not rename, and a
 * field let go where it was does not move.
 */
import { mount } from '@vue/test-utils'
import { afterEach, describe, expect, it } from 'vitest'
import CardModel from './CardModelEditor.vue'
import type { CardModelTemplate } from '../../lib/card-model'

const FIELDS = ['Name', 'Height', 'Weight']

const TEMPLATES: readonly CardModelTemplate[] = [
  {
    id: 'recognise',
    name: 'Recognise',
    front: '{{Name}}',
    back: '**Height:** {{Height}}',
  },
]

const mountCardModel = (props: Record<string, unknown> = {}) =>
  mount(CardModel, {
    attachTo: document.body,
    props: { fields: FIELDS, templates: TEMPLATES, ...props },
  })

type Editor = ReturnType<typeof mountCardModel>

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
    await wrapper.get('.card-model__fields').trigger('dragover')
    await wrapper.get('.card-model__fields').trigger('drop')
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
const dragOff = async (
  wrapper: Editor,
  field: string,
  targetField: string | null,
): Promise<void> => {
  const grip = getRow(wrapper, field).get('[data-grip]')
  await grip.trigger('dragstart')
  if (targetField !== null) await getRow(wrapper, targetField).trigger('dragover')
  await grip.trigger('dragend')
}

afterEach(() => {
  document.body.innerHTML = ''
})

describe('Card model, the fields', () => {
  it('draws a row per field, in the order they were handed in', () => {
    expect(getDrawnFields(mountCardModel())).toEqual(FIELDS)
  })

  it('draws the silence, and no rows, for a card model naming nothing', () => {
    const target = mountCardModel({ fields: [] })
    expect(target.findAll('[data-field]')).toHaveLength(0)
    expect(target.get('.card-model__silence').text()).toBe('No fields yet')
  })

  it('marks the first field as the one naming the cards', () => {
    const target = mountCardModel()
    expect(getRow(target, 'Name').attributes('data-names')).toBe('true')
    expect(getRow(target, 'Height').attributes('data-names')).toBeUndefined()
  })

  it('gives every row the same handle and the same way to remove it', () => {
    const target = mountCardModel()
    for (const field of FIELDS) {
      expect(getRow(target, field).findAll('[data-grip]')).toHaveLength(1)
      expect(getRow(target, field).findAll('button')).toHaveLength(1)
    }
  })

  it('turns off the first field’s handle, and leaves every other one on', () => {
    const target = mountCardModel()
    const first = getRow(target, 'Name').get('[data-grip]')
    expect(first.attributes('draggable')).toBe('false')
    expect(first.attributes('data-disabled')).toBe('true')

    const other = getRow(target, 'Height').get('[data-grip]')
    expect(other.attributes('draggable')).toBe('true')
    expect(other.attributes('data-disabled')).toBeUndefined()
  })

  it('turns off the first field’s way to be removed, and leaves every other one on', () => {
    const target = mountCardModel()
    expect(getRow(target, 'Name').get('button').attributes('disabled')).toBeDefined()
    expect(getRow(target, 'Height').get('button').attributes('disabled')).toBeUndefined()
  })

  it('redoes not move when the first field’s button is pressed', async () => {
    const target = mountCardModel()
    await getRow(target, 'Name').get('button').trigger('click')
    expect(target.emitted('remove-field')).toBeUndefined()
  })

  it('does not move where a field is let go above the first', async () => {
    const target = mountCardModel()
    await dragTo(target, 'Weight', 'Name')
    expect(target.emitted('move-field')).toBeUndefined()
  })

  it('renames the first field like any other', async () => {
    const target = mountCardModel()
    await type(target, 'Name', 'Question')
    await getBox(target, 'Name').trigger('change')
    expect(target.emitted('rename-field')).toEqual([['Name', 'Question']])
  })

  it('remains the way to add a field on a divider that is announced as nothing', () => {
    const target = mountCardModel()
    const divider = target.get('.card-model__part .divider')
    expect(divider.attributes('role')).toBe('presentation')
    expect(divider.get('button').text()).toBe('Add a field')
    expect(target.findAll('hr')).toHaveLength(0)
  })

  it('asks for a field under a name nothing has taken', async () => {
    const target = mountCardModel({ fields: ['Field', 'Field 2'] })
    await target
      .findAll('button')
      .filter((each) => each.text() === 'Add a field')[0]
      ?.trigger('click')
    expect(target.emitted('add-field')).toEqual([['Field 1']])
  })

  it('emits a name typed and committed', async () => {
    const target = mountCardModel()
    await type(target, 'Height', 'Tallness')
    await getBox(target, 'Height').trigger('change')
    expect(target.emitted('rename-field')).toEqual([['Height', 'Tallness']])
  })

  it('does not rename where the name typed is another field’s', async () => {
    const target = mountCardModel()
    await type(target, 'Height', 'Weight')
    await getBox(target, 'Height').trigger('change')
    expect(target.emitted('rename-field')).toBeUndefined()
  })

  it('says why a name typed cannot be used, while it is being typed', async () => {
    const target = mountCardModel()
    await type(target, 'Height', 'Weight')
    expect(getRow(target, 'Height').get('.card-model__objections').text()).toBe(
      'That name is taken',
    )

    // The line belongs to the row, and the box exposes the error to assistive
    // technology.
    expect(getRow(target, 'Height').get('.card-model__row').attributes('data-objections')).toBe(
      'taken',
    )
    expect(getBox(target, 'Height').attributes('aria-invalid')).toBe('true')
  })

  it('draws a row as one block: the handle, the box and the way to remove it stand inside it', () => {
    const target = mountCardModel()
    for (const field of FIELDS) {
      const row = getRow(target, field).get('.card-model__row')
      expect(row.find('[data-grip]').exists()).toBe(true)
      expect(row.find('input').exists()).toBe(true)
      expect(row.find('button').exists()).toBe(true)
    }
  })

  it('gives the turned-off row the same block, with nothing missing from it', () => {
    const target = mountCardModel()
    const first = getRow(target, 'Name').get('.card-model__row')
    const other = getRow(target, 'Height').get('.card-model__row')

    const shape = (row: typeof first): readonly string[] =>
      [...row.element.children].map((each) => each.tagName)
    expect(shape(first)).toEqual(shape(other))
  })

  it('says nothing about a name nothing is being typed over', () => {
    const target = mountCardModel()
    expect(target.find('.card-model__field .card-model__objections').exists()).toBe(false)
  })

  it('does not rename where the name typed is the one it already carries', async () => {
    const target = mountCardModel()
    await type(target, 'Height', 'Height')
    await getBox(target, 'Height').trigger('change')
    expect(target.emitted('rename-field')).toBeUndefined()
  })

  it('does not rename where a name is abandoned', async () => {
    const target = mountCardModel()
    await type(target, 'Height', 'Tallness')
    await getBox(target, 'Height').trigger('keydown', { key: 'Escape' })
    await getBox(target, 'Height').trigger('change')
    expect(target.emitted('rename-field')).toBeUndefined()
    expect(getBox(target, 'Height').element.value).toBe('Height')
  })

  it('emits the field requested to go', async () => {
    const target = mountCardModel()
    await getRow(target, 'Weight').findAll('button')[0]?.trigger('click')
    expect(target.emitted('remove-field')).toEqual([['Weight']])
  })

  it('emits a field let go before another below the first', async () => {
    const target = mountCardModel()
    await dragTo(target, 'Weight', 'Height')
    expect(target.emitted('move-field')).toEqual([['Weight', 'Height']])
  })

  it('emits a field let go past the last of them', async () => {
    const target = mountCardModel()
    await dragTo(target, 'Height', null)
    expect(target.emitted('move-field')).toEqual([['Height', null]])
  })

  it('emits a field let go on the rule the list is added by', async () => {
    const target = mountCardModel()
    await getRow(target, 'Height').get('[data-grip]').trigger('dragstart')
    const rule = target.findAll('section')[0]?.get('.divider')
    await rule?.trigger('dragover')
    await rule?.trigger('drop')
    expect(target.emitted('move-field')).toEqual([['Height', null]])
  })

  it('emits a field let go anywhere else in the part the fields stand in', async () => {
    const target = mountCardModel()
    await getRow(target, 'Height').get('[data-grip]').trigger('dragstart')
    const heading = target.get('.card-model__heading')
    await heading.trigger('dragover')
    await heading.trigger('drop')
    expect(target.emitted('move-field')).toEqual([['Height', null]])
  })

  it('does not move where a field is let go at its current position', async () => {
    const target = mountCardModel()
    await dragTo(target, 'Height', 'Height')
    expect(target.emitted('move-field')).toBeUndefined()
  })

  it('does not move where a drag ends with the field let go nowhere', async () => {
    const target = mountCardModel()
    await dragOff(target, 'Weight', null)
    expect(target.emitted('move-field')).toBeUndefined()
    expect(getRow(target, 'Weight').attributes('data-dragged')).toBeUndefined()
  })

  it('does not move where a drag over another field ends with it let go nowhere', async () => {
    const target = mountCardModel()
    await dragOff(target, 'Weight', 'Height')
    expect(target.emitted('move-field')).toBeUndefined()
  })

  it('marks the field on its way, and no other', async () => {
    const target = mountCardModel()
    await getRow(target, 'Weight').get('[data-grip]').trigger('dragstart')
    expect(getRow(target, 'Weight').attributes('data-dragged')).toBe('true')
    expect(getRow(target, 'Height').attributes('data-dragged')).toBeUndefined()
  })

  it('draws one row for a name the card model declares twice', () => {
    const target = mountCardModel({ fields: ['Name', 'Height', 'Height'] })
    expect(getDrawnFields(target)).toEqual(['Name', 'Height'])
    expect(target.findAll('.card-model__field input')).toHaveLength(2)
  })

  it('says what is wrong with a name to the box it is wrong about', async () => {
    const target = mountCardModel()
    await type(target, 'Height', 'Weight')

    const response = getRow(target, 'Height').get('.card-model__objections')
    expect(getBox(target, 'Height').attributes('aria-describedby')).toBe(response.attributes('id'))
    expect(response.attributes('id')).toBeTruthy()
  })

  it('describes a box by nothing while what is in it can be used', () => {
    expect(getBox(mountCardModel(), 'Height').attributes('aria-describedby')).toBeUndefined()
  })

  describe('dragging a field by the keyboard', () => {
    const getGrip = (wrapper: Editor, field: string) => getRow(wrapper, field).get('[data-grip]')

    it('names the handle, and gives it a place in the order', () => {
      const target = mountCardModel()
      const grip = getGrip(target, 'Height')
      expect(grip.attributes('aria-label')).toBe('Reorder: Height')
      expect(grip.attributes('tabindex')).toBe('0')
      expect(grip.attributes('role')).toBe('button')
      expect(grip.attributes('aria-keyshortcuts')).toBe('ArrowUp ArrowDown')
    })

    it('names the first field’s handle by what it is, and takes it out of the order', () => {
      const grip = getGrip(mountCardModel(), 'Name')
      expect(grip.attributes('aria-label')).toBe(
        'The first field names every card, and stays first',
      )
      expect(grip.attributes('tabindex')).toBe('-1')
      expect(grip.attributes('aria-disabled')).toBe('true')
      // A handle that cannot move along the order exposes no keyboard shortcuts.
      expect(grip.attributes('aria-keyshortcuts')).toBeUndefined()
    })

    it('emits a field dragged one place down the order', () => {
      const target = mountCardModel()
      const press = pressKey(getGrip(target, 'Height').element, 'ArrowDown')
      expect(press.defaultPrevented).toBe(true)
      expect(target.emitted('move-field')).toEqual([['Height', null]])
    })

    it('emits a field dragged one place up the order', () => {
      const target = mountCardModel()
      pressKey(getGrip(target, 'Weight').element, 'ArrowUp')
      expect(target.emitted('move-field')).toEqual([['Weight', 'Height']])
    })

    it('drags nothing above the first field, which names every card', () => {
      const target = mountCardModel()
      pressKey(getGrip(target, 'Height').element, 'ArrowUp')
      expect(target.emitted('move-field')).toBeUndefined()
    })

    it('drags the first field nowhere', () => {
      const target = mountCardModel()
      pressKey(getGrip(target, 'Name').element, 'ArrowDown')
      expect(target.emitted('move-field')).toBeUndefined()
    })
  })
})

describe('Card model, what the caller found wrong', () => {
  const WRONG = {
    at: new Map([['recognise', ['this template has no back']]]),
    fields: new Map([['Height', ['declared twice']]]),
  }

  it('says what is wrong with a field under that field’s row', () => {
    const target = mountCardModel({ wrong: WRONG })
    const response = getRow(target, 'Height').get('[data-wrong]')
    expect(response.text()).toBe('declared twice')
  })

  it('says it on no other row', () => {
    const target = mountCardModel({ wrong: WRONG })
    expect(getRow(target, 'Weight').find('[data-wrong]').exists()).toBe(false)
  })

  it('says what is wrong with a template under that template’s name', () => {
    const target = mountCardModel({ wrong: WRONG })
    const response = target.get('[data-template="recognise"] header [data-wrong]')
    expect(response.text()).toBe('this template has no back')
  })

  it('says a line for each of them', () => {
    const wrong = {
      at: new Map(),
      fields: new Map([['Height', ['declared twice', 'and nothing fills it']]]),
    }
    const target = mountCardModel({ wrong })
    expect(getRow(target, 'Height').get('[data-wrong]').findAll('li')).toHaveLength(2)
  })

  it('says nothing where the caller found nothing wrong', () => {
    expect(mountCardModel().findAll('[data-wrong]')).toHaveLength(0)
  })
})

describe('Card model, the templates', () => {
  const TWO: readonly CardModelTemplate[] = [
    { id: 'one', name: 'One', front: '', back: '' },
    { id: 'two', name: 'Two', front: '', back: '' },
  ]

  it('draws one of them per template', () => {
    expect(mountCardModel().findAll('[data-template]')).toHaveLength(1)
  })

  it('draws the silence, and no templates, for a card model showing nothing', () => {
    const target = mountCardModel({ templates: [] })
    expect(target.findAll('[data-template]')).toHaveLength(0)
    expect(target.text()).toContain('No templates yet')
  })

  it('passes every template the fields the card model declares', () => {
    const target = mountCardModel({ templates: TWO })
    for (const id of ['one', 'two']) {
      const response = target
        .findAll(`[data-template="${id}"] [data-insert]`)
        .map((each) => each.attributes('data-insert'))
      expect(response).toEqual(FIELDS)
    }
  })

  it('passes a template the names the other templates carry', async () => {
    const target = mountCardModel({ templates: TWO })
    const box = target.get<HTMLInputElement>('[data-template="one"] header input')
    box.element.value = 'Two'
    await box.trigger('input')
    expect(target.get('[data-template="one"] header [role="alert"]').text()).toBe(
      'That name is taken',
    )
  })

  it('emits the template and the half a box was typed into', async () => {
    const target = mountCardModel()
    await target.get('[data-template="recognise"] [data-half="back"]').setValue('nothing but words')
    expect(target.emitted('write')).toEqual([['recognise', 'back', 'nothing but words']])
  })

  it('remains the way to add a template on a rule of the same make', () => {
    const target = mountCardModel()
    const rules = target.findAll('.divider')
    expect(rules.map((rule) => rule.get('button').text())).toEqual([
      'Add a field',
      'Add a template',
    ])
    for (const rule of rules) expect(rule.attributes('role')).toBe('presentation')
  })

  it('asks for a template under a name nothing has taken', async () => {
    const target = mountCardModel({
      templates: [{ id: 'a', name: 'Template', front: '', back: '' }],
    })
    await target
      .findAll('button')
      .filter((each) => each.text() === 'Add a template')[0]
      ?.trigger('click')
    expect(target.emitted('add-template')).toEqual([['Template 1']])
  })

  it('emits the template requested to go', async () => {
    const target = mountCardModel()
    await target.get('[data-template="recognise"] .card-header__actions button').trigger('click')
    expect(target.emitted('remove-template')).toEqual([['recognise']])
  })

  it('emits a template renamed', async () => {
    const target = mountCardModel()
    const box = target.get<HTMLInputElement>('[data-template="recognise"] header input')
    box.element.value = 'Name it'
    await box.trigger('input')
    await box.trigger('change')
    expect(target.emitted('rename-template')).toEqual([['recognise', 'Name it']])
  })

  describe('dragging a template by the keyboard', () => {
    const THREE: readonly CardModelTemplate[] = [
      { id: 'one', name: 'One', front: '', back: '' },
      { id: 'two', name: 'Two', front: '', back: '' },
      { id: 'three', name: 'Three', front: '', back: '' },
    ]

    const getStrip = (wrapper: Editor, id: string) =>
      wrapper.get(`[data-template="${id}"] .card-header__grip`)

    it('names the handle a template is dragged by, and gives it a place in the order', () => {
      const strip = getStrip(mountCardModel({ templates: THREE }), 'two')
      expect(strip.attributes('aria-label')).toBe('Reorder: Two')
      expect(strip.attributes('tabindex')).toBe('0')
      expect(strip.attributes('role')).toBe('button')
    })

    it('emits a template dragged one place down the order', () => {
      const target = mountCardModel({ templates: THREE })
      const press = pressKey(getStrip(target, 'one').element, 'ArrowDown')
      expect(press.defaultPrevented).toBe(true)
      expect(target.emitted('move-template')).toEqual([['one', 'three']])
    })

    it('emits a template dragged one place up the order: nothing among them is fixed', () => {
      const target = mountCardModel({ templates: THREE })
      pressKey(getStrip(target, 'two').element, 'ArrowUp')
      expect(target.emitted('move-template')).toEqual([['two', 'one']])
    })

    it('does not move where there is no place that way', () => {
      const target = mountCardModel({ templates: THREE })
      pressKey(getStrip(target, 'one').element, 'ArrowUp')
      expect(target.emitted('move-template')).toBeUndefined()
    })
  })

  describe('reordering the templates', () => {
    const THREE: readonly CardModelTemplate[] = [
      { id: 'one', name: 'One', front: '', back: '' },
      { id: 'two', name: 'Two', front: '', back: '' },
      { id: 'three', name: 'Three', front: '', back: '' },
    ]

    const getTemplate = (editor: Editor, id: string) => editor.get(`[data-template="${id}"]`)

    /** A template picked up by its strip and let go over another, or over the tail. */
    const drag = async (editor: Editor, id: string, onto: string | null): Promise<void> => {
      await getTemplate(editor, id).get('.card-header').trigger('dragstart')
      const overElement = onto === null ? editor.findAll('section')[1] : getTemplate(editor, onto)
      await overElement?.trigger('dragover')
      await overElement?.trigger('drop')
    }

    it('emits a template let go before another', async () => {
      const editor = mountCardModel({ templates: THREE })
      await drag(editor, 'three', 'one')
      expect(editor.emitted('move-template')).toEqual([['three', 'one']])
    })

    it('emits a template let go past the last of them', async () => {
      const editor = mountCardModel({ templates: THREE })
      await drag(editor, 'one', null)
      expect(editor.emitted('move-template')).toEqual([['one', null]])
    })

    it('moves the first template like any other: nothing among the templates is fixed', async () => {
      const editor = mountCardModel({ templates: THREE })
      await drag(editor, 'one', 'three')
      expect(editor.emitted('move-template')).toEqual([['one', 'three']])
    })

    it('lands a template above the first, which no rule forbids', async () => {
      const editor = mountCardModel({ templates: THREE })
      await drag(editor, 'two', 'one')
      expect(editor.emitted('move-template')).toEqual([['two', 'one']])
    })

    it('does not move where a template is let go at its current position', async () => {
      const editor = mountCardModel({ templates: THREE })
      await drag(editor, 'two', 'two')
      expect(editor.emitted('move-template')).toBeUndefined()
    })

    /** A template picked up and the drag ended without it being let go anywhere. */
    const dragOff = async (editor: Editor, id: string, onto: string | null): Promise<void> => {
      const header = getTemplate(editor, id).get('.card-header')
      await header.trigger('dragstart')
      if (onto !== null) await getTemplate(editor, onto).trigger('dragover')
      await header.trigger('dragend')
    }

    it('does not move where a drag ends with the template let go nowhere', async () => {
      const editor = mountCardModel({ templates: THREE })
      await dragOff(editor, 'two', null)
      expect(editor.emitted('move-template')).toBeUndefined()
      expect(getTemplate(editor, 'two').attributes('data-dragged')).toBeUndefined()
    })

    it('does not move where a drag over another template ends with it let go nowhere', async () => {
      const editor = mountCardModel({ templates: THREE })
      await dragOff(editor, 'two', 'one')
      expect(editor.emitted('move-template')).toBeUndefined()
    })

    it('marks the template on its way, and no other', async () => {
      const editor = mountCardModel({ templates: THREE })
      await getTemplate(editor, 'two').get('.card-header').trigger('dragstart')
      expect(getTemplate(editor, 'two').attributes('data-dragged')).toBe('true')
      expect(getTemplate(editor, 'one').attributes('data-dragged')).toBeUndefined()
    })

    it('moves no field when a template is dragged', async () => {
      const editor = mountCardModel({ templates: THREE })
      await drag(editor, 'three', 'one')
      expect(editor.emitted('move-field')).toBeUndefined()
    })
  })
})
