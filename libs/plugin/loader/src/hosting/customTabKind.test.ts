import { describe, expect, it, vi } from 'vitest'
import { defineComponent, effectScope, h, ref } from 'vue'
import { mount } from '@vue/test-utils'
import { clearPluginStyles, loadPluginStyle, registerPluginStyle } from '../styles'
import { createBufferedNoteState } from './customTabBuffer'
import { createCustomTabKind, createPluginPaneWrapper, createReaderTabKind } from './customTabKind'
import { getReaderKindId } from './tabKinds'
import {
  attachPluginRoot,
  extractFilename,
  parseTitle,
  renderFallback,
  renderPaneContent,
  parsePluginComponent,
  stripMarkdownExtension,
} from './pane/pluginView'
import { adoptHostStyles, parseStatePath, PANE_LAYOUT_STYLE } from './pane/paneHost'
import type { PluginContext, ReaderHandle, TabKindRegistration } from '@numen/plugin-sdk'
import type { NoteBuffer } from './types'
import { STUB_COMPONENT } from '../testing/stubComponent'
import { createTestHostDeps, type TestHostDeps } from '../testing/hostDeps'

/** The deps of a custom tab kind written flat, with the context of its plugin and whether it edits notes. */
type FlatTabKindDeps = TestHostDeps & {
  readonly getContext?: () => PluginContext | undefined
  readonly isNoteEditor?: boolean
}

const createFlatCustomTabKind = (
  pluginId: string,
  registration: TabKindRegistration,
  deps?: FlatTabKindDeps,
) =>
  createCustomTabKind(
    pluginId,
    registration,
    deps && {
      ...createTestHostDeps(deps),
      ...(deps.getContext ? { getContext: deps.getContext } : {}),
      ...(deps.isNoteEditor ? { isNoteEditor: true } : {}),
    },
  )

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
  it('prefixes a reader id with reader: once', () => {
    expect(getReaderKindId('epub')).toBe('reader:epub')
    expect(getReaderKindId('reader:epub')).toBe('reader:epub')
  })

  it('computes dynamic title and book target for epub reader', () => {
    const readerKind = createReaderTabKind('md.numen.readers.epub', {
      id: 'reader-epub',
      component: STUB_COMPONENT,
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

    const target = readerKind.address?.getTarget!({
      path: 'books/alice.epub',
      title: 'Alice in Wonderland',
    })
    expect(target).toEqual({
      file: 'books/alice.epub',
      path: 'books/alice.epub',
      title: 'Alice in Wonderland',
    })
  })
})

describe('createCustomTabKind', () => {
  it('computes note title and note target', () => {
    const customKind = createFlatCustomTabKind('md.numen.notes', {
      kind: 'note',
      component: STUB_COMPONENT,
      title: 'Markdown Note',
    })

    const openState = customKind.open('notes/ideas.md', 'tab') as {
      at: string
      path: string
      title: string
      note: { value: { path: string; body: string } }
      navigate: (next: string, title?: string) => void
    }
    expect(openState.path).toBe('notes/ideas.md')
    expect(openState.title).toBe('ideas')
    expect(openState.note.value).toEqual({ path: 'notes/ideas.md', body: '' })

    openState.navigate('.md')
    expect(openState.title).toBe('')
    expect(openState.note.value).toEqual({ path: '.md', body: '' })

    const initialTitle = customKind.getTitle!({ path: 'notes/ideas.md' })
    expect(initialTitle).toBe('ideas')

    const customTitle = customKind.getTitle!({
      path: 'notes/ideas.md',
      title: 'Brainstorming Session',
    })
    expect(customTitle).toBe('Brainstorming Session')

    const target = customKind.address?.getTarget!({
      path: 'notes/ideas.md',
      title: 'Brainstorming Session',
    })
    expect(target).toEqual({
      file: 'notes/ideas.md',
      path: 'notes/ideas.md',
      title: 'Brainstorming Session',
    })
  })

  it('populates note body from readNote dependency', async () => {
    let savedBody = ''
    const customKind = createFlatCustomTabKind(
      'md.numen.notes',
      {
        kind: 'note',
        component: STUB_COMPONENT,
        title: 'Markdown Note',
      },
      {
        target: 'desktop.editor',
        registerKind: () => vi.fn(),
        registerExtensionReader: () => vi.fn(),
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
  const registration = { kind: 'note', component: STUB_COMPONENT, title: 'Markdown Note' }
  const createHost = (readNote?: (path: string) => Promise<string | undefined>) => ({
    target: 'desktop.editor',
    registerKind: () => vi.fn(),
    registerExtensionReader: () => vi.fn(),
    openTab: () => Promise.resolve(''),
    ...(readNote ? { readNote } : {}),
  })
  const open = (readNote?: (path: string) => Promise<string | undefined>) =>
    createFlatCustomTabKind('md.numen.notes', registration, createHost(readNote)).open(
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
    target: 'desktop.editor',
    registerKind: () => vi.fn(),
    registerExtensionReader: () => vi.fn(),
    openTab: () => Promise.resolve(''),
    writeNote,
  })
  const open = (writeNote: (path: string, body: string) => Promise<void>) =>
    createFlatCustomTabKind(
      'md.numen.notes',
      { kind: 'note', component: STUB_COMPONENT, title: 'Markdown Note' },
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
    state.errorMessage.value = 'frontmatter unreadable'
    state.updateBody('one')
    await new Promise((resolve) => setTimeout(resolve, 0))
    expect(state.errorMessage.value).toBe('frontmatter unreadable')
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

describe('createReaderTabKind (place)', () => {
  it('opens at the place saved before a restart and saves the place the reader shows', () => {
    const kind = createReaderTabKind('viewer-plugin', { id: 'viewer', component: STUB_COMPONENT })

    const state = kind.open('a.bin', 'tab', { page: 4 }) as {
      selector: unknown
      savePlace: (place: unknown) => void
    }
    expect(state.selector).toEqual({ page: 4 })
    expect(kind.address?.getState?.(state)).toEqual({ page: 4 })

    state.savePlace({ page: 9 })
    expect(kind.address?.getState?.(state)).toEqual({ page: 9 })
  })

  it('opens a new tab at no place and saves none', () => {
    const kind = createReaderTabKind('viewer-plugin', { id: 'viewer', component: STUB_COMPONENT })

    const state = kind.open('a.bin', 'tab')

    expect((state as { selector: unknown }).selector).toBeNull()
    expect(kind.address?.getState?.(state)).toBeUndefined()
  })

  it('reports to the open tabs the last place the reader reports', () => {
    const kind = createReaderTabKind('viewer-plugin', { id: 'viewer', component: STUB_COMPONENT })
    const state = kind.open('library/Wading Birds.pdf', 'tab') as {
      reportPlace: (place: unknown) => void
    }

    state.reportPlace({ page: 7, pageCount: 12 })
    state.reportPlace({ page: 8, pageCount: 12 })

    expect(kind.address?.getOpenTab?.(state)).toStrictEqual({
      path: 'library/Wading Birds.pdf',
      place: { page: 8, pageCount: 12 },
    })
  })

  it('leaves the place out of the open tab while the reader has reported none', () => {
    const kind = createReaderTabKind('viewer-plugin', { id: 'viewer', component: STUB_COMPONENT })

    const state = kind.open('library/Wading Birds.pdf', 'tab', { page: 4 })

    expect(kind.address?.getOpenTab?.(state)).toStrictEqual({ path: 'library/Wading Birds.pdf' })
  })

  it('keeps the place it saves for a restart apart from the place it reports', () => {
    const kind = createReaderTabKind('viewer-plugin', { id: 'viewer', component: STUB_COMPONENT })
    const state = kind.open('talks/Heron Talk.mp3', 'tab') as {
      savePlace: (place: unknown) => void
      reportPlace: (place: unknown) => void
    }

    state.savePlace({ t: [66.5, 70] })
    expect(kind.address?.getOpenTab?.(state)).toStrictEqual({ path: 'talks/Heron Talk.mp3' })

    state.reportPlace({ positionSeconds: 66.5 })
    expect(kind.address?.getState?.(state)).toEqual({ t: [66.5, 70] })
    expect(kind.address?.getOpenTab?.(state)).toStrictEqual({
      path: 'talks/Heron Talk.mp3',
      place: { positionSeconds: 66.5 },
    })
  })

  it('carries the icon the reader registers', () => {
    const icon = { name: 'Icon' }
    const kind = createReaderTabKind('viewer-plugin', {
      id: 'viewer',
      component: STUB_COMPONENT,
      icon,
    })

    expect(kind.display?.icon).toBe(icon)
  })

  it('omits title from top-level VNode props so it never leaks as a native HTML tooltip', () => {
    const FakeReader = { name: 'FakeReader', render: () => null }
    const kind = createReaderTabKind('viewer-plugin', { id: 'viewer', component: FakeReader })
    const state = kind.open('audio/730707BG.LON.mp3', 'tab')
    const nodes = renderPaneContent({
      pluginId: 'viewer-plugin',
      component: FakeReader,
      state,
      shadowRoot: null,
      containerClass: 'reader-pane',
    }) as unknown[]
    const wrapperVNode = nodes[2] as { children?: Array<{ props?: Record<string, unknown> }> }
    const childProps = wrapperVNode.children?.[0]?.props ?? {}
    expect(childProps.path).toBe('audio/730707BG.LON.mp3')
    expect(childProps.pluginId).toBe('viewer-plugin')
    expect(childProps.title).toBeUndefined()
    expect((childProps.state as { title?: string })?.title).toBe('730707BG.LON.mp3')
  })
})

describe('createCustomTabKind (notes)', () => {
  it('navigates note tab in place and syncs path when shown', async () => {
    const syncPath = vi.fn(() => true)
    const writeCalls: Array<{ path: string; body: string }> = []
    const kind = createFlatCustomTabKind(
      'md.numen.notes',
      {
        kind: 'note',
        component: STUB_COMPONENT,
        title: 'Note',
      },
      {
        target: 'desktop.editor',
        registerKind: () => vi.fn(),
        registerExtensionReader: () => vi.fn(),
        openTab: async () => '',
        readNote: async (path: string) =>
          path === 'notes/first.md' ? '# First' : path === 'notes/second.md' ? '# Second' : '',
        writeNote: async (path: string, body: string) => {
          writeCalls.push({ path, body })
        },
        getWorkspace: () => ({ files: { syncPath } }) as never,
      },
    )

    expect(typeof kind.address?.navigate).toBe('function')

    const state = kind.open('notes/first.md', 'note:notes/first.md') as {
      path: string
      at: string
      title: string
      body: { value: string }
      updateBody: (body: string) => void
    }
    await new Promise((resolve) => setTimeout(resolve, 10))
    expect(state.body.value).toBe('# First')
    expect(kind.getTitle!(state)).toBe('first')

    await kind.address?.navigate!(state, 'notes/second.md', 'Second Note')
    await new Promise((resolve) => setTimeout(resolve, 10))

    expect(state.path).toBe('notes/second.md')
    expect(state.at).toBe('notes/second.md')
    expect(kind.getTitle!(state)).toBe('Second Note')
    expect(state.body.value).toBe('# Second')

    state.updateBody('# Second Edited')
    expect(writeCalls).toEqual([{ path: 'notes/second.md', body: '# Second Edited' }])

    kind.display?.onShow!(state, 'note:notes/first.md')
    expect(syncPath).toHaveBeenCalledWith('note:notes/first.md', 'notes/second.md', 'Second Note')
  })

  it('covers reader and custom tab kind edge cases, action collisions, and in-flight navigation', async () => {
    const measure = vi.fn()
    const focusTab = vi.fn()

    const prefixedReader = createReaderTabKind('p', {
      id: 'reader:pdf',
      component: STUB_COMPONENT,
    })
    expect(prefixedReader.kind).toBe('reader:pdf')
    const emptyReaderTab = prefixedReader.open('', 't0')
    expect(prefixedReader.getTitle!(emptyReaderTab)).toBe('')
    expect(prefixedReader.address?.getTarget!(emptyReaderTab)).toStrictEqual({
      file: '',
      path: '',
      title: '',
    })
    expect(prefixedReader.address?.getOpenTab!(emptyReaderTab)).toStrictEqual({ path: '' })
    const readerTab = prefixedReader.open('a.pdf', 't1')
    expect(prefixedReader.address?.getOpenTab!(readerTab)).toStrictEqual({ path: 'a.pdf' })
    expect(prefixedReader.address?.identity!('a.pdf')).toBe('a.pdf')
    expect(prefixedReader.address?.getState!(readerTab)).toBeUndefined()

    const middleReader = createReaderTabKind('p', {
      id: 'my-reader:2',
      component: STUB_COMPONENT,
      title: 'Fallback Reader',
    })
    expect(middleReader.kind).toBe('reader:my-reader:2')
    expect(middleReader.getTitle!(middleReader.open('', 't'))).toBe('Fallback Reader')
    expect(
      (middleReader.open('a.pdf', 't', ['array']) as { selector: unknown }).selector,
    ).toBeNull()
    expect((middleReader.open('a.pdf', 't', 'str') as { selector: unknown }).selector).toBeNull()

    prefixedReader.display?.onShow!(readerTab, 't1')
    ;(readerTab as { attachReader(reader: ReaderHandle): void }).attachReader({
      measure,
      focus: focusTab,
    })
    prefixedReader.display?.onShow!(readerTab, 't1')
    expect(measure).toHaveBeenCalledTimes(1)
    expect(focusTab).toHaveBeenCalledTimes(1)

    const syncPath = vi.fn()
    let resolveFirstRead: ((val: string) => void) | undefined

    const kindWithDispatch = createFlatCustomTabKind(
      'p',
      {
        kind: 'custom',
        component: STUB_COMPONENT,
      },
      {
        target: 'desktop.editor',
        registerKind: () => vi.fn(),
        registerExtensionReader: () => vi.fn(),
        openTab: async () => '',
        getWorkspace: () => ({ files: { syncPath } }) as never,
        readNote: (path: string) =>
          path === 'notes/slow.md'
            ? new Promise<string>((r) => {
                resolveFirstRead = r
              })
            : Promise.resolve('# Fast'),
      },
    )

    const st = kindWithDispatch.open('notes/slow.md', 't1') as Record<string, unknown>
    // Save while loading & unedited is ignored
    const writeNote = vi.fn(async () => undefined)
    const kindSaveGuard = createFlatCustomTabKind(
      'p',
      { kind: 'note', component: STUB_COMPONENT },
      {
        target: 'desktop.editor',
        registerKind: () => vi.fn(),
        registerExtensionReader: () => vi.fn(),
        openTab: async () => '',
        readNote: () => new Promise(() => undefined),
        writeNote,
      },
    )
    const loadingState = kindSaveGuard.open('notes/load.md', 't2') as {
      save: () => void
      updateBody: (b: string) => void
    }
    loadingState.save()
    expect(writeNote).not.toHaveBeenCalled()
    loadingState.updateBody('edited while loading')
    writeNote.mockClear()
    loadingState.save()
    expect(writeNote).toHaveBeenCalledWith('notes/load.md', 'edited while loading')

    // Navigating while slow.md read is in flight ignores slow.md resolution
    const navFn = st.navigate as (next: string, title?: string) => void
    ;(st.errorMessage as { value: string }).value = 'old error'
    navFn('')
    expect(st.path).toBe('notes/slow.md')
    expect((st.errorMessage as { value: string }).value).toBe('old error')
    navFn('notes/slow.md')
    expect(st.path).toBe('notes/slow.md')
    expect((st.errorMessage as { value: string }).value).toBe('old error')

    navFn('notes/fast.md')
    expect(st.path).toBe('notes/fast.md')
    expect(st.title).toBe('fast')
    expect((st.errorMessage as { value: string }).value).toBe('')
    expect((st.note as { value: { path: string; body: string } }).value).toEqual({
      path: 'notes/fast.md',
      body: '',
    })

    resolveFirstRead?.('# Stale Slow Body')
    await new Promise((r) => setTimeout(r, 10))
    expect((st.body as { value: string }).value).toBe('# Fast')

    // navigate hook ignores non-note paths
    const navSpy = vi.fn()
    await kindWithDispatch.address?.navigate!({ navigate: navSpy }, 'books/book.epub', 'Book')
    expect(navSpy).not.toHaveBeenCalled()
    await kindWithDispatch.address?.navigate!({ navigate: navSpy }, 'notes/a.md', 'A')
    expect(navSpy).toHaveBeenCalledWith('notes/a.md', 'A')

    // onShow measures the state and syncs its path
    const m1 = vi.fn()
    kindWithDispatch.display?.onShow!({ path: 'notes/no-id.md', title: '', measure: vi.fn() }, '')
    kindWithDispatch.display?.onShow!({ measure: m1, path: '', title: '' }, 't1')
    expect(m1).toHaveBeenCalledTimes(1)
    expect(syncPath).not.toHaveBeenCalled()

    const kindNoDeps = createFlatCustomTabKind('p', { kind: 'custom', component: STUB_COMPONENT })
    expect(() =>
      kindNoDeps.display?.onShow!({ path: 'notes/a.md', title: '', measure: vi.fn() }, 't1'),
    ).not.toThrow()
    const kindUndefWorkspace = createFlatCustomTabKind(
      'p',
      { kind: 'custom', component: STUB_COMPONENT },
      {
        target: 'desktop.editor',
        registerKind: () => vi.fn(),
        registerExtensionReader: () => vi.fn(),
        openTab: async () => '',
        getWorkspace: () => undefined,
      },
    )
    expect(() =>
      kindUndefWorkspace.display?.onShow!(
        { path: 'notes/a.md', title: '', measure: vi.fn() },
        't1',
      ),
    ).not.toThrow()

    kindWithDispatch.display?.onShow!(
      { path: 'notes/at-only.md', title: 'At Title', measure: vi.fn() },
      't1',
    )
    expect(syncPath).toHaveBeenCalledWith('t1', 'notes/at-only.md', 'At Title')

    // getTitle, identity, getTarget, getOpenTab fallbacks
    expect(kindWithDispatch.getTitle!({ title: '', path: '' })).toBe('custom')
    expect(kindWithDispatch.getTitle!({ title: '', path: 'notes/from-at.md' })).toBe('from-at')
    expect(kindWithDispatch.address?.identity!('notes/a.md')).toBe('notes/a.md')
    expect(kindWithDispatch.address?.identity!('')).toBe('custom')
    expect(kindWithDispatch.address?.getTarget!({ title: '', path: '' })).toStrictEqual({
      title: 'custom',
    })
    expect(
      kindWithDispatch.address?.getTarget!({ title: '', path: 'notes/from-at.md' }),
    ).toStrictEqual({
      file: 'notes/from-at.md',
      path: 'notes/from-at.md',
      title: 'from-at',
    })
    expect(kindWithDispatch.address?.getOpenTab!({ path: 'notes/from-path.md' })).toStrictEqual({
      path: 'notes/from-path.md',
    })
  })
})

describe('the pane of a custom note tab', () => {
  it('keeps the refs of a tab state out of the props of its component', async () => {
    const messages: string[] = []
    const Probe = defineComponent({
      props: {
        errorMessage: { type: String, default: '' },
        state: { type: Object, default: null },
      },
      setup: (props) => () => {
        messages.push(props.errorMessage.trim())
        return h('span', 'note')
      },
    })
    const kind = createFlatCustomTabKind('p', { kind: 'custom', component: Probe })
    const state = kind.open('notes/a.md', 't1')
    const view = mount(kind.pane, { props: { state } })
    await view.vm.$nextTick()
    expect(view.element.shadowRoot?.textContent).toContain('note')
    expect(messages).toEqual([''])
  })
})

describe('createBufferedNoteState', () => {
  function createMockBuffer() {
    const noteRef = ref({ path: 'notes/buf.md', body: 'initial', state: 'loading' })
    const errRef = ref('')
    const buffer: NoteBuffer = {
      open: vi.fn(),
      close: vi.fn(async () => true),
      reload: vi.fn(),
      flushPending: vi.fn(async () => undefined),
      getOpenNote: vi.fn(() => noteRef.value),
      getErrorMessage: vi.fn(() => errRef.value),
      setBody: vi.fn(),
      save: vi.fn(),
      keep: vi.fn(),
      take: vi.fn(),
    }
    return { buffer, noteRef, errRef }
  }

  it('reveals the requested line once the note is read and the editor is attached', async () => {
    const { buffer, noteRef } = createMockBuffer()
    const scope = effectScope()
    const state = scope.run(() =>
      createBufferedNoteState(
        { targetPath: 'notes/buf.md', id: 'tab-buf', buffer },
        { pluginId: 'md.numen.notes' },
      ),
    )!
    const reveal = vi.fn(() => true)

    state.revealLine(9)
    state.attachEditor({ reveal })
    await new Promise((r) => setTimeout(r, 0))
    expect(reveal).not.toHaveBeenCalled()

    noteRef.value = { path: 'notes/buf.md', body: 'text', state: 'ready' }
    await new Promise((r) => setTimeout(r, 0))

    expect(reveal).toHaveBeenCalledExactlyOnceWith(9)
  })

  it('binds note tab state, wikilinks, link following, navigation, and scope disposal', async () => {
    const { buffer, noteRef, errRef } = createMockBuffer()
    const findNotes = vi.fn(async (q: string) =>
      q
        ? [
            { title: 'Alpha Note', path: 'notes/alpha.md' },
            { title: 'Beta', path: 'notes/beta.md' },
          ]
        : [],
    )
    const getHeadings = vi.fn(
      async (paths: readonly string[]) => new Map([[paths[0]!, [{ text: 'H1' }, { text: 'H2' }]]]),
    )
    const openFile = vi.fn()
    const resolveLink = vi.fn(async (_from: string, addr: string) =>
      addr === 'missing' ? '' : `notes/${addr}.md`,
    )

    const customKind = createFlatCustomTabKind(
      'md.numen.notes',
      { kind: 'note', component: STUB_COMPONENT, title: 'Note' },
      {
        target: 'desktop.editor',
        registerKind: () => vi.fn(),
        registerExtensionReader: () => vi.fn(),
        openTab: async () => '',
        getNoteBuffer: () => buffer,
        isNoteEditor: true,
        getNotes: () => ({ findNotes, getHeadings }) as never,
        getWorkspace: () => ({ files: { open: openFile } }) as never,
        resolveLink,
      },
    )

    const scope = effectScope()
    const state = scope.run(() => customKind.open('notes/buf.md', 'tab-buf')) as ReturnType<
      typeof createBufferedNoteState
    >

    expect(buffer.open).toHaveBeenCalledWith('tab-buf', 'notes/buf.md')
    expect(state.at).toBe('notes/buf.md')
    expect(state.path).toBe('notes/buf.md')
    expect(state.pluginId).toBe('md.numen.notes')
    expect(state.title).toBe('buf')
    expect(state.body.value).toBe('initial')
    expect(state.note.value).toEqual({ path: 'notes/buf.md', body: 'initial' })
    expect(state.isLoading.value).toBe(true)
    expect(state.fileConflict.value).toBeNull()

    noteRef.value = { path: '', body: 'updated', state: 'stale' }
    errRef.value = 'conflict err'
    await new Promise((r) => setTimeout(r, 0))
    expect(state.path).toBe('notes/buf.md')
    expect(state.title).toBe('buf')
    expect(state.note.value).toEqual({ path: 'notes/buf.md', body: 'updated' })
    expect(state.isLoading.value).toBe(false)
    expect(state.fileConflict.value).toBe('stale')
    expect(state.errorMessage.value).toBe('conflict err')

    noteRef.value = { path: 'notes/buf.md', body: 'gone body', state: 'gone' }
    await new Promise((r) => setTimeout(r, 0))
    expect(state.fileConflict.value).toBe('gone')

    state.keepMine()
    expect(buffer.keep).toHaveBeenCalledWith('tab-buf')
    state.takeFile()
    expect(buffer.take).toHaveBeenCalledWith('tab-buf')
    state.updateBody('new text')
    expect(buffer.setBody).toHaveBeenCalledWith('tab-buf', 'new text')
    state.save()
    expect(buffer.save).toHaveBeenCalledWith('tab-buf')

    // Wikilinks search and headings
    expect(await state.wikilinks.search('al')).toEqual([
      { title: 'Alpha Note', type: 'note' },
      { title: 'Beta', type: 'note' },
    ])
    expect(findNotes).toHaveBeenCalledWith('al', 20)

    expect(await state.wikilinks.headings('alpha note')).toEqual(['H1', 'H2'])
    expect(findNotes).toHaveBeenCalledWith('alpha note', 5)
    expect(getHeadings).toHaveBeenCalledWith(['notes/alpha.md'])

    expect(await state.wikilinks.headings('nonexistent')).toEqual([])
    expect(await state.wikilinks.headings('')).toEqual(['H1', 'H2'])
    expect(getHeadings).toHaveBeenCalledWith(['notes/buf.md'])

    // Follow link
    state.followLink('target')
    state.followLink('missing')
    await new Promise((r) => setTimeout(r, 0))
    expect(resolveLink).toHaveBeenCalledWith('notes/buf.md', 'target')
    expect(openFile).toHaveBeenCalledTimes(1)
    expect(openFile).toHaveBeenCalledWith('notes/target.md', { placement: 'split' })

    // Navigate
    state.navigate('')
    state.navigate('notes/buf.md')
    expect(buffer.flushPending).not.toHaveBeenCalled()

    vi.mocked(buffer.flushPending).mockRejectedValueOnce(new Error('flush fail'))
    vi.mocked(buffer.reload).mockRejectedValueOnce(new Error('reload fail'))
    state.navigate('notes/next.md', 'Custom Next')
    expect(state.path).toBe('notes/next.md')
    expect(state.at).toBe('notes/next.md')
    expect(state.title).toBe('Custom Next')
    await new Promise((r) => setTimeout(r, 0))
    expect(buffer.flushPending).toHaveBeenCalledWith('tab-buf')
    expect(buffer.reload).toHaveBeenCalledWith('tab-buf', 'notes/next.md')

    state.navigate('notes/third.md')
    expect(state.title).toBe('third')

    // Watcher follows renamed file in buffer, and does not reset custom title when path is unchanged
    state.navigate('notes/renamed.md', 'Custom Renamed Title')
    noteRef.value = { path: 'notes/renamed.md', body: 'same-path', state: 'ready' }
    await new Promise((r) => setTimeout(r, 0))
    expect(state.title).toBe('Custom Renamed Title')

    noteRef.value = { path: 'notes/moved.md', body: 'moved', state: 'ready' }
    await new Promise((r) => setTimeout(r, 0))
    expect(state.path).toBe('notes/moved.md')
    expect(state.at).toBe('notes/moved.md')
    expect(state.title).toBe('moved')

    vi.mocked(buffer.close).mockRejectedValueOnce(new Error('close fail'))
    scope.stop()
    expect(buffer.close).toHaveBeenCalledWith('tab-buf')

    // A kind the manifest does not declare as the note editor opens no note buffer, whatever its name
    vi.mocked(buffer.open).mockClear()
    const nonNoteWithBuffer = createFlatCustomTabKind(
      'p',
      { kind: 'note', component: STUB_COMPONENT },
      {
        target: 'desktop.editor',
        registerKind: () => vi.fn(),
        registerExtensionReader: () => vi.fn(),
        openTab: async () => '',
        getNoteBuffer: () => buffer,
      },
    )
    nonNoteWithBuffer.open('notes/x.md', 'tab-x')
    expect(buffer.open).not.toHaveBeenCalled()

    // Fallbacks when deps/notes/workspace are missing
    const bareState = createBufferedNoteState(
      { targetPath: 'notes/bare.md', id: 'bare', buffer },
      { pluginId: 'p', title: 'Explicit Title' },
    )
    expect(bareState.title).toBe('Explicit Title')
    expect(await bareState.wikilinks.search('q')).toEqual([])
    expect(await bareState.wikilinks.headings('q')).toEqual([])
    expect(() => bareState.followLink('addr')).not.toThrow()

    const depsWithoutNotesOrWorkspace = createBufferedNoteState(
      { targetPath: 'notes/nonotes.md', id: 'nonotes', buffer },
      {
        pluginId: 'p',
        deps: createTestHostDeps({
          target: 'desktop.editor',
          registerKind: () => vi.fn(),
          registerExtensionReader: () => vi.fn(),
          openTab: async () => '',
          resolveLink: vi.fn(async () => 'a.md'),
        }),
      },
    )
    expect(await depsWithoutNotesOrWorkspace.wikilinks.search('q')).toEqual([])
    expect(await depsWithoutNotesOrWorkspace.wikilinks.headings('q')).toEqual([])
    depsWithoutNotesOrWorkspace.followLink('addr')

    const resolveWhenNoWorkspace = vi.fn(async () => 'a.md')
    const emptyMapState = createBufferedNoteState(
      { targetPath: '', id: 'empty', buffer },
      {
        pluginId: 'p',
        deps: createTestHostDeps({
          target: 'desktop.editor',
          registerKind: () => vi.fn(),
          registerExtensionReader: () => vi.fn(),
          openTab: async () => '',
          getNotes: () => undefined,
          getWorkspace: () => undefined,
          resolveLink: resolveWhenNoWorkspace,
        }),
      },
    )
    expect(await emptyMapState.wikilinks.search('q')).toEqual([])
    expect(await emptyMapState.wikilinks.headings('')).toEqual([])
    emptyMapState.followLink('addr')
    expect(resolveWhenNoWorkspace).not.toHaveBeenCalled()

    const undefFindNotesState = createBufferedNoteState(
      { targetPath: '', id: 'undef-find', buffer },
      {
        pluginId: 'p',
        deps: createTestHostDeps({
          target: 'desktop.editor',
          registerKind: () => vi.fn(),
          registerExtensionReader: () => vi.fn(),
          openTab: async () => '',
          getNotes: () =>
            ({
              findNotes: async () => undefined,
              getHeadings: async () => new Map(),
            }) as never,
        }),
      },
    )
    expect(await undefFindNotesState.wikilinks.search('q')).toEqual([])
    expect(await undefFindNotesState.wikilinks.headings('')).toEqual([])

    const unmappedHeadingsState = createBufferedNoteState(
      { targetPath: 'notes/unmapped.md', id: 'unmapped', buffer },
      {
        pluginId: 'p',
        deps: createTestHostDeps({
          target: 'desktop.editor',
          registerKind: () => vi.fn(),
          registerExtensionReader: () => vi.fn(),
          openTab: async () => '',
          getNotes: () =>
            ({
              findNotes: async () => [],
              getHeadings: async () => new Map(),
            }) as never,
          getWorkspace: () => ({ files: { open: openFile } }) as never,
          resolveLink: () => Promise.reject(new Error('resolve fail')),
        }),
      },
    )
    expect(await unmappedHeadingsState.wikilinks.headings('')).toEqual([])
    expect(() => unmappedHeadingsState.followLink('addr')).not.toThrow()
  })
})

describe('pluginView helpers and stylesheet adoption', () => {
  it('resolves sync, async, and functional components and extracts paths/titles', () => {
    expect(parsePluginComponent(null)).toBeNull()
    expect(parsePluginComponent(undefined)).toBeNull()
    expect(parsePluginComponent({ default: null })).toBeNull()
    expect(parsePluginComponent({ default: undefined })).toBeNull()
    expect(parsePluginComponent('div')).toBe('div')

    const syncComp = { name: 'Sync', setup: () => () => null }
    expect(parsePluginComponent(syncComp)).toBe(syncComp)
    expect(parsePluginComponent({ default: syncComp })).toBe(syncComp)

    const objWithLoaderOnly = { loader: () => Promise.resolve(syncComp) }
    expect(parsePluginComponent(objWithLoaderOnly)).toBe(objWithLoaderOnly)

    const fnWithSetup = Object.assign(() => null, { setup: () => null })
    const fnWithRender = Object.assign(() => null, { render: () => null })
    const fnWithProps = Object.assign(() => null, { props: ['a'] })
    const fnWithName = Object.assign(() => null, { __name: 'Named' })
    const fnWithArity = (props: unknown) => props
    expect(parsePluginComponent(fnWithSetup)).toBe(fnWithSetup)
    expect(parsePluginComponent(fnWithRender)).toBe(fnWithRender)
    expect(parsePluginComponent(fnWithProps)).toBe(fnWithProps)
    expect(parsePluginComponent(fnWithName)).toBe(fnWithName)
    expect(parsePluginComponent(fnWithArity)).toBe(fnWithArity)

    const asyncObj = { type: 'async', loader: () => Promise.resolve(syncComp) }
    expect(parsePluginComponent(asyncObj)).not.toBe(asyncObj)

    const nonLoaderAsync = { type: 'async', loader: 'invalid' }
    expect(parsePluginComponent(nonLoaderAsync)).toBe(nonLoaderAsync)

    expect(parseTitle('Plain')).toBe('Plain')
    expect(parseTitle({ value: 'RefVal' })).toBe('RefVal')
    expect(parseTitle(null)).toBe('')
    expect(parseTitle(42)).toBe('')

    expect(extractFilename(undefined)).toBe('')
    expect(extractFilename('')).toBe('')
    expect(extractFilename('folder/note.md')).toBe('note.md')

    expect(stripMarkdownExtension('note.md')).toBe('note')
    expect(stripMarkdownExtension('note.md.bak')).toBe('note.md.bak')

    expect(parseStatePath(null)).toBe('')
    expect(parseStatePath('str')).toBe('')
    expect(parseStatePath(Object.assign(() => null, { path: 'fn.md' }))).toBe('')
    expect(parseStatePath({ path: 123 })).toBe('')
    expect(parseStatePath({ path: 'a.md' })).toBe('a.md')
    expect(parseStatePath({ file: 'b.md' })).toBe('b.md')
    expect(parseStatePath({ at: 'c.md' })).toBe('c.md')
  })

  it('copies and adopts host stylesheets or falls back to cloning head nodes', () => {
    class MockSheet {
      text = ''
      baseURL?: string | undefined
      constructor(opts?: { baseURL?: string }) {
        this.baseURL = opts?.baseURL
      }
      replaceSync(t: string) {
        this.text = t
      }
    }
    vi.stubGlobal('CSSStyleSheet', MockSheet)
    Object.defineProperty(document, 'adoptedStyleSheets', {
      value: [],
      writable: true,
      configurable: true,
    })

    const headStyle = document.createElement('style')
    headStyle.textContent = '.host { color: blue; }'
    document.head.appendChild(headStyle)

    const pluginStyle = document.createElement('style')
    pluginStyle.setAttribute('data-plugin-id', 'p1')
    document.head.appendChild(pluginStyle)

    const bodyStyle = document.createElement('style')
    document.body.appendChild(bodyStyle)

    const normalSheet = {
      disabled: false,
      ownerNode: headStyle,
      cssRules: [{ cssText: '.host { color: blue; }' }, { cssText: '.b { margin: 0; }' }],
      media: { length: 0, mediaText: '' },
      href: null,
    } as unknown as CSSStyleSheet
    const mediaSheet = {
      disabled: false,
      ownerNode: null,
      cssRules: [{ cssText: '.media { display: block; }' }],
      media: { length: 1, mediaText: 'screen' },
      href: 'https://example.org/app.css',
    } as unknown as CSSStyleSheet
    const disabledSheet = {
      disabled: true,
      ownerNode: headStyle,
      cssRules: [],
      media: { length: 0, mediaText: '' },
    } as unknown as CSSStyleSheet
    const pluginOwnedSheet = {
      disabled: false,
      ownerNode: pluginStyle,
      cssRules: [],
      media: { length: 0, mediaText: '' },
    } as unknown as CSSStyleSheet
    const bodyOwnedSheet = {
      disabled: false,
      ownerNode: bodyStyle,
      cssRules: [],
      media: { length: 0, mediaText: '' },
    } as unknown as CSSStyleSheet

    const sheetsSpy = vi
      .spyOn(document, 'styleSheets', 'get')
      .mockReturnValue([
        normalSheet,
        mediaSheet,
        disabledSheet,
        pluginOwnedSheet,
        bodyOwnedSheet,
      ] as unknown as StyleSheetList)

    try {
      const fakeShadow = {
        adoptedStyleSheets: [] as CSSStyleSheet[],
        appendChild: vi.fn(),
      } as unknown as ShadowRoot

      adoptHostStyles(fakeShadow)
      expect(fakeShadow.adoptedStyleSheets.length).toBe(2)
      const firstCopy = fakeShadow.adoptedStyleSheets[0] as unknown as MockSheet
      const secondCopy = fakeShadow.adoptedStyleSheets[1] as unknown as MockSheet
      expect(firstCopy.text).toBe('.host { color: blue; }\n.b { margin: 0; }')
      expect(secondCopy.text).toBe('@media screen {\n.media { display: block; }\n}')
      expect(secondCopy.baseURL).toBe('https://example.org/app.css')

      // Second adoption reuses cached copy
      const secondShadow = {
        adoptedStyleSheets: [] as CSSStyleSheet[],
        appendChild: vi.fn(),
      } as unknown as ShadowRoot
      adoptHostStyles(secondShadow)
      expect(secondShadow.adoptedStyleSheets[0]).toBe(fakeShadow.adoptedStyleSheets[0])

      // When a sheet throws on cssRules, falls back to cloning head styles
      const throwingSheet = {
        disabled: false,
        ownerNode: null,
        get cssRules(): never {
          throw new Error('cors')
        },
      } as unknown as CSSStyleSheet
      sheetsSpy.mockReturnValue([throwingSheet] as unknown as StyleSheetList)

      const fallbackShadow = {
        adoptedStyleSheets: [] as CSSStyleSheet[],
        appendChild: vi.fn(),
      } as unknown as ShadowRoot
      adoptHostStyles(fallbackShadow)
      expect(fallbackShadow.appendChild).toHaveBeenCalled()
      expect(
        (fallbackShadow.appendChild as ReturnType<typeof vi.fn>).mock.calls[0]?.[0]?.textContent,
      ).toBe('.host { color: blue; }')

      const realHostDiv = document.createElement('div')
      const attachedShadow = attachPluginRoot(realHostDiv)
      expect(attachedShadow?.querySelector('style')?.textContent).toBe('.host { color: blue; }')

      // attachPluginRoot edge cases
      expect(attachPluginRoot(null)).toBeNull()
      const existingShadow = {} as ShadowRoot
      expect(attachPluginRoot({ shadowRoot: existingShadow } as HTMLElement)).toBe(existingShadow)
      let noAttachReads = 0
      expect(
        attachPluginRoot({
          get shadowRoot() {
            return noAttachReads++ > 0 ? existingShadow : null
          },
        } as unknown as HTMLElement),
      ).toBeNull()
      let lateShadowReads = 0
      expect(
        attachPluginRoot({
          get shadowRoot() {
            return lateShadowReads++ > 0 ? existingShadow : null
          },
          attachShadow: () => {
            throw new Error('already attached')
          },
        } as unknown as HTMLElement),
      ).toBe(existingShadow)
      const nonShadowEl = {
        shadowRoot: null,
        attachShadow: () => {
          throw new Error('fail')
        },
      } as unknown as HTMLElement
      expect(attachPluginRoot(nonShadowEl)).toBeNull()
    } finally {
      sheetsSpy.mockRestore()
      headStyle.remove()
      pluginStyle.remove()
      bodyStyle.remove()
      vi.unstubAllGlobals()
      Reflect.deleteProperty(document, 'adoptedStyleSheets')
    }
  })

  it('renders wrapper, pane content, adopted styles, and fallback VNodes', async () => {
    const fallback = renderFallback('my.plugin', new Error('crash detail'))
    expect(fallback.props?.class).toBe('plugin-error-fallback p-4 text-red-500')
    const children = fallback.children as Array<{
      props?: Record<string, unknown>
      children?: unknown
    }>
    expect(children[0]?.props?.class).toBe('font-bold')
    expect(children[0]?.children).toBe('Plugin "my.plugin" crashed')
    expect(children[1]?.props?.class).toBe('text-xs mt-2')
    expect(children[1]?.children).toBe('crash detail')

    expect(
      renderPaneContent({
        pluginId: 'p',
        component: null,
        state: {},
        shadowRoot: null,
        containerClass: 'c',
      }),
    ).toBeNull()

    registerPluginStyle('p', '/p.css')
    const Comp = defineComponent({ setup: () => () => h('span', 'hi') })
    const adoptedNodes = renderPaneContent({
      pluginId: 'p',
      component: Comp,
      state: {},
      shadowRoot: null,
      containerClass: 'custom-container',
      hasAdoptedStyle: true,
    }) as Array<{ type?: unknown; props?: Record<string, unknown>; children?: unknown }>
    expect(adoptedNodes[0]?.type).toBe('style')
    expect(adoptedNodes[0]?.children).toBe(PANE_LAYOUT_STYLE)
    expect(adoptedNodes[1]).toBeNull()
    expect(adoptedNodes[2]?.props?.class).toBe(
      'custom-container h-full w-full min-h-0 min-w-0 flex flex-col flex-1',
    )
    clearPluginStyles()

    // 1-arg createPluginPaneWrapper and non-Error throw
    let singleCompSetups = 0
    const CountingComp = defineComponent({
      setup: () => {
        singleCompSetups++
        return () => h('span', 'hi')
      },
    })
    const SingleArgWrapper = createPluginPaneWrapper('plugin', CountingComp)
    const mountedSingle = mount(SingleArgWrapper, { props: { state: 'primitive' } })
    await mountedSingle.vm.$nextTick()
    expect(singleCompSetups).toBe(1)
    expect(mountedSingle.attributes('data-plugin-id')).toBe('plugin')
    expect(mountedSingle.classes()).toContain('plugin-pane-plugin')
    expect(mountedSingle.element.shadowRoot?.textContent).toContain('hi')
    mountedSingle.unmount()

    const StringThrower = defineComponent({
      setup: () => () => {
        throw 'string error'
      },
    })
    const ThrowWrapper = createPluginPaneWrapper('a/b:c', StringThrower)
    const mountedThrow = mount(ThrowWrapper, { props: { state: {} } })
    await mountedThrow.vm.$nextTick()
    expect(mountedThrow.text()).toContain('string error')
    mountedThrow.unmount()

    // Adopted style in createPluginPaneWrapper
    class FakeSheet {
      cssText = ''
      replaceSync(text: string) {
        this.cssText = text
      }
    }
    vi.stubGlobal('CSSStyleSheet', FakeSheet)
    Object.defineProperty(document, 'adoptedStyleSheets', {
      value: [],
      writable: true,
      configurable: true,
    })
    Object.defineProperty(ShadowRoot.prototype, 'adoptedStyleSheets', {
      value: [],
      writable: true,
      configurable: true,
    })
    try {
      await loadPluginStyle(
        'styled.pane',
        '/styled.css',
        async () => ({ ok: true, text: async () => '.a {}' }) as unknown as Response,
      )
      const StyledWrapper = createPluginPaneWrapper('styled.pane', Comp)
      const mountedStyled = mount(StyledWrapper, {
        props: { state: {} },
        attachTo: document.body,
      })
      await mountedStyled.vm.$nextTick()
      const sr = mountedStyled.element.shadowRoot!
      expect(sr.adoptedStyleSheets.length).toBeGreaterThan(0)
      expect(sr.querySelector('link')).toBeNull()
      expect(sr.querySelector('.plugin-view-styled-pane')).not.toBeNull()
      mountedStyled.unmount()
      expect(sr.adoptedStyleSheets).toHaveLength(0)

      // When attachShadow fails (shadowRoot is null), mounting does not attempt to adopt plugin style onto null
      const attachSpy = vi.spyOn(HTMLElement.prototype, 'attachShadow').mockImplementation(() => {
        throw new Error('no shadow')
      })
      const mountedNoShadow = mount(StyledWrapper, { props: { state: {} } })
      await mountedNoShadow.vm.$nextTick()
      expect(mountedNoShadow.text()).toContain('hi')
      mountedNoShadow.unmount()
      attachSpy.mockRestore()
    } finally {
      clearPluginStyles()
      vi.unstubAllGlobals()
      Reflect.deleteProperty(document, 'adoptedStyleSheets')
      Reflect.deleteProperty(ShadowRoot.prototype, 'adoptedStyleSheets')
    }
  })
})
