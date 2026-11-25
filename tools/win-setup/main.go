// منظومة تراخيص الآبار بالخارجة — برنامج التثبيت والتشغيل لويندوز
// نفس الملف بيثبّت نفسه في %LOCALAPPDATA%\WellsKharga ويشتغل كـ launcher (Edge/Chrome --app).
// الاختصارات بتتكتب كملف .lnk مباشرة (من غير PowerShell)، ولو فشل: PowerShell، ولو فشل: ملف .url.
// وكل مرة البرنامج يفتح بيصلّح الاختصارات لو ناقصة.
// جميع الحقوق محفوظة للمصمم والمطور - م/ أسامه حسن بدري - 01145060007
package main

import (
	_ "embed"
	"encoding/base64"
	"encoding/binary"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"unicode/utf16"
	"unsafe"
)

const (
	appName = "منظومة تراخيص الآبار بالخارجة"
	appURL  = "https://osama-badry.github.io/Groundwater.khg/"
	appVer  = "1.2.0"
	regKey  = `HKCU\Software\Microsoft\Windows\CurrentVersion\Uninstall\WellsKharga`
	sig     = "جميع الحقوق محفوظة للمصمم والمطور - م/ أسامه حسن بدري - 01145060007"
	exeName = "WellsKharga.exe"
)

//go:embed app.ico
var icoData []byte

var (
	user32   = syscall.NewLazyDLL("user32.dll")
	shell32  = syscall.NewLazyDLL("shell32.dll")
	ole32    = syscall.NewLazyDLL("ole32.dll")
	pMsgBox  = user32.NewProc("MessageBoxW")
	pKnown   = shell32.NewProc("SHGetKnownFolderPath")
	pNotify  = shell32.NewProc("SHChangeNotify")
	pCoFree  = ole32.NewProc("CoTaskMemFree")
)

func u16(s string) *uint16 { p, _ := syscall.UTF16PtrFromString(s); return p }

// flags: 0x40 info, 0x10 error, 0x30 warn, 0x24 yes/no question; 0x00100000 RTL reading, 0x00080000 right
func msg(text string, flags uintptr) int {
	r, _, _ := pMsgBox.Call(0, uintptr(unsafe.Pointer(u16(text))), uintptr(unsafe.Pointer(u16(appName))), flags|0x00180000|0x00010000)
	return int(r)
}

func hidden(c *exec.Cmd) *exec.Cmd {
	c.SysProcAttr = &syscall.SysProcAttr{HideWindow: true, CreationFlags: 0x08000000}
	return c
}

func installDir() string {
	b := os.Getenv("LOCALAPPDATA")
	if b == "" {
		b = filepath.Join(os.Getenv("USERPROFILE"), "AppData", "Local")
	}
	return filepath.Join(b, "WellsKharga")
}

// ---------- المجلدات الخاصة (بتتعامل صح مع سطح مكتب OneDrive أو المنقول لمكان تاني) ----------
type guid struct {
	a    uint32
	b, c uint16
	d    [8]byte
}

var (
	folderDesktop  = guid{0xB4BFCC3A, 0xDB2C, 0x424C, [8]byte{0xB0, 0x29, 0x7F, 0xE9, 0x9A, 0x87, 0xC6, 0x41}}
	folderPrograms = guid{0xA77F5D77, 0x2E2B, 0x44C3, [8]byte{0xA6, 0xA2, 0xAB, 0xA6, 0x01, 0x05, 0x4A, 0x51}}
)

func knownFolder(g *guid) string {
	if pKnown.Find() != nil {
		return ""
	}
	var p *uint16
	r, _, _ := pKnown.Call(uintptr(unsafe.Pointer(g)), 0x8000 /*KF_FLAG_CREATE*/, 0, uintptr(unsafe.Pointer(&p)))
	if r != 0 || p == nil {
		return ""
	}
	defer pCoFree.Call(uintptr(unsafe.Pointer(p)))
	n := 0
	for *(*uint16)(unsafe.Pointer(uintptr(unsafe.Pointer(p)) + uintptr(n*2))) != 0 {
		n++
	}
	return string(utf16.Decode(unsafe.Slice(p, n)))
}

func isDir(p string) bool { st, err := os.Stat(p); return err == nil && st.IsDir() }

func desktopDirs() []string {
	var out []string
	add := func(p string) {
		if p == "" || !isDir(p) {
			return
		}
		for _, x := range out {
			if strings.EqualFold(x, p) {
				return
			}
		}
		out = append(out, p)
	}
	add(knownFolder(&folderDesktop))
	up := os.Getenv("USERPROFILE")
	add(filepath.Join(up, "Desktop"))
	for _, k := range []string{"OneDrive", "OneDriveCommercial", "OneDriveConsumer"} {
		if od := os.Getenv(k); od != "" {
			add(filepath.Join(od, "Desktop"))
			add(filepath.Join(od, "سطح المكتب"))
		}
	}
	return out
}

func programsDir() string {
	if p := knownFolder(&folderPrograms); p != "" {
		return p
	}
	p := filepath.Join(os.Getenv("APPDATA"), `Microsoft\Windows\Start Menu\Programs`)
	os.MkdirAll(p, 0755)
	return p
}

// ---------- كتابة ملف .lnk مباشرة (MS-SHLLINK) ----------
func utf16le(s string) []byte {
	u := utf16.Encode([]rune(s))
	b := make([]byte, len(u)*2)
	for i, c := range u {
		binary.LittleEndian.PutUint16(b[i*2:], c)
	}
	return b
}

func strData(s string) []byte {
	u := utf16.Encode([]rune(s))
	b := make([]byte, 2+len(u)*2)
	binary.LittleEndian.PutUint16(b, uint16(len(u)))
	for i, c := range u {
		binary.LittleEndian.PutUint16(b[2+i*2:], c)
	}
	return b
}

func ansi(s string) []byte {
	b := []byte{}
	for _, r := range s {
		if r < 128 {
			b = append(b, byte(r))
		} else {
			b = append(b, '?')
		}
	}
	return append(b, 0)
}

func buildLnk(target, workDir, desc, icon string) []byte {
	// LinkFlags: HasLinkInfo|HasName|HasWorkingDir|HasIconLocation|IsUnicode|ForceNoLinkTrack(0x40000)
	const flags = 0x02 | 0x04 | 0x10 | 0x40 | 0x80 | 0x40000
	h := make([]byte, 0x4C)
	binary.LittleEndian.PutUint32(h[0:], 0x4C)
	copy(h[4:], []byte{0x01, 0x14, 0x02, 0x00, 0x00, 0x00, 0x00, 0x00, 0xC0, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x46})
	binary.LittleEndian.PutUint32(h[0x14:], flags)
	binary.LittleEndian.PutUint32(h[0x18:], 0x20) // FILE_ATTRIBUTE_ARCHIVE
	if st, err := os.Stat(target); err == nil {
		binary.LittleEndian.PutUint32(h[0x34:], uint32(st.Size()))
	}
	binary.LittleEndian.PutUint32(h[0x38:], 0) // icon index
	binary.LittleEndian.PutUint32(h[0x3C:], 1) // SW_SHOWNORMAL

	// LinkInfo مع المسار ANSI + Unicode (عشان أسماء المستخدمين العربي)
	vol := []byte{}
	vb := make([]byte, 0x10)
	binary.LittleEndian.PutUint32(vb[0:], 0x11)
	binary.LittleEndian.PutUint32(vb[4:], 3) // DRIVE_FIXED
	binary.LittleEndian.PutUint32(vb[0xC:], 0x10)
	vol = append(vb, 0) // empty label
	lbpA := ansi(target)
	suffixA := []byte{0}
	if (0x24+len(vol)+len(lbpA)+1)%2 == 1 {
		suffixA = []byte{0, 0} // محاذاة النص Unicode على عدد زوجي
	}
	lbpU := append(utf16le(target), 0, 0)
	suffixU := []byte{0, 0}
	const hdr = 0x24
	volOff := hdr
	lbpOff := volOff + len(vol)
	sufOff := lbpOff + len(lbpA)
	lbpUOff := sufOff + len(suffixA)
	sufUOff := lbpUOff + len(lbpU)
	total := sufUOff + len(suffixU)
	li := make([]byte, hdr, total)
	binary.LittleEndian.PutUint32(li[0:], uint32(total))
	binary.LittleEndian.PutUint32(li[4:], hdr)
	binary.LittleEndian.PutUint32(li[8:], 1) // VolumeIDAndLocalBasePath
	binary.LittleEndian.PutUint32(li[0xC:], uint32(volOff))
	binary.LittleEndian.PutUint32(li[0x10:], uint32(lbpOff))
	binary.LittleEndian.PutUint32(li[0x14:], 0)
	binary.LittleEndian.PutUint32(li[0x18:], uint32(sufOff))
	binary.LittleEndian.PutUint32(li[0x1C:], uint32(lbpUOff))
	binary.LittleEndian.PutUint32(li[0x20:], uint32(sufUOff))
	li = append(li, vol...)
	li = append(li, lbpA...)
	li = append(li, suffixA...)
	li = append(li, lbpU...)
	li = append(li, suffixU...)

	out := append(h, li...)
	out = append(out, strData(desc)...)
	out = append(out, strData(workDir)...)
	out = append(out, strData(icon)...)
	out = append(out, 0, 0, 0, 0) // TerminalBlock
	return out
}

func writeLnk(path, target, dir, ico string) error {
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, buildLnk(target, dir, appName, ico), 0644); err != nil {
		return err
	}
	os.Remove(path)
	return os.Rename(tmp, path)
}

func psq(s string) string { return "'" + strings.ReplaceAll(s, "'", "''") + "'" }

func b64(s string) string { return base64.StdEncoding.EncodeToString(utf16le(s)) }

func psLnk(path, target, dir, ico string) error {
	sc := "$ErrorActionPreference='Stop';$w=New-Object -ComObject WScript.Shell;$s=$w.CreateShortcut(" + psq(path) +
		");$s.TargetPath=" + psq(target) + ";$s.WorkingDirectory=" + psq(dir) + ";$s.IconLocation=" + psq(ico+",0") +
		";$s.Description=" + psq(appName) + ";$s.Save();"
	return hidden(exec.Command("powershell.exe", "-NoProfile", "-NonInteractive", "-ExecutionPolicy", "Bypass", "-EncodedCommand", b64(sc))).Run()
}

func urlFile(path, target, ico string) error {
	u := "file:///" + strings.ReplaceAll(target, `\`, "/")
	c := "[InternetShortcut]\r\nURL=" + u + "\r\nIconFile=" + ico + "\r\nIconIndex=0\r\n"
	return os.WriteFile(strings.TrimSuffix(path, ".lnk")+".url", append([]byte{0xEF, 0xBB, 0xBF}, []byte(c)...), 0644)
}

// بيجرّب 3 طرق بالترتيب؛ بيرجع true لو اتعمل
func makeOne(path, target, dir, ico string) bool {
	if writeLnk(path, target, dir, ico) == nil {
		return true
	}
	os.Remove(path + ".tmp")
	if psLnk(path, target, dir, ico) == nil {
		if _, err := os.Stat(path); err == nil {
			return true
		}
	}
	return urlFile(path, target, ico) == nil
}

func shortcutPaths() []string {
	var out []string
	for _, d := range desktopDirs() {
		out = append(out, filepath.Join(d, appName+".lnk"))
	}
	if p := programsDir(); p != "" {
		out = append(out, filepath.Join(p, appName+".lnk"))
	}
	return out
}

func exists(p string) bool { _, err := os.Stat(p); return err == nil }

// بيرجع عدد الاختصارات اللي اتعملت/موجودة: سطح المكتب (أول واحد بس) + قائمة ابدأ
func makeShortcuts(force bool) (desk, start bool) {
	dir := installDir()
	target := filepath.Join(dir, exeName)
	ico := filepath.Join(dir, "app.ico")
	for _, d := range desktopDirs() {
		p := filepath.Join(d, appName+".lnk")
		if !force && (exists(p) || exists(strings.TrimSuffix(p, ".lnk")+".url")) {
			desk = true
			break
		}
		if makeOne(p, target, dir, ico) {
			desk = true
			break
		}
	}
	if pd := programsDir(); pd != "" {
		p := filepath.Join(pd, appName+".lnk")
		start = (!force && (exists(p) || exists(strings.TrimSuffix(p, ".lnk")+".url"))) || makeOne(p, target, dir, ico)
	}
	if pNotify.Find() == nil {
		pNotify.Call(0x08000000, 0, 0, 0) // SHCNE_ASSOCCHANGED → تحديث الأيقونات
	}
	return
}

func copyFile(src, dst string) error {
	b, err := os.ReadFile(src)
	if err != nil {
		return err
	}
	tmp := dst + ".new"
	if err := os.WriteFile(tmp, b, 0755); err != nil {
		return err
	}
	if exists(dst) {
		old := dst + ".old"
		os.Remove(old)
		if err := os.Rename(dst, old); err != nil {
			os.Remove(tmp)
			return err
		}
	}
	return os.Rename(tmp, dst)
}

func regSet(name, typ, val string) {
	hidden(exec.Command("reg.exe", "add", regKey, "/v", name, "/t", typ, "/d", val, "/f")).Run()
}

func install() {
	dir := installDir()
	if err := os.MkdirAll(dir, 0755); err != nil {
		msg("تعذر إنشاء مجلد البرنامج:\n"+err.Error(), 0x10)
		os.Exit(1)
	}
	self, _ := os.Executable()
	dst := filepath.Join(dir, exeName)
	if err := copyFile(self, dst); err != nil {
		msg("تعذر نسخ البرنامج. أغلق المنظومة إن كانت مفتوحة ثم أعد المحاولة.\n"+err.Error(), 0x10)
		os.Exit(1)
	}
	os.WriteFile(filepath.Join(dir, "app.ico"), icoData, 0644)
	os.MkdirAll(filepath.Join(dir, "profile"), 0755)
	regSet("DisplayName", "REG_SZ", appName)
	regSet("DisplayIcon", "REG_SZ", filepath.Join(dir, "app.ico"))
	regSet("DisplayVersion", "REG_SZ", appVer)
	regSet("Publisher", "REG_SZ", "م/ أسامه حسن بدري - 01145060007")
	regSet("InstallLocation", "REG_SZ", dir)
	regSet("UninstallString", "REG_SZ", `"`+dst+`" --uninstall`)
	regSet("NoModify", "REG_DWORD", "1")
	regSet("NoRepair", "REG_DWORD", "1")
	desk, start := makeShortcuts(true)
	if desk || start {
		where := "• أيقونة على سطح المكتب وفي قائمة ابدأ"
		if !desk {
			where = "• أيقونة في قائمة ابدأ (اكتب «منظومة» في البحث)"
		} else if !start {
			where = "• أيقونة على سطح المكتب"
		}
		msg("✓ تم تثبيت "+appName+"\n\n"+where+"\n• التحديثات تصل تلقائياً عند فتح البرنامج\n• يعمل بدون إنترنت بعد أول دخول\n\n"+sig, 0x40)
	} else {
		msg("✓ تم تثبيت البرنامج، لكن تعذر إنشاء الاختصارات تلقائياً.\nيمكنك تشغيله من:\n"+dst+"\n\n"+sig, 0x40)
	}
}

func uninstall() {
	if msg("هل تريد إزالة "+appName+" من هذا الجهاز؟\n\nالبيانات محفوظة على السحابة ولن تتأثر.", 0x24) != 6 {
		return
	}
	for _, p := range shortcutPaths() {
		os.Remove(p)
		os.Remove(strings.TrimSuffix(p, ".lnk") + ".url")
	}
	hidden(exec.Command("reg.exe", "delete", regKey, "/f")).Run()
	msg("تمت إزالة البرنامج.\n\n"+sig, 0x40)
	hidden(exec.Command("cmd.exe", "/c", `ping 127.0.0.1 -n 3 >nul & rmdir /s /q "`+installDir()+`"`)).Start()
}

func findBrowser() string {
	for _, rel := range []string{`Microsoft\Edge\Application\msedge.exe`, `Google\Chrome\Application\chrome.exe`} {
		for _, env := range []string{"ProgramFiles", "ProgramFiles(x86)", "LOCALAPPDATA"} {
			if b := os.Getenv(env); b != "" {
				if p := filepath.Join(b, rel); exists(p) {
					return p
				}
			}
		}
	}
	return ""
}

func launch() {
	dir := installDir()
	br := findBrowser()
	if br == "" {
		msg("لم يتم العثور على متصفح Microsoft Edge أو Google Chrome على هذا الجهاز.\nسيتم فتح المنظومة في المتصفح الافتراضي.", 0x30)
		exec.Command("rundll32.exe", "url.dll,FileProtocolHandler", appURL).Start()
		return
	}
	prof := filepath.Join(dir, "profile")
	os.MkdirAll(prof, 0755)
	c := exec.Command(br, "--app="+appURL, "--user-data-dir="+prof, "--no-first-run", "--no-default-browser-check",
		"--disable-features=Translate", "--lang=ar", "--start-maximized")
	if err := c.Start(); err != nil {
		msg("تعذر تشغيل المنظومة:\n"+err.Error(), 0x10)
	}
}

func main() {
	for _, a := range os.Args[1:] {
		switch a {
		case "--uninstall":
			uninstall()
			return
		case "--shortcuts":
			d, s := makeShortcuts(true)
			if d || s {
				msg("✓ تم إنشاء الاختصارات.\n\n"+sig, 0x40)
			} else {
				msg("تعذر إنشاء الاختصارات.\n\n"+sig, 0x10)
			}
			return
		}
	}
	self, _ := os.Executable()
	dst := filepath.Join(installDir(), exeName)
	if !strings.EqualFold(filepath.Clean(self), filepath.Clean(dst)) {
		install()
	} else {
		os.Remove(dst + ".old")
		if !exists(filepath.Join(installDir(), "app.ico")) {
			os.WriteFile(filepath.Join(installDir(), "app.ico"), icoData, 0644)
		}
		makeShortcuts(false) // تصليح ذاتي: لو الاختصار اتمسح أو ماتعملش وقت التثبيت
	}
	launch()
}
