<script setup lang="ts">
/**
 * A key cap: the keys held drawn as icons, and the letter held with them set in
 * type.
 *
 * A cap is one height whatever it holds, and every clearance in it is in `em`
 * against the type it is set in, so it is one shape at every size the interface
 * is drawn at.
 */
import { computed } from 'vue'
import { ICONS } from './icons'
import type { PaletteKeys } from './item'

const props = defineProps<{ keys: PaletteKeys }>()

/**
 * The keystroke in words, in the order it is held. This is the whole of what a
 * reader who is listening is told, and everything drawn is hidden from them.
 */
const spoken = computed(() =>
  [...props.keys.icons.map((icon) => ICONS[icon].said), props.keys.letter]
    .filter((word) => word !== '')
    .join(' '),
)
</script>

<template>
  <kbd class="cap">
    <span class="sr-only">{{ spoken }}</span>
    <component
      :is="ICONS[icon].icon"
      v-for="icon in keys.icons"
      :key="icon"
      class="cap__icon"
      :style="{ '--fills': ICONS[icon].fills }"
      :stroke-width="ICONS[icon].stroke"
      aria-hidden="true"
      focusable="false"
    />
    <span v-if="keys.letter" aria-hidden="true">{{ keys.letter }}</span>
  </kbd>
</template>

<style scoped>
/* Small print on the ground a box is drawn on, inside a line that is a quarter
   of the ink it is set in. The type is a token, so a cap in the foot of the
   palette and a cap on a row are one object. */
.cap {
  /* The corner of a key cap, which is tighter than the corner of a node. */
  --cap-radius: 0.25rem;
  /* How wide an icon filling the cap is drawn: the height of the letter beside
     it. */
  --cap-icon: 0.85em;

  display: inline-flex;
  align-items: center;
  justify-content: center;
  gap: 0.1em;
  block-size: 1.6em;
  min-inline-size: 1.6em;
  padding-inline: 0.45em;
  border: var(--numen-stroke) solid
    color-mix(in oklab, var(--numen-node-bg), var(--numen-node-fg) 25%);
  border-radius: var(--cap-radius);
  background: var(--numen-node-bg);
  color: var(--numen-node-fg);
  font-family: var(--numen-font-sans);
  font-size: var(--numen-text-1);
  line-height: 1;
}

/* Lucide draws every icon centred on its own grid, so one box and one middle
   put every icon and the letter on one line. An icon drawn larger than the box
   is drawn over the air its own grid leaves at the sides, and the clearance
   between icons stays the box's. */
.cap__icon {
  flex: none;
  inline-size: var(--cap-icon);
  block-size: var(--cap-icon);
  transform: scale(var(--fills));
}
</style>
