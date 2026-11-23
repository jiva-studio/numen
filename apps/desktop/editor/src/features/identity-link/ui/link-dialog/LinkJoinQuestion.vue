<script setup lang="ts">
/** The account the new device is offered: join it or not. */
import { Button } from '@numen/ui'
import { WORDS as words } from '../../words'

/* ----------------------------- Props & Emits ------------------------------ */
defineProps<{ account: string; isBusy: boolean }>()

const emit = defineEmits<{ (e: 'answer', isJoining: boolean): void }>()
</script>

<template>
  <div class="link-join">
    <p>{{ words.linkJoinQuestion(account) }}</p>
    <div class="link-join__actions">
      <Button type="button" variant="solid" :disabled="isBusy" @click="emit('answer', true)">
        {{ words.linkJoin }}
      </Button>
      <Button type="button" variant="outline" :disabled="isBusy" @click="emit('answer', false)">
        {{ words.linkCancel }}
      </Button>
    </div>
  </div>
</template>

<style scoped>
.link-join {
  display: flex;
  flex-direction: column;
  gap: 0.75rem;
}

.link-join__actions {
  display: flex;
  flex-wrap: wrap;
  gap: 0.5rem;
}
</style>
