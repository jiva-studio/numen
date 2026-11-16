import type { Size } from './arrange'

/** What settling where and how wide the headings are drawn needs to know. */
export interface HeadingsDeps {
  readonly measure?: ((text: string) => number) | undefined
  readonly viewport: Size
  readonly margin: number
}
