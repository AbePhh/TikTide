package transcode

import (
	"strings"
	"testing"
)

func TestBuildScaleFilterEnforcesEvenDimensions(t *testing.T) {
	t.Parallel()

	filter := buildScaleFilter(transcodeProfile{
		Resolution: "720p",
		Width:      1280,
		Height:     720,
		BitrateK:   1800,
	})

	if !strings.Contains(filter, "force_original_aspect_ratio=decrease") {
		t.Fatalf("expected force_original_aspect_ratio in filter, got: %s", filter)
	}
	if !strings.Contains(filter, "force_divisible_by=2") {
		t.Fatalf("expected force_divisible_by=2 in filter, got: %s", filter)
	}
	if filter != "scale=1280:720:force_original_aspect_ratio=decrease:force_divisible_by=2" {
		t.Fatalf("unexpected filter: %s", filter)
	}
}
