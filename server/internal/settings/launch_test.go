package settings

import (
	"strings"
	"testing"
)

func TestResolveLaunchModeDefaultsToPlanningOnly(t *testing.T) {
	cases := []struct {
		name          string
		live, dry     bool
		overlay       string
		wantPlanOnly  bool
		wantErrSubstr string
	}{
		{name: "default", wantPlanOnly: true},
		{name: "overlay live", overlay: LaunchLive},
		{name: "overlay dry-run", overlay: LaunchDryRun, wantPlanOnly: true},
		{name: "flag live beats overlay dry-run", live: true, overlay: LaunchDryRun},
		{name: "flag dry-run beats overlay live", dry: true, overlay: LaunchLive, wantPlanOnly: true},
		{name: "both flags", live: true, dry: true, wantErrSubstr: "cannot be used together"},
		{name: "junk in overlay", overlay: "sometimes", wantErrSubstr: "must be"},
	}
	for _, c := range cases {
		got, err := ResolveLaunchMode(c.live, c.dry, Overlay{Launch: c.overlay})
		if c.wantErrSubstr != "" {
			if err == nil || !strings.Contains(err.Error(), c.wantErrSubstr) {
				t.Errorf("%s: err = %v", c.name, err)
			}
			continue
		}
		if err != nil || got != c.wantPlanOnly {
			t.Errorf("%s: planOnly=%v err=%v, want %v", c.name, got, err, c.wantPlanOnly)
		}
	}
}

func TestLaunchModeSurvivesTheOverlayFileAndRejectsBadPatches(t *testing.T) {
	encoded := EncodeOverlay(Overlay{Launch: LaunchLive, SessionTimeout: "5m"})
	decoded, err := ParseOverlay(encoded)
	if err != nil || decoded.Launch != LaunchLive || decoded.SessionTimeout != "5m" {
		t.Fatalf("decoded = %+v, %v", decoded, err)
	}
	if _, err := DecodePatch([]byte(`{"launch":"yolo"}`)); err == nil {
		t.Fatal("a bad launch mode was accepted")
	}
	patch, err := DecodePatch([]byte(`{"launch":"live"}`))
	if err != nil {
		t.Fatal(err)
	}
	if merged := MergeOverlay(Overlay{}, patch); merged.Launch != LaunchLive {
		t.Fatalf("merged = %+v", merged)
	}
	cleared, _ := DecodePatch([]byte(`{"launch":""}`))
	if merged := MergeOverlay(Overlay{Launch: LaunchLive}, cleared); merged.Launch != "" {
		t.Fatalf("launch was not cleared: %+v", merged)
	}
}

func TestLaunchFieldExplainsTheRiskAndTheOverride(t *testing.T) {
	field := launchField(Overlay{Launch: LaunchLive}, "--live")
	if field.Source != SourceLocal || field.OverriddenBy != "fleet serve --live" || !strings.Contains(field.Warning, "without asking") {
		t.Fatalf("field = %+v", field)
	}
	if def := launchField(Overlay{}, ""); def.Source != SourceDefault || def.OverriddenBy != "" {
		t.Fatalf("default field = %+v", def)
	}
}
