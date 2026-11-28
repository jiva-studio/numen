package md.numen.spike;

import android.util.Log;

import com.getcapacitor.JSObject;
import com.getcapacitor.Plugin;
import com.getcapacitor.PluginCall;
import com.getcapacitor.PluginMethod;
import com.getcapacitor.annotation.CapacitorPlugin;

/** NumenCorePlugin starts the Go core and says where the page may reach it. */
@CapacitorPlugin(name = "NumenCore")
public class NumenCorePlugin extends Plugin {

    private static final String TAG = "NumenCore";

    private final bind.Core core = bind.Bind.newCore();

    @Override
    public void load() {
        core.setKeyVault(new AndroidKeyVault(getContext()));
    }

    @PluginMethod
    public void start(PluginCall call) {
        // Opening a vault reads every file in it, so it happens off the thread
        // that draws.
        new Thread(() -> {
            try {
                String dir = getContext().getFilesDir().getAbsolutePath();
                long port = core.start(dir);
                JSObject said = new JSObject();
                said.put("port", port);
                said.put("token", core.token());
                said.put("dir", dir);
                call.resolve(said);
            } catch (Exception why) {
                call.reject(why.getMessage(), why);
            }
        }).start();
    }

    @PluginMethod
    public void stop(PluginCall call) {
        new Thread(() -> {
            try {
                core.stop();
                call.resolve();
            } catch (Exception why) {
                call.reject(why.getMessage(), why);
            }
        }).start();
    }

    /** Tells the plugins of the core that the application left the screen. */
    @PluginMethod
    public void suspend(PluginCall call) {
        follow(false, call);
    }

    /** Tells the plugins of the core that the application is on the screen again. */
    @PluginMethod
    public void resume(PluginCall call) {
        follow(true, call);
    }

    /** Called by the activity as it comes to the screen and leaves it. */
    void follow(boolean isOnScreen) {
        follow(isOnScreen, null);
    }

    // Suspend waits for the last pass of the sync, so neither call runs on the thread that draws.
    private void follow(boolean isOnScreen, PluginCall call) {
        new Thread(() -> {
            try {
                if (isOnScreen) {
                    core.resume();
                } else {
                    core.suspend();
                }
                if (call != null) {
                    call.resolve();
                }
            } catch (Exception why) {
                Log.w(TAG, "the plugins did not follow the screen", why);
                if (call != null) {
                    call.reject(why.getMessage(), why);
                }
            }
        }).start();
    }
}
