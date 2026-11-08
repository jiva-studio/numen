Feature: A note that changes on disk
  As a person who writes in Numen next to other programs
  I want my writing and their changes never to overwrite each other silently
  So that no text is lost

  Background:
    Given the person has the note "Lock gate" open

  Scenario: A note without unsaved writing follows the change at once
    When another program changes the note to "The lock gate opens at high water."
    Then the note has the text "The lock gate opens at high water."
    And the note reports no conflict

  Scenario: Saving pauses when the file changes under unsaved writing
    Given the person has written "Checked at dawn." that is not saved yet
    When another program changes the note to "The lock gate is closed."
    Then the note reports that its file changed on disk
    And the file holds "The lock gate is closed."
    And the note has the text "Checked at dawn."

  Scenario: Keeping their version saves the written text over the file
    Given the person has written "Checked at dawn." that is not saved yet
    And another program changed the note to "The lock gate is closed."
    When the person keeps their version
    Then the file holds the written text in place of the disk text
    And the note reports no conflict

  Scenario: Taking the disk version drops the written text
    Given the person has written "Checked at dawn." that is not saved yet
    And another program changed the note to "The lock gate is closed."
    When the person takes the disk version
    Then the note holds "The lock gate is closed." in place of the written text
    And the note reports no conflict

  Scenario: A file changed twice gives its latest version
    Given the person has written "Checked at dawn." that is not saved yet
    And another program changed the note to "The lock gate is closed."
    And another program changed the note to "The lock gate is open."
    When the person takes the disk version
    Then the note holds "The lock gate is open." in place of the written text

  Scenario: A note removed from disk keeps its text
    When another program removes the note
    Then the note reports that it was removed from the vault
    And the note keeps the text it had

  Scenario: A removed note is written again at its own path
    Given another program removed the note
    When the person recreates the note
    Then the vault holds the note again with its text
    And the note reports no conflict
