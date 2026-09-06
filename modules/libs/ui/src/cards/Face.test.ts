/**
 * What one face of a stencil draws from what it was handed, and what it emits.
 *
 * The negatives are here: a preview draws no braces, a part of the window
 * carries no outline of its own, a part something stands in says nothing about
 * being empty, one row of fields serves both halves, a name that objects
 * renames nothing, and a name abandoned renames nothing.
 */
import { mount } from '@vue/test-utils'
import { afterEach, describe, expect, it } from 'vitest'
import Face from './Face.vue'
import type { FieldValue } from './deck'
import { declared } from './order'
import { faceRows, type FaceRow, type StencilFace } from './stencil'
import { sampled } from './fill'

const FIELDS = ['Name', 'Height', 'Weight']

const FACE: StencilFace = {
  id: 'recognise',
  name: 'Recognise',
  front: '{{Name}}',
  back: '<b>Height:</b> {{Height}}',
}

/** One face, laid out against the fields a stencil declares. */
const faceOf = (
  face: StencilFace,
  fields: readonly string[] = FIELDS,
  sample?: readonly FieldValue[],
): FaceRow => {
  const laid = faceRows([face], fields, sample ?? sampled(declared(fields)))[0]
  if (!laid) throw new Error('a corpus holding no face')
  return laid
}

const mountFace = (face: FaceRow = faceOf(FACE), props: Record<string, unknown> = {}) =>
  mount(Face, {
    attachTo: document.body,
    props: { face, ...props },
  })

type Drawn = ReturnType<typeof mountFace>

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

describe('Face, the window', () => {
  it('divides its window into four parts', () => {
    const held = mountFace()
    expect(held.findAll('[data-pane]').map((pane) => pane.attributes('data-pane'))).toEqual([
      'front-written',
      'front-preview',
      'back-written',
      'back-preview',
    ])
  })

  it('draws the parts with nothing cut into an outline of their own', () => {
    const held = mountFace()
    expect(held.findAll('h3')).toHaveLength(0)
    expect(held.findAll('.face__body fieldset')).toHaveLength(0)
    expect(held.findAll('.face__body legend')).toHaveLength(0)
    expect(held.findAll('.face__body label')).toHaveLength(0)
  })

  it('names each box it is written in, and names it to a reader alone', () => {
    const held = mountFace()
    for (const half of ['front', 'back']) {
      expect(boxIn(held, half).attributes('aria-label')).toBe(half === 'front' ? 'Front' : 'Back')
    }
  })

  it('names the preview by the face and the half it is of, taking no box’s name', () => {
    const held = mountFace()
    const preview = held.get('[data-preview="front"]')
    expect(preview.attributes('aria-label')).toBe('Preview: Recognise Front')
    // The preview is no control, so what names it names no box.
    expect(preview.attributes('role')).toBe('group')
    expect(preview.attributes('for')).toBeUndefined()
  })

  it('stands what an empty part is called in the part, and says it to nobody twice', () => {
    const held = mountFace(faceOf({ id: 'one', name: 'One', front: '', back: '' }))
    const ghosts = held.findAll('.face__ghost')
    expect(ghosts.map((each) => each.text())).toEqual(['Front', 'Preview', 'Back', 'Preview'])
    for (const ghost of ghosts) expect(ghost.attributes('aria-hidden')).toBe('true')
    expect(held.findAll('[data-pane][data-blank]')).toHaveLength(4)
  })

  it('says nothing in a part something stands in', () => {
    const held = mountFace()
    expect(held.find('[data-pane="front-written"] .face__ghost').exists()).toBe(false)
    expect(held.find('[data-pane="front-preview"] .face__ghost').exists()).toBe(false)
    expect(held.get('[data-pane="front-written"]').attributes('data-blank')).toBeUndefined()
  })

  it('says a part is empty where what is written in it fills out to nothing', () => {
    const face = faceOf({ id: 'one', name: 'One', front: '{{Height}}', back: 'said' }, FIELDS, [
      { field: 'Height', text: '' },
    ])
    const held = mountFace(face)

    expect(held.get('[data-pane="front-written"]').attributes('data-blank')).toBeUndefined()
    expect(held.get('[data-pane="front-preview"]').attributes('data-blank')).toBe('true')
    expect(held.get('[data-pane="front-preview"] .face__ghost').text()).toBe('Preview')
  })

  it('shows the braces in the box it is written in', () => {
    expect(boxIn(mountFace(), 'back').element.value).toContain('{{Height}}')
  })

  it('draws no braces in the preview, and the tags around them as tags', () => {
    const held = mountFace(faceOf(FACE, FIELDS, [{ field: 'Height', text: 'about 45"' }]))
    const preview = held.get('[data-preview="back"]')
    expect(preview.text()).toContain('about 45"')
    expect(preview.text()).not.toContain('{{')
    expect(preview.text()).not.toContain('<b>')
    expect(preview.get('b').text()).toBe('Height:')
  })

  /* A face is HTML, so the marks of another format are characters a person
     typed and are drawn as themselves. */
  it('draws the marks a face carries as the text they are', () => {
    const held = mountFace(
      faceOf({ id: 'one', name: 'One', front: '**Name**: {{Name}}', back: '' }),
    )
    const preview = held.get('[data-preview="front"]')
    expect(preview.text()).toBe('**Name**: Name')
    expect(preview.find('strong').exists()).toBe(false)
  })

  it('emits a half as it now reads when it is typed into', async () => {
    const held = mountFace()
    await boxIn(held, 'back').setValue('nothing but words')
    expect(held.emitted('write')).toEqual([['back', 'nothing but words']])
  })

  it('follows no link a preview draws: the window stays where it is', () => {
    const held = mountFace(
      faceOf({ id: 'one', name: 'One', front: '<a href="https://example.org">there</a>', back: '' }),
    )
    const press = new MouseEvent('click', { bubbles: true, cancelable: true })
    held.get('[data-preview="front"] a').element.dispatchEvent(press)
    expect(press.defaultPrevented).toBe(true)
  })

  it('draws no script it was written with', () => {
    const held = mountFace(
      faceOf({ id: 'one', name: 'One', front: '<script>alert(1)</script>after', back: '' }),
    )
    const preview = held.get('[data-preview="front"]')
    expect(preview.find('script').exists()).toBe(false)
    expect(preview.html()).not.toContain('alert(1)')
    expect(preview.text()).toContain('after')
  })
})

describe('Face, the fields it is written with', () => {
  it('offers one row of fields for the whole face, and not one to each half', () => {
    const held = mountFace()
    expect(held.findAll('.face__slots')).toHaveLength(1)
    expect(held.findAll('[data-pane] .face__slots')).toHaveLength(0)
  })

  it('offers a button per field, and none for a name of its own', () => {
    const held = mountFace()
    expect(held.findAll('.face__slots button').map((each) => each.text())).toEqual(FIELDS)
  })

  it('stands the name and the fields on one strip, and no row of their own', () => {
    const held = mountFace()
    const header = held.get('.card-header')
    expect(header.attributes('data-grip')).toBeDefined()
    expect(header.attributes('draggable')).toBe('true')
    expect(header.find('input').exists()).toBe(true)
    expect(header.findAll('[data-insert]')).toHaveLength(FIELDS.length)
    expect(held.findAll('.face__slots')).toHaveLength(1)
  })

  it('aims the row of fields at the part last typed in, and at one part only', async () => {
    const held = mountFace()
    expect(held.findAll('[data-aimed]').map((each) => each.attributes('data-pane'))).toEqual([
      'front-written',
    ])

    await boxIn(held, 'back').trigger('focus')
    expect(held.findAll('[data-aimed]').map((each) => each.attributes('data-pane'))).toEqual([
      'back-written',
    ])
  })

  it('writes a field into the front while no half has been typed in', async () => {
    const held = mountFace()
    await held.get('[data-insert="Weight"]').trigger('click')
    expect(held.emitted('write')).toEqual([['front', '{{Name}}{{Weight}}']])
  })

  /* A box nothing has been typed in reads a caret of zero, which is not a
     caret standing at the head of it. */
  it('writes a field after what a box nothing has been typed in holds', async () => {
    const held = mountFace()
    boxIn(held, 'front').element.setSelectionRange(0, 0)

    await held.get('[data-insert="Weight"]').trigger('click')

    expect(held.emitted('write')).toEqual([['front', '{{Name}}{{Weight}}']])
  })

  it('writes a field where the caret stands in a box that has been typed in', async () => {
    const held = mountFace()
    const written = boxIn(held, 'front')
    await written.trigger('focus')
    written.element.setSelectionRange(0, 0)

    await held.get('[data-insert="Weight"]').trigger('click')

    expect(held.emitted('write')).toEqual([['front', '{{Weight}}{{Name}}']])
  })

  it('writes a field into the half last typed in', async () => {
    const held = mountFace()
    const back = boxIn(held, 'back')
    await back.trigger('focus')
    back.element.setSelectionRange(0, 0)

    await held.get('[data-insert="Weight"]').trigger('click')

    expect(held.emitted('write')).toEqual([['back', '{{Weight}}<b>Height:</b> {{Height}}']])
  })
})

describe('Face, what is wrong with it', () => {
  it('says a stray slot over the markup naming it, and nowhere else', () => {
    const held = mountFace(faceOf({ id: 'one', name: 'One', front: '{{Name}}', back: '{{Colour}}' }))

    expect(held.get('[data-pane="back-written"] .face__objects').text()).toBe(
      'Not a field: Colour',
    )
    expect(held.find('[data-pane="back-preview"] .face__objects').exists()).toBe(false)
    expect(held.find('[data-pane="front-written"] .face__objects').exists()).toBe(false)
    expect(held.find('header .face__objects').exists()).toBe(false)
  })

  it('marks a stray slot in the preview where the slot itself stands', () => {
    const held = mountFace(
      faceOf({ id: 'one', name: 'One', front: 'before {{Colour}} after', back: '' }),
    )
    const preview = held.get('[data-preview="front"]')
    expect(preview.get('mark').text()).toBe('{{Colour}}')
    expect(preview.text()).toContain('before')
    expect(preview.text()).toContain('after')
  })

  it('says nothing stray of a face naming only fields that are declared', () => {
    expect(mountFace().find('.face__objects').exists()).toBe(false)
  })

  it('says what the caller found wrong beside the name it is wrong about', () => {
    const held = mountFace(undefined, { wrong: ['this face has no back'] })
    const said = held.get('header [data-wrong]')
    expect(said.text()).toBe('this face has no back')
    expect(said.attributes('aria-label')).toBe('What is wrong')
  })

  it('stands what is wrong outside the rows it is wrong about', () => {
    const held = mountFace(faceOf({ id: 'one', name: 'One', front: '{{Colour}}', back: '' }), {
      wrong: ['this face has no back'],
    })
    expect(held.find('.face__head .face__objects').exists()).toBe(false)
    expect(held.get('header .face__amiss').findAll('.face__objects')).toHaveLength(1)

    const stray = held.get('[data-pane="front-written"] .face__amiss')
    expect(stray.findAll('.face__objects')).toHaveLength(1)
  })

  it('says a line for each of them', () => {
    const held = mountFace(undefined, { wrong: ['no back', 'and nothing fills it'] })
    expect(held.get('[data-wrong]').findAll('li')).toHaveLength(2)
  })

  it('says nothing where the caller found nothing wrong', () => {
    expect(mountFace().findAll('[data-wrong]')).toHaveLength(0)
  })
})

describe('Face, its name', () => {
  it('emits a name typed and committed', async () => {
    const held = mountFace()
    await type(held, 'Name it')
    await nameOf(held).trigger('change')
    expect(held.emitted('rename')).toEqual([['Name it']])
  })

  it('drops the space around a name it commits', async () => {
    const held = mountFace()
    await type(held, '  Recall  ')
    await nameOf(held).trigger('change')
    expect(held.emitted('rename')).toEqual([['Recall']])
  })

  it('renames nothing where the name typed is another face’s', async () => {
    const held = mountFace({ ...faceOf(FACE), taken: ['Recall'] })
    await type(held, 'Recall')
    await nameOf(held).trigger('change')
    expect(held.emitted('rename')).toBeUndefined()
  })

  it('renames nothing where the name typed has nothing in it', async () => {
    const held = mountFace()
    await type(held, '   ')
    await nameOf(held).trigger('change')
    expect(held.emitted('rename')).toBeUndefined()
  })

  it('renames nothing where the name typed is the one it already carries', async () => {
    const held = mountFace()
    await type(held, 'Recognise')
    await nameOf(held).trigger('change')
    expect(held.emitted('rename')).toBeUndefined()
  })

  it('renames nothing where a name is abandoned', async () => {
    const held = mountFace()
    await type(held, 'Recall')
    await nameOf(held).trigger('keydown', { key: 'Escape' })
    await nameOf(held).trigger('change')
    expect(held.emitted('rename')).toBeUndefined()
    expect(nameOf(held).element.value).toBe('Recognise')
  })

  it('says why a name typed cannot be used, and says it to the box', async () => {
    const held = mountFace({ ...faceOf(FACE), taken: ['Recall'] })
    await type(held, 'Recall')

    const said = held.get('header .face__objects')
    expect(said.text()).toBe('That name is taken')
    expect(nameOf(held).attributes('aria-invalid')).toBe('true')
    expect(nameOf(held).attributes('aria-describedby')).toBe(said.attributes('id'))
  })

  it('says nothing about a name nothing is being typed over', () => {
    const held = mountFace()
    expect(held.find('header .face__objects').exists()).toBe(false)
    expect(nameOf(held).attributes('aria-invalid')).toBeUndefined()
  })

  describe('a name holding a brace', () => {
    it('is a name like any other, a face standing in no brace', async () => {
      const held = mountFace()
      await type(held, 'What {{Name}} is')
      await nameOf(held).trigger('change')
      expect(held.emitted('rename')).toEqual([['What {{Name}} is']])
    })

    it('is said nothing about while it is being typed', async () => {
      const held = mountFace()
      await type(held, 'What {{Name}} is')
      expect(held.find('header .face__objects').exists()).toBe(false)
      expect(nameOf(held).attributes('aria-invalid')).toBeUndefined()
    })
  })
})

describe('Face, what it is asked', () => {
  it('emits when it is asked to go', async () => {
    const held = mountFace()
    await held.get('.card-header__actions button').trigger('click')
    expect(held.emitted('remove')).toEqual([[]])
  })

  it('emits the drag its strip is taken up by, and the drag let go', async () => {
    const held = mountFace()
    await held.get('.card-header').trigger('dragstart')
    expect(held.emitted('lift')).toHaveLength(1)
    await held.get('.card-header').trigger('dragend')
    expect(held.emitted('release')).toEqual([[]])
  })

  it('emits the way it is asked to go along the order', async () => {
    const held = mountFace()
    await held.get('.card-header__grip').trigger('keydown', { key: 'ArrowDown' })
    expect(held.emitted('step')?.[0]?.[0]).toBe('down')
  })
})
