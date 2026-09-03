/**
 * The settings file drawn: the editor, and nothing standing over it.
 *
 * The tab already carries the file's name, and the text is kept the way a note
 * is kept, so there is no bar above it saying either.
 */
import { describe, expect, it } from 'vitest'
import { mount } from '@vue/test-utils'
import ConfigurationTab from './ConfigurationTab.vue'
import { holding, type Called } from './kind'
import { WORDS as words } from './words'

const HELD = '{\n  "agent": { "use": "claude" }\n}\n'

const standing = async (answers: Partial<Called> = {}) => {
  const wrote: string[] = []
  const core: Called = {
    settingsFile: () => Promise.resolve({ written: HELD, path: '/numen.json' }),
    writesSettingsFile: (written) => {
      wrote.push(written)
      return Promise.resolve()
    },
    ...answers,
  }
  const held = holding(core, () => {})
  await held.again()
  return { wrote, held, tab: mount(ConfigurationTab, { props: { held } }) }
}

describe('the file drawn', () => {
  it('stands in the editor, read as JSON', async () => {
    const { tab } = await standing()
    const editor = tab.getComponent({ name: 'Editor' })

    expect(editor.props('language')).toBe('json')
    expect(editor.props('modelValue')).toBe(HELD)
    // Nothing here is markdown, so no mark is drawn as what it means.
    expect(editor.props('live')).toBe(false)
  })

  it('carries no bar of its own over the text', async () => {
    const { tab } = await standing()

    expect(tab.findAll('button')).toHaveLength(0)
    expect(tab.text()).not.toContain('/numen.json')
  })

  it('is kept the way a note is kept', async () => {
    const { tab, wrote } = await standing()
    const editor = tab.getComponent({ name: 'Editor' })

    await editor.vm.$emit('update:modelValue', '{}\n')
    await editor.vm.$emit('save')

    expect(wrote).toStrictEqual(['{}\n'])
  })

  it('says what is wrong where the settings could not be read out of it', async () => {
    const { tab, held } = await standing({
      writesSettingsFile: () =>
        Promise.reject(new Error('not a setting: it does not read as JSON, at byte 12')),
    })
    held.types('{ "agent": ')
    await held.keeps()
    await tab.vm.$nextTick()

    const said = tab.get('[role="alert"]').text()
    expect(said).toContain(words.unwritten)
    expect(said).toContain('at byte 12')
  })
})
