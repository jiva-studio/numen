Feature: A revoked device has no access

  A device that was revoked cannot read or write the notes of the identity.

  Background:
    Given the empty vault "Test"
    And a sync server runs
    And the person creates an identity
    And the application "editor" starts
    And the second computer is linked to the identity of the person

  Rule: A revoked device cannot read or write

    Scenario: A revoked device does not get a new note
      Given the second computer connects its vault "Copy" to the synced vault "Test"
      When the person revokes the second computer
      And the person adds the file "list.md" to the vault "Test"
      Then the second computer does not have the file "list.md" in its vault "Copy"

    Scenario: A revoked device does not write to the identity
      Given the second computer connects its vault "Copy" to the synced vault "Test"
      When the person revokes the second computer
      And the second computer adds the file "bread.md" to its vault "Copy"
      Then the person does not have the file "bread.md" in the vault "Test"

    Scenario: A device linked after the revoke reads the notes of both periods
      Given the person writes the note "Eggs" with the text "eggs" in the vault "Test"
      When the person revokes the second computer
      And the person writes the note "Milk" with the text "milk" in the vault "Test"
      And the second computer is linked to the identity of the person
      And the second computer connects its vault "Copy" to the synced vault "Test"
      Then the second computer has the note "Eggs" with the text "eggs" in its vault "Copy"
      And the second computer has the note "Milk" with the text "milk" in its vault "Copy"
