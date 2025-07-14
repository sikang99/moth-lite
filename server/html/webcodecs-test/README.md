# moth-webcodecs

## DEMO

Encoder : <https://cobot.center:8287/static/webcodecs/encoder/>

Decoder : <https://cobot.center:8287/static/webcodecs/decoder/>

## Feature

- Supports 4 Video Codecs
  - VP9 (vp09.00.31.08) - Default
  - H.264 (encoder - avc1.42E01F, decoder - avc1.64001E)
  - VP8
  - AV1 (av01.0.05M.10)

- Support 2 Image Codecs
  - JPEG
  - PNG

- Supports 1 Audio Codec
  - Opus - Default

- Support Recording
  - WebM Container (VP9 + Opus only)

- Decoder Automatically Detects changes in the encoding codec (by Moth Mime Policy).

## Update

### v0.3.1 - Apr 28, 2022

- Support Image
  - JPEG, PNG

- Fix Minor Bugs
  - Moth Config Name works properly
  - Record av1 works properly

### v0.3 - Apr 08, 2022

- Support Recording
  - WebM Container (VP9 + Opus only)

- Lists Browser Supporting Codecs on Encoder

- Edit H.264 Profile
  - Encoder uses avc1.42E01F
  - Decoder uses avc1.64001E

- Tested on
  - MacOS 12.3 (x86) + Chrome 100.0.4896.75 + 2018 MBP 15"
  - Android 11 + Chrome 100.0.4896.79 + Galaxy S21 5G
  - Ubuntu 18.04 (ARM) + Chromium 97.0.4692.71 + Jetson Nano 4GB

### v0.2 - Apr 06, 2022

- Support Video Codec
  - VP9 (vp09.00.31.08)

- Support Audio Codec
  - Opus

### v0.1 - Apr 03, 2022

- Supports 3 Video Codecs
  - H.264 (avc1.64001E)
  - VP8
  - AV1 (av01.0.05M.10)
- Decoder Automatically Detects changes in the encoding codec (by Moth Mime Policy).

## How to Use

### Encoder

1. Save Moth Config (Placeholders are default values).
2. Save Video Codec Config (Placeholders are default values).
3. Save Audio Codec Config (Placeholders are default values).
4. Open Webcam Stream.
5. Open Mic Stream.
6. Start Encoding.

### Decoder

1. Save Moth Config (Placeholders are default values).
2. Start Decoding.
3. Decoder is automatically configured by Moth MIME Policy.

**Codec Change Test** **(While Streaming)**

1. Change Encoding Codec Config and Save.
2. Stop and Re-Start Encoding.
3. Decoder Automatically detects changes and restart Decoding.

## Features Under-development

### v0.4

- Dynamic bitRate Control

## Author

sjk (sjk@teamgrit.kr)
