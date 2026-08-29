package md.numen.spike;

import android.os.Bundle;

import com.getcapacitor.BridgeActivity;

public class MainActivity extends BridgeActivity {
    @Override
    public void onCreate(Bundle state) {
        registerPlugin(NumenCorePlugin.class);
        super.onCreate(state);
    }
}
