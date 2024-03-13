# ChatGPT tui README

My first ever terminal UI! Everything is stored locally on sqlite and written in Go!

## Technologies

- Go
- [bubbletea](https://github.com/charmbracelet/bubbletea): A Go framework for
  terminal user interfaces. It's a great framework that makes it easy to create
  TUIs in Go.
- [openai](https://platform.openai.com/docs/api-reference): OpenAI's REST Api
- [sqlite](https://www.sqlite.org/): A C library that provides a lightweight
  disk-based database that doesn't require a separate server process (perfect
  for terminal apps, in my opinion).
- [lipgloss](https://github.com/charmbracelet/lipgloss): Style definitions for
  nice terminal layouts!
- [bubbles](https://github.com/charmbracelet/bubbles): Some general use
  components for Bubble Tea apps!

## Installation

Please make sure that you expose a `OPENAI_API_KEY` inside of your environment; we require it to make api calls!

Set up your [api key](https://platform.openai.com/api-keys)

```bash
export OPENAI_API_KEY="some-key" # you would want to export this in your .zshrc
brew tap tearingitup786/tearingitup786
brew install chatgpt-tui
chatgpt-tui
```

To get access to the release candidates, install command:

```bash
brew install rc-chatgpt-tui
rc-chatgpt-tui
```

## Config

We provide a `config.json` file within your directory for easy access to essential settings.
On most Macs, the path is `~/.chatgpt-tui/config.json`.
This file includes the URL used for network calls to the TUI,
specified as `chatGPTApiUrl: "https://api.openai.com/v1/chat/completions"`.
Additionally, the `systemMessage` field is available for customizing system prompt messages.

## Demo

![tui demo](./tui-demo.gif)

## Global Keybindings

- `Tab`: \*Change focus between panes. The currently focused pane will be highlighted with a pink border.
  - You can only change focus if Prompt Pane is not in `insert mode`
- `Ctrl+o`: Toggles zen mode
- `Ctrl+c`: Exit the program

## Prompt Pane

- `i`: Enters insert mode (you can now safely paste messages into the tui)
- `esc`: Exit insert mode for the prompt

## Chat Messages Pane

- `y`: Copies the last message from ChatGPT into your clipboard.
- `Y`: Copies all messages from the ChatGPT session into your clipboard.

## Settings Pane

- `m`: Opens an input dialog to change the model.
- `f`: Opens an input dialog to change the frequency of updates.
- `t`: Opens an input dialog to set the maximum number of tokens per message.

## Sessions Pane

- `Ctrl+N`: Creates a new session.
- `d`: Deletes the currently selected session from the list.
- `Enter`: Switches to the session that is currently selected.

Please refer to this guide as you navigate the TUI. Happy exploring!

### Dev notes

The SQL db is stored in you `your/home/directory/.chatgpt-tui`, as well as the debug log. To enable `debug` mode, `export DEBUG=1` before running the program.

## Contributors

<!-- ALL-CONTRIBUTORS-LIST:START - Do not remove or modify this section -->
<!-- prettier-ignore-start -->
<!-- markdownlint-disable -->
<table>
  <tbody>
    <tr>
      <td align="center" valign="top" width="14.28%"><a href="https://twitter.com/tearingitup786"><img src="https://avatars.githubusercontent.com/u/16584942?v=4?s=100" width="100px;" alt="Taranveer (Taran) Bains"/><br /><sub><b>Taranveer (Taran) Bains</b></sub></a><br /><a href="#doc-tearingItUp786" title="Documentation">📖</a> <a href="#maintenance-tearingItUp786" title="Maintenance">🚧</a> <a href="#review-tearingItUp786" title="Reviewed Pull Requests">👀</a> <a href="#code-tearingItUp786" title="Code">💻</a></td>
      <td align="center" valign="top" width="14.28%"><a href="https://www.tjmiller.me"><img src="https://avatars.githubusercontent.com/u/5108034?v=4?s=100" width="100px;" alt="TJ Miller"/><br /><sub><b>TJ Miller</b></sub></a><br /><a href="#doc-sixlive" title="Documentation">📖</a> <a href="#code-sixlive" title="Code">💻</a></td>
    </tr>
  </tbody>
</table>

<!-- markdownlint-restore -->
<!-- prettier-ignore-end -->

<!-- ALL-CONTRIBUTORS-LIST:END -->
