/** Screen measurements of the tree list and row height. */
export interface TreeMetrics {
  readonly hoverRect: {
    readonly left: number
    readonly right: number
    readonly top: number
    readonly bottom: number
  }
  readonly top: number
  readonly height: number
}
