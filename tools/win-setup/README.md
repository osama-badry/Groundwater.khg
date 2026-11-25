# برنامج تثبيت ويندوز
بناء: `rsrc -manifest app.manifest -ico app.ico -arch amd64 -o rsrc_windows_amd64.syso && GOOS=windows GOARCH=amd64 go build -trimpath -ldflags "-s -w -H windowsgui" -o ../../WellsKharga-Setup.exe .`

جميع الحقوق محفوظة للمصمم والمطور - م/ أسامه حسن بدري - 01145060007
