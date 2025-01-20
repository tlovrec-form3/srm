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
	hk        *hotkey.Hotkey
	listenKey combo
	sendKey   combo
	appSkip   []string
}

func (h *hk) Register() {
	if h.hk == nil {
		h.hk = hotkey.New(h.listenKey.modifier, h.listenKey.key)
	}
	if err := h.hk.Register(); err != nil {
		panic(err)
	}
}

func (h *hk) Handle() {
	for range h.hk.Keydown() {
		sendKey(h.listenKey, h.sendKey, h.appSkip...)
	}
}
func (h *hk) Unregister() {
	h.hk.Unregister()
}

var keys = []*hk{
	// general edit
	{listenKey: combo{[]hotkey.Modifier{hotkey.ModCtrl}, hotkey.KeyA}, sendKey: combo{[]hotkey.Modifier{hotkey.ModCmd}, keybd_event.VK_A}},
	{listenKey: combo{[]hotkey.Modifier{hotkey.ModCtrl}, hotkey.KeyB}, sendKey: combo{[]hotkey.Modifier{hotkey.ModCmd}, keybd_event.VK_B}},
	{listenKey: combo{[]hotkey.Modifier{hotkey.ModCtrl}, hotkey.KeyI}, sendKey: combo{[]hotkey.Modifier{hotkey.ModCmd}, keybd_event.VK_I}},
	{listenKey: combo{[]hotkey.Modifier{hotkey.ModCtrl, hotkey.ModShift}, hotkey.KeyX}, sendKey: combo{[]hotkey.Modifier{hotkey.ModCmd, hotkey.ModShift}, keybd_event.VK_X}},

	// general copy-pasta
	{listenKey: combo{[]hotkey.Modifier{hotkey.ModCtrl}, hotkey.KeyX}, sendKey: combo{[]hotkey.Modifier{hotkey.ModCmd}, keybd_event.VK_X}, appSkip: []string{"kitty"}},
	{listenKey: combo{[]hotkey.Modifier{hotkey.ModCtrl}, hotkey.KeyC}, sendKey: combo{[]hotkey.Modifier{hotkey.ModCmd}, keybd_event.VK_C}, appSkip: []string{"kitty"}},
	{listenKey: combo{[]hotkey.Modifier{hotkey.ModCtrl}, hotkey.KeyV}, sendKey: combo{[]hotkey.Modifier{hotkey.ModCmd}, keybd_event.VK_V}, appSkip: []string{"kitty"}},

	// browser tabs
	{listenKey: combo{[]hotkey.Modifier{hotkey.ModCtrl}, hotkey.KeyR}, sendKey: combo{[]hotkey.Modifier{hotkey.ModCmd}, keybd_event.VK_R}},
	{listenKey: combo{[]hotkey.Modifier{hotkey.ModCtrl}, hotkey.KeyT}, sendKey: combo{[]hotkey.Modifier{hotkey.ModCmd}, keybd_event.VK_T}},
	{listenKey: combo{[]hotkey.Modifier{hotkey.ModCtrl, hotkey.ModShift}, hotkey.KeyT}, sendKey: combo{[]hotkey.Modifier{hotkey.ModCmd, hotkey.ModShift}, keybd_event.VK_T}},
	{listenKey: combo{[]hotkey.Modifier{hotkey.ModCtrl}, hotkey.KeyW}, sendKey: combo{[]hotkey.Modifier{hotkey.ModCmd}, keybd_event.VK_W}},

	// slack switch channel/dm window
	{listenKey: combo{[]hotkey.Modifier{hotkey.ModCtrl}, hotkey.KeyK}, sendKey: combo{[]hotkey.Modifier{hotkey.ModCmd}, keybd_event.VK_K}},
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

func sendKey(listen combo, send combo, appSkip ...string) {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	objc.WithAutoreleasePool(func() {
		ws := appkit.Workspace_SharedWorkspace()
		app := ws.FrontmostApplication()

		for _, appName := range appSkip {
			if app.LocalizedName() == appName {
				send.modifier = listen.modifier
				send.key = listen.key
				break
			}
		}
	})

	kb, err := keybd_event.NewKeyBonding()
	if err != nil {
		log.Printf("failed creating key bonding: %w", err)
	}

	kb.SetKeys(int(send.key))
	for _, m := range send.modifier {
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
