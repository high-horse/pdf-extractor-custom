

build-static:
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -a -installsuffix cgo -o bin/linux/extractor-static


build-win64:
	GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go build -o bin/win/extractor-static-wind64.exe
	
build-win32:
	GOOS=windows GOARCH=386 CGO_ENABLED=0 go build -o bin/win/extractor-static-wind32.exe

