package photo

import (
	"bytes"
	"encoding/binary"
	"image"
	"image/color"
)

type orientation int

const (
	upright orientation = iota + 1
	mirrored
	upsideDown
	flipped
	transposed
	turnedClockwise
	transversed
	turnedCounterClockwise
)

const (
	markerPrefix       = 0xFF
	startOfScan        = 0xDA
	endOfImage         = 0xD9
	app1Marker         = 0xE1
	orientationTag     = 0x0112
	shortType          = 3
	tiffMagic          = 42
	ifdEntrySize       = 12
	segmentHeaderSize  = 4
	jpegSignatureSize  = 2
	segmentLengthBytes = 2
	maxSegmentLength   = 0xFFFF
)

var exifHeader = []byte("Exif\x00\x00")

func uprightOrientation([]byte) orientation {
	return upright
}

func jpegOrientation(raw []byte) orientation {
	exif, found := exifSegment(raw)
	if !found {
		return upright
	}
	return tiffOrientation(exif)
}

func exifSegment(raw []byte) ([]byte, bool) {
	position := jpegSignatureSize
	for position+segmentHeaderSize <= len(raw) {
		if raw[position] != markerPrefix {
			return nil, false
		}
		marker := raw[position+1]
		if marker == markerPrefix {
			position++
			continue
		}
		if marker == startOfScan || marker == endOfImage {
			return nil, false
		}
		length := int(binary.BigEndian.Uint16(raw[position+2:]))
		end := position + segmentLengthBytes + length
		if length < segmentLengthBytes || end > len(raw) {
			return nil, false
		}
		payload := raw[position+segmentHeaderSize : end]
		if marker == app1Marker && bytes.HasPrefix(payload, exifHeader) {
			return payload[len(exifHeader):], true
		}
		position = end
	}
	return nil, false
}

func tiffOrientation(tiff []byte) orientation {
	order, found := byteOrderOf(tiff)
	if !found || len(tiff) < 8 || order.Uint16(tiff[2:]) != tiffMagic {
		return upright
	}
	offset := order.Uint32(tiff[4:])
	if offset > maxSegmentLength {
		return upright
	}
	directory := int(offset)
	if directory+2 > len(tiff) {
		return upright
	}
	entries := int(order.Uint16(tiff[directory:]))
	for index := range entries {
		entry := directory + 2 + index*ifdEntrySize
		if entry+ifdEntrySize > len(tiff) {
			return upright
		}
		if order.Uint16(tiff[entry:]) == orientationTag {
			return orientationEntry(order, tiff[entry:entry+ifdEntrySize])
		}
	}
	return upright
}

func byteOrderOf(tiff []byte) (binary.ByteOrder, bool) {
	switch {
	case bytes.HasPrefix(tiff, []byte("II")):
		return binary.LittleEndian, true
	case bytes.HasPrefix(tiff, []byte("MM")):
		return binary.BigEndian, true
	}
	return nil, false
}

func orientationEntry(order binary.ByteOrder, entry []byte) orientation {
	if order.Uint16(entry[2:]) != shortType || order.Uint32(entry[4:]) != 1 {
		return upright
	}
	value := orientation(order.Uint16(entry[8:]))
	if value < upright || value > turnedCounterClockwise {
		return upright
	}
	return value
}

func (o orientation) swapsSides() bool {
	return o >= transposed
}

func oriented(source image.Image, o orientation) image.Image {
	if o == upright {
		return source
	}
	return orientedView{source: source, orientation: o}
}

type orientedView struct {
	source      image.Image
	orientation orientation
}

func (v orientedView) ColorModel() color.Model {
	return v.source.ColorModel()
}

func (v orientedView) Bounds() image.Rectangle {
	size := v.source.Bounds().Size()
	if v.orientation.swapsSides() {
		return image.Rect(0, 0, size.Y, size.X)
	}
	return image.Rect(0, 0, size.X, size.Y)
}

func (v orientedView) At(x, y int) color.Color {
	bounds := v.source.Bounds()
	sourceX, sourceY := v.sourcePoint(x, y, bounds.Dx()-1, bounds.Dy()-1)
	return v.source.At(bounds.Min.X+sourceX, bounds.Min.Y+sourceY)
}

func (v orientedView) sourcePoint(x, y, lastX, lastY int) (int, int) {
	switch v.orientation {
	case mirrored:
		return lastX - x, y
	case upsideDown:
		return lastX - x, lastY - y
	case flipped:
		return x, lastY - y
	case transposed:
		return y, x
	case turnedClockwise:
		return y, lastY - x
	case transversed:
		return lastX - y, lastY - x
	case turnedCounterClockwise:
		return lastX - y, x
	case upright:
		return x, y
	}
	return x, y
}
