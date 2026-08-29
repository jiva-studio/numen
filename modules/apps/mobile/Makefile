# The phone: the core bound as a library, and a page that asks it.
#
# ANDROID_SDK_ROOT must name an SDK holding an NDK and build-tools 35.
# JAVA_HOME must name a JDK 21.

ANDROID_NDK_HOME ?= $(ANDROID_SDK_ROOT)/ndk/28.2.13676358
GOMOBILE ?= $(shell go env GOPATH)/bin/gomobile

# The architectures a phone and an emulator run.
TARGETS ?= android/arm64,android/amd64

# The core is bound against this API level, and variables.gradle says the same.
API ?= 24

.PHONY: core interface apk install run clean

## core binds the Go core as an Android library.
core:
	ANDROID_NDK_HOME=$(ANDROID_NDK_HOME) $(GOMOBILE) bind \
		-target=$(TARGETS) -androidapi $(API) -ldflags="-s -w" \
		-o android/app/libs/numen.aar ./bind

## interface builds the page and copies it into the Android project.
interface:
	npm run build
	npx cap sync android

## apk builds the debug application.
apk: core interface
	cd android && ./gradlew assembleDebug

## install puts it on whatever adb is talking to.
install: apk
	adb install -r android/app/build/outputs/apk/debug/app-debug.apk

## run installs it and opens it.
run: install
	adb shell am start -n md.numen.spike/.MainActivity

clean:
	rm -rf dist android/app/libs/numen.aar android/app/libs/numen-sources.jar
	cd android && ./gradlew clean
