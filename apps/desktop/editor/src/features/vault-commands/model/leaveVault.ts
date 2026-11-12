/**
 * Switching the vault of the window without its open tabs reaching the vault it switches to.
 */
import type { Vaults } from '@/entities/vault'

/** What holds back the writes of the open tabs while the window changes vault. */
export interface TabsHold {
  /** Resolves once the tabs as they are now have reached the core, and later writes are held. */
  pause(): Promise<void>
  /** Lets the tabs be written again. */
  resume(): void
}

/**
 * `open` with the tabs written to the vault being left first. The writes name no vault, so one
 * made after the core shows the other vault would replace that vault's tabs. A refused switch
 * leaves the vault where it was, and its tabs are written again.
 */
export const openAfterSavingTabs = (open: Vaults['open'], hold: TabsHold): Vaults['open'] => {
  return async (id) => {
    await hold.pause()
    let error: Awaited<ReturnType<Vaults['open']>>
    try {
      error = await open(id)
    } catch (thrown) {
      hold.resume()
      throw thrown
    }
    if (error) hold.resume()
    return error
  }
}
