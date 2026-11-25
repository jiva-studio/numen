Feature: Create an identity
  A person keeps an identity on this device, made with 12 recovery words.

  As a person who uses Numen on more than one device
  I want an identity with 12 recovery words
  So that my identity can be the same on each of my devices

  Background:
    Given the empty vault "Test"
    And the application "editor" starts

  Rule: The settings have the page "Identity and licenses"

    Scenario: The settings list the page "Identity and licenses"
      When the person runs the command "Open settings"
      Then the settings list the page "Identity and licenses"

  Rule: A new identity shows its words once, then checks three of them

    Scenario: A new identity has 12 recovery words
      When the person starts a new identity
      Then the identity shows 12 recovery words

    Scenario: The words are checked before the identity is kept
      Given the person starts a new identity
      When the person writes the recovery words down
      Then the identity asks for 3 of the recovery words
      And this device has no identity

    Scenario: The identity is kept after three words are confirmed
      Given the person starts a new identity
      And the person writes the recovery words down
      When the person confirms 3 of the recovery words
      Then this device has an identity

    Scenario: A wrong word keeps no identity
      Given the person starts a new identity
      And the person writes the recovery words down
      When the person confirms the recovery words with the word "zoo" in place of one word
      Then this device has no identity

    Scenario: The words are not shown again
      Given the person creates an identity
      When the person opens the page "Identity and licenses"
      Then the identity does not show the recovery words
