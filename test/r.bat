@REM 自动部署 android
set GOARCH=arm64&& set GOOS=android&& set CGO_ENABLED=0&& go build -o test test.go
adb push test /data/local/tmp
adb shell "cd data/local/tmp && chmod 777 test && exit"
