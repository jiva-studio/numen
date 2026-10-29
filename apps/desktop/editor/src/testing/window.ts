/**
 * The mounted window with every port it uses mocked.
 *
 * A test declares the application's responses in `said`, reads the requests it
 * received in `asked`, and mounts the window with `mountWindow`.
 */
import './vaultMock'
import './moduleMocks'
import { afterEach } from 'vitest'
import { mount, type VueWrapper } from '@vue/test-utils'
import { getPanes, WorkspaceLayout, type Workspace } from '@numen/shell'
import { requests, forgetRequests } from './requests'
import {
  folders,
  forgetAnswers,
  mockVaultList,
  maker,
  nameAnswer,
  outside,
  passageAnswer,
  mockVaultState,
  sourceAnswer,
} from './answers'
import { book, editor, reader } from './stubs'

const App = (await import('@/app/shell/App.vue')).default

/** Waits for the vault requests issued by the window to complete. */
const settle = () => new Promise((isDone) => setTimeout(isDone, 0))

/** Longer than the palette's keystroke debounce before it queries the vault. */
const DEBOUNCE = 200

/**
 * Every window mounted by a test. A mounted window listens for the keystrokes
 * that open the palette, so each is unmounted after the test.
 */
const windows: { unmount(): void }[] = []

afterEach(() => {
  for (const window of windows.splice(0)) window.unmount()
  forgetAnswers()
  forgetRequests()
})

/**
 * The window mounted with stubs for the content of each tab kind. The tabs
 * themselves are the window's own, and they are under test.
 */
async function mountWindow() {
  const window = mount(App, {
    global: {
      // A stub is named by the binding the component is drawn through, and the
      // window's command palette draws the library's under `Palette`.
      stubs: {
        Plex: true,
        Editor: editor,
        Agent: true,
        Reader: reader,
        Book: book,
        Palette: true,
        Tree: true,
      },
    },
    // Drawn in the document, because what holds the keyboard is a question only
    // a window standing in one can answer.
    attachTo: document.body,
  })
  windows.push(window)
  await settle()
  await settle()
  await new Promise((isDone) => setTimeout(isDone, 50))
  return window
}

/**
 * The window with a palette the test can type into. The palette renders at the
 * end of the document and is queried from the document.
 */
async function mountWindowWithPalette() {
  const window = mount(App, {
    global: {
      stubs: { Plex: true, Editor: editor, Agent: true, Reader: reader, Book: book, Tree: true },
    },
    attachTo: document.body,
  })
  windows.push(window)
  await settle()
  await settle()
  await settle()
  return window
}

/** What the corner of the window is saying, one string per card. */
const cards = (window: VueWrapper): readonly string[] =>
  window.findAll('article.notice').map((card) => card.text())

/**
 * The id the plex gives the note it is focused on, carried by every gesture it
 * reports. The vault is queried by path, and this is not a path.
 *
 * The tab is found by the name the window addresses it by, so this harness
 * holds no screen's file.
 */
const nodeInPlex = (window: VueWrapper): string => {
  const state = window.findComponent({ name: 'PlexTab' }).props('state') as {
    picture: { value: { nodes: readonly { id: string }[] } | null }
  }
  return state.picture.value?.nodes[0]?.id ?? ''
}

/** How the window is split, as the workspace it draws has it. */
const getLayout = (window: VueWrapper): Workspace =>
  window.findComponent(WorkspaceLayout).props('modelValue') as Workspace

/** What each pane holds, by the kind each of its tabs is filed under. */
const paneKinds = (window: VueWrapper): readonly (readonly string[])[] =>
  getPanes(getLayout(window).root).map((one) => one.tabs.map((tab) => tab.split(':')[0] ?? ''))

/** What each tab of the window is called, in the order the strip has them. */
const getTabs = (window: VueWrapper): readonly { id: string; title: string }[] =>
  (window.findComponent(WorkspaceLayout).props('tabs') as readonly {
    id: string
    title: string
  }[]) ?? []

export {
  requests,
  cards,
  maker,
  DEBOUNCE,
  mountWindow,
  mountWindowWithPalette,
  editor,
  folders,
  getLayout,
  mockVaultList,
  nameAnswer,
  nodeInPlex,
  outside,
  paneKinds,
  passageAnswer,
  reader,
  mockVaultState,
  mockVaultState as said,
  mockVaultList as listed,
  settle,
  sourceAnswer,
  getTabs,
}
