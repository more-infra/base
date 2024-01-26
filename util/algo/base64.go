package algoutil

import "encoding/base64"

func Base64StdEncode(data []byte) string {
	return base64Encode(base64.StdEncoding, data)
}

func Base64StdDecode(base64Str string) ([]byte, error) {
	return base64Decode(base64.StdEncoding, base64Str)
}

func Base64URLEncode(data []byte) string {
	return base64Encode(base64.URLEncoding, data)
}

func Base64URLDecode(base64Str string) ([]byte, error) {
	return base64Decode(base64.URLEncoding, base64Str)
}

func Base64RawStdEncode(data []byte) string {
	return base64Encode(base64.RawStdEncoding, data)
}

func Base64RawStdDecode(base64Str string) ([]byte, error) {
	return base64Decode(base64.RawStdEncoding, base64Str)
}

func Base64RawURLEncode(data []byte) string {
	return base64Encode(base64.RawURLEncoding, data)
}

func Base64RawURLDecode(base64Str string) ([]byte, error) {
	return base64Decode(base64.RawURLEncoding, base64Str)
}

func base64Encode(encoding *base64.Encoding, data []byte) string {
	buf := make([]byte, encoding.EncodedLen(len(data)))
	encoding.Encode(buf, data)
	return string(buf)
}

func base64Decode(encoding *base64.Encoding, base64Str string) ([]byte, error) {
	data := []byte(base64Str)
	buf := make([]byte, encoding.DecodedLen(len(data)))
	if _, err := encoding.Decode(buf, data); err != nil {
		return nil, err
	}
	return buf, nil
}
