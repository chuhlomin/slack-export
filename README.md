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
        "redirect_urls": [
            "https://exporter.local"
        ],
        "scopes": {
            "user": [
                "channels:history",
                "groups:history",
                "im:history",
                "mpim:history",
                "users:read",
                "channels:read",
                "files:read"
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

## 2. Prepare the environment for OAuth flow

To perform operations on behalf of the user, the app needs to be authorized by the user (token starts with `xoxp-`). This is done by the OAuth flow. Once token is obtained, it can be passed to the app as `API_TOKEN` environment variable to avoid OAuth flow.

App will run a local HTTP server to receive the OAuth callback. To make it work, you need to add `exporter.local` to your `/etc/hosts` file:

```
127.0.0.1       exporter.local
::1     exporter.local
```

Because Slack requires HTTPS for OAuth, you need to run the app with a self-signed certificate. One way to do this to run Caddy from the root of this repo:

```sh
caddy run
```

It will forward all requests to the app running on `localhost:8079` (controlled by `ADDRESS` and `PORT` environment variable) and serve the app on `https://exporter.local`.

## 3. Run the app

Copy the file `.env-template` to `.env` and add values to the placeholders for `APP_CLIENT_ID` and `APP_CLIENT_SECRET` with the values from "Basic info" in your Slack app settings.

Find channel, group or DM ID by copying its link and extracting the last part of the URL. For example, the ID for `https://myworkspace.slack.com/archives/D0000000000` is `D0000000000`.

Build and run the main executable from the root of this repo with: 

```shell
go run . --channel="D0000000000"
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

## 4. (Optionally) Convert JSON to HTML

To convert JSON to HTML, you can use the `json2html` tool from the `cmd` directory.

```shell
go run cmd/json2html/main.go --input D0000000000.json --output D0000000000.html
```

By default, only the standard Slack are supported. To add custom emoji, first download them by running the `emoji` tool from the `cmd` directory:

```shell
go run cmd/emoji/main.go --output emoji
```

It will create `emoji` directory with all the emoji images and `emoji.json` file with the mapping from emoji name to the file name.

Then re-run the `json2html` tool with the `--emoji` flag:

```shell
go run cmd/json2html/main.go --input D0000000000.json --output D0000000000.html --emoji emoji
```
