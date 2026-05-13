---
status: completed
goal_ready: false
---

# Feature: Magvideo External App

## Goal

Create `magvideo`, an external Tya + GTK4 desktop app for quickly recording
vertical selfie videos for chat. The app opens a 9:16 camera preview, records
from the selected camera and microphone, opens the finished video automatically,
and places either the video file or uploaded URL on the clipboard.

## Context

Short video messages are useful in chat, but current desktop workflows often
require opening a heavy editor or manually stitching together camera capture,
audio, transcoding, upload, and clipboard steps. `magvideo` should be a small
single-purpose Linux GUI app that is pleasant on Omarchy/Hyprland and similar
Wayland desktops.

The app should be implemented in Tya using the planned GTK4 external binding.
GTK4 covers the UI, but camera preview, microphone capture, encoding, and muxing
need a media backend. The first version should include a native GStreamer-based
media layer inside the app repository rather than waiting for a separate
complete media framework binding.

Assumed repository and package identity:

- repository: `https://github.com/komagata/magvideo`
- executable: `magvideo`
- first release target: `v0.1.0`

## Behavior

- Provide a standalone external app repository:

  ```text
  magvideo/
    tya.toml
    src/
      main.tya
      magvideo/
        App.tya
        Recorder.tya
        Devices.tya
        Settings.tya
        Upload.tya
        Clipboard.tya
        Storage.tya
    native/
      magvideo_media.c
      magvideo_upload.c
    include/
      magvideo_media.h
      magvideo_upload.h
    assets/
      style.css
      icons/
    tests/
    examples/
    README.md
  ```

- The repository builds one `magvideo` binary.
- The app starts directly into the recording view.
- The app uses a vertical preview area matching YouTube Shorts style aspect
  ratio: 9:16.
- The preview shows the currently selected PC camera.
- Recording captures video from the selected camera and audio from the selected
  microphone.
- Start and stop are controlled by icon buttons.
- The main screen shows:
  - live vertical preview,
  - recording start/stop icon,
  - recording progress bar,
  - live elapsed seconds display,
  - camera device dropdown,
  - microphone input dropdown,
  - settings icon.
- After recording stops:
  - the video file is finalized,
  - the video opens automatically with the system default video player,
  - clipboard handling runs according to settings.

## UI

- Main window:
  - GTK4 application window.
  - No traditional menu bar.
  - Suitable for undecorated or minimal Omarchy/Hyprland use.
  - Preview container keeps 9:16 aspect ratio.
  - Default preview target size is `360x640` CSS pixels.
  - Window may be resizable, but preview remains 9:16.
- Controls:
  - record button shows a record icon when idle,
  - record button shows a stop icon while recording,
  - progress bar fills from `0` to `max_duration_seconds`,
  - elapsed label updates at least once per second,
  - camera dropdown updates the active camera,
  - microphone dropdown updates the active input device,
  - settings button opens settings window/dialog.
- Device changes:
  - switching camera while idle immediately updates the preview,
  - switching microphone while idle updates the next recording,
  - switching devices while recording is disabled or prompts to stop first.
- Error states:
  - no camera found shows a clear inline error,
  - no microphone found allows video-only recording only when enabled in
    settings,
  - media permission or device-open failures show a clear error dialog.

## Recording

- Native media layer uses GStreamer through `pkg-config` dependencies:
  - `gstreamer-1.0`
  - `gstreamer-video-1.0`
  - `gstreamer-audio-1.0`
  - `gstreamer-app-1.0`
  - GTK-compatible video sink support where available.
- Output format:
  - default compatibility format: MP4 container, H.264 video, AAC audio,
  - the default exists to maximize Discord compatibility, including Discord on
    iOS/iPadOS,
  - the app must not make WebM the default output format,
  - optional open format: WebM container, VP9 video, Opus audio,
  - WebM may be offered as an explicit "open format" option, but the UI and
    README must document that it may not play in Discord on iOS/iPadOS,
  - fallback codecs are allowed only when documented and playable by common
    Linux video players.
- Default video geometry:
  - `720x1280`,
  - 30 fps,
  - center-crop camera input into 9:16.
- Quality presets:
  - `low`: 540x960, lower bitrate,
  - `standard`: 720x1280,
  - `high`: 1080x1920.
- Settings may override:
  - resolution preset,
  - fps,
  - video bitrate,
  - audio bitrate,
  - maximum duration seconds,
  - output directory,
  - filename template.
- Default maximum duration is 60 seconds.
- Recording auto-stops at the maximum duration.
- Output filenames include timestamp and a safe suffix, such as:
  - `magvideo-2026-05-14-231530.mp4`.
- When the selected output format is WebM, filenames use `.webm` and upload
  content type uses `video/webm`.

## Clipboard Behavior

- Clipboard mode is configurable:
  - local file,
  - S3 uploaded URL,
  - Google Cloud Storage uploaded URL.
- Local file mode:
  - places the recorded video on the clipboard as a file/URI list when GTK and
    the desktop support it,
  - also stores the file path as text fallback,
  - shows a clear success message.
- URL modes:
  - uploads the file,
  - places the resulting HTTPS URL as text on the clipboard,
  - shows the copied URL in the UI.
- Clipboard failures do not delete the video file.
- Upload failures leave the local file intact and show retry/copy-local-file
  actions.

## Settings

- Settings are stored in a local config file, for example:
  - `~/.config/magvideo/settings.toml`.
- Settings screen includes:
  - recording quality preset,
  - fps,
  - max duration,
  - output directory,
  - post-record clipboard mode,
  - upload destination,
  - local storage settings,
  - S3 settings,
  - Google Cloud Storage settings.
- Local storage settings:
  - output directory,
  - filename template.
- S3 settings:
  - endpoint URL, optional for AWS S3-compatible storage,
  - region,
  - bucket,
  - key prefix,
  - access key ID,
  - secret access key,
  - optional public base URL,
  - ACL/public-read option when supported.
- Google Cloud Storage settings:
  - bucket,
  - object prefix,
  - access key ID,
  - secret access key,
  - optional public base URL.
- First-version GCS upload uses Google Cloud Storage interoperability HMAC keys
  through the XML/S3-compatible API. Service-account JSON OAuth upload is out of
  scope for v0.1.
- Secret values should not be printed in logs.
- Settings screen must include a "test upload settings" action for S3 and GCS.

## Uploads

- Uploads happen after the local file is finalized.
- S3 upload:
  - uses HTTPS,
  - signs requests with AWS Signature Version 4,
  - supports AWS S3 and S3-compatible endpoints.
- GCS upload:
  - uses HTTPS,
  - uses GCS interoperability HMAC credentials,
  - uploads through the XML/S3-compatible endpoint.
- Uploaded object content type follows the selected output format:
  - MP4 uses `video/mp4`,
  - WebM uses `video/webm`.
- Upload key is based on configured prefix and output filename.
- Result URL:
  - if public base URL is set, use `public_base_url + key`,
  - otherwise derive provider default HTTPS URL where practical.
- Large upload progress may be shown in the same progress area after recording.
- Upload retries:
  - one automatic retry for transient network failures,
  - manual retry action in the completion UI.

## Native Boundary

- Native C wrappers should be thin and explicit.
- `magvideo_media.c` owns GStreamer initialization, device enumeration, preview
  pipeline, recording pipeline, start/stop/finalize, and error retrieval.
- `magvideo_upload.c` may own signing helpers only if pure Tya implementation is
  impractical.
- Public Tya APIs must not expose raw pointers.
- Media resources expose explicit stop/close methods.
- Double stop/close is a no-op.
- The app must not rely on finalizers for media cleanup.
- Long-running media and upload work must not block the GTK main loop.

## Scope

- New external repository `komagata/magvideo`.
- Tya GTK4 app implementation.
- Native media layer using GStreamer.
- Recording view and settings UI.
- Device enumeration and selection.
- Vertical preview and 9:16 recording.
- MP4 output with audio.
- Optional WebM/VP9/Opus output as an explicit open-format mode.
- Local file clipboard behavior.
- S3 upload and URL clipboard behavior.
- GCS interoperability-key upload and URL clipboard behavior.
- Config file persistence.
- Basic logs and user-facing error messages.
- README documenting dependencies, permissions, settings, upload credentials,
  clipboard behavior, and troubleshooting.
- Example GitHub release build workflow once Tya can build the app in CI.

## Out of Scope

- Mobile app.
- Windows/macOS support in v0.1.
- Browser/WebRTC implementation.
- Video editing timeline.
- Filters, stickers, captions, background replacement, or effects.
- Multiple clips per recording.
- Screen recording.
- Multi-camera compositing.
- OAuth browser flows for S3 or GCS.
- GCS service-account JSON upload in v0.1.
- Upload destinations other than local, S3, and GCS.
- Hosted backend service.

## Acceptance Criteria

- A separate `komagata/magvideo` repository builds one `magvideo` binary.
- Launching `magvideo` opens a GTK4 window with a 9:16 live camera preview.
- Camera and microphone dropdowns list available devices.
- Changing the selected camera while idle updates the preview.
- Pressing record starts recording and changes the icon to stop.
- Progress bar and elapsed seconds update during recording.
- Pressing stop finalizes an MP4 file.
- MP4/H.264/AAC is the default finalized file format.
- Optional WebM output is available only when explicitly selected and is labeled
  as less compatible with Discord on iOS/iPadOS.
- Recording auto-stops at the configured maximum duration.
- The finalized video opens automatically in the system default video player.
- Local file clipboard mode places the video file/URI on the clipboard with text
  fallback.
- S3 mode uploads the file and places the resulting URL on the clipboard.
- GCS mode uploads via interoperability HMAC credentials and places the
  resulting URL on the clipboard.
- Settings persist across app restarts.
- Invalid upload settings can be tested and produce clear errors.
- No camera, no microphone, media permission failures, encoder failures, upload
  failures, and clipboard failures have user-facing error paths.
- The GTK main loop remains responsive during recording and upload.
- Headless CI documents which tests are compile-only or skipped when no camera
  or display is present.

## Verification

In the external repository:

```sh
pkg-config --exists gtk4
pkg-config --exists gstreamer-1.0
pkg-config --exists gstreamer-video-1.0
pkg-config --exists gstreamer-audio-1.0
tya install
tya doctor native
tya test
tya run src/main.tya
```

Manual smoke:

```sh
magvideo
# select camera and microphone
# record a 5-second clip
# verify MP4 opens
# verify clipboard contains the configured file or URL
```

For this repository's spec tracking only:

```sh
test -f docs/prd/magvideo-external-app.md
rg -n "Magvideo External App" docs/prd/magvideo-external-app.md
```

## Dependencies

- Depends on the planned GTK4 external library.
- Requires native package support.
- Requires host GTK4 and GStreamer development/runtime packages.
- Uses Tya config, TOML, path, file, process/open, and clipboard-capable GTK4
  APIs.
- S3/GCS upload benefits from HTTP/TLS and crypto helpers; if those are not
  available in Tya at implementation time, keep signing/upload helpers in the
  app's native layer.

## Open Questions

None.
