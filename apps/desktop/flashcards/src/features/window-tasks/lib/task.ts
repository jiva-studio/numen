/** A piece of work the core is doing for the window. */
export interface WindowTask {
  readonly id: string
  readonly label: string
  readonly about: string
  readonly error: string
  readonly isUserStarted: boolean
}
