-- Hypr-Dock defaults for Hyprland's Lua configuration.
-- Add `require("config.hypr-dock")` to hyprland.lua after installing.

hl.layer_rule({ match = { namespace = "hypr-dock" }, blur = true, ignore_alpha = 0 })
hl.layer_rule({ match = { namespace = "dock-popup" }, blur = true, ignore_alpha = 0 })

hl.bind("SUPER + D", hl.dsp.exec_cmd("hypr-dock"), {
  description = "Toggle Hypr-Dock",
})

hl.on("hyprland.start", function()
  hl.exec_cmd("hypr-dock")
end)
