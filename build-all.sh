#! /usr/bin/env sh

BUILDDIR="build"

[ "$1" = "clean" ] && echo "Cleaning $BUILDDIR" && rm -rf "$BUILDDIR" && exit 0

build() {
	# [linux|dragonfly|freebsd|netbsd|openbsd|plan9|solaris|darwin|windows]"
	OS="$1"
	# [arm|arm64|ppc64|ppc64le|mips64|386|amd64]
	ARCH="$2"

	echo "Building for $OS on $ARCH"

	[ "$OS" = "windows" ] && EXT=".exe"

	mkdir -p "$BUILDDIR"

	env GOOS="$OS" GOARCH="$ARCH" CGO_ENABLED=0 go build -ldflags "-s -w" -o "$BUILDDIR/$OS-$ARCH$EXT"
}

build linux 386
build linux amd64
build linux arm
build linux arm64
build darwin amd64
build windows 386
build windows amd64
