package menu

import (
	"math"

	"github.com/libretro/ludo/state"
	"github.com/libretro/ludo/video"
	"github.com/tanema/gween"
	"github.com/tanema/gween/ease"
)

// entry is a menu entry. It can also represent a scene.
// The menu data is a tree of entries.
type entry struct {
	alpha           float32
	yp, scale       float32
	width, height   float32
	margin          float32
	label, subLabel string
	path            string // full path of the rom linked to the entry
	system          string // name of the game system
	labelAlpha      float32
	icon            string
	iconAlpha       float32
	tagAlpha        float32
	subLabelAlpha   float32
	borderAlpha     float32
	callbackOK      func() // callback executed when user presses OK
	value           func() interface{}
	stringValue     func() string
	widget          func(*entry) // widget draw callback used in settings
	incr            func(int)    // increment callback used in settings
	tags            []string     // flags extracted from game title
	thumbnail       uint32       // thumbnail texture id
	gameName        string       // title of the game in db, used for thumbnails
	cursor          struct {
		alpha float32
		yp    float32
	}
	children []entry // children entries
	ptr      int     // index of the active child
	indexes  []struct {
		Char  byte
		Index int
	}
}

// Scene represents a page of the UI
// A scene is typically an entry displaying its own children
// A segue is a smooth transition between two scenes.
type Scene interface {
	segueMount()
	segueNext()
	segueBack()
	update(dt float32)
	render()
	drawHintBar()
	Entry() *entry
}

// genericSegueMount is the smooth transition of the menu entries first appearance
func genericSegueMount(list *entry) {
	for i := range list.children {
		e := &list.children[i]

		if i == list.ptr {
			e.yp = 0.5 + 0.3
			e.scale = 1.5
		} else if i < list.ptr {
			e.yp = 0.4 + 0.3 + 0.08*float32(i-list.ptr)
			e.scale = 0.5
		} else if i > list.ptr {
			e.yp = 0.6 + 0.3 + 0.08*float32(i-list.ptr)
			e.scale = 0.5
		}
		e.labelAlpha = 0
		e.iconAlpha = 0
		e.tagAlpha = 0
		e.subLabelAlpha = 0
	}
	list.cursor.alpha = 0
	list.cursor.yp = 0.5 + 0.3

	genericAnimate(list)
}

// genericAnimate is the generic animation of entries when the user scrolls up or down
func genericAnimate(list *entry) {
	for i := range list.children {
		e := &list.children[i]

		// performance improvement
		// if math.Abs(float64(i-list.ptr)) > 6 && i > 6 && i < len(list.children)-6 {
		// 	continue
		// }

		var yp, tagAlpha, subLabelAlpha, scale float32
		if i == list.ptr {
			yp = 0.5
			tagAlpha = 1
			subLabelAlpha = 1
			scale = 1.5
		} else if i < list.ptr {
			yp = 0.4 + 0.08*float32(i-list.ptr)
			tagAlpha = 0
			subLabelAlpha = 0
			scale = 0.5
		} else if i > list.ptr {
			yp = 0.6 + 0.08*float32(i-list.ptr)
			tagAlpha = 0
			subLabelAlpha = 0
			scale = 0.5
		}

		menu.tweens[&e.yp] = gween.New(e.yp, yp, 0.15, ease.OutSine)
		menu.tweens[&e.labelAlpha] = gween.New(e.labelAlpha, 1, 0.15, ease.OutSine)
		menu.tweens[&e.iconAlpha] = gween.New(e.iconAlpha, 1, 0.15, ease.OutSine)
		menu.tweens[&e.tagAlpha] = gween.New(e.tagAlpha, tagAlpha, 0.15, ease.OutSine)
		menu.tweens[&e.subLabelAlpha] = gween.New(e.subLabelAlpha, subLabelAlpha, 0.15, ease.OutSine)
		menu.tweens[&e.scale] = gween.New(e.scale, scale, 0.15, ease.OutSine)
	}
	menu.tweens[&list.cursor.alpha] = gween.New(list.cursor.alpha, 1, 0.15, ease.OutSine)
	menu.tweens[&list.cursor.yp] = gween.New(list.cursor.yp, 0.5, 0.15, ease.OutSine)
}

// genericSegueNext is a smooth transition that fades out the current list
// to leave room for the next list to appear
func genericSegueNext(list *entry) {
	for i := range list.children {
		e := &list.children[i]

		var yp, scale float32
		if i == list.ptr {
			yp = 0.5 - 0.3
			scale = 1.5
		} else if i < list.ptr {
			yp = 0.4 - 0.3 + 0.08*float32(i-list.ptr)
			scale = 0.5
		} else if i > list.ptr {
			yp = 0.6 - 0.3 + 0.08*float32(i-list.ptr)
			scale = 0.5
		}

		menu.tweens[&e.yp] = gween.New(e.yp, yp, 0.15, ease.OutSine)
		menu.tweens[&e.labelAlpha] = gween.New(e.labelAlpha, 0, 0.15, ease.OutSine)
		menu.tweens[&e.iconAlpha] = gween.New(e.iconAlpha, 0, 0.15, ease.OutSine)
		menu.tweens[&e.tagAlpha] = gween.New(e.tagAlpha, 0, 0.15, ease.OutSine)
		menu.tweens[&e.subLabelAlpha] = gween.New(e.subLabelAlpha, 0, 0.15, ease.OutSine)
		menu.tweens[&e.scale] = gween.New(e.scale, scale, 0.15, ease.OutSine)
	}
	menu.tweens[&list.cursor.alpha] = gween.New(list.cursor.alpha, 0, 0.15, ease.OutSine)
	menu.tweens[&list.cursor.yp] = gween.New(list.cursor.yp, 0.5-0.3, 0.15, ease.OutSine)
}

// genericDrawCursor draws the blinking rectangular background of the active
// menu entry
func genericDrawCursor(list *entry) {
	w, h := vid.Window.GetFramebufferSize()
	if menu.focus > 1 {
		blink := float32(math.Cos(menu.t))
		vid.DrawImage(
			menu.icons["selection"],
			360*menu.ratio-8*menu.ratio,
			float32(h)*list.cursor.yp-50*menu.ratio-8*menu.ratio,
			float32(w)-720*menu.ratio+16*menu.ratio,
			100*menu.ratio+16*menu.ratio,
			1, 0.15, video.Color{R: 1, G: 1, B: 1, A: list.cursor.alpha - list.cursor.alpha*blink})
	}
	vid.DrawRect(
		360*menu.ratio, float32(h)*list.cursor.yp-50*menu.ratio,
		float32(w)-720*menu.ratio, 100*menu.ratio, 0.1,
		video.Color{R: 1, G: 1, B: 1, A: list.cursor.alpha})
}

// thumbnailDrawCursor draws the blinking rectangular background of the active
// menu entry when there is a thumbnail
func thumbnailDrawCursor(list *entry) {
	w, h := vid.Window.GetFramebufferSize()
	if menu.focus > 1 {
		blink := float32(math.Cos(menu.t))
		vid.DrawImage(
			menu.icons["selection"],
			360*menu.ratio-8*menu.ratio,
			float32(h)*list.cursor.yp-120*menu.ratio-8*menu.ratio,
			float32(w)-720*menu.ratio+16*menu.ratio,
			240*menu.ratio+16*menu.ratio,
			1, 0.15, video.Color{R: 1, G: 1, B: 1, A: list.cursor.alpha - list.cursor.alpha*blink})
	}
	vid.DrawRect(
		360*menu.ratio, float32(h)*list.cursor.yp-120*menu.ratio,
		float32(w)-720*menu.ratio, 240*menu.ratio, 0.1,
		video.Color{R: 1, G: 1, B: 1, A: list.cursor.alpha})
}

// genericRender renders a vertical list of menu entries
// It also display values of settings if we are displaying a settings scene
func genericRender(list *entry) {
	w, h := vid.Window.GetFramebufferSize()

	genericDrawCursor(list)

	for _, e := range list.children {
		if e.yp < -0.1 || e.yp > 1.1 {
			continue
		}

		fontOffset := 64 * 0.7 * menu.ratio * 0.3

		color := video.Color{R: 0, G: 0, B: 0, A: e.iconAlpha}

		vid.DrawImage(menu.icons[e.icon],
			420*menu.ratio-64*0.35*menu.ratio,
			float32(h)*e.yp-14*menu.ratio-64*0.35*menu.ratio+fontOffset,
			128*menu.ratio, 128*menu.ratio,
			0.35, 0, color)

		if e.labelAlpha > 0 {
			vid.Font.SetColor(color.R, color.G, color.B, e.labelAlpha)
			vid.Font.Printf(
				480*menu.ratio,
				float32(h)*e.yp+fontOffset,
				0.5*menu.ratio, e.label)

			if e.widget != nil {
				e.widget(&e)
			} else if e.stringValue != nil {
				lw := vid.Font.Width(0.5*menu.ratio, e.stringValue())
				vid.Font.Printf(
					float32(w)-lw-400*menu.ratio,
					float32(h)*e.yp+fontOffset,
					0.5*menu.ratio, e.stringValue())
			}
		}
	}
}

func genericDrawHintBar() {
	w, h := vid.Window.GetFramebufferSize()
	vid.DrawRect(0, float32(h)-88*menu.ratio, float32(w), 88*menu.ratio, 0, video.Color{R: 1, G: 1, B: 1, A: 1})
	vid.DrawRect(0, float32(h)-88*menu.ratio, float32(w), 2*menu.ratio, 0, video.Color{R: 0.85, G: 0.85, B: 0.85, A: 1})

	_, upDown, _, a, b, _, _, _, _, guide := hintIcons()

	lstack := float32(75) * menu.ratio
	rstack := float32(w) - 96*menu.ratio
	stackHintLeft(&lstack, upDown, "Navigate", h)
	stackHintRight(&rstack, a, "Ok", h)
	stackHintRight(&rstack, b, "Back", h)
	if state.Global.CoreRunning {
		stackHintRight(&rstack, guide, "Resume", h)
	}
}
