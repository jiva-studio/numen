Feature: Ask agent
  A person asks the agent about the vault in the Agent tab.

  Background:
    Given the empty vault "Test"
    And the vault has the following:
      | path             | text                                 |
      | notes/Entropy.md | Entropy is a measure of uncertainty. |
    And the application "editor" starts

  Rule: The agent answers a question

    Background:
      Given the person runs the command "Open agent chat"

    Scenario: A new chat is empty
      Then the Agent tab titled "Agent" has "No messages yet"
      And the composer of the agent cannot send

    Scenario: A question is answered
      When the person sends the message "Summarize thermodynamics principles" to the agent
      Then the Agent tab titled "Agent" has "Summarize thermodynamics principles"
      And the Agent tab titled "Agent" has "The agent answered."
      And the Agent tab titled "Agent" does not have "No messages yet"
      And the composer of the agent is empty

  Rule: The agent works with notes

    Background:
      Given the person runs the command "Open agent chat"

    Scenario: A note is written
      When the person sends the message "Write the tide table note" to the agent
      Then the Agent tab titled "Agent" has "The tide table is written."
      And the vault contains the following:
        | path                | text                                                           |
        | notes/Tide table.md | # Tide table\n\nHigh water at the quay at 06:40 and at 19:05.\n |

    Scenario: A link opens its note
      Given the person sends the message "Link to Entropy" to the agent
      When the person clicks the link "Entropy" in the Agent tab titled "Agent"
      Then the note tab at "notes/Entropy.md" is active
      And the Agent tab titled "Agent" is open
