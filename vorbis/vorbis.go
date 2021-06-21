package vorbis

import (
	"bytes"
	"errors"
	"io"
	"strconv"
	"strings"
)

var (
	ErrNoHeaderSegment = errors.New("vorbis: no header segment")
)

type Extractor struct {
	hasMetadata bool
	metadata    *VorbisComment // Used for filename.
	checksum    uint32         // Used for an alternate filename when there's no metadata.
}

func NewExtractor() (*Extractor, error) {
	return new(Extractor), nil
}

func (d *Extractor) ReadBlock(reader io.Reader, w io.Writer) (isFirst bool, err error) {
	// Everything we read here is part of the music data so we can just use a
	// tee reader.
	r := io.TeeReader(reader, w)

	// Decode page.
	page, err := OggDecode(r)
	if err != nil {
		return false, err
	}

	// We need to be able to access `page.Segments[0]`.
	if len(page.Segments) == 0 {
		return false, ErrNoHeaderSegment
	}

	// Decode Vorbis header, stored in `page.Segments[0]`.
	hdr, err := VorbisHeaderDecode(bytes.NewBuffer(page.Segments[0]))
	if err != nil {
		return false, err
	}

	// Extract potential metadata.
	if hdr.PackType == PackTypeComment {
		d.hasMetadata = true
		d.metadata = hdr.Comment
		d.checksum = page.Header.Checksum
	}

	// Return true for isFirst if this block is the beginning of a new file.
	return (page.Header.HeaderType & FHeaderTypeBOS) > 0, nil
}

func (d *Extractor) TryGetFilename() (filename string, hasFilename bool) {
	if !d.hasMetadata {
		return "", false
	}
	d.hasMetadata = false

	// Use relevant metadata to create a filename.
	var base string // Filename without extension.
	artist, artistOk := d.metadata.FieldByName("Artist")
	title, titleOk := d.metadata.FieldByName("Title")
	if artistOk || titleOk {
		if !artistOk {
			artist = "Unknown"
		} else if !titleOk {
			title = "Unknown"
		}
		base = artist + " -- " + title
	} else {
		base = "Unknown_" + strconv.FormatInt(int64(d.checksum), 10)
	}
	base = strings.ReplaceAll(base, "/", "_") // Replace invalid characters.

	return base + ".ogg", true
}
