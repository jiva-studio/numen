/**
 * The settings file drawn: the editor, and nothing standing over it.
 *
 * The tab already carries the file's name, and the text is kept the way a note
 * is kept, so there is no bar above it saying either.
 */
import { describe, expect, it } from 'vitest'
import { Code, ConnectError } from '@connectrpc/connect'
import { mount } from '@vue/test-utils'
import TextEditorTab from './TextEditorTab.vue'
import { useTextEditor, type TextEditorTabDeps } from '../composables/useTextEditor'
import { WORDS as words } from '../words'

const HELD = '{\n  "agent": { "use": "claude" }\n}\n'

const drawn = async (answers: Partial<TextEditorTabDeps> = {}) => {
  const wrote: string[] = []
  const core: TextEditorTabDeps = {
    getSettingsFile: () => Promise.resolve({ written: HELD, path: '/numen.json' }),
    saveSettingsFile: (written) => {
      wrote.push(written)
      return Promise.resolve({ changed: false })
    },
    ...answers,
  }
  const state = useTextEditor(core, () => {})
  await state.again()
  return { wrote, state, tab: mount(TextEditorTab, { props: { state } }) }
}

describe('the file drawn', () => {
  it('stands in the editor, read as JSON', async () => {
    const { tab } = await drawn()
    const editor = tab.getComponent({ name: 'Editor' })

    expect(editor.props('language')).toBe('json')
    expect(editor.props('modelValue')).toBe(HELD)
    // Nothing here is markdown, so no mark is drawn as what it means.
    expect(editor.props('live')).toBe(false)
  })

  it('carries no bar of its own over the text', async () => {
    const { tab } = await drawn()

    expect(tab.findAll('button')).toHaveLength(0)
    expect(tab.text()).not.toContain('/numen.json')
  })

  it('is kept the way a note is kept', async () => {
    const { tab, wrote } = await drawn()
    const editor = tab.getComponent({ name: 'Editor' })

    await editor.vm.$emit('update:modelValue', '{}\n')
    await editor.vm.$emit('save')

    expect(wrote).toStrictEqual(['{}\n'])
  })

  it('says what is wrong where the settings could not be read out of it', async () => {
    const { tab, state } = await drawn({
      saveSettingsFile: () =>
        Promise.reject(
          new ConnectError('not a setting: it does not read as JSON, at byte 12', Code.InvalidArgument),
        ),
    })
    state.types('{ "agent": ')
    await state.keeps()
    await tab.vm.$nextTick()

    const said = tab.get('[role="alert"]').text()
    expect(said).toContain(words.unwritten)
    expect(said).toContain('at byte 12')
  })

  it('puts the two answers where the file moved past what was read', async () => {
    const wrote: string[] = []
    const { tab, state } = await drawn({
      saveSettingsFile: (written, seen) => {
        if (seen !== null) return Promise.resolve({ changed: true })
        wrote.push(written)
        return Promise.resolve({ changed: false })
      },
    })
    state.types('{}\n')
    await state.keeps()
    await tab.vm.$nextTick()

    expect(tab.get('[role="status"]').text()).toContain(words.stale)
    const answers = tab.findAll('button')
    expect(answers.map((one) => one.text())).toStrictEqual([words.keep, words.take])

    await answers[0]!.trigger('click')
    await tab.vm.$nextTick()

    expect(wrote).toStrictEqual(['{}\n'])
    expect(tab.findAll('button')).toHaveLength(0)
  })
})
