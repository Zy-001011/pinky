package ui

import (
	"testing"

	mg "github.com/egoist/mygo/ui"

	"github.com/DIYgod/pinky/cmd/pinky/locales"
	"github.com/DIYgod/pinky/internal/driver"
	"github.com/DIYgod/pinky/ui/fakeapi"
)

// olderIPhoneApp opens Add phone › iPhone with an iPhone that was already
// in the list when the step began, so fresh cannot see it.
func olderIPhoneApp(t *testing.T) (*testApp, *mg.Tester, string) {
	t.Helper()
	ta := newTestAppWith(t, fakeapi.Options{Scenario: fakeapi.OnePhone, Speed: 50, HoldSetup: true})
	ta.api.ConnectIPhone()
	ta.waitFor(t, "the iPhone", func() bool {
		for _, d := range ta.devices {
			if d.Info.Platform == driver.IOS {
				return true
			}
		}
		return false
	})
	var id string
	for _, d := range ta.devices {
		if d.Info.Platform == driver.IOS {
			id = d.Info.ID
		}
	}
	ta.api.SetIPhone(id, driver.Info{State: driver.StateOnline, Backend: "coredevice"})
	ta.sync()
	ta.page = pageAddPhone
	ta.add.reset(ta.app)
	ta.add.step = "iphone"
	tt := mg.NewTester(ta.view, 1280, 820)
	tt.Frame()
	if ta.add.fresh(ta.app, driver.IOS) != nil {
		t.Fatal("fresh sees an iPhone that was already in the list")
	}
	return ta, tt, id
}

func openOlder(t *testing.T, tt *mg.Tester) {
	t.Helper()
	if err := clickInView(tt, locales.AddPhoneOlderIPhone()); err != nil {
		t.Fatal(err)
	}
	tt.Frame()
}

// TestAddOlderIPhoneHelper: choosing the helper no longer shows the M3
// note, asks the gateway to connect through WebDriverAgent once, and, when
// the helper is missing, says how to install it. Another setup step uses
// the steps' own words. Once the phone is online through the helper, Done
// opens it and leaves the older-iPhone page.
func TestAddOlderIPhoneHelper(t *testing.T) {
	opened := new(string)
	saved := openURL
	openURL = func(_ *mg.Context, u string, done func(error)) { *opened = u; done(nil) }
	t.Cleanup(func() { openURL = saved })

	ta, tt, id := olderIPhoneApp(t)
	openOlder(t, tt)
	if hasText(tt, locales.AddPhoneOlderNotYet()) {
		t.Fatalf("the M3 note shows before a choice: %q", tt.Texts())
	}
	if err := clickInView(tt, locales.AddPhoneOlderHelper()); err != nil {
		t.Fatal(err)
	}
	tt.Frame()
	if hasText(tt, locales.AddPhoneOlderNotYet()) {
		t.Fatalf("choosing the helper still shows the M3 note: %q", tt.Texts())
	}
	ta.waitFor(t, "WebDriverAgent chosen", func() bool {
		tt.Frame()
		got, err := ta.api.Connections(ta.ctx, id)
		return err == nil && got.Chosen == "wda" && ta.add.oldSet == id
	})
	for range 4 {
		tt.Frame()
	}
	if sets := ta.api.ConnectionSets(); len(sets) != 1 || sets[0] != [2]string{id, "wda"} {
		t.Fatalf("SetConnection calls: %v", sets)
	}
	ta.waitFor(t, "how to install the helper", func() bool {
		tt.Frame()
		return hasText(tt, locales.CheckHelperMissing()) && hasText(tt, locales.CheckHelperHow())
	})
	if hasText(tt, locales.AddPhoneOlderNotYet()) {
		t.Fatalf("the install guidance still includes the M3 note: %q", tt.Texts())
	}
	if err := clickInView(tt, locales.CheckHelperHow()); err != nil || *opened != helperDocs {
		t.Fatalf("how to install it: %v, opened %q", err, *opened)
	}

	// Some other step the phone still needs, in the words the steps use.
	ta.api.SetIPhone(id, driver.Info{State: driver.StateNeedsSetup, Setup: driver.SetupDeveloperMode})
	ta.waitFor(t, "Developer Mode", func() bool {
		tt.Frame()
		return hasText(tt, locales.CheckDeveloperModeTitle()) && hasText(tt, locales.AddPhoneIPhoneStepDevModeDetail())
	})
	if hasText(tt, locales.CheckHelperMissing()) || hasText(tt, locales.AddPhoneOlderNotYet()) {
		t.Fatalf("Developer Mode still talks about the helper or M3: %q", tt.Texts())
	}

	ta.api.SetIPhone(id, driver.Info{State: driver.StateOnline, Backend: "wda"})
	ready := locales.AddPhoneReady("iPhone 16")
	ta.waitFor(t, "the phone ready", func() bool { tt.Frame(); return hasText(tt, ready) })
	if err := clickInView(tt, locales.AddPhoneDone()); err != nil {
		t.Fatal(err)
	}
	if ta.page != pagePhones || ta.sel != id {
		t.Fatalf("Done: page %v, selected %q", ta.page, ta.sel)
	}
	if ta.add.old || ta.add.oldPick != "" || ta.add.oldSet != "" {
		t.Fatalf("Done left the older-iPhone path: %+v", ta.add)
	}
}

// TestAddOlderIPhoneHelperNeedsCable asks for a cable when no iPhone is
// connected, and does not choose a connection or show the M3 note.
func TestAddOlderIPhoneHelperNeedsCable(t *testing.T) {
	ta, tt := addIPhoneApp(t)
	tt.Frame()
	openOlder(t, tt)
	if err := clickInView(tt, locales.AddPhoneOlderHelper()); err != nil {
		t.Fatal(err)
	}
	tt.Frame()
	if !hasText(tt, locales.AddPhoneOlderPlugIn()) || hasText(tt, locales.AddPhoneOlderNotYet()) {
		t.Fatalf("no iPhone: %q", tt.Texts())
	}
	if ta.add.oldSet != "" || len(ta.api.ConnectionSets()) != 0 {
		t.Fatalf("SetConnection without a phone: oldSet %q, calls %v", ta.add.oldSet, ta.api.ConnectionSets())
	}
	// Back leaves the page and clears the choice.
	if err := tt.Click(locales.AddPhoneBack()); err != nil {
		t.Fatal(err)
	}
	if ta.add.old || ta.add.oldPick != "" || ta.add.oldSet != "" || ta.add.step != "choose" {
		t.Fatalf("Back left %+v", ta.add)
	}
}

// TestAddOlderIPhoneMirror does not pretend iPhone Mirroring connected.
// Off a Mac it says mirroring is Mac-only and points at the helper; on a
// Mac it says this screen does not set mirroring up.
func TestAddOlderIPhoneMirror(t *testing.T) {
	defer func(os string) { guideOS = os }(guideOS)
	ta, tt, id := olderIPhoneApp(t)
	openOlder(t, tt)
	if err := clickInView(tt, locales.AddPhoneOlderMirroring()); err != nil {
		t.Fatal(err)
	}
	tt.Frame()
	guideOS = "linux"
	tt.Frame()
	if !hasText(tt, locales.AddPhoneOlderMirrorMacOnly()) || !hasText(tt, locales.AddPhoneOlderHelper()) {
		t.Fatalf("linux: %q", tt.Texts())
	}
	if hasText(tt, locales.AddPhoneOlderNotYet()) || hasText(tt, locales.AddPhoneReady("iPhone 16")) {
		t.Fatalf("linux pretended mirroring worked: %q", tt.Texts())
	}
	guideOS = "windows"
	tt.Frame()
	if !hasText(tt, locales.AddPhoneOlderMirrorMacOnly()) {
		t.Fatalf("windows: %q", tt.Texts())
	}
	guideOS = "darwin"
	tt.Frame()
	if !hasText(tt, locales.AddPhoneOlderNotYet()) || hasText(tt, locales.AddPhoneOlderMirrorMacOnly()) {
		t.Fatalf("mac: %q", tt.Texts())
	}
	if hasText(tt, locales.AddPhoneReady("iPhone 16")) || ta.add.oldSet != "" {
		t.Fatalf("mac chose a connection: oldSet %q, %q", ta.add.oldSet, tt.Texts())
	}
	got, err := ta.api.Connections(ta.ctx, id)
	if err != nil || got.Chosen != "" || len(ta.api.ConnectionSets()) != 0 {
		t.Fatalf("mirroring called SetConnection: chosen %q err %v calls %v", got.Chosen, err, ta.api.ConnectionSets())
	}
	// Leaving Add phone from the sidebar drops the choice.
	if err := tt.Click("Pixel 9"); err != nil {
		t.Fatal(err)
	}
	if ta.page == pageAddPhone || ta.add.old || ta.add.oldPick != "" || ta.add.oldSet != "" {
		t.Fatalf("leaving kept the older path: page %v %+v", ta.page, ta.add)
	}
}
