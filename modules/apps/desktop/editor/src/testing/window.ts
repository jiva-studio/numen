/**
 * The window drawn in a document, with every port it reaches mocked away.
 *
 * A test declares what the application answers in `said`, reads what it was
 * asked in `asked`, and puts the window on the screen with `mountWindow`.
 */
import './vaultMock'
import './moduleMocks'
import { afterEach } from 'vitest'
import { mount, type VueWrapper } from '@vue/test-utils'
import { panesOf, WorkspaceLayout, type Workspace } from '@numen/ui'
import { requests, forgetRequests } from './requests'
import {
  folders,
  forgetAnswers,
  listed,
  maker,
  nameAnswer,
  outside,
  passageAnswer,
  said,
  sourceAnswer,
} from './answers'
import { book, editor, reader } from './stubs'

const App = (await import('@/app/App.vue')).default

/** A moment for whatever the window asked the vault for to come back. */
const settle = () => new Promise((done) => setTimeout(done, 0))

/** Longer than the palette debounces a keystroke before it asks the vault. */
const DEBOUNCE = 200

/**
 * Every window a test drew. A window listens for the keystrokes that open the
 * palette for as long as it is mounted, and the next test draws its own.
 */
const windows: { unmount(): void }[] = []

afterEach(() => {
  for (const window of windows.splice(0)) window.unmount()
  forgetAnswers()
  forgetRequests()
})

/**
 * The window drawn, with what each kind draws inside it stubbed. The tabs
 * themselves are the window's own, and they are what is asked about here.
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
  await settle()
  return window
}

/**
 * The window with a palette a person can type into. The palette draws itself at
 * the end of the document, and is read off the document.
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
 * What the plex calls the note it is standing on, which is what every gesture
 * it reports carries. The vault is asked about a path, and this is not one.
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
const layoutOf = (window: VueWrapper): Workspace =>
  window.findComponent(WorkspaceLayout).props('modelValue') as Workspace

/** What each pane holds, by the kind each of its tabs is filed under. */
const paneKinds = (window: VueWrapper): readonly (readonly string[])[] =>
  panesOf(layoutOf(window).root).map((one) => one.tabs.map((tab) => tab.split(':')[0] ?? ''))

/** What each tab of the window is called, in the order the strip has them. */
const tabsOf = (window: VueWrapper): readonly { id: string; title: string }[] =>
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
  layoutOf,
  listed,
  nameAnswer,
  nodeInPlex,
  outside,
  paneKinds,
  passageAnswer,
  reader,
  said,
  settle,
  sourceAnswer,
  tabsOf,
}
