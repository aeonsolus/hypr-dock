SYSTEM_CONFIG_DIR = /etc/hypr-dock

PROJECT_BIN_DIR = bin
PROJECT_CONFIG_DIR = configs/default

EXECUTABLE_DOCK = hypr-dock
EXECUTABLE_ALTTAB = hypr-alttab
EXECUTABLE_CTL = hypr-dockctl
EXECUTABLE_SETTINGS = hypr-dock-settings

CMD_DOCK = ./cmd/hypr-dock/.
CMD_ALTTAB = ./cmd/hypr-alttab/.

RESET := \033[0m
GREEN := \033[32m
YELLOW := \033[33m

LOG_LEVEL ?= trace

get:
	go mod tidy

warn:
	@if [ ! -f "$(PROJECT_BIN_DIR)/$(EXECUTABLE_DOCK)" ]; then \
		echo -e "$(YELLOW)The first build may take an extremely long time due to linking with gtk3...$(RESET)"; \
	fi

build-all:
	$(MAKE) build-dock
	$(MAKE) build-alttab
	$(MAKE) build-ctl
	$(MAKE) build-settings

build: build-all

build-dock:
	$(MAKE) warn
	go build -v -o $(PROJECT_BIN_DIR)/$(EXECUTABLE_DOCK) $(CMD_DOCK)

build-alttab:
	$(MAKE) warn
	go build -v -o $(PROJECT_BIN_DIR)/$(EXECUTABLE_ALTTAB) $(CMD_ALTTAB)

build-ctl:
	go build -v -o $(PROJECT_BIN_DIR)/$(EXECUTABLE_CTL) ./cmd/hypr-dockctl/.

build-settings:
	go build -v -o $(PROJECT_BIN_DIR)/$(EXECUTABLE_SETTINGS) ./cmd/hypr-dock-settings/.

install: install-all

install-dock:
	-sudo killall $(EXECUTABLE_DOCK) 2>/dev/null || true
	sudo cp $(PROJECT_BIN_DIR)/$(EXECUTABLE_DOCK) /usr/bin/
	@echo -e "$(GREEN)hypr-dock installed$(RESET)"

install-alttab:
	-sudo killall $(EXECUTABLE_ALTTAB) 2>/dev/null || true
	sudo cp $(PROJECT_BIN_DIR)/$(EXECUTABLE_ALTTAB) /usr/bin/
	@echo -e "$(GREEN)hypr-alttab installed$(RESET)"

install-ctl:
	sudo cp $(PROJECT_BIN_DIR)/$(EXECUTABLE_CTL) /usr/bin/
	@echo -e "$(GREEN)hypr-dockctl installed$(RESET)"

install-settings:
	-sudo killall $(EXECUTABLE_SETTINGS) 2>/dev/null || true
	sudo cp $(PROJECT_BIN_DIR)/$(EXECUTABLE_SETTINGS) /usr/bin/
	@echo -e "$(GREEN)hypr-dock-settings installed$(RESET)"

update-config:
	sudo -rf $(PROJECT_CONFIG_DIR)/. $(SYSTEM_CONFIG_DIR)/
	@echo -e "$(GREEN)Configs copied to $(SYSTEM_CONFIG_DIR)$(RESET)"

# Deploy this fork's complete workstation defaults, including the current
# theme, pins, absolute dock order, and Lua Hyprland integration.
install-user-config:
	mkdir -p $(HOME)/.config/hypr-dock $(HOME)/.config/hypr/config $(HOME)/.local/share/hypr-dock
	cp -a $(PROJECT_CONFIG_DIR)/hypr-dock.conf $(HOME)/.config/hypr-dock/
	cp -a $(PROJECT_CONFIG_DIR)/themes $(HOME)/.config/hypr-dock/
	cp -a $(PROJECT_CONFIG_DIR)/pinned $(HOME)/.local/share/hypr-dock/
	@if [ -f "$(PROJECT_CONFIG_DIR)/order" ]; then cp -a $(PROJECT_CONFIG_DIR)/order $(HOME)/.local/share/hypr-dock/; fi
	cp -a $(PROJECT_CONFIG_DIR)/hypr-dock.lua $(HOME)/.config/hypr/config/
	cp -a $(PROJECT_CONFIG_DIR)/hypr-dock-hyprland.conf $(HOME)/.config/hypr/config/
	@if [ -f "$(HOME)/.config/hypr/hyprland.lua" ] && ! grep -Fq 'require("config.hypr-dock")' "$(HOME)/.config/hypr/hyprland.lua"; then printf '\nrequire("config.hypr-dock")\n' >> "$(HOME)/.config/hypr/hyprland.lua"; fi
	@if [ -f "$(HOME)/.config/hypr/hyprland.conf" ] && ! grep -Fq 'config/hypr-dock-hyprland.conf' "$(HOME)/.config/hypr/hyprland.conf"; then printf '\nsource = ~/.config/hypr/config/hypr-dock-hyprland.conf\n' >> "$(HOME)/.config/hypr/hyprland.conf"; fi
	@echo -e "$(GREEN)User dock config, themes, pins, order, and Hyprland hook installed$(RESET)"

install-all:
	$(MAKE) install-dock
	$(MAKE) install-alttab
	$(MAKE) install-ctl
	$(MAKE) install-settings
	$(MAKE) update-config
	$(MAKE) install-user-config

uninstall:
	sudo rm -f /usr/bin/$(EXECUTABLE_DOCK)
	sudo rm -f /usr/bin/$(EXECUTABLE_ALTTAB)
	sudo rm -rf $(SYSTEM_CONFIG_DIR)
	@echo -e "$(GREEN)Uninstalled.$(RESET)"

exec:
	./bin/hypr-dock -dev -log-level $(LOG_LEVEL)

release:
	@test -n "$(VERSION)" || (echo "usage: make release VERSION=1.3.2-custom"; exit 2)
	./scripts/release.sh "$(VERSION)"
