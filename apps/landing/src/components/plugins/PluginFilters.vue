<script setup lang="ts">
/**
 * The search and the filters of the plugin catalog. The cards are the page's own HTML; the bar
 * hides the ones its filter does not pass.
 */
import { computed, onMounted, onUnmounted, reactive, ref, watch } from 'vue'
import { Select, Switch } from '@numen/ui'
import {
  type CatalogFilter,
  showMatchingCards,
  watchPageLanguage,
} from '../../scripts/plugin-catalog'
import { getDictionary } from '../../scripts/site-controls'

type Choice = { id: string; text: string }

/* --------------------------------- Props ---------------------------------- */
const props = defineProps<{
  categories: readonly Choice[]
  apps: readonly Choice[]
}>()

/* --------------------------------- State ---------------------------------- */
const filter = reactive<CatalogFilter>({
  query: '',
  category: '',
  app: '',
  price: '',
  isOfficialOnly: false,
})
const lang = ref('en')
const words = computed(() => getDictionary(lang.value))
const categoryChoices = computed(() => [
  { id: '', text: words.value.plugins_category_all ?? '' },
  ...props.categories,
])
const appChoices = computed(() => [
  { id: '', text: words.value.plugins_app_all ?? '' },
  ...props.apps,
])
const priceChoices = computed(() => [
  { id: '', text: words.value.plugins_price_all ?? '' },
  { id: 'free', text: words.value.plugins_free ?? '' },
  { id: 'paid', text: words.value.plugins_paid ?? '' },
])
let stopWatchingLanguage: (() => void) | null = null

/* --------------------------------- Hooks ---------------------------------- */
watch(filter, () => showMatchingCards(filter))

onMounted(() => {
  stopWatchingLanguage = watchPageLanguage(onChangeLanguage)
  showMatchingCards(filter)
})

onUnmounted(() => stopWatchingLanguage?.())

/* -------------------------------- Handlers -------------------------------- */
function onChangeLanguage(next: string): void {
  lang.value = next
}

function onClearQuery(): void {
  filter.query = ''
}
</script>

<template>
  <div class="plugin-filters" role="search">
    <span class="plugin-filters__search">
      <svg class="plugin-filters__search-icon" viewBox="0 0 24 24" aria-hidden="true">
        <circle cx="11" cy="11" r="7" />
        <path d="m20 20-3.5-3.5" />
      </svg>
      <input
        v-model="filter.query"
        type="search"
        autocomplete="off"
        :placeholder="words.plugins_search"
        :aria-label="words.plugins_search"
        @keydown.esc="onClearQuery"
      />
    </span>
    <Select
      v-model="filter.category"
      class="plugin-filters__select"
      :choices="categoryChoices"
      :name="words.plugins_category"
      :aria-label="words.plugins_category"
    />
    <Select
      v-model="filter.app"
      class="plugin-filters__select"
      :choices="appChoices"
      :name="words.plugins_app"
      :aria-label="words.plugins_app"
    />
    <Select
      v-model="filter.price"
      class="plugin-filters__select"
      :choices="priceChoices"
      :name="words.plugins_price"
      :aria-label="words.plugins_price"
    />
    <label class="plugin-filters__switch">
      <Switch v-model="filter.isOfficialOnly" />
      <span>{{ words.plugins_official_only }}</span>
    </label>
  </div>
</template>
