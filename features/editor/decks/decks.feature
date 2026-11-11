Feature: Decks
  A person makes a deck of cards and edits the values of its cards.

  Background:
    Given the empty vault "Test"

  Rule: A deck or a card model is made

    Background:
      Given the vault has the following:
        | path      | text |
        | Glaciers/ |      |
      And the application "editor" starts

    Scenario: A deck is made at the top
      When the person runs the command "Create deck" with:
        | Name | Rivers |
      Then the Deck tab titled "Rivers" is active
      And the vault contains the file at "Rivers.md"
      And the Files tab titled "Files" has "Rivers.md"

    Scenario: A deck is made in a folder
      When the person runs the command "Create deck" on the folder at "Glaciers" with:
        | Name | Moraines |
      Then the vault contains the file at "Glaciers/Moraines.md"
      And the Deck tab at "Glaciers/Moraines.md" is active

    Scenario: A card model is made in a folder
      When the person runs the command "Create card model" on the folder at "Glaciers" with:
        | Name | Ice cores |
      Then the vault contains the file at "Glaciers/Ice cores.md"
      And the Card Model tab at "Glaciers/Ice cores.md" is active

  Rule: A deck is a file of the vault

    Background:
      Given the vault has the following card models:
        | card model | fields       | template | front    | back       |
        | Animal     | Name, Height | Size     | {{Name}} | {{Height}} |
      And the vault has the following cards:
        | deck        | card model | Name  | Height      |
        | Shore birds | Animal     | Heron | about 1 m   |
        | Shore birds | Animal     | Otter | about 30 cm |
      And the application "editor" starts

    Scenario: A value of a card is saved
      Given the person opens the file at "cards/Shore birds.md"
      When the person writes "1.2 m" in the field "Height" of the card "Card 1" of the Deck tab titled "Shore birds"
      Then the vault contains the following:
        | path                 | text  |
        | cards/Shore birds.md | 1.2 m |

    Scenario: A saved value is shown after the application starts again
      Given the person opens the file at "cards/Shore birds.md"
      And the person writes "1.2 m" in the field "Height" of the card "Card 1" of the Deck tab titled "Shore birds"
      When the application "editor" restarts
      Then the Deck tab titled "Shore birds" has "1.2 m" in the field "Height" of the card "Card 1"
      And the Deck tab titled "Shore birds" has "about 30 cm" in the field "Height" of the card "Card 2"

    Scenario: A removed deck closes its tab
      Given the person opens the file at "cards/Shore birds.md"
      And the Deck tab titled "Shore birds" is open
      When the person runs the command "Move note to trash" on the file at "cards/Shore birds.md"
      Then the Deck tab titled "Shore birds" is not open
      And the vault does not contain the file at "cards/Shore birds.md"

    Scenario: A removed card model closes its tab
      Given the person opens the file at "cards/Animal.md"
      And the Card Model tab titled "Animal" is open
      When the person runs the command "Move note to trash" on the file at "cards/Animal.md"
      Then the Card Model tab titled "Animal" is not open
      And the vault does not contain the file at "cards/Animal.md"
