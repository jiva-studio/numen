/**
 * What a vault was answered on, for the grid of days the vault screen draws.
 *
 * Apart from the template because it is asked for when a vault is opened and
 * again whenever that vault moves, and what is held between the two is a rule.
 */
import { ref } from 'vue'

/** What the application answers about a vault's days. */
export interface Asks {
  reviewed(said: { vaultId: string }): Promise<Said>
}

export interface Said {
  days: readonly { day: string; answered: number }[]
  streak: number
  answered: number
}

export interface Reviewing {
  cards: Asks
  failed(why: unknown): void
}

export function reviewed(deps: Reviewing) {
  /** How much was answered on each day, by the day it was answered on. */
  const days = ref<ReadonlyMap<string, number>>(new Map())
  const streak = ref(0)
  const answered = ref(0)

  /** Which vault the days on hand belong to, so another vault's are not drawn. */
  const of = ref('')

  const forget = () => {
    days.value = new Map()
    streak.value = 0
    answered.value = 0
    of.value = ''
  }

  const read = async (vaultId: string) => {
    if (!vaultId) {
      forget()
      return
    }
    of.value = vaultId
    try {
      const said = await deps.cards.reviewed({ vaultId })
      // A person who moved to another vault while this was on its way is
      // looking at that one, and these days are not its days.
      if (of.value !== vaultId) return
      days.value = new Map(said.days.map((one) => [one.day, one.answered]))
      streak.value = said.streak
      answered.value = said.answered
    } catch (why) {
      if (of.value !== vaultId) return
      deps.failed(why)
      forget()
    }
  }

  return { days, streak, answered, of, read, forget }
}
