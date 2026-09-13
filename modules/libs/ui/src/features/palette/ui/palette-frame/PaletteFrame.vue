<script setup lang="ts">
/**
 * The ground the palette stands on, and the panel it is drawn in.
 *
 * The ground catches every press the panel does not, and the window behind is
 * out of reach for as long as the panel stands.
 *
 * `data-palette` names each part: `ground` and `panel`.
 */
defineProps<{
  /** What the panel is announced as. */
  name: string
  /** Where it is drawn. */
  to: string | HTMLElement
}>()

defineEmits<{
  /** A press on the ground itself. */
  (event: 'ground'): void
  /** A press anywhere in the panel. */
  (event: 'press'): void
}>()

defineSlots<{
  /** What the panel holds. */
  default(): unknown
}>()

// What it is drawn into stands between whatever draws it and the ground, so
// what is handed down is put on the ground itself.
defineOptions({ inheritAttrs: false })
</script>

<template>
  <Teleport :to="to">
    <div
      class="palette numen font-sans text-base text-ink"
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
  --widest: 640px;
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
