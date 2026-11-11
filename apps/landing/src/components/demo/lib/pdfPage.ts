import { BOLTZMANN_PATH, PDF_DOCS, type PdfDoc } from './documents'

export const PAPER = { width: 612, height: 792 }
const MARGIN = 84
const FIRST = 132
const LEADING = 27

const DOCS_BY_PATH: ReadonlyMap<string, PdfDoc> = new Map(Object.entries(PDF_DOCS))

export const getPdfDoc = (path: string): PdfDoc =>
  DOCS_BY_PATH.get(path) ?? PDF_DOCS[BOLTZMANN_PATH]

function escapeXml(line: string): string {
  return line.replace(/&/g, '&amp;').replace(/</g, '&lt;').replace(/>/g, '&gt;')
}

function drawLine(line: string, index: number, isTitle: boolean): string {
  const y = FIRST + index * LEADING
  const weight = isTitle ? ' font-weight="600" letter-spacing="1.2"' : ''
  return `<text x="${MARGIN}" y="${y}" font-family="Georgia, serif" font-size="16"${weight} fill="#1b1b1b">${escapeXml(line)}</text>`
}

/** The picture of one page: the sample leaves of the paper repeat from its first page on. */
export function getPagePicture(path: string, pageNumber: number): string {
  const doc = getPdfDoc(path)
  const count = doc.pages.length
  const leaf = (((pageNumber - doc.firstPage) % count) + count) % count
  const lines = (doc.pages[leaf] ?? []).map((line, index) =>
    drawLine(line, index, index === 0 && leaf === 0),
  )
  const number = `<text x="${PAPER.width / 2}" y="${PAPER.height - 54}" text-anchor="middle" font-family="Georgia, serif" font-size="13" fill="#5a5a5a">${pageNumber + 1}</text>`
  const svg = `<svg xmlns="http://www.w3.org/2000/svg" width="${PAPER.width}" height="${PAPER.height}"><rect width="100%" height="100%" fill="#fbfaf7"/>${lines.join('')}${number}</svg>`
  return `data:image/svg+xml;charset=utf-8,${encodeURIComponent(svg)}`
}

/** A box over a page, as fractions of its width and height. */
export interface Highlight {
  readonly minX: number
  readonly maxX: number
  readonly minY: number
  readonly maxY: number
}

function overLines(from: number, to: number, ends: number): Highlight {
  return {
    minX: (MARGIN - 4) / PAPER.width,
    maxX: (MARGIN + ends) / PAPER.width,
    minY: (FIRST + from * LEADING - 18) / PAPER.height,
    maxY: (FIRST + to * LEADING + 6) / PAPER.height,
  }
}

const HIGHLIGHTS: readonly Highlight[] = [overLines(8, 9, 420), overLines(10, 11, 360)]

/** The passages marked on a page: only the page the paper opens on has any. */
export function getHighlights(path: string, pageIndex: number): readonly Highlight[] {
  return pageIndex === getPdfDoc(path).firstPage ? HIGHLIGHTS : []
}
