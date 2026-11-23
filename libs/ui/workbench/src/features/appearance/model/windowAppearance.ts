/**
 * The window appearance: the theme applied, which half of a color pair its
 * tokens are read as, and the interface and reading-text sizes.
 *
 * A theme and a mode are applied as soon as the keyboard focuses them. A size
 * is applied after the keyboard has rested on the row for HELD, and the
 * settings values are restored as soon as the keyboard leaves the list. What a
 * list offers and what choosing a row writes belong to the adapter of the settings.
 */
import { ref, shallowRef, watch, type Ref } from 'vue'
import {
  getFailureText,
  type AppearanceOutput,
  type AppearanceWords,
  type AppearanceWriter,
  type CreateFollower,
  type Mode,
  type Sizes,
  type Theme,
  type ThemeSource,
  type ThemeState as Appearance,
} from '../lib/appearance'
import { createAppearancePreview } from './preview'

type Themes = ThemeSource

interface WindowAppearanceDeps {
  core: Themes
  createFollower: CreateFollower
  words: AppearanceWords
  write: AppearanceWriter
  output: AppearanceOutput
  wait?: (ms: number) => Promise<void>
}

function createThemeApplier(
  { core, output, write, words }: WindowAppearanceDeps,
  applied: Ref<string>,
) {
  const files = new Map<string, string>()
  let revision = 0

  const readThemeFile = async (name: string): Promise<string> => {
    const cached = files.get(name)
    if (cached !== undefined) return cached
    const css = await core.readTheme(name)
    files.set(name, css)
    return css
  }

  const applyTheme = async (theme: string, half: Mode) => {
    if (!applied.value) return
    const mine = ++revision
    output.writeMode(half)
    try {
      const css = await readThemeFile(theme)
      if (mine === revision) output.writeTheme(css)
    } catch {
      if (mine === revision) write(words.themeUnreadable, 'error')
    }
  }

  return { files, applyTheme }
}

interface LoaderDeps {
  core: Themes
  write: AppearanceWriter
  words: AppearanceWords
  themes: Ref<readonly Theme[]>
  applied: Ref<string>
  mode: Ref<Mode>
  settings: Ref<Sizes>
  files: Map<string, string>
  served: string
}

function createAppearanceLoader(deps: LoaderDeps) {
  let revision = 0
  /** The first read caches the stylesheet the page was served with under the theme applied. */
  return async (isFirstRead = false) => {
    const mine = ++revision
    let answer: Appearance
    try {
      answer = await deps.core.getAppearance()
    } catch {
      deps.write(deps.words.unlisted, 'error')
      return
    }
    // An answer to an earlier read is not applied over the answer to a later one.
    if (mine !== revision) return
    deps.themes.value = answer.themes
    deps.applied.value = answer.applied
    deps.mode.value = answer.mode
    deps.settings.value = answer.sizes
    if (isFirstRead) {
      deps.files.set(answer.applied, deps.served)
    }
  }
}

function createThemeWatcher(deps: {
  core: WindowAppearanceDeps['core']
  createFollower: CreateFollower
  files: Map<string, string>
  preview: ReturnType<typeof createAppearancePreview>
  applyAppearance: () => Promise<void>
  loadAppearance: () => Promise<void>
  state: ReturnType<typeof initAppearanceState>
  wait: (ms: number) => Promise<void>
}) {
  let open = true
  const abortController = new AbortController()
  const getChoice = () => `${deps.state.applied.value} ${deps.state.mode.value}`
  const follows = deps.createFollower({
    isOpen: () => open,
    setLost: (removed) => {
      deps.state.lost.value = removed
    },
    wait: deps.wait,
  })

  const follow = () =>
    follows(
      () => deps.core.watchThemes(abortController.signal),
      async (names) => {
        if (!names.length) return
        for (const name of names) deps.files.delete(name)
        const before = getChoice()
        await deps.loadAppearance()
        // A theme that vanished takes the settings to another one without a write of them.
        const isChanged = getChoice() !== before
        if (isChanged || names.includes(deps.preview.display.current.value)) {
          await deps.applyAppearance()
        }
      },
    )

  const stop = () => {
    open = false
    abortController.abort()
  }

  return { follow, stop }
}

function initAppearanceState() {
  const themes = shallowRef<readonly Theme[]>([])
  const applied = ref('')
  const mode = ref<Mode>('system')
  const settings = ref<Sizes>({ interfaceScale: 1, textScale: 1 })
  const missing = ref('')
  return { themes, applied, mode, settings, lost: missing }
}

export function createWindowAppearance(deps: WindowAppearanceDeps) {
  const { core, createFollower, words, write, output, wait = sleep } = deps
  const state = initAppearanceState()
  const { files, applyTheme } = createThemeApplier(deps, state.applied)

  const preview = createAppearancePreview({
    write,
    applyTheme,
    applied: state.applied,
    mode: state.mode,
    settings: state.settings,
  })

  const applyAppearance = () =>
    applyTheme(preview.display.current.value, preview.display.half.value)
  const targetDrawing = watch(preview.display.sized, output.writeSizes)

  const loadAppearance = createAppearanceLoader({
    core,
    write,
    words,
    files,
    served: output.servedTheme,
    themes: state.themes,
    applied: state.applied,
    mode: state.mode,
    settings: state.settings,
  })

  const watcher = createThemeWatcher({
    core,
    createFollower,
    files,
    preview,
    applyAppearance,
    loadAppearance,
    state,
    wait,
  })

  return {
    state,
    preview,
    /** Reads the theme, the mode and the sizes the settings now name, and applies them. */
    reload: async () => {
      await loadAppearance()
      preview.clear()
      await applyAppearance()
    },
    /** Applies a theme and a mode the settings now hold, written by the window itself. */
    show: async (theme: string, half: Mode) => {
      state.applied.value = theme
      state.mode.value = half
      preview.clear()
      await applyAppearance()
    },
    start: async () => {
      await loadAppearance(true)
      // Stryker disable next-line ArrowFunction: createFollower handles stream errors internally.
      watcher.follow().catch((error: unknown) => write(getFailureText(error), 'error'))
    },
    close: () => {
      watcher.stop()
      preview.close()
      targetDrawing()
    },
  }
}

/** The appearance of one window: its state, its preview, and how it is read and applied. */
export type WindowAppearance = ReturnType<typeof createWindowAppearance>

const sleep = (ms: number) => new Promise<void>((resolve) => setTimeout(resolve, ms))
