import { defineComponent, h } from 'vue'
import { fn } from 'storybook/test'
import { encode } from 'uqr'
import type { LinkInfo } from '@/entities/identity'
import type { LinkCalls } from '../lib/types'

export const QR_TEXT =
  'numen://link?v=1&c=AAAAAAAAAAAAAAAAAAAAAA&k=BBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBB&s=CCCCCCCCCCCCCCCCCCCCCC'

const WINDOW_FRAME =
  'position:relative;inline-size:100vw;block-size:100vh;background:var(--numen-surface);color:var(--numen-ink);font-family:var(--numen-font-sans)'

/** The window the dialogs are drawn in: they are centred in all of it. */
export const WindowFrame = defineComponent({
  setup: (_, { slots }) => () => h('div', { class: 'numen', style: WINDOW_FRAME }, slots['default']?.()),
})

const INFO: LinkInfo = {
  phase: 'waitingForPeer',
  mode: 'show',
  role: 'host',
  qrText: QR_TEXT,
  peerName: '',
  peerFingerprint: '',
  accountFingerprint: '3F2A 91C0 7B15 22D4 90AA 11C3 5E6F 0B87',
  reason: undefined,
  deviceName: 'Studio laptop',
  deviceFingerprint: 'D5AF 4DDB 6D92 FA46 0390 D870 1186 EB41',
}

export const createInfo = (overrides: Partial<LinkInfo> = {}): LinkInfo =>
  Object.assign({}, INFO, overrides)

const ok = <T>(value: T) => Promise.resolve({ ok: true as const, value })

export const createCalls = (link: LinkInfo): LinkCalls => ({
  start: fn(() => ok(link)),
  get: fn(() => ok(link)),
  decide: fn(() => ok(undefined)),
  confirmJoin: fn(() => ok(undefined)),
  cancel: fn(() => ok(undefined)),
})

export const createRefusal = (text: string): LinkCalls =>
  Object.assign({}, createCalls(createInfo()), {
    start: fn(() => Promise.resolve({ ok: false as const, error: text })),
  })

const CELL = 8
const QUIET = 4
const FRAME_RATE = 8

export const createCodeCamera = (text: string) => (): Promise<MediaStream> => {
  const { data: modules } = encode(text, { ecc: 'M', border: 0 })
  const canvas = document.createElement('canvas')
  canvas.width = canvas.height = (modules.length + 2 * QUIET) * CELL
  const drawing = canvas.getContext('2d')
  if (drawing !== null) {
    drawing.fillStyle = '#ffffff'
    drawing.fillRect(0, 0, canvas.width, canvas.height)
    drawing.fillStyle = '#000000'
    modules.forEach((row, y) =>
      row.forEach((isDark, x) => {
        if (isDark) drawing.fillRect((x + QUIET) * CELL, (y + QUIET) * CELL, CELL, CELL)
      }),
    )
  }
  return Promise.resolve(canvas.captureStream(FRAME_RATE))
}
