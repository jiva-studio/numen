/**
 * What the stencil editor draws from the fields and faces it was handed, and
 * what it emits.
 *
 * The negatives are here: the first field has no handle and no way to go, a
 * name that objects renames nothing, a name abandoned renames nothing, a field
 * let go where it was moves nothing, one row of fields serves both halves, a
 * preview draws no braces, a part of a face's window carries no outline of its
 * own, and a part something stands in says nothing about being empty.
 */
import { mount } from '@vue/test-utils'
import { afterEach, describe, expect, it } from 'vitest'
import Stencil from './Stencil.vue'
import type { Shown } from './model'

const FIELDS = ['Name', 'Height', 'Weight']

const FACES: readonly Shown[] = [
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

const drawnFields = (held: Editor) =>
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

/** A field picked up and the carry ended without it being let go anywhere. */
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
    expect(drawnFields(mountStencil())).toEqual(FIELDS)
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

  it('stands the way to add a field on a rule that is announced as nothing', () => {
    const held = mountStencil()
    const rule = held.get('.stencil__part .rule')
    expect(rule.attributes('role')).toBe('presentation')
    expect(rule.get('button').text()).toBe('Add a field')
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

    // The line belongs to the row, so what is wrong is said by the row and not
    // by the box standing inside it.
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

  it('moves nothing where a field is let go where it stands', async () => {
    const held = mountStencil()
    await dragTo(held, 'Height', 'Height')
    expect(held.emitted('move-field')).toBeUndefined()
  })

  it('moves nothing where a carry ends with the field let go nowhere', async () => {
    const held = mountStencil()
    await dragOff(held, 'Weight', null)
    expect(held.emitted('move-field')).toBeUndefined()
    expect(rowFor(held, 'Weight').attributes('data-carried')).toBeUndefined()
  })

  it('moves nothing where a carry over another field ends with it let go nowhere', async () => {
    const held = mountStencil()
    await dragOff(held, 'Weight', 'Height')
    expect(held.emitted('move-field')).toBeUndefined()
  })

  it('marks the field on its way, and no other', async () => {
    const held = mountStencil()
    await rowFor(held, 'Weight').get('[data-grip]').trigger('dragstart')
    expect(rowFor(held, 'Weight').attributes('data-carried')).toBe('true')
    expect(rowFor(held, 'Height').attributes('data-carried')).toBeUndefined()
  })
})

describe('Stencil, the faces', () => {
  it('draws a block per face, its window divided into four parts', () => {
    const held = mountStencil()
    expect(held.findAll('[data-face-block]')).toHaveLength(1)
    expect(held.findAll('[data-pane]').map((pane) => pane.attributes('data-pane'))).toEqual([
      'front-written',
      'front-preview',
      'back-written',
      'back-preview',
    ])
  })

  it('draws the silence, and no blocks, for a stencil showing nothing', () => {
    const held = mountStencil({ faces: [] })
    expect(held.findAll('[data-face-block]')).toHaveLength(0)
    expect(held.text()).toContain('No faces yet')
  })

  it('draws the parts with nothing cut into an outline of their own', () => {
    const block = mountStencil().get('[data-face-block="recognise"]')
    expect(block.findAll('h3')).toHaveLength(0)
    expect(block.findAll('.stencil__face-body fieldset')).toHaveLength(0)
    expect(block.findAll('.stencil__face-body legend')).toHaveLength(0)
    expect(block.findAll('.stencil__face-body label')).toHaveLength(0)
  })

  it('names each box it is written in, and names it to a reader alone', () => {
    const block = mountStencil().get('[data-face-block="recognise"]')
    for (const half of ['front', 'back']) {
      const said = half === 'front' ? 'Front' : 'Back'
      const written = block.get(`[data-face="recognise"][data-half="${half}"]`)
      expect(written.attributes('aria-label')).toBe(said)
    }
  })

  it('names the preview by the face and the half it is of, taking no box’s name', () => {
    const held = mountStencil()
    const preview = held.get('[data-preview="front"]')
    expect(preview.attributes('aria-label')).toBe('Preview: Recognise Front')
    // The preview is no control, so what names it names no box.
    expect(preview.attributes('role')).toBe('group')
    expect(preview.attributes('for')).toBeUndefined()
  })

  it('stands what an empty part is called in the part, and says it to nobody twice', () => {
    const held = mountStencil({ faces: [{ id: 'one', name: 'One', front: '', back: '' }] })
    const ghosts = held.findAll('.stencil__ghost')
    expect(ghosts.map((each) => each.text())).toEqual(['Front', 'Preview', 'Back', 'Preview'])
    for (const ghost of ghosts) expect(ghost.attributes('aria-hidden')).toBe('true')
    expect(held.findAll('[data-pane][data-blank]')).toHaveLength(4)
  })

  it('says nothing in a part something stands in', () => {
    const held = mountStencil()
    expect(held.find('[data-pane="front-written"] .stencil__ghost').exists()).toBe(false)
    expect(held.find('[data-pane="front-preview"] .stencil__ghost').exists()).toBe(false)
    expect(held.get('[data-pane="front-written"]').attributes('data-blank')).toBeUndefined()
  })

  it('says a part is empty where what is written in it fills out to nothing', () => {
    const faces: readonly Shown[] = [
      { id: 'one', name: 'One', front: '{{Height}}', back: 'said' },
    ]
    const held = mountStencil({ faces, sample: [{ field: 'Height', text: '' }] })

    expect(held.get('[data-pane="front-written"]').attributes('data-blank')).toBeUndefined()
    expect(held.get('[data-pane="front-preview"]').attributes('data-blank')).toBe('true')
    expect(held.get('[data-pane="front-preview"] .stencil__ghost').text()).toBe('Preview')
  })

  it('aims the row of fields at the part last typed in, and at one part only', async () => {
    const held = mountStencil()
    expect(held.findAll('[data-aimed]').map((each) => each.attributes('data-pane'))).toEqual([
      'front-written',
    ])

    await held.get('[data-face="recognise"][data-half="back"]').trigger('focus')
    expect(held.findAll('[data-aimed]').map((each) => each.attributes('data-pane'))).toEqual([
      'back-written',
    ])
  })

  it('shows the braces in the box a face is written in', () => {
    const held = mountStencil()
    const written = held.get<HTMLTextAreaElement>('[data-face="recognise"][data-half="back"]')
    expect(written.element.value).toContain('{{Height}}')
  })

  it('draws no braces in the preview, and no markup either', () => {
    const held = mountStencil({ sample: [{ field: 'Height', text: 'about 45"' }] })
    const preview = held.get('[data-preview="back"]')
    expect(preview.text()).toContain('about 45"')
    expect(preview.text()).not.toContain('{{')
    expect(preview.text()).not.toContain('**')
    expect(preview.get('strong').text()).toBe('Height:')
  })

  it('offers one row of fields for the whole face, and not one to each half', () => {
    const held = mountStencil()
    const block = held.get('[data-face-block="recognise"]')
    expect(block.findAll('.stencil__slots')).toHaveLength(1)
    expect(block.findAll('[data-pane] .stencil__slots')).toHaveLength(0)
  })

  it('offers a button per field, and none for a name of its own', () => {
    const held = mountStencil()
    const said = held
      .findAll('.stencil__slots button')
      .map((each) => each.text())
    expect(said).toEqual(['Name', 'Height', 'Weight'])
  })

  it('writes a field into the front while no half has been typed in', async () => {
    const held = mountStencil()

    await held.get('[data-insert="Weight"]').trigger('click')

    expect(held.emitted('write')).toEqual([['recognise', 'front', '{{Name}}{{Weight}}']])
  })

  /* A box nothing has been typed in reads a caret of zero, which is not a
     caret standing at the head of it. */
  it('writes a field after what a box nothing has been typed in holds', async () => {
    const held = mountStencil()
    const written = held.get('[data-face="recognise"][data-half="front"]')
    ;(written.element as HTMLTextAreaElement).setSelectionRange(0, 0)

    await held.get('[data-insert="Weight"]').trigger('click')

    expect(held.emitted('write')).toEqual([['recognise', 'front', '{{Name}}{{Weight}}']])
  })

  it('writes a field where the caret stands in a box that has been typed in', async () => {
    const held = mountStencil()
    const written = held.get('[data-face="recognise"][data-half="front"]')
    await written.trigger('focus')
    ;(written.element as HTMLTextAreaElement).setSelectionRange(0, 0)

    await held.get('[data-insert="Weight"]').trigger('click')

    expect(held.emitted('write')).toEqual([['recognise', 'front', '{{Weight}}{{Name}}']])
  })

  it('writes a field into the half last typed in', async () => {
    const held = mountStencil()
    const back = held.get('[data-face="recognise"][data-half="back"]')
    await back.trigger('focus')
    ;(back.element as HTMLTextAreaElement).setSelectionRange(0, 0)

    await held.get('[data-insert="Weight"]').trigger('click')

    expect(held.emitted('write')).toEqual([
      ['recognise', 'back', '{{Weight}}**Height:** {{Height}}'],
    ])
  })

  it('emits a half as it now reads when it is typed into', async () => {
    const held = mountStencil()
    const written = held.get('[data-face="recognise"][data-half="back"]')
    await written.setValue('nothing but words')
    expect(held.emitted('write')).toEqual([['recognise', 'back', 'nothing but words']])
  })

  it('says a stray slot under the markup naming it, and nowhere else', () => {
    const faces: readonly Shown[] = [
      { id: 'one', name: 'One', front: '{{Name}}', back: '{{Colour}}' },
    ]
    const held = mountStencil({ faces })
    const block = held.get('[data-face-block="one"]')

    expect(block.get('[data-pane="back-written"] .stencil__objects').text()).toBe(
      'Not a field: Colour',
    )
    expect(block.find('[data-pane="back-preview"] .stencil__objects').exists()).toBe(false)
    expect(block.find('[data-pane="front-written"] .stencil__objects').exists()).toBe(false)
    expect(block.find('header .stencil__objects').exists()).toBe(false)
  })

  it('marks a stray slot in the preview where the slot itself stands', () => {
    const faces: readonly Shown[] = [
      { id: 'one', name: 'One', front: 'before {{Colour}} after', back: '' },
    ]
    const held = mountStencil({ faces })
    const preview = held.get('[data-preview="front"]')
    expect(preview.get('mark').text()).toBe('{{Colour}}')
    expect(preview.text()).toContain('before')
    expect(preview.text()).toContain('after')
  })

  it('says nothing stray of a face naming only fields that are declared', () => {
    const held = mountStencil()
    expect(held.find('[data-face-block] .stencil__objects').exists()).toBe(false)
  })

  it('stands the way to add a face on a rule of the same make', () => {
    const held = mountStencil()
    const rules = held.findAll('.rule')
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
    await held.get('[data-face-block="recognise"] .bar__deeds button').trigger('click')
    expect(held.emitted('remove-face')).toEqual([['recognise']])
  })

  it('carries a face by its strip, which holds its name and the fields it is written with', () => {
    const bar = mountStencil().get('[data-face-block="recognise"] .bar')
    expect(bar.attributes('data-grip')).toBeDefined()
    expect(bar.attributes('draggable')).toBe('true')
    expect(bar.find('input').exists()).toBe(true)
    expect(bar.findAll('[data-insert]')).toHaveLength(FIELDS.length)
  })

  it('stands the name and the fields on one strip, and no row of their own', () => {
    const block = mountStencil().get('[data-face-block="recognise"]')
    expect(block.findAll('.stencil__slots')).toHaveLength(1)
    expect(block.get('.bar').findAll('.stencil__slots')).toHaveLength(1)
  })

  it('emits a face renamed', async () => {
    const held = mountStencil()
    const box = held.get<HTMLInputElement>('[data-face-block="recognise"] header input')
    box.element.value = 'Name it'
    await box.trigger('change')
    expect(held.emitted('rename-face')).toEqual([['recognise', 'Name it']])
  })

  describe('reordering the faces', () => {
    const THREE: readonly Shown[] = [
      { id: 'one', name: 'One', front: '', back: '' },
      { id: 'two', name: 'Two', front: '', back: '' },
      { id: 'three', name: 'Three', front: '', back: '' },
    ]

    const blockFor = (editor: Editor, id: string) => editor.get(`[data-face-block="${id}"]`)

    /** A face picked up by its strip and let go over another, or over the tail. */
    const carry = async (editor: Editor, id: string, onto: string | null): Promise<void> => {
      await blockFor(editor, id).get('.bar').trigger('dragstart')
      const over = onto === null ? editor.findAll('section')[1] : blockFor(editor, onto)
      await over?.trigger('dragover')
      await over?.trigger('drop')
    }

    it('emits a face let go before another', async () => {
      const editor = mountStencil({ faces: THREE })
      await carry(editor, 'three', 'one')
      expect(editor.emitted('move-face')).toEqual([['three', 'one']])
    })

    it('emits a face let go past the last of them', async () => {
      const editor = mountStencil({ faces: THREE })
      await carry(editor, 'one', null)
      expect(editor.emitted('move-face')).toEqual([['one', null]])
    })

    it('moves the first face like any other: nothing among the faces is fixed', async () => {
      const editor = mountStencil({ faces: THREE })
      await carry(editor, 'one', 'three')
      expect(editor.emitted('move-face')).toEqual([['one', 'three']])
    })

    it('lands a face above the first, which no rule forbids', async () => {
      const editor = mountStencil({ faces: THREE })
      await carry(editor, 'two', 'one')
      expect(editor.emitted('move-face')).toEqual([['two', 'one']])
    })

    it('moves nothing where a face is let go where it stands', async () => {
      const editor = mountStencil({ faces: THREE })
      await carry(editor, 'two', 'two')
      expect(editor.emitted('move-face')).toBeUndefined()
    })

    /** A face picked up and the carry ended without it being let go anywhere. */
    const carryOff = async (editor: Editor, id: string, onto: string | null): Promise<void> => {
      const bar = blockFor(editor, id).get('.bar')
      await bar.trigger('dragstart')
      if (onto !== null) await blockFor(editor, onto).trigger('dragover')
      await bar.trigger('dragend')
    }

    it('moves nothing where a carry ends with the face let go nowhere', async () => {
      const editor = mountStencil({ faces: THREE })
      await carryOff(editor, 'two', null)
      expect(editor.emitted('move-face')).toBeUndefined()
      expect(blockFor(editor, 'two').attributes('data-carried')).toBeUndefined()
    })

    it('moves nothing where a carry over another face ends with it let go nowhere', async () => {
      const editor = mountStencil({ faces: THREE })
      await carryOff(editor, 'two', 'one')
      expect(editor.emitted('move-face')).toBeUndefined()
    })

    it('marks the face on its way, and no other', async () => {
      const editor = mountStencil({ faces: THREE })
      await blockFor(editor, 'two').get('.bar').trigger('dragstart')
      expect(blockFor(editor, 'two').attributes('data-carried')).toBe('true')
      expect(blockFor(editor, 'one').attributes('data-carried')).toBeUndefined()
    })

    it('moves no field when a face is carried', async () => {
      const editor = mountStencil({ faces: THREE })
      await carry(editor, 'three', 'one')
      expect(editor.emitted('move-field')).toBeUndefined()
    })
  })

  it('draws no script a face was written with', () => {
    const faces: readonly Shown[] = [
      { id: 'one', name: 'One', front: '<script>alert(1)</script>after', back: '' },
    ]
    const held = mountStencil({ faces })
    const preview = held.get('[data-preview="front"]')
    expect(preview.find('script').exists()).toBe(false)
    expect(preview.html()).not.toContain('alert(1)')
    expect(preview.text()).toContain('after')
  })
})
