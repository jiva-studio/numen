Feature: Recognition
  A person makes the text of a scan and finds it.

  Background:
    Given the application "editor" starts

  Rule: The text of a scan is found by search

    Scenario: A scan is found by the words recognized in it
      Given the vault "Test" contains the file at "assets/regenerative-braking.pdf"
      And the person opens the file at "assets/regenerative-braking.pdf"
      And the person runs the command "Create text"
      And the person enters "Recognize text"
      When the person searches the vault for "recovers kinetic energy"
      Then the search has the result "assets/regenerative-braking.pdf" in the group "Text"

    Scenario: A passage of a scan is opened with the highlight
      Given the vault "Test" contains the file at "assets/regenerative-braking.pdf"
      And the person opens the file at "assets/regenerative-braking.pdf"
      And the person runs the command "Create text"
      And the person enters "Recognize text"
      And the person searches the vault for "recovers kinetic energy"
      When the person chooses the search result "assets/regenerative-braking.pdf" in the group "Text"
      Then the tab at "assets/regenerative-braking.pdf" shows a highlight

  Rule: The text of a scan is found when the plugin "Document OCR" is disabled

    Scenario: The recognized text is found
      Given the vault "Test" contains the file at "assets/flywheel.pdf"
      And the person opens the file at "assets/flywheel.pdf"
      And the person runs the command "Create text"
      And the person enters "Recognize text"
      When the person disables the plugin "Document OCR"
      And the person searches the vault for "rotor spinning in a vacuum"
      Then the search has the result "assets/flywheel.pdf" in the group "Text"

    Scenario: The proofread text is found
      Given the vault "Test" contains the file at "assets/flywheel.pdf"
      And the person opens the file at "assets/flywheel.pdf"
      And the person runs the command "Create text"
      And the person enters "Recognize text"
      And the person runs the command "Proofread text"
      And the person enters "Recognized text (current)"
      When the person disables the plugin "Document OCR"
      And the person searches the vault for "proofread"
      Then the search has the result "assets/flywheel.pdf" in the group "Text"
