package photoessai

import (
	"bytes"
	"compress/zlib"
	"encoding/binary"
	"hash/crc32"
	"image"
	"image/color"
	"image/gif"
	"image/jpeg"
	"image/png"
	"math"
)

const (
	marqueurDebut       = 0xD8
	marqueurBalayage    = 0xDA
	marqueurFin         = 0xD9
	marqueurApp1        = 0xE1
	marqueurTrame       = 0xC0
	tailleEnTetePNG     = 8
	tailleBlocIHDR      = 25
	niveauGrisPNG       = 0
	profondeurPNG       = 8
	tagGPS              = 0x8825
	typeASCII           = 2
	typeLONG            = 4
	typeRATIONAL        = 5
	debutIFDGPS         = 26
	debutDonneesGPS     = 80
	tailleTripletGPS    = 24
	ScriptPiege         = "<script>alert('collector')</script>"
	CommentaireDeLieu   = "lieu: 48.8566N 2.3522E"
	enTeteHTMLPolyglote = "<html><body>" + ScriptPiege + "</body></html>"
)

var (
	latitudeFictive  = [3][2]uint32{{48, 1}, {51, 1}, {2959, 100}}
	longitudeFictive = [3][2]uint32{{2, 1}, {17, 1}, {4012, 100}}
	signaturePNG     = []byte("\x89PNG\r\n\x1a\n")
)

func degrade(largeur, hauteur int, opaque bool) *image.NRGBA {
	toile := image.NewNRGBA(image.Rect(0, 0, largeur, hauteur))
	alpha := uint8(255)
	if !opaque {
		alpha = 0
	}
	for x := range largeur {
		for y := range hauteur {
			toile.SetNRGBA(x, y, color.NRGBA{R: uint8(x * 7), G: uint8(y * 5), B: 120, A: alpha})
		}
	}
	return toile
}

func JPEG(largeur, hauteur int) []byte {
	var tampon bytes.Buffer
	if err := jpeg.Encode(&tampon, degrade(largeur, hauteur, true), &jpeg.Options{Quality: 90}); err != nil {
		panic(err)
	}
	return tampon.Bytes()
}

func PNG(largeur, hauteur int) []byte {
	var tampon bytes.Buffer
	if err := png.Encode(&tampon, degrade(largeur, hauteur, true)); err != nil {
		panic(err)
	}
	return tampon.Bytes()
}

func PNGTransparent(largeur, hauteur int) []byte {
	var tampon bytes.Buffer
	if err := png.Encode(&tampon, degrade(largeur, hauteur, false)); err != nil {
		panic(err)
	}
	return tampon.Bytes()
}

func GIF() []byte {
	var tampon bytes.Buffer
	if err := gif.Encode(&tampon, degrade(16, 16, true), nil); err != nil {
		panic(err)
	}
	return tampon.Bytes()
}

func WebP() []byte {
	charge := []byte{0x2f, 0x0f, 0xc0, 0x03, 0x00, 0x00, 0x00, 0x00}
	corps := append([]byte("WEBPVP8L"), le32(longueur32(charge))...)
	corps = append(corps, charge...)
	return append(append([]byte("RIFF"), le32(longueur32(corps))...), corps...)
}

func SVG() []byte {
	return []byte(`<?xml version="1.0"?><svg xmlns="http://www.w3.org/2000/svg" width="10" height="10" onload="alert(1)"><rect width="10" height="10"/></svg>`)
}

func FauxJPEG() []byte {
	return []byte("Ceci n'est pas une image, juste du texte nommé photo.jpg\n")
}

func PolyglotteJPEGHTML() []byte {
	return append([]byte{0xFF, marqueurDebut, 0xFF, 0xE0}, []byte(enTeteHTMLPolyglote)...)
}

func PolyglotteGIFZip() []byte {
	return append(GIF(), []byte("PK\x03\x04"+enTeteHTMLPolyglote)...)
}

func JPEGSuiviDUnScript() []byte {
	return append(JPEG(24, 16), []byte(enTeteHTMLPolyglote)...)
}

func JPEGTronque() []byte {
	complet := JPEG(64, 48)
	return complet[:len(complet)/2]
}

func JPEGAuxDimensionsAnnoncees(largeur, hauteur uint16) []byte {
	contenu := JPEG(16, 16)
	for i := 0; i+8 < len(contenu); i++ {
		if contenu[i] == 0xFF && contenu[i+1] == marqueurTrame {
			binary.BigEndian.PutUint16(contenu[i+5:], hauteur)
			binary.BigEndian.PutUint16(contenu[i+7:], largeur)
			return contenu
		}
	}
	panic("trame JPEG introuvable")
}

func longueur32(octets []byte) uint32 {
	n := len(octets)
	if n > math.MaxUint32 {
		panic("bloc trop long")
	}
	return uint32(n)
}

func longueur16(octets []byte) uint16 {
	n := len(octets)
	if n > math.MaxUint16-2 {
		panic("segment trop long")
	}
	return uint16(n)
}

func le32(v uint32) []byte {
	return binary.LittleEndian.AppendUint32(nil, v)
}

func le16(v uint16) []byte {
	return binary.LittleEndian.AppendUint16(nil, v)
}

func entree(tag, typ uint16, nombre, valeur uint32) []byte {
	return append(append(append(le16(tag), le16(typ)...), le32(nombre)...), le32(valeur)...)
}

func rationnels(triplet [3][2]uint32) []byte {
	var octets []byte
	for _, r := range triplet {
		octets = append(octets, le32(r[0])...)
		octets = append(octets, le32(r[1])...)
	}
	return octets
}

func LatitudeFictive() []byte {
	return rationnels(latitudeFictive)
}

func TIFFAvecGPS() []byte {
	tiff := []byte("II*\x00")
	tiff = append(tiff, le32(8)...)
	tiff = append(tiff, le16(1)...)
	tiff = append(tiff, entree(tagGPS, typeLONG, 1, debutIFDGPS)...)
	tiff = append(tiff, le32(0)...)
	tiff = append(tiff, le16(4)...)
	tiff = append(tiff, entree(1, typeASCII, 2, uint32('N'))...)
	tiff = append(tiff, entree(2, typeRATIONAL, 3, debutDonneesGPS)...)
	tiff = append(tiff, entree(3, typeASCII, 2, uint32('E'))...)
	tiff = append(tiff, entree(4, typeRATIONAL, 3, debutDonneesGPS+tailleTripletGPS)...)
	tiff = append(tiff, le32(0)...)
	tiff = append(tiff, rationnels(latitudeFictive)...)
	return append(tiff, rationnels(longitudeFictive)...)
}

func JPEGAvecGPS() []byte {
	originale := JPEG(48, 32)
	exif := append([]byte("Exif\x00\x00"), TIFFAvecGPS()...)
	segment := []byte{0xFF, marqueurApp1}
	segment = binary.BigEndian.AppendUint16(segment, longueur16(exif)+2)
	segment = append(segment, exif...)
	return append(append(append([]byte{}, originale[:2]...), segment...), originale[2:]...)
}

func bloc(nature string, donnees []byte) []byte {
	octets := binary.BigEndian.AppendUint32(nil, longueur32(donnees))
	corps := append([]byte(nature), donnees...)
	octets = append(octets, corps...)
	return binary.BigEndian.AppendUint32(octets, crc32.ChecksumIEEE(corps))
}

func PNGAvecGPS() []byte {
	originale := PNG(40, 30)
	fin := tailleEnTetePNG + tailleBlocIHDR
	ajouts := append(bloc("eXIf", TIFFAvecGPS()), bloc("tEXt", []byte("Comment\x00"+CommentaireDeLieu))...)
	return append(append(append([]byte{}, originale[:fin]...), ajouts...), originale[fin:]...)
}

func enTeteIHDR(largeur, hauteur uint32) []byte {
	donnees := binary.BigEndian.AppendUint32(nil, largeur)
	donnees = binary.BigEndian.AppendUint32(donnees, hauteur)
	donnees = append(donnees, profondeurPNG, niveauGrisPNG, 0, 0, 0)
	return append(append([]byte{}, signaturePNG...), bloc("IHDR", donnees)...)
}

func EnTetePNGGeant(largeur, hauteur uint32) []byte {
	return append(append(enTeteIHDR(largeur, hauteur), bloc("IDAT", nil)...), bloc("IEND", nil)...)
}

func BombePNG(largeur, hauteur uint32) []byte {
	var compresse bytes.Buffer
	ecrivain, err := zlib.NewWriterLevel(&compresse, zlib.BestCompression)
	if err != nil {
		panic(err)
	}
	ligne := make([]byte, largeur+1)
	for range hauteur {
		if _, err := ecrivain.Write(ligne); err != nil {
			panic(err)
		}
	}
	if err := ecrivain.Close(); err != nil {
		panic(err)
	}
	return append(append(enTeteIHDR(largeur, hauteur), bloc("IDAT", compresse.Bytes())...), bloc("IEND", nil)...)
}

func SegmentsAvantImage(contenu []byte) []byte {
	var marqueurs []byte
	for i := 2; i+3 < len(contenu) && contenu[i] == 0xFF; {
		marqueur := contenu[i+1]
		marqueurs = append(marqueurs, marqueur)
		if marqueur == marqueurBalayage || marqueur == marqueurFin {
			return marqueurs
		}
		i += 2 + int(binary.BigEndian.Uint16(contenu[i+2:]))
	}
	return marqueurs
}

func EstUnJPEG(contenu []byte) bool {
	return len(contenu) > 3 && contenu[0] == 0xFF && contenu[1] == marqueurDebut
}

func Tailles(contenu []byte) (int, int, error) {
	config, err := jpeg.DecodeConfig(bytes.NewReader(contenu))
	return config.Width, config.Height, err
}

const (
	LargeurRedressee   = 48
	HauteurRedressee   = 32
	tagOrientation     = 0x0112
	typeSHORT          = 3
	qualiteOrientation = 95
)

var CouleursDesQuarts = [4]color.NRGBA{
	{R: 220, G: 30, B: 30, A: 255},
	{R: 30, G: 200, B: 30, A: 255},
	{R: 30, G: 30, B: 220, A: 255},
	{R: 230, G: 230, B: 40, A: 255},
}

func quart(x, y, largeur, hauteur int) int {
	rang := 0
	if x >= largeur/2 {
		rang++
	}
	if y >= hauteur/2 {
		rang += 2
	}
	return rang
}

func pointStocke(orientation uint16, x, y, largeurStockee, hauteurStockee int) (int, int) {
	switch orientation {
	case 2:
		return largeurStockee - 1 - x, y
	case 3:
		return largeurStockee - 1 - x, hauteurStockee - 1 - y
	case 4:
		return x, hauteurStockee - 1 - y
	case 5:
		return y, x
	case 6:
		return y, hauteurStockee - 1 - x
	case 7:
		return largeurStockee - 1 - y, hauteurStockee - 1 - x
	case 8:
		return largeurStockee - 1 - y, x
	default:
		return x, y
	}
}

func ImageStockeePour(orientation uint16) *image.NRGBA {
	largeur, hauteur := LargeurRedressee, HauteurRedressee
	if orientation >= 5 && orientation <= 8 {
		largeur, hauteur = hauteur, largeur
	}
	stockee := image.NewNRGBA(image.Rect(0, 0, largeur, hauteur))
	for x := range LargeurRedressee {
		for y := range HauteurRedressee {
			sx, sy := pointStocke(orientation, x, y, largeur, hauteur)
			stockee.SetNRGBA(sx, sy, CouleursDesQuarts[quart(x, y, LargeurRedressee, HauteurRedressee)])
		}
	}
	return stockee
}

func TIFFOrientation(orientation uint16, ordre binary.AppendByteOrder) []byte {
	marque := "II"
	if ordre.String() == binary.BigEndian.String() {
		marque = "MM"
	}
	tiff := ordre.AppendUint16([]byte(marque), 42)
	tiff = ordre.AppendUint32(tiff, 8)
	tiff = ordre.AppendUint16(tiff, 1)
	tiff = ordre.AppendUint16(tiff, tagOrientation)
	tiff = ordre.AppendUint16(tiff, typeSHORT)
	tiff = ordre.AppendUint32(tiff, 1)
	tiff = ordre.AppendUint16(tiff, orientation)
	tiff = ordre.AppendUint16(tiff, 0)
	return ordre.AppendUint32(tiff, 0)
}

func JPEGAvecAPP1(contenu, charge []byte, longueurDeclaree uint16) []byte {
	segment := binary.BigEndian.AppendUint16([]byte{0xFF, marqueurApp1}, longueurDeclaree)
	segment = append(segment, charge...)
	return append(append(append([]byte{}, contenu[:2]...), segment...), contenu[2:]...)
}

func JPEGAvecEXIF(contenu, tiff []byte) []byte {
	charge := append([]byte("Exif\x00\x00"), tiff...)
	return JPEGAvecAPP1(contenu, charge, LongueurDeSegment(charge))
}

func JPEGOriente(orientation uint16, ordre binary.AppendByteOrder) []byte {
	var tampon bytes.Buffer
	if err := jpeg.Encode(&tampon, ImageStockeePour(orientation), &jpeg.Options{Quality: qualiteOrientation}); err != nil {
		panic(err)
	}
	return JPEGAvecEXIF(tampon.Bytes(), TIFFOrientation(orientation, ordre))
}

func PNGOriente(orientation uint16) []byte {
	var tampon bytes.Buffer
	if err := png.Encode(&tampon, ImageStockeePour(orientation)); err != nil {
		panic(err)
	}
	originale := tampon.Bytes()
	fin := tailleEnTetePNG + tailleBlocIHDR
	ajout := bloc("eXIf", TIFFOrientation(orientation, binary.LittleEndian))
	return append(append(append([]byte{}, originale[:fin]...), ajout...), originale[fin:]...)
}

func QuartsDe(contenu []byte) ([4]color.RGBA, image.Rectangle, error) {
	var quarts [4]color.RGBA
	decodee, err := jpeg.Decode(bytes.NewReader(contenu))
	if err != nil {
		return quarts, image.Rectangle{}, err
	}
	cadre := decodee.Bounds()
	largeur, hauteur := cadre.Dx(), cadre.Dy()
	centres := [4]image.Point{
		{X: largeur / 4, Y: hauteur / 4},
		{X: 3 * largeur / 4, Y: hauteur / 4},
		{X: largeur / 4, Y: 3 * hauteur / 4},
		{X: 3 * largeur / 4, Y: 3 * hauteur / 4},
	}
	for rang, centre := range centres {
		quarts[rang] = color.RGBAModel.Convert(decodee.At(cadre.Min.X+centre.X, cadre.Min.Y+centre.Y)).(color.RGBA)
	}
	return quarts, cadre, nil
}

func Proche(a color.RGBA, b color.NRGBA) bool {
	ecart := func(u, v uint8) int {
		if u > v {
			return int(u - v)
		}
		return int(v - u)
	}
	return ecart(a.R, b.R) < 40 && ecart(a.G, b.G) < 40 && ecart(a.B, b.B) < 40
}

func LongueurDeSegment(charge []byte) uint16 {
	return longueur16(charge) + 2
}
