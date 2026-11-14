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
import { createFollower, createRequestGuard, doc, type Caught } from '@numen/ui'
import {
  DESIGNED,
  SCHEMES,
  getFailureText,
  type Appearance,
  type AppearanceWords,
  type AppearanceWriter,
  type Mode,
  type Sizes,
  type Theme,
  type Themes,
} from '../lib/appearance'
import { IS_SIZES, addAfter, getSizesCss, getStyleElements } from '../api/head'
import { createAppearancePreview } from './preview'

export interface WindowAppearanceDeps {
  core: Themes
  words: AppearanceWords
  write: AppearanceWriter
  sheet?: Document
  wait?: (ms: number) => Promise<void>
}

interface ThemeApplierDeps {
  core: Themes
  styleElements: ReturnType<typeof getStyleElements>
  write: AppearanceWriter
  words: AppearanceWords
  applied: Ref<string>
}

function createThemeApplier(deps: ThemeApplierDeps) {
  const { core, styleElements, write, words, applied } = deps
  const files = new Map<string, string>()
  const guard = createRequestGuard()

  const fileOf = async (name: string): Promise<string> => {
    const cached = files.get(name)
    if (cached !== undefined) return cached
    const css = await core.readTheme(name)
    files.set(name, css)
    return css
  }

  const applyTheme = async (theme: string, half: Mode) => {
    if (!applied.value) return
    const mine = guard.acquireRequest()
    styleElements.mode.textContent = `:root { color-scheme: ${SCHEMES[half]}; }`
    try {
      const css = await fileOf(theme)
      if (mine.isCurrent) styleElements.theme.textContent = css
    } catch {
      if (mine.isCurrent) write(words.themeUnreadable, 'error')
    }
  }

  return { files, applyTheme }
}

function createSizesWriter(styleElements: ReturnType<typeof getStyleElements>, sheet: Document) {
  // Stryker disable next-line StringLiteral: Empty initial string represents unapplied state.
  let appliedSizesCss = ''
  return (sizes: Sizes) => {
    const css = getSizesCss(sizes)
    // Stryker disable next-line ConditionalExpression: Cached css skips redundant DOM style mutations.
    if (css === appliedSizesCss) return
    appliedSizesCss = css
    styleElements.sizes ??= addAfter(styleElements.theme, IS_SIZES, sheet)
    styleElements.sizes.textContent = css
  }
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
  const guard = createRequestGuard()
  /** The first read caches the stylesheet the page was served with under the theme applied. */
  return async (isFirstRead = false) => {
    const mine = guard.acquireRequest()
    let answer: Appearance
    try {
      answer = await deps.core.getAppearance()
    } catch {
      deps.write(deps.words.unlisted, 'error')
      return
    }
    // An answer to an earlier read is not applied over the answer to a later one.
    if (!mine.isCurrent) return
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
  files: Map<string, string>
  preview: ReturnType<typeof createAppearancePreview>
  applyAppearance: () => Promise<void>
  loadAppearance: () => Promise<void>
  getChoice: () => string
  lost: Ref<string>
  wait: (ms: number) => Promise<void>
}) {
  let open = true
  const abortController = new AbortController()
  const follows = createFollower({
    isOpen: () => open,
    setLost: (removed) => {
      deps.lost.value = removed
    },
    wait: deps.wait,
  })

  const follow = () =>
    follows(
      () => deps.core.watchThemes(abortController.signal),
      async (names) => {
        if (!names.length) return
        for (const name of names) deps.files.delete(name)
        const before = deps.getChoice()
        await deps.loadAppearance()
        // A theme that vanished takes the settings to another one without a write of them.
        const isChanged = deps.getChoice() !== before
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
  const settings = ref<Sizes>({ interfaceScale: DESIGNED, textScale: DESIGNED })
  const missing = ref('')
  return { themes, applied, mode, settings, lost: missing }
}

export function createWindowAppearance(deps: WindowAppearanceDeps) {
  const { core, words, write, sheet = doc, wait = sleep } = deps
  const styleElements = getStyleElements(sheet)
  const served = styleElements.theme.textContent
  const state = initAppearanceState()
  const { files, applyTheme } = createThemeApplier({
    core,
    styleElements,
    write,
    words,
    applied: state.applied,
  })

  const preview = createAppearancePreview({
    write,
    applyTheme,
    applied: state.applied,
    mode: state.mode,
    settings: state.settings,
  })

  const applyAppearance = () =>
    applyTheme(preview.display.current.value, preview.display.half.value)
  const writeSizes = createSizesWriter(styleElements, sheet)
  const targetDrawing = watch(preview.display.sized, writeSizes)

  const loadAppearance = createAppearanceLoader({
    core,
    write,
    words,
    files,
    served,
    themes: state.themes,
    applied: state.applied,
    mode: state.mode,
    settings: state.settings,
  })

  const watcher = createThemeWatcher({
    core,
    files,
    preview,
    applyAppearance,
    loadAppearance,
    getChoice: () => `${state.applied.value} ${state.mode.value}`,
    lost: state.lost,
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
      watcher.follow().catch((error: Caught) => write(getFailureText(error), 'error'))
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
