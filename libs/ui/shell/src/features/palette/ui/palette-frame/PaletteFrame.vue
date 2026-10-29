<script setup lang="ts">
/**
 * The backdrop of the palette, and the panel it is rendered in.
 *
 * The backdrop catches every press the panel does not, and the window behind
 * is not interactive while the panel is open.
 *
 * `data-palette` names each part: `ground` and `panel`.
 */
defineOptions({ inheritAttrs: false })

defineProps<{
  /** What the panel is announced as. */
  name: string
  /** Where it is drawn. */
  to: string | HTMLElement
}>()

defineEmits<{
  /** A press on the backdrop itself. */
  (event: 'ground'): void
  /** A press anywhere in the panel. */
  (event: 'press'): void
}>()

defineSlots<{
  /** What the panel holds. */
  default(): unknown
}>()
</script>

<template>
  <Teleport :to="to">
    <div
      class="palette numen text-ink font-sans text-base"
      data-palette="ground"
      v-bind="$attrs"
      @pointerdown.self="$emit('ground')"
    >
      <div
        class="palette__panel panel-numen relative flex min-h-0 flex-col"
        data-palette="panel"
        role="dialog"
        aria-modal="true"
        :aria-label="name"
        @pointerdown="$emit('press')"
      >
        <slot />
      </div>
    </div>
  </Teleport>
</template>

<style scoped>
/* Over the window, with what it covers showing through the panel and dimmed
   everywhere else. */
.palette {
  --lift: var(--numen-lift-palette);
  /* How far down the window it hangs, how wide it may be, and how much of the
     screen its list takes before it scrolls. */
  --drop: 12vh;
  --widest: 40rem;
  --tallest: 50vh;

  position: fixed;
  inset: 0;
  z-index: var(--lift);
  display: grid;
  justify-items: center;
  align-content: start;
  padding: var(--drop) var(--numen-inset-wide);
  background: var(--numen-scrim);
}

.palette__panel {
  inline-size: 100%;
  max-inline-size: var(--widest);
  max-block-size: calc(var(--tallest) + 2 * var(--numen-action-size));
}
</style>
