#!/usr/bin/env python3
"""A bare WebKitGTK window set up as Wails sets up the numen window (hardware acceleration always).

It loads a URL and runs a page script once the page has loaded. Lines written to the FIFO named by
MINI_FIFO are run in the page too, and a line that starts with @ names a file whose text is run.
What the page posts through window.webkit.messageHandlers.log is printed with a PAGE prefix.

Usage: browser.py URL SCRIPT
"""
import os
import sys

import gi

gi.require_version('Gtk', '4.0')
gi.require_version('WebKit', '6.0')
from gi.repository import GLib, Gtk, WebKit  # noqa: E402

url = sys.argv[1]
with open(sys.argv[2], encoding='utf-8') as handle:
    script = handle.read()

app = Gtk.Application(application_id='md.numen.epubframes')


def run_script(view, text):
    def finish(source, result):
        try:
            source.evaluate_javascript_finish(result)
        except GLib.Error as error:
            print('SCRIPT-ERROR', error.message, flush=True)

    view.evaluate_javascript(text, -1, None, None, None, finish)


def watch_fifo(view):
    path = os.environ.get('MINI_FIFO')
    if not path:
        return
    descriptor = os.open(path, os.O_RDWR | os.O_NONBLOCK)
    pending = b''

    def read_lines(_descriptor, _condition):
        nonlocal pending
        pending += os.read(descriptor, 65536)
        while b'\n' in pending:
            line, pending = pending.split(b'\n', 1)
            text = line.decode()
            if text.startswith('@'):
                with open(text[1:], encoding='utf-8') as handle:
                    text = handle.read()
            run_script(view, text)
        return True

    GLib.io_add_watch(descriptor, GLib.IO_IN, read_lines)


def activate(application):
    window = Gtk.ApplicationWindow(application=application, default_width=1400, default_height=900)
    WebKit.NetworkSession.get_default().set_tls_errors_policy(WebKit.TLSErrorsPolicy.IGNORE)
    manager = WebKit.UserContentManager()
    manager.register_script_message_handler('log', None)
    manager.connect('script-message-received::log', lambda _m, value: print('PAGE', value.to_string(), flush=True))
    view = WebKit.WebView(user_content_manager=manager)
    settings = view.get_settings()
    settings.set_hardware_acceleration_policy(WebKit.HardwareAccelerationPolicy.ALWAYS)
    view.set_settings(settings)

    def on_load(_view, event):
        if event == WebKit.LoadEvent.FINISHED:
            window.set_focus(view)
            view.grab_focus()
            run_script(view, script)

    view.connect('load-changed', on_load)
    window.set_child(view)
    view.load_uri(url)
    window.present()
    watch_fifo(view)


app.connect('activate', activate)
app.run([])
