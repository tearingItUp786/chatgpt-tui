# ChatGPT tui README

A terminal util for chatting with LLMs

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
specified as `chatGPTApiUrl: "https://api.openai.com"`.
The url can be anything that follows OpenAI API standard ( [ollama](https://ollama.com/), [lmstudio](https://lmstudio.ai/), etc)
Additional fields:
 - `systemMessage` field is available for customizing system prompt messages.
 - `defaultModel` field sets the default model 

### Themes
You can change colorscheme using the `colorScheme` field.

Available themes:
 * `Pink` **default**
 * `Blue`
 * `Groove`

## Cache invalidation

Models list is cached for 14 days upon loading. If you need to invalidate cache use `--purge-cache` flag:
```bash
./chatgpt-tui --purge-cache
```

## Demo

![tui demo](./docs/images/tui-demo.gif)

## Global Keybindings

- `Tab`: \*Change focus between panes. The currently focused pane will be highlighted with a pink border.
  - You can only change focus if Prompt Pane is not in `insert mode`
- `1`: Jump to prompt pane
- `2`: Jump to chat pane
- `3`: Jump to settings pane
- `4`: Jump to sessions pane
- `Ctrl+b`: Interrupt inference
- `Ctrl+o`: Toggles zen mode
- `Ctrl+c`: Exit the program

## Prompt Pane

- `i`: Enters insert mode (you can now safely paste messages into the tui)
- `Ctrl+e`: Open/Close prompt editor 
- `Ctrl+r`: Clear prompt
- `Ctrl+v`: Paste text from buffer
- `Ctrl+s`: Paste text from buffer as a code block (only in editor mode)
    * if current line contains text, that text will be used as a language for the code block
    * Example: if a line contains `go` the result of `Ctrl+s` will be:

    \```go <br>
    {bufferContent} <br>
    \```
- `esc`: Exit insert mode for the prompt
    * When in 'Prompt editor' mode, pressing `esc` second time will close editor

## Chat Messages Pane

- `y`: Copies the last message from ChatGPT into your clipboard.
- `Shift+y`: Copies all messages from the ChatGPT session into your clipboard.
- `v`: Enters navigation mode when chat pane is focused (allows to move accross the chat content lines)

### Selection mode

![selection demo](./docs/images/selection-mode.gif)

Selection mode allows to navigate the chat pane and select lines to copy. Supports basic vim-motions.  

<b>Navigation</b>
 - `j`, `k` - go down and up a line
   - Multiline jumps like `3j` (3 lines down), `99k` (99 lines up) are also supported
 - `d`, `u`, `Ctrl+d`, `Ctrl+u` - go up or down half page
 - `g` - go to top
 - `Shift+g` - go to bottom

<b>Selection</b>
- `v`, `Shift+v` or `space` to enter or quit line selection mode
- `y` to copy selected text
- `Esc` to quit selection or navigation modes

## Settings Pane

- `m`: Opens a model picker to change the model. (use `j` to go up and `k` to go down the list)
- `f`: Opens an input dialog to change the frequency of updates.
- `t`: Opens an input dialog to set the maximum number of tokens per message.

## Sessions Pane

- `Ctrl+n`: Creates a new session.
- `d`: Deletes the currently selected session from the list.
- `e`: Edit session name
- `Enter`: Switches to the session that is currently selected.

## Info pane

Information pane displays processing state of inference (`IDLE`, `PROCESSING`) as well as token stats for the current session:
 - `IN`: shows the total amount of input tokens LLM consumed per session
 - `OUT`: shows the total amount of output tokens LLM produced per session

Please refer to this guide as you navigate the TUI. Happy exploring!

### Dev notes

The SQL db is stored in you `your/home/directory/.chatgpt-tui`, as well as the debug log. To enable `debug` mode, `export DEBUG=1` before running the program.

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

## Contributors

<!-- ALL-CONTRIBUTORS-LIST:START - Do not remove or modify this section -->
<!-- prettier-ignore-start -->
<!-- markdownlint-disable -->
<table>
  <tbody>
    <tr>
      <td align="center" valign="top" width="14.28%"><a href="https://twitter.com/tearingitup786"><img src="https://avatars.githubusercontent.com/u/16584942?v=4?s=100" width="100px;" alt="Taranveer (Taran) Bains"/><br /><sub><b>Taranveer (Taran) Bains</b></sub></a><br /><a href="#doc-tearingItUp786" title="Documentation">📖</a> <a href="#maintenance-tearingItUp786" title="Maintenance">🚧</a> <a href="#review-tearingItUp786" title="Reviewed Pull Requests">👀</a> <a href="#code-tearingItUp786" title="Code">💻</a></td>
      <td align="center" valign="top" width="14.28%"><a href="https://www.tjmiller.me"><img src="https://avatars.githubusercontent.com/u/5108034?v=4?s=100" width="100px;" alt="TJ Miller"/><br /><sub><b>TJ Miller</b></sub></a><br /><a href="#doc-sixlive" title="Documentation">📖</a> <a href="#code-sixlive" title="Code">💻</a></td>
      <td align="center" valign="top" width="14.28%"><a href="https://github.com/BalanceBalls"><img src="https://avatars.githubusercontent.com/u/29193297?v=4?s=100" width="100px;" alt="BalanceBalls"/><br /><sub><b>BalanceBalls</b></sub></a><br /><a href="#doc-BalanceBalls" title="Documentation">📖</a> <a href="#code-BalanceBalls" title="Code">💻</a></td>
    </tr>
  </tbody>
</table>

<!-- markdownlint-restore -->
<!-- prettier-ignore-end -->

<!-- ALL-CONTRIBUTORS-LIST:END -->
