Feature: Plugin settings
  A person reads and changes the settings of a plugin in the Settings tab.

  Background:
    Given the empty vault "Test"
    And the application "editor" starts

  Scenario: The settings list the plugins
    When the person runs the command "Open settings"
    Then the Settings tab titled "Settings" has "Audio/Video Transcription"
    And the Settings tab titled "Settings" has "Document OCR"

  Rule: A setting is read and changed

    Scenario: A model is shown with its languages
      Then the setting "Model" of the plugin "Audio/Video Transcription" is "Parakeet TDT 0.6B v3"
      And the setting "Model" of the plugin "Audio/Video Transcription" has the option "25 European languages"

    Scenario: A model is kept when the Settings tab is closed
      Given the person sets the setting "Model" of the plugin "Audio/Video Transcription" to "Parakeet TDT 0.6B v2"
      And the person closes the Settings tab titled "Settings"
      When the person runs the command "Open settings"
      Then the setting "Model" of the plugin "Audio/Video Transcription" is "Parakeet TDT 0.6B v2"

  Rule: A setting shows the settings it needs

    Scenario: The files of a custom model are asked for
      When the person sets the setting "Model" of the plugin "Document OCR" to "Custom model"
      Then the settings of the plugin "Document OCR" have the field "Model file"
      And the settings of the plugin "Document OCR" have the field "Character dictionary"

    Scenario: The files of a custom model are not asked for
      Then the settings of the plugin "Document OCR" do not have the field "Model file"

    Scenario: The address of a custom service is asked for
      When the person sets the setting "Provider" of the plugin "Vector Embeddings" to "Custom service"
      Then the settings of the plugin "Vector Embeddings" have the field "Service URL"

    Scenario: The address of a service is kept when another provider is chosen
      Given the person sets the setting "Provider" of the plugin "Vector Embeddings" to "Custom service"
      And the person sets the setting "Service URL" of the plugin "Vector Embeddings" to "http://127.0.0.1:9/v1"
      And the person sets the setting "Provider" of the plugin "Vector Embeddings" to "OpenAI"
      When the person sets the setting "Provider" of the plugin "Vector Embeddings" to "Custom service"
      Then the setting "Service URL" of the plugin "Vector Embeddings" is "http://127.0.0.1:9/v1"
