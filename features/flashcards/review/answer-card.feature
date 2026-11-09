Feature: Answer card
  A person reviews the cards of a deck and answers them.

  Background:
    Given the application "flashcards" starts
    And the person chooses the vault "Test"

  Rule: The answer of the person to a card is saved

    Scenario Outline: A card is answered
      Given the person starts the review of the vault "Test"
      And the person shows the answer of the card
      When the person answers the card with "<answer>"
      Then the card has the answer "<answer>"

      Examples:
        | answer |
        | Again  |
        | Hard   |
        | Good   |
        | Easy   |

  Rule: A deck is reviewed without the other decks

    Scenario: The review of a deck has the number of its cards left
      When the person starts the review of the deck "crlf-deck"
      Then the review session has "1 left"

    Scenario: A card of a deck is answered in the review of the deck
      When the person starts the review of the deck "crlf-deck"
      And the person shows the answer of the card
      And the person answers the card with "Easy"
      Then the review session has "crlf-deck"
      And the card has the answer "Easy"
