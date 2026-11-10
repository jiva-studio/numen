<script setup lang="ts">
/**
 * The controls over the text, standing together at the end of the media strip.
 */
import { LocateFixed } from '@lucide/vue'
import { WORDS as words } from '../words'

/* ----------------------------- Props & Emits ------------------------------ */
const props = defineProps<{
  isTimed?: boolean
  isFollowing?: boolean
}>()

const emit = defineEmits<{
  'toggle-follow': []
}>()

/* -------------------------------- Handlers -------------------------------- */
function onToggleFollow() {
  emit('toggle-follow')
}
</script>

<template>
  <div class="media__actions">
    <button
      v-if="props.isTimed"
      type="button"
      class="media__follow"
      :aria-label="words.follow"
      :aria-pressed="props.isFollowing ? 'true' : 'false'"
      @click="onToggleFollow"
    >
      <LocateFixed class="media__icon" />
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

.media__follow {
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

.media__follow:hover {
  background: var(--numen-field-bg);
}

.media__follow[aria-pressed='true'] {
  color: var(--numen-accent);
}

.media__icon {
  --icon-inline-size: 1rem;
  --icon-block-size: 1rem;
  inline-size: var(--icon-inline-size);
  block-size: var(--icon-block-size);
}
</style>
