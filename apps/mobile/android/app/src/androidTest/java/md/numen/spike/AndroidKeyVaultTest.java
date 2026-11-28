package md.numen.spike;

import static org.junit.Assert.assertEquals;
import static org.junit.Assert.assertFalse;
import static org.junit.Assert.assertNotEquals;
import static org.junit.Assert.assertTrue;

import android.content.Context;
import androidx.test.ext.junit.runners.AndroidJUnit4;
import androidx.test.platform.app.InstrumentationRegistry;
import java.io.File;
import java.nio.charset.StandardCharsets;
import java.nio.file.Files;
import org.junit.After;
import org.junit.Before;
import org.junit.Test;
import org.junit.runner.RunWith;

/** Runs on an emulator or a device: the Android Keystore is not there on a computer's JVM. */
@RunWith(AndroidJUnit4.class)
public class AndroidKeyVaultTest {
    private static final String SECRET = "twelve recovery words that must not be written in plain text";

    private Context context;
    private AndroidKeyVault vault;

    @Before
    public void setUp() throws Exception {
        context = InstrumentationRegistry.getInstrumentation().getTargetContext();
        vault = new AndroidKeyVault(context);
        vault.delete("identity");
    }

    @After
    public void tearDown() throws Exception {
        vault.delete("identity");
    }

    @Test
    public void aNameWithNoKeyReadsAsEmptyAndHasNothing() throws Exception {
        assertEquals("", vault.get("identity"));
        assertFalse(vault.has("identity"));
    }

    @Test
    public void aKeyIsReadBackAfterItIsSet() throws Exception {
        vault.set("identity", SECRET);

        assertTrue(vault.has("identity"));
        assertEquals(SECRET, vault.get("identity"));
    }

    @Test
    public void aKeySurvivesANewVaultOverTheSameFolder() throws Exception {
        vault.set("identity", SECRET);

        assertEquals(SECRET, new AndroidKeyVault(context).get("identity"));
    }

    @Test
    public void aSecondSetReplacesTheKey() throws Exception {
        vault.set("identity", SECRET);
        vault.set("identity", "another");

        assertEquals("another", vault.get("identity"));
    }

    @Test
    public void aDeletedKeyIsGone() throws Exception {
        vault.set("identity", SECRET);
        vault.delete("identity");

        assertFalse(vault.has("identity"));
        assertEquals("", vault.get("identity"));
    }

    @Test
    public void noFileOfTheApplicationHoldsTheKeyInPlainText() throws Exception {
        vault.set("identity", SECRET);

        assertFalse(anyFileHolds(context.getFilesDir(), SECRET));
    }

    @Test
    public void theFilesAreInTheFolderOfTheApplicationNotOnExternalStorage() throws Exception {
        vault.set("identity", SECRET);

        File external = context.getExternalFilesDir(null);
        assertTrue(external == null || !anyFileHolds(external, SECRET));
        assertNotEquals(0, new File(context.getFilesDir(), "keys").list().length);
    }

    private static boolean anyFileHolds(File folder, String text) throws Exception {
        File[] files = folder.listFiles();
        if (files == null) {
            return false;
        }
        for (File file : files) {
            if (file.isDirectory() && anyFileHolds(file, text)) {
                return true;
            }
            if (file.isFile()
                    && new String(Files.readAllBytes(file.toPath()), StandardCharsets.ISO_8859_1).contains(text)) {
                return true;
            }
        }
        return false;
    }
}
