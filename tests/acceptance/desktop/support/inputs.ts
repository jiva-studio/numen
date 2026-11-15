import type { Page, TestInfo } from 'playwright/test'

export const INPUTS = 'inputs'

const BINDING = '__numenRecordInput'

export type InputKind = 'down' | 'move' | 'up' | 'key' | 'type'

export type PageInput = {
  readonly kind: InputKind
  readonly at: number
  readonly x: number
  readonly y: number
  readonly w?: number
  readonly h?: number
  readonly width: number
  readonly height: number
  readonly key?: string
  readonly mods?: readonly string[]
  readonly isField?: boolean
  readonly text?: string
  readonly isAppend?: boolean
}

export type RecordedInput = PageInput & { readonly page: number }

const listenForInputs = (binding: string): void => {
  if (window !== window.top) return
  const moveInterval = 30
  const textLength = 40
  const modifiers = ['Meta', 'Control', 'Alt', 'Shift'] as const
  let lastMove = 0
  const send = (input: Omit<PageInput, 'at' | 'width' | 'height'>): void => {
    const record = Reflect.get(window, binding) as ((input: PageInput) => void) | undefined
    record?.({ ...input, at: Date.now(), width: window.innerWidth, height: window.innerHeight })
  }
  const getRect = (target: EventTarget | null) => {
    if (!(target instanceof Element) || target === document.body) return { x: 0, y: 0, w: 0, h: 0 }
    const rect = target.getBoundingClientRect()
    return { x: rect.left, y: rect.top, w: rect.width, h: rect.height }
  }
  const isField = (target: EventTarget | null): boolean =>
    target instanceof HTMLInputElement ||
    target instanceof HTMLTextAreaElement ||
    (target instanceof HTMLElement && target.isContentEditable)
  const getMods = (event: KeyboardEvent): string[] =>
    modifiers.filter((name) => event.getModifierState(name))
  const getInputText = (event: Event) => {
    const target = event.target
    if (target instanceof HTMLInputElement && target.type === 'password')
      return { text: '••••', isAppend: false }
    if (target instanceof HTMLInputElement || target instanceof HTMLTextAreaElement)
      return { text: target.value.slice(-textLength), isAppend: false }
    const data = event instanceof InputEvent ? (event.data ?? '') : ''
    return { text: data.slice(-textLength), isAppend: true }
  }
  const sendPointer = (kind: InputKind, event: PointerEvent): void =>
    send({ kind, x: event.clientX, y: event.clientY })
  const sendMove = (event: PointerEvent): void => {
    if (event.buttons === 0 || event.timeStamp - lastMove < moveInterval) return
    lastMove = event.timeStamp
    sendPointer('move', event)
  }
  const sendKey = (event: KeyboardEvent): void => {
    if ((modifiers as readonly string[]).includes(event.key)) return
    const target = document.activeElement
    send({
      kind: 'key',
      ...getRect(target),
      key: event.key,
      mods: getMods(event),
      isField: isField(target),
    })
  }
  const sendTyped = (event: Event): void => {
    if (!isField(event.target)) return
    send({ kind: 'type', ...getRect(event.target), ...getInputText(event) })
  }
  window.addEventListener('pointerdown', (event) => sendPointer('down', event), { capture: true })
  window.addEventListener('pointermove', sendMove, { capture: true })
  window.addEventListener('pointerup', (event) => sendPointer('up', event), { capture: true })
  window.addEventListener('keydown', sendKey, { capture: true })
  window.addEventListener('input', sendTyped, { capture: true })
}

export const recordInputs = async (
  page: Page,
  index: number,
  inputs: RecordedInput[],
): Promise<void> => {
  await page.exposeBinding(BINDING, (_source, input: PageInput) => {
    inputs.push({ page: index, ...input })
  })
  await page.addInitScript(listenForInputs, BINDING)
}

export const attachInputs = async (
  testInfo: TestInfo,
  inputs: readonly RecordedInput[],
): Promise<void> => {
  if (inputs.length === 0) return
  await testInfo.attach(INPUTS, {
    body: JSON.stringify(inputs),
    contentType: 'application/json',
  })
}
