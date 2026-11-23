Feature: Revoke a device

  A person removes a lost device from the account. The removed device loses its access.

  As a person who uses Numen on more than one device
  I want to revoke a device I lost
  So that nobody who has it can read or write my notes

  Background:
    Given the empty vault "Test"
    And a sync server runs
    And the person creates an identity
    And the application "editor" starts
    And the second computer is linked to the account of the person

  Rule: A revoked device leaves the account

    Scenario: A revoked device leaves the list of devices
      When the person revokes the second computer
      Then the account has 1 device

    Scenario: A device does not revoke itself
      Then the person cannot revoke this device

    Scenario: A revoked device does not sign in to the sync server
      When the person revokes the second computer
      Then the sync server has no session for the second computer

  Rule: A revoked device cannot read or write

    Scenario: A revoked device does not get a new note
      Given the second computer connects its vault "Copy" to the synced vault "Test"
      When the person revokes the second computer
      And the person writes the file "list.md" in the vault "Test"
      Then the second computer does not have the file "list.md" in its vault "Copy"

    Scenario: A revoked device does not write to the account
      Given the second computer connects its vault "Copy" to the synced vault "Test"
      When the person revokes the second computer
      And the second computer writes the file "bread.md" in its vault "Copy"
      Then the person does not have the file "bread.md" in the vault "Test"

    Scenario: A device linked after the revoke reads the notes of both periods
      Given the person writes the note "Eggs" with the text "eggs" in the vault "Test"
      When the person revokes the second computer
      And the person writes the note "Milk" with the text "milk" in the vault "Test"
      And the second computer is linked to the account of the person
      And the second computer connects its vault "Copy" to the synced vault "Test"
      Then the second computer has the note "Eggs" with the text "eggs" in its vault "Copy"
      And the second computer has the note "Milk" with the text "milk" in its vault "Copy"

    Scenario: The sync server holds no text of a note after the revoke
      When the person revokes the second computer
      And the person writes the note "Milk" with the text "milk" in the vault "Test"
      Then the sync server holds no stored text "milk"

  Rule: A revoked device says it was revoked and keeps its files

    Scenario: A revoked device says it was revoked
      When the person revokes the second computer
      Then the second computer warns about "This device was revoked"

    Scenario: A revoked device keeps its local notes
      Given the second computer writes the note "Eggs" with the text "eggs" in its vault "Copy"
      When the person revokes the second computer
      Then the second computer has the note "Eggs" with the text "eggs" in its vault "Copy"
