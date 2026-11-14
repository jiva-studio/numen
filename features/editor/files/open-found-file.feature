Feature: Open found file
  A person opens a file from the search and gets the tab that is made for its type.

  As a person who keeps files in a vault
  I want to open a file that the search finds
  So that I see it in the tab that fits its type

  Background:
    Given the empty vault "Test"
    And the vault has the following card models:
      | card model | fields     | template | front    | back     |
      | Animal     | Name, Fact | Recall   | {{Name}} | {{Fact}} |
    And the vault has the following cards:
      | deck    | card model | Name  | Fact           |
      | Animals | Animal     | Heron | A heron wades. |
    And the vault has the following:
      | path             | text                  |
      | notes/Entropy.md | Entropy is a measure. |
    And the application "editor" starts

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
