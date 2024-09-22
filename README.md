# Slack Export

Export messages from a Slack channel to a JSON file.

This application was developed with Go 1.22.4.

## 1. Create Slack App

Go to https://api.slack.com/apps and create a new Slack app with app manifest:

```json
{
  "display_information": {
    "name": "Exporter"
  },
  "oauth_config": {
    "redirect_urls": ["https://oauth-redirect.pages.dev"],
    "scopes": {
      "user": [
        "users:read",
        "files:read",
        "emoji:read",
        "channels:read",
        "channels:history",
        "groups:read",
        "groups:history",
        "im:read",
        "im:history",
        "mpim:read",
        "mpim:history"
      ]
    }
  },
  "settings": {
    "org_deploy_enabled": false,
    "socket_mode_enabled": false,
    "token_rotation_enabled": false
  }
}
```

Install the app in the Slack Workspace.

## 2. Run the app

Find channel, group or DM ID by copying its link and extracting the last part of the URL. For example, the ID for `https://myworkspace.slack.com/archives/D0000000000` is `D0000000000`.

Download (or build and run) the main binary.

```shell
./slack-exporter
```

App will create a JSON file with the messages named like `D0000000000.json` with structure like:

```json
{
    "messages": [
        {
            "user": "U0000000000",
            "ts": "0000000000.000000",
            "blocks": [...],
            "reactions": [...],
            "replies": [
                {
                    ...
                }
            ]
        },
        ...
    ],
    "users": [
        {
            "id": "U0000000000",
            "name": "Name",
            ...
        }
    ],
    "channel": {
        "name": "...",
        ...
    },
    "files": {
        "F0000000000": "image.png",
        ...
    }
}
```

## 3. (Optionally) Convert JSON to HTML

To convert JSON to HTML, you can use the `json2html` tool from the `cmd` directory.

```shell
go run cmd/json2html/*.go --input D0000000000.json --output D0000000000.html
```

By default, only the standard Slack are supported. To add custom emoji, first download them by running the `emoji` tool from the `cmd` directory:

```shell
go run cmd/emoji/main.go --output emoji
```

It will create `emoji` directory with all the emoji images and `emoji.json` file with the mapping from emoji name to the file name.

Then re-run the `json2html` tool with the `--emoji` flag:

```shell
go run cmd/json2html/*.go --input D0000000000.json --output D0000000000.html --emoji emoji
```
