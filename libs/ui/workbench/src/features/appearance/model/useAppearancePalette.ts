/**
 * The appearance choices as a palette of the window, opened and closed with Ctrl+P (Cmd+P) and read
 * again each time it opens. A choice is written by the window, then shown.
 */
import { computed, onMounted, onUnmounted, ref } from 'vue'
import { runInBackground } from '@numen/ui'
import { getFailureText, type AppearanceWriter, type Mode } from '../lib/appearance'
import {
  getAppearanceGroups,
  readAppearancePick,
  type AppearanceChoiceWords,
} from '../lib/choices'
import type { WindowAppearance } from './appearance'

export interface AppearancePaletteDeps {
  readonly appearance: Pick<WindowAppearance, 'state' | 'show'>
  readonly words: AppearanceChoiceWords
  readonly getPluginName: (pluginId: string) => string
  /** Writes a choice, and answers why it was not written; empty where it was. */
  readonly writeChoice: (theme: string, mode: Mode) => Promise<string>
  /** Reads again what the palette offers, as it opens. */
  readonly onOpen: () => Promise<void>
  readonly write: AppearanceWriter
}

const isToggleChord = (event: KeyboardEvent): boolean =>
  !event.defaultPrevented &&
  (event.ctrlKey || event.metaKey) &&
  !event.altKey &&
  !event.shiftKey &&
  event.key.toLowerCase() === 'p'

export function useAppearancePalette(deps: AppearancePaletteDeps) {
  const { appearance, words, getPluginName, write } = deps
  const { state } = appearance
  const isOpen = ref(false)
  const query = ref('')
  const groups = computed(() =>
    getAppearanceGroups(
      {
        themes: state.themes.value,
        applied: state.applied.value,
        mode: state.mode.value,
        getPluginName,
      },
      words,
      query.value,
    ),
  )

  const choose = async (theme: string, mode: Mode): Promise<void> => {
    const reason = await deps.writeChoice(theme, mode)
    if (reason !== '') {
      write(reason, 'error')
      return
    }
    await appearance.show(theme, mode)
  }

  /* -------------------------------- Handlers -------------------------------- */
  function onDismiss() {
    isOpen.value = false
  }

  function onKeyDown(event: KeyboardEvent) {
    if (!isToggleChord(event)) return
    event.preventDefault()
    if (isOpen.value) {
      onDismiss()
      return
    }
    query.value = ''
    isOpen.value = true
    runInBackground(deps.onOpen(), (error) => write(getFailureText(error), 'error'))
  }

  /** Applies the choice the palette reports; an identity of neither kind changes nothing. */
  async function onChoose(id: string) {
    isOpen.value = false
    const pick = readAppearancePick(id)
    if (pick?.kind === 'theme') await choose(pick.theme, state.mode.value)
    if (pick?.kind === 'mode') await choose(state.applied.value, pick.mode)
  }

  /* ---------------------------------- Hooks --------------------------------- */
  onMounted(() => globalThis.addEventListener('keydown', onKeyDown))
  onUnmounted(() => globalThis.removeEventListener('keydown', onKeyDown))

  return { isOpen, query, groups, onChoose, onDismiss }
}
