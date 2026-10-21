/**
 * What is read out of the corner, and what is read out over whatever else is
 * being read.
 *
 * Both regions are emptied and filled again a tick later. Two changes can land
 * inside one tick, and the second reads out everything neither has read yet.
 */
import { nextTick, onMounted, ref, shallowRef, watch, type Ref } from 'vue'
import type { Notice } from '../lib/notice'

/** What each card reads out as. */
const getWords = (one: Notice): string => (one.about ? `${one.text} — ${one.about}` : one.text)

/** The two regions the corner is read out through. */
export interface Announcer {
  /** What is read out politely, and what is read out over everything. */
  readonly told: Ref<string>
  readonly cried: Ref<string>
}

/**
 * The cards drawn now are read out as they arrive and as their words change.
 * Nothing is read out until the regions themselves have been drawn.
 */
export function useAnnouncer(getNotices: () => readonly Notice[]): Announcer {
  const told = ref('')
  const cried = ref('')

  /** The words each card was last read out by. */
  const announced = shallowRef<ReadonlyMap<string, string>>(new Map())
  /** Whether both regions have stood empty, which is what makes them read. */
  let listening = false
  /** Which reading is the one in hand, and what is waiting to be read out. */
  let reading = 0
  let waiting: readonly Notice[] = []

  const readOut = async (all: readonly Notice[]): Promise<void> => {
    if (!listening) return
    const unannounced = all.filter((one) => announced.value.get(one.id) !== getWords(one))
    announced.value = new Map(all.map((one) => [one.id, getWords(one)]))
    if (unannounced.length === 0) return
    waiting = [...waiting, ...unannounced]

    const mine = ++reading
    told.value = ''
    cried.value = ''
    await nextTick()
    if (mine !== reading) return

    const pending = waiting
    waiting = []
    const loud = pending.filter((one) => one.tone === 'alarm')
    const regular = pending.filter((one) => one.tone !== 'alarm')
    if (loud.length) cried.value = loud.map(getWords).join('. ')
    if (regular.length) told.value = regular.map(getWords).join('. ')
  }

  watch(getNotices, (all) => void readOut(all))

  onMounted(async () => {
    await nextTick()
    listening = true
    void readOut(getNotices())
  })

  return { told, cried }
}
