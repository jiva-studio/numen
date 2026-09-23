{
  lib,
  stdenv,
  fetchurl,
  autoPatchelfHook,
  wrapGAppsHook4,
  cairo,
  gdk-pixbuf,
  glib,
  glib-networking,
  graphene,
  gst_all_1,
  gtk4,
  harfbuzz,
  libsoup_3,
  libx11,
  pango,
  vulkan-loader,
  webkitgtk_6_0,
  assets,
}:

# Which build this is, and what it hashes to. `nix/update.sh` writes it.
let
  release = lib.importJSON ./release.json;
in

stdenv.mkDerivation {
  pname = "numen";
  version = release.version;

  src = fetchurl {
    url = release.url;
    hash = release.hash;
  };

  # The archive holds the two binaries and nothing around them.
  sourceRoot = ".";

  nativeBuildInputs = [
    autoPatchelfHook
    wrapGAppsHook4
  ];

  # The first of these are what the window is linked against. The plugins after
  # them are what the browser plays sound and video with, and it opens those by
  # name at the moment it needs them: they are here so that their own hook puts
  # them on the path the wrapper carries.
  buildInputs = [
    cairo
    gdk-pixbuf
    glib
    graphene
    gtk4
    harfbuzz
    libsoup_3
    libx11
    pango
    stdenv.cc.cc.lib
    vulkan-loader
    webkitgtk_6_0
  ]
  ++ (with gst_all_1; [
    gstreamer
    gst-plugins-base
    gst-plugins-good
    gst-plugins-bad
    gst-libav
  ]);

  installPhase = ''
    runHook preInstall

    install -Dm755 numen -t $out/bin
    install -Dm755 numen-flashcards -t $out/bin

    install -Dm644 ${assets}/numen.desktop -t $out/share/applications
    install -Dm644 ${assets}/numen-flashcards.desktop -t $out/share/applications

    install -Dm644 ${assets}/numen.svg \
      $out/share/icons/hicolor/scalable/apps/numen.svg
    install -Dm644 ${assets}/numen-flashcards.svg \
      $out/share/icons/hicolor/scalable/apps/numen-flashcards.svg

    for size in 16 24 32 48 64 128 256 512; do
      install -Dm644 ${assets}/icons/$size.png \
        $out/share/icons/hicolor/''${size}x''${size}/apps/numen.png
      install -Dm644 ${assets}/icons/flashcards/$size.png \
        $out/share/icons/hicolor/''${size}x''${size}/apps/numen-flashcards.png
    done

    runHook postInstall
  '';

  # TLS through the browser's network backend, and the C++ runtime that the
  # model runtime asks the loader for once it has a model to run.
  preFixup = ''
    gappsWrapperArgs+=(
      --prefix GIO_EXTRA_MODULES : "${glib-networking}/lib/gio/modules"
      --prefix LD_LIBRARY_PATH : "${lib.makeLibraryPath [ stdenv.cc.cc.lib ]}"
    )
  '';

  meta = {
    description = "Notes, a hierarchy with several parents, and the sources under them";
    homepage = "https://numen.md";
    downloadPage = "https://numen.md/#get";
    license = lib.licenses.unfree;
    sourceProvenance = [ lib.sourceTypes.binaryNativeCode ];
    mainProgram = "numen";
    platforms = [ "x86_64-linux" ];
  };
}
