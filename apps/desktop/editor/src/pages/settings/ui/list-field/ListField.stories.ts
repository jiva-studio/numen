/**
 * The control of a list setting in a settings row. The story holds the list the way the page does:
 * an item the control adds goes last, an item it removes leaves the list. Every value here is invented.
 */
import type { Meta, StoryObj } from '@storybook/vue3-vite'
import { ref } from 'vue'
import { expect, fn, userEvent, within } from 'storybook/test'
import SettingRow from '../setting-row/SettingRow.vue'
import { WORDS } from '../../words'
import ListField from './ListField.vue'

interface Knobs {
  items: readonly string[]
  onAdd: (item: string) => void
  onRemove: (item: string) => void
}

const TITLE = 'Ignored folders'

const render = (args: Knobs) => ({
  components: { ListField, SettingRow },
  setup: () => {
    const items = ref<readonly string[]>([...args.items])
    const onAdd = (item: string) => {
      args.onAdd(item)
      items.value = [...items.value, item]
    }
    const onRemove = (item: string) => {
      args.onRemove(item)
      items.value = items.value.filter((one) => one !== item)
    }
    return { items, words: WORDS, onAdd, onRemove }
  },
  template: `
    <div class="numen" style="padding:1rem;background:var(--numen-surface)">
      <SettingRow v-slot="{ labelledBy, describedBy }" at="folders" name="${TITLE}" detail="Folders whose notes are not searched for tasks.">
        <ListField
          :items="items"
          :labelled-by="labelledBy"
          :described-by="describedBy"
          :words="words"
          @add="onAdd"
          @remove="onRemove"
        />
      </SettingRow>
    </div>
  `,
})

const meta: Meta<Knobs> = {
  title: 'Window/List field',
  component: ListField,
  parameters: { layout: 'fullscreen' },
  args: { items: ['templates', 'archive/old'], onAdd: fn(), onRemove: fn() },
  render,
}

export default meta
type Story = StoryObj<Knobs>

const getField = (canvasElement: HTMLElement) => {
  const group = within(canvasElement).getByRole('group', { name: TITLE })
  const field = within(group)
  return {
    group,
    field,
    input: () => field.getByRole<HTMLInputElement>('textbox', { name: TITLE }),
    add: () => field.getByRole('button', { name: WORDS.listAdd }),
    remove: (item: string) => field.getByRole('button', { name: WORDS.listRemove(item) }),
    listItems: () =>
      field
        .queryAllByRole('button', { name: /^Remove / })
        .map((one) => one.getAttribute('aria-label')?.replace(/^Remove /, '')),
  }
}

export const ItemRows: Story = {
  name: 'The list setting shows one row per item with a remove button',
  play: async ({ canvasElement }) => {
    const { field, listItems, input, add } = getField(canvasElement)

    await expect(listItems()).toStrictEqual(['templates', 'archive/old'])
    await expect(field.getByText('templates')).toBeVisible()
    await expect(field.getByText('archive/old')).toHaveAttribute('title', 'archive/old')
    await expect(input()).toBeVisible()
    await expect(add()).toBeVisible()
  },
}

export const EmptyList: Story = {
  name: 'An empty list shows only the input and the add button',
  args: { items: [] },
  play: async ({ canvasElement }) => {
    const { field, input } = getField(canvasElement)

    await expect(field.getAllByRole('button').map((one) => one.textContent?.trim())).toStrictEqual([
      WORDS.listAdd,
    ])
    await expect(input()).toHaveAttribute('placeholder', WORDS.listNew)
  },
}

export const AddByEnter: Story = {
  name: 'Pressing Enter in the input adds the trimmed text as the last item and clears the input',
  play: async ({ canvasElement, args }) => {
    const { listItems, input } = getField(canvasElement)

    await userEvent.type(input(), '  drafts  {Enter}')

    await expect(args.onAdd).toHaveBeenCalledTimes(1)
    await expect(args.onAdd).toHaveBeenCalledWith('drafts')
    await expect(listItems()).toStrictEqual(['templates', 'archive/old', 'drafts'])
    await expect(input()).toHaveValue('')
  },
}

export const AddByButton: Story = {
  name: 'Pressing the add button adds the text as the last item',
  play: async ({ canvasElement, args }) => {
    const { listItems, input, add } = getField(canvasElement)

    await userEvent.type(input(), 'drafts')
    await userEvent.click(add())

    await expect(args.onAdd).toHaveBeenCalledTimes(1)
    await expect(args.onAdd).toHaveBeenCalledWith('drafts')
    await expect(listItems()).toStrictEqual(['templates', 'archive/old', 'drafts'])
    await expect(input()).toHaveValue('')
  },
}

export const EmptyText: Story = {
  name: 'Empty text adds nothing and the add button is disabled',
  play: async ({ canvasElement, args }) => {
    const { listItems, input, add } = getField(canvasElement)

    await expect(add()).toBeDisabled()
    await userEvent.type(input(), '   ')
    await expect(add()).toBeDisabled()
    await userEvent.keyboard('{Enter}')

    await expect(args.onAdd).not.toHaveBeenCalled()
    await expect(listItems()).toStrictEqual(['templates', 'archive/old'])
    await expect(input()).toHaveValue('   ')
  },
}

export const TextAlreadyInTheList: Story = {
  name: 'Text already in the list adds nothing and says it is already in the list',
  play: async ({ canvasElement, args }) => {
    const { field, listItems, input, add } = getField(canvasElement)

    await userEvent.type(input(), ' templates {Enter}')

    await expect(args.onAdd).not.toHaveBeenCalled()
    await expect(listItems()).toStrictEqual(['templates', 'archive/old'])
    await expect(input()).toHaveValue(' templates ')
    await expect(field.getByRole('alert')).toHaveTextContent(WORDS.listItemTwice('templates'))

    await userEvent.click(add())
    await expect(args.onAdd).not.toHaveBeenCalled()

    await userEvent.type(input(), '2')
    await expect(field.queryByRole('alert')).toBeNull()
  },
}

export const RemoveOneItem: Story = {
  name: 'Pressing the remove button of an item removes only that item',
  args: { items: ['templates', 'archive/old', 'drafts'] },
  play: async ({ canvasElement, args }) => {
    const { listItems, remove } = getField(canvasElement)

    await userEvent.click(remove('archive/old'))

    await expect(args.onRemove).toHaveBeenCalledTimes(1)
    await expect(args.onRemove).toHaveBeenCalledWith('archive/old')
    await expect(listItems()).toStrictEqual(['templates', 'drafts'])
  },
}

export const EscapeClearsTheInput: Story = {
  name: 'Escape in the input clears it',
  play: async ({ canvasElement, args }) => {
    const { listItems, input } = getField(canvasElement)

    await userEvent.type(input(), 'drafts')
    await userEvent.keyboard('{Escape}')

    await expect(input()).toHaveValue('')
    await expect(args.onAdd).not.toHaveBeenCalled()
    await expect(listItems()).toStrictEqual(['templates', 'archive/old'])
  },
}
