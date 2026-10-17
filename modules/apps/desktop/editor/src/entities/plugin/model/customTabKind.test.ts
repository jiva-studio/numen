import { describe, expect, it, vi } from 'vitest'
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
    await wrapper.vm.$nextTick()

    const root = wrapper.element.shadowRoot ?? wrapper.element
    expect(root.querySelector('.inner-component')).not.toBeNull()
    expect(root.querySelector('.prop-path')?.textContent).toBe('notes/meeting.md')
    expect(root.querySelector('.prop-filepath')?.textContent).toBe('notes/meeting.md')
    expect(root.querySelector('.prop-file')?.textContent).toBe('notes/meeting.md')
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

    await new Promise((resolve) => setTimeout(resolve, 50))
    await wrapper.vm.$nextTick()

    const root = wrapper.element.shadowRoot ?? wrapper.element
    expect(root.textContent).toContain('async ready')
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

describe('createReaderTabKind (source resolution)', () => {
  it('identifies source via registration source property', () => {
    const kind = createReaderTabKind('audio-plugin', {
      id: 'custom-audio',
      fileExtensions: ['.custom'],
      component: null,
      source: 'recording',
    })

    const target = kind.getTarget?.({ path: 'track.custom' })
    expect(target).toEqual({
      file: 'track.custom',
      path: 'track.custom',
      title: '',
      source: 'recording',
    })
  })

  it('identifies source via capability flags', () => {
    const kind = createReaderTabKind('media-plugin', {
      id: 'media-stream',
      fileExtensions: ['.stream'],
      component: null,
      capabilities: ['audio', 'media'],
    })

    const target = kind.getTarget?.({ path: 'live.stream' })
    expect(target).toEqual({
      file: 'live.stream',
      path: 'live.stream',
      title: '',
      source: 'recording',
    })
  })

  it('identifies source via metadata', () => {
    const kind = createReaderTabKind('doc-plugin', {
      id: 'doc-reader',
      fileExtensions: ['.doc'],
      component: null,
      metadata: { source: 'book' },
    })

    const target = kind.getTarget?.({ path: 'novel.doc' })
    expect(target).toEqual({ file: 'novel.doc', path: 'novel.doc', title: '', source: 'book' })
  })

  it('falls back to file extension when no media flags or metadata are present', () => {
    const kind = createReaderTabKind('generic-plugin', {
      id: 'generic-reader',
      fileExtensions: ['.mp3'],
      component: null,
    })

    const target = kind.getTarget?.({ path: 'sound.mp3' })
    expect(target).toEqual({ file: 'sound.mp3', path: 'sound.mp3', title: '', source: 'recording' })
  })
})

describe('createCustomTabKind (actions)', () => {
  it('binds extensible action handlers from registration and host deps', () => {
    const customAction = vi.fn()
    const hostAction = vi.fn()

    const kind = createCustomTabKind(
      'cards-plugin',
      {
        kind: 'deck',
        component: null,
        actions: {
          exportDeck: customAction,
        },
      },
      {
        registerKind: vi.fn(() => () => {}),
        registerExtensionReader: vi.fn(() => () => {}),
        openTab: vi.fn(async () => 'tab-1'),
        tabActionHandlers: {
          syncDeck: hostAction,
        },
      },
    )

    const state = kind.open('decks/test.md') as Record<string, unknown>
    expect(typeof state.exportDeck).toBe('function')
    expect(typeof state.syncDeck).toBe('function')

    ;(state.exportDeck as (arg: string) => void)('json')
    expect(customAction).toHaveBeenCalledWith('decks/test.md', 'json')

    ;(state.syncDeck as (force: boolean) => void)(true)
    expect(hostAction).toHaveBeenCalledWith('decks/test.md', true)
  })

  it('supports backwards-compatible fallback for onCustomCardAdded', () => {
    const legacyAddCard = vi.fn()

    const kind = createCustomTabKind(
      'cards-plugin',
      {
        kind: 'deck',
        component: null,
      },
      {
        registerKind: vi.fn(() => () => {}),
        registerExtensionReader: vi.fn(() => () => {}),
        openTab: vi.fn(async () => 'tab-1'),
        onCustomCardAdded: legacyAddCard,
      },
    )

    const state = kind.open('decks/legacy.md') as {
      addCard: (stencil: string, values: readonly { field: string; text: string }[]) => void
    }
    expect(typeof state.addCard).toBe('function')

    state.addCard('stencil-1', [{ field: 'Front', text: 'Q' }])
    expect(legacyAddCard).toHaveBeenCalledWith('decks/legacy.md', [{ field: 'Front', text: 'Q' }])
  })

  it('prevents action names from overwriting tab-state fields', () => {
    const collidingAction = vi.fn()

    const kind = createCustomTabKind(
      'cards-plugin',
      {
        kind: 'deck',
        component: null,
        actions: {
          path: collidingAction,
          title: collidingAction,
        },
      },
      {
        registerKind: vi.fn(() => () => {}),
        registerExtensionReader: vi.fn(() => () => {}),
        openTab: vi.fn(async () => 'tab-1'),
        tabActionHandlers: {
          file: collidingAction,
        },
      },
    )

    const state = kind.open('decks/secure.md') as Record<string, unknown>
    expect(state.path).toBe('decks/secure.md')
    expect(state.file).toBeUndefined()
    expect(typeof state.path).toBe('string')
  })
})
