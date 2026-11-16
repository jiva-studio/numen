/** What the editor shows against its lines. */
export interface TimeGutterState {
  readonly times: readonly string[]
  readonly current: number
  readonly isFollowing: boolean
}
