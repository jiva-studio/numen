Feature: Restore an identity
  A person deletes the identity from this device and gets it back from its 12 recovery words.

  As a person who uses Numen on more than one device
  I want an identity that 12 words restore
  So that my identity is the same after I lose this device

  Background:
    Given the empty vault "Test"
    And the application "editor" starts

  Rule: The words restore the same identity

    Scenario: A deleted identity leaves this device
      Given the person creates an identity
      When the person deletes the identity from this device
      Then this device has no identity

    Scenario: The words give back the same identity
      Given the person creates an identity
      And the person deletes the identity from this device
      When the person restores the identity from its recovery words
      Then the identity fingerprint is the same as before

    Scenario: Words with a mistake are refused
      When the person restores the identity from 12 wrong words
      Then the person is told "words are not valid"
      And this device has no identity

    Scenario: A 13th word is not accepted
      When the person restores the identity from 13 wrong words
      Then the person is told "13 words"
      And this device has no identity
