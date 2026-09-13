<script setup lang="ts">
/**
 * The controls over the text, standing together at the end of the media strip.
 */
import { Ellipsis, LocateFixed } from '@lucide/vue'
import { WORDS as words } from '../../words'

// --- Props & Emits ---
const props = defineProps<{
  /** Whether the text carries times, which is what following goes by. */
  timed?: boolean
  /** Whether the view keeps the line being said in sight. */
  follows?: boolean
  /** Whether the menu at the end of the strip offers anything. */
  hasMenu?: boolean
}>()

const emit = defineEmits<{ toggleFollow: []; openMenu: [event: Event] }>()

// --- Handlers ---
function onToggleFollow() {
  emit('toggleFollow')
}

function onOpenMenu(event: Event) {
  emit('openMenu', event)
}
</script>

<template>
  <div class="media__actions">
    <button
      v-if="props.timed"
      type="button"
      class="media__follow"
      :aria-label="words.follow"
      :title="words.follow"
      :aria-pressed="props.follows ? 'true' : 'false'"
      @click="onToggleFollow"
    >
      <LocateFixed class="media__icon" />
    </button>
    <button
      v-if="props.hasMenu"
      type="button"
      class="media__more"
      :aria-label="words.more"
      :title="words.more"
      aria-haspopup="menu"
      @click="onOpenMenu"
    >
      <Ellipsis class="media__icon" />
    </button>
  </div>
</template>

<style scoped>
.media__actions {
  display: flex;
  align-items: center;
  flex: none;
  gap: var(--media-close);
}

.media__follow,
.media__more {
  display: grid;
  place-items: center;
  flex: none;
  inline-size: var(--numen-action-size);
  block-size: var(--numen-action-size);
  padding: 0;
  border: 0;
  border-radius: var(--numen-radius-pill);
  background: none;
  color: var(--numen-hushed);
  cursor: pointer;
}

.media__follow:hover,
.media__more:hover {
  background: var(--numen-field-bg);
}

.media__follow[aria-pressed='true'] {
  color: var(--numen-accent);
}

.media__icon {
  inline-size: 1rem;
  block-size: 1rem;
}
</style>
