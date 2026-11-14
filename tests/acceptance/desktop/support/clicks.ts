import type { Page, TestInfo } from 'playwright/test'

export const CLICKS = 'clicks'

const BINDING = '__numenRecordClick'

export type RecordedClick = {
  readonly page: number
  readonly at: number
  readonly x: number
  readonly y: number
  readonly width: number
  readonly height: number
}

type PageClick = Omit<RecordedClick, 'page'>

const listenForClicks = (binding: string): void => {
  if (window !== window.top) return
  const send = (event: PointerEvent): void => {
    const record = Reflect.get(window, binding) as ((click: PageClick) => void) | undefined
    record?.({
      at: Date.now(),
      x: event.clientX,
      y: event.clientY,
      width: window.innerWidth,
      height: window.innerHeight,
    })
  }
  window.addEventListener('pointerdown', send, { capture: true })
}

export const recordClicks = async (
  page: Page,
  index: number,
  clicks: RecordedClick[],
): Promise<void> => {
  await page.exposeBinding(BINDING, (_source, click: PageClick) => {
    clicks.push({ page: index, ...click })
  })
  await page.addInitScript(listenForClicks, BINDING)
}

export const attachClicks = async (
  testInfo: TestInfo,
  clicks: readonly RecordedClick[],
): Promise<void> => {
  if (clicks.length === 0) return
  await testInfo.attach(CLICKS, {
    body: JSON.stringify(clicks),
    contentType: 'application/json',
  })
}
