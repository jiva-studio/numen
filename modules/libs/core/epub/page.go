package epub

import "unicode/utf8"

// A reflowable book has no pages until it is drawn, so its pages are counted
// over the text: a run of it apiece, numbered from one. They are not Pages, which
// are the pages of the printed book this file was made from.
//
// lettersPerPage is how many letters stand on one. A page of a paperback holds
// about this many.
const lettersPerPage = 1024

// pageBytes is how many bytes of one book's text make a page. The text is bytes
// and a page is letters, so the letter is measured in this book's own script.
func pageBytes(text string) int {
	letters := utf8.RuneCountInString(text)
	if letters == 0 {
		return lettersPerPage
	}
	return max(1, lettersPerPage*len(text)/letters)
}

// PageCount is how many pages the book is read in. A book with no text is one
// page, which is the page a person is looking at.
func (b *Book) PageCount() int {
	size := b.PageBytes()
	return max(1, (len(b.Text)+size-1)/size)
}

// PageOf is the page an offset falls on.
func (b *Book) PageOf(offset int) int {
	return min(max(1, offset/b.PageBytes()+1), b.PageCount())
}

// PageStart is where a page begins in the text. A page begins at a letter, and
// a letter is several bytes in most of the scripts a book is written in.
func (b *Book) PageStart(page int) int {
	at := min(max(0, (page-1)*b.PageBytes()), len(b.Text))
	for at < len(b.Text) && !utf8.RuneStart(b.Text[at]) {
		at++
	}
	return at
}

// PageBytes is how many bytes of this book's text stand on one page. Whoever
// counts a page of this book counts it by this number.
func (b *Book) PageBytes() int {
	if b.pageBytes <= 0 {
		return lettersPerPage
	}
	return b.pageBytes
}
