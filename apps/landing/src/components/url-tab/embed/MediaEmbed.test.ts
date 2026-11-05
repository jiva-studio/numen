import { afterEach, expect, test } from 'vitest'
import { createApp, h, type App } from 'vue'
import MediaEmbed from './MediaEmbed.vue'
import type { HeightStore } from '../lib/storage'

let app: App | undefined

afterEach(() => {
  app?.unmount()
  app = undefined
  document.body.replaceChildren()
})

/** Draws the player in a pane of the given height, under a head that sizes itself from the player. */
function mountInPane(paneHeight: number, store: HeightStore): HTMLElement {
  const pane = document.createElement('div')
  pane.style.cssText = `container-type: size; block-size: ${paneHeight}px; inline-size: 800px; display: flex; flex-direction: column;`
  const head = document.createElement('div')
  head.style.cssText = 'flex: 0 1 auto;'
  pane.append(head)
  document.body.append(pane)
  app = createApp({
    render: () =>
      h(MediaEmbed, {
        embed: '',
        copy: 'data:video/mp4;base64,',
        words: { playing: 'Sample clip', taller: 'Resize sample' },
        store,
      }),
  })
  app.mount(head)
  return pane
}

function waitForFrames(count: number): Promise<void> {
  return new Promise((resolve) => {
    const step = (left: number) =>
      left === 0 ? resolve() : requestAnimationFrame(() => step(left - 1))
    step(count)
  })
}

test('a drawn height that fits the pane is kept once the player is measured', async () => {
  const written: number[] = []
  const pane = mountInPane(900, { read: () => 500, write: (height) => written.push(height) })
  await waitForFrames(10)

  const player = pane.querySelector('video')
  const handle = pane.querySelector('[role="separator"]')
  expect(player?.clientHeight).toBe(500)
  expect(handle?.getAttribute('aria-valuenow')).toBe('500')
  expect(handle?.getAttribute('aria-valuemax')).toBe('740')
  expect(written).toEqual([])
})
