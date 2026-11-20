/** Wait for the reader to finish drawing its columns and moving its pages. */
export async function waitForPageLayout(
  area: HTMLElement,
  paper: HTMLElement,
  frames: number,
  timeout: number,
) {
  await document.fonts.ready
  let revision = 0
  const onLayoutChange = () => {
    revision += 1
  }
  const content = new MutationObserver(onLayoutChange)
  const size = new ResizeObserver(onLayoutChange)
  content.observe(area, { attributes: true, childList: true, subtree: true })
  size.observe(area)
  size.observe(paper)
  const deadline = performance.now() + timeout
  try {
    let stable = 0
    while (stable < frames) {
      if (performance.now() > deadline) throw new Error('the page layout did not settle')
      const before = revision
      await new Promise<void>((resolve) => requestAnimationFrame(() => resolve()))
      stable = before === revision ? stable + 1 : 0
    }
  } finally {
    size.disconnect()
    content.disconnect()
  }
}

/** Whether the reader occupies the given point on screen. */
export function isOnPage(area: HTMLElement, x: number, y: number): boolean {
  return area.contains(area.ownerDocument.elementFromPoint(x, y))
}
