package main

import (
	"bytes"
	"fmt"
	"net/http"
	"os"
	"path"
	"strconv"

	"rsr/model"
	"rsr/mp3"
	"rsr/util"
	"rsr/vorbis"
)

var client = new(http.Client)

const (
	colRed    = "\033[31m"
	colYellow = "\033[33m"
	colReset  = "\033[m"
)

var (
	nTracksRecorded int // Number of recorded tracks.
	limitTracks     bool
	maxTracks       int
)

func usage(arg0 string, exitStatus int) {
	fmt.Fprintln(os.Stderr, `Usage:
  `+arg0+` [options...] <STREAM_URL>

Options:
  -dir <DIRECTORY>  --  Output directory (default: ".").
  -n <NUM>          --  Stop after <NUM> tracks.

Output types:
  * <INFO>
  `+colYellow+`! <WARNING>`+colReset+`
  `+colRed+`! <ERROR>`+colReset)
	os.Exit(exitStatus)
}

func printInfo(f string, v ...interface{}) {
	fmt.Printf("* "+f+"\n", v...)
}

func printWarn(f string, v ...interface{}) {
	fmt.Fprintf(os.Stderr, colYellow+"! "+f+colReset+"\n", v...)
}

func printNonFatalErr(f string, v ...interface{}) {
	fmt.Fprintf(os.Stderr, colRed+"! "+f+colReset+"\n", v...)
}

func printErr(f string, v ...interface{}) {
	printNonFatalErr(f, v...)
	os.Exit(1)
}

func record(url, dir string) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		printErr("HTTP request error: %v", err)
	}
	req.Header.Add("Icy-MetaData", "1") // Request metadata for icecast mp3 streams.
	resp, err := client.Do(req)
	if err != nil {
		printErr("HTTP error: %v", err)
	}
	defer resp.Body.Close()

	var extractor model.Extractor

	// Set up extractor depending on content type.
	contentType := resp.Header.Get("content-type")
	supported := "Ogg/Vorbis ('application/ogg'), mp3 ('audio/mpeg')"
	err = nil
	switch contentType {
	case "application/ogg":
		extractor, err = vorbis.NewExtractor()
	case "audio/mpeg":
		extractor, err = mp3.NewExtractor(resp.Header)
	default:
		printErr("Content type '%v' not supported, supported formats: %v", contentType, supported)
	}
	if err != nil {
		printErr("%v", err)
	}

	printInfo("Stream type: '%v'", contentType)

	// Make reader blocking.
	r := util.NewWaitReader(resp.Body)

	// The first track is always discarded, as streams usually don't start at
	// the exact end of a track, meaning it is almost certainly going to be
	// incomplete.
	discard := true

	var rawFile bytes.Buffer
	var filename string
	var hasFilename bool

	for {
		var block bytes.Buffer

		wasFirst, err := extractor.ReadBlock(r, &block)
		if err != nil {
			printNonFatalErr("Error reading block: %v", err)
			// Reconnect, because this error is usually caused by a
			// file corruption or a network error.
			return
		}

		if wasFirst &&
			// We only care about the beginning of a new file when it marks an
			// old file's end, which is not the case in the beginning of the
			// first file.
			rawFile.Len() > 0 {
			if !discard {
				// Save previous track.
				if !hasFilename {
					printNonFatalErr("Error: Could not get a track filename")
					continue
				}
				filePath := path.Join(dir, filename)
				err := os.WriteFile(filePath, rawFile.Bytes(), 0666)
				if err != nil {
					printNonFatalErr("Error writing file: %v", err)
					continue
				}
				printInfo("Saved track as: %v", filePath)

				// Stop after the defined number of tracks (if the option was
				// given).
				nTracksRecorded++
				if limitTracks && nTracksRecorded >= maxTracks {
					printInfo("Successfully recorded %v tracks, exiting", nTracksRecorded)
					os.Exit(0)
				}
			} else {
				// See declaration of `discard`.
				discard = false
			}

			// Reset everything.
			rawFile.Reset()
			hasFilename = false
		}

		// Try to find out the current track's filename.
		if !hasFilename {
			if f, ok := extractor.TryGetFilename(); ok {
				if discard {
					printInfo("Discarding track: %v", f)
				} else {
					printInfo("Recording track: %v", f)
				}
				filename = f
				hasFilename = true
			}
		}

		// Append block to the current file byte buffer.
		rawFile.Write(block.Bytes())
	}
}

func main() {
	var url string
	dir := "."

	if len(os.Args) < 2 {
		usage(os.Args[0], 1)
	}

	// Parse command line arguments.
	for i := 1; i < len(os.Args); i++ {
		// Returns the argument after the given option. Errors if there is no
		// argument.
		expectArg := func(currArg string) string {
			i++
			if i >= len(os.Args) {
				printErr("Expected argument after option '%v'", currArg)
			}
			return os.Args[i]
		}

		arg := os.Args[i]
		if len(arg) >= 1 && arg[0] == '-' {
			switch arg {
			case "-dir":
				dir = expectArg(arg)
			case "-n":
				nStr := expectArg(arg)
				n, err := strconv.ParseInt(nStr, 10, 32)
				if err != nil || n <= 0 {
					printErr("'%v' is not an integer larger than zero", nStr)
				}
				limitTracks = true
				maxTracks = int(n)
			case "--help", "-h":
				usage(os.Args[0], 0)
			default:
				printErr("Unknown option: '%v'", arg)
			}
		} else {
			if url == "" {
				url = arg
			} else {
				printErr("Expected option, but got '%v'", arg)
			}
		}
	}

	if url == "" {
		printInfo("Please specify a stream URL")
		os.Exit(1)
	}

	printInfo("URL: %v", url)
	printInfo("Output directory: %v", dir)
	printInfo("Stopping after %v tracks", maxTracks)

	// Record the actual stream.
	for {
		record(url, dir)
		printInfo("Reconnecting due to previous error")
	}
}
