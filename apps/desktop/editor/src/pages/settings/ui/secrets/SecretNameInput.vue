<script setup lang="ts">
/**
 * The field a secret's name is typed into. It emits `commit` on Enter, on a change and on leaving
 * the field, and `cancel` on Escape.
 */
import { useTemplateRef } from 'vue'
import { WORDS as words } from '../../words'
import SecretInput from './SecretInput.vue'

const model = defineModel<string>({ required: true })

const emit = defineEmits<{
  (e: 'commit'): void
  (e: 'cancel'): void
}>()

defineExpose({ focus: () => input.value?.focus() })

const input = useTemplateRef<InstanceType<typeof SecretInput>>('input')
</script>

<template>
  <SecretInput
    ref="input"
    v-model="model"
    type="text"
    :aria-label="words.secretRenameName"
    @keydown.enter="emit('commit')"
    @keydown.esc="emit('cancel')"
    @change="emit('commit')"
    @blur="emit('commit')"
  />
</template>
