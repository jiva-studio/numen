/**
 * The row of an appearance list the keyboard stands on, and what standing on it
 * or choosing it does.
 *
 * A theme and a mode are worn while the keyboard is on their row. A size is
 * worn once the keyboard has stood on it for HELD, and what the settings name
 * is put back the moment the keyboard leaves the list.
 */
import { computed, ref } from 'vue'
import type { Ref } from 'vue'
import { isInBounds } from '@/entities/settings'
import type { Mode, Ranges, Sizes, Theme, Themes } from '@/entities/settings'
import type { MessageWriter } from '@/shared/notices/messages'
import { HELD, its, modeOf, onto, sizeOf } from '../lib/appearanceValues'
import type { ScaleChoice } from '../lib/appearanceValues'

/** What choosing a row reaches: the settings it writes, and what it wears. */
export interface ChoiceDeps {
  readonly core: Pick<Themes, 'writeAppearance'>
  readonly write: MessageWriter
  readonly list: Ref<readonly Theme[]>
  readonly applied: Ref<string>
  readonly mode: Ref<Mode>
  readonly settings: Ref<Sizes>
  readonly bounds: Ref<Ranges>
  /** The theme and the mode named, worn at once. */
  readonly wear: (theme: string, half: Mode) => Promise<void>
}

export function useAppearanceChoice(on: ChoiceDeps) {
  const { core, write, list, applied, mode, settings, bounds, wear } = on

  /** The row the keyboard is standing on, and nothing while the list is shut. */
  const stood = ref('')

  /**
   * The size the keyboard has stood on long enough for the window to be drawn
   * at it, and nothing while it stands anywhere else.
   */
  const holding = ref<ScaleChoice | null>(null)
  let hold: ReturnType<typeof setTimeout> | undefined

  /** Whether a row names a theme, which the rows of the other lists do not. */
  const isTheme = (item: string): boolean => item !== '' && !modeOf(item) && !sizeOf(item)

  /** The theme worn now: the one the keyboard is on, else the one applied. */
  const worn = computed(() => (isTheme(stood.value) ? stood.value : applied.value))

  /** The mode read now, the same way. */
  const half = computed<Mode>(() => modeOf(stood.value) ?? mode.value)

  /** The sizes the window is drawn at now: the one being held, over the settings'. */
  const sized = computed<Sizes>(() => {
    const held = holding.value
    return held ? onto(settings.value, held.which, held.size) : settings.value
  })

  /** Whether the theme worn declares light and dark itself. */
  const isPinned = computed(() => {
    const theme = list.value.find((one) => one.name === worn.value)
    return theme?.isPinned ?? false
  })

  /** What the window wears now, which is the row stood on over the settings. */
  const applyAppearance = () => wear(worn.value, half.value)

  /**
   * The row the keyboard is standing on, worn while it stands there. A size is
   * drawn only once the keyboard has stood on it for HELD. Nothing standing
   * puts back what the settings name.
   */
  const previewItem = (item: string) => {
    stood.value = item
    clearTimeout(hold)
    const size = sizeOf(item)
    if (size && isInBounds(its(bounds.value, size.which), size.size)) {
      hold = setTimeout(() => (holding.value = size), HELD)
      return
    }
    holding.value = null
    void applyAppearance()
  }

  /**
   * The size chosen: the window is drawn at it at once, whatever the hold was
   * waiting for, and it is written into the settings beside the theme. A number
   * the settings refuse is said, and the window goes back to the size they hold.
   */
  const chooseSize = async (choice: ScaleChoice) => {
    if (!isInBounds(its(bounds.value, choice.which), choice.size)) return
    const was = settings.value
    write('')
    clearTimeout(hold)
    holding.value = null
    stood.value = ''
    settings.value = onto(was, choice.which, choice.size)

    const failed = await core.writeAppearance(applied.value, mode.value, settings.value)
    if (!failed) return
    write(failed, 'error')
    settings.value = was
  }

  /**
   * The row chosen: it is worn at once and written into the settings. Settings
   * that could not be written say so, and the window wears what they hold.
   */
  const chooseItem = async (item: string) => {
    const size = sizeOf(item)
    if (size) return await chooseSize(size)
    const was = { applied: applied.value, mode: mode.value }
    const chosen = modeOf(item)
    if (!chosen && !list.value.some((one) => one.name === item)) return
    if (chosen && isPinned.value) return
    write('')
    applied.value = chosen ? was.applied : item
    mode.value = chosen ?? was.mode
    stood.value = ''
    await applyAppearance()

    const failed = await core.writeAppearance(applied.value, mode.value, settings.value)
    if (!failed) return
    write(failed, 'error')
    applied.value = was.applied
    mode.value = was.mode
    await applyAppearance()
  }

  const close = () => clearTimeout(hold)

  return { worn, half, sized, isPinned, previewItem, chooseItem, close }
}
