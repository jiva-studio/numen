<script setup lang="ts">
/**
 * The questions of a template, asked in a small window over the app before the note is made: a line
 * for each text field and a menu for each select field, with Create and Cancel. Enter creates, and
 * Escape cancels.
 */
import { onMounted, reactive, useTemplateRef } from 'vue'
import { Button, Select } from '@numen/ui'
import type { Field } from '../api/templates'
import { getDefaults } from '../model/getDefaults'
import { WORDS as words } from '../words'

/* --------------------------------- Props ---------------------------------- */
const props = defineProps<{
  title: string
  fields: readonly Field[]
}>()

/* --------------------------------- Events --------------------------------- */
const emit = defineEmits<{
  (event: 'submit', values: Record<string, string>): void
  (event: 'cancel'): void
}>()

/* --------------------------------- State ---------------------------------- */
const values = reactive<Record<string, string>>(getDefaults(props.fields))
const form = useTemplateRef<HTMLFormElement>('form')

/* --------------------------------- Hooks ---------------------------------- */
onMounted(() => {
  form.value?.querySelector<HTMLElement>('input, button')?.focus()
})

/* -------------------------------- Handlers -------------------------------- */
function onSubmit() {
  emit('submit', { ...values })
}

function onCancel() {
  emit('cancel')
}

function getChoices(field: Field) {
  return field.options.map((option) => ({ id: option, text: option }))
}
</script>

<template>
  <div class="templates-form" @keydown.esc.stop="onCancel">
    <div class="templates-form__window" role="dialog" aria-modal="true" :aria-label="title">
      <form ref="form" class="templates-form__body" @submit.prevent="onSubmit">
        <h2 class="templates-form__title">{{ title }}</h2>
        <div
          v-for="field in fields"
          :key="field.label"
          class="templates-form__field"
          :data-field="field.label"
        >
          <label class="templates-form__label">
            <span>{{ field.label }}</span>
            <Select
              v-if="field.kind === 'select'"
              :model-value="values[field.label] ?? ''"
              :choices="getChoices(field)"
              :name="field.label"
              @update:model-value="values[field.label] = $event"
            />
            <input
              v-else
              v-model="values[field.label]"
              class="templates-form__input"
              type="text"
              :aria-label="field.label"
            />
          </label>
        </div>
        <div class="templates-form__actions">
          <Button type="button" variant="outline" data-role="cancel" @click="onCancel">
            {{ words.cancel }}
          </Button>
          <Button type="submit" data-role="create">{{ words.create }}</Button>
        </div>
      </form>
    </div>
  </div>
</template>
