package mp3

import (
	"bytes"
	"encoding/binary"
	"errors"
	"hash/crc32"
	"html"
	"io"
	"net/http"
	"strconv"
	"strings"
)

var (
	ErrNoMetaint         = errors.New("mp3: key 'icy-metaint' not found in HTTP header")
	ErrCorruptedMetadata = errors.New("mp3: corrupted metadata")
	ErrNoStreamTitle     = errors.New("mp3: no 'StreamTitle' tag in metadata")
)

type Extractor struct {
	metaint        int64 // Distance between two metadata chunks
	hasStreamTitle bool
	streamTitle    string // Metadata tag determining the filename
}

func NewExtractor(respHdr http.Header) (*Extractor, error) {
	mi := respHdr.Get("icy-metaint")
	if mi == "" {
		return nil, ErrNoMetaint
	}
	miNum, _ := strconv.ParseInt(mi, 10, 64)
	return &Extractor{
		metaint: miNum,
	}, nil
}

func (d *Extractor) ReadBlock(r io.Reader, w io.Writer) (isFirst bool, err error) {
	var musicData bytes.Buffer

	// We want to write everything to the output, as well as musicData for
	// calculating the checksum.
	multi := io.MultiWriter(w, &musicData)

	// Read until the metadata chunk. The part that is read here is also what
	// contains the actual mp3 music data.
	io.CopyN(multi, r, d.metaint)

	// Read number of metadata blocks (blocks within this function are not what
	// is meant with `ReadBlock()`).
	var numBlocks uint8
	err = binary.Read(r, binary.LittleEndian, &numBlocks)

	// Whether this block is the beginning of a new track.
	var isBOF bool

	// Read metadata blocks.
	if numBlocks > 0 {
		// Metadata is only actually stored in the first metadata chunk
		// of a given file. Therefore, every metadata chunk with more than 1
		// block always marks the beginning of a file.
		isBOF = true

		// Each block is 16 bytes in size. Any excess bytes in the last block
		// are set to '\0', which is great because the `string()` conversion
		// function ignores null bytes. The whole string is escaped via HTML.
		// Metadata format: k0='v0';k1='v1';
		raw := make([]byte, numBlocks*16)
		if _, err := r.Read(raw); err != nil {
			return false, err
		}
		rawString := html.UnescapeString(string(raw))
		for _, data := range strings.Split(rawString, ";") {
			s := strings.Split(data, "=")
			if len(s) == 2 {
				if s[0] == "StreamTitle" {
					d.hasStreamTitle = true
					// Strip stream title's first and last character (single
					// quotes).
					t := s[1]
					if len(t) < 2 {
						return false, ErrCorruptedMetadata
					}
					t = t[1 : len(t)-1]
					if t == "Unknown" {
						// If there is no stream title, use format:
						// Unknown_<crc32 checksum>
						// Where the checksum is only that of the first block.
						sumStr := strconv.FormatInt(int64(crc32.ChecksumIEEE(musicData.Bytes())), 10)
						d.streamTitle = "Unknown_" + sumStr
					} else {
						d.streamTitle = t
					}
				}
			} else if len(s) != 1 {
				return false, ErrCorruptedMetadata
			}
		}
		if !d.hasStreamTitle {
			return false, ErrNoStreamTitle
		}
	}

	return isBOF, nil
}

func (d *Extractor) TryGetFilename() (filename string, hasFilename bool) {
	if !d.hasStreamTitle {
		return "", false
	}
	base := strings.ReplaceAll(d.streamTitle, "/", "_") // Replace invalid characters.
	return base + ".mp3", true
}
