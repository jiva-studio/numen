<script setup lang="ts">
/** The six digits the host shows, typed on the new device. */
import { ref } from 'vue'
import { Button } from '@numen/ui'
import { WORDS as words } from '../../words'

/* ----------------------------- Props & Emits ------------------------------ */
defineProps<{ triesLeft: number; isBusy: boolean }>()

const emit = defineEmits<{ (e: 'submit', code: string): void }>()

/* --------------------------------- State ---------------------------------- */
const typed = ref('')
</script>

<template>
  <form class="link-code" @submit.prevent="emit('submit', typed)">
    <label class="link-code__field">
      <span>{{ words.linkCodeField }}</span>
      <input v-model="typed" type="text" inputmode="numeric" maxlength="6" autocomplete="off" />
    </label>
    <p v-if="triesLeft > 0 && triesLeft < 3">{{ words.linkTriesLeft(triesLeft) }}</p>
    <Button type="submit" variant="solid" :disabled="isBusy">{{ words.linkCodeConfirm }}</Button>
  </form>
</template>

<style scoped>
.link-code,
.link-code__field {
  display: flex;
  flex-direction: column;
  gap: 0.25rem;
}
</style>
