# pkTrade build helpers.
#
# The Android library is produced by gomobile from the ./mobile package and is
# copied into the Flutter app's libs directory. That artifact is not stored in
# the repository, so `make aar` has to run before the app can be assembled.

GO ?= go
FLUTTER ?= flutter
GOMOBILE ?= gomobile
ANDROID_SDK ?= $(HOME)/Android/Sdk
ANDROID_NDK ?= $(ANDROID_SDK)/ndk/28.2.13676358
ANDROID_API ?= 21
AAR_DIR := pktrade_app/android/app/libs

.PHONY: all build test vet check aar apk clean

all: build test

build:
	$(GO) build ./...

test:
	$(GO) test ./...

vet:
	$(GO) vet ./...

check: vet test
	cd pktrade_app && $(FLUTTER) analyze
	cd pktrade_app && $(FLUTTER) test

aar:
	@mkdir -p $(AAR_DIR)
	ANDROID_HOME=$(ANDROID_SDK) ANDROID_NDK_HOME=$(ANDROID_NDK) \
		$(GOMOBILE) bind -target=android -androidapi $(ANDROID_API) \
		-o $(AAR_DIR)/mobile.aar ./mobile

apk: aar
	cd pktrade_app && $(FLUTTER) build apk --debug

clean:
	$(GO) clean
	rm -f $(AAR_DIR)/mobile.aar
