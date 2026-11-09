Feature: Answer card
  A person reviews the cards of a deck and answers them.

  Background:
    Given the empty vault "Test"
    And the vault has the following card models:
      | card model | fields       | template  | front                                        | back |
      | Animal     | Name, Height | Recognise | {{Name}}                                     | {{Height}} |
      | Animal     |              | Name it   | Which animal is {{Height}} at the shoulder?  | {{Name}} |
    And the vault has the following cards:
      | deck      | card model | Name     | Height |
      | crlf-deck | Animal     | Guanaco  | about 43" (shoulder) |
    And the application "flashcards" starts
    And the person chooses the vault "Test"

  Rule: An answer is saved

    Scenario Outline: A card is answered with <answer>
      Given the person starts the review of the vault "Test"
      And the person shows the answer of the card
      When the person answers the card with "<answer>"
      Then the card has the answer "<answer>"

      Examples:
        | answer |
        | Again |
        | Hard |
        | Good |
        | Easy |

  Rule: A deck is reviewed alone

    Scenario: The review counts the cards of the deck
      When the person starts the review of the deck "crlf-deck"
      Then the review session has "2 left"

    Scenario: A card of the deck is answered
      When the person starts the review of the deck "crlf-deck"
      And the person shows the answer of the card
      And the person answers the card with "Easy"
      Then the review session has "crlf-deck"
      And the card has the answer "Easy"
