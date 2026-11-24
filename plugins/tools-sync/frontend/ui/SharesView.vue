<script setup lang="ts">
/**
 * The card "Shared folders" of the plugin's settings: the folders this account shares with the devices that
 * read them, and the folders other people share with this account. It only lists; a folder is shared from the
 * file tree and a shared folder is accepted by a command.
 */
import { FolderOpen } from '@lucide/vue'
import { Button, IslandCard, type MenuItem } from '@numen/ui'
import type { DeviceRow, GivenRow, SharesView } from '../model/shares'
import { WORDS as words } from '../words'
import EndSyncDialog from './EndSyncDialog.vue'
import EmptyBlock from './EmptyBlock.vue'
import RowMenu from './RowMenu.vue'
import SyncRow from './SyncRow.vue'
import SyncStatus from './SyncStatus.vue'

/* --------------------------------- Props ---------------------------------- */
const props = defineProps<{ view: SharesView }>()

/* --------------------------------- State ---------------------------------- */
const ITEMS: readonly MenuItem[] = [{ id: 'stop', text: words.stopSharing, group: 'danger' }]
const LEAVE_ITEMS: readonly MenuItem[] = [{ id: 'leave', text: words.leaveShare, group: 'danger' }]

/* -------------------------------- Handlers -------------------------------- */
function onStop(row: GivenRow): void {
  props.view.stopSharing(row).catch(props.view.fail)
}

function onConfirmLeave(): void {
  props.view.confirmLeave().catch(props.view.fail)
}

function onRemove(row: GivenRow, device: DeviceRow): void {
  props.view.removeAccess(row, device).catch(props.view.fail)
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
    <EmptyBlock
      v-if="view.gives.length === 0 && view.receives.length === 0"
      :icon="FolderOpen"
      :title="words.noSharedTitle"
      :text="words.noSharedText"
    />
    <template v-if="view.gives.length > 0">
      <h3 class="sync-card__group">{{ words.sharedByYou }}</h3>
      <p class="sync-card__lead">{{ words.sharedByYouText }}</p>
      <ul class="sync-card__list" :aria-label="words.sharedByYou">
        <li v-for="row in view.gives" :key="`${row.vaultId}/${row.folder}`">
          <ul class="sync-card__list">
            <SyncRow :name="row.folder">
              <span class="sync-row__detail">
                {{ row.isCreating ? words.preparingShare : words.readOnly }}
              </span>
              <template #actions>
                <RowMenu
                  :label="words.rowActions(row.folder)"
                  :items="ITEMS"
                  @choose="onStop(row)"
                />
              </template>
            </SyncRow>
            <SyncRow v-for="device in row.devices" :key="device.fingerprint" :name="device.name">
              <template #actions>
                <Button variant="ghost" size="small" @click="onRemove(row, device)">
                  {{ words.removeAccess }}
                </Button>
              </template>
            </SyncRow>
          </ul>
          <p v-if="row.devices.length === 0" class="sync-row__detail">{{ words.noDevices }}</p>
        </li>
      </ul>
    </template>
    <template v-if="view.receives.length > 0">
      <h3 class="sync-card__group">{{ words.sharedWithYou }}</h3>
      <p class="sync-card__lead">{{ words.sharedWithYouText }}</p>
      <ul class="sync-card__list" :aria-label="words.sharedWithYou">
        <SyncRow v-for="row in view.receives" :key="row.vaultId" :name="row.name">
          <SyncStatus :text="row.stateLabel" :tone="row.tone" />
          <span class="sync-row__detail">
            {{ words.readOnly }} · {{ words.fromOwner(row.ownerShort) }}
          </span>
          <template #actions>
            <RowMenu
              :label="words.rowActions(row.name)"
              :items="LEAVE_ITEMS"
              @choose="view.askLeave(row)"
            />
          </template>
        </SyncRow>
      </ul>
    </template>
    <EndSyncDialog
      v-if="view.pendingLeave !== null"
      kind="leave"
      @confirm="onConfirmLeave"
      @cancel="view.cancelLeave"
    />
  </IslandCard>
</template>
