/**
 * The theme, the half of a color pair and the size the keyboard is standing on in the list of a
 * setting that is previewed. They are applied while the keyboard is on the row and put back when the
 * keyboard exits the list.
 *
 * A theme and a half are applied as soon as the keyboard is on their row. A size is applied once the
 * keyboard has rested on its row for HELD.
 */
import { formatError, type Caught } from '@/shared/api/errors'
import { computed, ref } from 'vue'
import type { Ref } from 'vue'
import type { Mode, Sizes } from '@/entities/settings'
import type { MessageWriter } from '@/shared/model/notices/messages'
import { HELD, onto } from '../lib/appearanceValues'
import type { ScaleKind } from '../lib/appearanceValues'

/** What previewing reads, and what it applies. */
export interface PreviewDeps {
  readonly write: MessageWriter
  readonly applied: Ref<string>
  readonly mode: Ref<Mode>
  readonly settings: Ref<Sizes>
  readonly applyTheme: (theme: string, half: Mode) => Promise<void>
}

/** One size previewed, and which of the two it is. */
interface PreviewedSize {
  readonly which: ScaleKind
  readonly size: number
}

export function createAppearancePreview(on: PreviewDeps) {
  const { write, applied, mode, settings, applyTheme } = on
  const theme = ref('')
  const half = ref<Mode | ''>('')
  const pendingSize = ref<PreviewedSize | null>(null)
  let hold: ReturnType<typeof setTimeout> | undefined

  const current = computed(() => theme.value || applied.value)
  const shownHalf = computed<Mode>(() => half.value || mode.value)
  const sized = computed<Sizes>(() => {
    const previewed = pendingSize.value
    return previewed ? onto(settings.value, previewed.which, previewed.size) : settings.value
  })

  const apply = () => {
    applyTheme(current.value, shownHalf.value).catch((error: Caught) =>
      write(formatError(error), 'error'),
    )
  }

  const previewTheme = (name: string) => {
    theme.value = name
    apply()
  }

  const previewMode = (value: Mode) => {
    half.value = value
    apply()
  }

  const previewSize = (which: ScaleKind, size: number) => {
    clearTimeout(hold)
    hold = setTimeout(() => (pendingSize.value = { which, size }), HELD)
  }

  /** Forgets what the keyboard stood on. */
  const clear = () => {
    clearTimeout(hold)
    pendingSize.value = null
    theme.value = ''
    half.value = ''
  }

  /** Shows what the settings name again. */
  const restore = () => {
    clear()
    apply()
  }

  const close = () => clearTimeout(hold)

  return {
    display: { current, half: shownHalf, sized },
    keyboard: { previewTheme, previewMode, previewSize },
    clear,
    restore,
    close,
  }
}
