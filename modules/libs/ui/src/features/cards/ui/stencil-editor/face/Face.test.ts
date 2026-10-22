/**
 * What one face of a stencil draws from what it was handed, and what it emits.
 *
 * The negatives are here: a preview draws no braces, a part of the window
 * being empty, one row of fields serves both halves, a name that objects
 * does not rename, and a name abandoned does not rename.
 */
import { mount } from '@vue/test-utils'
import { afterEach, describe, expect, it } from 'vitest'
import Face from './Face.vue'
import type { FieldValue } from '../../../lib/card'
import { getDeclaredFields } from '../../../lib/order'
import { faceRows, type FaceRow, type StencilFace } from '../../../lib/stencil'
import { sampleValues } from '../../../lib/fill'

const FIELDS = ['Name', 'Height', 'Weight']

const FACE: StencilFace = {
  id: 'recognise',
  name: 'Recognise',
  front: '{{Name}}',
  back: '<b>Height:</b> {{Height}}',
}

/** One face, laid out against the fields a stencil declares. */
const createFace = (
  face: StencilFace,
  fields: readonly string[] = FIELDS,
  sample?: readonly FieldValue[],
): FaceRow => {
  const arrangedRow = faceRows([face], fields, sample ?? sampleValues(getDeclaredFields(fields)))[0]
  if (!arrangedRow) throw new Error('a corpus holding no face')
  return arrangedRow
}

const mountFace = (face: FaceRow = createFace(FACE), props: Record<string, unknown> = {}) =>
  mount(Face, {
    attachTo: document.body,
    props: { face, ...props },
  })

type Drawn = ReturnType<typeof mountFace>

const getBox = (wrapper: Drawn, half: string) =>
  wrapper.get<HTMLTextAreaElement>(`[data-half="${half}"]`)

const getName = (wrapper: Drawn) => wrapper.get<HTMLInputElement>('header input')

/** A name typed into the name box and not yet committed. */
const type = async (wrapper: Drawn, name: string): Promise<void> => {
  const box = getName(wrapper)
  box.element.value = name
  await box.trigger('input')
}

afterEach(() => {
  document.body.innerHTML = ''
})

describe('Face, the window', () => {
  it('divides its window into four parts', () => {
    const target = mountFace()
    expect(target.findAll('[data-pane]').map((pane) => pane.attributes('data-pane'))).toEqual([
      'front-written',
      'front-preview',
      'back-written',
      'back-preview',
    ])
  })

  it('draws the parts with nothing cut into an outline of their own', () => {
    const target = mountFace()
    expect(target.findAll('h3')).toHaveLength(0)
    expect(target.findAll('.face__body fieldset')).toHaveLength(0)
    expect(target.findAll('.face__body legend')).toHaveLength(0)
    expect(target.findAll('.face__body label')).toHaveLength(0)
  })

  it('names each box it is written in, and names it to a reader alone', () => {
    const target = mountFace()
    for (const half of ['front', 'back']) {
      expect(getBox(target, half).attributes('aria-label')).toBe(half === 'front' ? 'Front' : 'Back')
    }
  })

  it('names the preview by the face and the half it is of, taking no box’s name', () => {
    const target = mountFace()
    const preview = target.get('[data-preview="front"]')
    expect(preview.attributes('aria-label')).toBe('Preview: Recognise Front')
    // The preview is no control, so what names it names no box.
    expect(preview.attributes('role')).toBe('group')
    expect(preview.attributes('for')).toBeUndefined()
  })

  it('remains what an empty part is called in the part, and says it to nobody twice', () => {
    const target = mountFace(createFace({ id: 'one', name: 'One', front: '', back: '' }))
    const ghosts = target.findAll('.face__ghost')
    expect(ghosts.map((each) => each.text())).toEqual(['Front', 'Preview', 'Back', 'Preview'])
    for (const ghost of ghosts) expect(ghost.attributes('aria-hidden')).toBe('true')
    expect(target.findAll('[data-pane][data-blank]')).toHaveLength(4)
  })

  it('says nothing in a part something remains in', () => {
    const target = mountFace()
    expect(target.find('[data-pane="front-written"] .face__ghost').exists()).toBe(false)
    expect(target.find('[data-pane="front-preview"] .face__ghost').exists()).toBe(false)
    expect(target.get('[data-pane="front-written"]').attributes('data-blank')).toBeUndefined()
  })

  it('says a part is empty where what is written in it fills out to nothing', () => {
    const face = createFace({ id: 'one', name: 'One', front: '{{Height}}', back: 'said' }, FIELDS, [
      { field: 'Height', text: '' },
    ])
    const target = mountFace(face)

    expect(target.get('[data-pane="front-written"]').attributes('data-blank')).toBeUndefined()
    expect(target.get('[data-pane="front-preview"]').attributes('data-blank')).toBe('true')
    expect(target.get('[data-pane="front-preview"] .face__ghost').text()).toBe('Preview')
  })

  it('shows the braces in the box it is written in', () => {
    expect(getBox(mountFace(), 'back').element.value).toContain('{{Height}}')
  })

  it('draws no braces in the preview, and the tags around them as tags', () => {
    const target = mountFace(createFace(FACE, FIELDS, [{ field: 'Height', text: 'about 45"' }]))
    const preview = target.get('[data-preview="back"]')
    expect(preview.text()).toContain('about 45"')
    expect(preview.text()).not.toContain('{{')
    expect(preview.text()).not.toContain('<b>')
    expect(preview.get('b').text()).toBe('Height:')
  })

  /* A face is HTML, so the marks of another format are characters a person
     typed and are drawn as themselves. */
  it('draws the marks a face carries as the text they are', () => {
    const target = mountFace(
      createFace({ id: 'one', name: 'One', front: '**Name**: {{Name}}', back: '' }),
    )
    const preview = target.get('[data-preview="front"]')
    expect(preview.text()).toBe('**Name**: Name')
    expect(preview.find('strong').exists()).toBe(false)
  })

  it('emits a half as it now reads when it is typed into', async () => {
    const target = mountFace()
    await getBox(target, 'back').setValue('nothing but words')
    expect(target.emitted('write')).toEqual([['back', 'nothing but words']])
  })

  it('follows no link a preview draws: the window stays where it is', () => {
    const target = mountFace(
      createFace({
        id: 'one',
        name: 'One',
        front: '<a href="https://example.org">there</a>',
        back: '',
      }),
    )
    const press = new MouseEvent('click', { bubbles: true, cancelable: true })
    target.get('[data-preview="front"] a').element.dispatchEvent(press)
    expect(press.defaultPrevented).toBe(true)
  })

  it('draws no script it was written with', () => {
    const target = mountFace(
      createFace({ id: 'one', name: 'One', front: '<script>alert(1)</script>after', back: '' }),
    )
    const preview = target.get('[data-preview="front"]')
    expect(preview.find('script').exists()).toBe(false)
    expect(preview.html()).not.toContain('alert(1)')
    expect(preview.text()).toContain('after')
  })
})

describe('Face, the fields it is written with', () => {
  it('offers one row of fields for the whole face, and not one to each half', () => {
    const target = mountFace()
    expect(target.findAll('.face__slots')).toHaveLength(1)
    expect(target.findAll('[data-pane] .face__slots')).toHaveLength(0)
  })

  it('offers a button per field, and none for a name of its own', () => {
    const target = mountFace()
    expect(target.findAll('.face__slots button').map((each) => each.text())).toEqual(FIELDS)
  })

  it('remains the name and the fields on one strip, and no row of their own', () => {
    const target = mountFace()
    const header = target.get('.card-header')
    expect(header.attributes('data-grip')).toBeDefined()
    expect(header.attributes('draggable')).toBe('true')
    expect(header.find('input').exists()).toBe(true)
    expect(header.findAll('[data-insert]')).toHaveLength(FIELDS.length)
    expect(target.findAll('.face__slots')).toHaveLength(1)
  })

  it('aims the row of fields at the part last typed in, and at one part only', async () => {
    const target = mountFace()
    expect(target.findAll('[data-focused]').map((each) => each.attributes('data-pane'))).toEqual([
      'front-written',
    ])

    await getBox(target, 'back').trigger('focus')
    expect(target.findAll('[data-focused]').map((each) => each.attributes('data-pane'))).toEqual([
      'back-written',
    ])
  })

  it('writes a field into the front while no half has been typed in', async () => {
    const target = mountFace()
    await target.get('[data-insert="Weight"]').trigger('click')
    expect(target.emitted('write')).toEqual([['front', '{{Name}}{{Weight}}']])
  })

  /* A box nothing has been typed in reads a caret of zero, which is not a
     caret standing at the head of it. */
  it('writes a field after what a box nothing has been typed in holds', async () => {
    const target = mountFace()
    getBox(target, 'front').element.setSelectionRange(0, 0)

    await target.get('[data-insert="Weight"]').trigger('click')

    expect(target.emitted('write')).toEqual([['front', '{{Name}}{{Weight}}']])
  })

  it('writes a field where the caret remains in a box that has been typed in', async () => {
    const target = mountFace()
    const content = getBox(target, 'front')
    await content.trigger('focus')
    content.element.setSelectionRange(0, 0)

    await target.get('[data-insert="Weight"]').trigger('click')

    expect(target.emitted('write')).toEqual([['front', '{{Weight}}{{Name}}']])
  })

  it('writes a field into the half last typed in', async () => {
    const target = mountFace()
    const back = getBox(target, 'back')
    await back.trigger('focus')
    back.element.setSelectionRange(0, 0)

    await target.get('[data-insert="Weight"]').trigger('click')

    expect(target.emitted('write')).toEqual([['back', '{{Weight}}<b>Height:</b> {{Height}}']])
  })
})

describe('Face, what is wrong with it', () => {
  it('says a stray slot over the markup naming it, and nowhere else', () => {
    const target = mountFace(
      createFace({ id: 'one', name: 'One', front: '{{Name}}', back: '{{Colour}}' }),
    )

    expect(target.get('[data-pane="back-written"] .face__objections').text()).toBe(
      'Not a field: Colour',
    )
    expect(target.find('[data-pane="back-preview"] .face__objections').exists()).toBe(false)
    expect(target.find('[data-pane="front-written"] .face__objections').exists()).toBe(false)
    expect(target.find('header .face__objections').exists()).toBe(false)
  })

  it('marks a stray slot in the preview where the slot itself remains', () => {
    const target = mountFace(
      createFace({ id: 'one', name: 'One', front: 'before {{Colour}} after', back: '' }),
    )
    const preview = target.get('[data-preview="front"]')
    expect(preview.get('mark').text()).toBe('{{Colour}}')
    expect(preview.text()).toContain('before')
    expect(preview.text()).toContain('after')
  })

  it('says nothing stray of a face naming only fields that are declared', () => {
    expect(mountFace().find('.face__objections').exists()).toBe(false)
  })

  it('says what the caller found wrong alongside the name it is wrong about', () => {
    const target = mountFace(undefined, { wrong: ['this face has no back'] })
    const response = target.get('header [data-wrong]')
    expect(response.text()).toBe('this face has no back')
    expect(response.attributes('aria-label')).toBe('What is wrong')
  })

  it('remains what is wrong outside the rows it is wrong about', () => {
    const target = mountFace(createFace({ id: 'one', name: 'One', front: '{{Colour}}', back: '' }), {
      wrong: ['this face has no back'],
    })
    expect(target.find('.face__head .face__objections').exists()).toBe(false)
    expect(target.get('header .face__amiss').findAll('.face__objections')).toHaveLength(1)

    const stray = target.get('[data-pane="front-written"] .face__amiss')
    expect(stray.findAll('.face__objections')).toHaveLength(1)
  })

  it('says a line for each of them', () => {
    const target = mountFace(undefined, { wrong: ['no back', 'and nothing fills it'] })
    expect(target.get('[data-wrong]').findAll('li')).toHaveLength(2)
  })

  it('says nothing where the caller found nothing wrong', () => {
    expect(mountFace().findAll('[data-wrong]')).toHaveLength(0)
  })
})

describe('Face, its name', () => {
  it('emits a name typed and committed', async () => {
    const target = mountFace()
    await type(target, 'Name it')
    await getName(target).trigger('change')
    expect(target.emitted('rename')).toEqual([['Name it']])
  })

  it('drops the space around a name it commits', async () => {
    const target = mountFace()
    await type(target, '  Recall  ')
    await getName(target).trigger('change')
    expect(target.emitted('rename')).toEqual([['Recall']])
  })

  it('does not rename where the name typed is another face’s', async () => {
    const target = mountFace({ ...createFace(FACE), taken: ['Recall'] })
    await type(target, 'Recall')
    await getName(target).trigger('change')
    expect(target.emitted('rename')).toBeUndefined()
  })

  it('does not rename where the name typed has nothing in it', async () => {
    const target = mountFace()
    await type(target, '   ')
    await getName(target).trigger('change')
    expect(target.emitted('rename')).toBeUndefined()
  })

  it('does not rename where the name typed is the one it already carries', async () => {
    const target = mountFace()
    await type(target, 'Recognise')
    await getName(target).trigger('change')
    expect(target.emitted('rename')).toBeUndefined()
  })

  it('does not rename where a name is abandoned', async () => {
    const target = mountFace()
    await type(target, 'Recall')
    await getName(target).trigger('keydown', { key: 'Escape' })
    await getName(target).trigger('change')
    expect(target.emitted('rename')).toBeUndefined()
    expect(getName(target).element.value).toBe('Recognise')
  })

  it('says why a name typed cannot be used, and says it to the box', async () => {
    const target = mountFace({ ...createFace(FACE), taken: ['Recall'] })
    await type(target, 'Recall')

    const response = target.get('header .face__objections')
    expect(response.text()).toBe('That name is taken')
    expect(getName(target).attributes('aria-invalid')).toBe('true')
    expect(getName(target).attributes('aria-describedby')).toBe(response.attributes('id'))
  })

  it('says nothing about a name nothing is being typed over', () => {
    const target = mountFace()
    expect(target.find('header .face__objections').exists()).toBe(false)
    expect(getName(target).attributes('aria-invalid')).toBeUndefined()
  })

  describe('a name holding a brace', () => {
    it('is a name like any other, a face standing in no brace', async () => {
      const target = mountFace()
      await type(target, 'What {{Name}} is')
      await getName(target).trigger('change')
      expect(target.emitted('rename')).toEqual([['What {{Name}} is']])
    })

    it('is returned nothing about while it is being typed', async () => {
      const target = mountFace()
      await type(target, 'What {{Name}} is')
      expect(target.find('header .face__objections').exists()).toBe(false)
      expect(getName(target).attributes('aria-invalid')).toBeUndefined()
    })
  })
})

describe('Face, what it is requested', () => {
  it('emits when it is requested to go', async () => {
    const target = mountFace()
    await target.get('.card-header__actions button').trigger('click')
    expect(target.emitted('remove')).toEqual([[]])
  })

  it('emits the drag its strip is taken up by, and the drag let go', async () => {
    const target = mountFace()
    await target.get('.card-header').trigger('dragstart')
    expect(target.emitted('lift')).toHaveLength(1)
    await target.get('.card-header').trigger('dragend')
    expect(target.emitted('release')).toEqual([[]])
  })

  it('emits the way it is requested to go along the order', async () => {
    const target = mountFace()
    await target.get('.card-header__grip').trigger('keydown', { key: 'ArrowDown' })
    expect(target.emitted('step')?.[0]?.[0]).toBe('down')
  })
})
