// Minimal 'Vorbis comment' metadata reader oriented around
// https://xiph.org/vorbis/doc/Vorbis_I_spec.html and its reference
// implementation written in C.
package vorbis

import (
	"encoding/binary"
	"errors"
	"io"
	"strings"
)

var (
	ErrVorbisHeaderType           = errors.New("vorbis: header not Vorbis")
	ErrVorbisInvalidCommentFormat = errors.New("vorbis: invalid Vorbis comment")
)

type VorbisCommentField struct {
	// According to the spec, the key capitalization doesn't matter, which is
	// why we're using uppercase letters only in this implementation
	// (see `VorbisCommentDecode()`).
	Key string
	// The value has some restrictions for what characters it can contain, but
	// we're ignoring them for now.
	Val string
}

// Track metadata represented by a Vorbis comment. The fields should be
// self-explanatory.
type VorbisComment struct {
	Vendor string
	Fields []VorbisCommentField
}

// Attempts to decode a Vorbis comment, leaving the reader `r` right past the
// data it decoded.
func VorbisCommentDecode(r io.Reader) (VorbisComment, error) {
	var ret VorbisComment

	// In Vorbis comment, strings are always preceded by a 32-bit length
	// specifier.
	getNextString := func() ([]byte, error) {
		var sz uint32
		err := binary.Read(r, binary.LittleEndian, &sz)
		if err != nil {
			return nil, err
		}

		content := make([]byte, sz)
		_, err = r.Read(content)
		if err != nil {
			return nil, err
		}

		return content, nil
	}

	content, err := getNextString()
	if err != nil {
		return ret, err
	}
	ret.Vendor = string(content)

	var numCommentFields uint32
	err = binary.Read(r, binary.LittleEndian, &numCommentFields)
	if err != nil {
		return ret, err
	}

	for i := 0; i < int(numCommentFields); i++ {
		content, err := getNextString()
		if err != nil {
			return ret, err
		}

		splits := strings.Split(string(content), "=")
		if len(splits) != 2 {
			return ret, ErrVorbisInvalidCommentFormat
		}

		var newField VorbisCommentField

		newField.Key = strings.ToUpper(splits[0])
		newField.Val = splits[1]

		ret.Fields = append(ret.Fields, newField)
	}
	return ret, nil
}

// Field names are searched case insensitively, as specified in the spec.
// `found` is set to false if the field doesn't exist.
func (c *VorbisComment) FieldByName(name string) (val string, found bool) {
	// All field names are stored as upper case strings in this implementation.
	// That is why we only need to transform the search query string to
	// uppercase.
	upperName := strings.ToUpper(name)
	// Linearly search through field names.
	for _, v := range c.Fields {
		if v.Key == upperName {
			return v.Val, true
		}
	}
	return "", false
}

var (
	PackTypeInfo    = uint8(0x1)
	PackTypeComment = uint8(0x3) // Comment is the only one we really care about here.
	PackTypeBooks   = uint8(0x5)
)

type VorbisHeader struct {
	PackType uint8
	Comment  *VorbisComment
}

func VorbisHeaderDecode(r io.Reader) (VorbisHeader, error) {
	var ret VorbisHeader

	err := binary.Read(r, binary.LittleEndian, &ret.PackType)
	if err != nil {
		return ret, err
	}

	switch ret.PackType {
	case PackTypeComment:
		buf := make([]byte, 6)
		_, err = r.Read(buf)
		if err != nil {
			return ret, err
		}
		if string(buf) != "vorbis" {
			return ret, ErrVorbisHeaderType
		}

		comment, err := VorbisCommentDecode(r)
		if err != nil {
			return ret, err
		}
		ret.Comment = &comment
	}
	return ret, nil
}
