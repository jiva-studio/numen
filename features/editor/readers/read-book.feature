Feature: Read a book
  A person reads an EPUB book in a tab and goes to a chapter from the contents.

  Scenario: A book without contents lists its files
    Given the empty vault "Test"
    And the vault has the following:
      | path               | sample      |
      | assets/sample.epub | sample.epub |
    And the application "editor" starts
    And the person opens the file at "assets/sample.epub"
    When the person clicks the button "Show contents" in the book tab at "assets/sample.epub"
    Then the contents of the book tab at "assets/sample.epub" are:
      | chapter |
      | doc1    |
    When the person chooses the chapter "doc1" in the contents of the book tab at "assets/sample.epub"
    Then the contents of the book tab at "assets/sample.epub" are hidden
