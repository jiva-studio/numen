package md.numen.spike;

import android.os.Bundle;

import com.getcapacitor.BridgeActivity;
import com.getcapacitor.PluginHandle;

public class MainActivity extends BridgeActivity {
    @Override
    public void onCreate(Bundle state) {
        registerPlugin(NumenCorePlugin.class);
        super.onCreate(state);
    }

    @Override
    public void onStart() {
        super.onStart();
        followScreen(true);
    }

    @Override
    public void onStop() {
        // A rotation recreates the activity, and that is not leaving the screen.
        if (!isChangingConfigurations()) {
            followScreen(false);
        }
        super.onStop();
    }

    private void followScreen(boolean isOnScreen) {
        if (getBridge() == null) {
            return;
        }
        PluginHandle handle = getBridge().getPlugin("NumenCore");
        if (handle != null && handle.getInstance() instanceof NumenCorePlugin) {
            ((NumenCorePlugin) handle.getInstance()).follow(isOnScreen);
        }
    }
}
