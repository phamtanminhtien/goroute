import Foundation

struct GoRouteClient {
    private let baseURL = URL(string: "http://127.0.0.1:12232")!
    private let adminToken = "change-me"
    private let session = URLSession.shared

    func loadSnapshot() async throws -> GoRouteSnapshot {
        let healthResponse: HealthResponse = try await get("/healthz", requiresAuth: false)
        let usageResponse: UsageSummaryResponse = try await get(
            "/admin/api/analytics/usage/summary\(usageQueryString())",
            requiresAuth: true
        )
        let providerResponse: ProviderListResponse = try await get("/admin/api/providers", requiresAuth: true)

        let connections = providerResponse.data
            .filter { $0.id == "cx" }
            .flatMap { provider in
                provider.connections.map { connection in
                    ProviderConnectionContext(providerID: provider.id, connection: connection)
                }
            }

        let quotas = await withTaskGroup(of: ConnectionQuotaSnapshot.self) { group in
            for context in connections {
                group.addTask {
                    await loadQuota(for: context)
                }
            }

            var values: [ConnectionQuotaSnapshot] = []
            for await value in group {
                values.append(value)
            }
            return values.sorted { $0.name.localizedCaseInsensitiveCompare($1.name) == .orderedAscending }
        }

        return GoRouteSnapshot(
            proxyOnline: healthResponse.status == "ok",
            usage: usageResponse,
            quotas: quotas
        )
    }

    private func loadQuota(for context: ProviderConnectionContext) async -> ConnectionQuotaSnapshot {
        do {
            let usage: ProviderUsageResponse = try await get(
                "/admin/api/connections/\(context.connection.id.addingPercentEncoding(withAllowedCharacters: .urlPathAllowed) ?? context.connection.id)/usage",
                requiresAuth: true
            )
            return ConnectionQuotaSnapshot(
                enabled: context.connection.enabled ?? true,
                id: context.connection.id,
                name: context.connection.name,
                providerID: context.providerID,
                status: context.connection.status,
                usage: usage,
                errorMessage: nil
            )
        } catch {
            return ConnectionQuotaSnapshot(
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
            throw GoRouteClientError.invalidURL(path)
        }

        var request = URLRequest(url: resolvedURL)
        request.setValue("application/json", forHTTPHeaderField: "Accept")
        if requiresAuth {
            request.setValue("Bearer \(adminToken)", forHTTPHeaderField: "Authorization")
        }

        let (data, response) = try await session.data(for: request)
        guard let httpResponse = response as? HTTPURLResponse else {
            throw GoRouteClientError.invalidResponse
        }
        guard (200..<300).contains(httpResponse.statusCode) else {
            throw GoRouteClientError.httpStatus(httpResponse.statusCode, decodeErrorMessage(from: data))
        }

        return try JSONDecoder().decode(T.self, from: data)
    }

    private func usageQueryString() -> String {
        let to = Date()
        let from = Calendar.current.date(byAdding: .hour, value: -24, to: to) ?? to
        return "?from=\(formatUsageDate(from))&to=\(formatUsageDate(to))"
    }

    private func formatUsageDate(_ date: Date) -> String {
        ISO8601DateFormatter.goroute.string(from: date)
    }

    private func decodeErrorMessage(from data: Data) -> String? {
        if let envelope = try? JSONDecoder().decode(ErrorEnvelope.self, from: data) {
            return envelope.error?.message ?? envelope.message
        }
        return String(data: data, encoding: .utf8)
    }

    static func describe(_ error: Error) -> String {
        if let clientError = error as? GoRouteClientError {
            return clientError.localizedDescription
        }
        if let decodingError = error as? DecodingError {
            return "decode response: \(decodingError)"
        }
        return error.localizedDescription
    }
}

enum GoRouteClientError: LocalizedError {
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

private struct ProviderConnectionContext {
    let providerID: String
    let connection: ProviderConnection
}

struct HealthResponse: Decodable {
    let status: String
}

struct UsageSummaryResponse: Decodable {
    let estimatedCostUSD: UsageMetric
    let inputTokens: UsageMetric
    let outputTokens: UsageMetric
    let requests: UsageMetric

    enum CodingKeys: String, CodingKey {
        case estimatedCostUSD = "estimated_cost_usd"
        case inputTokens = "input_tokens"
        case outputTokens = "output_tokens"
        case requests
    }
}

struct UsageMetric: Decodable {
    let value: Double
}

struct ProviderListResponse: Decodable {
    let data: [ProviderItem]
}

struct ProviderItem: Decodable {
    let id: String
    let connections: [ProviderConnection]

    enum CodingKeys: String, CodingKey {
        case id
        case connections
    }

    init(from decoder: Decoder) throws {
        let container = try decoder.container(keyedBy: CodingKeys.self)
        id = try container.decode(String.self, forKey: .id)
        connections = try container.decodeIfPresent([ProviderConnection].self, forKey: .connections) ?? []
    }
}

struct ProviderConnection: Decodable {
    let enabled: Bool?
    let id: String
    let name: String
    let status: String
}

struct ProviderUsageResponse: Decodable {
    let limitReached: Bool
    let message: String?
    let plan: String?
    let quotas: [String: ProviderUsageQuotaWindow]?
    let reviewLimitReached: Bool
}

struct ProviderUsageQuotaWindow: Decodable {
    let remaining: Int
    let resetAt: String?
    let total: Int
    let unlimited: Bool
    let used: Int
}

private struct ErrorEnvelope: Decodable {
    let error: ErrorBody?
    let message: String?
}

private struct ErrorBody: Decodable {
    let message: String?
}

private extension ISO8601DateFormatter {
    static let goroute: ISO8601DateFormatter = {
        let formatter = ISO8601DateFormatter()
        formatter.formatOptions = [.withInternetDateTime]
        return formatter
    }()
}
