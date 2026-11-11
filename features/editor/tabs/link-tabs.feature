Feature: Link tabs
  A note opened in a tab opens in the tab that is linked to it.

  Scenario: A note is opened in the tab it is linked to
    Given the empty vault "Test"
    And the vault has the following:
      | path                 | text                                        |
      | notes/Weir notes.md  | Water over the weir.                        |
      | notes/Flume notes.md | The flume carries the water onto the wheel. |
    And the application "editor" starts
    And the person opens the note titled "Weir notes"
    And the person runs the command "Link with tab" on the tab titled "Files" with:
      | Search | Weir notes |
    When the person opens the file at "notes/Flume notes.md"
    Then the tab at "notes/Flume notes.md" is active
    And the tab titled "Weir notes" is not open
    And the note tab titled "Flume notes" has "The flume carries the water onto the wheel."
