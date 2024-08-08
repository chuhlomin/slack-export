# Slack Channel Export

## Background

I wanted to extract Slack messages and their replies for further analysis in ChatGPT. This tool writes that data into a JSON file.

## Try it yourself

This application was developed with Go 1.22.4.

1. Go to https://api.slack.com/apps and create a new Slack app.
1. Add the scope `channels:history` to the app's Bot Token Scopes.
1. Install the app in the Slack Workspace.
1. Add the app to the Slack channel you want to export data from: `/invite @,YOUR_APP_NAME>`.
1. Copy the Bot User OAuth Token of the Slack app from "OAuth & Permissions" in your Slack app settings.
1. Copy the file `.env-template` to `.env` and add values to the placeholders for `SLACK_CHANNEL_ID` and `SLACK_API_TOKEN`. The Bot User OAuth Token is the value for `SLACK_API_TOKEN`.
1. Build and run the main executable from the root of this repo with: `go run src/main.go`.

## Slack APIs

The following two Slack API endpoints are being used:

- https://api.slack.com/methods/conversations.history
- https://api.slack.com/methods/conversations.replies
