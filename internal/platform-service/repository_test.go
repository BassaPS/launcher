package platformService

import "testing"

func TestReleaseRepositoryMigrationBoundary(t *testing.T) {
	for _, repo := range []string{"launcher", "bassaPSLauncher"} {
		base := "https://github.com/BassaPS/" + repo + "/releases"
		for _, suffix := range []string{"/tag/v1.2.0", "/download/v1.2.0/BASSAPSLauncher-1.2.0-windows-x64.zip"} {
			if !trustedURL(base+suffix, suffix) {
				t.Fatalf("rejected supported repository: %s%s", base, suffix)
			}
		}
	}
	for _, raw := range []string{
		"https://github.com/BassaPS/launcher-evil/releases/tag/v1.2.0",
		"https://github.com/another/launcher/releases/tag/v1.2.0",
		"https://github.com.evil.invalid/BassaPS/launcher/releases/tag/v1.2.0",
		"https://user@github.com/BassaPS/launcher/releases/tag/v1.2.0",
		"https://github.com/BassaPS/launcher/releases/tag/v1.2.0?redirect=1",
		"https://github.com/BassaPS/launcher/releases/tag/v1.2.0#fragment",
		"https://github.com/BassaPS/launcher/releases/../tag/v1.2.0",
		"http://github.com/BassaPS/launcher/releases/tag/v1.2.0",
	} {
		if trustedURL(raw, "/tag/v1.2.0") {
			t.Fatalf("accepted untrusted release: %s", raw)
		}
	}
}
