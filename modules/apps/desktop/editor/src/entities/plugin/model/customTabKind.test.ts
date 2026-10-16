import { describe, expect, it } from 'vitest'
import { defineComponent, h, ref } from 'vue'
import { mount } from '@vue/test-utils'
import { createCustomTabKind, createPluginPaneWrapper, createReaderTabKind } from './customTabKind'

describe('createPluginPaneWrapper', () => {
  it('unwraps es module default export and forwards normalized props', async () => {
    const InnerComponent = defineComponent({
      name: 'InnerComponent',
      props: {
        path: { type: String, default: '' },
        filePath: { type: String, default: '' },
        file: { type: String, default: '' },
        state: { type: null, default: null },
      },
      setup(props) {
        return () =>
          h('div', { class: 'inner-component' }, [
            h('span', { class: 'prop-path' }, props.path),
            h('span', { class: 'prop-filepath' }, props.filePath),
            h('span', { class: 'prop-file' }, props.file),
          ])
      },
    })

    const esModule = { default: InnerComponent }
    const Wrapper = createPluginPaneWrapper('test.plugin', esModule)
    const wrapper = mount(Wrapper, {
      props: {
        state: { path: 'notes/meeting.md' },
      },
    })

    expect(wrapper.find('.inner-component').exists()).toBe(true)
    expect(wrapper.find('.prop-path').text()).toBe('notes/meeting.md')
    expect(wrapper.find('.prop-filepath').text()).toBe('notes/meeting.md')
    expect(wrapper.find('.prop-file').text()).toBe('notes/meeting.md')
  })

  it('handles async component loaders', async () => {
    const LoadedComponent = defineComponent({
      name: 'LoadedComponent',
      setup() {
        return () => h('div', { class: 'loaded-async' }, 'async ready')
      },
    })

    const asyncLoader = () => Promise.resolve(LoadedComponent)
    const Wrapper = createPluginPaneWrapper('async.plugin', asyncLoader)
    const wrapper = mount(Wrapper, {
      props: {
        state: { path: 'notes/async.md' },
      },
    })

    await new Promise((resolve) => setTimeout(resolve, 10))
    await wrapper.vm.$nextTick()

    expect(wrapper.text()).toContain('async ready')
  })
})

describe('createReaderTabKind', () => {
  it('computes dynamic title and book target for epub reader', () => {
    const readerKind = createReaderTabKind('md.numen.epub', {
      id: 'reader-epub',
      fileExtensions: ['.epub'],
      component: null,
      title: 'EPUB Reader',
    })

    const openState = readerKind.open('books/alice.epub') as {
      path: string
      file: string
      title: string
    }
    expect(openState.path).toBe('books/alice.epub')
    expect(openState.file).toBe('books/alice.epub')
    expect(openState.title).toBe('alice.epub')

    const fallbackTitle = readerKind.getTitle!({ path: 'books/alice.epub' })
    expect(fallbackTitle).toBe('alice.epub')

    const dynamicStringTitle = readerKind.getTitle!({
      path: 'books/alice.epub',
      title: 'Alice in Wonderland',
    })
    expect(dynamicStringTitle).toBe('Alice in Wonderland')

    const dynamicRefTitle = readerKind.getTitle!({
      path: 'books/alice.epub',
      title: ref('Through the Looking-Glass'),
    })
    expect(dynamicRefTitle).toBe('Through the Looking-Glass')

    const target = readerKind.getTarget!({
      path: 'books/alice.epub',
      title: 'Alice in Wonderland',
    })
    expect(target).toEqual({
      file: 'books/alice.epub',
      path: 'books/alice.epub',
      title: 'Alice in Wonderland',
      source: 'book',
    })
  })
})

describe('createCustomTabKind', () => {
  it('computes note title and note target', () => {
    const customKind = createCustomTabKind('md.numen.editor-markdown', {
      kind: 'note',
      component: null,
      title: 'Markdown Note',
    })

    const openState = customKind.open('notes/ideas.md') as {
      at: string
      path: string
      title: string
    }
    expect(openState.path).toBe('notes/ideas.md')
    expect(openState.title).toBe('ideas')

    const initialTitle = customKind.getTitle!({ path: 'notes/ideas.md' })
    expect(initialTitle).toBe('ideas')

    const customTitle = customKind.getTitle!({
      path: 'notes/ideas.md',
      title: 'Brainstorming Session',
    })
    expect(customTitle).toBe('Brainstorming Session')

    const target = customKind.getTarget!({
      path: 'notes/ideas.md',
      title: 'Brainstorming Session',
    })
    expect(target).toEqual({
      file: 'notes/ideas.md',
      path: 'notes/ideas.md',
      title: 'Brainstorming Session',
      source: 'note',
    })
  })

  it('populates note body from readNote dependency', async () => {
    let savedBody = ''
    const customKind = createCustomTabKind(
      'md.numen.editor-markdown',
      {
        kind: 'note',
        component: null,
        title: 'Markdown Note',
      },
      {
        registerKind: () => () => {},
        registerExtensionReader: () => () => {},
        openTab: async () => '',
        readNote: async (path: string) => (path === 'notes/test.md' ? '# Test Content' : undefined),
        writeNote: async (_path: string, body: string) => {
          savedBody = body
        },
      },
    )

    const openState = customKind.open('notes/test.md') as {
      note: { value: { path: string; body: string } }
      body: { value: string }
      updateBody: (body: string) => void
      save: () => void
    }

    await new Promise((resolve) => setTimeout(resolve, 10))

    expect(openState.note.value.body).toBe('# Test Content')
    expect(openState.body.value).toBe('# Test Content')

    openState.updateBody('# Updated Content')
    expect(openState.note.value.body).toBe('# Updated Content')
    expect(savedBody).toBe('# Updated Content')
  })
})

