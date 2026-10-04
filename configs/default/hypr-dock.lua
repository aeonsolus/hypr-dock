-- Hypr-Dock defaults for Hyprland's Lua configuration.
-- Add `require("config.hypr-dock")` to hyprland.lua after installing.

hl.layer_rule({ match = { namespace = "hypr-dock" }, blur = true, ignore_alpha = 0 })
hl.layer_rule({ match = { namespace = "dock-popup" }, blur = true, ignore_alpha = 0 })

-- Preferences are a compact resizable dialog, independent of desktop opacity.
hl.window_rule({
  match = { class = "^hypr-dock-settings$" },
  float = true,
  center = true,
  size = { 900, 740 },
  opacity = "1.0 override 1.0 override",
})

hl.bind("SUPER + D", hl.dsp.exec_cmd("hypr-dock"), {
  description = "Toggle Hypr-Dock",
})

hl.on("hyprland.start", function()
  hl.exec_cmd("hypr-dock")
end)
