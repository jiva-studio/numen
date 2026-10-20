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

    const openState = readerKind.open('books/alice.epub', 'tab') as {
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

    const openState = customKind.open('notes/ideas.md', 'tab') as {
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

    const openState = customKind.open('notes/test.md', 'tab') as {
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

describe('createCustomTabKind (loading)', () => {
  const registration = { kind: 'note', component: null, title: 'Markdown Note' }
  const createHost = (readNote?: (path: string) => Promise<string | undefined>) => ({
    registerKind: () => vi.fn(),
    registerExtensionReader: () => vi.fn(),
    openTab: () => Promise.resolve(''),
    ...(readNote ? { readNote } : {}),
  })
  const open = (readNote?: (path: string) => Promise<string | undefined>) =>
    createCustomTabKind('md.numen.editor-markdown', registration, createHost(readNote)).open(
      'notes/test.md',
      'tab',
    ) as { isLoading: { value: boolean } }

  it('is loading until the note has been read', async () => {
    const reads: ((body: string) => void)[] = []
    const state = open(() => new Promise<string>((resolve) => reads.push(resolve)))
    expect(state.isLoading.value).toBe(true)

    reads[0]?.('# Body')
    await new Promise((resolve) => setTimeout(resolve, 0))
    expect(state.isLoading.value).toBe(false)
  })

  it('stops loading when the read fails', async () => {
    const state = open(() => Promise.reject(new Error('unreachable')))
    expect(state.isLoading.value).toBe(true)

    await new Promise((resolve) => setTimeout(resolve, 0))
    expect(state.isLoading.value).toBe(false)
  })

  it('is not loading when the host reads no notes', () => {
    expect(open().isLoading.value).toBe(false)
  })
})

describe('createCustomTabKind (failed writes)', () => {
  const host = (writeNote: (path: string, body: string) => Promise<void>) => ({
    registerKind: () => vi.fn(),
    registerExtensionReader: () => vi.fn(),
    openTab: () => Promise.resolve(''),
    writeNote,
  })
  const open = (writeNote: (path: string, body: string) => Promise<void>) =>
    createCustomTabKind(
      'md.numen.editor-markdown',
      { kind: 'note', component: null, title: 'Markdown Note' },
      host(writeNote),
    ).open('notes/test.md', 'tab') as {
      errorMessage: { value: string }
      updateBody: (body: string) => void
      save: () => void
    }

  it('shows the message of a write that was rejected', async () => {
    const state = open(() => Promise.reject(new Error('the note could not be written')))
    state.updateBody('edited')
    await new Promise((resolve) => setTimeout(resolve, 0))
    expect(state.errorMessage.value).toBe('the note could not be written')
  })

  it('clears the message once a newer write succeeds', async () => {
    const answers = [Promise.reject(new Error('unreachable')), Promise.resolve()]
    const state = open(() => answers.shift() ?? Promise.resolve())
    state.updateBody('one')
    await new Promise((resolve) => setTimeout(resolve, 0))
    expect(state.errorMessage.value).toBe('unreachable')

    state.updateBody('two')
    await new Promise((resolve) => setTimeout(resolve, 0))
    expect(state.errorMessage.value).toBe('')
  })

  it('keeps the message of a read when a write succeeds', async () => {
    const state = open(() => Promise.resolve())
    state.errorMessage.value = 'the frontmatter of this note cannot be read'
    state.updateBody('one')
    await new Promise((resolve) => setTimeout(resolve, 0))
    expect(state.errorMessage.value).toBe('the frontmatter of this note cannot be read')
  })

  it('ignores the failure of a write that a newer write replaced', async () => {
    let fail: (reason: Error) => void = () => undefined
    const answers = [new Promise<void>((_, reject) => (fail = reject)), Promise.resolve()]
    const state = open(() => answers.shift() ?? Promise.resolve())
    state.updateBody('one')
    state.updateBody('two')
    fail(new Error('stale'))
    await new Promise((resolve) => setTimeout(resolve, 0))
    expect(state.errorMessage.value).toBe('')
  })

  it('shows the message of a save that was rejected', async () => {
    const state = open(() => Promise.reject(new Error('unreachable')))
    state.save()
    await new Promise((resolve) => setTimeout(resolve, 0))
    expect(state.errorMessage.value).toBe('unreachable')
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

    const state = kind.open('decks/test.md', 'tab') as Record<string, unknown>
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

    const state = kind.open('decks/legacy.md', 'tab') as {
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

    const state = kind.open('decks/secure.md', 'tab') as Record<string, unknown>
    expect(state.path).toBe('decks/secure.md')
    expect(state.file).toBeUndefined()
    expect(typeof state.path).toBe('string')
  })
})
