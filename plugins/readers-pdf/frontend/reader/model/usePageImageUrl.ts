import type { Ref } from 'vue'

export interface PageImageSources {
  readonly getPicture: () => (page: number) => string
  readonly drawnWidth: Readonly<Ref<number>>
  readonly sharpPages: Readonly<Ref<ReadonlySet<number>>>
}

/** The address of the picture of a page, empty while the page is not due to be sharp or no width is known to ask it at. */
export function usePageImageUrl({ getPicture, drawnWidth, sharpPages }: PageImageSources) {
  return (page: number): string =>
    drawnWidth.value > 0 && sharpPages.value.has(page) ? getPicture()(page) : ''
}
