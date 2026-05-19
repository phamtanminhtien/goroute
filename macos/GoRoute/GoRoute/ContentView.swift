import AppKit
import SwiftUI

struct ContentView: View {
    @ObservedObject var model: StatusModel

    var body: some View {
        VStack(alignment: .leading, spacing: 16) {
            header

            if model.isLoading && model.snapshot == nil {
                loadingView
            } else if let errorMessage = model.errorMessage {
                errorView(errorMessage)
            } else if let snapshot = model.snapshot {
                snapshotView(snapshot)
            } else {
                emptyView
            }

            footer
        }
        .frame(width: 508, alignment: .topLeading)
        .padding(16)
    }

    private var header: some View {
        HStack(spacing: 10) {
            Image(systemName: model.statusIconName)
                .font(.system(size: 20, weight: .semibold))
                .symbolRenderingMode(.hierarchical)

            VStack(alignment: .leading, spacing: 2) {
                Text("GoRoute")
                    .font(.system(size: 16, weight: .semibold))
                Text(statusText)
                    .font(.caption)
                    .foregroundStyle(.secondary)
            }

            Spacer()

            Button {
                Task {
                    await model.refresh()
                }
            } label: {
                Image(systemName: "arrow.clockwise")
                    .font(.system(size: 13, weight: .semibold))
                    .rotationEffect(.degrees(model.isLoading ? 360 : 0))
                    .animation(
                        model.isLoading
                            ? .linear(duration: 0.8).repeatForever(autoreverses: false)
                            : .default,
                        value: model.isLoading
                    )
            }
            .buttonStyle(.borderless)
            .disabled(model.isLoading)
        }
    }

    private var loadingView: some View {
        HStack(spacing: 10) {
            ProgressView()
                .controlSize(.small)
            Text("Refreshing proxy status")
                .font(.callout)
                .foregroundStyle(.secondary)
        }
        .frame(maxWidth: .infinity, alignment: .leading)
        .padding(.vertical, 24)
    }

    private func errorView(_ message: String) -> some View {
        VStack(alignment: .leading, spacing: 8) {
            Label("Proxy unavailable", systemImage: "exclamationmark.triangle.fill")
                .font(.system(size: 13, weight: .semibold))
                .foregroundStyle(.red)
            Text(message)
                .font(.caption)
                .foregroundStyle(.secondary)
                .textSelection(.enabled)
        }
        .padding(12)
        .frame(maxWidth: .infinity, alignment: .leading)
        .background(.red.opacity(0.08), in: RoundedRectangle(cornerRadius: 12, style: .continuous))
    }

    private var emptyView: some View {
        VStack(alignment: .leading, spacing: 6) {
            Text("No snapshot loaded")
                .font(.system(size: 13, weight: .semibold))
            Text("Click the menu bar item to fetch the current proxy status and quota snapshot.")
                .font(.caption)
                .foregroundStyle(.secondary)
        }
        .padding(.vertical, 18)
    }

    private func snapshotView(_ snapshot: GoRouteSnapshot) -> some View {
        VStack(alignment: .leading, spacing: 14) {
            metricsGrid(snapshot.usage)

            VStack(alignment: .leading, spacing: 8) {
                Text("Account Quotas")
                    .font(.system(size: 12, weight: .semibold))
                    .foregroundStyle(.secondary)

                if snapshot.quotas.isEmpty {
                    Text("No enabled account connections found.")
                        .font(.caption)
                        .foregroundStyle(.secondary)
                } else {
                    LazyVGrid(columns: twoColumns, alignment: .leading, spacing: 10) {
                        ForEach(snapshot.quotas) { quota in
                            quotaCard(quota)
                        }
                    }
                }
            }
        }
    }

    private func metricsGrid(_ usage: UsageSummaryResponse) -> some View {
        LazyVGrid(columns: fourColumns, spacing: 8) {
            metricCard("Requests", formatInteger(usage.requests.value), "arrow.up.arrow.down")
            metricCard("Input", formatCompact(usage.inputTokens.value), "text.alignleft")
            metricCard("Output", formatCompact(usage.outputTokens.value), "text.alignright")
            metricCard("Cost", formatCurrency(usage.estimatedCostUSD.value), "dollarsign.circle")
        }
    }

    private func metricCard(_ label: String, _ value: String, _ icon: String) -> some View {
        VStack(alignment: .leading, spacing: 8) {
            Image(systemName: icon)
                .font(.system(size: 13, weight: .semibold))
                .foregroundStyle(.secondary)
            Text(value)
                .font(.system(size: 17, weight: .semibold, design: .rounded))
                .lineLimit(1)
                .minimumScaleFactor(0.7)
            Text(label)
                .font(.caption2)
                .foregroundStyle(.secondary)
                .lineLimit(1)
        }
        .padding(10)
        .frame(maxWidth: .infinity, alignment: .leading)
        .background(.quaternary.opacity(0.42), in: RoundedRectangle(cornerRadius: 12, style: .continuous))
    }

    private func quotaCard(_ quota: ConnectionQuotaSnapshot) -> some View {
        return VStack(alignment: .leading, spacing: 10) {
            HStack(alignment: .top) {
                VStack(alignment: .leading, spacing: 2) {
                    Text(quota.name)
                        .font(.system(size: 13, weight: .semibold))
                        .lineLimit(1)
                    Text("\(quota.providerID) · \(quota.id)")
                        .font(.caption2)
                        .foregroundStyle(.secondary)
                        .lineLimit(1)
                }

                Spacer()

                statusPill(quota.enabled ? "Enabled" : "Disabled", tint: quota.enabled ? .green : .secondary)
            }

            if let errorMessage = quota.errorMessage {
                Text(errorMessage)
                    .font(.caption)
                    .foregroundStyle(.red)
                    .lineLimit(2)
            } else if let usage = quota.usage {
                if let message = usage.message {
                    Text(message)
                        .font(.caption)
                        .foregroundStyle(.secondary)
                } else {
                    HStack(spacing: 6) {
                        if let plan = usage.plan, !plan.isEmpty {
                            statusPill("Plan \(plan)", tint: .blue)
                        }
                        statusPill(usage.limitReached ? "Normal limited" : "Normal ready", tint: usage.limitReached ? .orange : .green)
                    }

                    ForEach(quotaRows(from: usage), id: \.key) { row in
                        quotaRow(label: row.label, quota: row.quota)
                    }
                }
            }
        }
        .padding(10)
        .frame(maxWidth: .infinity, alignment: .leading)
        .background(.quaternary.opacity(0.32), in: RoundedRectangle(cornerRadius: 12, style: .continuous))
    }

    private func statusPill(_ status: String, tint: Color) -> some View {
        Text(status)
            .font(.caption2.weight(.semibold))
            .padding(.horizontal, 7)
            .padding(.vertical, 3)
            .foregroundStyle(tint)
            .background(tint.opacity(0.14), in: Capsule())
    }

    private func quotaRow(label: String, quota: ProviderUsageQuotaWindow) -> some View {
        let tint = quotaWindowTint(quota)

        return VStack(alignment: .leading, spacing: 8) {
            HStack {
                Text(label)
                    .font(.system(size: 10, weight: .semibold))
                    .tracking(1.4)
                    .textCase(.uppercase)
                    .foregroundStyle(.secondary)
                Spacer()
                Text(formatCountdown(quota.resetAt))
                    .font(.system(size: 10, weight: .medium))
                    .foregroundStyle(.secondary)
            }

            HStack(alignment: .bottom, spacing: 10) {
                VStack(alignment: .leading, spacing: 2) {
                    Text("\(quota.remaining)%")
                        .font(.system(size: 24, weight: .semibold, design: .rounded))
                        .foregroundStyle(tint)
                        .lineLimit(1)
                        .minimumScaleFactor(0.75)
                    Text("remaining quota")
                        .font(.system(size: 10))
                        .foregroundStyle(.secondary)
                }

                Spacer()

                VStack(alignment: .trailing, spacing: 2) {
                    Text(quota.unlimited ? "Unlimited" : "\(quota.remaining)/\(quota.total)")
                        .font(.system(size: 11, weight: .semibold))
                    Text(quota.unlimited ? "No cap reported" : "\(quota.total) total")
                        .font(.system(size: 10))
                        .foregroundStyle(.secondary)
                }
            }

            ProgressView(value: quota.unlimited ? 100 : Double(quota.remaining), total: 100)
                .controlSize(.small)
                .tint(tint)
        }
        .padding(.top, 6)
        .overlay(alignment: .top) {
            Divider()
        }
    }

    private var footer: some View {
        HStack {
            Text(lastUpdatedText)
                .font(.caption2)
                .foregroundStyle(.secondary)

            Spacer()

            Button {
                openInSafari()
            } label: {
                Label("Safari", systemImage: "safari")
            }
            .buttonStyle(.bordered)

            Button {
                NSApp.terminate(nil)
            } label: {
                Label("Quit", systemImage: "power")
            }
            .buttonStyle(.bordered)
        }
    }

    private var statusText: String {
        if model.isLoading {
            return "Refreshing"
        }
        if model.errorMessage != nil {
            return "Offline"
        }
        if model.snapshot?.proxyOnline == true {
            return "Proxy online"
        }
        return "Ready"
    }

    private var lastUpdatedText: String {
        guard let lastUpdated = model.lastUpdated else {
            return "Not refreshed yet"
        }
        return "Updated \(lastUpdated.formatted(date: .omitted, time: .shortened))"
    }

    private func quotaRows(from usage: ProviderUsageResponse) -> [(key: String, label: String, quota: ProviderUsageQuotaWindow)] {
        let labels = [
            "session": "Session",
            "weekly": "Weekly",
            "review_session": "Review Session",
            "review_weekly": "Review Weekly",
        ]

        return (usage.quotas ?? [:])
            .map { key, quota in (key: key, label: labels[key] ?? key.replacingOccurrences(of: "_", with: " ").capitalized, quota: quota) }
            .sorted { $0.label < $1.label }
    }

    private func formatInteger(_ value: Double) -> String {
        value.formatted(.number.precision(.fractionLength(0)))
    }

    private func formatCompact(_ value: Double) -> String {
        value.formatted(.number.notation(.compactName).precision(.fractionLength(0...1)))
    }

    private func formatCurrency(_ value: Double) -> String {
        value.formatted(.currency(code: "USD").precision(.fractionLength(2...4)))
    }

    private var fourColumns: [GridItem] {
        Array(repeating: GridItem(.flexible(), spacing: 8), count: 4)
    }

    private var twoColumns: [GridItem] {
        Array(repeating: GridItem(.flexible(), spacing: 10), count: 2)
    }

    private func quotaWindowTint(_ quota: ProviderUsageQuotaWindow) -> Color {
        if quota.unlimited {
            return .blue
        }
        if quota.used >= 85 {
            return .red
        }
        if quota.used >= 60 {
            return .orange
        }
        return .green
    }

    private func formatCountdown(_ resetAt: String?) -> String {
        guard let resetAt, let date = parseResetDate(resetAt) else {
            return "No reset"
        }

        let seconds = Int(date.timeIntervalSinceNow)
        if seconds <= 0 {
            return "0m"
        }

        let minutes = seconds / 60
        let days = minutes / (24 * 60)
        let hours = (minutes % (24 * 60)) / 60
        let remainingMinutes = minutes % 60

        if days > 0 {
            return "\(days)d \(hours)h \(remainingMinutes)m"
        }
        if hours > 0 {
            return "\(hours)h \(remainingMinutes)m"
        }
        return "\(remainingMinutes)m"
    }

    private func parseResetDate(_ value: String) -> Date? {
        ISO8601DateFormatter.gorouteFractional.date(from: value)
            ?? ISO8601DateFormatter.goroute.date(from: value)
    }

    private func openInSafari() {
        guard let url = URL(string: "http://127.0.0.1:12232") else {
            return
        }

        let safariURL = URL(fileURLWithPath: "/Applications/Safari.app")
        let configuration = NSWorkspace.OpenConfiguration()
        NSWorkspace.shared.open(
            [url],
            withApplicationAt: safariURL,
            configuration: configuration
        )
    }
}

private extension ISO8601DateFormatter {
    static let goroute: ISO8601DateFormatter = {
        let formatter = ISO8601DateFormatter()
        formatter.formatOptions = [.withInternetDateTime]
        return formatter
    }()

    static let gorouteFractional: ISO8601DateFormatter = {
        let formatter = ISO8601DateFormatter()
        formatter.formatOptions = [.withInternetDateTime, .withFractionalSeconds]
        return formatter
    }()
}

struct LiquidGlassContainer<Content: View>: NSViewRepresentable {
    let content: Content

    init(@ViewBuilder content: () -> Content) {
        self.content = content()
    }

    func makeCoordinator() -> Coordinator {
        Coordinator()
    }

    func makeNSView(context: Context) -> NSView {
        let hostingView = NSHostingView(rootView: content)
        hostingView.translatesAutoresizingMaskIntoConstraints = false
        context.coordinator.hostingView = hostingView

        if #available(macOS 26.0, *) {
            let glassView = NSGlassEffectView()
            glassView.style = .regular
            glassView.cornerRadius = 22
            glassView.contentView = hostingView
            pin(hostingView, to: glassView)
            return glassView
        }

        let container = NSVisualEffectView()
        container.material = .popover
        container.blendingMode = .behindWindow
        container.state = .active
        container.addSubview(hostingView)
        pin(hostingView, to: container)
        return container
    }

    func updateNSView(_ nsView: NSView, context: Context) {
        context.coordinator.hostingView?.rootView = content
    }

    private func pin(_ child: NSView, to parent: NSView) {
        NSLayoutConstraint.activate([
            child.leadingAnchor.constraint(equalTo: parent.leadingAnchor),
            child.trailingAnchor.constraint(equalTo: parent.trailingAnchor),
            child.topAnchor.constraint(equalTo: parent.topAnchor),
            child.bottomAnchor.constraint(equalTo: parent.bottomAnchor),
        ])
    }

    final class Coordinator {
        var hostingView: NSHostingView<Content>?
    }
}
