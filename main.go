// ff-installer: creates Start Menu and Desktop shortcuts that open a web app in its own window
// (Edge/Chrome --app mode). Which app is chosen by this file's name: "<app-id>-setup.exe".
// The app's name, start address and icon are read at run time from the app's public web manifest
// (https://frenchfries777.com/games/pwa/<app-id>.webmanifest). Nothing else is downloaded or run.
package main

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"syscall"
	"time"
	"unsafe"

	ole "github.com/go-ole/go-ole"
	"github.com/go-ole/go-ole/oleutil"
)

const host = "frenchfries777.com"

type manifest struct {
	Name     string `json:"name"`
	StartURL string `json:"start_url"`
	Icons    []struct {
		Src   string `json:"src"`
		Sizes string `json:"sizes"`
		Type  string `json:"type"`
	} `json:"icons"`
}

func box(text string, flags uintptr) int {
	t, _ := syscall.UTF16PtrFromString(text)
	c, _ := syscall.UTF16PtrFromString("Frenchfries777")
	r, _, _ := syscall.NewLazyDLL("user32.dll").NewProc("MessageBoxW").Call(0, uintptr(unsafe.Pointer(t)), uintptr(unsafe.Pointer(c)), flags)
	return int(r)
}

func get(client *http.Client, u string, limit int64) ([]byte, error) {
	r, err := client.Get(u)
	if err != nil {
		return nil, err
	}
	defer r.Body.Close()
	if r.StatusCode != 200 {
		return nil, os.ErrNotExist
	}
	return io.ReadAll(io.LimitReader(r.Body, limit))
}

func findBrowser() string {
	pf, pf86, la := os.Getenv("ProgramFiles"), os.Getenv("ProgramFiles(x86)"), os.Getenv("LOCALAPPDATA")
	for _, p := range []string{
		filepath.Join(pf86, `Microsoft\Edge\Application\msedge.exe`),
		filepath.Join(pf, `Microsoft\Edge\Application\msedge.exe`),
		filepath.Join(pf, `Google\Chrome\Application\chrome.exe`),
		filepath.Join(pf86, `Google\Chrome\Application\chrome.exe`),
		filepath.Join(la, `Google\Chrome\Application\chrome.exe`),
	} {
		if filepath.IsAbs(p) {
			if _, err := os.Stat(p); err == nil {
				return p
			}
		}
	}
	return ""
}

func icoFromPNG(png []byte) []byte {
	w := int(binary.BigEndian.Uint32(png[16:20]))
	h := int(binary.BigEndian.Uint32(png[20:24]))
	bw, bh := byte(w), byte(h)
	if w >= 256 {
		bw = 0
	}
	if h >= 256 {
		bh = 0
	}
	var o bytes.Buffer
	binary.Write(&o, binary.LittleEndian, []uint16{0, 1, 1})
	o.Write([]byte{bw, bh, 0, 0})
	binary.Write(&o, binary.LittleEndian, []uint16{1, 32})
	binary.Write(&o, binary.LittleEndian, uint32(len(png)))
	binary.Write(&o, binary.LittleEndian, uint32(22))
	o.Write(png)
	return o.Bytes()
}

func shortcut(sh *ole.IDispatch, dir, name, target, args, icon string) {
	l, err := oleutil.CallMethod(sh, "CreateShortcut", filepath.Join(dir, name+".lnk"))
	if err != nil {
		return
	}
	d := l.ToIDispatch()
	defer d.Release()
	oleutil.PutProperty(d, "TargetPath", target)
	oleutil.PutProperty(d, "Arguments", args)
	oleutil.PutProperty(d, "IconLocation", icon)
	oleutil.PutProperty(d, "Description", name)
	oleutil.CallMethod(d, "Save")
}

func specialDirs(sh *ole.IDispatch) []string {
	var out []string
	for _, kind := range []string{"Programs", "Desktop"} {
		if v, err := oleutil.GetProperty(sh, "SpecialFolders", kind); err == nil && v.ToString() != "" {
			out = append(out, v.ToString())
		}
	}
	return out
}

func main() {
	self, _ := os.Executable()
	m := regexp.MustCompile(`^([a-z0-9]+)[-_ ]setup`).FindStringSubmatch(strings.ToLower(filepath.Base(self)))
	if m == nil {
		box("Please do not rename this file. Download it again from the install page.", 0x10)
		return
	}
	id := m[1]
	uninstall := len(os.Args) > 1 && strings.EqualFold(os.Args[1], "/uninstall")
	base := filepath.Join(os.Getenv("LOCALAPPDATA"), "Frenchfries777", id)

	client := &http.Client{Timeout: 20 * time.Second}
	murl := "https://" + host + "/games/pwa/" + id + ".webmanifest"
	raw, err := get(client, murl, 256<<10)
	var mf manifest
	if err != nil || json.Unmarshal(raw, &mf) != nil || mf.Name == "" {
		box("Could not read the app information. Check your internet connection.", 0x10)
		return
	}
	start, err := url.Parse(mf.StartURL)
	base0, _ := url.Parse(murl)
	if err != nil {
		return
	}
	start = base0.ResolveReference(start)
	if start.Scheme != "https" || start.Hostname() != host {
		return
	}
	safe := strings.TrimSpace(regexp.MustCompile(`[<>:"/\\|?*]`).ReplaceAllString(mf.Name, ""))

	ole.CoInitializeEx(0, ole.COINIT_APARTMENTTHREADED)
	defer ole.CoUninitialize()
	unk, err := oleutil.CreateObject("WScript.Shell")
	if err != nil {
		box("Could not create shortcuts: "+err.Error(), 0x10)
		return
	}
	sh, _ := unk.QueryInterface(ole.IID_IDispatch)

	if uninstall {
		for _, d := range specialDirs(sh) {
			os.Remove(filepath.Join(d, safe+".lnk"))
		}
		os.RemoveAll(base)
		box("Removed: "+mf.Name, 0x40)
		return
	}

	// Announce system changes and let the user decline.
	if box("Create shortcuts for \""+mf.Name+"\" in the Start menu and on the Desktop?\n\n"+
		"It opens "+start.String()+" in its own window.\n"+
		"To remove it later, delete those two shortcuts (and the folder "+base+"), or run this file with /uninstall.",
		0x24) != 6 { // MB_YESNO | MB_ICONQUESTION, IDYES
		return
	}

	ico := filepath.Join(base, "icon.ico")
	if os.MkdirAll(base, 0o755) == nil {
		best, bestSize := "", 0
		for _, ic := range mf.Icons {
			var n int
			if _, e := fmtSscan(ic.Sizes, &n); e == nil && strings.Contains(ic.Type, "png") && n >= 128 && (bestSize == 0 || n < bestSize) {
				best, bestSize = ic.Src, n
			}
		}
		if best != "" {
			if iu, e := url.Parse(best); e == nil {
				iu = base0.ResolveReference(iu)
				if iu.Scheme == "https" && iu.Hostname() == host {
					if png, e := get(client, iu.String(), 2<<20); e == nil && len(png) > 24 {
						os.WriteFile(ico, icoFromPNG(png), 0o644)
					}
				}
			}
		}
	}
	if _, e := os.Stat(ico); e != nil {
		ico = ""
	}

	target, args := findBrowser(), "--app="+start.String()
	if target == "" {
		target, args = filepath.Join(os.Getenv("SystemRoot"), "explorer.exe"), start.String()
	}
	for _, d := range specialDirs(sh) {
		shortcut(sh, d, safe, target, args, ico)
	}

	p, _ := syscall.UTF16PtrFromString(target)
	cl, _ := syscall.UTF16PtrFromString(`"` + target + `" ` + args)
	var si syscall.StartupInfo
	var pi syscall.ProcessInformation
	if syscall.CreateProcess(p, cl, nil, nil, false, 0, nil, nil, &si, &pi) == nil {
		syscall.CloseHandle(pi.Thread)
		syscall.CloseHandle(pi.Process)
	}
}

// fmtSscan reads the first integer of a "192x192" style size string.
func fmtSscan(s string, n *int) (int, error) {
	i := strings.IndexAny(s, "xX")
	if i < 0 {
		i = len(s)
	}
	v := 0
	if i == 0 {
		return 0, os.ErrInvalid
	}
	for _, c := range s[:i] {
		if c < '0' || c > '9' {
			return 0, os.ErrInvalid
		}
		v = v*10 + int(c-'0')
	}
	*n = v
	return 1, nil
}
