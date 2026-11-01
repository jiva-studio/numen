/**
 * The window appearance: the theme applied, which half of a color pair its
 * tokens are read as, and the interface and reading-text sizes.
 *
 * A theme and a mode are applied as soon as the keyboard focuses them. A size
 * is applied after the keyboard has rested on the row for HELD, and the
 * settings values are restored as soon as the keyboard leaves the list.
 */
import { formatErrorMessage } from '@numen/core-client'
import { ref, shallowRef, watch, type Ref, type ShallowRef } from 'vue'
import { createFollower } from '@numen/ui'
import { createRequestGuard } from '@/shared/requestGuard'
import type { MessageWriter } from '@/shared/notices/messages'
import { DESIGNED, EMPTY_RANGE, SCHEMES } from '@/entities/settings'
import type { Appearance, Mode, Ranges, Sizes, Theme, Themes } from '@/entities/settings'
import type { AppearanceWords } from '../words'
import { IS_SIZES, addAfter, getSizesCss, getStyleElements } from '../lib/head'
import { getModeGroups, getSizeGroups, getThemeGroups, type PluginNamer } from '../lib/offers'
import { useAppearanceChoice } from './choice'

export interface WindowAppearanceDeps {
  core: Themes
  words: AppearanceWords
  write: MessageWriter
  /** The name a plugin is listed under, given its id. */
  getPluginName?: PluginNamer
  sheet?: Document
  wait?: (ms: number) => Promise<unknown>
}

interface ThemeApplierDeps {
  core: Themes
  styleElements: ReturnType<typeof getStyleElements>
  write: MessageWriter
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
  write: MessageWriter
  words: AppearanceWords
  list: ShallowRef<readonly Theme[]>
  applied: Ref<string>
  mode: Ref<Mode>
  settings: Ref<Sizes>
  bounds: Ref<Ranges>
  files: Map<string, string>
  served: string
}

function createAppearanceLoader(deps: LoaderDeps) {
  let isInitialLoad = true
  return async () => {
    let answer: Appearance
    try {
      answer = await deps.core.getAppearance()
    } catch {
      deps.write(deps.words.unlisted, 'error')
      return
    }
    deps.list.value = answer.themes
    deps.applied.value = answer.applied
    deps.mode.value = answer.mode
    deps.settings.value = answer.sizes
    deps.bounds.value = answer.bounds
    if (isInitialLoad) {
      deps.files.set(answer.applied, deps.served)
    }
    isInitialLoad = false
  }
}

function createThemeWatcher(deps: {
  core: WindowAppearanceDeps['core']
  files: Map<string, string>
  choice: ReturnType<typeof useAppearanceChoice>
  applyAppearance: () => Promise<void>
  loadAppearance: () => Promise<void>
  lost: Ref<string>
  wait: (ms: number) => Promise<unknown>
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
        await deps.loadAppearance()
        if (names.includes(deps.choice.current.value)) await deps.applyAppearance()
      },
    )

  const stop = () => {
    open = false
    abortController.abort()
  }

  return { follow, stop }
}

function initAppearanceState() {
  const list = shallowRef<readonly Theme[]>([])
  const applied = ref('')
  const mode = ref<Mode>('system')
  const settings = ref<Sizes>({ interfaceScale: DESIGNED, textScale: DESIGNED })
  const bounds = ref<Ranges>({ interfaceScale: EMPTY_RANGE, textScale: EMPTY_RANGE })
  const missing = ref('')
  return { list, applied, mode, settings, bounds, lost: missing }
}

function createAppearanceGroups(
  state: ReturnType<typeof initAppearanceState>,
  choice: ReturnType<typeof useAppearanceChoice>,
  words: WindowAppearanceDeps['words'],
  getPluginName: PluginNamer | undefined,
) {
  return {
    getThemeGroups: () =>
      getThemeGroups(state.list.value, state.applied.value, words, getPluginName),
    getModeGroups: () => getModeGroups(state.mode.value, choice.isPinned.value, words),
    // Stryker disable next-line StringLiteral: Default empty text evaluates to null in parseSize.
    getSizeGroups: (command: string, text = '') =>
      getSizeGroups(command, text, {
        sizes: state.settings.value,
        bounds: state.bounds.value,
        words,
      }),
  }
}

export function createWindowAppearance(deps: WindowAppearanceDeps) {
  const { core, words, write, getPluginName, sheet = document, wait = sleep } = deps
  const styleElements = getStyleElements(sheet)
  const served = styleElements.theme.textContent ?? ''
  const state = initAppearanceState()
  const { files, applyTheme } = createThemeApplier({
    core,
    styleElements,
    write,
    words,
    applied: state.applied,
  })

  const choice = useAppearanceChoice({
    core,
    write,
    applyTheme,
    list: state.list,
    applied: state.applied,
    mode: state.mode,
    settings: state.settings,
    bounds: state.bounds,
  })

  const applyAppearance = () => applyTheme(choice.current.value, choice.half.value)
  const writeSizes = createSizesWriter(styleElements, sheet)
  const targetDrawing = watch(choice.sized, writeSizes)

  const loadAppearance = createAppearanceLoader({
    core,
    write,
    words,
    files,
    served,
    list: state.list,
    applied: state.applied,
    mode: state.mode,
    settings: state.settings,
    bounds: state.bounds,
  })

  const watcher = createThemeWatcher({
    core,
    files,
    choice,
    applyAppearance,
    loadAppearance,
    lost: state.lost,
    wait,
  })

  return {
    ...state,
    sized: choice.sized,
    isPinned: choice.isPinned,
    pinned: choice.isPinned,
    ...createAppearanceGroups(state, choice, words, getPluginName),
    previewItem: choice.previewItem,
    chooseItem: choice.chooseItem,
    start: async () => {
      await loadAppearance()
      watcher.follow().catch((error: unknown) => write(formatErrorMessage(error), 'error'))
    },
    close: () => {
      watcher.stop()
      choice.close()
      targetDrawing()
    },
  }
}

const sleep = (ms: number) => new Promise((resolve) => setTimeout(resolve, ms))
