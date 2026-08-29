<script setup lang="ts">
/**
 * The review window: the vaults and what each owes, and the cards themselves.
 *
 * It opens on what a person owes today and nothing else. They need not know an
 * editor exists.
 */
import { computed, onMounted, onUnmounted, ref } from 'vue'
import { Undo2, X } from '@lucide/vue'
import { Button, KeyCap, Notices, following } from '@numen/ui'
import type { Notice, Tone } from '@numen/ui'
import '@numen/ui/styles.css'
import Vaults from './Vaults.vue'
import Decks from './Decks.vue'
import Card from './Card.vue'
import { VERSION } from './version'
import { ahead, called, deckName, rated, review, said } from './core'
import type { Asked, Owing as OwedVault, Said } from './core'

/** Which of the three the window is on. */
const on = ref<'vaults' | 'decks' | 'session'>('vaults')

const vaults = ref<readonly OwedVault[]>([])
const counting = ref(true)

/**
 * What the window has to say, in the corner it says it in. Trouble stands there
 * until a person puts it away.
 */
const notices = ref<readonly Notice[]>([])

/** How many notices the window has raised, which is what names the next one. */
let raised = 0

/** Something the window has to say, in the tone it says it in. */
const says = (said: string, tone: Tone) => {
  raised += 1
  notices.value = [
    ...notices.value,
    {
      id: String(raised),
      says: said,
      tone,
      stay: 'kept',
      // A person pressed something and is waiting to hear. A card that waits
      // for the work to be worth drawing is a card they read ten seconds late.
      asked: true,
    },
  ]
}

const failed = (why: unknown) => says(String(why), 'alarm')

const putAway = (id: string) => {
  notices.value = notices.value.filter((one) => one.id !== id)
}

/** The sitting: which vault, which run, what is left to ask, and where in it. */
const vault = ref('')
const run = ref('')
const asked = ref<readonly Asked[]>([])
const at = ref(0)
const shown = ref(false)
const answers = ref<string[]>([])

/**
 * Whether an answer is on its way to the vault. A card is answered once: the
 * line is written before the next card is put up, and a second press while that
 * is happening would write the same card twice and skip the one after it.
 */
const writing = ref(false)

/** How many answers this sitting has written, which is what stands at the end. */
const done = computed(() => answers.value.length)

/** When the card now in front of the person was put there. */
let put = Date.now()

const card = computed<Asked | null>(() => asked.value[at.value] ?? null)
const over = computed(() => on.value === 'session' && card.value === null)

/** The vault whose decks are open, and whose cards are being asked. */
const chosen = computed<OwedVault | null>(
  () => vaults.value.find((one) => one.vaultId === vault.value) ?? null,
)

/** Whether a count is on its way, so two never run at once. */
let counted: Promise<void> | null = null

const count = async () => {
  if (counted) return counted
  counted = counting_()
  try {
    await counted
  } finally {
    counted = null
  }
}

const counting_ = async () => {
  counting.value = true
  try {
    const answer = await review.owing({})
    vaults.value = answer.vaults.map((one) => ({
      vaultId: one.vaultId,
      name: one.name,
      path: one.path,
      faces: one.faces,
      due: one.due,
      new: one.new,
      decks: one.decks.map((deck) => ({
        deck: deck.deck,
        faces: deck.faces,
        due: deck.due,
        new: deck.new,
      })),
      unread: one.unread,
    }))
  } catch (why) {
    failed(why)
  } finally {
    counting.value = false
  }
}

const choose = (id: string) => {
  vault.value = id
  on.value = 'decks'
}

/**
 * What the sitting could not act on, said once as it opens: a deck whose cards
 * could not be given marks holds cards this sitting does not ask, and a line of
 * the vault's answers that could not be read is a card standing where the rest
 * of its history left it.
 */
const reported = (unwritten: readonly string[], skipped: number) => {
  if (unwritten.length) {
    says(
      `Not asked from ${unwritten.map(deckName).join(', ')}: the deck could not be written.`,
      'caution',
    )
  }
  if (skipped > 0) {
    says(`${skipped} answers in this vault could not be read.`, 'caution')
  }
}

const start = async (deck: string) => {
  try {
    const answer = await review.start({ vaultId: vault.value, deck })
    run.value = answer.run
    asked.value = answer.asked.map((one) => ({
      ...one,
      ahead: one.ahead
        ? {
            again: Number(one.ahead.again),
            hard: Number(one.ahead.hard),
            good: Number(one.ahead.good),
            easy: Number(one.ahead.easy),
          }
        : null,
    }))
    at.value = 0
    shown.value = false
    answers.value = []
    put = Date.now()
    on.value = 'session'
    reported(answer.unwritten, answer.skipped)
  } catch (why) {
    failed(why)
  }
}

const show = () => {
  shown.value = true
}

const answer = async (how: Said) => {
  const one = card.value
  if (!one || !shown.value || writing.value) return
  const took = Date.now() - put
  writing.value = true
  try {
    const given = await review.answer({
      vaultId: vault.value,
      run: run.value,
      card: one.card,
      face: one.face,
      rating: rated[how],
      tookMs: BigInt(took),
    })
    answers.value.push(given.answer)
  } catch (why) {
    failed(why)
    return
  } finally {
    writing.value = false
  }
  at.value += 1
  shown.value = false
  put = Date.now()
}

const takeBack = async () => {
  const last = answers.value[answers.value.length - 1]
  if (last === undefined || writing.value) return
  writing.value = true
  try {
    await review.takeBack({ vaultId: vault.value, run: run.value, answer: last })
  } catch (why) {
    failed(why)
    return
  } finally {
    writing.value = false
  }
  answers.value.pop()
  at.value = Math.max(0, at.value - 1)
  shown.value = true
  put = Date.now()
}

/**
 * The sitting let go of. What was answered is in the vault, and the next
 * sitting is opened whole: a card kept here is one a screen could fall back to
 * showing with nothing able to answer it.
 */
const forget = () => {
  asked.value = []
  at.value = 0
  shown.value = false
  answers.value = []
  run.value = ''
}

/** Out of a sitting and back to the decks, with the counts as they now stand. */
const leave = async () => {
  forget()
  on.value = 'decks'
  await count()
}

/** Back to the vaults, which is where a person picks another collection. */
const vaultsAgain = async () => {
  forget()
  on.value = 'vaults'
  vault.value = ''
  await count()
}

/**
 * The keys the whole of a sitting is done with: the space bar turns a card
 * over, the four numbers say how it went, `u` takes the last answer back and
 * escape goes back to the decks.
 */
const keyed = (press: KeyboardEvent) => {
  if (on.value !== 'session') return
  // A key held down repeats, and a card is answered once. A key pressed with a
  // modifier is the machine's own shortcut and is not an answer.
  if (press.repeat || press.altKey || press.ctrlKey || press.metaKey) return
  if (press.key === 'Escape') {
    void leave()
    return
  }
  if (press.key === 'u' || press.key === 'U') {
    void takeBack()
    return
  }
  // The space bar turns the card over. Once it is over, the keys that mean
  // something are the four, and space and enter are left to whatever the person
  // has moved focus to.
  if (press.key === ' ' && !shown.value) {
    press.preventDefault()
    show()
    return
  }
  const which = Number(press.key)
  if (Number.isInteger(which) && which >= 1 && which <= 4) {
    const how = said[which - 1]
    if (how) void answer(how)
  }
}

/** Whether the window is still open, which is how long anything is followed. */
let open = true

const follows = following({
  open: () => open,
  lost: failed,
  wait: (ms) => new Promise((then) => setTimeout(then, ms)),
})

onMounted(() => {
  window.addEventListener('keydown', keyed)
  void count()
  // A deck written or a card changed underneath the window is counted again
  // without a person asking. A sitting is left alone: its cards were laid out
  // when it opened, and what a deck says now is read at the next one.
  void follows(
    () => review.changes({}),
    () => (on.value === 'session' ? undefined : count()),
  )
})
onUnmounted(() => {
  open = false
  window.removeEventListener('keydown', keyed)
})
</script>

<template>
  <main class="review">

    <Vaults
      v-if="on === 'vaults'"
      :vaults="vaults"
      :counting="counting"
      :version="VERSION"
      @choose="choose"
    />

    <Decks
      v-else-if="on === 'decks' && chosen"
      :vault="chosen"
      @start="start"
      @back="vaultsAgain"
    />

    <section v-else-if="over" class="review__over">
      <h1 class="review__over-said">Nothing left today.</h1>
      <p class="review__over-count">{{ done }} answered.</p>
      <Button variant="outline" @click="leave">Back to the decks</Button>
    </section>

    <section v-else-if="card" class="review__session">
      <header class="review__where">
        <span class="review__deck">{{ deckName(card.deck) }}</span>
        <span v-if="card.section">{{ card.section }}</span>
        <span>{{ card.face }}</span>
        <span v-if="!card.seen" class="review__new">new</span>
        <span class="review__left">{{ asked.length - at }} left</span>

        <!-- What a person does beside answering, each one mark. They stand at
             the end of the line that says where the card is from. -->
        <Button
          variant="ghost"
          size="icon-small"
          title="Take the last answer back"
          aria-label="Take the last answer back"
          :disabled="!answers.length"
          @click="takeBack"
        >
          <Undo2 />
        </Button>
        <Button variant="ghost" size="icon-small" title="Leave" aria-label="Leave" @click="leave">
          <X />
        </Button>
      </header>

      <!-- The card is what changes under a person as they work, so a reader
           that is not looking at the screen is told when the answer appears. -->
      <div class="review__card" aria-live="polite">
        <Card :front="card.front" :back="card.back" :shown="shown" @show="show" />
      </div>

      <footer class="review__answers">
        <!-- The key first and the word after it: a person answering with the
             keyboard reads down the row of keys, and one answering with the
             mouse reads the words either way. -->
        <template v-if="shown">
          <Button
            v-for="(how, i) in said"
            :key="how"
            variant="outline"
            class="review__answer"
            @click="answer(how)"
          >
            <KeyCap :keys="{ marks: [], letter: String(i + 1) }" />
            {{ called[how] }}
            <!-- What the answer does to the card, said where the answer is
                 chosen: a person picking between the four is picking between
                 these. It is read off the screen and not out of the button's
                 own name, which is the word a person means to press. -->
            <span v-if="card.ahead" class="review__ahead" aria-hidden="true">{{
              ahead(card.ahead[how])
            }}</span>
          </Button>
        </template>
        <Button v-else variant="outline" class="review__answer" @click="show">
          <KeyCap :keys="{ marks: [], letter: 'space' }" />
          Show the answer
        </Button>
      </footer>

    </section>
  </main>

  <!-- What the window has to say, in the corner every window says it in. -->
  <Notices :notices="notices" @gone="putAway" />
</template>

<style scoped>
.review {
  display: flex;
  flex-direction: column;
  block-size: 100%;
  padding: var(--numen-inset-wide);
  gap: var(--numen-inset);
}

.review__session {
  display: flex;
  flex: 1;
  flex-direction: column;
  min-block-size: 0;
  gap: var(--numen-inset);
}

/* The card takes what the row of answers leaves, and it is the card itself that
   scrolls. */
.review__card {
  display: flex;
  flex: 1;
  min-block-size: 0;
}

/* Where the card stands, said once and quietly: a person answering is reading
   the card, not the line above it. Text and marks are centred against each
   other, because a button has no baseline to put a word on. */
.review__where {
  display: flex;
  align-items: center;
  gap: var(--numen-inset);
  color: var(--numen-edge-label);
  font-size: var(--numen-font-size);
}

.review__deck {
  color: var(--numen-node-fg);
  font-weight: 600;
}

/* A pill in the same row as the one that counts what is waiting, so it is the
   same shape as that one. */
.review__new {
  padding: 0.0625rem 0.4rem;
  border-radius: var(--numen-radius-pill);
  background: var(--numen-caution-bg);
  color: var(--numen-caution-fg);
}

.review__left {
  margin-inline-start: auto;
}

.review__answers {
  display: flex;
  flex: none;
  gap: var(--numen-inset);
}

/* What the answer does, said quietly beside it: it is read once, when a person
   is learning what the four mean, and glanced at after that. */
.review__ahead {
  color: var(--numen-edge-label);
  font-size: var(--numen-edge-label-size);
  font-variant-numeric: tabular-nums;
}

/* An answer is a target a person hits without looking, so it takes the whole
   width it can and stands taller than a button in a row of controls. */
.review__answer {
  flex: 1;
  block-size: auto;
  padding-block: var(--numen-inset-wide);
}

.review__over {
  display: flex;
  flex: 1;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: var(--numen-inset);
}

.review__over-said {
  margin: 0;
  font-size: var(--numen-display-size);
  font-weight: 600;
}

.review__over-count {
  margin: 0;
  color: var(--numen-edge-label);
}

</style>
