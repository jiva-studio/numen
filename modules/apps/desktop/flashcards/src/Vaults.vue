<script setup lang="ts">
/**
 * The front door: every vault the installation holds, and what each owes today.
 *
 * It is the screen the editor opens on, drawn from the same component. The
 * screen knows nothing about cards; what stands at the end of a row is put
 * there from here.
 */
import { computed } from 'vue'
import { Owed, Waiting, Welcome } from '@numen/ui'
import type { Held } from '@numen/ui'
import type { Owing } from './core'

const props = defineProps<{
  vaults: readonly Owing[]
  counting: boolean
  version: string
}>()

defineEmits<{ (event: 'choose', vaultId: string): void }>()

/** A vault as a row of the list. One that could not be counted says why. */
const listed = computed<readonly Held[]>(() =>
  props.vaults.map((one) => ({
    id: one.vaultId,
    name: one.name,
    path: one.path,
    ...(one.unread ? { detail: one.unread } : {}),
  })),
)

/**
 * How many cards a vault has waiting, by the identity of the vault. A vault
 * that could not be counted is absent: what stands in its row is why, and not a
 * number.
 */
const waiting = computed(
  () =>
    new Map(
      props.vaults.filter((one) => !one.unread).map((one) => [one.vaultId, one.due + one.new]),
    ),
)
</script>

<template>
  <Welcome
    name="flashcards"
    :vaults="counting ? [] : listed"
    heading="Vaults"
    :version="version"
    @opens="$emit('choose', $event)"
  >
    <!-- The list is where the room is shortest, so the number stands alone. -->
    <template #vault="{ vault }">
      <Owed v-if="waiting.has(vault.id)" :waiting="waiting.get(vault.id)!" bare />
    </template>

    <!-- The cards of every vault are being counted, which is the one thing
         the screen has to say until they are. -->
    <template v-if="counting" #waiting>
      <p class="vaults__counting" role="status">
        <Waiting />
        Counting the vaults
      </p>
    </template>
  </Welcome>
</template>

<style scoped>
/* The screen's own quiet voice, which is what everything it says beside a row
   is set in. */
.vaults__counting {
  display: flex;
  margin: 0;
  align-items: center;
  gap: var(--numen-inset);
  color: var(--numen-edge-label);
  font-size: var(--numen-edge-label-size);
}
</style>
