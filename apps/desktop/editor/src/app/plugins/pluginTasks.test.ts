import { describe, expect, it, vi } from 'vitest'
import { nextTick, shallowRef } from 'vue'
import { createPluginTasks } from './pluginTasks'
import type { Task } from '@/shared/notices/task'

const createTask = (id: string, pluginId: string): Task => ({
  id,
  label: 'Working',
  about: `notes/${id}.md`,
  completedCount: 1,
  total: 2,
  counting: 'items',
  error: '',
  isUserStarted: false,
  pluginId,
  pluginName: pluginId ? 'Example' : '',
})

describe('createPluginTasks', () => {
  it('watch reports only the tasks of the calling plugin', () => {
    const tasks = shallowRef<readonly Task[]>([
      createTask('a', 'md.example.mine'),
      createTask('b', 'md.example.other'),
      createTask('c', ''),
    ])
    const handler = vi.fn()

    createPluginTasks(tasks, 'md.example.mine').watch(handler)

    expect(handler).toHaveBeenCalledWith([
      {
        id: 'a',
        pluginId: 'md.example.mine',
        pluginName: 'Example',
        title: 'Working',
        about: 'notes/a.md',
        done: 1,
        total: 2,
        error: '',
      },
    ])
  })

  it('watchAll reports the tasks of every plugin and not the core tasks', () => {
    const tasks = shallowRef<readonly Task[]>([
      createTask('a', 'md.example.mine'),
      createTask('b', 'md.example.other'),
      createTask('c', ''),
    ])
    const handler = vi.fn()

    createPluginTasks(tasks, 'md.example.mine').watchAll(handler)

    expect(handler.mock.lastCall?.[0].map((one: { id: string }) => one.id)).toEqual(['a', 'b'])
  })

  it('reports each change until disposed', async () => {
    const tasks = shallowRef<readonly Task[]>([])
    const handler = vi.fn()
    const watching = createPluginTasks(tasks, 'md.example.mine').watch(handler)

    tasks.value = [createTask('a', 'md.example.mine')]
    await nextTick()
    expect(handler).toHaveBeenCalledTimes(2)

    watching.dispose()
    tasks.value = []
    await nextTick()
    expect(handler).toHaveBeenCalledTimes(2)
  })
})
