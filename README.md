# orgcal

Bidirectional sync between org-mode files and Google Calendar via CLI.

## Features

- Import all Google Calendar events into org files
- Export org `TODO`/`NEXT` headings with `SCHEDULED`/`DEADLINE` to Google Calendar
- Delete GCal events when org heading is marked `DONE`/`CANCELLED`
- All-day event support
- Time range support (`<start>--<end>`)
- Multi-calendar support via `:CALENDAR_ID:` property
- Opt-in export via `:EXPORT_TO_GCAL: t`
- Neovim integration with auto-sync on save

## Requirements

- Go 1.21+
- A Google Cloud project with Calendar API enabled

## Google Cloud Setup

1. Go to [Google Cloud Console](https://console.cloud.google.com/)
2. Create a new project
3. Enable the **Google Calendar API**: APIs & Services → Library → search "Google Calendar API" → Enable
4. Create OAuth credentials: APIs & Services → Credentials → Create Credentials → OAuth client ID
   - Application type: **Desktop app**
5. Under **Authorized redirect URIs**, add: `http://localhost:8765`
6. Copy the **Client ID** and **Client Secret**

## Installation

```sh
git clone https://github.com/angshumanhalder/orgcal
cd orgcal
go install .
```

Add to your shell config (`~/.zshrc` or `~/.bashrc`):

```sh
export ORGCAL_CLIENT_ID="your-client-id.apps.googleusercontent.com"
export ORGCAL_CLIENT_SECRET="your-client-secret"
```

Reload your shell:

```sh
source ~/.zshrc
```

Authenticate once:

```sh
orgcal auth
```

A browser window opens. Authorize the app. The token is saved to `~/.local/share/orgcal/token.json` and auto-refreshes.

## Usage

```sh
orgcal auth                      # authenticate with Google
orgcal sync   --dir ~/org        # bidirectional sync
orgcal import --dir ~/org        # GCal → org only
orgcal export --dir ~/org        # org → GCal only
```

Imported events are written to `~/org/gcal/calendar.org`.

## Org File Format

### Imported events

```org
* Team Standup
  SCHEDULED: <2026-06-27 Sat 10:00>--<2026-06-27 Sat 10:30>
  :PROPERTIES:
  :GCAL_ID: abc123xyz
  :LOCATION: Conference Room A
  :GCAL_UPDATED: 2026-06-27T10:00:00Z
  :END:
```

### Exporting a TODO to GCal

Only headings with `SCHEDULED` or `DEADLINE` are exported. Mark with `:EXPORT_TO_GCAL: t` to opt in (required unless heading state is `TODO`/`NEXT`):

```org
* TODO Team Meeting
  SCHEDULED: <2026-06-27 Sat 14:00>--<2026-06-27 Sat 15:00>
  :PROPERTIES:
  :EXPORT_TO_GCAL: t
  :END:
  Discuss Q3 roadmap.
```

### Sync to a specific calendar

```org
* TODO Work Review
  SCHEDULED: <2026-06-27 Sat 09:00>
  :PROPERTIES:
  :EXPORT_TO_GCAL: t
  :CALENDAR_ID: work@company.com
  :END:
```

### Delete from GCal when done

```org
* DONE Team Meeting
  :PROPERTIES:
  :GCAL_ID: abc123xyz
  :END:
```

Marking a heading `DONE` or `CANCELLED` with a `:GCAL_ID:` deletes the event from Google Calendar on next sync.

## Neovim Integration

Add to your `init.lua`:

```lua
local orgcal_dir = "~/org"

local function orgcal_run(subcmd, cb)
  local cmd = { "orgcal", subcmd, "--dir", orgcal_dir }
  local output = {}
  vim.fn.jobstart(cmd, {
    stdout_buffered = true,
    on_stdout = function(_, data)
      for _, line in ipairs(data) do
        if line ~= "" then table.insert(output, line) end
      end
    end,
    on_stderr = function(_, data)
      for _, line in ipairs(data) do
        if line ~= "" then vim.notify("orgcal: " .. line, vim.log.levels.ERROR) end
      end
    end,
    on_exit = function(_, code)
      if code == 0 then
        local msg = #output > 0 and table.concat(output, " ") or subcmd .. " done"
        vim.notify("orgcal: " .. msg, vim.log.levels.INFO)
        if cb then cb() end
      end
    end,
  })
end

vim.api.nvim_create_user_command("OrgCalAuth", function()
  vim.fn.jobstart({ "orgcal", "auth" }, {
    on_exit = function(_, code)
      if code == 0 then vim.notify("orgcal: authenticated", vim.log.levels.INFO) end
    end,
  })
end, { desc = "Authenticate orgcal with Google Calendar" })

vim.api.nvim_create_user_command("OrgCalSync",   function() orgcal_run("sync",   nil) end, { desc = "Bidirectional sync" })
vim.api.nvim_create_user_command("OrgCalImport", function() orgcal_run("import", nil) end, { desc = "Import GCal → org" })
vim.api.nvim_create_user_command("OrgCalExport", function() orgcal_run("export", nil) end, { desc = "Export org → GCal" })

-- auto-sync on org file save
vim.api.nvim_create_autocmd("BufWritePost", {
  pattern = "*.org",
  callback = function() orgcal_run("sync", nil) end,
})
```

### Commands

| Command | Description |
|---|---|
| `:OrgCalAuth` | Authenticate with Google Calendar |
| `:OrgCalSync` | Bidirectional sync |
| `:OrgCalImport` | Import GCal events to org |
| `:OrgCalExport` | Export org TODOs to GCal |

## Environment Variables

| Variable | Description |
|---|---|
| `ORGCAL_CLIENT_ID` | Google OAuth client ID |
| `ORGCAL_CLIENT_SECRET` | Google OAuth client secret |

## License

MIT
