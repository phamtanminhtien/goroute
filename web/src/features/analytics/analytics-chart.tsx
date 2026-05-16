import { type ECharts, type EChartsOption, init } from "echarts";
import { useEffect, useRef } from "react";

type AnalyticsChartProps = {
  className?: string;
  option: EChartsOption;
};

export function AnalyticsChart({ className, option }: AnalyticsChartProps) {
  const containerRef = useRef<HTMLDivElement | null>(null);
  const chartRef = useRef<ECharts | null>(null);

  useEffect(() => {
    if (!containerRef.current) {
      return;
    }

    const chart = init(containerRef.current);
    chartRef.current = chart;

    const resizeChart = () => {
      chart.resize();
    };

    let resizeObserver: ResizeObserver | null = null;
    if (typeof ResizeObserver !== "undefined") {
      resizeObserver = new ResizeObserver(() => {
        resizeChart();
      });
      resizeObserver.observe(containerRef.current);
    } else {
      window.addEventListener("resize", resizeChart);
    }

    return () => {
      resizeObserver?.disconnect();
      window.removeEventListener("resize", resizeChart);
      chart.dispose();
      chartRef.current = null;
    };
  }, []);

  useEffect(() => {
    if (!chartRef.current) {
      return;
    }

    chartRef.current.setOption(option, true);
  }, [option]);

  return <div className={className} ref={containerRef} />;
}
