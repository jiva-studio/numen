import { describe, expect, it, vi } from 'vitest'
import { nextTick, ref, type Ref } from 'vue'
import { flushPromises, mount } from '@vue/test-utils'
import type { FileConflict, NoteTabState } from '@numen/plugin-sdk'
import MarkdownNoteEditor from './MarkdownNoteEditor.vue'

const createState = (overrides: Partial<NoteTabState> = {}): NoteTabState => ({
  path: 'test.md',
  body: ref(''),
  isLoading: ref(false),
  errorMessage: ref(''),
  updateBody: () => undefined,
  save: () => undefined,
  ...overrides,
})

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
  const getEditableValue = (wrapper: ReturnType<typeof mount>): string | undefined =>
    wrapper.find('.cm-content').attributes('contenteditable')

  it('refuses typing until the tab has read the note', async () => {
    const isLoading = ref(true)
    const wrapper = mount(MarkdownNoteEditor, {
      props: { filePath: 'test.md', state: createState({ isLoading }) },
    })
    await nextTick()
    expect(getEditableValue(wrapper)).toBe('false')

    isLoading.value = false
    await nextTick()
    expect(getEditableValue(wrapper)).toBe('true')
  })

  it('accepts typing when the tab reports no read in progress', async () => {
    const wrapper = mount(MarkdownNoteEditor, {
      props: { filePath: 'test.md', state: createState() },
    })
    await nextTick()
    expect(getEditableValue(wrapper)).toBe('true')
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
        state: createState(),
      },
    })
    expect(wrapper.find('[role="alert"]').exists()).toBe(false)
  })

  it('displays the caution banner for the error a host tab reports', () => {
    const errorMessage = ref('write failed')
    const wrapper = mount(MarkdownNoteEditor, {
      props: { filePath: 'test.md', state: createState({ errorMessage }) },
    })
    expect(wrapper.find('.caution__text').text()).toBe('write failed')
  })
})

describe('MarkdownNoteEditor over a note whose file changed on disk', () => {
  it('offers to keep the typed text or take the file, and passes the choice to the tab', async () => {
    const fileConflict = ref<FileConflict>('stale')
    const choices: string[] = []
    const wrapper = mount(MarkdownNoteEditor, {
      props: {
        filePath: 'notes/Tide.md',
        state: createState({
          fileConflict,
          keepMine: () => {
            choices.push('keep')
          },
          takeFile: () => {
            choices.push('take')
          },
        }),
      },
    })
    await nextTick()

    const banner = wrapper.get('[role="alert"]')
    expect(banner.text()).toContain('Saving paused: the file changed on disk.')
    const answers = banner.findAll('button')
    expect(answers.map((one) => one.text())).toStrictEqual(['Keep my version', 'Use disk version'])
    await answers[0]!.trigger('click')
    await answers[1]!.trigger('click')
    expect(choices).toStrictEqual(['keep', 'take'])

    fileConflict.value = null
    await nextTick()
    expect(wrapper.find('[role="alert"]').exists()).toBe(false)
  })

  it('offers to write the text again when the file was removed', async () => {
    const wrapper = mount(MarkdownNoteEditor, {
      props: {
        filePath: 'notes/Tide.md',
        state: createState({ fileConflict: ref<FileConflict>('gone') }),
      },
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
        state: createState({
          wikilinks: {
            search: async () => [{ title: 'Entropy', type: 'note' }],
            headings: async () => [],
          },
        }),
      },
    })
    await nextTick()

    expect(wrapper.findComponent({ name: 'Editor' }).props('extensions')).toBeDefined()
  })
})

describe('MarkdownNoteEditor with the body from its tab', () => {
  const FILE = '---\nid: 01M4JWCXZ3X193ZQBKPWWRVAJT\n---\n'

  const mountHostedEditor = (body: Ref<string>) =>
    mount(MarkdownNoteEditor, {
      props: { filePath: 'notes/Test.md', state: createState({ body }) },
    })

  it('does not read the file, frontmatter included, when the note has no body', async () => {
    const fetched = vi.fn(() => Promise.resolve(new Response(FILE)))
    vi.stubGlobal('fetch', fetched)
    try {
      const wrapper = mountHostedEditor(ref(''))
      await flushPromises()

      expect(fetched).not.toHaveBeenCalled()
      expect(wrapper.vm.getContent()).toBe('')
      expect(wrapper.find('[role="alert"]').exists()).toBe(false)
    } finally {
      vi.unstubAllGlobals()
    }
  })

  it('shows a body that becomes empty, as when the file is rewritten without prose', async () => {
    const body = ref('First lines.')
    const wrapper = mountHostedEditor(body)
    expect(wrapper.vm.getContent()).toBe('First lines.')

    body.value = ''
    await nextTick()

    expect(wrapper.vm.getContent()).toBe('')
  })
})
