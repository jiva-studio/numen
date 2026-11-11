Feature: Open file
  A person opens a file and gets the tab that is made for its type.

  Background:
    Given the empty vault "Test"
    And the vault has the following card models:
      | card model | fields     | template | front    | back     |
      | Animal     | Name, Fact | Recall   | {{Name}} | {{Fact}} |
    And the vault has the following cards:
      | deck    | card model | Name   | Fact                  |
      | Animals | Animal     | Heron  | A heron wades.        |
    And the vault has the following:
      | path              | text                      |
      | notes/Entropy.md  | Entropy is a measure.     |
      | assets/shore.pdf  | shore.pdf                 |
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

  Rule: A file found by search opens in the tab of its type

    Scenario: A deck found by its name opens in a Deck tab
      When the person searches the vault for "Animals"
      And the person chooses the search result "Animals" in the group "Names"
      Then the Deck tab at "cards/Animals.md" is open

    Scenario: A card model found by its name opens in a Card Model tab
      When the person searches the vault for "Animal"
      And the person chooses the search result "Animal" in the group "Names"
      Then the Card Model tab at "cards/Animal.md" is open

    Scenario: A note found by its name opens in a note tab
      When the person searches the vault for "Entropy"
      And the person chooses the search result "Entropy" in the group "Names"
      Then the note tab at "notes/Entropy.md" is open
