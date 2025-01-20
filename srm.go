package main

import (
	"log"
	"os"
	"os/signal"
	"runtime"
	"syscall"

	"github.com/micmonay/keybd_event"
	"github.com/progrium/darwinkit/macos/appkit"
	"github.com/progrium/darwinkit/objc"
	"golang.design/x/hotkey"
	"golang.design/x/hotkey/mainthread"
)

func main() {
	mainthread.Init(fn)
}

type combo struct {
	modifier []hotkey.Modifier
	key      hotkey.Key
}

type hk struct {
	hk       *hotkey.Hotkey
	combo    combo
	modifier []hotkey.Modifier
	key      int
	appSkip  []string
}

func (h *hk) Register() {
	if h.hk == nil {
		h.hk = hotkey.New(h.combo.modifier, h.combo.key)
	}
	if err := h.hk.Register(); err != nil {
		panic(err)
	}
}

func (h *hk) Handle() {
	for range h.hk.Keydown() {
		sendKey(h.combo, h.modifier, h.key, h.appSkip...)
	}
}
func (h *hk) Unregister() {
	h.hk.Unregister()
}

var keys = []*hk{
	// general editing
	{combo: combo{[]hotkey.Modifier{hotkey.ModCtrl}, hotkey.KeyX}, modifier: []hotkey.Modifier{hotkey.ModCmd}, key: keybd_event.VK_X, appSkip: []string{"kitty"}},
	{combo: combo{[]hotkey.Modifier{hotkey.ModCtrl}, hotkey.KeyC}, modifier: []hotkey.Modifier{hotkey.ModCmd}, key: keybd_event.VK_C, appSkip: []string{"kitty"}},
	{combo: combo{[]hotkey.Modifier{hotkey.ModCtrl}, hotkey.KeyV}, modifier: []hotkey.Modifier{hotkey.ModCmd}, key: keybd_event.VK_V, appSkip: []string{"kitty"}},
	{combo: combo{[]hotkey.Modifier{hotkey.ModCtrl}, hotkey.KeyA}, modifier: []hotkey.Modifier{hotkey.ModCmd}, key: keybd_event.VK_A, appSkip: []string{"kitty"}},

	// browser tabs
	{combo: combo{[]hotkey.Modifier{hotkey.ModCtrl}, hotkey.KeyR}, modifier: []hotkey.Modifier{hotkey.ModCmd}, key: keybd_event.VK_R},
	{combo: combo{[]hotkey.Modifier{hotkey.ModCtrl}, hotkey.KeyT}, modifier: []hotkey.Modifier{hotkey.ModCmd}, key: keybd_event.VK_T},
	{combo: combo{[]hotkey.Modifier{hotkey.ModCtrl, hotkey.ModShift}, hotkey.KeyT}, modifier: []hotkey.Modifier{hotkey.ModCmd, hotkey.ModShift}, key: keybd_event.VK_T},
	{combo: combo{[]hotkey.Modifier{hotkey.ModCtrl}, hotkey.KeyW}, modifier: []hotkey.Modifier{hotkey.ModCmd}, key: keybd_event.VK_W},

	// slack switch channel/dm window
	{combo: combo{[]hotkey.Modifier{hotkey.ModCtrl}, hotkey.KeyK}, modifier: []hotkey.Modifier{hotkey.ModCmd}, key: keybd_event.VK_K},
}

func fn() {
	log.Println("Registering handlers...")
	for _, k := range keys {
		k.Register()
		go k.Handle()
	}
	log.Println("Handlers registered.")
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	<-sigs

	log.Println("Unregistering handlers...")
	for _, k := range keys {
		go k.Unregister()
	}
	log.Println("Handlers unregistered. Bye!")
}

func sendKey(c combo, modifiers []hotkey.Modifier, key int, appSkip ...string) {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	objc.WithAutoreleasePool(func() {
		ws := appkit.Workspace_SharedWorkspace()
		app := ws.FrontmostApplication()

		for _, appName := range appSkip {
			if app.LocalizedName() == appName {
				modifiers = c.modifier
				key = int(c.key)
				break
			}
		}
	})

	kb, err := keybd_event.NewKeyBonding()
	if err != nil {
		log.Printf("failed creating key bonding: %w", err)
	}

	kb.SetKeys(key)
	for _, m := range modifiers {
		switch m {
		case hotkey.ModCmd:
			kb.HasSuper(true)
		case hotkey.ModCtrl:
			kb.HasCTRL(true)
		case hotkey.ModOption:
			kb.HasALT(true)
		case hotkey.ModShift:
			kb.HasSHIFT(true)
		}
	}

	err = kb.Launching()
	if err != nil {
		log.Printf("failed sending key bonding: %w", err)
	}
}
