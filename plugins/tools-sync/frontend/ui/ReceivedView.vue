<script setup lang="ts">
/**
 * The card "Shared folders" of the plugin's settings: the folders other people share with this account, the
 * identity code of the account to send to a person who wants to share one, and the header action that
 * accepts an invitation. A folder is shared from the file tree.
 */
import { Copy, FolderOpen } from '@lucide/vue'
import { Button, IslandCard } from '@numen/ui'
import type { ReceivedView } from '../model/shares'
import { WORDS as words } from '../words'
import EmptyBlock from './EmptyBlock.vue'
import SyncRow from './SyncRow.vue'
import SyncStatus from './SyncStatus.vue'

/* --------------------------------- Props ---------------------------------- */
const props = defineProps<{
  view: ReceivedView
}>()

/* -------------------------------- Handlers -------------------------------- */
function onCopyIdentity(fingerprint: string): void {
  props.view.copyFingerprint(fingerprint).catch(props.view.fail)
}
</script>

<template>
  <IslandCard
    class="sync-card"
    :title="words.sharedFoldersTitle"
    :aria-label="words.sharedFoldersTitle"
  >
    <p class="sync-card__lead">{{ words.sharedFoldersText }}</p>
    <p v-if="view.error !== ''" class="sync-card__note sync-card__note--alarm" role="alert">
      {{ view.error }}
    </p>
    <h3 v-if="view.rows.length > 0" class="sync-card__group">{{ words.sharedWithYou }}</h3>
    <p v-if="view.rows.length > 0" class="sync-card__lead">{{ words.sharedWithYouText }}</p>
    <EmptyBlock
      v-if="view.rows.length === 0"
      :icon="FolderOpen"
      :title="words.noSharedTitle"
      :text="words.noSharedText"
    />
    <ul v-else class="sync-card__list" :aria-label="words.sharedWithYou">
      <SyncRow v-for="row in view.rows" :key="row.vaultId" :name="row.name">
        <SyncStatus :text="row.stateLabel" :tone="row.tone" />
        <span class="sync-row__detail">
          {{ words.readOnly }} · {{ words.fromPerson(row.ownerShort) }}
        </span>
        <template #actions>
          <Button
            variant="ghost"
            size="small"
            :aria-label="words.copyIdentity"
            @click="onCopyIdentity(row.ownerFingerprint)"
          >
            <Copy :size="14" aria-hidden="true" />
          </Button>
        </template>
      </SyncRow>
    </ul>
  </IslandCard>
</template>
