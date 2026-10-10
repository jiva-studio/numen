import { describe, expect, it, vi } from 'vitest'
import { mount } from '@vue/test-utils'
import { ref, shallowRef } from 'vue'
import NoteTab from './NoteTab.vue'

describe('NoteTab', () => {
  const createMockState = () => ({
    id: 'tab-1',
    note: ref({
      path: 'test.md',
      title: 'test',
      body: 'Hello world',
      state: 'clean',
    }),
    change: shallowRef({ from: 0, to: 0, text: '' }),
    errorMessage: ref(''),
    extensions: [],
    measure: vi.fn(),
    keepMine: vi.fn(),
    takeFile: vi.fn(),
    setEditor: vi.fn(),
    updateBody: vi.fn(),
    save: vi.fn(),
    followLink: vi.fn(),
  })

  it('renders editor and binds event handlers', async () => {
    const state = createMockState()
    const wrapper = mount(NoteTab, {
      props: { state: state as any },
      global: {
        stubs: {
          Editor: {
            name: 'Editor',
            template: '<div class="editor-stub" />',
            emits: ['update:modelValue', 'save', 'open'],
          },
          FileConflictPrompt: {
            name: 'FileConflictPrompt',
            template: '<div class="prompt-stub" />',
            emits: ['keep', 'take'],
          },
        },
      },
    })

    expect(wrapper.find('.note').exists()).toBe(true)

    const editor = wrapper.findComponent({ name: 'Editor' })
    expect(editor.exists()).toBe(true)

    editor.vm.$emit('update:modelValue', 'New body')
    expect(state.updateBody).toHaveBeenCalledWith('New body')

    editor.vm.$emit('save')
    expect(state.save).toHaveBeenCalled()

    editor.vm.$emit('open', 'other.md')
    expect(state.followLink).toHaveBeenCalledWith('other.md')

    const prompt = wrapper.findComponent({ name: 'FileConflictPrompt' })
    prompt.vm.$emit('keep')
    expect(state.keepMine).toHaveBeenCalled()

    prompt.vm.$emit('take')
    expect(state.takeFile).toHaveBeenCalled()
  })

  it('calls measure when body changes from empty to non-empty', async () => {
    const state = createMockState()
    state.note.value.body = ''
    const wrapper = mount(NoteTab, {
      props: { state: state as any },
      global: {
        stubs: {
          Editor: true,
          FileConflictPrompt: true,
        },
      },
    })

    state.note.value = { ...state.note.value, body: 'Non-empty' }
    await wrapper.vm.$nextTick()

    expect(state.measure).toHaveBeenCalled()
  })
})
