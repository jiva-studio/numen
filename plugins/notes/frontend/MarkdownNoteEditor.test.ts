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

describe('MarkdownNoteEditor error banner', () => {
  it('displays caution banner when error is present and dismisses on click', async () => {
    const wrapper = mount(MarkdownNoteEditor, {
      props: {
        filePath: 'test.md',
        errorMessage: 'frontmatter unreadable',
      },
    })
    expect(wrapper.find('[role="alert"]').exists()).toBe(true)
    expect(wrapper.find('.caution__text').text()).toBe('frontmatter unreadable')

    await wrapper.find('.caution__dismiss').trigger('click')
    expect(wrapper.find('[role="alert"]').exists()).toBe(false)
  })

  it('does not display caution banner for valid frontmatter with multiline list or wikilinks', () => {
    const validMarkdown = `---
title: Thermodynamics
links:
  - to: "[[notes/Entropy]]"
tags: [
  physics,
  heat
]
---
# Content`
    const wrapper = mount(MarkdownNoteEditor, {
      props: {
        modelValue: validMarkdown,
        filePath: 'Thermodynamics.md',
      },
    })
    expect(wrapper.find('[role="alert"]').exists()).toBe(false)
  })

  it('displays caution banner for malformed frontmatter with unclosed brackets', () => {
    const malformed = `---
title: [unclosed bracket
key: value:
---
# Content`
    const wrapper = mount(MarkdownNoteEditor, {
      props: {
        modelValue: malformed,
        filePath: 'malformed.md',
      },
    })
    expect(wrapper.find('[role="alert"]').exists()).toBe(true)
    expect(wrapper.find('.caution__text').text()).toBe('frontmatter unreadable')
  })

  it('does not display caution banner when errorMessage and the state error are empty', () => {
    const wrapper = mount(MarkdownNoteEditor, {
      props: {
        filePath: 'test.md',
        errorMessage: '',
        state: { errorMessage: ref('') },
      },
    })
    expect(wrapper.find('[role="alert"]').exists()).toBe(false)
  })

  it('reads the error ref a host tab passes as errorMessage', () => {
    const errorMessage = ref('')
    const wrapper = mount(MarkdownNoteEditor, {
      props: { filePath: 'test.md', errorMessage, state: { errorMessage } },
    })
    expect(wrapper.find('[role="alert"]').exists()).toBe(false)
  })

  it('displays the caution banner for an error ref a host tab passes', () => {
    const errorMessage = ref('write failed')
    const wrapper = mount(MarkdownNoteEditor, {
      props: { filePath: 'test.md', errorMessage },
    })
    expect(wrapper.find('.caution__text').text()).toBe('write failed')
  })
})

describe('MarkdownNoteEditor over a note whose file changed on disk', () => {
  it('offers to keep the typed text or take the file, and passes the choice to the tab', async () => {
    const fileConflict = ref<string | null>('stale')
    const chosen: string[] = []
    const wrapper = mount(MarkdownNoteEditor, {
      props: {
        filePath: 'notes/Tide.md',
        state: {
          fileConflict,
          keepMine: () => chosen.push('keep'),
          takeFile: () => chosen.push('take'),
        },
      },
    })
    await nextTick()

    const banner = wrapper.get('[role="alert"]')
    expect(banner.text()).toContain('Saving paused: the file changed on disk.')
    const answers = banner.findAll('button')
    expect(answers.map((one) => one.text())).toStrictEqual(['Keep my version', 'Use disk version'])
    await answers[0]!.trigger('click')
    await answers[1]!.trigger('click')
    expect(chosen).toStrictEqual(['keep', 'take'])

    fileConflict.value = null
    await nextTick()
    expect(wrapper.find('[role="alert"]').exists()).toBe(false)
  })

  it('offers to write the text again when the file was removed', async () => {
    const wrapper = mount(MarkdownNoteEditor, {
      props: { filePath: 'notes/Tide.md', state: { fileConflict: 'gone' } },
    })
    await nextTick()

    const banner = wrapper.get('[role="alert"]')
    expect(banner.text()).toContain('the note was removed from the vault')
    expect(banner.findAll('button').map((one) => one.text())).toStrictEqual(['Recreate it'])
  })
})

describe('MarkdownNoteEditor over a tab that offers links', () => {
  it('draws the completion of wikilinks from the tab', async () => {
    const wrapper = mount(MarkdownNoteEditor, {
      props: {
        filePath: 'notes/Tide.md',
        state: { wikilinks: { search: async () => [{ title: 'Entropy', type: 'note' }] } },
      },
    })
    await nextTick()

    expect(wrapper.findComponent({ name: 'Editor' }).props('extensions')).toBeDefined()
  })
})
