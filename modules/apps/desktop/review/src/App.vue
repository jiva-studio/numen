<script setup lang="ts">
/**
 * The review window: the vaults and what each owes, and the cards themselves.
 *
 * It opens on what a person owes today and nothing else. They need not know an
 * editor exists.
 */
import { computed, onMounted, onUnmounted, ref } from 'vue'
import { SquarePen, Undo2, X } from '@lucide/vue'
import { Button, KeyCap, Notices } from '@numen/ui'
import type { Notice } from '@numen/ui'
import '@numen/ui/styles.css'
import type { Fingerprint } from '@numen/protocol'
import Vaults from './Vaults.vue'
import Decks from './Decks.vue'
import Card from './Card.vue'
import Editing from './Editing.vue'
import { VERSION } from './version'
import { called, deckName, rated, review, said } from './core'
import type { Asked, Held, Owing as OwedVault, Said } from './core'

/** Which of the three the window is on. */
const on = ref<'vaults' | 'decks' | 'session'>('vaults')

const vaults = ref<readonly OwedVault[]>([])
const counting = ref(true)

/**
 * What the window has to say, in the corner it says it in. Trouble stands there
 * until a person puts it away.
 */
const notices = ref<readonly Notice[]>([])

const failed = (why: unknown) => {
  notices.value = [
    ...notices.value,
    {
      id: String(notices.value.length),
      says: String(why),
      tone: 'alarm',
      stay: 'kept',
      // A person pressed something and is waiting to hear. A card that waits
      // for the work to be worth drawing is a card they read ten seconds late.
      asked: true,
    },
  ]
}

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
const done = ref(0)

/** When the card now in front of the person was put there. */
let put = Date.now()

const card = computed<Asked | null>(() => asked.value[at.value] ?? null)
const over = computed(() => on.value === 'session' && card.value === null)

/** The vault whose decks are open, and whose cards are being asked. */
const chosen = computed<OwedVault | null>(
  () => vaults.value.find((one) => one.vaultId === vault.value) ?? null,
)

const count = async () => {
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

const start = async (deck: string) => {
  try {
    const answer = await review.start({ vaultId: vault.value, deck })
    run.value = answer.run
    asked.value = answer.asked.map((one) => ({ ...one }) as Asked)
    at.value = 0
    shown.value = false
    answers.value = []
    done.value = 0
    put = Date.now()
    on.value = 'session'
  } catch (why) {
    failed(why)
  }
}

const show = () => {
  shown.value = true
}

const answer = async (how: Said) => {
  const one = card.value
  if (!one || !shown.value) return
  const took = Date.now() - put
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
  }
  done.value += 1
  at.value += 1
  shown.value = false
  put = Date.now()
}

const takeBack = async () => {
  const last = answers.value[answers.value.length - 1]
  if (last === undefined) return
  try {
    await review.takeBack({ vaultId: vault.value, run: run.value, answer: last })
  } catch (why) {
    failed(why)
    return
  }
  answers.value.pop()
  done.value = Math.max(0, done.value - 1)
  at.value = Math.max(0, at.value - 1)
  shown.value = true
  put = Date.now()
}

/** The card open to be put right, and what a read of its deck gave us. */
const editing = ref(false)
const values = ref<readonly Held[]>([])
const writing = ref(false)
let stood: Fingerprint | undefined

const edit = async () => {
  const one = card.value
  if (!one) return
  try {
    const answer = await review.readCard({
      vaultId: vault.value,
      deck: one.deck,
      card: one.card,
    })
    if (answer.refused) {
      failed(answer.refused)
      return
    }
    values.value = answer.values.map((held) => ({ field: held.field, text: held.text }))
    stood = answer.at
    editing.value = true
  } catch (why) {
    failed(why)
  }
}

const wrote = (field: string, text: string) => {
  values.value = values.value.map((one) => (one.field === field ? { field, text } : one))
}

const save = async () => {
  const one = card.value
  if (!one) return
  writing.value = true
  try {
    const answer = await review.writeCard({
      vaultId: vault.value,
      deck: one.deck,
      card: one.card,
      face: one.face,
      values: values.value.map((held) => ({ field: held.field, text: held.text })),
      ...(stood ? { at: stood } : {}),
    })
    if (answer.changed) {
      failed('the deck moved since it was read, and nothing was written')
      return
    }
    if (answer.refused) {
      failed(answer.refused)
      return
    }
    // What the card now lays out comes from the file, so the person is shown
    // what stands there.
    asked.value = asked.value.map((was, i) =>
      i === at.value
        ? { ...was, front: answer.front, back: answer.back, heading: answer.heading }
        : was,
    )
    stood = answer.at
    editing.value = false
  } catch (why) {
    failed(why)
  } finally {
    writing.value = false
  }
}

/** Out of a sitting and back to the decks, with the counts as they now stand. */
const leave = async () => {
  on.value = 'decks'
  await count()
}

/** Back to the vaults, which is where a person picks another collection. */
const vaultsAgain = async () => {
  on.value = 'vaults'
  vault.value = ''
  await count()
}

/**
 * The keys the whole of a sitting is done with: the space bar turns a card
 * over, the four numbers say how it went, `u` takes the last answer back and
 * escape goes back to the vaults.
 */
const keyed = (press: KeyboardEvent) => {
  if (on.value !== 'session') return

  // A card open to be put right is a card being typed into, and a number typed
  // into a box is not an answer. Escape closes it.
  if (editing.value) {
    if (press.key === 'Escape') editing.value = false
    return
  }

  if (press.key === 'Escape') {
    void leave()
    return
  }
  if (press.key === 'u') {
    void takeBack()
    return
  }
  if (press.key === ' ' || press.key === 'Enter') {
    press.preventDefault()
    if (!shown.value) show()
    return
  }
  const which = Number(press.key)
  if (Number.isInteger(which) && which >= 1 && which <= 4) {
    const how = said[which - 1]
    if (how) void answer(how)
  }
}

onMounted(() => {
  window.addEventListener('keydown', keyed)
  void count()
})
onUnmounted(() => window.removeEventListener('keydown', keyed))
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
      <button class="review__back" type="button" @click="leave">Back to the decks</button>
    </section>

    <section v-else-if="card" class="review__session">
      <header class="review__where">
        <span class="review__deck">{{ deckName(card.deck) }}</span>
        <span v-if="card.section" class="review__section">{{ card.section }}</span>
        <span class="review__face">{{ card.face }}</span>
        <span class="review__left">{{ asked.length - at }} left</span>

        <!-- What a person does beside answering, each one mark. They stand at
             the end of the line that says where the card is from. -->
        <Button variant="ghost" size="icon-small" title="Edit this card" @click="edit">
          <SquarePen />
        </Button>
        <Button
          variant="ghost"
          size="icon-small"
          title="Take the last answer back"
          :disabled="!answers.length"
          @click="takeBack"
        >
          <Undo2 />
        </Button>
        <Button variant="ghost" size="icon-small" title="Leave" @click="leave">
          <X />
        </Button>
      </header>

      <Editing
        v-if="editing"
        :card="card.card"
        :section="card.section"
        :values="values"
        :writing="writing"
        @write="wrote"
        @save="save"
        @close="editing = false"
      />
      <Card
        v-else
        :front="card.front"
        :back="card.back"
        :shown="shown"
        :fresh="!card.seen"
        @show="show"
      />

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
            :class="`review__answer--${how}`"
            @click="answer(how)"
          >
            <KeyCap :keys="{ marks: [], letter: String(i + 1) }" />
            {{ called[how] }}
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

/* Where the card stands, said once and quietly: a person answering is reading
   the card, not the line above it. */
.review__where {
  display: flex;
  align-items: baseline;
  gap: var(--numen-inset);
  color: var(--numen-edge-label);
  font-size: var(--numen-font-size);
}

.review__deck {
  color: var(--numen-node-fg);
  font-weight: 600;
}

.review__left {
  margin-inline-start: auto;
}

.review__answers {
  display: flex;
  flex: none;
  gap: var(--numen-inset);
}

/* An answer is a target a person hits without looking, so it takes the whole
   width it can and stands taller than a button in a row of controls. */
.review__answer {
  flex: 1;
  block-size: auto;
  padding-block: var(--numen-inset-wide);
}

/* The answer that says a card was lost is the one worth telling apart at a
   glance, because it is the one a person reaches for without reading. */
.review__answer--again {
  border-color: var(--numen-alarm);
  color: var(--numen-alarm);
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

.review__back {
  padding: var(--numen-inset) var(--numen-inset-wide);
  border: 1px solid var(--numen-node-border);
  border-radius: var(--numen-radius);
  background: var(--numen-node-bg);
  color: var(--numen-node-fg);
  font: inherit;
  cursor: pointer;
}
</style>
