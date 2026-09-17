<script setup lang="ts">
/**
 * The front door: every vault the installation holds, and what each owes today.
 *
 * It is the screen the editor opens on, drawn from the same component. The
 * screen knows nothing about cards; what stands at the end of a row is put
 * there from here.
 *
 * The list is on screen before any of it is counted, and each vault's number
 * arrives on its own. A vault whose count has not arrived shows the shape that
 * number will take and is not opened until it has one.
 */
import { computed } from 'vue'
import { DueCount, Spinner, WelcomePage } from '@numen/ui'
import type { VaultRow } from '@numen/ui'
import type { VaultCardsDue } from '@/entities/vault'

const props = defineProps<{
  vaults: readonly VaultCardsDue[]
  counting: boolean
  version: string
}>()

defineEmits<{ (event: 'choose', vault: string): void }>()

/**
 * A vault as a row of the list. One being read into the index says so, and one
 * that could not be counted says why.
 */
const listed = computed<readonly VaultRow[]>(() =>
  props.vaults.map((one) => {
    const said = one.reading ? 'Reading the vault' : one.unread
    return {
      id: one.vault,
      name: one.name,
      path: one.path,
      isWorking: !one.counted,
      ...(said ? { detail: said } : {}),
    }
  }),
)

/**
 * How many cards a vault has waiting, by the identity of the vault, and nothing
 * for a vault whose count has not arrived. A vault that could not be counted is
 * absent as well: what stands in its row is why, and not a number.
 */
const waiting = computed(
  () =>
    new Map(
      props.vaults
        .filter((one) => !one.unread && !one.reading)
        .map((one) => [one.vault, one.counted ? one.due + one.new : null]),
    ),
)
</script>

<template>
  <WelcomePage
    name="flashcards"
    :vaults="listed"
    heading="Vaults"
    :version="version"
    @open="$emit('choose', $event)"
  >
    <!-- The list is where the room is shortest, so the number stands alone. -->
    <template #vault="{ vault }">
      <DueCount v-if="waiting.has(vault.id)" :due="waiting.get(vault.id) ?? null" bare />
    </template>

    <!-- Nothing is known about the installation yet, not even which vaults it
         holds, which is the one thing the screen has to say until it is. -->
    <template v-if="counting" #waiting>
      <p class="vaults__counting" role="status">
        <Spinner />
        Reading the vaults
      </p>
    </template>
  </WelcomePage>
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
  font-size: var(--numen-text-1);
}
</style>
