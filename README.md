# radio-stream-recorder

A program that extracts the individual tracks from an Ogg/Vorbis http radio stream (without any loss in quality). Written in go without any non-standard dependencies.

MP3 support is planned but not yet implemented.

## Building

- Install [go](https://golang.org/) (preferably a recent version)

- `go build`

## Usage

- General usage: `./rsr [-dir <OUTPUT_DIRECTORY>] <RADIO_STREAM_URL>`

- see `./rsr -h` for integrated usage documentation
