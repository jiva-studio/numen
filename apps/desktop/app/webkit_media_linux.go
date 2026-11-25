//go:build linux && cgo && !gtk3 && !android && !server && !purego

package app

/*
#cgo pkg-config: gtk4 webkitgtk-6.0
#include <gtk/gtk.h>
#include <webkit/webkit.h>

static WebKitWebView *find_web_view(GtkWidget *widget) {
	if (WEBKIT_IS_WEB_VIEW(widget)) {
		return WEBKIT_WEB_VIEW(widget);
	}
	for (GtkWidget *child = gtk_widget_get_first_child(widget); child != NULL; child = gtk_widget_get_next_sibling(child)) {
		WebKitWebView *found = find_web_view(child);
		if (found != NULL) {
			return found;
		}
	}
	return NULL;
}

// Returns 1 when the web view of the window now allows media capture, 0 when the window has none.
static int enable_media_capture(void *window) {
	WebKitWebView *view = find_web_view(GTK_WIDGET(window));
	if (view == NULL) {
		return 0;
	}
	WebKitSettings *settings = webkit_web_view_get_settings(view);
	webkit_settings_set_enable_media_stream(settings, TRUE);
	webkit_settings_set_enable_mediasource(settings, TRUE);
	return 1;
}
*/
import "C"

import (
	"unsafe"

	"github.com/wailsapp/wails/v3/pkg/application"
	"github.com/wailsapp/wails/v3/pkg/events"
)

func enableMediaCapture(app *application.App) {
	app.Window.OnCreate(func(window application.Window) {
		window.OnWindowEvent(events.Linux.WindowLoadStarted, func(*application.WindowEvent) {
			application.InvokeAsync(func() { setMediaCapture(app, window) })
		})
	})
}

func setMediaCapture(app *application.App, window application.Window) {
	native := window.NativeWindow()
	if native == nil {
		return
	}
	if C.enable_media_capture(unsafe.Pointer(native)) == 0 {
		app.Logger.Debug("media capture not enabled, the window has no web view", "window", window.ID())
		return
	}
	app.Logger.Debug("media capture enabled in the web view", "window", window.ID())
}
