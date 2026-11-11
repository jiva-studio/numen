Feature: Open file
  A person opens a file and gets the tab that is made for its type.

  Background:
    Given the empty vault "Test"
    And the vault has the following card models:
      | card model | fields     | template | front    | back     |
      | Animal     | Name, Fact | Recall   | {{Name}} | {{Fact}} |
    And the vault has the following cards:
      | deck    | card model | Name  | Fact           |
      | Animals | Animal     | Heron | A heron wades. |
    And the vault has the following:
      | path               | text                  | sample      |
      | notes/Entropy.md   | Entropy is a measure. |             |
      | assets/shore.pdf   |                       | shore.pdf   |
      | assets/ledger.epub |                       | ledger.epub |
      | assets/talk.mp3    |                       | talk.mp3    |
      | assets/data.xyz    |                       |             |
    And the application "editor" starts

  Rule: A file opens in the tab of its type

    Scenario: A deck opens in a Deck tab
      When the person opens the file at "cards/Animals.md"
      Then the Deck tab at "cards/Animals.md" is open

    Scenario: A card model opens in a Card Model tab
      When the person opens the file at "cards/Animal.md"
      Then the Card Model tab at "cards/Animal.md" is open

    Scenario: A note opens in a note tab
      When the person opens the file at "notes/Entropy.md"
      Then the note tab at "notes/Entropy.md" is open

    Scenario: A PDF opens in a PDF tab
      When the person opens the file at "assets/shore.pdf"
      Then the PDF tab at "assets/shore.pdf" is open

    Scenario: A book opens in a book tab
      When the person opens the file at "assets/ledger.epub"
      Then the book tab at "assets/ledger.epub" is open

    Scenario: A recording opens in a recording tab
      When the person opens the file at "assets/talk.mp3"
      Then the recording tab at "assets/talk.mp3" is open

  Rule: A file does not open twice or without a tab

    Scenario: An open file keeps one tab
      Given the person opens the file at "notes/Entropy.md"
      When the person opens the file at "notes/Entropy.md"
      Then the tab at "notes/Entropy.md" is open once

    Scenario: A file without a tab type is not opened
      Given the person opens the file at "notes/Entropy.md"
      When the person opens the file at "assets/data.xyz"
      Then the tab at "notes/Entropy.md" is active
      And the tab at "assets/data.xyz" is not open
