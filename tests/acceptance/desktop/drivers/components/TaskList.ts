/** The list of background tasks, opened from the status bar. */
import { expect, type Locator, type Page } from 'playwright/test'

const TASK_TRIGGER = '.tasks-status__trigger'
const TASK_LIST = '.task-popover'
const TASK = '.task-item'
const FAILED_TASK = '.task-item--failed'

export class TaskList {
  constructor(private readonly page: Page) {}

  /** Opens the list, where the failed tasks are listed, unless it is open. */
  async open(): Promise<Locator> {
    const list = this.page.locator(TASK_LIST)
    if (!(await list.isVisible())) {
      await this.page.locator(TASK_TRIGGER).click()
      await expect(list).toBeVisible()
    }
    return list
  }

  /** The rows with the text. The list closes when the focus leaves it, so every look opens it again. */
  async find(text: string, isFailed: boolean): Promise<Locator> {
    const rows = (await this.open()).locator(isFailed ? FAILED_TASK : TASK)
    return rows.filter({ hasText: text })
  }

  async count(text: string, isFailed: boolean): Promise<number> {
    return (await this.find(text, isFailed)).count()
  }

  async cancel(text: string): Promise<void> {
    const task = (await this.find(text, false)).first()
    await task.hover()
    await task.getByRole('button', { name: 'Cancel task' }).click()
  }

  /** Whether a task of the list failed; the list stays as it was found. */
  async hasFailed(): Promise<boolean> {
    const list = this.page.locator(TASK_LIST)
    const wasOpen = await list.isVisible()
    if (!wasOpen) await this.page.locator(TASK_TRIGGER).click()
    const hasFailed = (await list.locator(FAILED_TASK).count()) > 0
    if (!wasOpen) await this.page.keyboard.press('Escape')
    return hasFailed
  }
}
