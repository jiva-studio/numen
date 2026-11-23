package md.numen.spike;

import android.content.Context;
import android.os.Build;
import android.security.keystore.KeyGenParameterSpec;
import android.security.keystore.KeyProperties;
import android.security.keystore.StrongBoxUnavailableException;
import android.util.Base64;
import java.io.ByteArrayOutputStream;
import java.io.File;
import java.io.FileInputStream;
import java.io.FileOutputStream;
import java.io.IOException;
import java.nio.charset.StandardCharsets;
import java.security.KeyStore;
import java.util.Arrays;
import javax.crypto.Cipher;
import javax.crypto.KeyGenerator;
import javax.crypto.SecretKey;
import javax.crypto.spec.GCMParameterSpec;

/**
 * The keys of the identity, sealed with an AES-256-GCM key of the Android Keystore and kept as files under
 * the application's own folder. A file holds the nonce and the sealed key; the name of the key is bound into
 * the seal, so a file moved to another name does not open.
 */
public class AndroidKeyVault implements bind.KeyVault {
    private static final String KEYSTORE = "AndroidKeyStore";
    private static final String KEY_ALIAS = "numen-keyvault";
    private static final String TRANSFORMATION = "AES/GCM/NoPadding";
    private static final int NONCE_BYTES = 12;
    private static final int TAG_BITS = 128;

    private final File folder;

    public AndroidKeyVault(Context context) {
        this.folder = new File(context.getFilesDir(), "keys");
    }

    /** Returns the key under the name, and the empty string where there is none. */
    @Override
    public synchronized String get(String name) throws Exception {
        File file = fileOf(name);
        if (!file.isFile()) {
            return "";
        }
        byte[] stored = readAll(file);
        if (stored.length <= NONCE_BYTES) {
            throw new IOException("the file of the key " + name + " is too short");
        }
        Cipher cipher = Cipher.getInstance(TRANSFORMATION);
        cipher.init(Cipher.DECRYPT_MODE, getSecretKey(), new GCMParameterSpec(TAG_BITS, stored, 0, NONCE_BYTES));
        cipher.updateAAD(name.getBytes(StandardCharsets.UTF_8));
        byte[] plain = cipher.doFinal(stored, NONCE_BYTES, stored.length - NONCE_BYTES);
        try {
            return new String(plain, StandardCharsets.UTF_8);
        } finally {
            Arrays.fill(plain, (byte) 0);
        }
    }

    @Override
    public synchronized boolean has(String name) {
        return fileOf(name).isFile();
    }

    @Override
    public synchronized void set(String name, String value) throws Exception {
        Cipher cipher = Cipher.getInstance(TRANSFORMATION);
        cipher.init(Cipher.ENCRYPT_MODE, getSecretKey());
        cipher.updateAAD(name.getBytes(StandardCharsets.UTF_8));
        byte[] plain = value.getBytes(StandardCharsets.UTF_8);
        byte[] sealed = cipher.doFinal(plain);
        Arrays.fill(plain, (byte) 0);
        byte[] nonce = cipher.getIV();
        if (!folder.isDirectory() && !folder.mkdirs()) {
            throw new IOException("cannot make the folder of the keys");
        }
        File target = fileOf(name);
        File part = new File(folder, target.getName() + ".part");
        try (FileOutputStream out = new FileOutputStream(part)) {
            out.write(nonce);
            out.write(sealed);
            out.getFD().sync();
        }
        if (!part.renameTo(target)) {
            throw new IOException("cannot place the file of the key " + name);
        }
    }

    @Override
    public synchronized void delete(String name) throws Exception {
        File file = fileOf(name);
        if (file.exists() && !file.delete()) {
            throw new IOException("cannot delete the file of the key " + name);
        }
    }

    private static byte[] readAll(File file) throws IOException {
        try (FileInputStream in = new FileInputStream(file); ByteArrayOutputStream out = new ByteArrayOutputStream()) {
            byte[] buffer = new byte[1024];
            for (int read = in.read(buffer); read >= 0; read = in.read(buffer)) {
                out.write(buffer, 0, read);
            }
            return out.toByteArray();
        }
    }

    private File fileOf(String name) {
        String encoded = Base64.encodeToString(
                name.getBytes(StandardCharsets.UTF_8), Base64.URL_SAFE | Base64.NO_WRAP | Base64.NO_PADDING);
        return new File(folder, encoded + ".key");
    }

    private static SecretKey getSecretKey() throws Exception {
        KeyStore store = KeyStore.getInstance(KEYSTORE);
        store.load(null);
        if (store.containsAlias(KEY_ALIAS)) {
            return (SecretKey) store.getKey(KEY_ALIAS, null);
        }
        if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.P) {
            try {
                return generateKey(true);
            } catch (StrongBoxUnavailableException unavailable) {
                return generateKey(false);
            }
        }
        return generateKey(false);
    }

    private static SecretKey generateKey(boolean isStrongBox) throws Exception {
        KeyGenParameterSpec.Builder spec = new KeyGenParameterSpec.Builder(
                        KEY_ALIAS, KeyProperties.PURPOSE_ENCRYPT | KeyProperties.PURPOSE_DECRYPT)
                .setBlockModes(KeyProperties.BLOCK_MODE_GCM)
                .setEncryptionPaddings(KeyProperties.ENCRYPTION_PADDING_NONE)
                .setKeySize(256)
                .setRandomizedEncryptionRequired(true);
        if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.P) {
            spec.setIsStrongBoxBacked(isStrongBox).setUnlockedDeviceRequired(false);
        }
        KeyGenerator generator = KeyGenerator.getInstance(KeyProperties.KEY_ALGORITHM_AES, KEYSTORE);
        generator.init(spec.build());
        return generator.generateKey();
    }
}
