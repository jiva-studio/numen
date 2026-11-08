<script setup lang="ts">
/**
 * A secret that does not exist yet, drawn as a saved one: a name to type, a masked field and a
 * discard button, or the disabled actions button where it cannot be discarded. The secret is created, under the name typed, when a value is entered. Its
 * plugin list stays disabled until the secret exists. The empty secret of a group a plugin asks
 * for has no way to be discarded.
 */
import { ref } from 'vue'
import { Ellipsis, Trash2 } from '@lucide/vue'
import { Button } from '@numen/ui'
import { trimGroupName } from '../../lib/trimGroupName'
import { WORDS as words } from '../../words'
import SecretNameInput from './SecretNameInput.vue'
import SecretPermissions from './SecretPermissions.vue'
import SecretRowFrame from './SecretRowFrame.vue'
import SecretValueInput from './SecretValueInput.vue'

/* ----------------------------- Props & Emits ------------------------------ */
const props = defineProps<{
  name: string
  canDiscard: boolean
  /** Whether the row shows its name, as the group holds more than one secret. */
  hasName: boolean
  /** Creates the secret and resolves true when it was created; the field then keeps a failed value. */
  submit?: ((name: string, value: string) => Promise<boolean>) | undefined
}>()

const emit = defineEmits<{
  (e: 'create', name: string, value: string): void
  (e: 'discard'): void
}>()

/* --------------------------------- State ---------------------------------- */
const typedName = ref(props.name)

/* -------------------------------- Handlers -------------------------------- */
async function onSubmit(value: string): Promise<boolean> {
  const name = trimGroupName(typedName.value)
  if (name === '' || !props.submit) return false
  return props.submit(name, value)
}
</script>

<template>
  <SecretRowFrame :has-name="hasName">
    <template #name>
      <SecretNameInput v-model="typedName" @cancel="typedName = name" />
    </template>

    <template #value>
      <SecretValueInput
        :label="words.secretValue(typedName)"
        :placeholder="words.keyPlaceholder"
        :submit="submit ? onSubmit : undefined"
        @commit="(value) => emit('create', trimGroupName(typedName), value)"
      />
    </template>

    <template #permissions>
      <SecretPermissions :declarations="[]" />
    </template>

    <template #actions>
      <Button
        v-if="canDiscard"
        variant="outline"
        class="secret-action"
        size="icon"
        :aria-label="words.secretDiscard"
        :title="words.secretDiscard"
        @click="emit('discard')"
      >
        <Trash2 :size="14" aria-hidden="true" />
      </Button>
      <Button
        v-else
        variant="outline"
        class="secret-action"
        size="icon"
        disabled
        :aria-label="words.secretActions"
        :title="words.secretSaveFirst"
      >
        <Ellipsis :size="14" aria-hidden="true" />
      </Button>
    </template>
  </SecretRowFrame>
</template>
