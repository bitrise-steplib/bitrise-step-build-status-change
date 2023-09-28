# Build Status Change

> Exports environment variables to be able to detect if the currently running build's status has changed to a previous one.

## Inputs

- access_token: __(required)__ __(sensitive)__
    > Your access token for the account that has access to the Bitrise app.

## Outputs

### Exported Environment variables

- BUILD_STATUS_CHANGED: Build Status Changed
    > True if the actual build status is different from the previous one.
- PREVIOUS_BUILD_STATUS: Previous Build Status
    > Status text of the previous build.
- PREVIOUS_BUILD_SLUG: Previous Build Slug
    > The BITRISE_BUILD_SLUG of the previous build.

## Contribute

1. Fork this repository
1. Make changes
1. Submit a PR

## How to run this step from source

1. Clone this repository
1. `cd` to the cloned repository's root
1. Create a bitrise.yml (if not yet created)
1. Prepare a workflow that contains a step with the id: `path::./`
    > For example:
    > ```yaml
    > format_version: "6"
    > default_step_lib_source: https://github.com/bitrise-io/bitrise-steplib.git
    > 
    > workflows:
    >   my-workflow:
    >     steps:
    >     - path::./:
    >         inputs: 
    >         - my_input: "my input value"
    > ```
1. Run the workflow: `bitrise run my-workflow`

## About
This is an official Step managed by Bitrise.io and is available in the [Workflow Editor](https://www.bitrise.io/features/workflow-editor) and in our [Bitrise CLI](https://github.com/bitrise-io/bitrise) tool. If you seen something in this readme that never before please visit some of our knowledge base to read more about that:
  - devcenter.bitrise.io
  - discuss.bitrise.io
  - blog.bitrise.io
