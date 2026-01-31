# slackmd

Go library and CLI tool for converting Markdown to Slack mrkdwn format.

## Library

```go
import "github.com/snormore/slackmd"

// Convert markdown to Slack mrkdwn
output := slackmd.Convert("**bold** and _italic_")
// output: "*bold* and _italic_"

// Convert and post to Slack in one call
err := slackmd.Post(webhookURL, "**bold** and _italic_")

// Post pre-converted mrkdwn
err := slackmd.PostMrkdwn(webhookURL, output)

// Split long text into chunks (for custom posting logic)
chunks := slackmd.Chunk(text, 3900)
```

Long messages are automatically chunked at paragraph boundaries, preserving code blocks intact.

## CLI

### Install

```
go install github.com/snormore/slackmd/cmd/slackmd@latest
```

### Usage

Convert markdown and print to stdout:

```
slackmd input.md
echo "**bold** and _italic_" | slackmd
```

Post directly to Slack via incoming webhook:

```
export SLACK_WEBHOOK_URL="https://hooks.slack.com/services/T.../B.../..."
slackmd -post input.md
echo "# hello" | slackmd -post
```

Or pass the webhook URL directly:

```
slackmd -post -webhook "https://hooks.slack.com/services/T.../B.../..." input.md
```

Long messages are automatically split into multiple posts to stay within Slack's message size limits. Code blocks are kept intact when possible.

### Setting up a Slack webhook

1. Go to https://api.slack.com/apps and click **Create New App** > **From an app manifest**
2. Select your workspace, then paste the contents of [`slack-app-manifest.yaml`](slack-app-manifest.yaml)
3. Click **Create**
4. Go to **Incoming Webhooks** and click **Add New Webhook to Workspace**
5. Pick a channel and authorize
6. Copy the webhook URL and set it as an environment variable:
   ```
   export SLACK_WEBHOOK_URL="https://hooks.slack.com/services/T.../B.../..."
   ```

### Testing it end to end

```
# Convert only (no webhook needed)
echo "**bold** and _italic_" | slackmd

# Convert a sample file
slackmd examples/release-notes.md

# Post to Slack (requires webhook)
echo "hello from slackmd" | slackmd -post

# Post a full example
slackmd -post examples/mixed.md
```
