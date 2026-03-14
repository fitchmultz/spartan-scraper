package aiauthoring

import (
	"encoding/base64"
	"strings"
	"testing"

	"github.com/fitchmultz/spartan-scraper/internal/apperrors"
	"github.com/fitchmultz/spartan-scraper/internal/extract"
)

func TestNormalizeDirectAIImagesNormalizesPayload(t *testing.T) {
	images, err := normalizeDirectAIImages([]extract.AIImageInput{{
		Data:     " Zm\nFr\tZQ== ",
		MimeType: " IMAGE/PNG ",
	}})
	if err != nil {
		t.Fatalf("normalizeDirectAIImages error: %v", err)
	}
	if len(images) != 1 {
		t.Fatalf("expected 1 image, got %d", len(images))
	}
	if images[0].MimeType != "image/png" {
		t.Fatalf("expected normalized mime type, got %q", images[0].MimeType)
	}
	if images[0].Data != "ZmFrZQ==" {
		t.Fatalf("expected compact base64 payload, got %q", images[0].Data)
	}
}

func TestNormalizeDirectAIImagesRejectsInvalidMimeType(t *testing.T) {
	_, err := normalizeDirectAIImages([]extract.AIImageInput{{
		Data:     "ZmFrZQ==",
		MimeType: "image/svg+xml",
	}})
	if err == nil {
		t.Fatal("expected validation error")
	}
	if !apperrors.IsKind(err, apperrors.KindValidation) {
		t.Fatalf("expected validation error kind, got %v", err)
	}
	if !strings.Contains(err.Error(), "mime_type") {
		t.Fatalf("expected mime_type error, got %v", err)
	}
}

func TestNormalizeDirectAIImagesRejectsInvalidBase64(t *testing.T) {
	_, err := normalizeDirectAIImages([]extract.AIImageInput{{
		Data:     "not-base64",
		MimeType: "image/png",
	}})
	if err == nil {
		t.Fatal("expected validation error")
	}
	if !apperrors.IsKind(err, apperrors.KindValidation) {
		t.Fatalf("expected validation error kind, got %v", err)
	}
	if !strings.Contains(err.Error(), "valid base64") {
		t.Fatalf("expected base64 validation error, got %v", err)
	}
}

func TestNormalizeDirectAIImagesRejectsTooManyImages(t *testing.T) {
	input := make([]extract.AIImageInput, 5)
	for i := range input {
		input[i] = extract.AIImageInput{Data: "ZmFrZQ==", MimeType: "image/png"}
	}
	_, err := normalizeDirectAIImages(input)
	if err == nil {
		t.Fatal("expected validation error")
	}
	if !strings.Contains(err.Error(), "at most 4 items") {
		t.Fatalf("expected image count validation error, got %v", err)
	}
}

func TestNormalizeDirectAIImagesAcceptsFourMaxSizedImages(t *testing.T) {
	payload := base64.StdEncoding.EncodeToString(make([]byte, maxDirectAIImageBytes))
	input := make([]extract.AIImageInput, maxDirectAIImages)
	for i := range input {
		input[i] = extract.AIImageInput{Data: payload, MimeType: "image/png"}
	}
	images, err := normalizeDirectAIImages(input)
	if err != nil {
		t.Fatalf("expected four 1 MiB images to be accepted, got %v", err)
	}
	if len(images) != maxDirectAIImages {
		t.Fatalf("expected %d normalized images, got %d", maxDirectAIImages, len(images))
	}
}

func TestNormalizeDirectAIImagesRejectsPerImageByteLimit(t *testing.T) {
	payload := base64.StdEncoding.EncodeToString(make([]byte, maxDirectAIImageBytes+1))
	_, err := normalizeDirectAIImages([]extract.AIImageInput{{
		Data:     payload,
		MimeType: "image/png",
	}})
	if err == nil {
		t.Fatal("expected validation error")
	}
	if !strings.Contains(err.Error(), "1 MiB per-image limit") {
		t.Fatalf("expected per-image limit error, got %v", err)
	}
}

func TestAppendAIImagesCopiesInputs(t *testing.T) {
	first := []extract.AIImageInput{{Data: "ZmFrZQ==", MimeType: "image/png"}}
	second := []extract.AIImageInput{{Data: "YmFy", MimeType: "image/jpeg"}}
	combined := appendAIImages(first, second)
	if len(combined) != 2 {
		t.Fatalf("expected 2 images, got %d", len(combined))
	}
	first[0].Data = "changed"
	if combined[0].Data != "ZmFrZQ==" {
		t.Fatalf("expected combined slice copy, got %q", combined[0].Data)
	}
}
