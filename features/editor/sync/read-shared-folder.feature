Feature: Read a shared folder

  A shared folder is read only. The reader needs an identity and no subscription.

  As a person who gets a shared folder
  I want to read it and not change it by mistake
  So that the notes stay the same as the owner wrote them

  Background:
    Given the empty vault "Test"
    And a sync server that sells subscriptions runs
    And the person creates an identity
    And the application "editor" starts
    And the person opens the page "Identity and licenses"
    And the person buys the subscription
    And the person connects the vault "Test" as a new synced vault
    And the person writes the note "Home/Food/Shopping list" with the text "milk" in the vault "Test"
    And the reader creates an identity
    And the second reader creates an identity
    And the person shares the folder "Home/Food" of the vault "Test" with the reader

  Rule: The reader needs no subscription

    Scenario: A reader without a subscription reads the folder
      Then the reader has the note "Shopping list" with the text "milk" in the shared vault "Food"
      And the subscription shows "No subscription" on the reader

  Rule: The reader only reads

    Scenario: A note cannot be made in a shared vault
      When the reader creates the note "Plan" in the shared vault "Food"
      Then the reader is told "This vault is read only"

    Scenario: A shared vault cannot be shared again
      When the reader shares the shared vault "Food" with the second reader
      Then the reader is told "A shared vault cannot be shared again"

    Scenario: A shared vault cannot be synced as a vault of the reader
      When the reader adds the shared vault "Food" to its synced vaults
      Then the reader is told "A shared vault cannot be synced again"

  Rule: A folder is shared once

    Scenario: A folder inside a shared folder cannot be shared
      Given the person writes the note "Home/Food/Fruit/Apple" with the text "red" in the vault "Test"
      When the person shares the folder "Home/Food/Fruit" of the vault "Test" with the second reader
      Then the person is told "A folder inside a shared folder cannot be shared"

    Scenario: A folder that holds a shared folder cannot be shared
      When the person shares the folder "Home" of the vault "Test" with the second reader
      Then the person is told "A folder containing a shared folder cannot be shared"
