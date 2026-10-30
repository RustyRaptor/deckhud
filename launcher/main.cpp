// deckhud-ui: loads the overlay QML. Replaces the system `qml` runner where
// it isn't available (the Flatpak runtime only ships it in the SDK).
// Usage: deckhud-ui <main.qml> <ws://daemon-url>
#include <QGuiApplication>
#include <QQmlApplicationEngine>
#include <QUrl>

int main(int argc, char **argv)
{
    QGuiApplication app(argc, argv);
    app.setApplicationName(QStringLiteral("deckhud"));
    app.setDesktopFileName(QStringLiteral("io.github.RustyRaptor.deckhud"));
    if (argc < 2) {
        qCritical("usage: deckhud-ui <main.qml> <ws://url>");
        return 2;
    }
    QQmlApplicationEngine engine;
    engine.load(QUrl::fromLocalFile(QString::fromLocal8Bit(argv[1])));
    if (engine.rootObjects().isEmpty())
        return 1;
    return app.exec();
}
