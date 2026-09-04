/**
 * What is read out of the corner, and what is read out over whatever else is
 * being read.
 *
 * Both regions are emptied and filled again a tick later. Two changes can land
 * inside one tick, and the second reads out everything neither has read yet.
 */
import { nextTick, onMounted, ref, watch, type Ref } from 'vue'
import type { Notice } from './notice'

/** What each card reads out as. */
const wordsOf = (one: Notice): string => (one.about ? `${one.says} — ${one.about}` : one.says)

/** The two regions the corner is read out through. */
export interface Announcer {
  /** What is read out politely, and what is read out over everything. */
  readonly told: Ref<string>
  readonly cried: Ref<string>
}

/**
 * The cards standing now are read out as they arrive and as their words change.
 * Nothing is read out until the regions themselves have been drawn.
 */
export function useAnnouncer(standing: () => readonly Notice[]): Announcer {
  const told = ref('')
  const cried = ref('')

  /** The words each card was last read out by. */
  const announced = ref<ReadonlyMap<string, string>>(new Map())
  /** Whether both regions have stood empty, which is what makes them read. */
  let listening = false
  /** Which reading is the one in hand, and what is waiting to be read out. */
  let reading = 0
  let waiting: readonly Notice[] = []

  const reads = async (all: readonly Notice[]): Promise<void> => {
    if (!listening) return
    const fresh = all.filter((one) => announced.value.get(one.id) !== wordsOf(one))
    announced.value = new Map(all.map((one) => [one.id, wordsOf(one)]))
    if (fresh.length === 0) return
    waiting = [...waiting, ...fresh]

    const mine = ++reading
    told.value = ''
    cried.value = ''
    await nextTick()
    if (mine !== reading) return

    const said = waiting
    waiting = []
    const loud = said.filter((one) => one.tone === 'alarm')
    const quiet = said.filter((one) => one.tone !== 'alarm')
    if (loud.length) cried.value = loud.map(wordsOf).join('. ')
    if (quiet.length) told.value = quiet.map(wordsOf).join('. ')
  }

  watch(standing, (all) => void reads(all))

  onMounted(async () => {
    await nextTick()
    listening = true
    void reads(standing())
  })

  return { told, cried }
}
