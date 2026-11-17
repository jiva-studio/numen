export interface DropOptions {
  readonly share: number
  readonly limit: number
}

export const DEFAULT_DROP: DropOptions = { share: 0.2, limit: 96 }
