package hive

import (
	"bytes"
	"fmt"
	"image"
	"image/color"
	"image/gif"
	"math"
	"strings"
	"testing"
	"time"
	"unicode/utf8"
)

// Every letter half as wide as the type is tall, so what a layout comes to can be
// worked out by hand.
type uiFake struct{ copied string }

func (d *uiFake) measure(f uiFont, s string) float64 { return float64(utf8.RuneCountInString(s)) * f.px / 2 }
func (d *uiFake) tall(f uiFont) float64                 { return f.px * 1.25 }
func (d *uiFake) average(f uiFont) float64              { return f.px / 2 }
func (d *uiFake) text(uiFont, int, int, string, uiRGBA, uiClip) {}
func (d *uiFake) clipboard() string                     { return d.copied }
func (d *uiFake) setClipboard(s string)                 { d.copied = s }
func (d *uiFake) openLink(string)                       {}
func (d *uiFake) datePattern() string                   { return "dd/MM/yyyy" }

// A window with a mailbox of its own, which nothing reads but the test.
func uiLaidOut(v UiView, w, h float64) (*uiNative, *mailbox) {
	n := uiNewNative()
	n.dev = &uiFake{}
	box := newMailbox(nil, 0)
	n.addr = SyslinkAddress{Name: atomNone, Id: box.id, box: box}
	n.resize(w, h, 1)
	uiShowAgain(n, v)
	return n, box
}

func uiShowAgain(n *uiNative, v UiView) {
	n.publish(v)
	n.adopt()
	n.layout()
}

func uiClick(n *uiNative, x, y float64) {
	n.move(x, y)
	n.down(x, y)
	n.up(x, y)
}

// Everything posted so far, in order, and nothing twice.
func uiPosted(box *mailbox) string {
	box.mu.Lock()
	defer box.mu.Unlock()
	var out []string
	for _, d := range box.queue {
		out = append(out, fmt.Sprint(d.value))
	}
	box.queue = nil
	return strings.Join(out, ",")
}

func uiSaid(s string) any { return s }

func uiNear(a, b float64) bool { return math.Abs(a-b) < 0.01 }

func TestASpacerTakesWhatTheRowLeaves(t *testing.T) {
	n, _ := uiLaidOut(UiRow(nil, []UiView{UiText(nil, "abcd"), UiSpacer(), UiButton(nil, "Go")}), 400, 300)
	button := n.boxes["r/0/2"]
	if !uiNear(button.w, 44) || !uiNear(button.x, 356) {
		t.Errorf("the button is %.2f wide at %.2f, wanted 44 at 356", button.w, button.x)
	}
}

func TestATableSharesItsWidthByWhatEachColumnHolds(t *testing.T) {
	rows := Table{{"Name", "City"}, {"Ana", "Recife"}, {"Bo", "Porto Alegre do Norte"}}
	n, _ := uiLaidOut(UiColumn([]UiAttr{UiAttrWidth(300)}, []UiView{UiTable(nil, rows)}), 800, 600)
	g := n.boxes["r/0/0"].grid
	if !uiNear(g.cols[0]+g.cols[1], 300) || !uiNear(g.cols[0], 48+85*48.0/215) {
		t.Errorf("columns %v, wanted the 85 left over shared as 48 to 167", g.cols)
	}
	n, _ = uiLaidOut(UiColumn([]UiAttr{UiAttrWidth(100)}, []UiView{UiTable(nil, rows)}), 800, 600)
	g = n.boxes["r/0/0"].grid
	if !uiNear(g.cols[0], 27+21*46.0/161) {
		t.Errorf("columns %v, wanted each between its least and its most", g.cols)
	}
	if lines := len(g.wrapped[2][1]); lines < 2 || !uiNear(g.heights[2], float64(lines)*21+17) {
		t.Errorf("a long cell wrapped to %d lines and a row %.2f tall", lines, g.heights[2])
	}
}

func TestATableReportsTheRowAndTheColumn(t *testing.T) {
	rows := Table{{"Name", "City"}, {"Ana", "Recife"}, {"Bo", "Olinda"}}
	pick := UiAttrOnPick(func(i int) any { return fmt.Sprint("row ", i) })
	sort := UiAttrOnSort(func(i int) any { return fmt.Sprint("column ", i) })
	n, box := uiLaidOut(UiTable([]UiAttr{pick, sort}, rows), 400, 300)
	g := n.boxes["r/0"].grid
	uiClick(n, g.cols[0]+5, 5)
	uiClick(n, 5, g.heights[0]+g.heights[1]+5)
	if got := uiPosted(box); got != "column 1,row 1" {
		t.Errorf("heard %s", got)
	}
}

func TestAnImageKeepsItsShape(t *testing.T) {
	n := uiNewNative()
	n.dev = &uiFake{}
	n.resize(400, 300, 1)
	n.pics["p"] = &uiPicture{done: true, w: 64, h: 40, pix: make([]uint8, 64*40*4)}
	n.pics["gone"] = &uiPicture{done: true}
	uiShowAgain(n, UiColumn(nil, []UiView{
		UiImage(nil, "p", ""),
		UiImage([]UiAttr{UiAttrWidth(96)}, "p", ""),
		UiImage(nil, "gone", "x"),
	}))
	for path, want := range map[string][2]float64{"r/0/0": {400, 250}, "r/0/1": {96, 60}, "r/0/2": {400, 21}} {
		if b := n.boxes[path]; !uiNear(b.w, want[0]) || !uiNear(b.h, want[1]) {
			t.Errorf("%s is %.2f by %.2f, wanted %.0f by %.0f", path, b.w, b.h, want[0], want[1])
		}
	}
}

func TestADialogIsCentredAndItsBackdropDismissesIt(t *testing.T) {
	v := UiColumn(nil, []UiView{
		UiText(nil, "under"),
		UiOverlay([]UiAttr{UiAttrOnDismiss("dismissed")}, UiText(nil, "hi")),
	})
	n, box := uiLaidOut(v, 400, 300)
	if len(n.layers) != 1 {
		t.Fatalf("%d layers, wanted the one", len(n.layers))
	}
	p := n.layers[0].panel
	if !uiNear(p.x, 193) || !uiNear(p.y, 139.5) || !uiNear(p.w, 14) {
		t.Errorf("the panel is %.2f wide at %.2f, %.2f", p.w, p.x, p.y)
	}
	uiClick(n, 195, 145)
	if got := uiPosted(box); got != "" {
		t.Errorf("a click inside the dialog said %s", got)
	}
	uiClick(n, 10, 10)
	if got := uiPosted(box); got != "dismissed" {
		t.Errorf("a click on the backdrop said %q", got)
	}
}

func TestAPinnedPanelLetsClicksThroughAroundIt(t *testing.T) {
	v := UiColumn(nil, []UiView{
		UiButton([]UiAttr{UiAttrOn("under")}, "under"),
		UiOverlay([]UiAttr{UiAttrAlign("End"), UiAttrJustify("End")}, UiButton([]UiAttr{UiAttrOn("over")}, "x")),
	})
	n, box := uiLaidOut(v, 400, 300)
	p := n.layers[0].panel
	if !uiNear(p.x, 351) || !uiNear(p.y, 253) {
		t.Errorf("the panel is at %.2f, %.2f, wanted its corner 12px in from the window's", p.x, p.y)
	}
	uiClick(n, 5, 5)
	uiClick(n, 360, 260)
	if got := uiPosted(box); got != "under,over" {
		t.Errorf("heard %s", got)
	}
}

func TestATextareaMovesALineAtATimeAsItWraps(t *testing.T) {
	n, _ := uiLaidOut(UiColumn(nil, []UiView{UiTextarea([]UiAttr{UiAttrWidth(100)}, "aaa bbb ccc ddd")}), 400, 300)
	b := n.boxes["r/0/0"]
	n.tab(false)
	f := n.fields[b.path]
	if spans := n.flow(b, f); len(spans) != 2 || spans[1] != (uiSpan{12, 15}) {
		t.Fatalf("wrapped as %v", spans)
	}
	f.caret, f.anchor = 5, 5
	n.key(uiKeyDown, false, false)
	if f.caret != 15 {
		t.Errorf("down went to %d, wanted the end of the short line", f.caret)
	}
	n.key(uiKeyUp, false, false)
	if f.caret != 5 {
		t.Errorf("up went to %d, wanted back to the column it left", f.caret)
	}
	n.key(uiKeyEnter, false, false)
	if string(f.text) != "aaa b\nbb ccc ddd" {
		t.Errorf("enter made %q", string(f.text))
	}
}

func TestANumberFieldReportsOnlyNumbers(t *testing.T) {
	n, box := uiLaidOut(UiInput([]UiAttr{UiAttrKind("Number"), UiAttrOnInput(uiSaid)}, ""), 400, 300)
	n.tab(false)
	n.typed('1')
	n.typed('x')
	n.typed('e')
	n.key(uiKeyBack, false, false)
	n.key(uiKeyUp, false, false)
	if got := uiPosted(box); got != "1,,1,2" {
		t.Errorf("heard %s", got)
	}
	for text, valid := range map[string]bool{"1": true, "-2.5e3": true, ".5": true, "1.": false, "e3": false, "": false, "+1": false} {
		if uiValidNumber(text) != valid {
			t.Errorf("%q valid: wanted %v", text, valid)
		}
	}
}

func TestADateIsTypedASegmentAtATime(t *testing.T) {
	n, box := uiLaidOut(UiInput([]UiAttr{UiAttrKind("Date"), UiAttrOnInput(uiSaid)}, ""), 400, 300)
	n.tab(false)
	for _, r := range "30092026" {
		n.typed(r)
	}
	n.key(uiKeyBack, false, false)
	n.key(uiKeyLeft, false, false)
	n.key(uiKeyUp, false, false)
	// A year is a value from its first digit, as Chrome reports it.
	if got := uiPosted(box); !strings.HasSuffix(got, "0202-09-30,2026-09-30,") {
		t.Errorf("heard %s", got)
	}
	if d := n.dates["r/0"]; d.parts != [3]int{0, 10, 30} {
		t.Errorf("the date holds %v", d.parts)
	}
}

func TestAClosedSelectChoosesWithTheArrowsAndItsLetters(t *testing.T) {
	n, box := uiLaidOut(UiSelect([]UiAttr{UiAttrOnChoose(uiSaid)}, []string{"a", "b", "c"}, "a"), 400, 300)
	n.tab(false)
	n.key(uiKeyDown, false, false)
	n.key(uiKeyDown, false, false)
	n.key(uiKeyDown, false, false)
	n.typed('a')
	if got := uiPosted(box); got != "b,c,a" {
		t.Errorf("heard %s", got)
	}
}

func TestACheckboxShowsItsTickUntilTheProgramDrawsAgain(t *testing.T) {
	v := UiCheckbox([]UiAttr{UiAttrOnToggle(func(on bool) any { return on })}, "done", false)
	n, box := uiLaidOut(v, 400, 300)
	b := n.boxes["r/0"]
	uiClick(n, b.x+5, b.y+5)
	if got := uiPosted(box); got != "true" || !n.checked(n.boxes["r/0"]) {
		t.Errorf("heard %s", got)
	}
	uiShowAgain(n, v)
	if n.checked(n.boxes["r/0"]) {
		t.Errorf("still ticked after a view that says it is not")
	}
}

func TestASliderFollowsTheKeys(t *testing.T) {
	n, box := uiLaidOut(UiInput([]UiAttr{UiAttrKind("Range"), UiAttrOnInput(uiSaid)}, "30"), 400, 300)
	n.tab(false)
	n.key(uiKeyEnd, false, false)
	n.key(uiKeyLeft, false, false)
	n.key(uiKeyPageDown, false, false)
	if got := uiPosted(box); got != "100,99,89" {
		t.Errorf("heard %s", got)
	}
}

func TestTabSkipsWhatIsDisabled(t *testing.T) {
	n, _ := uiLaidOut(UiColumn(nil, []UiView{
		UiButton([]UiAttr{UiAttrDisabled(true)}, "no"),
		UiInput(nil, ""),
		UiButton(nil, "yes"),
	}), 400, 300)
	n.tab(false)
	first := n.focus
	n.tab(false)
	if first != "r/0/1" || n.focus != "r/0/2" {
		t.Errorf("tab went to %s then %s", first, n.focus)
	}
}

func TestARasterCoversWhatIsInsideAndHalfOfAnEdge(t *testing.T) {
	var r uiRaster
	cover := func(pts []float64, x, y int) uint8 {
		r.reset(10, 10)
		r.polygon(pts)
		return r.mask(nil)[y*10+x]
	}
	cases := []struct {
		pts  []float64
		x, y int
		want uint8
	}{
		{[]float64{2, 2, 6, 2, 6, 6, 2, 6}, 3, 3, 255},
		{[]float64{2, 2, 6, 2, 6, 6, 2, 6}, 7, 7, 0},
		{[]float64{2.5, 2, 6, 2, 6, 6, 2.5, 6}, 2, 3, 128},
		{[]float64{-5, 2, 6, 2, 6, 6, -5, 6}, 0, 3, 255},
		{[]float64{2, 2, 50, 2, 50, 6, 2, 6}, 9, 3, 255},
	}
	for _, c := range cases {
		if got := cover(c.pts, c.x, c.y); got != c.want {
			t.Errorf("%v at %d,%d covers %d, wanted %d", c.pts, c.x, c.y, got, c.want)
		}
	}
	r.reset(12, 12)
	for _, p := range uiStroke([]float64{2, 2, 8, 8, 2, 8}, 1) {
		r.polygon(p)
	}
	if m := r.mask(nil); m[7*12+7] != 255 || m[5*12+5] != 255 {
		t.Errorf("a stroke's join and its middle cover %d and %d", m[7*12+7], m[5*12+5])
	}
}

func TestDraggingOverTextSelectsItAndCopyTakesIt(t *testing.T) {
	n, _ := uiLaidOut(UiColumn(nil, []UiView{UiText(nil, "hello world"), UiText(nil, "again")}), 400, 300)
	n.down(0, 5)
	n.move(35, 5)
	n.up(35, 5)
	if got := n.selectedText(); got != "hello" {
		t.Errorf("dragging selected %q", got)
	}
	n.key(uiKeyAll, false, true)
	n.key(uiKeyCopy, false, true)
	if got := n.dev.clipboard(); got != "hello world\nagain" {
		t.Errorf("copied %q", got)
	}
	uiClick(n, 60, 5)
	uiClick(n, 60, 5)
	if got := n.selectedText(); got != "world" {
		t.Errorf("a double click selected %q", got)
	}
}

func TestAFieldUndoesARunOfTypingAtOnce(t *testing.T) {
	n, box := uiLaidOut(UiInput([]UiAttr{UiAttrOnInput(uiSaid)}, ""), 400, 300)
	n.tab(false)
	for _, r := range "ab cd" {
		n.typed(r)
	}
	f := n.fields["r/0"]
	n.key(uiKeyUndo, false, true)
	n.key(uiKeyUndo, false, true)
	after := string(f.text)
	n.key(uiKeyRedo, false, true)
	if after != "ab" || string(f.text) != "ab " {
		t.Errorf("undone to %q, redone to %q", after, string(f.text))
	}
	if got := uiPosted(box); !strings.HasSuffix(got, "ab cd,ab ,ab,ab ") {
		t.Errorf("heard %s", got)
	}
}

func TestAFieldsMenuActsOnIt(t *testing.T) {
	n, _ := uiLaidOut(UiInput(nil, "one two"), 400, 300)
	b := n.boxes["r/0"]
	n.menu(b.x+15, b.y+10)
	if n.popup == nil || n.popup.kind != "menu" {
		t.Fatalf("no menu opened")
	}
	for i, it := range n.menuItems {
		if it.label == "Select all" {
			n.runMenu(i)
		}
	}
	f := n.fields["r/0"]
	if f.anchor != 0 || f.caret != 7 || n.popup != nil {
		t.Errorf("select all left %d..%d and the menu %v", f.anchor, f.caret, n.popup != nil)
	}
}

func TestAScrollBarCanBeDragged(t *testing.T) {
	var rows []UiView
	for i := 0; i < 40; i++ {
		rows = append(rows, UiText(nil, "line"))
	}
	n, _ := uiLaidOut(UiColumn([]UiAttr{UiAttrHeight(210), UiAttrScroll("Vertical")}, rows), 400, 300)
	b := n.boxes["r/0"]
	n.down(b.x+b.w-3, b.y+5)
	n.move(b.x+b.w-3, b.y+55)
	n.up(b.x+b.w-3, b.y+55)
	n.layout()
	want := 50 * (b.extentH - b.h) / (b.h - math.Max(b.h*b.h/b.extentH, 24))
	if got := n.scrolls["r/0"][1]; !uiNear(got, want) {
		t.Errorf("scrolled to %.2f, wanted %.2f", got, want)
	}
}

func TestATextareaGrowsFromItsCorner(t *testing.T) {
	n, _ := uiLaidOut(UiColumn(nil, []UiView{UiTextarea(nil, "")}), 400, 300)
	b := n.boxes["r/0/0"]
	n.down(b.x+b.w-3, b.y+b.h-3)
	n.move(b.x+b.w-3, b.y+b.h+37)
	n.up(b.x+b.w-3, b.y+b.h+37)
	n.layout()
	if got := n.boxes["r/0/0"].h; !uiNear(got, 104) {
		t.Errorf("the textarea is %.2f tall, wanted 104", got)
	}
}

func TestAMovingGifIsItsFramesInTurn(t *testing.T) {
	palette := color.Palette{color.RGBA{0, 0, 0, 0}, color.RGBA{255, 0, 0, 255}, color.RGBA{0, 0, 255, 255}}
	still := func(c uint8) *image.Paletted {
		img := image.NewPaletted(image.Rect(0, 0, 2, 2), palette)
		for i := range img.Pix {
			img.Pix[i] = c
		}
		return img
	}
	var out bytes.Buffer
	if err := gif.EncodeAll(&out, &gif.GIF{Image: []*image.Paletted{still(1), still(2)}, Delay: []int{5, 20}}); err != nil {
		t.Fatal(err)
	}
	frames, w, h, ok := uiDecodeFrames(out.Bytes())
	if !ok || len(frames) != 2 || w != 2 || h != 2 || frames[1].delay != 200*time.Millisecond || frames[1].pix[2] != 255 {
		t.Fatalf("decoded %d frames of %dx%d", len(frames), w, h)
	}
	p := &uiPicture{frames: frames, start: time.Now().Add(-60 * time.Millisecond)}
	if at, left := p.frameNow(); at != 1 || left <= 0 || left > 200*time.Millisecond {
		t.Errorf("60ms in shows frame %d for %v more", at, left)
	}
}
