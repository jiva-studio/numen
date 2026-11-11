Feature: Window commands
  A person runs the commands of the window from the palette and with keys.

  Background:
    Given the empty vault "Test"
    And the vault has the following:
      | path             | text                  |
      | notes/Entropy.md | Entropy is a measure. |
    And the application "editor" starts

  Rule: A command runs from the palette

    Scenario: The file tree is opened
      Given the person closes the Files tab titled "Files"
      When the person runs the command "Open file tree"
      Then the Files tab titled "Files" is open

    Scenario: The tab in front is closed
      Given the person opens the note titled "Entropy"
      When the person runs the command "Close tab"
      Then the note tab titled "Entropy" is not open

  Rule: A command runs with its keys

    Scenario: The file tree is opened with Ctrl+Shift+E
      Given the person closes the Files tab titled "Files"
      When the person presses "Ctrl+Shift+E"
      Then the Files tab titled "Files" is open

    Scenario: An agent is opened with Ctrl+Shift+A
      Given the person closes the Agent tab titled "Agent"
      When the person presses "Ctrl+Shift+A"
      Then the Agent tab titled "Agent" is open

    Scenario: The plex is opened with Ctrl+Shift+X
      Given the person opens the note titled "Entropy"
      When the person presses "Ctrl+Shift+X"
      Then the plex has the focus note titled "Entropy"

    Scenario: The tab in front is closed with Ctrl+Shift+W
      Given the vault has the following:
        | path             | sample    |
        | assets/shore.pdf | shore.pdf |
      And the person opens the file at "assets/shore.pdf"
      When the person presses "Ctrl+Shift+W"
      Then the PDF tab at "assets/shore.pdf" is not open
