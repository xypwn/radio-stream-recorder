# radio-stream-recorder

A program that extracts the individual tracks from an Ogg/Vorbis or mp3 radio stream. Written in go without any non-standard dependencies.

## Obtaining the binary

### Via [releases](https://git.nobrain.org/r4/radio-stream-recorder/releases/latest)

- Download the binary for your system from https://git.nobrain.org/r4/radio-stream-recorder/releases/latest

- That should be it, just open a terminal and run it; see usage

### From source

- Clone the git repo and cd into it

- Install [go](https://golang.org/) (preferably a recent version)

- `go build`

## Usage

- General usage: `./rsr [-dir <OUTPUT_DIRECTORY>] <RADIO_STREAM_URL>`

- see `./rsr -h` for integrated usage documentation
