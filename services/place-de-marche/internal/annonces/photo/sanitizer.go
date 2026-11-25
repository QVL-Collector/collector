package photo

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"image"
	"image/color"
	"image/jpeg"
	"image/png"
	"io"
	"time"

	"collector/services/place-de-marche/internal/annonces"
)

const (
	encodingQuality = 85
	slotWait        = 5 * time.Second
	opaqueAlpha     = 0xffff
	maxJPEGSide     = 0xffff
)

type Limits struct {
	MaxPixels         int
	ConcurrentDecodes int
}

type format struct {
	signature    []byte
	decodeConfig func(io.Reader) (image.Config, error)
	decode       func(io.Reader) (image.Image, error)
	orientation  func([]byte) orientation
}

var acceptedFormats = []format{
	{signature: []byte{0xFF, 0xD8, 0xFF}, decodeConfig: jpeg.DecodeConfig, decode: jpeg.Decode, orientation: jpegOrientation},
	{signature: []byte("\x89PNG\r\n\x1a\n"), decodeConfig: png.DecodeConfig, decode: png.Decode, orientation: uprightOrientation},
}

type Sanitizer struct {
	maxPixels int
	slots     chan struct{}
}

func NewSanitizer(limits Limits) (*Sanitizer, error) {
	if limits.MaxPixels < 1 || limits.ConcurrentDecodes < 1 {
		return nil, errors.New("plafond de pixels et décodages simultanés strictement positifs attendus")
	}
	return &Sanitizer{maxPixels: limits.MaxPixels, slots: make(chan struct{}, limits.ConcurrentDecodes)}, nil
}

func (s *Sanitizer) Sanitize(ctx context.Context, raw []byte) (annonces.EncodedPhoto, error) {
	accepted, found := formatOf(raw)
	if !found {
		return annonces.EncodedPhoto{}, annonces.ErrUnsupportedPhoto
	}
	header, err := accepted.decodeConfig(bytes.NewReader(raw))
	if err != nil {
		return annonces.EncodedPhoto{}, annonces.ErrUnsupportedPhoto
	}
	if !s.withinLimits(header) {
		return annonces.EncodedPhoto{}, annonces.ErrPhotoTooManyPixels
	}
	if err = s.acquire(ctx); err != nil {
		return annonces.EncodedPhoto{}, err
	}
	defer s.release()
	return reencode(accepted, raw)
}

func formatOf(raw []byte) (format, bool) {
	for _, candidate := range acceptedFormats {
		if bytes.HasPrefix(raw, candidate.signature) {
			return candidate, true
		}
	}
	return format{}, false
}

func (s *Sanitizer) withinLimits(header image.Config) bool {
	if header.Width < 1 || header.Height < 1 || header.Width > maxJPEGSide || header.Height > maxJPEGSide {
		return false
	}
	return int64(header.Width)*int64(header.Height) <= int64(s.maxPixels)
}

func (s *Sanitizer) acquire(ctx context.Context) error {
	waiting, cancel := context.WithTimeout(ctx, slotWait)
	defer cancel()
	select {
	case s.slots <- struct{}{}:
		return nil
	case <-waiting.Done():
		return annonces.ErrPhotoProcessingUnavailable
	}
}

func (s *Sanitizer) release() {
	<-s.slots
}

func reencode(accepted format, raw []byte) (annonces.EncodedPhoto, error) {
	decoded, err := accepted.decode(bytes.NewReader(raw))
	if err != nil {
		return annonces.EncodedPhoto{}, annonces.ErrUnsupportedPhoto
	}
	displayed := oriented(flattened(decoded), accepted.orientation(raw))
	var encoded bytes.Buffer
	if err = jpeg.Encode(&encoded, displayed, &jpeg.Options{Quality: encodingQuality}); err != nil {
		return annonces.EncodedPhoto{}, fmt.Errorf("ré-encodage de la photo : %w", err)
	}
	bounds := displayed.Bounds()
	return annonces.EncodedPhoto{
		Content:    encoded.Bytes(),
		Dimensions: annonces.Dimensions{Width: bounds.Dx(), Height: bounds.Dy()},
	}, nil
}

type opaqueImage interface {
	Opaque() bool
}

func flattened(decoded image.Image) image.Image {
	if opaque, known := decoded.(opaqueImage); known && opaque.Opaque() {
		return decoded
	}
	return onWhite{Image: decoded}
}

type onWhite struct {
	image.Image
}

func (o onWhite) ColorModel() color.Model {
	return color.RGBA64Model
}

func (o onWhite) At(x, y int) color.Color {
	r, g, b, a := o.Image.At(x, y).RGBA()
	white := opaqueAlpha - a
	return opaqueColor{r: r + white, g: g + white, b: b + white}
}

type opaqueColor struct {
	r, g, b uint32
}

func (c opaqueColor) RGBA() (r, g, b, a uint32) {
	return c.r, c.g, c.b, opaqueAlpha
}
