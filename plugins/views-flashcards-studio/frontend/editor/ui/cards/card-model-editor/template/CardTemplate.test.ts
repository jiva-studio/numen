import { mount } from '@vue/test-utils'
import { afterEach, describe, expect, it } from 'vitest'
import CardTemplate from './CardTemplate.vue'
import type { FieldValue } from '../../../../../lib/cards'
import { getDeclaredFields } from '../../../../../lib/cards'
import { templateRows, type TemplateRow, type CardModelTemplate } from '../../../../../lib/cards'
import { sampleValues } from '../../../../../lib/cards'

const FIELDS = ['Name', 'Height', 'Weight']

const TEMPLATE: CardModelTemplate = {
  id: 'recognise',
  name: 'Recognise',
  front: '{{Name}}',
  back: '<b>Height:</b> {{Height}}',
}

const createTemplate = (
  template: CardModelTemplate,
  fields: readonly string[] = FIELDS,
  sample?: readonly FieldValue[],
): TemplateRow => {
  const arrangedRow = templateRows(
    [template],
    fields,
    sample ?? sampleValues(getDeclaredFields(fields)),
  )[0]
  if (!arrangedRow) throw new Error('missing template row')
  return arrangedRow
}

const mountTemplate = (
  template: TemplateRow = createTemplate(TEMPLATE),
  props: Record<string, unknown> = {},
) =>
  mount(CardTemplate, {
    attachTo: document.body,
    props: { template, ...props },
  })

type Drawn = ReturnType<typeof mountTemplate>

const getBox = (wrapper: Drawn, half: string) =>
  wrapper.get<HTMLTextAreaElement>(`[data-half="${half}"]`)

const getName = (wrapper: Drawn) => wrapper.get<HTMLInputElement>('header input')

const type = async (wrapper: Drawn, name: string): Promise<void> => {
  const box = getName(wrapper)
  box.element.value = name
  await box.trigger('input')
}

afterEach(() => {
  document.body.innerHTML = ''
})

describe('CardTemplate layout', () => {
  it('renders four panes', () => {
    const target = mountTemplate()
    expect(target.findAll('[data-pane]').map((pane) => pane.attributes('data-pane'))).toEqual([
      'front-written',
      'front-preview',
      'back-written',
      'back-preview',
    ])
  })

  it('renders panes without headings or fieldsets', () => {
    const target = mountTemplate()
    expect(target.findAll('h3')).toHaveLength(0)
    expect(target.findAll('.template__body fieldset')).toHaveLength(0)
    expect(target.findAll('.template__body legend')).toHaveLength(0)
    expect(target.findAll('.template__body label')).toHaveLength(0)
  })

  it('sets aria-label on front and back textareas', () => {
    const target = mountTemplate()
    for (const half of ['front', 'back']) {
      expect(getBox(target, half).attributes('aria-label')).toBe(
        half === 'front' ? 'Front' : 'Back',
      )
    }
  })

  it('sets aria-label and group role on the preview pane', () => {
    const target = mountTemplate()
    const preview = target.get('[data-preview="front"]')
    expect(preview.attributes('aria-label')).toBe('Preview: Recognise Front')
    expect(preview.attributes('role')).toBe('group')
    expect(preview.attributes('for')).toBeUndefined()
  })

  it('shows ghost labels in empty panes', () => {
    const target = mountTemplate(createTemplate({ id: 'one', name: 'One', front: '', back: '' }))
    const ghosts = target.findAll('.template__ghost')
    expect(ghosts.map((each) => each.text())).toEqual(['Front', 'Preview', 'Back', 'Preview'])
    for (const ghost of ghosts) expect(ghost.attributes('aria-hidden')).toBe('true')
    expect(target.findAll('[data-pane][data-blank]')).toHaveLength(4)
  })

  it('hides ghost labels in non-empty panes', () => {
    const target = mountTemplate()
    expect(target.find('[data-pane="front-written"] .template__ghost').exists()).toBe(false)
    expect(target.find('[data-pane="front-preview"] .template__ghost').exists()).toBe(false)
    expect(target.get('[data-pane="front-written"]').attributes('data-blank')).toBeUndefined()
  })

  it('marks preview pane as blank when rendered field value is empty', () => {
    const template = createTemplate(
      { id: 'one', name: 'One', front: '{{Height}}', back: 'said' },
      FIELDS,
      [{ field: 'Height', text: '' }],
    )
    const target = mountTemplate(template)

    expect(target.get('[data-pane="front-written"]').attributes('data-blank')).toBeUndefined()
    expect(target.get('[data-pane="front-preview"]').attributes('data-blank')).toBe('true')
    expect(target.get('[data-pane="front-preview"] .template__ghost').text()).toBe('Preview')
  })

  it('shows field braces in the editor textarea', () => {
    expect(getBox(mountTemplate(), 'back').element.value).toContain('{{Height}}')
  })

  it('renders field values and HTML tags in the preview', () => {
    const target = mountTemplate(
      createTemplate(TEMPLATE, FIELDS, [{ field: 'Height', text: 'about 45"' }]),
    )
    const preview = target.get('[data-preview="back"]')
    expect(preview.text()).toContain('about 45"')
    expect(preview.text()).not.toContain('{{')
    expect(preview.text()).not.toContain('<b>')
    expect(preview.get('b').text()).toBe('Height:')
  })

  it('renders markdown characters literally in the preview', () => {
    const target = mountTemplate(
      createTemplate({ id: 'one', name: 'One', front: '**Name**: {{Name}}', back: '' }),
    )
    const preview = target.get('[data-preview="front"]')
    expect(preview.text()).toBe('**Name**: Name')
    expect(preview.find('strong').exists()).toBe(false)
  })

  it('emits write when textarea content changes', async () => {
    const target = mountTemplate()
    await getBox(target, 'back').setValue('nothing but words')
    expect(target.emitted('write')).toEqual([['back', 'nothing but words']])
  })

  it('prevents navigation when clicking links in the preview', () => {
    const target = mountTemplate(
      createTemplate({
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

  it('strips script tags from the preview', () => {
    const target = mountTemplate(
      createTemplate({ id: 'one', name: 'One', front: '<script>alert(1)</script>after', back: '' }),
    )
    const preview = target.get('[data-preview="front"]')
    expect(preview.find('script').exists()).toBe(false)
    expect(preview.html()).not.toContain('alert(1)')
    expect(preview.text()).toContain('after')
  })
})

describe('CardTemplate field insertion', () => {
  it('renders a single row of field buttons in the header', () => {
    const target = mountTemplate()
    expect(target.findAll('.template__slots')).toHaveLength(1)
    expect(target.findAll('[data-pane] .template__slots')).toHaveLength(0)
  })

  it('renders one button for each field', () => {
    const target = mountTemplate()
    expect(target.findAll('.template__slots button').map((each) => each.text())).toEqual(FIELDS)
  })

  it('places the name input and field buttons inside the draggable header', () => {
    const target = mountTemplate()
    const header = target.get('.card-header')
    expect(header.attributes('data-grip')).toBeDefined()
    expect(header.attributes('draggable')).toBe('true')
    expect(header.find('input').exists()).toBe(true)
    expect(header.findAll('[data-insert]')).toHaveLength(FIELDS.length)
    expect(target.findAll('.template__slots')).toHaveLength(1)
  })

  it('tracks the focused editor pane', async () => {
    const target = mountTemplate()
    expect(target.findAll('[data-focused]').map((each) => each.attributes('data-pane'))).toEqual([
      'front-written',
    ])

    await getBox(target, 'back').trigger('focus')
    expect(target.findAll('[data-focused]').map((each) => each.attributes('data-pane'))).toEqual([
      'back-written',
    ])
  })

  it('inserts a field into the front pane by default', async () => {
    const target = mountTemplate()
    await target.get('[data-insert="Weight"]').trigger('click')
    expect(target.emitted('write')).toEqual([['front', '{{Name}}{{Weight}}']])
  })

  it('appends a field to the end when the textarea has not been focused', async () => {
    const target = mountTemplate()
    getBox(target, 'front').element.setSelectionRange(0, 0)

    await target.get('[data-insert="Weight"]').trigger('click')

    expect(target.emitted('write')).toEqual([['front', '{{Name}}{{Weight}}']])
  })

  it('inserts a field at the caret position when the textarea was focused', async () => {
    const target = mountTemplate()
    const content = getBox(target, 'front')
    await content.trigger('focus')
    content.element.setSelectionRange(0, 0)

    await target.get('[data-insert="Weight"]').trigger('click')

    expect(target.emitted('write')).toEqual([['front', '{{Weight}}{{Name}}']])
  })

  it('inserts a field into the last focused pane', async () => {
    const target = mountTemplate()
    const back = getBox(target, 'back')
    await back.trigger('focus')
    back.element.setSelectionRange(0, 0)

    await target.get('[data-insert="Weight"]').trigger('click')

    expect(target.emitted('write')).toEqual([['back', '{{Weight}}<b>Height:</b> {{Height}}']])
  })
})

describe('CardTemplate validation', () => {
  it('displays an unknown field warning in the editor pane', () => {
    const target = mountTemplate(
      createTemplate({ id: 'one', name: 'One', front: '{{Name}}', back: '{{Colour}}' }),
    )

    expect(target.get('[data-pane="back-written"] .template__objections').text()).toBe(
      'Not a field: Colour',
    )
    expect(target.find('[data-pane="back-preview"] .template__objections').exists()).toBe(false)
    expect(target.find('[data-pane="front-written"] .template__objections').exists()).toBe(false)
    expect(target.find('header .template__objections').exists()).toBe(false)
  })

  it('highlights an unknown field in the preview', () => {
    const target = mountTemplate(
      createTemplate({ id: 'one', name: 'One', front: 'before {{Colour}} after', back: '' }),
    )
    const preview = target.get('[data-preview="front"]')
    expect(preview.get('mark').text()).toBe('{{Colour}}')
    expect(preview.text()).toContain('before')
    expect(preview.text()).toContain('after')
  })

  it('shows no warning when all fields are declared', () => {
    expect(mountTemplate().find('.template__objections').exists()).toBe(false)
  })

  it('displays validation errors in the header', () => {
    const target = mountTemplate(undefined, { wrong: ['this template has no back'] })
    const response = target.get('header [data-wrong]')
    expect(response.text()).toBe('this template has no back')
    expect(response.attributes('aria-label')).toBe('What is wrong')
  })

  it('renders header and pane warnings in their own containers', () => {
    const target = mountTemplate(
      createTemplate({ id: 'one', name: 'One', front: '{{Colour}}', back: '' }),
      {
        wrong: ['this template has no back'],
      },
    )
    expect(target.find('.template__head .template__objections').exists()).toBe(false)
    expect(target.get('header .template__amiss').findAll('.template__objections')).toHaveLength(1)

    const stray = target.get('[data-pane="front-written"] .template__amiss')
    expect(stray.findAll('.template__objections')).toHaveLength(1)
  })

  it('renders a list item for each validation error', () => {
    const target = mountTemplate(undefined, { wrong: ['no back', 'and nothing fills it'] })
    expect(target.get('[data-wrong]').findAll('li')).toHaveLength(2)
  })

  it('omits the error list when there are no validation errors', () => {
    expect(mountTemplate().findAll('[data-wrong]')).toHaveLength(0)
  })
})

describe('CardTemplate rename', () => {
  it('emits rename when a new name is committed', async () => {
    const target = mountTemplate()
    await type(target, 'Name it')
    await getName(target).trigger('change')
    expect(target.emitted('rename')).toEqual([['Name it']])
  })

  it('trims whitespace from the committed name', async () => {
    const target = mountTemplate()
    await type(target, '  Recall  ')
    await getName(target).trigger('change')
    expect(target.emitted('rename')).toEqual([['Recall']])
  })

  it('does not emit rename when the name is already taken', async () => {
    const target = mountTemplate({ ...createTemplate(TEMPLATE), taken: ['Recall'] })
    await type(target, 'Recall')
    await getName(target).trigger('change')
    expect(target.emitted('rename')).toBeUndefined()
  })

  it('does not emit rename when the name is blank', async () => {
    const target = mountTemplate()
    await type(target, '   ')
    await getName(target).trigger('change')
    expect(target.emitted('rename')).toBeUndefined()
  })

  it('does not emit rename when the name is unchanged', async () => {
    const target = mountTemplate()
    await type(target, 'Recognise')
    await getName(target).trigger('change')
    expect(target.emitted('rename')).toBeUndefined()
  })

  it('reverts the name on Escape without emitting rename', async () => {
    const target = mountTemplate()
    await type(target, 'Recall')
    await getName(target).trigger('keydown', { key: 'Escape' })
    await getName(target).trigger('change')
    expect(target.emitted('rename')).toBeUndefined()
    expect(getName(target).element.value).toBe('Recognise')
  })

  it('shows an error message when the typed name is already taken', async () => {
    const target = mountTemplate({ ...createTemplate(TEMPLATE), taken: ['Recall'] })
    await type(target, 'Recall')

    const response = target.get('header .template__objections')
    expect(response.text()).toBe('That name is taken')
    expect(getName(target).attributes('aria-invalid')).toBe('true')
    expect(getName(target).attributes('aria-describedby')).toBe(response.attributes('id'))
  })

  it('shows no name error initially', () => {
    const target = mountTemplate()
    expect(target.find('header .template__objections').exists()).toBe(false)
    expect(getName(target).attributes('aria-invalid')).toBeUndefined()
  })

  describe('name with braces', () => {
    it('allows braces in the template name', async () => {
      const target = mountTemplate()
      await type(target, 'What {{Name}} is')
      await getName(target).trigger('change')
      expect(target.emitted('rename')).toEqual([['What {{Name}} is']])
    })

    it('shows no validation error when typing braces in the name', async () => {
      const target = mountTemplate()
      await type(target, 'What {{Name}} is')
      expect(target.find('header .template__objections').exists()).toBe(false)
      expect(getName(target).attributes('aria-invalid')).toBeUndefined()
    })
  })
})

describe('CardTemplate actions', () => {
  it('emits remove when the remove button is clicked', async () => {
    const target = mountTemplate()
    await target.get('.card-header__actions button').trigger('click')
    expect(target.emitted('remove')).toEqual([[]])
  })

  it('emits drag-start and release from the header', async () => {
    const target = mountTemplate()
    await target.get('.card-header').trigger('dragstart')
    expect(target.emitted('drag-start')).toHaveLength(1)
    await target.get('.card-header').trigger('dragend')
    expect(target.emitted('release')).toEqual([[]])
  })

  it('emits step on ArrowDown from the grip', async () => {
    const target = mountTemplate()
    await target.get('.card-header__grip').trigger('keydown', { key: 'ArrowDown' })
    expect(target.emitted('step')?.[0]?.[0]).toBe('down')
  })
})
