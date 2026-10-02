# hypr-dock
### Interactive dock panel for Hyprland

Translations: [`Русский`](README_RU.md)

<img width="1360" height="768" alt="250725_16h02m52s_screenshot" src="https://github.com/user-attachments/assets/041d2cf6-13ba-4c89-a960-1903073ff2d4" />
<img width="1360" height="768" alt="250725_16h03m09s_screenshot" src="https://github.com/user-attachments/assets/0c1ad8ca-37c1-4fd6-a48d-46f74c2d2609" />

[![YouTube](https://img.shields.io/badge/YouTube-Video-FF0000?logo=youtube)](https://youtu.be/HHUZWHfNAl0?si=ZrRv2ggnPBEBS5oY)
[![AUR](https://img.shields.io/badge/AUR-Package-1793D1?logo=arch-linux)](https://aur.archlinux.org/packages/hypr-dock)

## Installation

### Dependencies

- `go` (make)
- `gtk3`
- `gtk-layer-shell`

### Installation
! The first build may take a very long time due to gtk3 bindings !
```bash
git clone https://github.com/lotos-linux/hypr-dock.git
cd hypr-dock
make get
make build
make install
```

### Uninstallation
```bash
make uninstall
```

### Local run (dev mode)
```bash
make exec
```

## Running

### Launch parameters:

```text
  -config string
    	config file (default "~/.config/hypr-dock")
  -dev
    	enable developer mode
  -log-level string
    	log level (default "info")
  -theme string
    	theme dir
```
#### All parameters are optional.

Default configuration and themes are installed in `/etc/hypr-dock`
On first run, they are copied to `~/.config/hypr-dock`
### Add to `hyprland.conf`:

```text
exec-once = hypr-dock
bind = Super, D, exec, hypr-dock
```

For Lua-based Hyprland configurations, copy
`configs/default/hypr-dock.lua` to `~/.config/hypr/config/hypr-dock.lua` and
add this line to `~/.config/hypr/hyprland.lua`:

```lua
require("config.hypr-dock")
```

That provides the macOS-style startup, `Super+D` toggle, and blur rules.

### And configure blur if needed
```text
layerrule = blur true,match:namespace hypr-dock
layerrule = ignore_alpha 0,match:namespace hypr-dock
layerrule = blur true,match:namespace dock-popup
layerrule = ignore_alpha 0,match:namespace dock-popup
```

#### The dock only supports one running instance, so running it again will close the previous one.

## Configuration

#### !Important!
If you already had `hypr-dock` installed or you are updating, please note that starting from version 1.2.0, an `ini`-like configuration format is now used. If you have old configs remaining in your user folder, be sure to delete or move them. Then simply adapt your settings to the new format.

### Available parameters in `hypr-dock.conf`

```ini
[General]
CurrentTheme = lotos

# Icon size (px) (default 23)
IconSize = 23

# Window overlay layer height (background, bottom, top, overlay) (default top)
Layer = top

# Exclusive Zone (true, false) (default true)
Exclusive = true

# SmartView (true, false) (default false)
SmartView = false

# Window position on screen (top, bottom, left, right) (default bottom)
Position = bottom

# Delay before hiding the dock (ms) (default 400)
AutoHideDelay = 400   # Only for SmartView

# Use system gap (true, false) (default true)
SystemGapUsed = true

# Indent from the edge of the screen (px) (default 8)
Margin = 8

# Distance of the context menu from the window (px) (default 5)
ContextPos = 5

# Dock follows the cursor across monitors (default true)
FollowMouse = true

# Dock sections (default: pinned+running shown, recent off)
ShowPinnedApps = true
ShowRunningApps = true
ShowRecentApps = false

# Running indicator dots (default on, 56% of icon size)
ShowWindowCount = true
IndicatorSize = 56

# Click semantics for running apps (launch, focus, minimize, show, cycle, none)
ClickAction = focus          # on an unfocused running app
ClickActionFocused = minimize # on the already-focused app
MiddleClickAction = launch    # launch, none

# Launcher button at the start/end of the dock
ShowLauncherButton = false
LauncherCommand =            # empty = auto-detect (omarchy-launch-* / rofi / fuzzel / wofi)
LauncherPosition = start     # start, end

# Follow Omarchy's active theme colors live (see below)
FollowOmarchy = false

# Window classes that never get a dock icon; comma separated,
# `*` wildcards allowed (e.g. "scratch-*, *-test")
HiddenApps =

[Appearance]
# Structured overrides for the active theme (0 = keep theme value)
# PanelOpacity - panel alpha in percent (1-100)
PanelOpacity = 0
BorderRadius = 0
BorderWidth = 0
PanelPadding = 0

# Force-disable hover effects / tint running apps with an accent color
HoverEffects = true
ActiveTint = false

# Accent color for the active tint; empty = theme/omarchy accent
Accent =

[Displays]
# Show the dock on (focused, primary, specific, all)
Mode = focused

# Monitor name when Mode = specific
MonitorName =

[General.preview]
# Window thumbnail mode selection (none, live, static) (default none)
Mode = none
# "none"   - disabled (text menus)
# "static" - last window frame
# "live"   - window streaming
      
# !WARNING! 
# BY SETTING "Mode" TO "live" OR "static", YOU AGREE TO THE CAPTURE 
# OF WINDOW CONTENTS.
# THE "HYPR-DOCK" PROGRAM DOES NOT COLLECT, STORE, OR TRANSMIT ANY DATA.
# WINDOW CAPTURE OCCURS ONLY FOR THE DURATION OF THE THUMBNAIL DISPLAY!
#   
# Source code: https://github.com/lotos-linux/hypr-dock

# Live preview fps (0 - ∞) (default 30)
FPS = 30

# Live preview bufferSize (1 - 20) (default 5)
BufferSize = 5

# Popup show/hide/move delays (ms)
ShowDelay = 500  # (default 500)
HideDelay = 350  # (default 350)
MoveDelay = 100  # (default 100)
```
#### If a parameter is not specified, the default value will be used

## Understanding non-obvious parameters

### SmartView
Similar to auto-hide: if `true`, the dock stays beneath all windows, but moves the cursor to the screen edge - the dock rises above them

### Exclusive
Activates special layer behavior where tiling windows do not overlap the dock

### SystemGapUsed
- When `SystemGapUsed = true`, the dock sets its margin from the screen edge using values from the `hyprland` configuration, specifically `general:gaps_out` values, and dynamically updates when the `hyprland` configuration changes
- When `SystemGapUsed = false`, the margin from the screen edge is set by the `Margin` parameter

### General.preview
- `ShowDelay`, `HideDelay`, `MoveDelay` - preview popup action delays in milliseconds
- `FPS`, `BufferSize` - only used when `Mode = live`

#### Preview appearance settings are configured through theme files

## Settings application & control CLI

This build ships a GTK settings application and a control plane CLI:

- `hypr-dock-settings` — a windowed settings app (Appearance, Behavior,
  Applications, Windows, Displays, Themes, Advanced). Changes are written to
  the same config files and applied to the running dock live. Open it via the
  dock's context menu (`Dock Settings`), `hypr-dockctl settings`, or directly.
- `hypr-dockctl` — command-line control for a running dock:

```text
  reload                       apply config/theme/pinned from disk
  set <Section.Key> <value>    write one key, apply live
  get <Section.Key>            read one key
  status                       dock summary
  settings                     open the settings application
  pin <class> [position]       pin an application (insert at position)
  unpin <class>                unpin an application
  quit                         stop the dock
  doctor                       diagnostics
  version                      print version
```

`set`/`get` use dotted paths matching the config sections, e.g.
`hypr-dockctl set General.IconSize 56` or `hypr-dockctl get Appearance.PanelOpacity`.
Every write preserves comments and unknown keys in the config file, and is
applied to the running dock without a restart.

### Omarchy integration

When `FollowOmarchy = true` (General section), the dock tracks Omarchy's active
theme colors and regenerates its `omarchy` theme (panel colors, hover and
indicator dots) when the theme changes — live, without a restart. The
`doctor` subcommand verifies the integration.

### Pinned applications are stored in `~/.local/share/hypr-dock/pinned`
To pin/unpin, open the application's context menu in the dock and click `pin`/`unpin`
#### Example
```text
firefox
code-oss
kitty
org.telegram.desktop
nemo
org.kde.ark
sublime_text
qt6ct
one.ablaze.floorp
```
You can edit it manually. But why? ¯\_(ツ)_/¯

## Themes

#### Themes are located in `~/.config/hypr-dock/themes/`

### A theme consists of
- `theme.conf`
- `style.css`
- A folder with `svg` files for indicating the number of running applications (see [themes_EN.md](https://github.com/lotos-linux/hypr-dock/blob/main/docs/customize/themes_EN.md))

### Theme config
```ini
[Theme]
# Distance between elements (px) (default 9)
Spacing = 5


[Theme.preview]
# Size (px) (default 120)
Size = 120

# Image/Stream border-radius (px) (default 0)
BorderRadius = 0

# Popup padding (px) (default 10)
Padding = 10
```
#### Customize `style.css` as you wish. Detailed styling documentation will be provided later.
