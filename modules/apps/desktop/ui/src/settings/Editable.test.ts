/**
 * A setting no control can say, opened in the editor.
 *
 * The negatives are the ones worth having: nothing is written until the editor
 * is kept on something that reads, and text that is not JSON5 leaves the file
 * as it was and says why.
 */
import { describe, expect, it } from 'vitest'
import { defineComponent, h } from 'vue'
import { mount } from '@vue/test-utils'

import Editable from './Editable.vue'
import { WORDS as words } from './words'

/** The editor, standing in for the one the window draws. */
const Editor = defineComponent({
  name: 'Editor',
  props: { modelValue: { type: String, default: '' } },
  emits: ['update:modelValue'],
  setup: (props, { emit }) =>
    () =>
      h('textarea', {
        value: props.modelValue,
        onInput: (event: Event) =>
          emit('update:modelValue', (event.target as HTMLTextAreaElement).value),
      }),
})

const draw = (value: unknown) =>
  mount(Editable, {
    props: { name: 'The command', detail: 'What is run', value },
    global: { stubs: { Editor } },
  })

type Row = ReturnType<typeof draw>

const pencil = (row: Row) => row.get('button')
const cancel = (row: Row) => row.findAll('button')[1]!
const keep = (row: Row) => row.findAll('button')[2]!
const typed = (row: Row) => row.get('textarea')

describe('the pencil', () => {
  it('opens on the value as it stands, written out', async () => {
    const row = draw(['claude', '--verbose'])
    expect(row.find('textarea').exists()).toBe(false)

    await pencil(row).trigger('click')

    expect((typed(row).element as HTMLTextAreaElement).value).toBe(
      '[\n  "claude",\n  "--verbose"\n]',
    )
  })

  it('opens on nothing where the file names the setting nowhere', async () => {
    const row = draw(undefined)
    await pencil(row).trigger('click')
    expect((typed(row).element as HTMLTextAreaElement).value).toBe('null')
  })

  it('shuts the editor again', async () => {
    const row = draw([])
    await pencil(row).trigger('click')
    await pencil(row).trigger('click')
    expect(row.find('textarea').exists()).toBe(false)
  })
})

describe('what was typed', () => {
  it('is handed on as what it reads, and the editor shuts', async () => {
    const row = draw([])
    await pencil(row).trigger('click')
    await typed(row).setValue("['claude', /* mine */ '--verbose',]")
    await keep(row).trigger('click')

    expect(row.emitted('keeps')).toStrictEqual([[['claude', '--verbose']]])
    expect(row.find('textarea').exists()).toBe(false)
  })

  it('is not handed on where it is not JSON5, and says why', async () => {
    const row = draw([])
    await pencil(row).trigger('click')
    await typed(row).setValue('[ "claude"')
    await keep(row).trigger('click')

    expect(row.emitted('keeps')).toBeUndefined()
    expect(row.text()).toContain(words.unreadable)
    expect(row.find('textarea').exists()).toBe(true)
  })

  it('is left where it stands where the editor is shut on it', async () => {
    const row = draw([])
    await pencil(row).trigger('click')
    await typed(row).setValue('["claude"]')
    await cancel(row).trigger('click')

    expect(row.emitted('keeps')).toBeUndefined()
    expect(row.find('textarea').exists()).toBe(false)
  })
})
