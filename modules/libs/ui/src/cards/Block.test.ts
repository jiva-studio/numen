/**
 * What one face of a stencil draws from the block it was handed, and what it
 * emits.
 *
 * The negatives are here: a preview draws no braces, a part of the window
 * carries no outline of its own, a part something stands in says nothing about
 * being empty, one row of fields serves both halves, a name that objects
 * renames nothing, and a name abandoned renames nothing.
 */
import { mount } from '@vue/test-utils'
import { afterEach, describe, expect, it } from 'vitest'
import Block from './Block.vue'
import { declared, faceBlocks, type FaceBlock, type Filled, type Shown } from './model'
import { sampled } from './fill'

const FIELDS = ['Name', 'Height', 'Weight']

const FACE: Shown = {
  id: 'recognise',
  name: 'Recognise',
  front: '{{Name}}',
  back: '**Height:** {{Height}}',
}

/** One face, laid out against the fields a stencil declares. */
const blockOf = (
  face: Shown,
  fields: readonly string[] = FIELDS,
  sample?: readonly Filled[],
): FaceBlock => {
  const laid = faceBlocks([face], fields, sample ?? sampled(declared(fields)))[0]
  if (!laid) throw new Error('a corpus holding no face')
  return laid
}

const mountBlock = (block: FaceBlock = blockOf(FACE), props: Record<string, unknown> = {}) =>
  mount(Block, {
    attachTo: document.body,
    props: { block, fields: FIELDS, taken: [], ...props },
  })

type Drawn = ReturnType<typeof mountBlock>

const boxIn = (held: Drawn, half: string) =>
  held.get<HTMLTextAreaElement>(`[data-half="${half}"]`)

const nameOf = (held: Drawn) => held.get<HTMLInputElement>('header input')

/** A name typed into the name box and not yet committed. */
const type = async (held: Drawn, name: string): Promise<void> => {
  const box = nameOf(held)
  box.element.value = name
  await box.trigger('input')
}

afterEach(() => {
  document.body.innerHTML = ''
})

describe('Block, the window', () => {
  it('divides its window into four parts', () => {
    const held = mountBlock()
    expect(held.findAll('[data-pane]').map((pane) => pane.attributes('data-pane'))).toEqual([
      'front-written',
      'front-preview',
      'back-written',
      'back-preview',
    ])
  })

  it('draws the parts with nothing cut into an outline of their own', () => {
    const held = mountBlock()
    expect(held.findAll('h3')).toHaveLength(0)
    expect(held.findAll('.block__body fieldset')).toHaveLength(0)
    expect(held.findAll('.block__body legend')).toHaveLength(0)
    expect(held.findAll('.block__body label')).toHaveLength(0)
  })

  it('names each box it is written in, and names it to a reader alone', () => {
    const held = mountBlock()
    for (const half of ['front', 'back']) {
      expect(boxIn(held, half).attributes('aria-label')).toBe(half === 'front' ? 'Front' : 'Back')
    }
  })

  it('names the preview by the face and the half it is of, taking no box’s name', () => {
    const held = mountBlock()
    const preview = held.get('[data-preview="front"]')
    expect(preview.attributes('aria-label')).toBe('Preview: Recognise Front')
    // The preview is no control, so what names it names no box.
    expect(preview.attributes('role')).toBe('group')
    expect(preview.attributes('for')).toBeUndefined()
  })

  it('stands what an empty part is called in the part, and says it to nobody twice', () => {
    const held = mountBlock(blockOf({ id: 'one', name: 'One', front: '', back: '' }))
    const ghosts = held.findAll('.block__ghost')
    expect(ghosts.map((each) => each.text())).toEqual(['Front', 'Preview', 'Back', 'Preview'])
    for (const ghost of ghosts) expect(ghost.attributes('aria-hidden')).toBe('true')
    expect(held.findAll('[data-pane][data-blank]')).toHaveLength(4)
  })

  it('says nothing in a part something stands in', () => {
    const held = mountBlock()
    expect(held.find('[data-pane="front-written"] .block__ghost').exists()).toBe(false)
    expect(held.find('[data-pane="front-preview"] .block__ghost').exists()).toBe(false)
    expect(held.get('[data-pane="front-written"]').attributes('data-blank')).toBeUndefined()
  })

  it('says a part is empty where what is written in it fills out to nothing', () => {
    const block = blockOf({ id: 'one', name: 'One', front: '{{Height}}', back: 'said' }, FIELDS, [
      { field: 'Height', text: '' },
    ])
    const held = mountBlock(block)

    expect(held.get('[data-pane="front-written"]').attributes('data-blank')).toBeUndefined()
    expect(held.get('[data-pane="front-preview"]').attributes('data-blank')).toBe('true')
    expect(held.get('[data-pane="front-preview"] .block__ghost').text()).toBe('Preview')
  })

  it('shows the braces in the box it is written in', () => {
    expect(boxIn(mountBlock(), 'back').element.value).toContain('{{Height}}')
  })

  it('draws no braces in the preview, and no markup either', () => {
    const held = mountBlock(blockOf(FACE, FIELDS, [{ field: 'Height', text: 'about 45"' }]))
    const preview = held.get('[data-preview="back"]')
    expect(preview.text()).toContain('about 45"')
    expect(preview.text()).not.toContain('{{')
    expect(preview.text()).not.toContain('**')
    expect(preview.get('strong').text()).toBe('Height:')
  })

  it('emits a half as it now reads when it is typed into', async () => {
    const held = mountBlock()
    await boxIn(held, 'back').setValue('nothing but words')
    expect(held.emitted('write')).toEqual([['back', 'nothing but words']])
  })

  it('follows no link a preview draws: the window stays where it is', () => {
    const held = mountBlock(
      blockOf({ id: 'one', name: 'One', front: '[there](https://example.org)', back: '' }),
    )
    const press = new MouseEvent('click', { bubbles: true, cancelable: true })
    held.get('[data-preview="front"] a').element.dispatchEvent(press)
    expect(press.defaultPrevented).toBe(true)
  })

  it('draws no script it was written with', () => {
    const held = mountBlock(
      blockOf({ id: 'one', name: 'One', front: '<script>alert(1)</script>after', back: '' }),
    )
    const preview = held.get('[data-preview="front"]')
    expect(preview.find('script').exists()).toBe(false)
    expect(preview.html()).not.toContain('alert(1)')
    expect(preview.text()).toContain('after')
  })
})

describe('Block, the fields it is written with', () => {
  it('offers one row of fields for the whole face, and not one to each half', () => {
    const held = mountBlock()
    expect(held.findAll('.block__slots')).toHaveLength(1)
    expect(held.findAll('[data-pane] .block__slots')).toHaveLength(0)
  })

  it('offers a button per field, and none for a name of its own', () => {
    const held = mountBlock()
    expect(held.findAll('.block__slots button').map((each) => each.text())).toEqual(FIELDS)
  })

  it('stands the name and the fields on one strip, and no row of their own', () => {
    const held = mountBlock()
    const bar = held.get('.bar')
    expect(bar.attributes('data-grip')).toBeDefined()
    expect(bar.attributes('draggable')).toBe('true')
    expect(bar.find('input').exists()).toBe(true)
    expect(bar.findAll('[data-insert]')).toHaveLength(FIELDS.length)
    expect(held.findAll('.block__slots')).toHaveLength(1)
  })

  it('aims the row of fields at the part last typed in, and at one part only', async () => {
    const held = mountBlock()
    expect(held.findAll('[data-aimed]').map((each) => each.attributes('data-pane'))).toEqual([
      'front-written',
    ])

    await boxIn(held, 'back').trigger('focus')
    expect(held.findAll('[data-aimed]').map((each) => each.attributes('data-pane'))).toEqual([
      'back-written',
    ])
  })

  it('writes a field into the front while no half has been typed in', async () => {
    const held = mountBlock()
    await held.get('[data-insert="Weight"]').trigger('click')
    expect(held.emitted('write')).toEqual([['front', '{{Name}}{{Weight}}']])
  })

  /* A box nothing has been typed in reads a caret of zero, which is not a
     caret standing at the head of it. */
  it('writes a field after what a box nothing has been typed in holds', async () => {
    const held = mountBlock()
    boxIn(held, 'front').element.setSelectionRange(0, 0)

    await held.get('[data-insert="Weight"]').trigger('click')

    expect(held.emitted('write')).toEqual([['front', '{{Name}}{{Weight}}']])
  })

  it('writes a field where the caret stands in a box that has been typed in', async () => {
    const held = mountBlock()
    const written = boxIn(held, 'front')
    await written.trigger('focus')
    written.element.setSelectionRange(0, 0)

    await held.get('[data-insert="Weight"]').trigger('click')

    expect(held.emitted('write')).toEqual([['front', '{{Weight}}{{Name}}']])
  })

  it('writes a field into the half last typed in', async () => {
    const held = mountBlock()
    const back = boxIn(held, 'back')
    await back.trigger('focus')
    back.element.setSelectionRange(0, 0)

    await held.get('[data-insert="Weight"]').trigger('click')

    expect(held.emitted('write')).toEqual([['back', '{{Weight}}**Height:** {{Height}}']])
  })
})

describe('Block, what is wrong with it', () => {
  it('says a stray slot under the markup naming it, and nowhere else', () => {
    const held = mountBlock(blockOf({ id: 'one', name: 'One', front: '{{Name}}', back: '{{Colour}}' }))

    expect(held.get('[data-pane="back-written"] .block__objects').text()).toBe(
      'Not a field: Colour',
    )
    expect(held.find('[data-pane="back-preview"] .block__objects').exists()).toBe(false)
    expect(held.find('[data-pane="front-written"] .block__objects').exists()).toBe(false)
    expect(held.find('header .block__objects').exists()).toBe(false)
  })

  it('marks a stray slot in the preview where the slot itself stands', () => {
    const held = mountBlock(
      blockOf({ id: 'one', name: 'One', front: 'before {{Colour}} after', back: '' }),
    )
    const preview = held.get('[data-preview="front"]')
    expect(preview.get('mark').text()).toBe('{{Colour}}')
    expect(preview.text()).toContain('before')
    expect(preview.text()).toContain('after')
  })

  it('says nothing stray of a face naming only fields that are declared', () => {
    expect(mountBlock().find('.block__objects').exists()).toBe(false)
  })

  it('says what the caller found wrong under the name it is wrong about', () => {
    const held = mountBlock(undefined, { wrong: ['this face has no back'] })
    const said = held.get('header [data-wrong]')
    expect(said.text()).toBe('this face has no back')
    expect(said.attributes('aria-label')).toBe('What is wrong')
  })

  it('says a line for each of them', () => {
    const held = mountBlock(undefined, { wrong: ['no back', 'and nothing fills it'] })
    expect(held.get('[data-wrong]').findAll('li')).toHaveLength(2)
  })

  it('says nothing where the caller found nothing wrong', () => {
    expect(mountBlock().findAll('[data-wrong]')).toHaveLength(0)
  })
})

describe('Block, its name', () => {
  it('emits a name typed and committed', async () => {
    const held = mountBlock()
    await type(held, 'Name it')
    await nameOf(held).trigger('change')
    expect(held.emitted('rename')).toEqual([['Name it']])
  })

  it('drops the space around a name it commits', async () => {
    const held = mountBlock()
    await type(held, '  Recall  ')
    await nameOf(held).trigger('change')
    expect(held.emitted('rename')).toEqual([['Recall']])
  })

  it('renames nothing where the name typed is another face’s', async () => {
    const held = mountBlock(undefined, { taken: ['Recall'] })
    await type(held, 'Recall')
    await nameOf(held).trigger('change')
    expect(held.emitted('rename')).toBeUndefined()
  })

  it('renames nothing where the name typed has nothing in it', async () => {
    const held = mountBlock()
    await type(held, '   ')
    await nameOf(held).trigger('change')
    expect(held.emitted('rename')).toBeUndefined()
  })

  it('renames nothing where the name typed is the one it already carries', async () => {
    const held = mountBlock()
    await type(held, 'Recognise')
    await nameOf(held).trigger('change')
    expect(held.emitted('rename')).toBeUndefined()
  })

  it('renames nothing where a name is abandoned', async () => {
    const held = mountBlock()
    await type(held, 'Recall')
    await nameOf(held).trigger('keydown', { key: 'Escape' })
    await nameOf(held).trigger('change')
    expect(held.emitted('rename')).toBeUndefined()
    expect(nameOf(held).element.value).toBe('Recognise')
  })

  it('says why a name typed cannot be used, and says it to the box', async () => {
    const held = mountBlock(undefined, { taken: ['Recall'] })
    await type(held, 'Recall')

    const said = held.get('header .block__objects')
    expect(said.text()).toBe('That name is taken')
    expect(nameOf(held).attributes('aria-invalid')).toBe('true')
    expect(nameOf(held).attributes('aria-describedby')).toBe(said.attributes('id'))
  })

  it('says nothing about a name nothing is being typed over', () => {
    const held = mountBlock()
    expect(held.find('header .block__objects').exists()).toBe(false)
    expect(nameOf(held).attributes('aria-invalid')).toBeUndefined()
  })

  describe('a name holding a brace', () => {
    it('is a name like any other, a face standing in no brace', async () => {
      const held = mountBlock()
      await type(held, 'What {{Name}} is')
      await nameOf(held).trigger('change')
      expect(held.emitted('rename')).toEqual([['What {{Name}} is']])
    })

    it('is said nothing about while it is being typed', async () => {
      const held = mountBlock()
      await type(held, 'What {{Name}} is')
      expect(held.find('header .block__objects').exists()).toBe(false)
      expect(nameOf(held).attributes('aria-invalid')).toBeUndefined()
    })
  })
})

describe('Block, what it is asked', () => {
  it('emits when it is asked to go', async () => {
    const held = mountBlock()
    await held.get('.bar__deeds button').trigger('click')
    expect(held.emitted('remove')).toEqual([[]])
  })

  it('emits the carry its strip is taken up by, and the carry let go', async () => {
    const held = mountBlock()
    await held.get('.bar').trigger('dragstart')
    expect(held.emitted('lift')).toHaveLength(1)
    await held.get('.bar').trigger('dragend')
    expect(held.emitted('release')).toEqual([[]])
  })

  it('emits the way it is asked to go along the order', async () => {
    const held = mountBlock()
    await held.get('.bar').trigger('keydown', { key: 'ArrowDown' })
    expect(held.emitted('step')?.[0]?.[0]).toBe('down')
  })
})
