#!/usr/bin/env bash
# Runs inside the packaging/appimage container with the repo at /src/deckhud
# and a prebuilt static bin/deckhudd. Produces deckhud-<version>-x86_64.AppImage.
set -euo pipefail
cd /src/deckhud
version=${VERSION:-dev}
out=build-appimage
rm -rf "$out" && mkdir -p "$out"
appdir=$PWD/$out/AppDir

cmake -S launcher -B "$out/launcher" -G Ninja -DCMAKE_BUILD_TYPE=Release -DCMAKE_INSTALL_PREFIX=/usr
cmake --build "$out/launcher"
DESTDIR=$appdir cmake --install "$out/launcher"

install -Dm755 bin/deckhudd "$appdir/usr/bin/deckhudd"
install -Dm644 -t "$appdir/usr/share/deckhud/ui" ui/main.qml
install -Dm644 -t "$appdir/usr/share/deckhud/ui/components" ui/components/*.qml
install -Dm644 data/io.github.RustyRaptor.deckhud.metainfo.xml \
    "$appdir/usr/share/metainfo/io.github.RustyRaptor.deckhud.appdata.xml"

# linuxdeploy-plugin-qt skips the Wayland client buffer plugins; without them
# Qt Quick falls back to software buffers. Copy them in so their deps resolve.
qtplugins=$(qmake6 -query QT_INSTALL_PLUGINS)
mkdir -p "$appdir/usr/plugins"
cp -r "$qtplugins/wayland-graphics-integration-client" "$appdir/usr/plugins/"

export QMAKE=/usr/bin/qmake6
export QML_SOURCES_PATHS=$appdir/usr/share/deckhud/ui   # installed UI only, not ui/tests
export EXTRA_QT_MODULES="waylandclient"
export EXTRA_PLATFORM_PLUGINS="libqwayland-egl.so;libqwayland-generic.so"
export LDAI_OUTPUT="deckhud-$version-x86_64.AppImage"
export LDAI_UPDATE_INFORMATION="gh-releases-zsync|RustyRaptor|deckhud|latest|deckhud-*x86_64.AppImage.zsync"

cd "$out"
linuxdeploy --appdir "$appdir" \
    --executable "$appdir/usr/bin/deckhud-ui" \
    --desktop-file ../data/io.github.RustyRaptor.deckhud.desktop \
    --icon-file ../data/io.github.RustyRaptor.deckhud.svg \
    --plugin qt
# The copied plugins keep Debian's library path; point them at the bundle.
for so in "$appdir"/usr/plugins/wayland-graphics-integration-client/*.so; do
    patchelf --set-rpath '$ORIGIN/../../lib' "$so"
done
linuxdeploy --appdir "$appdir" --output appimage
ls -la ./*.AppImage
