<script setup lang="ts">
/**
 * A secret that does not exist yet, drawn as a saved one: a name to type, a masked field and a
 * discard button, or the disabled actions button where it cannot be discarded. The secret is
 * created, under the name typed, when a value is entered. Its plugin list stays disabled until the
 * secret exists. The empty secret of a group a plugin asks for has no way to be discarded. A row
 * with autofocus takes the focus into its name when mounted, else into its value.
 */
import { onMounted, ref, useTemplateRef } from 'vue'
import { Ellipsis, Trash2 } from '@lucide/vue'
import { trimName } from '@/entities/settings'
import { WORDS as words } from '../../words'
import SecretActionButton from './SecretActionButton.vue'
import SecretNameInput from './SecretNameInput.vue'
import SecretPermissions from './SecretPermissions.vue'
import SecretRowFrame from './SecretRowFrame.vue'
import SecretValueInput from './SecretValueInput.vue'

/* ----------------------------- Props & Emits ------------------------------ */
const props = defineProps<{
  name: string
  canDiscard: boolean
  hasName: boolean
  hasAutofocus?: boolean
  submit: (name: string, value: string) => Promise<boolean>
}>()

const emit = defineEmits<{
  (e: 'discard'): void
}>()

/* --------------------------------- State ---------------------------------- */
const typedName = ref(props.name)
const nameInput = useTemplateRef<InstanceType<typeof SecretNameInput>>('nameInput')
const valueInput = useTemplateRef<InstanceType<typeof SecretValueInput>>('valueInput')

onMounted(() => {
  if (!props.hasAutofocus) return
  if (props.hasName) nameInput.value?.focus()
  else valueInput.value?.focus()
})

/* -------------------------------- Handlers -------------------------------- */
function onSubmit(value: string): Promise<boolean> {
  const name = trimName(typedName.value)
  if (name === '') return Promise.resolve(false)
  return props.submit(name, value)
}
</script>

<template>
  <SecretRowFrame :has-name="hasName" :label="typedName">
    <template #name>
      <SecretNameInput ref="nameInput" v-model="typedName" @cancel="typedName = name" />
    </template>

    <template #value>
      <SecretValueInput
        ref="valueInput"
        :label="words.secretValue(typedName)"
        :has-value="false"
        :submit="onSubmit"
      />
    </template>

    <template #permissions>
      <SecretPermissions :declarations="[]" />
    </template>

    <template #actions>
      <SecretActionButton
        v-if="canDiscard"
        :aria-label="words.secretDiscard"
        :title="words.secretDiscard"
        @click="emit('discard')"
      >
        <Trash2 :size="14" aria-hidden="true" />
      </SecretActionButton>
      <SecretActionButton
        v-else
        disabled
        :aria-label="words.secretActions"
        :title="words.secretSaveFirst"
      >
        <Ellipsis :size="14" aria-hidden="true" />
      </SecretActionButton>
    </template>
  </SecretRowFrame>
</template>
