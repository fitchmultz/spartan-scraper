package aiauthoring

import (
	"encoding/base64"
	"strconv"
	"strings"

	"github.com/fitchmultz/spartan-scraper/internal/apperrors"
	"github.com/fitchmultz/spartan-scraper/internal/extract"
)

const (
	maxDirectAIImages          = 4
	maxDirectAIImageBytes      = 1 << 20
	maxTotalDirectAIImageBytes = 4 << 20
)

var allowedDirectAIImageMIMETypes = map[string]struct{}{
	"image/png":  {},
	"image/jpeg": {},
	"image/webp": {},
	"image/gif":  {},
}

func normalizeDirectAIImages(images []extract.AIImageInput) ([]extract.AIImageInput, error) {
	if len(images) == 0 {
		return nil, nil
	}
	if len(images) > maxDirectAIImages {
		return nil, apperrors.Validation("images supports at most 4 items")
	}

	out := make([]extract.AIImageInput, 0, len(images))
	totalBytes := 0
	for idx, image := range images {
		mimeType := strings.ToLower(strings.TrimSpace(image.MimeType))
		if _, ok := allowedDirectAIImageMIMETypes[mimeType]; !ok {
			return nil, apperrors.Validation("images[" + strconv.Itoa(idx) + "].mime_type must be one of image/png, image/jpeg, image/webp, or image/gif")
		}
		data := compactBase64(strings.TrimSpace(image.Data))
		if data == "" {
			return nil, apperrors.Validation("images[" + strconv.Itoa(idx) + "].data is required")
		}
		decoded, err := base64.StdEncoding.DecodeString(data)
		if err != nil {
			return nil, apperrors.Validation("images[" + strconv.Itoa(idx) + "].data must be valid base64")
		}
		if len(decoded) == 0 {
			return nil, apperrors.Validation("images[" + strconv.Itoa(idx) + "].data decoded to an empty payload")
		}
		if len(decoded) > maxDirectAIImageBytes {
			return nil, apperrors.Validation("images[" + strconv.Itoa(idx) + "] exceeds the 1 MiB per-image limit")
		}
		totalBytes += len(decoded)
		if totalBytes > maxTotalDirectAIImageBytes {
			return nil, apperrors.Validation("images exceed the 4 MiB total decoded size limit")
		}
		out = append(out, extract.AIImageInput{Data: data, MimeType: mimeType})
	}
	return out, nil
}

func appendAIImages(parts ...[]extract.AIImageInput) []extract.AIImageInput {
	count := 0
	for _, part := range parts {
		count += len(part)
	}
	if count == 0 {
		return nil
	}
	out := make([]extract.AIImageInput, 0, count)
	for _, part := range parts {
		for _, image := range part {
			out = append(out, extract.AIImageInput{Data: image.Data, MimeType: image.MimeType})
		}
	}
	return out
}

func compactBase64(value string) string {
	return strings.Map(func(r rune) rune {
		switch r {
		case '\n', '\r', '\t', ' ':
			return -1
		default:
			return r
		}
	}, value)
}
