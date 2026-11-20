/** The fraction of an element visible within its scrolling ancestor or the viewport. */
export function getVisibleFraction(element: Element, root: Element | null = null): Promise<number> {
  return new Promise((resolve) => {
    const observer = new IntersectionObserver(
      ([entry]) => {
        observer.disconnect()
        resolve(entry!.intersectionRatio)
      },
      { root },
    )
    observer.observe(element)
  })
}
