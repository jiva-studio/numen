# Index stats

Shows in the title bar how many vectors the search index holds, so you can watch it grow when you add a book or a PDF or edit a note, and shrink when you delete something.

<!-- more -->

## What You Can Do

- **Watch the index**: The title bar shows `⬡ 12 480`, the number of chunks of text that carry a vector. While chunks still wait for their vector, the mark pulses softly in green; when none wait, it stands still. The item never shows the total of chunks: the details are in the popover.
- **See the details**: Click the item for a list of the counts and settings behind the number. Press Escape or click elsewhere to close it.

The popover lists:

- **Vectors**, **Chunks** and **Waiting for a vector**: the chunks that carry a vector, all the chunks that can, and the queue between the two.
- **Notes and files**: how many sources the index has read, with the chunks and files of each kind: notes, PDF, EPUB and media transcripts. A kind that holds nothing is not listed.
- **Model**, **Recipe** and **Vector size**: the embedding model in use and the dimension count of its vectors.
- **Performance profile**: the profile you chose in the settings and the threads it grants.
- **Embedding speed**: chunks embedded each second over the last 10 seconds, measured while the popover is open and shown only while chunks wait.
- **Last change**: when the counts last changed while the window was open.
- **Index file size** and **Vector data size**: the disk space of the index and of the tables that hold vectors, refreshed every 30 seconds.

The plugin is enabled by default and has no settings. It only reads the index and changes nothing.

## How It Works

The plugin's backend asks the host for `GetIndexStats` of `HostSearchService` (the permission `index:read`), which reads the counts with indexed queries and keeps the answer for one second. The window asks the backend once when it opens and again each time a background task starts or ends. While the popover is open it asks every 2 seconds while the window is visible and has the focus, and once when it gets the focus back. The backend derives the kind of each source from its file extension and measures the speed and the time of the last change from the answers it sees. Its service `md.numen.tools.index.stats` has the method `GetStats`.
