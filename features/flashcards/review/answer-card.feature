Feature: Answer card
  A person reviews the cards of a deck and answers them.

  Background:
    Given the empty vault "Test"
    And the vault has the following card models:
      | card model | fields       | template  | front                                       | back       |
      | Animal     | Name, Height | Recognize | {{Name}}                                    | {{Height}} |
      | Animal     |              | Name it   | Which animal is {{Height}} at the shoulder? | {{Name}}   |
    And the vault has the following cards:
      | deck    | card model | Name    | Height    |
      | Animals | Animal     | Guanaco | about 1 m |
    And the application "flashcards" starts
    And the person chooses the vault "Test"

  Rule: A vault is reviewed

    Scenario Outline: A card is answered with <answer>
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

    Scenario: The review ends when no card is left
      Given the person starts the review of the vault "Test"
      And the person shows the answer of the card
      And the person answers the card with "Easy"
      And the person shows the answer of the card
      When the person answers the card with "Easy"
      Then the review session has "Nothing left today."

  Rule: A deck is reviewed alone

    Scenario: The review counts the cards of the deck
      When the person starts the review of the deck "Animals"
      Then the review session has "2 left"

    Scenario: A card of the deck is answered
      When the person starts the review of the deck "Animals"
      And the person shows the answer of the card
      And the person answers the card with "Easy"
      Then the review session has "Animals"
      And the card has the answer "Easy"
