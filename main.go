package main

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"os"
	"path"
	"strconv"
	"time"

	"rsr/util"
	"rsr/vorbis"
)

var (
	colRed = "\033[31m"
	colYellow = "\033[33m"
	colReset = "\033[m"
)

func usage(arg0 string, exitStatus int) {
	fmt.Fprintln(os.Stderr, `Usage:
  ` + arg0 + ` [options...] <STREAM_URL>

Options:
  -dir <DIRECTORY>  --  Output directory (default: ".").

Output types:
  * <INFO>
  ` + colYellow + `! <WARNING>` + colReset + `
  ` + colRed + `! <ERROR>` + colReset)
	os.Exit(exitStatus)
}

func printInfo(f string, v ...interface{}) {
	fmt.Printf("* " + f + "\n", v...)
}

func printWarn(f string, v ...interface{}) {
	fmt.Fprintf(os.Stderr, colYellow + "! " + f + colReset + "\n", v...)
}

func printNonFatalErr(f string, v ...interface{}) {
	fmt.Fprintf(os.Stderr, colRed + "! " + f + colReset + "\n", v...)
}

func printErr(f string, v ...interface{}) {
	printNonFatalErr(f, v...)
	os.Exit(1)
}

func main() {
	var url string
	dir := "."

	if len(os.Args) < 2 {
		usage(os.Args[0], 1)
	}

	// Parse command line arguments.
	for i := 1; i < len(os.Args); i++ {
		arg := os.Args[i]
		if len(arg) >= 1 && arg[0] == '-' {
			switch(arg) {
			case "-dir":
				i++
				if i >= len(os.Args) {
					printErr("Expected string after flag '%v'", arg)
				}
				dir = os.Args[i]
			case "--help":
				usage(os.Args[0], 0)
			case "-h":
				usage(os.Args[0], 0)
			default:
				printErr("Unknown flag: '%v'", arg)
			}
		} else {
			if url == "" {
				url = arg
			} else {
				printErr("Expected flag, but got '%v'", arg)
			}
		}
	}

	if url == "" {
		printInfo("Please specify a stream URL")
		os.Exit(1)
	}

	printInfo("URL: %v", url)
	printInfo("Output directory: %v", dir)

	resp, err := http.Get(url)
	if err != nil {
		printErr("HTTP error: %v", err)
	}
	defer resp.Body.Close()

	contentType := resp.Header.Get("content-type")
	if contentType != "application/ogg" {
		printErr("Expected content type 'application/ogg', but got: '%v'", contentType)
	}

	waitReader := util.NewWaitReader(resp.Body)

	// The first track is always discarded, as streams usually don't start at
	// the exact end of a track, meaning it is almost certainly going to be
	// incomplete.
	discard := true

	printErrWhileRecording := func(f string, v ...interface{}) {
		printNonFatalErr(f, v...)
		printWarn("Unable to download track, skipping.")
		discard = true
	}

	for {
		var raw bytes.Buffer

		// Write all the bytes of the stream we'll read into a buffer to be able
		// save it to a file later.
		r := io.TeeReader(waitReader, &raw)

		d := vorbis.NewDecoder(r)

		// Read until metadata of the track. Keep in mind that the read bytes
		// are also copied to the buffer `raw` because of the tee reader.
		md, checksum, err := d.ReadMetadata()
		if err != nil {
			printErrWhileRecording("Error reading metadata: %v", err)
			printInfo("Retrying in 1s")
			time.Sleep(1 * time.Second)
			continue
		}

		// Create filename based on the extracted metadata
		var base string // File name without path or extension.
		artist, artistOk := md.FieldByName("Artist")
		title, titleOk := md.FieldByName("Title")
		if artistOk || titleOk {
			base = artist + " -- " + title
		} else {
			base = "Unknown_" + strconv.FormatInt(int64(checksum), 10)
		}
		base = strings.ReplaceAll(base, "/", "_") // Replace invalid characters

		if discard {
			printInfo("Going to discard incomplete track: %v", base)
		} else {
			printInfo("Recording track: %v", base)
		}

		filename := path.Join(dir, base+".ogg")

		// Determine the (extent of) the rest of the track by reading it, saving
		// the exact contents of the single track to our buffer `raw` using the
		// tee reader we set up previously.
		err = d.ReadRest()
		if err != nil {
			printErrWhileRecording("Error reading stream: %v", err)
			continue
		}

		// See declaration of `discard`.
		if !discard {
			err := os.WriteFile(filename, raw.Bytes(), 0666)
			if err != nil {
				printErrWhileRecording("Error writing file: %v", err)
				continue
			}
			printInfo("Saved track as: %v", filename)
		}

		discard = false
	}
}
