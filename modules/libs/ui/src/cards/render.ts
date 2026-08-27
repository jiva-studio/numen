/**
 * What a card is written with, drawn.
 *
 * A face is markdown, and the tags a person writes among the marks are tags.
 * A deck may come from another person, so what comes out is measured against
 * what a card may be drawn with before it reaches the screen.
 */
import MarkdownIt from 'markdown-it'
import { safe } from './safe'

const marks = new MarkdownIt({ html: true, linkify: true })

/** Text as the HTML a card draws, with nothing in it that a card may not. */
export const drawn = (text: string): string => safe(marks.render(text))
