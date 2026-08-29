package md.numen.spike;

import com.getcapacitor.JSObject;
import com.getcapacitor.Plugin;
import com.getcapacitor.PluginCall;
import com.getcapacitor.PluginMethod;
import com.getcapacitor.annotation.CapacitorPlugin;

/** NumenCorePlugin starts the Go core and says where the page may reach it. */
@CapacitorPlugin(name = "NumenCore")
public class NumenCorePlugin extends Plugin {

    @PluginMethod
    public void start(PluginCall call) {
        // Opening a vault reads every file in it, so it happens off the thread
        // that draws.
        new Thread(() -> {
            try {
                String dir = getContext().getFilesDir().getAbsolutePath();
                long port = bind.Bind.start(dir);
                JSObject said = new JSObject();
                said.put("port", port);
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
                bind.Bind.stop();
                call.resolve();
            } catch (Exception why) {
                call.reject(why.getMessage(), why);
            }
        }).start();
    }
}
