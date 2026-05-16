export type UsageRange = "24h" | "7d" | "30d";

export type UsageMetricSnapshot = {
  deltaLabel: string;
  deltaTone: "critical" | "positive" | "warning";
  label: string;
  value: string;
};

export type UsageTimeseriesPoint = {
  cost: number;
  inputTokens: number;
  label: string;
  outputTokens: number;
  requests: number;
};

export type UsageBreakdownItem = {
  color: string;
  label: string;
  value: number;
};

export type RecentRequestItem = {
  estimatedCost: string;
  id: string;
  inputTokens: string;
  latency: string;
  model: string;
  outputTokens: string;
  route: string;
  status: "Cached" | "Completed" | "Failed" | "Streaming";
  timestamp: string;
};

export type UsageAnalyticsDataset = {
  breakdown: UsageBreakdownItem[];
  metrics: UsageMetricSnapshot[];
  recentRequests: RecentRequestItem[];
  timeseries: UsageTimeseriesPoint[];
};

export const usageRangeOptions: Array<{ label: string; value: UsageRange }> = [
  { label: "24H", value: "24h" },
  { label: "7D", value: "7d" },
  { label: "30D", value: "30d" },
];

export const usageAnalyticsDatasets: Record<UsageRange, UsageAnalyticsDataset> =
  {
    "24h": {
      breakdown: [
        { color: "#df7a52", label: "OpenAI", value: 48 },
        { color: "#9fd768", label: "Anthropic", value: 27 },
        { color: "#56d364", label: "Gemini", value: 15 },
        { color: "#f59e0b", label: "Fallback", value: 10 },
      ],
      metrics: [
        {
          deltaLabel: "+18.2% vs prev day",
          deltaTone: "positive",
          label: "Total Requests",
          value: "12,480",
        },
        {
          deltaLabel: "+9.4% context growth",
          deltaTone: "warning",
          label: "Total Input Tokens",
          value: "18.6M",
        },
        {
          deltaLabel: "+13.1% generation load",
          deltaTone: "positive",
          label: "Output Tokens",
          value: "7.9M",
        },
        {
          deltaLabel: "$0.014 avg / req",
          deltaTone: "critical",
          label: "Est. Cost",
          value: "$174.82",
        },
      ],
      recentRequests: [
        {
          estimatedCost: "$0.036",
          id: "req_01",
          inputTokens: "11.2k",
          latency: "1.2s",
          model: "gpt-5.4",
          outputTokens: "2.1k",
          route: "/v1/chat/completions",
          status: "Completed",
          timestamp: "2m ago",
        },
        {
          estimatedCost: "$0.009",
          id: "req_02",
          inputTokens: "3.6k",
          latency: "842ms",
          model: "claude-sonnet-4",
          outputTokens: "0.9k",
          route: "/v1/responses",
          status: "Streaming",
          timestamp: "6m ago",
        },
        {
          estimatedCost: "$0.001",
          id: "req_03",
          inputTokens: "0.8k",
          latency: "93ms",
          model: "gpt-4.1-mini",
          outputTokens: "0",
          route: "/v1/embeddings",
          status: "Cached",
          timestamp: "11m ago",
        },
        {
          estimatedCost: "$0.042",
          id: "req_04",
          inputTokens: "18.9k",
          latency: "2.8s",
          model: "gemini-2.5-pro",
          outputTokens: "3.4k",
          route: "/v1/chat/completions",
          status: "Completed",
          timestamp: "19m ago",
        },
        {
          estimatedCost: "$0.000",
          id: "req_05",
          inputTokens: "1.1k",
          latency: "441ms",
          model: "gpt-5.4",
          outputTokens: "0",
          route: "/v1/moderations",
          status: "Failed",
          timestamp: "27m ago",
        },
      ],
      timeseries: [
        {
          cost: 6.1,
          inputTokens: 680000,
          label: "00:00",
          outputTokens: 240000,
          requests: 420,
        },
        {
          cost: 5.4,
          inputTokens: 590000,
          label: "04:00",
          outputTokens: 210000,
          requests: 360,
        },
        {
          cost: 7.9,
          inputTokens: 910000,
          label: "08:00",
          outputTokens: 340000,
          requests: 610,
        },
        {
          cost: 9.8,
          inputTokens: 1240000,
          label: "12:00",
          outputTokens: 520000,
          requests: 860,
        },
        {
          cost: 8.7,
          inputTokens: 1160000,
          label: "16:00",
          outputTokens: 470000,
          requests: 790,
        },
        {
          cost: 10.6,
          inputTokens: 1310000,
          label: "20:00",
          outputTokens: 610000,
          requests: 980,
        },
      ],
    },
    "7d": {
      breakdown: [
        { color: "#df7a52", label: "OpenAI", value: 44 },
        { color: "#9fd768", label: "Anthropic", value: 25 },
        { color: "#56d364", label: "Gemini", value: 19 },
        { color: "#f59e0b", label: "Fallback", value: 12 },
      ],
      metrics: [
        {
          deltaLabel: "+12.6% vs previous week",
          deltaTone: "positive",
          label: "Total Requests",
          value: "78,930",
        },
        {
          deltaLabel: "+7.1% prompt depth",
          deltaTone: "warning",
          label: "Total Input Tokens",
          value: "112.4M",
        },
        {
          deltaLabel: "+10.8% completion load",
          deltaTone: "positive",
          label: "Output Tokens",
          value: "49.7M",
        },
        {
          deltaLabel: "$0.013 avg / req",
          deltaTone: "critical",
          label: "Est. Cost",
          value: "$1,024.18",
        },
      ],
      recentRequests: [
        {
          estimatedCost: "$0.021",
          id: "req_11",
          inputTokens: "8.4k",
          latency: "1.0s",
          model: "gpt-5.4",
          outputTokens: "1.6k",
          route: "/v1/chat/completions",
          status: "Completed",
          timestamp: "Today, 09:42",
        },
        {
          estimatedCost: "$0.014",
          id: "req_12",
          inputTokens: "5.9k",
          latency: "711ms",
          model: "claude-sonnet-4",
          outputTokens: "1.1k",
          route: "/v1/responses",
          status: "Completed",
          timestamp: "Today, 09:37",
        },
        {
          estimatedCost: "$0.004",
          id: "req_13",
          inputTokens: "1.9k",
          latency: "266ms",
          model: "text-embedding-3-large",
          outputTokens: "0",
          route: "/v1/embeddings",
          status: "Cached",
          timestamp: "Today, 09:31",
        },
        {
          estimatedCost: "$0.033",
          id: "req_14",
          inputTokens: "14.5k",
          latency: "1.9s",
          model: "gemini-2.5-pro",
          outputTokens: "2.7k",
          route: "/v1/chat/completions",
          status: "Streaming",
          timestamp: "Today, 09:18",
        },
        {
          estimatedCost: "$0.000",
          id: "req_15",
          inputTokens: "2.3k",
          latency: "518ms",
          model: "gpt-4.1-mini",
          outputTokens: "0",
          route: "/v1/responses",
          status: "Failed",
          timestamp: "Today, 08:54",
        },
      ],
      timeseries: [
        {
          cost: 128.2,
          inputTokens: 15200000,
          label: "Mon",
          outputTokens: 6200000,
          requests: 10540,
        },
        {
          cost: 136.8,
          inputTokens: 16100000,
          label: "Tue",
          outputTokens: 6850000,
          requests: 11120,
        },
        {
          cost: 142.4,
          inputTokens: 16900000,
          label: "Wed",
          outputTokens: 7010000,
          requests: 11490,
        },
        {
          cost: 151.9,
          inputTokens: 17700000,
          label: "Thu",
          outputTokens: 7360000,
          requests: 12080,
        },
        {
          cost: 147.3,
          inputTokens: 17100000,
          label: "Fri",
          outputTokens: 7120000,
          requests: 11740,
        },
        {
          cost: 159.1,
          inputTokens: 18100000,
          label: "Sat",
          outputTokens: 8060000,
          requests: 12620,
        },
        {
          cost: 158.48,
          inputTokens: 18300000,
          label: "Sun",
          outputTokens: 8220000,
          requests: 13340,
        },
      ],
    },
    "30d": {
      breakdown: [
        { color: "#df7a52", label: "OpenAI", value: 39 },
        { color: "#9fd768", label: "Anthropic", value: 29 },
        { color: "#56d364", label: "Gemini", value: 21 },
        { color: "#f59e0b", label: "Fallback", value: 11 },
      ],
      metrics: [
        {
          deltaLabel: "+24.8% vs previous month",
          deltaTone: "positive",
          label: "Total Requests",
          value: "341,280",
        },
        {
          deltaLabel: "+16.5% prompt expansion",
          deltaTone: "warning",
          label: "Total Input Tokens",
          value: "481.2M",
        },
        {
          deltaLabel: "+19.2% response volume",
          deltaTone: "positive",
          label: "Output Tokens",
          value: "208.6M",
        },
        {
          deltaLabel: "$0.012 avg / req",
          deltaTone: "critical",
          label: "Est. Cost",
          value: "$4,296.77",
        },
      ],
      recentRequests: [
        {
          estimatedCost: "$0.028",
          id: "req_21",
          inputTokens: "12.7k",
          latency: "1.5s",
          model: "gpt-5.4",
          outputTokens: "1.9k",
          route: "/v1/chat/completions",
          status: "Completed",
          timestamp: "May 16, 09:42",
        },
        {
          estimatedCost: "$0.018",
          id: "req_22",
          inputTokens: "7.4k",
          latency: "684ms",
          model: "claude-sonnet-4",
          outputTokens: "1.4k",
          route: "/v1/responses",
          status: "Streaming",
          timestamp: "May 16, 09:37",
        },
        {
          estimatedCost: "$0.005",
          id: "req_23",
          inputTokens: "2.2k",
          latency: "251ms",
          model: "text-embedding-3-large",
          outputTokens: "0",
          route: "/v1/embeddings",
          status: "Cached",
          timestamp: "May 16, 09:31",
        },
        {
          estimatedCost: "$0.047",
          id: "req_24",
          inputTokens: "19.2k",
          latency: "2.4s",
          model: "gemini-2.5-pro",
          outputTokens: "3.9k",
          route: "/v1/chat/completions",
          status: "Completed",
          timestamp: "May 16, 09:18",
        },
        {
          estimatedCost: "$0.000",
          id: "req_25",
          inputTokens: "1.5k",
          latency: "403ms",
          model: "gpt-4.1-mini",
          outputTokens: "0",
          route: "/v1/moderations",
          status: "Failed",
          timestamp: "May 16, 08:54",
        },
      ],
      timeseries: [
        {
          cost: 502.4,
          inputTokens: 53400000,
          label: "W1",
          outputTokens: 22500000,
          requests: 37240,
        },
        {
          cost: 611.8,
          inputTokens: 64800000,
          label: "W2",
          outputTokens: 27400000,
          requests: 45580,
        },
        {
          cost: 738.1,
          inputTokens: 76100000,
          label: "W3",
          outputTokens: 33100000,
          requests: 54890,
        },
        {
          cost: 842.7,
          inputTokens: 89400000,
          label: "W4",
          outputTokens: 38600000,
          requests: 63120,
        },
        {
          cost: 901.77,
          inputTokens: 97800000,
          label: "W5",
          outputTokens: 47900000,
          requests: 70450,
        },
      ],
    },
  };
