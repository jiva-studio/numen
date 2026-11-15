/** One section as the window holds it, under an identity of its own. */
export interface BufferSection {
  readonly id: string
  readonly name: string
  readonly preamble: string
}
