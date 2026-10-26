# How a document's pages are read

What a model reads off the pages of a scanned document, where it is kept, and what is done to it afterwards. The run is a recognition and what it produces is recognised text, written into the vault's own folder, under `.numen/ocr/`, and the source is cut from it.

## The files of one recognition

A recognition is named by the hash of the bytes it was made from. A document renamed or moved keeps its recognition, and two copies of one document in a vault share one. Five names are composed from the producer and that hash:

| Name | What it holds |
| --- | --- |
| `<hash>.txt` | the prose, with the pages marked in it |
| `<hash>.partial` | the prose of a run still going, and how far it has got |
| `<hash>.boxes` | one record a printed line: its page, its run of bytes in the prose, and its rectangle as a fraction of that page |
| `<hash>.parts` | one record a heading: where it begins in the prose, how far it runs, and how deep it sits |
| `<hash>.json` | which models read the document, at what resolution, and under which recipe |

One run made them and none of them means anything without the others.

Three compose into the text a chunk is a place in: the artifact — or the partial, where no artifact is finished — with the parts and the boxes. The other two are read by a person asking what produced a text, and by a sweep looking for everything a recogniser now known to be bad wrote.

```mermaid
graph TD
    TXT[".txt<br/>the prose, pages marked"]
    PARTIAL[".partial<br/>the prose of a run still going"]
    PARTS[".parts<br/>one record a heading"]
    BOXES[".boxes<br/>one rectangle a printed line"]
    BESIDE[".json<br/>which models read this"]
    TEXT["the text a chunk is a place in"]

    TXT --> TEXT
    PARTIAL -->|"where no .txt is finished"| TEXT
    PARTS --> TEXT
    BOXES --> TEXT
    BESIDE -.-> TXT
```

## What the artifact holds

Plain text. Each page opens with a pair of form feeds and the newline closing them, and the page's prose follows, one blank line between the regions the layout model found. A recogniser has no character for a form feed, so nothing in the prose is escaped.

The marks are taken out when the artifact is read. An offset in what comes back is an offset in the prose, and a chunk cut from it holds what the page says. Text standing before the first mark belongs to no page and is kept.

A page with nothing on it is written, and its mark is what makes the page after it findable. A document with nothing on it writes nothing at all, and the source goes back on its own text layer.

A partial is the source's text while it is being written. After every batch the source is cut again from what has been read, so a book answers about the pages already read while the rest is still being read; while a run is on, the part of the book beyond the count is out of search. Reading a text back tries the finished artifact and then the partial.

## How a page is named

A page is called by its position in the file, and the pane a person reads it in opens at that number. The marks are written one per page in order, so the number is the position and is stored nowhere.

What the paper printed is not kept. The page-label dictionary a born-digital PDF may carry is not read, and the region the layout model calls a page's number is not among the regions that carry what the document says.

An EPUB is the exception. A book made for a screen has no pages of its own, so the page breaks it names from the printed edition it was set from are the only page names it has, and a page of one is called by its label.

## Parts

A recognition names its parts. The layout model names the headings, and one record a heading goes into `.parts`: where the heading begins in the prose, how far it runs, and how deep it sits, counted from zero with a document title above the section titles within it.

Parts are built from that sidecar, so a recognised document names them as a book with an outline does. A part is named by the run of prose its heading occupies, read as the scan was read. Structure is kept, and never inferred from the recognised words.

A parts sidecar whose records run backwards, or reach past the end of the prose, was written for other bytes. None of it is used and the recognition is located by its pages alone.

The boundary the line detector draws falls inside the letters, so a found line is widened before it is read, by `indexing.recognition.detect.expand` pixels of the image the detector reads — eighteen where the file names nothing. One number serves a heading and a paragraph, because a part is read scaled so that its longest side is a fixed length. See [Settings](settings.md).

## Proofreading

A page was printed before it was scanned, so there is something to put the recognition right against: what the paper says. Proofreading is the proofreading plugin's: a person asks for it from the file's menu, and the corrected text is a version made from the recognition, which stays as it was read. While the OCR plugin is off, that version is dead with the recognition it was made from. How a proofreader is asked, what its reply must look like and what refuses one is [Proofreading](proofreading.md).

## A run

Recognition resumes where it stopped: pages are appended to the partial in batches — sixteen at a time — and the count is appended after them, on a line beginning with a byte no recogniser can write, which the artifact's own reader passes over. The count is what makes the batch before it count, so a batch no count claims is one the next run reads again.

A run claims the recognition for as long as it takes, and a second run against the same recognition reports that somebody else has it.

A chunk whose text did not change keeps the vector already made for it.

## Settings

Every key named above lives in [Settings](settings.md): `indexing.recognition.detect.expand`. The settings of proofreading are in [Proofreading](proofreading.md).
