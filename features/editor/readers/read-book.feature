Feature: Read a book
  A person reads an EPUB book in a tab and goes to a chapter from the contents.

  Background:
    Given the empty vault "Test"
    And the vault has the following:
      | path               | sample      |
      | assets/sample.epub | sample.epub |
    And the application "editor" starts
    And the person opens the file at "assets/sample.epub"

  Scenario: A book without contents lists its files
    When the person clicks the button "Show contents" in the book tab at "assets/sample.epub"
    Then the contents of the book tab at "assets/sample.epub" are:
      | chapter |
      | doc1    |

  Scenario: A chapter is chosen from the contents
    Given the person clicks the button "Show contents" in the book tab at "assets/sample.epub"
    When the person chooses the chapter "doc1" in the contents of the book tab at "assets/sample.epub"
    Then the contents of the book tab at "assets/sample.epub" are not shown
