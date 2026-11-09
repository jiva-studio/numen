/** Invented slot entries and scales for the stories of the bar. Not shipped. */
import { h, markRaw, type FunctionalComponent } from 'vue'
import type { SlotCommand, SlotSeparator, SlotView } from '@numen/plugin-sdk'

const DUNE = 'org.example.dune'

/** An icon that draws an empty 14px mark. */
const MARK: FunctionalComponent = markRaw(() =>
  h('svg', { 'data-icon': 'mark', width: 14, height: 14 }),
)

export const command = (id: string, extra: Partial<SlotCommand> = {}): SlotCommand => ({
  kind: 'command',
  key: `${DUNE}/command/${id}`,
  pluginId: DUNE,
  command: `${DUNE}:${id}`,
  title: `Open ${id}`,
  icon: MARK,
  shortcut: '',
  hasItems: false,
  ...extra,
})

export const view = (id: string, component: unknown): SlotView => ({
  kind: 'view',
  key: `${DUNE}/view/${id}`,
  pluginId: DUNE,
  view: id,
  component,
})

export const separator = (index: number): SlotSeparator => ({
  kind: 'separator',
  key: `separator/${index}`,
})

/** A view that draws a box of a fixed size in pixels, which a story finds by `data-box`. */
export const createBoxView = (id: string): FunctionalComponent =>
  markRaw(() =>
    h('span', { 'data-box': id, style: 'display: inline-block; inline-size: 40px; block-size: 20px' }),
  )

/** The scales, as [interface, text]: the designed size, each one larger, and both larger. */
export const SCALES: readonly (readonly [string, string])[] = [
  ['1', '1'],
  ['2', '1'],
  ['1', '2'],
  ['2', '2'],
]

export const setScales = (interfaceScale: string, textScale: string): void => {
  const style = document.documentElement.style
  style.setProperty('--numen-interface-scale', interfaceScale)
  style.setProperty('--numen-text-scale', textScale)
}

export const resetScales = (): void => {
  const style = document.documentElement.style
  style.removeProperty('--numen-interface-scale')
  style.removeProperty('--numen-text-scale')
}

/** Runs `measure` once at each scale and returns the readings in the order of `SCALES`. */
export const measureAtEveryScale = <T>(measure: () => T): T[] => {
  const found: T[] = []
  try {
    for (const [interfaceScale, textScale] of SCALES) {
      setScales(interfaceScale, textScale)
      found.push(measure())
    }
  } finally {
    resetScales()
  }
  return found
}
