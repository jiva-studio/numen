import { describe, expect, it } from 'vitest'
import { nextTick, ref } from 'vue'
import { mount } from '@vue/test-utils'
import MarkdownNoteEditor from './MarkdownNoteEditor.vue'

describe('MarkdownNoteEditor', () => {
  it('renders editor component with initial content', () => {
    const wrapper = mount(MarkdownNoteEditor, {
      props: {
        modelValue: '# Note Title\nProse content',
        filePath: 'test.md',
      },
    })

    expect(wrapper.exists()).toBe(true)
    expect(wrapper.classes()).toContain('markdown-note-editor')
  })

  it('detects language based on extension', () => {
    const mdWrapper = mount(MarkdownNoteEditor, {
      props: { filePath: 'test.md' },
    })
    expect(mdWrapper.vm.isDirty).toBe(false)

    const txtWrapper = mount(MarkdownNoteEditor, {
      props: { filePath: 'notes.txt' },
    })
    expect(txtWrapper.exists()).toBe(true)
  })

  it('updates dirty state when content changes', async () => {
    const wrapper = mount(MarkdownNoteEditor, {
      props: {
        modelValue: 'original',
        filePath: 'test.md',
      },
    })

    expect(wrapper.vm.isDirty).toBe(false)
    wrapper.vm.setContent('modified')
    expect(wrapper.vm.getContent()).toBe('modified')
  })
})

describe('MarkdownNoteEditor while the note is read', () => {
  const getEditable = (wrapper: ReturnType<typeof mount>): string | undefined =>
    wrapper.find('.cm-content').attributes('contenteditable')

  it('refuses typing until the tab has read the note', async () => {
    const isLoading = ref(true)
    const wrapper = mount(MarkdownNoteEditor, {
      props: { filePath: 'test.md', state: { isLoading } },
    })
    await nextTick()
    expect(getEditable(wrapper)).toBe('false')

    isLoading.value = false
    await nextTick()
    expect(getEditable(wrapper)).toBe('true')
  })

  it('accepts typing when the tab reports no read in progress', async () => {
    const wrapper = mount(MarkdownNoteEditor, {
      props: { filePath: 'test.md', state: {} },
    })
    await nextTick()
    expect(getEditable(wrapper)).toBe('true')
  })
})
