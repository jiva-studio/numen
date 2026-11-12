/** A signal: a producer emits a value and every listener subscribed at that moment receives it. */
export interface Emitter<T> {
  readonly emit: (value: T) => void
  /** Answers the call that stops the listener. */
  subscribe(listener: (value: T) => void): () => void
}

export function createEmitter<T>(): Emitter<T> {
  const listeners = new Set<(value: T) => void>()
  return {
    emit: (value) => {
      for (const listener of [...listeners]) listener(value)
    },
    subscribe: (listener) => {
      listeners.add(listener)
      return () => {
        listeners.delete(listener)
      }
    },
  }
}
