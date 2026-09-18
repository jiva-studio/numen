/**
 * Which cards stand in the corner, what each of them has left to say, and when
 * each of them goes.
 *
 * Work appears once it has lasted and goes when the work does; something that
 * is so stands while it is so; something that happened stands to be read and
 * then goes, unless it is trouble, which stands until it is put away. Nothing
 * here draws anything.
 */
import {
  computed,
  nextTick,
  ref,
  shallowRef,
  watch,
  watchEffect,
  type ComputedRef,
  type Ref,
  type ShallowRef,
} from 'vue'
import { arrivals, dwellOf, getFinishedNotices, getShownNotices } from '../lib/dwell'
import { foldNotices } from '../lib/fold'
import { measureMovement, type Movement } from '../lib/movement'
import { getStillAway, getReadable, tallyOf, type Notice } from '../lib/notice'
import { getRemainingWord } from '../lib/tally'
import { useNoticeStack } from './stack'

/** A card standing now, which carries the way away under it. */
interface CardHandle {
  readonly way: HTMLElement | null
}

export interface NoticeCardsOptions {
  /** What the window has to say, in the order it is drawn. */
  readonly getNotices: () => readonly Notice[]
  /** The corner itself, which a pointer and the keyboard are followed over. */
  readonly stack: Readonly<ShallowRef<HTMLElement | null>>
  /** How long work runs before it is worth a card. */
  readonly getWait: () => number
  /** How many cards stand at once. */
  readonly getRoom: () => number
  /** What the moment is. */
  readonly getNow: () => number
  /** Whether nobody is looking. */
  readonly isHidden: () => boolean
  /** A card is finished with: read long enough, or dismissed. */
  readonly dismiss: (id: string) => void
}

export interface NoticeCardsState {
  /** Everything standing, whether or not it is folded away behind the rest. */
  readonly drawn: ComputedRef<readonly Notice[]>
  /** What is drawn, and how many stand behind it. */
  readonly folds: ComputedRef<{ shown: readonly Notice[]; over: number }>
  /** Whether a person has asked to see what is folded away behind the rest. */
  readonly isOpened: Ref<boolean>
  /** What a counting notice has left to run, in words. */
  readonly leftOn: (one: Notice) => string
  /** A card as it is drawn, held under the notice it stands for. */
  readonly holdCard: (id: string, card: unknown) => void
  /** A card put away by hand. */
  readonly dismissByHand: (id: string) => Promise<void>
  readonly onPointerOver: (event: PointerEvent) => void
  readonly onPointerOut: (event: PointerEvent) => void
  readonly onFocusIn: () => void
  readonly onFocusOut: (event: FocusEvent) => void
}

export function useNoticeCards(options: NoticeCardsOptions): NoticeCardsState {
  /** How fast each count is moving. This is the clock the rate is read against. */
  const moving = shallowRef<ReadonlyMap<string, Movement>>(new Map())

  /** The ones a person has put away, and when each of the rest arrived. */
  const away = shallowRef<ReadonlySet<string>>(new Set())
  const arrived = shallowRef<ReadonlyMap<string, number>>(new Map())

  /** How long the corner has been held for, and the moment a card is read against. */
  const { now, read, beat, onPointerOver, onPointerOut, onFocusIn, onFocusOut } = useNoticeStack(
    options.stack,
    options.getNow,
    options.isHidden,
  )

  /** The ones whose caller has already been told they are finished with. */
  const forgotten = new Set<string>()

  /**
   * The clock forward, and every count read against it.
   *
   * A count is read on the clock rather than as it arrives, so how fast it is
   * moving is measured over stretches of time and not over however often the
   * work behind it happens to speak.
   */
  const sample = (): void => {
    beat()
    moving.value = measureMovement(moving.value, options.getNotices(), now.value)
  }

  watch(
    options.getNotices,
    (all) => {
      sample()
      arrived.value = arrivals(arrived.value, all, read.value)
      away.value = getStillAway(away.value, all)
      const here = new Set(getReadable(all).map((one) => one.id))
      for (const id of [...forgotten]) if (!here.has(id)) forgotten.delete(id)
    },
    { immediate: true },
  )

  const leftOn = (one: Notice): string => {
    const tally = tallyOf(one)
    if (tally === undefined) return ''
    return getRemainingWord(tally.total - tally.done, moving.value.get(one.id)?.rate ?? 0)
  }

  const drawn = computed(() =>
    getShownNotices(options.getNotices(), arrived.value, away.value, read.value, options.getWait()),
  )

  const isOpened = ref(false)
  const folds = computed(() =>
    foldNotices(drawn.value, isOpened.value ? drawn.value.length : options.getRoom()),
  )

  // Asking to see what is behind the rest is asked about what stands then. Once
  // it all fits again, the next stack over the room folds as any other would.
  watch(drawn, (all) => {
    if (all.length <= options.getRoom()) isOpened.value = false
  })

  /** Whether anything readable has not yet lasted long enough to be drawn. */
  const coming = computed(() => {
    const shown = new Set(drawn.value.map((one) => one.id))
    return getReadable(options.getNotices()).some(
      (one) => one.stay !== 'read' && !away.value.has(one.id) && !shown.has(one.id),
    )
  })

  /** Whether anything drawn is going to go by itself. */
  const dwelling = computed(() =>
    drawn.value.some((one) => one.stay === 'read' && dwellOf(one.says, one.about) !== Infinity),
  )

  /** Whether anything drawn is counting, and so has a rate to be read. */
  const counting = computed(() => drawn.value.some((one) => tallyOf(one) !== undefined))

  // The corner changes by itself while nothing else changes, so the moment is
  // watched for as long as something is waiting on it.
  watchEffect((clean) => {
    if (!coming.value && !dwelling.value && !counting.value) return
    const tick = setInterval(sample, 250)
    clean(() => clearInterval(tick))
  })

  watchEffect(() => {
    for (const id of getFinishedNotices(options.getNotices(), arrived.value, read.value)) {
      if (forgotten.has(id)) continue
      forgotten.add(id)
      options.dismiss(id)
    }
  })

  /** The cards drawn, each under the notice it stands for. */
  const cards = new Map<string, CardHandle>()

  const holdCard = (id: string, card: unknown): void => {
    if (card) cards.set(id, card as CardHandle)
    else cards.delete(id)
  }

  /**
   * A card put away, and the keyboard left where it can go on putting them
   * away: on the card that takes the place of the one that went, or on the last.
   */
  const dismissByHand = async (id: string): Promise<void> => {
    const at = folds.value.shown.findIndex((one) => one.id === id)
    const held = cards.get(id)?.way === document.activeElement
    forgotten.add(id)
    away.value = new Set([...away.value, id])
    options.dismiss(id)
    if (!held || at < 0) return
    await nextTick()
    const left = folds.value.shown
    const next = left[Math.min(at, left.length - 1)]
    if (next) cards.get(next.id)?.way?.focus()
  }

  return {
    drawn,
    folds,
    isOpened,
    leftOn,
    holdCard,
    dismissByHand,
    onPointerOver,
    onPointerOut,
    onFocusIn,
    onFocusOut,
  }
}
