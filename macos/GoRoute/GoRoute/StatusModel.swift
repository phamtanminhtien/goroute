import AppKit
import Combine
import Foundation

@MainActor
final class StatusModel: ObservableObject {
    private let client = GoRouteClient()

    @Published private(set) var snapshot: GoRouteSnapshot?
    @Published private(set) var isLoading = false
    @Published private(set) var errorMessage: String?
    @Published private(set) var lastUpdated: Date?

    var statusIconName: String {
        if isLoading {
            return "arrow.triangle.2.circlepath"
        }
        if errorMessage != nil {
            return "exclamationmark.triangle"
        }
        if snapshot?.proxyOnline == true {
            return "checkmark"
        }
        return "point.3.connected.trianglepath.dotted"
    }

    var statusColor: NSColor? {
        if isLoading {
            return .systemBlue
        }
        if errorMessage != nil {
            return .systemRed
        }
        if snapshot?.proxyOnline == true {
            return .systemGreen
        }
        return nil
    }

    var statusTooltip: String {
        if isLoading {
            return "GoRoute is refreshing"
        }
        if let errorMessage {
            return "GoRoute error: \(errorMessage)"
        }
        if snapshot?.proxyOnline == true {
            return "GoRoute proxy is online"
        }
        return "GoRoute"
    }

    func refresh() async {
        isLoading = true
        errorMessage = nil

        do {
            snapshot = try await client.loadSnapshot()
            lastUpdated = Date()
        } catch {
            errorMessage = GoRouteClient.describe(error)
            lastUpdated = Date()
        }

        isLoading = false
    }
}

struct GoRouteSnapshot {
    let proxyOnline: Bool
    let usage: UsageSummaryResponse
    let quotas: [ConnectionQuotaSnapshot]
}

struct ConnectionQuotaSnapshot: Identifiable {
    let enabled: Bool
    let id: String
    let name: String
    let providerID: String
    let status: String
    let usage: ProviderUsageResponse?
    let errorMessage: String?
}
