Feature: Link a device to the account

  A device that has the account links a new device with a QR code. The 12 recovery words are not typed.

  As a person who uses Numen on more than one device
  I want to link a new device with a QR code
  So that the new device has my account without the 12 words

  Background:
    Given the empty vault "Test"
    And a sync server runs
    And the person creates an identity
    And the application "editor" starts

  Rule: A new device scans the QR code of a device that has the account

    Scenario: A new device scans the QR code of this device
      When the person shows the QR code to link a new device
      And the second computer scans the QR code
      And the person approves the new device
      Then the second computer has the account of the person
      And the account has 2 devices

    Scenario: The new device shows the account before it joins
      When the person shows the QR code to link a new device
      And the second computer scans the QR code
      Then the second computer asks to join the account of the person

    Scenario: The person declines the new device
      When the person shows the QR code to link a new device
      And the second computer scans the QR code
      And the person declines the new device
      Then the second computer has no identity
      And the account has 1 device

    Scenario: A QR code links one device only
      Given the person shows the QR code to link a new device
      And the second computer scans the QR code
      And the person approves the new device
      And the second computer deletes the identity from this device
      When the second computer scans the QR code
      Then the second computer has no identity

  Rule: A device that has the account scans the QR code of a new device

    Scenario: This device scans the QR code of a new device
      When the second computer shows the QR code to join an account
      And the person scans the QR code
      And the person approves the new device
      And the second computer enters the confirmation code of the person
      Then the second computer has the account of the person
      And the account has 2 devices

    Scenario: The device that has the account shows the new device
      When the second computer shows the QR code to join an account
      And the person scans the QR code
      Then the person sees the name and the fingerprint of the second computer

    Scenario: A wrong confirmation code refuses the new device
      When the second computer shows the QR code to join an account
      And the person scans the QR code
      And the person approves the new device
      And the second computer enters the confirmation code "000000"
      Then the second computer has no identity
      And the account has 1 device

    Scenario: Three wrong confirmation codes close the link
      Given the second computer shows the QR code to join an account
      And the person scans the QR code
      And the person approves the new device
      When the second computer enters a wrong confirmation code 3 times
      And the second computer enters the confirmation code of the person
      Then the second computer has no identity

  Rule: A device that has an identity does not join another account

    Scenario: A device with an identity does not join another account
      Given the second computer creates an identity
      Then the second computer cannot join an account
