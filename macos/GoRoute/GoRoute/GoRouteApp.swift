import AppKit
import Combine
import SwiftUI

@main
struct GoRouteApp: App {
    @NSApplicationDelegateAdaptor(AppDelegate.self) private var appDelegate

    var body: some Scene {
        Settings {
            EmptyView()
        }
    }
}

@MainActor
final class AppDelegate: NSObject, NSApplicationDelegate {
    private let statusModel = StatusModel()
    private var statusItem: NSStatusItem?
    private var popover: NSPopover?
    private var cancellables = Set<AnyCancellable>()

    func applicationDidFinishLaunching(_ notification: Notification) {
        NSApp.setActivationPolicy(.accessory)

        let item = NSStatusBar.system.statusItem(withLength: NSStatusItem.variableLength)
        item.button?.image = makeStatusImage(named: "point.3.connected.trianglepath.dotted")
        item.button?.imagePosition = .imageLeading
        item.button?.title = "GoRoute"
        item.button?.target = self
        item.button?.action = #selector(togglePopover(_:))
        statusItem = item

        let popover = NSPopover()
        popover.behavior = .transient
        popover.contentSize = NSSize(width: 540, height: 560)
        popover.contentViewController = NSHostingController(
            rootView: LiquidGlassContainer {
                ContentView(model: statusModel)
            }
        )
        self.popover = popover

        statusModel.objectWillChange
            .sink { [weak self] _ in
                DispatchQueue.main.async {
                    self?.updateStatusItem()
                }
            }
            .store(in: &cancellables)

        updateStatusItem()
    }

    @objc private func togglePopover(_ sender: AnyObject?) {
        guard let button = statusItem?.button, let popover else {
            return
        }

        Task {
            await statusModel.refresh()
            updateStatusItem()
        }

        if popover.isShown {
            popover.performClose(sender)
        } else {
            NSApp.activate(ignoringOtherApps: true)
            popover.show(relativeTo: button.bounds, of: button, preferredEdge: .minY)
            popover.contentViewController?.view.window?.makeKey()
        }

        updateStatusItem()
    }

    private func updateStatusItem() {
        guard let button = statusItem?.button else {
            return
        }

        button.image = makeStatusImage(named: statusModel.statusIconName)
        button.title = "GoRoute"
        button.contentTintColor = nil
        button.toolTip = statusModel.statusTooltip
    }

    private func makeStatusImage(named name: String) -> NSImage? {
        let image = NSImage(systemSymbolName: name, accessibilityDescription: "GoRoute")
        image?.isTemplate = true
        return image
    }
}
