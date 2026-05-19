import Foundation

struct GoRouteWidgetClient {
    private let baseURL = URL(string: "http://127.0.0.1:12232")!
    private let adminToken = "change-me"
    private let session = URLSession.shared

    func loadSnapshot() async throws -> GoRouteWidgetSnapshot {
        let healthResponse: GoRouteHealthResponse = try await get("/healthz", requiresAuth: false)
        let usageResponse: GoRouteUsageSummaryResponse = try await get(
            "/admin/api/analytics/usage/summary\(usageQueryString())",
            requiresAuth: true
        )
        let providerResponse: GoRouteProviderListResponse = try await get("/admin/api/providers", requiresAuth: true)

        let connections = providerResponse.data
            .filter { $0.id == "cx" }
            .flatMap { provider in
                provider.connections.map { connection in
                    GoRouteProviderConnectionContext(providerID: provider.id, connection: connection)
                }
            }

        let quotas = await withTaskGroup(of: GoRouteConnectionQuota.self) { group in
            for context in connections {
                group.addTask {
                    await loadQuota(for: context)
                }
            }

            var values: [GoRouteConnectionQuota] = []
            for await value in group {
                values.append(value)
            }
            return values.sorted { $0.name.localizedCaseInsensitiveCompare($1.name) == .orderedAscending }
        }

        return GoRouteWidgetSnapshot(
            proxyOnline: healthResponse.status == "ok",
            usage: usageResponse,
            quotas: quotas
        )
    }

    private func loadQuota(for context: GoRouteProviderConnectionContext) async -> GoRouteConnectionQuota {
        do {
            let usage: GoRouteProviderUsageResponse = try await get(
                "/admin/api/connections/\(context.connection.id.addingPercentEncoding(withAllowedCharacters: .urlPathAllowed) ?? context.connection.id)/usage",
                requiresAuth: true
            )
            return GoRouteConnectionQuota(
                enabled: context.connection.enabled ?? true,
                id: context.connection.id,
                name: context.connection.name,
                providerID: context.providerID,
                status: context.connection.status,
                usage: usage,
                errorMessage: nil
            )
        } catch {
            return GoRouteConnectionQuota(
                enabled: context.connection.enabled ?? true,
                id: context.connection.id,
                name: context.connection.name,
                providerID: context.providerID,
                status: context.connection.status,
                usage: nil,
                errorMessage: Self.describe(error)
            )
        }
    }

    private func get<T: Decodable>(_ path: String, requiresAuth: Bool) async throws -> T {
        let normalizedPath = path.hasPrefix("/") ? path : "/\(path)"
        guard let resolvedURL = URL(string: normalizedPath, relativeTo: baseURL)?.absoluteURL else {
            throw GoRouteWidgetClientError.invalidURL(path)
        }

        var request = URLRequest(url: resolvedURL)
        request.setValue("application/json", forHTTPHeaderField: "Accept")
        if requiresAuth {
            request.setValue("Bearer \(adminToken)", forHTTPHeaderField: "Authorization")
        }

        let (data, response) = try await session.data(for: request)
        guard let httpResponse = response as? HTTPURLResponse else {
            throw GoRouteWidgetClientError.invalidResponse
        }
        guard (200..<300).contains(httpResponse.statusCode) else {
            throw GoRouteWidgetClientError.httpStatus(httpResponse.statusCode, decodeErrorMessage(from: data))
        }

        return try JSONDecoder().decode(T.self, from: data)
    }

    private func usageQueryString() -> String {
        let to = Date()
        let from = Calendar.current.date(byAdding: .hour, value: -24, to: to) ?? to
        return "?from=\(formatUsageDate(from))&to=\(formatUsageDate(to))"
    }

    private func formatUsageDate(_ date: Date) -> String {
        ISO8601DateFormatter.gorouteWidget.string(from: date)
    }

    private func decodeErrorMessage(from data: Data) -> String? {
        if let envelope = try? JSONDecoder().decode(GoRouteErrorEnvelope.self, from: data) {
            return envelope.error?.message ?? envelope.message
        }
        return String(data: data, encoding: .utf8)
    }

    static func describe(_ error: Error) -> String {
        if let clientError = error as? GoRouteWidgetClientError {
            return clientError.localizedDescription
        }
        if let decodingError = error as? DecodingError {
            return "decode response: \(decodingError)"
        }
        return error.localizedDescription
    }
}

enum GoRouteWidgetClientError: LocalizedError {
    case invalidURL(String)
    case invalidResponse
    case httpStatus(Int, String?)

    var errorDescription: String? {
        switch self {
        case .invalidURL(let path):
            return "invalid URL: \(path)"
        case .invalidResponse:
            return "invalid server response"
        case .httpStatus(let status, let message):
            if let message, !message.isEmpty {
                return "HTTP \(status): \(message)"
            }
            return "HTTP \(status)"
        }
    }
}

private struct GoRouteProviderConnectionContext {
    let providerID: String
    let connection: GoRouteProviderConnection
}

struct GoRouteWidgetSnapshot {
    let proxyOnline: Bool
    let usage: GoRouteUsageSummaryResponse
    let quotas: [GoRouteConnectionQuota]
}

struct GoRouteConnectionQuota: Identifiable {
    let enabled: Bool
    let id: String
    let name: String
    let providerID: String
    let status: String
    let usage: GoRouteProviderUsageResponse?
    let errorMessage: String?
}

struct GoRouteHealthResponse: Decodable {
    let status: String
}

struct GoRouteUsageSummaryResponse: Decodable {
    let estimatedCostUSD: GoRouteUsageMetric
    let inputTokens: GoRouteUsageMetric
    let outputTokens: GoRouteUsageMetric
    let requests: GoRouteUsageMetric

    enum CodingKeys: String, CodingKey {
        case estimatedCostUSD = "estimated_cost_usd"
        case inputTokens = "input_tokens"
        case outputTokens = "output_tokens"
        case requests
    }
}

struct GoRouteUsageMetric: Decodable {
    let value: Double
}

struct GoRouteProviderListResponse: Decodable {
    let data: [GoRouteProviderItem]
}

struct GoRouteProviderItem: Decodable {
    let id: String
    let connections: [GoRouteProviderConnection]

    enum CodingKeys: String, CodingKey {
        case id
        case connections
    }

    init(from decoder: Decoder) throws {
        let container = try decoder.container(keyedBy: CodingKeys.self)
        id = try container.decode(String.self, forKey: .id)
        connections = try container.decodeIfPresent([GoRouteProviderConnection].self, forKey: .connections) ?? []
    }
}

struct GoRouteProviderConnection: Decodable {
    let enabled: Bool?
    let id: String
    let name: String
    let status: String
}

struct GoRouteProviderUsageResponse: Decodable {
    let limitReached: Bool
    let message: String?
    let plan: String?
    let quotas: [String: GoRouteProviderUsageQuotaWindow]?
    let reviewLimitReached: Bool
}

struct GoRouteProviderUsageQuotaWindow: Decodable {
    let remaining: Int
    let resetAt: String?
    let total: Int
    let unlimited: Bool
    let used: Int
}

private struct GoRouteErrorEnvelope: Decodable {
    let error: GoRouteErrorBody?
    let message: String?
}

private struct GoRouteErrorBody: Decodable {
    let message: String?
}

private extension ISO8601DateFormatter {
    static let gorouteWidget: ISO8601DateFormatter = {
        let formatter = ISO8601DateFormatter()
        formatter.formatOptions = [.withInternetDateTime]
        return formatter
    }()
}

