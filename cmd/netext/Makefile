# go/Makefile
APPLE_OUT=$(PWD)/out/apple


all: macos ios
	rm -rf $(APPLE_OUT)/*.h $(APPLE_OUT)/macos_x86_64.a $(APPLE_OUT)/macos_arm64.a $(APPLE_OUT)/ios_x86_64.a $(APPLE_OUT)/ios_arm64.a

ios-arm64:
	cd ../.. && \
	CGO_ENABLED=1 \
	GOOS=ios \
	GOARCH=arm64 \
	SDK=iphoneos \
	CC=$(PWD)/clangwrap.sh \
	CGO_CFLAGS="-fembed-bitcode" \
	go build -buildmode=c-archive -tags ios,ts_macext -o $(APPLE_OUT)/ios_arm64.a tailscale.com/cmd/netext

ios-x86_64:
	cd ../.. && \
	CGO_ENABLED=1 \
	GOOS=ios \
	GOARCH=amd64 \
	SDK=iphonesimulator \
	CC=$(PWD)/clangwrap.sh \
	go build -buildmode=c-archive -tags ios,ts_macext -o $(APPLE_OUT)/ios_x86_64.a tailscale.com/cmd/netext

ios: ios-arm64 ios-x86_64
	lipo $(APPLE_OUT)/ios_x86_64.a $(APPLE_OUT)/ios_arm64.a -create -output $(APPLE_OUT)/ios_libmira.o

macos-x86_64:
	cd ../.. && \
	CGO_ENABLED=1 \
	GOOS=darwin \
	GOARCH=amd64 \
	MACOSX_DEPLOYMENT_TARGET=10.6 \
	go build -buildmode=c-archive -tags ts_macext -o $(APPLE_OUT)/macos_x86_64.a tailscale.com/cmd/netext

macos-arm64:
	cd ../.. && \
	CGO_ENABLED=1 \
	GOOS=darwin \
	GOARCH=arm64 \
	MACOSX_DEPLOYMENT_TARGET=12.0 \
	go build -buildmode=c-archive -tags ts_macext -o $(APPLE_OUT)/macos_arm64.a tailscale.com/cmd/netext

macos: macos-x86_64 macos-arm64
	lipo $(APPLE_OUT)/macos_x86_64.a $(APPLE_OUT)/macos_arm64.a -create -output $(APPLE_OUT)/macos_libmira.o


# https://rogchap.com/2020/09/14/running-go-code-on-ios-and-android/
# State file location:
# macos, ios: $HOME/Library/Application Support/tsnet-a.out
# Windows: %AppData%
