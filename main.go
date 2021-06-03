package main

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"os"
	"path"
	"strconv"

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

	// The first track is always discarded, as it is always going to be
	// incomplete.
	discard := true

	printErrWhileRecording := func(f string, v ...interface{}) {
		printNonFatalErr(f, v...)
		printWarn("Unable to download track, skipping.")
		discard = true
	}

	for {
		var raw bytes.Buffer

		r := io.TeeReader(waitReader, &raw)

		d := vorbis.NewDecoder(r)

		md, checksum, err := d.ReadMetadata()
		if err != nil {
			printErrWhileRecording("Error reading metadata: %v", err)
			continue
		}

		var base string // File name without path or extension.
		artist, artistOk := md.FieldByName("Artist")
		title, titleOk := md.FieldByName("Title")
		if artistOk || titleOk {
			base = artist + " -- " + title
		} else {
			base = "Unknown_" + strconv.FormatInt(int64(checksum), 10)
		}

		if discard {
			printInfo("Going to discard incomplete track: %v", base)
		} else {
			printInfo("Recording track: %v", base)
		}

		filename := path.Join(dir, base+".ogg")

		err = d.ReadRest()
		if err != nil {
			printErrWhileRecording("Error reading stream: %v", err)
			continue
		}

		if !discard {
			err := os.WriteFile(filename, raw.Bytes(), 0666)
			if err != nil {
				printErrWhileRecording("Error reading stream: %v", err)
				continue
			}
			printInfo("Saved track as: %v", filename)
		}

		discard = false
	}
}
