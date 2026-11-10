import { getShelfBase } from '../components/page/site'

interface Shelf {
  version?: string
  build?: number
  commit?: string
  released?: string
  files?: readonly { name: string; url: string; size: number }[]
}

async function fetchShelf(): Promise<Shelf | null> {
  try {
    const base = getShelfBase(import.meta.env.PUBLIC_NUMEN_DOWNLOADS_BASE)
    const answer = await fetch(new URL('latest.json', `${base}/`), { cache: 'no-cache' })
    return answer.ok ? ((await answer.json()) as Shelf) : null
  } catch {
    return null
  }
}

function describeVersion(shelf: Shelf & { version: string }): string {
  const day = new Date(shelf.released ?? '')
  const when = Number.isNaN(day.valueOf())
    ? ''
    : ` — ${day.toLocaleDateString('en-GB', { day: 'numeric', month: 'long', year: 'numeric' })}`
  const commit = shelf.commit ? `, ${shelf.commit}` : ''
  const which = shelf.build ? ` (build ${shelf.build}${commit})` : ''
  return `Version ${shelf.version}${which}${when}`
}

function updateTake(take: HTMLAnchorElement, shelf: Shelf): void {
  const file = shelf.files?.find((one) => one.name === take.dataset['file'])
  if (!file?.url) {
    take.hidden = true
    return
  }
  take.href = file.url
  const weight = take.querySelector('.take__weight')
  if (weight && file.size) weight.textContent = `${Math.round(file.size / 1e6)} MB`
}

/** Points the download links at the latest release and writes its version under them. */
export async function initReleaseLinks(): Promise<void> {
  const takes = [...document.querySelectorAll<HTMLAnchorElement>('.take-pill[data-file]')]
  if (takes.length === 0) return
  const shelf = await fetchShelf()
  if (!shelf) return
  for (const take of takes) updateTake(take, shelf)
  const note = document.querySelector('.version-note')
  if (note && shelf.version) {
    note.textContent = describeVersion({ ...shelf, version: shelf.version })
    note.removeAttribute('hidden')
  }
}
