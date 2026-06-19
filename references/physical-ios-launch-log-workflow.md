# Physical iOS Launch And Log Workflow

Use this workflow when an agent must start an iOS app on connected physical devices, optionally rebuild first, and keep runtime logs attached while the user performs a manual scenario.

## Contract

The project should expose two layers:

1. Generic scripts under the project-local `scripts/` directory.
2. A short Makefile facade in the generated Makefile's preserved custom section.

The scripts must not hardcode local device names, local UDIDs, user phone numbers, or project-specific test scenarios. Project defaults such as bundle id, scheme, app product name, and launch environment belong in Makefile variables or environment variables.

## Device Discovery

Prefer a small helper that lists currently available USB physical iOS devices:

```bash
scripts/list-usb-ios-devices.sh
scripts/list-usb-ios-devices.sh --ids-only
```

Implementation requirements:

- Use `xcrun xcdevice list`.
- Filter to `platform == com.apple.platform.iphoneos`.
- Exclude simulators.
- Require `available == true`.
- Prefer `interface == usb` for deterministic physical work.
- Print one UDID per line in `--ids-only` mode.

## Build And Install

The build/install script should accept an explicit device list:

```bash
scripts/build-connected-ios-devices.sh --devices "<udid1,udid2>" --derived-data-root ".temp/device-builds"
```

Implementation requirements:

- Fail with status `2` if the explicit device list is required but missing.
- Print the device-list helper command in the error message.
- Build each selected device with its own DerivedData/log directory.
- Install through `devicectl device install app` when CoreDevice can reach the device.
- Fall back to `ios-deploy --bundle <app> --noninteractive --nostart --no-wifi` for legacy devices that are visible to `xcdevice` but not launchable/installable through CoreDevice.

## Launch With Logs

The launch/log script should default to all currently connected USB devices, while still accepting an explicit list:

```bash
scripts/launch-connected-ios-devices-with-logs.sh --devices "<udid1,udid2>"
scripts/launch-connected-ios-devices-with-logs.sh --rebuild
scripts/launch-connected-ios-devices-with-logs.sh --duration 120 -- -SomeLaunchArgument value
```

Implementation requirements:

- Start log capture before launching the app.
- Keep the script in the foreground until interrupted, unless a duration was provided.
- Store all artifacts under a timestamped `.temp/` directory.
- Write per-device raw syslog, stderr, launch logs, and heartbeat/status files.
- Generate filtered logs at shutdown with a project-provided regex when `rg` is available.
- Kill child log processes in `INT`, `TERM`, and `EXIT` cleanup.
- Launch through `devicectl device process launch --device <udid> --terminate-existing <bundle-id>` when CoreDevice works.
- Fall back to `ios-deploy --id <udid> --bundle <app-bundle-path> --noinstall --justlaunch --noninteractive --no-wifi` for legacy devices. This fallback needs a local `.app` bundle path, so resolve it from the rebuild output or from the project's current DerivedData.

## Makefile Facade

Generated Makefiles preserve the custom section. Put project-specific defaults and short commands there:

```make
IOS_DEVICE_LAUNCH_REBUILD ?= 0
IOS_DEVICE_LAUNCH_OPTIONS ?=

.PHONY: launch-connected-ios-devices-with-logs rebuild-launch-connected-ios-devices-with-logs
launch-connected-ios-devices-with-logs: ## Launch app on connected physical iOS devices and attach runtime logs
	@set -e; \
	rebuild_flag="--no-rebuild"; \
	if [ "$(IOS_DEVICE_LAUNCH_REBUILD)" = "1" ]; then rebuild_flag="--rebuild"; fi; \
	scripts/launch-connected-ios-devices-with-logs.sh \
		--bundle-id "$(BUNDLE_ID)" \
		$$rebuild_flag \
		$(IOS_DEVICE_LAUNCH_OPTIONS)

rebuild-launch-connected-ios-devices-with-logs: ## Rebuild, install, launch, and attach runtime logs on connected physical iOS devices
	@$(MAKE) launch-connected-ios-devices-with-logs IOS_DEVICE_LAUNCH_REBUILD=1
```

Example usage:

```bash
DEVELOPER_DIR=/Applications/Xcode.app/Contents/Developer \
make launch-connected-ios-devices-with-logs
```

```bash
DEVELOPER_DIR=/Applications/Xcode.app/Contents/Developer \
make rebuild-launch-connected-ios-devices-with-logs
```

```bash
DEVELOPER_DIR=/Applications/Xcode.app/Contents/Developer \
IOS_DEVICE_LAUNCH_OPTIONS='--duration 60 -- -AppEnvironment dev' \
make launch-connected-ios-devices-with-logs
```

## Agent Operation

When the user asks to start this workflow:

1. Run the Makefile facade in a foreground terminal session.
2. Wait until the script prints a ready marker.
3. Tell the user they can start the manual scenario.
4. Keep the session alive until the user says to stop.
5. Stop the session with Ctrl-C.
6. Confirm child log processes are gone.
7. Inspect filtered logs first, then raw logs only when needed.

Do not use a background command unless a supervisor script records child PIDs, writes a heartbeat, and has an explicit stop command. Foreground capture is safer for manual physical-device debugging because it makes the lifecycle visible and prevents stale log streams.
