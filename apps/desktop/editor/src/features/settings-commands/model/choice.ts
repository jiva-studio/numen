/**
 * The row of an appearance list the keyboard focus is on, and what focusing or
 * choosing it does.
 *
 * A theme and a mode are applied while the keyboard is on their row. A size is
 * applied once the keyboard has stayed on it for HELD, and what the settings
 * name is restored as soon as the keyboard leaves the list.
 */
import { formatErrorMessage } from '@numen/plugin-sdk'
import { computed, ref } from 'vue'
import type { Ref } from 'vue'
import { isInBounds } from '@/entities/settings'
import type { Mode, Ranges, Sizes, Theme, Themes } from '@/entities/settings'
import type { MessageWriter } from '@/shared/notices/messages'
import { HELD, its, parseMode, onto, parseSize } from '../lib/appearanceValues'
import type { ScaleChoice } from '../lib/appearanceValues'

/** What choosing a row affects: the settings it writes, and what it applies. */
export interface ChoiceDeps {
  readonly core: Pick<Themes, 'writeAppearance'>
  readonly write: MessageWriter
  readonly list: Ref<readonly Theme[]>
  readonly applied: Ref<string>
  readonly mode: Ref<Mode>
  readonly settings: Ref<Sizes>
  readonly bounds: Ref<Ranges>
  /** The theme and the mode named, applied at once. */
  readonly wear: (theme: string, half: Mode) => Promise<void>
}

interface WriteSizeDeps {
  readonly core: ChoiceDeps['core']
  readonly write: ChoiceDeps['write']
  readonly applied: Ref<string>
  readonly mode: Ref<Mode>
  readonly settings: Ref<Sizes>
  readonly bounds: Ref<Ranges>
}

async function writeSizeChoice(choice: ScaleChoice, deps: WriteSizeDeps, onReset: () => void) {
  if (!isInBounds(its(deps.bounds.value, choice.which), choice.size)) return
  const was = deps.settings.value
  deps.write('')
  onReset()
  deps.settings.value = onto(was, choice.which, choice.size)

  const failed = await deps.core.writeAppearance(
    deps.applied.value,
    deps.mode.value,
    deps.settings.value,
  )
  if (!failed) return
  deps.write(failed, 'error')
  deps.settings.value = was
}

export function useAppearanceChoice(on: ChoiceDeps) {
  const { core, write, list, applied, mode, settings, bounds, wear } = on
  const status = ref('')
  const pendingItem = ref<ScaleChoice | null>(null)
  let hold: ReturnType<typeof setTimeout> | undefined

  const isTheme = (item: string): boolean => item !== '' && !parseMode(item) && !parseSize(item)
  const aged = computed(() => (isTheme(status.value) ? status.value : applied.value))
  const half = computed<Mode>(() => parseMode(status.value) ?? mode.value)
  const sized = computed<Sizes>(() => {
    const scaleChoice = pendingItem.value
    return scaleChoice ? onto(settings.value, scaleChoice.which, scaleChoice.size) : settings.value
  })
  const isPinned = computed(() => {
    const theme = list.value.find((one) => one.name === aged.value)
    return theme?.isPinned ?? false
  })

  const applyAppearance = () => wear(aged.value, half.value)

  const previewItem = (item: string) => {
    status.value = item
    clearTimeout(hold)
    const size = parseSize(item)
    if (size && isInBounds(its(bounds.value, size.which), size.size)) {
      hold = setTimeout(() => (pendingItem.value = size), HELD)
      return
    }
    pendingItem.value = null
    applyAppearance().catch((error: unknown) => write(formatErrorMessage(error), 'error'))
  }

  const chooseSize = (choice: ScaleChoice) =>
    writeSizeChoice(choice, { core, write, applied, mode, settings, bounds }, () => {
      clearTimeout(hold)
      pendingItem.value = null
      status.value = ''
    })

  const chooseItem = async (item: string) => {
    const size = parseSize(item)
    if (size) return await chooseSize(size)
    const was = { applied: applied.value, mode: mode.value }
    const targetMode = parseMode(item)
    if (!targetMode && !list.value.some((one) => one.name === item)) return
    if (targetMode && isPinned.value) return
    write('')
    applied.value = targetMode ? was.applied : item
    mode.value = targetMode ?? was.mode
    status.value = ''
    await applyAppearance()

    const failed = await core.writeAppearance(applied.value, mode.value, settings.value)
    if (!failed) return
    write(failed, 'error')
    applied.value = was.applied
    mode.value = was.mode
    await applyAppearance()
  }

  const close = () => clearTimeout(hold)

  return { worn: aged, half, sized, isPinned, previewItem, chooseItem, close }
}
