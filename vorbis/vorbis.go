package vorbis

import (
	"bytes"
	"errors"
	"io"
)

var (
	ErrNoHeaderSegment               = errors.New("no header segment")
	ErrNoMetadata                    = errors.New("no metadata found")
	ErrCallReadRestAfterReadMetadata = errors.New("please call vorbis.Decoder.ReadRest() after having called vorbis.Decoder.ReadMetadata()")
	ErrReadMetadataCalledTwice       = errors.New("cannot call vorbis.Decoder.ReadMetadata() twice on the same file")
)

type Decoder struct {
	r           io.Reader
	hasMetadata bool
}

func NewDecoder(r io.Reader) *Decoder {
	return &Decoder{
		r: r,
	}
}

func (d *Decoder) readPage() (page OggPage, hdr VorbisHeader, err error) {
	// Decode page.
	page, err = OggDecode(d.r)
	if err != nil {
		return page, hdr, err
	}

	// We need to be able to access `page.Segments[0]`.
	if len(page.Segments) == 0 {
		return page, hdr, ErrNoHeaderSegment
	}

	// Decode Vorbis header, stored in `page.Segments[0]`.
	hdr, err = VorbisHeaderDecode(bytes.NewBuffer(page.Segments[0]))
	if err != nil {
		return page, hdr, err
	}

	return page, hdr, nil
}

// Reads the Ogg/Vorbis file until it finds its metadata. Leaves the reader
// right after the end of the metadata. `crc32Sum` gives the crc32 checksum
// of the page containing the metadata. It is equivalent to the page checksum
// used in the Ogg container. Since the page contains more than just metadata,
// the checksum can usually be used as a unique identifier.
func (d *Decoder) ReadMetadata() (metadata *VorbisComment, crc32Sum uint32, err error) {
	if d.hasMetadata {
		return nil, 0, ErrReadMetadataCalledTwice
	}

	for {
		page, hdr, err := d.readPage()
		if err != nil {
			return nil, 0, err
		}

		if (page.Header.HeaderType & FHeaderTypeEOS) > 0 {
			// End of stream
			return nil, 0, ErrNoMetadata
		}

		if hdr.PackType == PackTypeComment {
			d.hasMetadata = true
			return hdr.Comment, page.Header.Checksum, nil
		}
	}
}

// Must to be called after `ReadMetadata()`. Reads the rest of the Ogg/Vorbis
// file, leaving the reader right after the end of the Ogg/Vorbis file.
func (d *Decoder) ReadRest() error {
	if !d.hasMetadata {
		return ErrCallReadRestAfterReadMetadata
	}

	for {
		page, _, err := d.readPage()
		if err != nil {
			return err
		}

		if (page.Header.HeaderType & FHeaderTypeEOS) > 0 {
			// End of stream
			break
		}
	}

	return nil
}
