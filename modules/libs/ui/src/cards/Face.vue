<script setup lang="ts">
/**
 * One card, drawn.
 *
 * What it draws is assembled already: what stood in the braces, and which face
 * was chosen, are settled before this is handed anything. It shows the front,
 * and the back once it is turned.
 */
import { computed } from 'vue'
import Marks from './Marks.vue'
import { Button } from '../components/ui/button'
import { parts } from './model'

const props = withDefaults(
  defineProps<{
    /** The front, as markdown with tags among the marks. */
    front: string
    /** The back, written the same way. */
    back: string
    /** The back is drawn. */
    turned?: boolean
    /** What the card is announced as. */
    name?: string
    /** What is said in place of a half with nothing in it. */
    silence?: string
    /** What the button turning the card says. */
    turning?: string
  }>(),
  { turned: false, name: 'Card', silence: 'Nothing here', turning: 'Turn' },
)

const emit = defineEmits<{
  /** The card asked to turn, with the side it would come to. */
  (event: 'turn', turned: boolean): void
  /** A link in the card was pressed. Where it goes is the caller's. */
  (event: 'follow', href: string, press: MouseEvent): void
}>()

const shown = computed(() => parts(props.front, props.back, props.turned))
</script>

<template>
  <article
    class="face numen flex flex-col bg-raised font-sans text-base text-ink"
    :aria-label="name"
    :data-turned="turned || undefined"
  >
    <section
      v-for="part in shown"
      :key="part.half"
      class="face__half"
      :data-half="part.half"
    >
      <p v-if="part.blank" class="face__silence text-small text-hushed">{{ silence }}</p>
      <Marks v-else :text="part.text" @follow="(href, press) => emit('follow', href, press)" />
    </section>

    <footer class="face__foot flex justify-end">
      <Button
        class="face__turn"
        type="button"
        size="small"
        :aria-pressed="turned"
        @click="emit('turn', !turned)"
        >{{ turning }}</Button
      >
    </footer>
  </article>
</template>

<style scoped>
/* A card is a box lifted off the window, with room around what it holds and a
   rule between its two halves. */
.face {
  --pad: 1rem;
  --half-gap: 0.875rem;

  gap: var(--half-gap);
  padding: var(--pad);
  border: var(--numen-stroke) solid var(--numen-node-border);
  border-radius: var(--numen-radius-panel);
  box-shadow: var(--numen-shadow-card);
  overflow-wrap: anywhere;
}

/* The back is told from the front by the rule above it. */
.face__half[data-half='back'] {
  padding-block-start: var(--half-gap);
  border-block-start: var(--numen-stroke) solid var(--numen-node-border);
}

.face__silence {
  margin: 0;
  letter-spacing: var(--numen-caps-tracking);
  text-transform: uppercase;
}

/* The one thing a card is pressed for stands under both halves, clear of them. */
.face__foot {
  padding-block-start: var(--numen-inset);
}
</style>
