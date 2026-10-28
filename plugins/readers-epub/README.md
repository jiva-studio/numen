# EPUB Book Reader

Opens `.epub` books with reflowable chapter rendering, table of contents navigation, and passage linking.

<!-- more -->

## Features

- **Reflowable reading view**: Renders EPUB chapters with customizable reading scale and theme colors.
- **Chapter & outline navigation**: Parses EPUB spine and NCX/Navigation documents via `numen.epub.v1.BookService`.
- **Passage citations**: Supports selecting and linking exact byte stretches (`numen:<path>?start=<offset>&length=<len>`) from notes and agent answers.
