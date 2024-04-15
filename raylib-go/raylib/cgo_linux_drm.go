//go:build linux && drm && !sdl && !android
// +build linux,drm,!sdl,!android

package rl

/*
#cgo linux,drm LDFLAGS: -lGLESv2 -lEGL -ldrm -lgbm -lpthread -lrt -lm -ldl
#cgo linux,drm CFLAGS: -DPLATFORM_DRM -DGRAPHICS_API_OPENGL_ES2 -DEGL_NO_X11 -I/usr/include/libdrm
*/
import "C"
