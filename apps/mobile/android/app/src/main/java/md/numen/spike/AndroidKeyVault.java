package md.numen.spike;

import android.content.Context;

/**
 * The keys of the identity, sealed with an AES-256-GCM key of the Android Keystore and kept as files under
 * the application's own folder.
 */
public class AndroidKeyVault {

    public AndroidKeyVault(Context context) {
        throw new UnsupportedOperationException("not implemented");
    }

    /** Returns the key under the name, and the empty string where there is none. */
    public String get(String name) throws Exception {
        throw new UnsupportedOperationException("not implemented");
    }

    public boolean has(String name) {
        throw new UnsupportedOperationException("not implemented");
    }

    public void set(String name, String value) throws Exception {
        throw new UnsupportedOperationException("not implemented");
    }

    public void delete(String name) throws Exception {
        throw new UnsupportedOperationException("not implemented");
    }
}
