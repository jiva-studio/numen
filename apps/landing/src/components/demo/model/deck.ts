import { ref } from 'vue'
import {
  CONCEPT_MODEL,
  INITIAL_CARDS,
  INITIAL_SECTIONS,
  LAW_MODEL,
  PRESET_CHOICES,
  type CardModel,
  type DeckCard,
  type ReviewPreset,
} from '../lib/deck'

const CARD_MODELS: readonly CardModel[] = [CONCEPT_MODEL, LAW_MODEL]

/** The deck of the deck tab: its sections and cards, and the review preset that schedules them. */
export function createDeck() {
  const cards = ref<readonly DeckCard[]>(INITIAL_CARDS)
  const sections = INITIAL_SECTIONS
  const preset = ref<ReviewPreset>(PRESET_CHOICES[1] ?? { path: '', name: 'Defaults' })
  let count = 1

  return {
    cards,
    sections,
    presets: PRESET_CHOICES,
    preset,
    models: CARD_MODELS,
    setPreset(path: string) {
      const found = PRESET_CHOICES.find((one) => one.path === path)
      if (found) preset.value = found
    },
    addCard(modelName: string, section: string) {
      const model = CARD_MODELS.find((one) => one.name === modelName)
      if (!model) return
      count += 1
      const filled = model.fields.map((field) => ({ field, text: '' }))
      cards.value = [...cards.value, { id: `card-${count}`, section, cardModel: modelName, filled }]
    },
    removeCard(id: string) {
      cards.value = cards.value.filter((one) => one.id !== id)
    },
    writeField(id: string, field: string, text: string) {
      cards.value = cards.value.map((card) =>
        card.id === id
          ? { ...card, filled: card.filled.map((one) => (one.field === field ? { ...one, text } : one)) }
          : card,
      )
    },
  }
}

export type Deck = ReturnType<typeof createDeck>
