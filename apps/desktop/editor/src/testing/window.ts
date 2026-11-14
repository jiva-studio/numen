/**
 * The mounted window with every port it uses mocked.
 *
 * A test declares the application's responses in `vaultState`, reads the requests it
 * received from the recorded log, and mounts the window with `mountWindow`.
 */
import { afterEach, beforeEach } from 'vitest'
import { installCoreAnswers } from '@numen/core-client/testing'
import { fileAnswers } from './mocks/file'
import { noteAnswers } from './mocks/note'
import { pluginAnswers } from './mocks/plugin'
import { searchAnswers } from './mocks/search'
import { vaultAnswers } from './mocks/vault'
import { settingsAnswers } from './mocks/settings'
import { tabAnswers } from './mocks/tab'
import { readWindowTasks, windowAnswers } from './mocks/window'
import { mergeAnswers } from './mergeAnswers'
import { mount, type VueWrapper } from '@vue/test-utils'
import { WorkspaceLayout } from '@numen/workbench'
import type { PluginHost } from '@numen/plugin-loader'
import { requests, forgetRequests } from './requests'
import {
  changed,
  forgetAnswers,
  mockVaultList,
  nameAnswer,
  outside,
  searchResultAnswer,
  mockVaultState,
  sourceAnswer,
} from './answers'
import { book, editor, reader } from './stubs'
import { openNoteTab, registerNoteTabs } from './mocks/noteTabs'
import type { WindowParts } from '@/app/window/createWindow'

const Root = (await import('@/app/createRoot')).createRoot()

const settle = () => new Promise((isDone) => setTimeout(isDone, 0))

/** Longer than the palette's keystroke debounce before it queries the vault. */
const DEBOUNCE = 200

/**
 * Every window mounted by a test. A mounted window handles the keystrokes
 * that open the palette, so each is unmounted after the test.
 */
const windows: { unmount(): void }[] = []

let restoreCore: () => void = () => undefined

beforeEach(() => {
  restoreCore = installCoreAnswers(
    mergeAnswers(
      vaultAnswers,
      windowAnswers,
      settingsAnswers,
      noteAnswers,
      fileAnswers,
      tabAnswers,
      pluginAnswers,
      searchAnswers,
    ),
  )
})

afterEach(() => {
  for (const window of windows.splice(0)) window.unmount()
  restoreCore()
  forgetAnswers()
  forgetRequests()
})

function mountStartingWindow() {
  const window = mount(Root, {
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
  lastHost = getWindowParts(window).pluginHost
  return window
}

async function mountWindow() {
  const window = mountStartingWindow()
  await settle()
  await settle()
  await new Promise((isDone) => setTimeout(isDone, 50))
  return window
}

async function mountWindowWithPalette() {
  const window = mount(Root, {
    global: {
      stubs: { Plex: true, Editor: editor, Agent: true, Reader: reader, Book: book, Tree: true },
    },
    attachTo: document.body,
  })
  windows.push(window)
  lastHost = getWindowParts(window).pluginHost
  await settle()
  await settle()
  await settle()
  return window
}

/** The plugin host of the window mounted last, whose tasks the cards of the window list. */
let lastHost: PluginHost | undefined

const cards = (): readonly string[] =>
  readWindowTasks(lastHost)
    .filter((task) => task.state === 'failed')
    .map((task) => [task.label, task.about].filter(Boolean).join(' '))

const hasWindowParts = (vm: object): vm is { parts: WindowParts } => 'parts' in vm

const getWindowParts = (window: VueWrapper) => {
  if (!hasWindowParts(window.vm)) throw new Error('The mounted window has no parts')
  const { parts } = window.vm
  return {
    tabs: parts.tabs,
    palette: {
      getTarget: parts.getTarget,
      runCommand: parts.palettes.palette.runs.runCommand,
      commandDeps: parts.palettes.commandDeps,
    },
    pluginHost: parts.views.pluginHost,
  }
}

const getPaletteField = (): HTMLInputElement | null =>
  document.body.querySelector<HTMLInputElement>('[data-palette="field"]')

const typeInPalette = async (text: string): Promise<void> => {
  const into = getPaletteField()
  if (!into) return
  into.value = text
  into.dispatchEvent(new Event('input'))
  await settle()
}

const submitPalette = async (): Promise<void> => {
  getPaletteField()?.dispatchEvent(
    new KeyboardEvent('keydown', { key: 'Enter', bubbles: true, cancelable: true }),
  )
  await settle()
}

const getTabs = (window: VueWrapper): readonly { id: string; title: string }[] =>
  window.findComponent(WorkspaceLayout).props('tabs')

export {
  requests,
  cards,
  changed,
  getWindowParts,
  submitPalette,
  typeInPalette,
  DEBOUNCE,
  mountWindow,
  mountStartingWindow,
  mountWindowWithPalette,
  nameAnswer,
  outside,
  searchResultAnswer,
  mockVaultState,
  mockVaultState as vaultState,
  mockVaultList as listed,
  settle,
  sourceAnswer,
  getTabs,
  openNoteTab,
  registerNoteTabs,
}
