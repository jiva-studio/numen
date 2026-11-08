/**
 * The mounted window with every port it uses mocked.
 *
 * A test declares the application's responses in `vaultState`, reads the requests it
 * received from the recorded log, and mounts the window with `mountWindow`.
 */
import './vaultMock'
import { readWindowTasks } from './moduleMocks'
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

const settle = () => new Promise((isDone) => setTimeout(isDone, 0))

/** Longer than the palette's keystroke debounce before it queries the vault. */
const DEBOUNCE = 200

/**
 * Every window mounted by a test. A mounted window handles the keystrokes
 * that open the palette, so each is unmounted after the test.
 */
const windows: { unmount(): void }[] = []

afterEach(() => {
  for (const window of windows.splice(0)) window.unmount()
  forgetAnswers()
  forgetRequests()
})

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

const cards = (): readonly string[] =>
  readWindowTasks()
    .filter((task) => task.state === 'failed')
    .map((task) => [task.label, task.about].filter(Boolean).join(' '))

const nodeInPlex = (window: VueWrapper): string => {
  const state = window.findComponent({ name: 'PlexTab' }).props('state') as {
    picture: { value: { nodes: readonly { id: string }[] } | null }
  }
  return state.picture.value?.nodes[0]?.id ?? ''
}

const getLayout = (window: VueWrapper): Workspace =>
  window.findComponent(WorkspaceLayout).props('modelValue') as Workspace

const paneKinds = (window: VueWrapper): readonly (readonly string[])[] =>
  getPanes(getLayout(window).root).map((one) => one.tabs.map((tab) => tab.split(':')[0] ?? ''))

const getTabs = (window: VueWrapper): readonly { id: string; title: string }[] =>
  (window.findComponent(WorkspaceLayout).props('tabs')) ?? []

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
  mockVaultState as vaultState,
  mockVaultList as listed,
  settle,
  sourceAnswer,
  getTabs,
}
