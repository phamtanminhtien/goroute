import SwiftUI
import WidgetKit

@main
struct GoRouteWidgetBundle: WidgetBundle {
    var body: some Widget {
        GoRouteWidget()
    }
}

struct GoRouteWidget: Widget {
    let kind = "GoRouteWidget"

    var body: some WidgetConfiguration {
        StaticConfiguration(kind: kind, provider: GoRouteTimelineProvider()) { entry in
            GoRouteWidgetView(entry: entry)
                .containerBackground(.background, for: .widget)
                .widgetURL(URL(string: "http://127.0.0.1:12232"))
        }
        .configurationDisplayName("GoRoute")
        .description("Proxy status, usage totals, and Codex quota snapshot.")
        .supportedFamilies([.systemSmall, .systemMedium, .systemLarge])
    }
}

struct GoRouteTimelineProvider: TimelineProvider {
    func placeholder(in context: Context) -> GoRouteWidgetEntry {
        GoRouteWidgetEntry(date: Date(), snapshot: .placeholder, errorMessage: nil, isPlaceholder: true)
    }

    func getSnapshot(in context: Context, completion: @escaping (GoRouteWidgetEntry) -> Void) {
        if context.isPreview {
            completion(placeholder(in: context))
            return
        }

        Task {
            completion(await loadEntry())
        }
    }

    func getTimeline(in context: Context, completion: @escaping (Timeline<GoRouteWidgetEntry>) -> Void) {
        Task {
            let entry = await loadEntry()
            let nextRefresh = Calendar.current.date(byAdding: .minute, value: 15, to: Date()) ?? Date()
            completion(Timeline(entries: [entry], policy: .after(nextRefresh)))
        }
    }

    private func loadEntry() async -> GoRouteWidgetEntry {
        do {
            let snapshot = try await GoRouteWidgetClient().loadSnapshot()
            return GoRouteWidgetEntry(date: Date(), snapshot: snapshot, errorMessage: nil, isPlaceholder: false)
        } catch {
            return GoRouteWidgetEntry(
                date: Date(),
                snapshot: nil,
                errorMessage: GoRouteWidgetClient.describe(error),
                isPlaceholder: false
            )
        }
    }
}

struct GoRouteWidgetEntry: TimelineEntry {
    let date: Date
    let snapshot: GoRouteWidgetSnapshot?
    let errorMessage: String?
    let isPlaceholder: Bool
}

struct GoRouteWidgetView: View {
    @Environment(\.widgetFamily) private var family

    let entry: GoRouteWidgetEntry

    var body: some View {
        VStack(alignment: .leading, spacing: 10) {
            header

            if let errorMessage = entry.errorMessage {
                errorView(errorMessage)
            } else if let snapshot = entry.snapshot {
                content(snapshot)
            } else {
                errorView("No snapshot")
            }
        }
        .frame(maxWidth: .infinity, maxHeight: .infinity, alignment: .topLeading)
        .padding(widgetPadding)
    }

    private var header: some View {
        HStack(spacing: 7) {
            Image(systemName: entry.snapshot?.proxyOnline == true ? "checkmark.circle.fill" : "point.3.connected.trianglepath.dotted")
                .font(.system(size: 15, weight: .semibold))
                .foregroundStyle(entry.snapshot?.proxyOnline == true ? .green : .secondary)

            Text("GoRoute")
                .font(.system(size: 14, weight: .semibold))
                .lineLimit(1)

            Spacer(minLength: 4)

            Text(entry.date, style: .time)
                .font(.caption2)
                .foregroundStyle(.secondary)
                .lineLimit(1)
        }
    }

    @ViewBuilder
    private func content(_ snapshot: GoRouteWidgetSnapshot) -> some View {
        switch family {
        case .systemSmall:
            smallContent(snapshot)
        case .systemLarge:
            largeContent(snapshot)
        default:
            mediumContent(snapshot)
        }
    }

    private func smallContent(_ snapshot: GoRouteWidgetSnapshot) -> some View {
        VStack(alignment: .leading, spacing: 8) {
            metricLine("Requests", formatInteger(snapshot.usage.requests.value))
            metricLine("Input", formatCompact(snapshot.usage.inputTokens.value))
            metricLine("Output", formatCompact(snapshot.usage.outputTokens.value))
            Spacer(minLength: 0)
            if let quota = snapshot.quotas.first {
                compactQuota(quota)
            }
        }
    }

    private func mediumContent(_ snapshot: GoRouteWidgetSnapshot) -> some View {
        VStack(alignment: .leading, spacing: 10) {
            LazyVGrid(columns: Array(repeating: GridItem(.flexible(), spacing: 8), count: 4), spacing: 8) {
                metricTile("Requests", formatInteger(snapshot.usage.requests.value))
                metricTile("Input", formatCompact(snapshot.usage.inputTokens.value))
                metricTile("Output", formatCompact(snapshot.usage.outputTokens.value))
                metricTile("Cost", formatCurrency(snapshot.usage.estimatedCostUSD.value))
            }

            quotaStrip(snapshot.quotas, limit: 2)
        }
    }

    private func largeContent(_ snapshot: GoRouteWidgetSnapshot) -> some View {
        VStack(alignment: .leading, spacing: 10) {
            LazyVGrid(columns: Array(repeating: GridItem(.flexible(), spacing: 8), count: 4), spacing: 8) {
                metricTile("Requests", formatInteger(snapshot.usage.requests.value))
                metricTile("Input", formatCompact(snapshot.usage.inputTokens.value))
                metricTile("Output", formatCompact(snapshot.usage.outputTokens.value))
                metricTile("Cost", formatCurrency(snapshot.usage.estimatedCostUSD.value))
            }

            VStack(alignment: .leading, spacing: 7) {
                Text("Codex Quotas")
                    .font(.caption.weight(.semibold))
                    .foregroundStyle(.secondary)

                LazyVGrid(columns: Array(repeating: GridItem(.flexible(), spacing: 8), count: 2), spacing: 8) {
                    ForEach(snapshot.quotas.prefix(6)) { quota in
                        quotaTile(quota)
                    }
                }
            }
        }
    }

    private func metricLine(_ label: String, _ value: String) -> some View {
        HStack {
            Text(label)
                .font(.caption2)
                .foregroundStyle(.secondary)
            Spacer()
            Text(value)
                .font(.system(size: 14, weight: .semibold, design: .rounded))
                .lineLimit(1)
                .minimumScaleFactor(0.7)
        }
    }

    private func metricTile(_ label: String, _ value: String) -> some View {
        VStack(alignment: .leading, spacing: 3) {
            Text(value)
                .font(.system(size: 15, weight: .semibold, design: .rounded))
                .lineLimit(1)
                .minimumScaleFactor(0.65)
            Text(label)
                .font(.caption2)
                .foregroundStyle(.secondary)
                .lineLimit(1)
        }
        .frame(maxWidth: .infinity, alignment: .leading)
    }

    private func quotaStrip(_ quotas: [GoRouteConnectionQuota], limit: Int) -> some View {
        HStack(spacing: 8) {
            ForEach(quotas.prefix(limit)) { quota in
                quotaTile(quota)
            }

            if quotas.isEmpty {
                Text("No Codex accounts")
                    .font(.caption)
                    .foregroundStyle(.secondary)
            }
        }
    }

    private func compactQuota(_ quota: GoRouteConnectionQuota) -> some View {
        VStack(alignment: .leading, spacing: 5) {
            HStack {
                Text(quota.name)
                    .font(.caption.weight(.semibold))
                    .lineLimit(1)
                Spacer(minLength: 4)
                statusPill(quota.enabled ? "On" : "Off", tint: quota.enabled ? .green : .secondary)
            }

            if let row = primaryQuotaRow(quota) {
                quotaProgress(row.quota)
            } else {
                Text(quota.errorMessage ?? "No quota")
                    .font(.caption2)
                    .foregroundStyle(.secondary)
                    .lineLimit(2)
            }
        }
    }

    private func quotaTile(_ quota: GoRouteConnectionQuota) -> some View {
        VStack(alignment: .leading, spacing: 6) {
            HStack(spacing: 6) {
                Text(quota.name)
                    .font(.caption.weight(.semibold))
                    .lineLimit(1)
                Spacer(minLength: 4)
                statusPill(quota.enabled ? "Enabled" : "Disabled", tint: quota.enabled ? .green : .secondary)
            }

            if let row = primaryQuotaRow(quota) {
                Text(row.label)
                    .font(.caption2)
                    .foregroundStyle(.secondary)
                    .lineLimit(1)
                quotaProgress(row.quota)
            } else {
                Text(quota.errorMessage ?? "No quota returned")
                    .font(.caption2)
                    .foregroundStyle(.secondary)
                    .lineLimit(2)
            }
        }
        .frame(maxWidth: .infinity, alignment: .leading)
    }

    private func quotaProgress(_ quota: GoRouteProviderUsageQuotaWindow) -> some View {
        VStack(alignment: .leading, spacing: 4) {
            HStack(alignment: .firstTextBaseline) {
                Text(quota.unlimited ? "Unlimited" : "\(quota.remaining)%")
                    .font(.system(size: 15, weight: .semibold, design: .rounded))
                    .foregroundStyle(quotaWindowTint(quota))
                Spacer(minLength: 4)
                Text(quota.unlimited ? "No cap" : "\(quota.remaining)/\(quota.total)")
                    .font(.caption2)
                    .foregroundStyle(.secondary)
                    .lineLimit(1)
            }

            ProgressView(value: quota.unlimited ? 100 : Double(quota.remaining), total: 100)
                .tint(quotaWindowTint(quota))
                .controlSize(.small)
        }
    }

    private func errorView(_ message: String) -> some View {
        VStack(alignment: .leading, spacing: 7) {
            Label("Offline", systemImage: "exclamationmark.triangle.fill")
                .font(.caption.weight(.semibold))
                .foregroundStyle(.red)
            Text(message)
                .font(.caption2)
                .foregroundStyle(.secondary)
                .lineLimit(family == .systemSmall ? 3 : 5)
        }
    }

    private func statusPill(_ text: String, tint: Color) -> some View {
        Text(text)
            .font(.system(size: 9, weight: .semibold))
            .padding(.horizontal, 5)
            .padding(.vertical, 2)
            .foregroundStyle(tint)
            .background(tint.opacity(0.14), in: Capsule())
            .lineLimit(1)
    }

    private func primaryQuotaRow(_ quota: GoRouteConnectionQuota) -> (label: String, quota: GoRouteProviderUsageQuotaWindow)? {
        let rows = quotaRows(from: quota.usage)
        guard let row = rows.first(where: { $0.key == "session" }) ?? rows.first else {
            return nil
        }
        return (label: row.label, quota: row.quota)
    }

    private func quotaRows(from usage: GoRouteProviderUsageResponse?) -> [(key: String, label: String, quota: GoRouteProviderUsageQuotaWindow)] {
        let labels = [
            "session": "Session",
            "weekly": "Weekly",
            "review_session": "Review Session",
            "review_weekly": "Review Weekly",
        ]

        return (usage?.quotas ?? [:])
            .map { key, quota in (key: key, label: labels[key] ?? key.replacingOccurrences(of: "_", with: " ").capitalized, quota: quota) }
            .sorted { $0.label < $1.label }
    }

    private func quotaWindowTint(_ quota: GoRouteProviderUsageQuotaWindow) -> Color {
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

    private var widgetPadding: CGFloat {
        family == .systemSmall ? 12 : 14
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
}

private extension GoRouteWidgetSnapshot {
    static let placeholder = GoRouteWidgetSnapshot(
        proxyOnline: true,
        usage: GoRouteUsageSummaryResponse(
            estimatedCostUSD: GoRouteUsageMetric(value: 0.042),
            inputTokens: GoRouteUsageMetric(value: 142_000),
            outputTokens: GoRouteUsageMetric(value: 31_500),
            requests: GoRouteUsageMetric(value: 128)
        ),
        quotas: [
            GoRouteConnectionQuota(
                enabled: true,
                id: "codex-main",
                name: "codex-main",
                providerID: "cx",
                status: "ready",
                usage: GoRouteProviderUsageResponse(
                    limitReached: false,
                    message: nil,
                    plan: "pro",
                    quotas: [
                        "session": GoRouteProviderUsageQuotaWindow(remaining: 82, resetAt: nil, total: 100, unlimited: false, used: 18),
                        "weekly": GoRouteProviderUsageQuotaWindow(remaining: 67, resetAt: nil, total: 100, unlimited: false, used: 33),
                    ],
                    reviewLimitReached: false
                ),
                errorMessage: nil
            ),
        ]
    )
}
