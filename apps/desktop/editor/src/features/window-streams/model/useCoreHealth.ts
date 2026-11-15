import { ref } from 'vue'

/** Whether the window can read the vault at all, and what it lost its connection to. */
export function useCoreHealth() {
  return {
    isIndexing: ref(true),
    /** The core could not be reached: nothing else in the window is true. */
    failure: ref(''),
    /** What the window itself lost its connection to, reported until restored. */
    lost: ref(''),
  }
}

export type CoreHealth = ReturnType<typeof useCoreHealth>
